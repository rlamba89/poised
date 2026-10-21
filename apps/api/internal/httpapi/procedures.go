package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// The trust's procedure list (plan C3, session S19; PRC-01): GET/POST /api/o/{oid}/procedures and
// PATCH /api/o/{oid}/procedures/{prid}.
//   - Everyone in the trust reads it: a whole-trust member the whole list, a member of some
//     hospitals the trust's procedures and those hospitals' own (hospital_id NULL is every
//     hospital's, S19 answers 2). ?hospital= is what a new episode there is offered.
//   - Whole-trust super clinicians and admins change it (answers 1), as they do the trust's
//     questionnaires. A procedure is never deleted: it's retired (answers 5).
//   - The default HQ is a questionnaire of the trust's with a published version; a new episode
//     gets that version (answers 6), or the hospital's default HQ (S10) when it has none.

const (
	procedureNotFound  = "Procedure not found."
	proceduresRole     = "Only super clinicians and admins of the whole trust can change the trust's procedures."
	procedureNameTaken = "The trust already has a procedure with this name (retired ones included)."
	procedureDefaultHQ = "Default HQ: choose one of the trust's published questionnaires."
	maxProcedureName   = 200
	maxProcedureCode   = 50
	maxSpecialty       = 100
)

// offeredProcedure is a procedure a new episode at one hospital can be given, with the HQ that is
// chosen first when it's picked: the published version of its own default HQ, else of the
// hospital's default HQ, else none (the clinician's choice stays).
type offeredProcedure struct {
	ID               uuid.UUID     `json:"id"`
	Name             string        `json:"name"`
	Code             string        `json:"code"`
	Specialty        string        `json:"specialty"`
	HospitalID       uuid.NullUUID `json:"hospitalId"`
	DefaultVersionID *uuid.UUID    `json:"defaultVersionId"`
}

func (s *server) listProcedures(w http.ResponseWriter, r *http.Request) {
	p := here(r)
	if raw := r.URL.Query().Get("hospital"); raw != "" {
		hid, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "The hospital could not be read.")
			return
		}
		if p.Role == role.None && !slices.Contains(p.Hospitals, hid) {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return
		}
		s.listProceduresAt(w, r, hid)
		return
	}
	hospitals := p.Hospitals
	if hospitals == nil {
		hospitals = []uuid.UUID{}
	}
	rows, err := s.q.ListProcedures(r.Context(), db.ListProceduresParams{OrgID: p.Org, AllHospitals: p.Role != role.None, HospitalIds: hospitals})
	if err != nil {
		serverError(w, "list procedures", err)
		return
	}
	if rows == nil {
		rows = []db.ListProceduresRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

// listProceduresAt is ?hospital=: the procedures offered there, each with its default HQ there.
func (s *server) listProceduresAt(w http.ResponseWriter, r *http.Request, hid uuid.UUID) {
	ctx, org := r.Context(), orgID(r)
	hqs, hospitalDefault, err := s.hospitalHQs(ctx, org, hid)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Hospital not found.")
		return
	}
	if err != nil {
		serverError(w, "hospital HQs", err)
		return
	}
	rows, err := s.q.ListProceduresOffered(ctx, db.ListProceduresOfferedParams{OrgID: org, HospitalID: hid})
	if err != nil {
		serverError(w, "list procedures offered", err)
		return
	}
	out := []offeredProcedure{}
	for _, p := range rows {
		v := versionOf(hqs, p.DefaultQuestionnaireID)
		if v == nil {
			v = versionOf(hqs, hospitalDefault)
		}
		out = append(out, offeredProcedure{ID: p.ID, Name: p.Name, Code: p.Code, Specialty: p.Specialty, HospitalID: p.HospitalID, DefaultVersionID: v})
	}
	writeJSON(w, http.StatusOK, out)
}

// hospitalHQs are the published HQs a new episode at the hospital can be given, and the
// questionnaire of its default HQ (its effective defaultHq setting, S10), if it has one.
// pgx.ErrNoRows: the hospital isn't the trust's.
func (s *server) hospitalHQs(ctx context.Context, org, hid uuid.UUID) ([]db.ListPublishedVersionsRow, uuid.NullUUID, error) {
	var def uuid.NullUUID
	st, err := s.q.GetHospitalAndOrgSettings(ctx, db.GetHospitalAndOrgSettingsParams{HospitalID: hid, OrgID: org})
	if err != nil {
		return nil, def, err
	}
	trust, err := settings.Decode(st.OrgSettings)
	if err != nil {
		return nil, def, fmt.Errorf("trust settings: %w", err)
	}
	own, err := settings.Decode(st.HospitalSettings)
	if err != nil {
		return nil, def, fmt.Errorf("hospital settings: %w", err)
	}
	var id string
	if json.Unmarshal(settings.Effective(settings.Platform(), trust, own)["defaultHq"], &id) == nil {
		if q, err := uuid.Parse(id); err == nil {
			def = uuid.NullUUID{UUID: q, Valid: true}
		}
	}
	hqs, err := s.q.ListPublishedVersions(ctx, db.ListPublishedVersionsParams{OrgID: org, HospitalID: hid})
	return hqs, def, err
}

// versionOf is the published version of questionnaire q among hqs, or nil if it has none there.
func versionOf(hqs []db.ListPublishedVersionsRow, q uuid.NullUUID) *uuid.UUID {
	if !q.Valid {
		return nil
	}
	for _, h := range hqs {
		if h.QuestionnaireID == q.UUID {
			return &h.VersionID
		}
	}
	return nil
}

// procedureFields are a procedure's changeable fields.
type procedureFields struct {
	Name, Code, Specialty string
	HospitalID            uuid.NullUUID
	DefaultQuestionnaire  uuid.NullUUID
	Retired               bool
}

// procedureChange is a POST or PATCH body, decoded over the current fields: a field left out
// keeps its value, a text field or retired sent as null is refused, and a hospital or default HQ
// sent as null is cleared.
type procedureChange struct {
	Name                   *string         `json:"name"`
	Code                   *string         `json:"code"`
	Specialty              *string         `json:"specialty"`
	HospitalID             json.RawMessage `json:"hospitalId"`
	DefaultQuestionnaireID json.RawMessage `json:"defaultQuestionnaireId"`
	Retired                *bool           `json:"retired"`
}

// readProcedureChange checks the caller may change the trust's procedures, reads the body over
// current, and returns the result, answering any refusal itself.
func readProcedureChange(w http.ResponseWriter, r *http.Request, current procedureFields) (procedureFields, bool) {
	if !here(r).TrustRole.AtLeast(role.SuperClinician) {
		writeError(w, http.StatusForbidden, proceduresRole)
		return current, false
	}
	next := current
	in := procedureChange{Name: &next.Name, Code: &next.Code, Specialty: &next.Specialty, Retired: &next.Retired}
	if !readJSON(w, r, 8<<10, &in) {
		return current, false
	}
	if msg := in.apply(&next); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return current, false
	}
	return next, true
}

// apply trims and checks the decoded change and puts it into p; it returns a plain-language problem, or "".
func (in procedureChange) apply(p *procedureFields) string {
	if in.Name == nil || in.Code == nil || in.Specialty == nil || in.Retired == nil {
		return "The request could not be read."
	}
	p.Name, p.Code, p.Specialty, p.Retired = strings.TrimSpace(*in.Name), strings.TrimSpace(*in.Code), strings.TrimSpace(*in.Specialty), *in.Retired
	switch {
	case p.Name == "":
		return "Enter the procedure's name."
	case len([]rune(p.Name)) > maxProcedureName:
		return fmt.Sprintf("The name is too long (%d characters at most).", maxProcedureName)
	case len([]rune(p.Code)) > maxProcedureCode:
		return fmt.Sprintf("The code is too long (%d characters at most).", maxProcedureCode)
	case len([]rune(p.Specialty)) > maxSpecialty:
		return fmt.Sprintf("The specialty is too long (%d characters at most).", maxSpecialty)
	}
	if in.HospitalID != nil {
		h, ok := parseHospitalID(in.HospitalID)
		if !ok {
			return hospitalNotHere
		}
		p.HospitalID = h
	}
	if in.DefaultQuestionnaireID != nil {
		q, ok := parseHospitalID(in.DefaultQuestionnaireID) // any optional id: null, or a uuid
		if !ok {
			return procedureDefaultHQ
		}
		p.DefaultQuestionnaire = q
	}
	return ""
}

// checkProcedure answers 400 unless the hospital (if changed) is the trust's and the default HQ
// (if changed) is one of its published questionnaires. Unchanged ones are kept as they are: a
// retired default HQ doesn't stop the procedure being renamed or retired.
func checkProcedure(w http.ResponseWriter, r *http.Request, q *db.Queries, current, next procedureFields) bool {
	if next.HospitalID != current.HospitalID && !checkHospital(w, r, q, next.HospitalID) {
		return false
	}
	if next.DefaultQuestionnaire == current.DefaultQuestionnaire || !next.DefaultQuestionnaire.Valid {
		return true
	}
	hqs, err := q.PublishedQuestionnairesInOrg(r.Context(), orgID(r))
	if err != nil {
		serverError(w, "questionnaires", err)
		return false
	}
	if !slices.ContainsFunc(hqs, func(h db.PublishedQuestionnairesInOrgRow) bool { return h.ID == next.DefaultQuestionnaire.UUID }) {
		writeError(w, http.StatusBadRequest, procedureDefaultHQ)
		return false
	}
	return true
}

func (s *server) createProcedure(w http.ResponseWriter, r *http.Request) {
	in, ok := readProcedureChange(w, r, procedureFields{})
	if !ok {
		return
	}
	var id uuid.UUID
	ok = s.inTx(w, r, func(q *db.Queries) bool {
		if !checkProcedure(w, r, q, procedureFields{}, in) {
			return false
		}
		var err error
		id, err = q.CreateProcedure(r.Context(), db.CreateProcedureParams{
			OrgID: orgID(r), HospitalID: in.HospitalID, Name: in.Name, Code: in.Code, Specialty: in.Specialty,
			DefaultQuestionnaireID: in.DefaultQuestionnaire, UpdatedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
		})
		return procedureSaved(w, err)
	})
	if ok {
		writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
	}
}

// updateProcedure changes the fields sent: a rename, its hospital, its default HQ, retiring it or
// bringing it back. Episodes that have it keep it whatever changes.
func (s *server) updateProcedure(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "prid", procedureNotFound)
	if !ok {
		return
	}
	if !here(r).TrustRole.AtLeast(role.SuperClinician) {
		writeError(w, http.StatusForbidden, proceduresRole)
		return
	}
	ok = s.inTx(w, r, func(q *db.Queries) bool {
		row, err := q.GetProcedure(r.Context(), db.GetProcedureParams{ID: id, OrgID: orgID(r)})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, procedureNotFound)
			return false
		}
		if err != nil {
			serverError(w, "get procedure", err)
			return false
		}
		current := procedureFields{Name: row.Name, Code: row.Code, Specialty: row.Specialty, HospitalID: row.HospitalID,
			DefaultQuestionnaire: row.DefaultQuestionnaireID, Retired: row.Retired}
		next, ok := readProcedureChange(w, r, current)
		if !ok || !checkProcedure(w, r, q, current, next) {
			return false
		}
		err = q.UpdateProcedure(r.Context(), db.UpdateProcedureParams{
			ID: id, OrgID: orgID(r), HospitalID: next.HospitalID, Name: next.Name, Code: next.Code, Specialty: next.Specialty,
			DefaultQuestionnaireID: next.DefaultQuestionnaire, Retired: next.Retired,
			UpdatedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
		})
		return procedureSaved(w, err)
	})
	if ok {
		w.WriteHeader(http.StatusNoContent)
	}
}

// procedureSaved answers a failed save itself: a name the trust already has is a 409.
func procedureSaved(w http.ResponseWriter, err error) bool {
	switch {
	case isUniqueViolation(err):
		writeError(w, http.StatusConflict, procedureNameTaken)
	case err != nil:
		serverError(w, "save procedure", err)
	default:
		return true
	}
	return false
}
