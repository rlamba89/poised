package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/chapter"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// Step 4 of plan-workflow.md: the clinician validates each Question Set, correcting the
// patient's answers and filling in their own questions, then completes the HQ review.

// episodeHQ returns every Question Set of the episode's version, in full, with both actors'
// answers: the patient's (frozen once submitted) and the clinician's (final).
func (s *server) episodeHQ(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	chapters, err := s.q.ListVersionChapters(ctx, e.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	answers, err := s.q.ListEpisodeAnswers(ctx, e.HqFormID)
	if err != nil {
		serverError(w, "list answers", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chapters": chapters, "answers": answers})
}

// staleAnswersMsg answers a save whose revision isn't the row's current one: another tab, device
// or person has saved the Question Set since this one loaded it (S02a).
const staleAnswersMsg = "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved."

// answersInput is the body of an answers save, the clinician's or the patient's.
type answersInput struct {
	Data      map[string]json.RawMessage `json:"data"`
	Validated bool                       `json:"validated"` // the clinician's only
	// Revision is the one the browser loaded or last saved: 0 for a Question Set it has no row for.
	Revision *int32 `json:"revision"`
}

// validate returns a plain-language problem, or "".
func (in *answersInput) validate() string {
	if in.Revision == nil || *in.Revision < 0 {
		return "The request could not be read."
	}
	if in.Data == nil {
		in.Data = map[string]json.RawMessage{}
	}
	return ""
}

// saveClinicianAnswers saves the clinician's answers for one Question Set, and stamps it as
// validated when `validated` is true. The browser has checked required questions (plan 2.1).
// It answers with the new revision, or 409 if the revision sent is stale.
func (s *server) saveClinicianAnswers(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	cid, ok := pathID(w, r, "cid", chapterNotFound)
	if !ok {
		return
	}
	if _, err := s.q.GetVersionChapter(r.Context(), db.GetVersionChapterParams{ID: cid, VersionID: e.VersionID}); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, chapterNotFound)
		return
	} else if err != nil {
		serverError(w, "get chapter", err)
		return
	}
	var in answersInput
	if !readJSON(w, r, chapter.MaxContentBytes, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	raw, err := json.Marshal(in.Data)
	if err != nil {
		serverError(w, "encode answers", err)
		return
	}
	rev, err := s.q.SaveClinicianAnswers(r.Context(), db.SaveClinicianAnswersParams{
		FormID: e.HqFormID, ChapterID: cid, Data: raw, Validated: in.Validated, Revision: *in.Revision,
		UpdatedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		if e.Status != episode.ReadyForReview {
			writeError(w, http.StatusConflict, notInReview(e.Status))
		} else {
			writeError(w, http.StatusConflict, staleAnswersMsg)
		}
		return
	}
	if err != nil {
		serverError(w, "save answers", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int32{"revision": rev})
}

// completeReview moves the episode to Ready for POA, through the transition table (S20). The
// browser enables it once every Question Set shown is validated (plan 2.1: Go doesn't evaluate
// Question Set conditions).
func (s *server) completeReview(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	if _, err := episode.Check(e.Status, episode.ReadyForPOA, episode.CompleteReview, here(r).Role); err != nil {
		writeError(w, http.StatusConflict, notInReview(e.Status))
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	uid := currentUser(r).ID
	n, err := q.CompleteReview(ctx, db.CompleteReviewParams{ID: e.ID, ReviewCompletedBy: uuid.NullUUID{UUID: uid, Valid: true}})
	if err != nil {
		serverError(w, "complete review", err)
		return
	}
	if n == 0 { // moved on meanwhile
		writeError(w, http.StatusConflict, notInReview(episode.ReadyForPOA))
		return
	}
	note := "HQ review completed by " + currentUser(r).Name
	if err := recordStatus(ctx, q, e.ID, episode.CompleteReview, episode.ReadyForReview, episode.ReadyForPOA, "", uid, note); err != nil {
		serverError(w, "status history", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func notInReview(status string) string {
	switch status {
	case episode.HQNotComplete:
		return "The patient hasn't sent their answers yet."
	case episode.Cancelled:
		return cancelledMsg
	case episode.Archived:
		return archivedMsg
	}
	return "The HQ review is already complete, so the answers can't be changed."
}
