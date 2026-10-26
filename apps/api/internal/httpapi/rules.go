package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/asarules"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// saveRules stores a draft's ASA rules (S40; ASA-02, ASA-03) if `versionId` is still the latest
// version and `revision` is still current: for
// whoever may edit the HQ's chapters (canEdit), never on a published version (409). Go checks the
// shape only (asarules.Check) and that the procedures a rule or test case names are the trust's;
// which rules fire is packages/clinical's. GET …/questionnaires/{qid} sends them back.
// Not audited: no patient data.
func (s *server) saveRules(w http.ResponseWriter, r *http.Request) {
	q, ok := s.loadQuestionnaire(w, r)
	if !ok || !canEdit(w, r, q.HospitalID, q.Status) {
		return
	}
	var in struct {
		Rules     json.RawMessage `json:"rules"`
		Revision  int32           `json:"revision"`
		VersionID uuid.UUID       `json:"versionId"` // the version the page loaded (verification 1, F1)
	}
	if !readJSON(w, r, asarules.MaxBytes+4<<10, &in) {
		return
	}
	if in.VersionID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "The rules could not be read.")
		return
	}
	// A page opened on an older version (since published, with a new draft made) mustn't save onto
	// the new draft, whose rules revision starts at 0 again. As publish does.
	if in.VersionID != q.VersionID {
		writeError(w, http.StatusConflict, "This version has changed since you opened it. Reload and try again.")
		return
	}
	set, problem := asarules.Check(in.Rules)
	if problem != "" {
		writeError(w, http.StatusBadRequest, problem)
		return
	}
	ctx := r.Context()
	if ids := set.ProcedureIDs(); len(ids) > 0 {
		n, err := s.q.CountTrustProcedures(ctx, db.CountTrustProceduresParams{OrgID: q.OrgID, Ids: ids})
		if err != nil {
			serverError(w, "check procedures", err)
			return
		}
		if int(n) != len(ids) {
			writeError(w, http.StatusBadRequest, "A rule or test case names a procedure that isn't one of this trust's.")
			return
		}
	}
	rev, err := s.q.SaveVersionRules(ctx, db.SaveVersionRulesParams{
		ID: q.VersionID, Rules: set.JSON(), Revision: in.Revision, UpdatedBy: currentUser(r).ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Published meanwhile, or someone else saved first.
		status, err := s.q.VersionStatus(ctx, q.VersionID)
		if err == nil && status != "draft" {
			writeError(w, http.StatusConflict, notDraft)
			return
		}
		writeError(w, http.StatusConflict, "Someone else changed these rules since you opened them. Reload to see their changes; your latest change was not saved.")
		return
	}
	if err != nil {
		serverError(w, "save rules", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revision": rev, "rules": set}) // as stored: trimmed
}
