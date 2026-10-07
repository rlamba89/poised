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
	chapters, err := s.forOrg(orgID(r)).ListVersionChapters(ctx, e.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	answers, err := s.forOrg(orgID(r)).ListEpisodeAnswers(ctx, e.ID)
	if err != nil {
		serverError(w, "list answers", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chapters": chapters, "answers": answers})
}

// saveClinicianAnswers saves the clinician's answers for one Question Set, and stamps it as
// validated when `validated` is true. The browser has checked required questions (plan 2.1).
func (s *server) saveClinicianAnswers(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	cid, ok := pathID(w, r, "cid", chapterNotFound)
	if !ok {
		return
	}
	if _, err := s.forOrg(orgID(r)).GetVersionChapter(r.Context(), db.GetVersionChapterParams{ID: cid, VersionID: e.VersionID}); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, chapterNotFound)
		return
	} else if err != nil {
		serverError(w, "get chapter", err)
		return
	}
	var in struct {
		Data      map[string]json.RawMessage `json:"data"`
		Validated bool                       `json:"validated"`
	}
	if !readJSON(w, r, chapter.MaxContentBytes, &in) {
		return
	}
	if in.Data == nil {
		in.Data = map[string]json.RawMessage{}
	}
	raw, err := json.Marshal(in.Data)
	if err != nil {
		serverError(w, "encode answers", err)
		return
	}
	n, err := s.forOrg(orgID(r)).SaveClinicianAnswers(r.Context(), db.SaveClinicianAnswersParams{
		EpisodeID: e.ID, ChapterID: cid, Data: raw, Validated: in.Validated,
		UpdatedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
	})
	if err != nil {
		serverError(w, "save answers", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, notInReview(e.Status))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// completeReview moves the episode to Ready for POA. The browser enables it once every
// Question Set shown is validated (plan 2.1: Go doesn't evaluate Question Set conditions).
func (s *server) completeReview(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	uid := currentUser(r).ID
	n, err := s.forOrg(orgID(r)).CompleteReview(r.Context(), db.CompleteReviewParams{ID: e.ID, ReviewCompletedBy: uuid.NullUUID{UUID: uid, Valid: true}})
	if err != nil {
		serverError(w, "complete review", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, notInReview(e.Status))
		return
	}
	s.event(r, e.ID, "HQ review completed by "+currentUser(r).Name, uid)
	w.WriteHeader(http.StatusNoContent)
}

func notInReview(status string) string {
	if status == episode.HQNotComplete {
		return "The patient hasn't sent their answers yet."
	}
	return "The HQ review is already complete, so the answers can't be changed."
}
