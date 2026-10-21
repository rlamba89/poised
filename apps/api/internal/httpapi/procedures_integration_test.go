//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The trust's procedure list (plan C3, session S19; PRC-01) at /api/o/{oid}/procedures. Whole-trust
// super clinicians and admins manage it (S19 answers 1); everyone in the trust reads it. A
// hospital-only procedure (answers 2) is offered at that hospital alone; a retired one (answers 5)
// at none, though it stays listed for managing and on the episodes that have it.

type procedureView struct {
	ID                       uuid.UUID
	Name, Code, Specialty    string
	HospitalID               *uuid.UUID
	HospitalName             *string
	DefaultQuestionnaireID   *uuid.UUID
	DefaultQuestionnaireName *string
	InfoContent, PromsPlan   json.RawMessage
	Retired                  bool
	DefaultVersionID         *uuid.UUID // ?hospital= only: the HQ chosen first on a new episode there
}

// procedureWorld is trust A (Hospitals A1 and A2) and trust B, with their people.
type procedureWorld struct {
	c                *apitest.Client
	fx               *apitest.Fixtures
	a1, a2, b1       apitest.Hospital
	alex, ada, bea   apitest.User
	alexC, adaC      *apitest.Client // a whole-trust super clinician and admin of A
	cara, sue, val   *apitest.Client // a clinician at A1; a super clinician at A1 only; a clinician at A2
	trustClin, beaC  *apitest.Client // a whole-trust clinician of A; Bea, a whole-trust super clinician of B
	hq, hq2, draftHQ apitest.HQ      // A's published HQs and a draft one
	otherHQ          apitest.HQ      // B's published HQ
}

func newProcedureWorld(t *testing.T) *procedureWorld {
	t.Helper()
	c, fx := apitest.New(t)
	w := &procedureWorld{c: c, fx: fx, a1: fx.Hospital(t), b1: fx.Hospital(t)}
	w.a2 = fx.HospitalIn(t, w.a1.Org)
	w.alex = fx.TrustUser(t, w.a1.Org, "super_clinician")
	w.ada = fx.TrustUser(t, w.a1.Org, "admin")
	w.bea = fx.TrustUser(t, w.b1.Org, "super_clinician")
	w.alexC, w.adaC, w.beaC = c.LoginAs(t, w.alex), c.LoginAs(t, w.ada), c.LoginAs(t, w.bea)
	w.cara = c.LoginAs(t, fx.User(t, w.a1, "clinician"))
	w.sue = c.LoginAs(t, fx.User(t, w.a1, "super_clinician"))
	w.val = c.LoginAs(t, fx.User(t, w.a2, "clinician"))
	w.trustClin = c.LoginAs(t, fx.TrustUser(t, w.a1.Org, "clinician"))
	w.hq = fx.PublishedHQ(t, w.a1, w.alex, "basic.json")
	w.hq2 = fx.PublishedHQ(t, w.a1, w.alex, "basic.json")
	w.draftHQ = fx.DraftHQ(t, w.a1, w.alex, "basic.json")
	w.otherHQ = fx.PublishedHQ(t, w.b1, w.bea, "basic.json")
	return w
}

func proceduresPath(org uuid.UUID, p string) string { return orgPath(org, "/procedures"+p) }

// addProcedure creates a procedure as Alex (or Bea, for trust B) and returns its id.
func (w *procedureWorld) addProcedure(t *testing.T, c *apitest.Client, org uuid.UUID, body map[string]any) uuid.UUID {
	t.Helper()
	r := c.Do("POST", proceduresPath(org, ""), body)
	r.Want(t, http.StatusCreated)
	var got struct{ ID uuid.UUID }
	r.Decode(t, &got)
	return got.ID
}

func listProcedures(t *testing.T, c *apitest.Client, path string) []procedureView {
	t.Helper()
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	var got []procedureView
	r.Decode(t, &got)
	return got
}

func procedureNames(ps []procedureView) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}

func findProcedure(ps []procedureView, name string) *procedureView {
	for i := range ps {
		if ps[i].Name == name {
			return &ps[i]
		}
	}
	return nil
}

func TestListProcedures(t *testing.T) {
	t.Parallel()
	w := newProcedureWorld(t)
	hip := w.addProcedure(t, w.alexC, w.a1.Org, map[string]any{"name": "Hip replacement", "code": "HIP-1",
		"specialty": "Orthopaedics", "defaultQuestionnaireId": w.hq.ID})
	w.addProcedure(t, w.alexC, w.a1.Org, map[string]any{"name": "Cataract", "hospitalId": w.a1.ID, "defaultQuestionnaireId": w.hq2.ID})
	w.addProcedure(t, w.alexC, w.a1.Org, map[string]any{"name": "Hernia repair", "specialty": "General surgery"})
	old := w.addProcedure(t, w.alexC, w.a1.Org, map[string]any{"name": "Appendix", "defaultQuestionnaireId": w.hq.ID})
	w.alexC.Do("PATCH", proceduresPath(w.a1.Org, "/"+old.String()), map[string]any{"retired": true}).Want(t, http.StatusNoContent)
	w.addProcedure(t, w.beaC, w.b1.Org, map[string]any{"name": "B's own"})

	t.Run("happy path: a whole-trust member sees the whole list, retired last", func(t *testing.T) {
		got := listProcedures(t, w.trustClin, proceduresPath(w.a1.Org, ""))
		if names := procedureNames(got); !slices.Equal(names, []string{"Cataract", "Hernia repair", "Hip replacement", "Appendix"}) {
			t.Errorf("procedures = %q", names)
		}
		p := findProcedure(got, "Hip replacement")
		if p == nil || p.ID != hip || p.Code != "HIP-1" || p.Specialty != "Orthopaedics" || p.HospitalID != nil ||
			p.DefaultQuestionnaireID == nil || *p.DefaultQuestionnaireID != w.hq.ID || p.DefaultQuestionnaireName == nil ||
			*p.DefaultQuestionnaireName != w.hq.Name || p.Retired || string(p.InfoContent) != "[]" || string(p.PromsPlan) != "[]" {
			t.Errorf("hip = %+v", p)
		}
		if c := findProcedure(got, "Cataract"); c == nil || c.HospitalID == nil || *c.HospitalID != w.a1.ID || c.HospitalName == nil {
			t.Errorf("cataract = %+v, want A1's own", c)
		}
		if a := findProcedure(got, "Appendix"); a == nil || !a.Retired {
			t.Errorf("appendix = %+v, want retired", a)
		}
	})
	t.Run("happy path: a member of one hospital sees the trust's and that hospital's", func(t *testing.T) {
		if names := procedureNames(listProcedures(t, w.val, proceduresPath(w.a1.Org, ""))); !slices.Equal(names, []string{"Hernia repair", "Hip replacement", "Appendix"}) {
			t.Errorf("Val (A2) sees %q, want no A1-only procedure", names)
		}
		if names := procedureNames(listProcedures(t, w.cara, proceduresPath(w.a1.Org, ""))); !slices.Equal(names, []string{"Cataract", "Hernia repair", "Hip replacement", "Appendix"}) {
			t.Errorf("Cara (A1) sees %q", names)
		}
	})
	t.Run("happy path: ?hospital= is what a new episode there is offered, with its default HQ", func(t *testing.T) {
		at1 := listProcedures(t, w.cara, proceduresPath(w.a1.Org, "?hospital="+w.a1.ID.String()))
		if names := procedureNames(at1); !slices.Equal(names, []string{"Cataract", "Hernia repair", "Hip replacement"}) {
			t.Errorf("offered at A1 = %q, want no retired one", names)
		}
		if p := findProcedure(at1, "Hip replacement"); p == nil || p.DefaultVersionID == nil || *p.DefaultVersionID != w.hq.VersionID {
			t.Errorf("hip at A1 = %+v, want its HQ's published version", p)
		}
		if p := findProcedure(at1, "Hernia repair"); p == nil || p.DefaultVersionID != nil {
			t.Errorf("hernia at A1 = %+v, want no default (the hospital has none)", p)
		}
		if names := procedureNames(listProcedures(t, w.val, proceduresPath(w.a1.Org, "?hospital="+w.a2.ID.String()))); !slices.Equal(names, []string{"Hernia repair", "Hip replacement"}) {
			t.Errorf("offered at A2 = %q, want no A1-only procedure", names)
		}
	})
	t.Run("a procedure with no default HQ falls back to the hospital's default HQ (S10's defaultHq)", func(t *testing.T) {
		w.setDefaultHQ(t, `UPDATE hospitals SET settings = jsonb_build_object('defaultHq', $2::text) WHERE id = $1`, w.a1.ID, w.hq2.ID)
		at1 := listProcedures(t, w.cara, proceduresPath(w.a1.Org, "?hospital="+w.a1.ID.String()))
		if p := findProcedure(at1, "Hernia repair"); p == nil || p.DefaultVersionID == nil || *p.DefaultVersionID != w.hq2.VersionID {
			t.Errorf("hernia at A1 = %+v, want A1's default HQ", p)
		}
		if p := findProcedure(at1, "Hip replacement"); p == nil || p.DefaultVersionID == nil || *p.DefaultVersionID != w.hq.VersionID {
			t.Errorf("hip at A1 = %+v, want its own default first", p)
		}
		// The trust's value, which A2 follows.
		w.setDefaultHQ(t, `UPDATE orgs SET settings = jsonb_build_object('defaultHq', $2::text) WHERE id = $1`, w.a1.Org, w.hq.ID)
		at2 := listProcedures(t, w.val, proceduresPath(w.a1.Org, "?hospital="+w.a2.ID.String()))
		if p := findProcedure(at2, "Hernia repair"); p == nil || p.DefaultVersionID == nil || *p.DefaultVersionID != w.hq.VersionID {
			t.Errorf("hernia at A2 = %+v, want the trust's default HQ", p)
		}
	})
	t.Run("a default HQ with no published version now falls back too", func(t *testing.T) {
		if _, err := w.c.DB.Exec(t.Context(), `UPDATE questionnaire_versions SET status = 'retired' WHERE id = $1`, w.hq2.VersionID); err != nil {
			t.Fatal(err)
		}
		// Cataract's own default and A1's are that questionnaire: no default at A1, the clinician chooses.
		at1 := listProcedures(t, w.cara, proceduresPath(w.a1.Org, "?hospital="+w.a1.ID.String()))
		if p := findProcedure(at1, "Cataract"); p == nil || p.DefaultVersionID != nil {
			t.Errorf("cataract at A1 = %+v, want no default", p)
		}
		if p := findProcedure(at1, "Hernia repair"); p == nil || p.DefaultVersionID != nil {
			t.Errorf("hernia at A1 = %+v, want no default", p)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.c.Do("GET", proceduresPath(w.a1.Org, ""), nil), http.StatusUnauthorized)
	})
	t.Run("role: reading needs no role, but a member of one hospital can't ask for another's", func(t *testing.T) {
		refused(t, w.val.Do("GET", proceduresPath(w.a1.Org, "?hospital="+w.a1.ID.String()), nil), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.beaC.Do("GET", proceduresPath(w.a1.Org, ""), nil), http.StatusNotFound)
		if names := procedureNames(listProcedures(t, w.beaC, proceduresPath(w.b1.Org, ""))); !slices.Equal(names, []string{"B's own"}) {
			t.Errorf("trust B's list = %q, want only its own", names)
		}
		refused(t, w.beaC.Do("GET", proceduresPath(w.b1.Org, "?hospital="+w.a1.ID.String()), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.alexC.Do("GET", proceduresPath(w.a1.Org, "?hospital=abc"), nil), http.StatusBadRequest)
		refused(t, w.alexC.Do("GET", proceduresPath(w.a1.Org, "?hospital="+uuid.NewString()), nil), http.StatusNotFound)
		refused(t, w.alexC.Do("GET", "/api/o/nope/procedures", nil), http.StatusNotFound)
	})
}

func (w *procedureWorld) setDefaultHQ(t *testing.T, sql string, id, hq uuid.UUID) {
	t.Helper()
	if _, err := w.c.DB.Exec(t.Context(), sql, id, hq.String()); err != nil {
		t.Fatal(err)
	}
}

// procedureRow is a procedures row as stored.
type procedureRow struct {
	Name, Code, Specialty string
	HospitalID, DefaultQ  *uuid.UUID
	Retired               bool
	UpdatedBy             *uuid.UUID
}

func readProcedure(t *testing.T, w *procedureWorld, id uuid.UUID) procedureRow {
	t.Helper()
	var p procedureRow
	if err := w.c.DB.QueryRow(t.Context(), `SELECT name, code, specialty, hospital_id, default_questionnaire_id, retired, updated_by
		FROM procedures WHERE id = $1`, id).Scan(&p.Name, &p.Code, &p.Specialty, &p.HospitalID, &p.DefaultQ, &p.Retired, &p.UpdatedBy); err != nil {
		t.Fatalf("procedure %s: %v", id, err)
	}
	return p
}

// auditActions are the audit rows' actions written for the trust, in order.
func auditActionsIn(t *testing.T, w *procedureWorld, org uuid.UUID) []string {
	t.Helper()
	rows, err := w.c.DB.Query(t.Context(), `SELECT action || ' ' || status FROM audit_log WHERE org_id = $1 ORDER BY id`, org)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func TestCreateProcedure(t *testing.T) {
	t.Parallel()
	w := newProcedureWorld(t)
	create := func(c *apitest.Client, org uuid.UUID, body any) *apitest.Resp {
		return c.Do("POST", proceduresPath(org, ""), body)
	}
	good := map[string]any{"name": " Total hip replacement ", "code": " HIP-01 ", "specialty": " Orthopaedics ", "defaultQuestionnaireId": w.hq.ID}
	procedures := func(t *testing.T) int { return count(t, w.c.DB, "SELECT count(*) FROM procedures") }

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, create(w.c, w.a1.Org, good), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician, and a super clinician of one hospital only", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.cara, w.trustClin, w.sue} {
			refused(t, create(c, w.a1.Org, good), http.StatusForbidden)
		}
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, create(w.beaC, w.a1.Org, good), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for name, bad := range map[string]any{
			"no name":                       map[string]any{"name": "  "},
			"a name too long":               map[string]any{"name": strings.Repeat("x", 201)},
			"a code too long":               map[string]any{"name": "X", "code": strings.Repeat("x", 51)},
			"a specialty too long":          map[string]any{"name": "X", "specialty": strings.Repeat("x", 101)},
			"a default HQ of another trust": map[string]any{"name": "X", "defaultQuestionnaireId": w.otherHQ.ID},
			"a draft default HQ":            map[string]any{"name": "X", "defaultQuestionnaireId": w.draftHQ.ID},
			"a default HQ that isn't an id": map[string]any{"name": "X", "defaultQuestionnaireId": "abc"},
			"another trust's hospital":      map[string]any{"name": "X", "hospitalId": w.b1.ID},
			"a hospital that isn't an id":   map[string]any{"name": "X", "hospitalId": 7},
			"unreadable":                    `{"name":`,
		} {
			r := create(w.alexC, w.a1.Org, bad)
			refused(t, r, http.StatusBadRequest)
			t.Logf("%s: %s", name, r.Message())
		}
		if msg := create(w.alexC, w.a1.Org, map[string]any{"name": "X", "defaultQuestionnaireId": w.draftHQ.ID}).Message(); !strings.Contains(msg, "published") {
			t.Errorf("draft HQ message = %q, want it to say published", msg)
		}
	})
	if n := procedures(t); n != 0 {
		t.Fatalf("refusals wrote %d procedures", n)
	}
	t.Run("happy path: trimmed, for every hospital, with its default HQ, audited", func(t *testing.T) {
		r := create(w.alexC, w.a1.Org, good)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		p := readProcedure(t, w, got.ID)
		if p.Name != "Total hip replacement" || p.Code != "HIP-01" || p.Specialty != "Orthopaedics" || p.HospitalID != nil ||
			p.DefaultQ == nil || *p.DefaultQ != w.hq.ID || p.Retired || p.UpdatedBy == nil || *p.UpdatedBy != w.alex.ID {
			t.Errorf("procedure = %+v", p)
		}
		if got := auditActionsIn(t, w, w.a1.Org); !slices.Contains(got, "procedure.create 201") {
			t.Errorf("audit = %q, want procedure.create 201", got)
		}
	})
	t.Run("happy path: an admin, one hospital's own, no code or default HQ", func(t *testing.T) {
		id := w.addProcedure(t, w.adaC, w.a1.Org, map[string]any{"name": "Cataract", "hospitalId": w.a1.ID})
		if p := readProcedure(t, w, id); p.HospitalID == nil || *p.HospitalID != w.a1.ID || p.Code != "" || p.DefaultQ != nil {
			t.Errorf("procedure = %+v", p)
		}
	})
	t.Run("a name the trust already has, in any case and retired ones included, is refused; another trust may", func(t *testing.T) {
		r := create(w.alexC, w.a1.Org, map[string]any{"name": "TOTAL HIP REPLACEMENT"})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "already") {
			t.Errorf("message = %q", r.Message())
		}
		w.addProcedure(t, w.beaC, w.b1.Org, map[string]any{"name": "Total hip replacement"})
	})
}

func TestUpdateProcedure(t *testing.T) {
	t.Parallel()
	w := newProcedureWorld(t)
	hip := w.addProcedure(t, w.alexC, w.a1.Org, map[string]any{"name": "Hip", "code": "H1", "specialty": "Ortho", "defaultQuestionnaireId": w.hq.ID})
	w.addProcedure(t, w.alexC, w.a1.Org, map[string]any{"name": "Knee"})
	theirs := w.addProcedure(t, w.beaC, w.b1.Org, map[string]any{"name": "Theirs"})
	path := proceduresPath(w.a1.Org, "/"+hip.String())
	patch := func(c *apitest.Client, p string, body any) *apitest.Resp { return c.Do("PATCH", p, body) }
	before := readProcedure(t, w, hip)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, patch(w.c, path, map[string]any{"name": "New"}), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician, and a super clinician of one hospital only", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.cara, w.trustClin, w.sue} {
			refused(t, patch(c, path, map[string]any{"retired": true}), http.StatusForbidden)
		}
	})
	t.Run("another trust: not a member, or its procedure by this trust's path", func(t *testing.T) {
		refused(t, patch(w.beaC, path, map[string]any{"name": "Mine"}), http.StatusNotFound)
		refused(t, patch(w.alexC, proceduresPath(w.a1.Org, "/"+theirs.String()), map[string]any{"name": "Mine"}), http.StatusNotFound)
		refused(t, patch(w.beaC, proceduresPath(w.b1.Org, "/"+hip.String()), map[string]any{"name": "Mine"}), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{
			map[string]any{"name": ""},
			map[string]any{"name": nil},
			map[string]any{"retired": "yes"},
			map[string]any{"defaultQuestionnaireId": w.draftHQ.ID},
			map[string]any{"defaultQuestionnaireId": w.otherHQ.ID},
			map[string]any{"hospitalId": w.b1.ID},
			`{"name":`,
		} {
			refused(t, patch(w.alexC, path, bad), http.StatusBadRequest)
		}
		refused(t, patch(w.alexC, path, map[string]any{"name": " knee "}), http.StatusConflict)
		refused(t, patch(w.alexC, proceduresPath(w.a1.Org, "/abc"), map[string]any{"name": "X"}), http.StatusNotFound)
		refused(t, patch(w.alexC, proceduresPath(w.a1.Org, "/"+uuid.NewString()), map[string]any{"name": "X"}), http.StatusNotFound)
	})
	if after := readProcedure(t, w, hip); !reflect.DeepEqual(after, before) {
		t.Fatalf("refusals changed the procedure: %+v, was %+v", after, before)
	}
	t.Run("happy path: fields sent change, the rest are kept; audited", func(t *testing.T) {
		patch(w.adaC, path, map[string]any{"name": " Hip replacement ", "specialty": "Orthopaedics", "hospitalId": w.a2.ID}).Want(t, http.StatusNoContent)
		p := readProcedure(t, w, hip)
		if p.Name != "Hip replacement" || p.Code != "H1" || p.Specialty != "Orthopaedics" || p.HospitalID == nil || *p.HospitalID != w.a2.ID ||
			p.DefaultQ == nil || *p.DefaultQ != w.hq.ID || p.UpdatedBy == nil || *p.UpdatedBy != w.ada.ID {
			t.Errorf("procedure = %+v", p)
		}
		if got := auditActionsIn(t, w, w.a1.Org); !slices.Contains(got, "procedure.update 204") {
			t.Errorf("audit = %q, want procedure.update 204", got)
		}
	})
	t.Run("happy path: null clears the hospital and the default HQ; another default", func(t *testing.T) {
		patch(w.alexC, path, map[string]any{"hospitalId": nil, "defaultQuestionnaireId": nil}).Want(t, http.StatusNoContent)
		if p := readProcedure(t, w, hip); p.HospitalID != nil || p.DefaultQ != nil {
			t.Errorf("procedure = %+v, want both cleared", p)
		}
		patch(w.alexC, path, map[string]any{"defaultQuestionnaireId": w.hq2.ID, "code": ""}).Want(t, http.StatusNoContent)
		if p := readProcedure(t, w, hip); p.DefaultQ == nil || *p.DefaultQ != w.hq2.ID || p.Code != "" {
			t.Errorf("procedure = %+v", p)
		}
	})
	t.Run("happy path: retire and bring back, even when its default HQ is no longer published", func(t *testing.T) {
		if _, err := w.c.DB.Exec(t.Context(), `UPDATE questionnaire_versions SET status = 'retired' WHERE id = $1`, w.hq2.VersionID); err != nil {
			t.Fatal(err)
		}
		patch(w.alexC, path, map[string]any{"retired": true}).Want(t, http.StatusNoContent)
		if p := readProcedure(t, w, hip); !p.Retired || p.Name != "Hip replacement" || p.DefaultQ == nil {
			t.Errorf("procedure = %+v, want retired and nothing else changed", p)
		}
		patch(w.alexC, path, map[string]any{"retired": false}).Want(t, http.StatusNoContent)
		if p := readProcedure(t, w, hip); p.Retired {
			t.Errorf("procedure = %+v, want back in use", p)
		}
	})
}
