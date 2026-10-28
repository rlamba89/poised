package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// The anaesthetist's ASA grade as a history (plan C6, session S44b; Rahul, 27 Oct). A grade, once
// given, is never changed, cleared or deleted, in any status (migration 00035 refuses an edit or a
// delete too):
//
//   - POST …/episodes/{eid}/anaesthetist-asa {grade, reason, revision}: any clinician at the
//     hospital adds a grade (as anyone could set it before). The episode's first needs no reason;
//     every later one must say why (422), even after the earlier were entered in error (S44b
//     answers 2). It counts from now on; the earlier stays.
//   - POST …/episodes/{eid}/anaesthetist-asa/{gid}/entered-in-error {reason, revision}: a grade
//     given by mistake is marked, with a reason, once (it can't be unmarked). It stays in the
//     history, struck through, and no longer counts: the one before it (if any) counts again.
//
// The grade that counts is the latest not entered in error: GET …/episodes/{eid} gives it as
// episode.anaesthetistAsa, with the whole history in anaesthetistAsaGrades. Like a task or a flag,
// each write sends the revision the page loaded and bumps it (a grade added in another tab
// meanwhile: 409 stale), with a General notes line. A cancelled or archived episode's grades can't
// change.
//
// S43's requirement reads the grade that counts. While a rule requires it and the episode is at
// Ready for admission or later, the only counting grade can't be marked entered in error (409), so
// the grade can't go missing there.

const (
	gradeNotFound     = "Grade not found."
	gradeReasonNeeded = "Say why the grade is changed."
	alreadyInError    = "This grade is already marked entered in error."
	onlyCountingGrade = "Add the correct grade first, or move the episode back from Ready for admission."
)

// countingGrade is the anaesthetist's grade that counts now (nil: none). Called after
// LockEpisodeStatus, in a statement of its own, so it sees the grades as they are under the lock.
func countingGrade(ctx context.Context, q *db.Queries, episodeID uuid.UUID) (*int32, error) {
	g, err := q.CountingAnaesthetistGrade(ctx, episodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// atOrPastAdmission: Ready for admission, Admitted or Discharged.
func atOrPastAdmission(status string) bool {
	return status == episode.ReadyForAdmission || status == episode.Admitted || status == episode.Discharged
}

func (s *server) addAnaesthetistGrade(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	var in struct {
		Grade    *int32 `json:"grade"`
		Reason   string `json:"reason"`
		Revision *int32 `json:"revision"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	reason, msg := optionalReason(in.Reason)
	switch {
	case in.Grade == nil || *in.Grade < 1 || *in.Grade > 6:
		msg = "Choose an ASA grade from 1 to 6."
	case msg == "" && (in.Revision == nil || *in.Revision < 0):
		msg = "The request could not be read."
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	grade := *in.Grade
	var id uuid.UUID
	if !s.changeLockedEpisode(w, r, e, *in.Revision, func(q *db.Queries, _ db.LockEpisodeStatusRow) (string, bool, error) {
		was, err := countingGrade(r.Context(), q, e.ID)
		if err != nil {
			return "", false, err
		}
		graded, err := q.HasAnaesthetistGrade(r.Context(), e.ID) // even one entered in error (verification 1)
		if err != nil {
			return "", false, err
		}
		if graded && reason == "" {
			writeError(w, http.StatusUnprocessableEntity, gradeReasonNeeded)
			return "", false, nil
		}
		var why *string
		if reason != "" {
			why = &reason
		}
		id, err = q.AddAnaesthetistGrade(r.Context(), db.AddAnaesthetistGradeParams{
			EpisodeID: e.ID, Grade: grade, UserID: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true}, Reason: why})
		if err != nil {
			return "", false, err
		}
		note := fmt.Sprintf("Anaesthetist ASA grade set to %d", grade) // S20's wording for the first
		switch {
		case was != nil && *was == grade: // e.g. a second anaesthetist agrees (test report 1, exploring note 1)
			note = fmt.Sprintf("Anaesthetist ASA grade confirmed as %d", grade)
		case was != nil:
			note = fmt.Sprintf("Anaesthetist ASA grade changed from %d to %d", *was, grade)
		}
		return withReason(note, reason), true, nil
	}) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

func (s *server) markAnaesthetistGradeInError(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	gid, ok := pathID(w, r, "gid", gradeNotFound)
	if !ok {
		return
	}
	var in struct {
		Reason   string `json:"reason"`
		Revision *int32 `json:"revision"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	reason, msg := requiredReason(in.Reason, "Say why the grade was entered in error.")
	if msg == "" && (in.Revision == nil || *in.Revision < 0) {
		msg = "The request could not be read."
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	g, err := s.q.GetAnaesthetistGrade(r.Context(), db.GetAnaesthetistGradeParams{ID: gid, EpisodeID: e.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, gradeNotFound)
		return
	}
	if err != nil {
		serverError(w, "get grade", err)
		return
	}
	if !s.changeLockedEpisode(w, r, e, *in.Revision, func(q *db.Queries, now db.LockEpisodeStatusRow) (string, bool, error) {
		if g, err := q.GetAnaesthetistGrade(r.Context(), db.GetAnaesthetistGradeParams{ID: gid, EpisodeID: e.ID}); err != nil || g.ErrorAt.Valid {
			if err == nil {
				writeError(w, http.StatusConflict, alreadyInError)
			}
			return "", false, err
		}
		// The only grade that counts can't go while a rule requires one there (S43's requirement).
		if now.AnaesthetistAsaRequiredBy != nil && atOrPastAdmission(now.Status) {
			others, err := q.CountingAnaesthetistGradesBut(r.Context(), db.CountingAnaesthetistGradesButParams{EpisodeID: e.ID, ID: gid})
			if err != nil {
				return "", false, err
			}
			if others == 0 {
				writeError(w, http.StatusConflict, onlyCountingGrade)
				return "", false, nil
			}
		}
		n, err := q.MarkAnaesthetistGradeInError(r.Context(), db.MarkAnaesthetistGradeInErrorParams{
			ID: gid, EpisodeID: e.ID, UserID: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true}, Reason: reason})
		if err == nil && n == 0 {
			writeError(w, http.StatusConflict, alreadyInError)
		}
		return fmt.Sprintf("Anaesthetist ASA grade %d marked entered in error: %s", g.Grade, reason), n > 0, err
	}) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
