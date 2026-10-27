package httpapi

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

// An episode's flags from the HQ's rules (plan C6, session S43; ASA-03). A rule that fires on a
// send adds its flag (ruleActions, patient.go); the flags come with the episode (GET …/episodes/{eid})
// and the list (each row's standing flags). Clearing one:
//
//   - POST …/episodes/{eid}/flags/{fid}/clear {reason, revision}: any clinician at the hospital
//     clears a standing flag with a short reason (S43 answers 1); who, when and why are recorded,
//     with a General notes line. Like a task, it's a change to the episode: it sends the revision
//     the page loaded and bumps it. A cancelled or archived episode's flags can't be cleared.

const flagNotFound = "Flag not found."

func (s *server) clearFlag(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	fid, ok := pathID(w, r, "fid", flagNotFound)
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
	reason, msg := requiredReason(in.Reason, "Say why the flag is cleared.")
	if msg == "" && (in.Revision == nil || *in.Revision < 0) {
		msg = "The request could not be read."
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	flag, err := s.q.GetEpisodeFlag(r.Context(), db.GetEpisodeFlagParams{ID: fid, EpisodeID: e.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, flagNotFound)
		return
	}
	if err != nil {
		serverError(w, "get flag", err)
		return
	}
	user := currentUser(r).ID
	if !s.changeEpisode(w, r, e, *in.Revision, "Flag cleared: "+flag.Name+": "+reason, func(q *db.Queries) (bool, error) {
		n, err := q.ClearEpisodeFlag(r.Context(), db.ClearEpisodeFlagParams{ID: fid, EpisodeID: e.ID, UserID: uuid.NullUUID{UUID: user, Valid: true}, Reason: reason})
		if err == nil && n == 0 {
			writeError(w, http.StatusConflict, "This flag is already cleared.")
		}
		return n > 0, err
	}) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
