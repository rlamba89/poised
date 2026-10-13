package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// Trust settings (plan C1, session S10; ORG-04): GET/PATCH /api/o/{oid}/settings, for the
// trust's whole-trust admins only (as Staff is, S09). They set the trust's values, which every
// hospital follows, and each hospital's own values where a hospital may override. Trust-only
// settings can't be set for a hospital, and the opt-ins are the platform admin's (PRM-08).

const (
	settingsRole  = "Only the trust's admins can change its settings."
	optInsRefused = "Features and opt-ins are set by Poised's platform admin. Ask them to change it."
)

type settingsHospitalView struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Status    string          `json:"status"`
	Values    settings.Values `json:"values"`    // set for this hospital
	Effective settings.Values `json:"effective"` // what applies there
}

type questionnaireChoice struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type settingsView struct {
	Defs           []settings.Def         `json:"defs"`
	Trust          settings.Values        `json:"trust"`     // set for the trust
	Effective      settings.Values        `json:"effective"` // the trust's, with the platform's defaults
	Hospitals      []settingsHospitalView `json:"hospitals"`
	Questionnaires []questionnaireChoice  `json:"questionnaires"` // the choices for the default HQ
}

// settingsAdmin answers 403 unless the caller is an admin of the whole trust.
func settingsAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !here(r).TrustRole.AtLeast(role.Admin) {
		writeError(w, http.StatusForbidden, settingsRole)
		return false
	}
	return true
}

// hospitalOnly keeps the values a hospital may set.
func hospitalOnly(v settings.Values) settings.Values {
	out := settings.Values{}
	for k, raw := range v {
		if d, ok := settings.Lookup(k); ok && d.Level == settings.Hospital {
			out[k] = raw
		}
	}
	return out
}

func (s *server) settingsView(ctx context.Context, org uuid.UUID) (settingsView, error) {
	o, err := s.q.GetOrgForAdmin(ctx, org)
	if err != nil {
		return settingsView{}, err
	}
	trust, err := settings.Decode(o.Settings)
	if err != nil {
		return settingsView{}, fmt.Errorf("trust settings: %w", err)
	}
	platform := settings.Platform()
	out := settingsView{Defs: settings.Defs, Trust: trust, Effective: settings.Effective(platform, trust, nil),
		Hospitals: []settingsHospitalView{}, Questionnaires: []questionnaireChoice{}}
	hospitals, err := s.q.ListOrgHospitals(ctx, org)
	if err != nil {
		return settingsView{}, err
	}
	for _, h := range hospitals {
		own, err := settings.Decode(h.Settings)
		if err != nil {
			return settingsView{}, fmt.Errorf("hospital settings: %w", err)
		}
		own = hospitalOnly(own)
		out.Hospitals = append(out.Hospitals, settingsHospitalView{ID: h.ID, Name: h.Name, Status: h.Status,
			Values: own, Effective: settings.Effective(platform, trust, own)})
	}
	hqs, err := s.q.PublishedQuestionnairesInOrg(ctx, org)
	if err != nil {
		return settingsView{}, err
	}
	for _, q := range hqs {
		out.Questionnaires = append(out.Questionnaires, questionnaireChoice{ID: q.ID, Name: q.Name})
	}
	return out, nil
}

func (s *server) writeSettings(w http.ResponseWriter, r *http.Request) {
	v, err := s.settingsView(r.Context(), orgID(r))
	if err != nil {
		serverError(w, "read settings", err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *server) getTrustSettings(w http.ResponseWriter, r *http.Request) {
	if settingsAdmin(w, r) {
		s.writeSettings(w, r)
	}
}

// settingsPatch sets values for the trust (hospitalId null) or one hospital. A null value removes
// it: the hospital follows the trust again, or the trust uses the platform's default.
type settingsPatch struct {
	HospitalID json.RawMessage            `json:"hospitalId"`
	Values     map[string]json.RawMessage `json:"values"`
}

// check validates every change before anything is written: the first refusal wins, by key order.
func (p settingsPatch) check(hospital uuid.NullUUID) (map[string]json.RawMessage, int, string) {
	if len(p.Values) == 0 {
		return nil, http.StatusBadRequest, "Nothing to change."
	}
	out := map[string]json.RawMessage{}
	for _, k := range slices.Sorted(maps.Keys(p.Values)) {
		d, ok := settings.Lookup(k)
		switch {
		case !ok:
			return nil, http.StatusBadRequest, fmt.Sprintf("%q isn't a setting.", k)
		case d.OptIn:
			return nil, http.StatusForbidden, optInsRefused
		case hospital.Valid && d.Level == settings.TrustOnly:
			return nil, http.StatusBadRequest, d.Label + " is set for the whole trust only."
		}
		if strings.TrimSpace(string(p.Values[k])) == "null" {
			out[k] = nil
			continue
		}
		v, msg := settings.Validate(k, p.Values[k])
		if msg != "" {
			return nil, http.StatusBadRequest, msg
		}
		out[k] = v
	}
	return out, 0, ""
}

// checkQuestionnaires answers 400 unless each questionnaire setting being set names one of the
// trust's published questionnaires.
func checkQuestionnaires(w http.ResponseWriter, r *http.Request, q *db.Queries, changes map[string]json.RawMessage) bool {
	for k, v := range changes {
		d, _ := settings.Lookup(k)
		var id string
		if d.Kind != settings.Questionnaire || v == nil || json.Unmarshal(v, &id) != nil || id == "" {
			continue
		}
		hqs, err := q.PublishedQuestionnairesInOrg(r.Context(), orgID(r))
		if err != nil {
			serverError(w, "questionnaires", err)
			return false
		}
		if !slices.ContainsFunc(hqs, func(h db.PublishedQuestionnairesInOrgRow) bool { return h.ID.String() == id }) {
			writeError(w, http.StatusBadRequest, d.Label+": choose one of the trust's published questionnaires.")
			return false
		}
	}
	return true
}

func (s *server) patchTrustSettings(w http.ResponseWriter, r *http.Request) {
	if !settingsAdmin(w, r) {
		return
	}
	var in settingsPatch
	if !readJSON(w, r, 64<<10, &in) {
		return
	}
	hospital, ok := parseHospitalID(in.HospitalID)
	if !ok {
		writeError(w, http.StatusBadRequest, hospitalNotHere)
		return
	}
	changes, status, msg := in.check(hospital)
	if msg != "" {
		writeError(w, status, msg)
		return
	}
	ctx, org := r.Context(), orgID(r)
	ok = s.inTx(w, r, func(q *db.Queries) bool {
		if !checkQuestionnaires(w, r, q, changes) {
			return false
		}
		var stored json.RawMessage
		var err error
		if hospital.Valid {
			stored, err = q.LockHospitalSettings(ctx, db.LockHospitalSettingsParams{OrgID: org, ID: hospital.UUID})
		} else {
			stored, err = q.LockOrgSettings(ctx, org)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, hospitalNotHere)
			return false
		}
		if err != nil {
			serverError(w, "lock settings", err)
			return false
		}
		next, err := changeSettings(stored, changes)
		if err == nil && hospital.Valid {
			err = q.SetHospitalSettings(ctx, db.SetHospitalSettingsParams{OrgID: org, ID: hospital.UUID, Settings: next})
		} else if err == nil {
			err = q.SetOrgSettings(ctx, db.SetOrgSettingsParams{ID: org, Settings: next})
		}
		if err != nil {
			serverError(w, "save settings", err)
			return false
		}
		return true
	})
	if ok {
		s.writeSettings(w, r)
	}
}
