//go:build integration

package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
)

// The superadmin console (plan C1, session S10; ORG-01…05, A-21, OPS-06). Only platform admins
// reach /api/admin/…: anyone else signed in gets 403, and nobody signed in 401. The platform admin
// has no memberships, so no trust's patient data is reachable (OPS-06).

type adminWorld struct {
	c      *apitest.Client // not signed in
	fx     *apitest.Fixtures
	a1, b1 apitest.Hospital

	sam, ada, cara  apitest.User // the platform admin; trust A's admin; a clinician at A1
	samC, adaC, caC *apitest.Client
}

func newAdminWorld(t *testing.T) *adminWorld {
	t.Helper()
	c, fx := apitest.New(t)
	w := &adminWorld{c: c, fx: fx, a1: fx.Hospital(t), b1: fx.Hospital(t)}
	w.sam = fx.PlatformAdmin(t)
	w.ada = fx.TrustUser(t, w.a1.Org, "admin")
	w.cara = fx.User(t, w.a1, "clinician")
	w.samC, w.adaC, w.caC = c.LoginAs(t, w.sam), c.LoginAs(t, w.ada), c.LoginAs(t, w.cara)
	return w
}

func adminPath(p string) string { return "/api/admin/orgs" + p }

func adminOrgPath(org uuid.UUID, p string) string { return adminPath("/" + org.String() + p) }

type adminHospital struct {
	ID              uuid.UUID
	Name, Address   string
	TimeZone        string
	OwnTimeZone     *string
	Status          string
	Phone, Email    string
	LogoURL         string `json:"logoUrl"`
	OptIns          map[string]bool
	EffectiveOptIns map[string]bool
}

type adminTrust struct {
	ID                                      uuid.UUID
	Name, Code, Status                      string
	ContactName, ContactEmail, ContactPhone string
	LogoURL                                 string `json:"logoUrl"`
	OptIns                                  map[string]bool
	EffectiveOptIns                         map[string]bool
	Hospitals                               []adminHospital
	Admins                                  []struct {
		UserID      uuid.UUID
		Name, Email string
		Status      string
	}
	OptInDefs       []struct{ Key, Label string }
	ExistingAccount bool
}

func (a *adminTrust) hospital(id uuid.UUID) *adminHospital {
	for i := range a.Hospitals {
		if a.Hospitals[i].ID == id {
			return &a.Hospitals[i]
		}
	}
	return nil
}

func newTrustBody(code, adminEmail string) map[string]any {
	return map[string]any{
		"name": "St Elsewhere Trust", "code": code,
		"contactName": "Pat Contact", "contactEmail": "pat.contact@elsewhere.example", "contactPhone": "0100 000 0000",
		"logoUrl": "https://cdn.elsewhere.example/logo.png", "adminEmail": adminEmail, "adminName": "Ann Admin",
		"optIns": map[string]any{"feature.proms": true, "instrument.eq5d5l": true},
	}
}

func createTrust(t *testing.T, w *adminWorld, code, adminEmail string) adminTrust {
	t.Helper()
	r := w.samC.Do("POST", adminPath(""), newTrustBody(code, adminEmail))
	r.Want(t, http.StatusCreated)
	var got adminTrust
	r.Decode(t, &got)
	return got
}

func addHospital(t *testing.T, w *adminWorld, org uuid.UUID, body map[string]any) adminTrust {
	t.Helper()
	r := w.samC.Do("POST", adminOrgPath(org, "/hospitals"), body)
	r.Want(t, http.StatusCreated)
	var got adminTrust
	r.Decode(t, &got)
	return got
}

func TestAdminListTrusts(t *testing.T) {
	w := newAdminWorld(t)
	w.fx.HospitalIn(t, w.a1.Org)

	t.Run("happy path: every trust, with its hospitals counted and no patient data", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a1, w.ada, "basic.json")
		w.fx.Episode(t, hq, w.ada)
		r := w.samC.Do("GET", adminPath(""), nil)
		r.Want(t, http.StatusOK)
		var rows []struct {
			ID                 uuid.UUID
			Name, Code, Status string
			HospitalCount      int
			CreatedAt          time.Time
		}
		r.Decode(t, &rows)
		counts := map[uuid.UUID]int{}
		for _, row := range rows {
			counts[row.ID] = row.HospitalCount
			if row.Status != "active" || row.Code == "" || row.CreatedAt.IsZero() {
				t.Errorf("row %+v", row)
			}
		}
		if counts[w.a1.Org] != 2 || counts[w.b1.Org] != 1 {
			t.Errorf("counts %v", counts)
		}
		if strings.Contains(string(r.Body), "Patient") || strings.Contains(string(r.Body), "1970-01-01") {
			t.Errorf("patient data in %s", r.Body)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("GET", adminPath(""), nil).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: a trust admin and a clinician get 403", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.adaC, w.caC} {
			r := c.Do("GET", adminPath(""), nil)
			r.Want(t, http.StatusForbidden)
			if r.Message() == "" {
				t.Errorf("no message: %s", r)
			}
		}
	})
	t.Run("a platform admin row removed takes effect on the next request", func(t *testing.T) {
		other := w.fx.PlatformAdmin(t)
		oc := w.c.LoginAs(t, other)
		oc.Do("GET", adminPath(""), nil).Want(t, http.StatusOK)
		if _, err := w.c.DB.Exec(t.Context(), `DELETE FROM platform_admins WHERE user_id = $1`, other.ID); err != nil {
			t.Fatal(err)
		}
		oc.Do("GET", adminPath(""), nil).Want(t, http.StatusForbidden)
	})
}

func TestAdminCreateTrust(t *testing.T) {
	w := newAdminWorld(t)

	t.Run("happy path: the trust, its settings, and its first admin invited (S09)", func(t *testing.T) {
		got := createTrust(t, w, " se-1 ", "Ann.Admin@Elsewhere.Example")
		if got.Name != "St Elsewhere Trust" || got.Code != "SE-1" || got.Status != "active" ||
			got.ContactName != "Pat Contact" || got.ContactEmail != "pat.contact@elsewhere.example" ||
			got.ContactPhone != "0100 000 0000" || got.LogoURL != "https://cdn.elsewhere.example/logo.png" {
			t.Errorf("trust %+v", got)
		}
		if !got.OptIns["feature.proms"] || !got.EffectiveOptIns["instrument.eq5d5l"] || got.EffectiveOptIns["instrument.oxfordHip"] ||
			!got.EffectiveOptIns["feature.assessments"] || len(got.OptInDefs) != 7 {
			t.Errorf("opt-ins %+v %+v (%d defs)", got.OptIns, got.EffectiveOptIns, len(got.OptInDefs))
		}
		if len(got.Hospitals) != 0 || len(got.Admins) != 1 || got.Admins[0].Email != "ann.admin@elsewhere.example" ||
			got.Admins[0].Name != "Ann Admin" || got.Admins[0].Status != "invited" || got.ExistingAccount {
			t.Errorf("hospitals %+v admins %+v", got.Hospitals, got.Admins)
		}
		var settings string
		if err := w.c.DB.QueryRow(t.Context(), `SELECT settings::text FROM orgs WHERE id = $1`, got.ID).Scan(&settings); err != nil {
			t.Fatal(err)
		}
		var stored map[string]any
		_ = json.Unmarshal([]byte(settings), &stored)
		if stored["branding.logoUrl"] != "https://cdn.elsewhere.example/logo.png" || stored["feature.proms"] != true || len(stored) != 3 {
			t.Errorf("stored settings %s", settings)
		}
		admin := got.Admins[0].UserID
		if n := count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2 AND hospital_id IS NULL AND role = 'admin'`, admin, got.ID); n != 1 {
			t.Errorf("memberships %d", n)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM staff_invites WHERE user_id = $1 AND org_id = $2 AND invited_by = $3
			AND accepted_at IS NULL AND expires_at = created_at + interval '7 days'`, admin, got.ID, w.sam.ID); n != 1 {
			t.Errorf("invites %d", n)
		}
		calls := w.c.Cognito.Calls()
		if len(calls) != 1 || calls[0] != (cognito.Call{Op: "CreateUser", Email: "ann.admin@elsewhere.example"}) {
			t.Errorf("cognito %+v", calls)
		}

		// The new admin signs in: their trust is listed, though it has no hospitals yet.
		annC := w.c.LoginAs(t, apitest.User{ID: admin, Name: "Ann Admin"})
		r := annC.Do("GET", "/api/me", nil)
		r.Want(t, http.StatusOK)
		var me struct {
			Trusts []struct {
				ID        uuid.UUID
				Role      *string
				Hospitals []json.RawMessage
			}
		}
		r.Decode(t, &me)
		if len(me.Trusts) != 1 || me.Trusts[0].ID != got.ID || me.Trusts[0].Role == nil || *me.Trusts[0].Role != "admin" ||
			me.Trusts[0].Hospitals == nil || len(me.Trusts[0].Hospitals) != 0 {
			t.Errorf("me %s", r.Body)
		}
	})
	t.Run("an existing account only gets the membership (STF-02)", func(t *testing.T) {
		before := len(w.c.Cognito.Calls())
		users := count(t, w.c.DB, `SELECT count(*) FROM users`)
		r := w.samC.Do("POST", adminPath(""), newTrustBody("SE-2", "  "+strings.ToUpper(adminEmailOf(t, w, w.cara))+" "))
		r.Want(t, http.StatusCreated)
		var got adminTrust
		r.Decode(t, &got)
		if !got.ExistingAccount || len(got.Admins) != 1 || got.Admins[0].UserID != w.cara.ID || got.Admins[0].Status != "active" {
			t.Errorf("got %+v", got)
		}
		if count(t, w.c.DB, `SELECT count(*) FROM users`) != users || len(w.c.Cognito.Calls()) != before {
			t.Error("a second account or a Cognito call")
		}
	})
	t.Run("a duplicate code is refused, whatever its case", func(t *testing.T) {
		orgs := count(t, w.c.DB, `SELECT count(*) FROM orgs`)
		r := w.samC.Do("POST", adminPath(""), newTrustBody("se-1", "someone@elsewhere.example"))
		r.Want(t, http.StatusConflict)
		if !strings.Contains(r.Message(), "SE-1") {
			t.Errorf("message %q", r.Message())
		}
		if count(t, w.c.DB, `SELECT count(*) FROM orgs`) != orgs || count(t, w.c.DB, `SELECT count(*) FROM users WHERE email = 'someone@elsewhere.example'`) != 0 {
			t.Error("something was written")
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("POST", adminPath(""), newTrustBody("SE-3", "x@elsewhere.example")).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: a trust admin can't create a trust", func(t *testing.T) {
		w.adaC.Do("POST", adminPath(""), newTrustBody("SE-4", "x@elsewhere.example")).Want(t, http.StatusForbidden)
		if count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE code = 'SE-4'`) != 0 {
			t.Error("created")
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		cases := map[string]func(b map[string]any){
			"no name":              func(b map[string]any) { b["name"] = "  " },
			"a long name":          func(b map[string]any) { b["name"] = strings.Repeat("a", 201) },
			"no code":              func(b map[string]any) { b["code"] = "" },
			"a one-letter code":    func(b map[string]any) { b["code"] = "A" },
			"a code with a space":  func(b map[string]any) { b["code"] = "A B" },
			"a long code":          func(b map[string]any) { b["code"] = "ABCDEFGHIJKLM" },
			"no contact name":      func(b map[string]any) { b["contactName"] = "" },
			"a bad contact email":  func(b map[string]any) { b["contactEmail"] = "nope" },
			"no contact email":     func(b map[string]any) { b["contactEmail"] = "" },
			"a bad contact phone":  func(b map[string]any) { b["contactPhone"] = "call me" },
			"a bad logo address":   func(b map[string]any) { b["logoUrl"] = "javascript:alert(1)" },
			"no admin email":       func(b map[string]any) { b["adminEmail"] = "" },
			"a bad admin email":    func(b map[string]any) { b["adminEmail"] = "ann@" },
			"a setting not opt-in": func(b map[string]any) { b["optIns"] = map[string]any{"contact.phone": true} },
			"an unknown opt-in":    func(b map[string]any) { b["optIns"] = map[string]any{"feature.nope": true} },
			"an opt-in not on/off": func(b map[string]any) { b["optIns"] = map[string]any{"feature.proms": "yes"} },
		}
		orgs := count(t, w.c.DB, `SELECT count(*) FROM orgs`)
		for name, change := range cases {
			b := newTrustBody("SE-9", "x9@elsewhere.example")
			change(b)
			r := w.samC.Do("POST", adminPath(""), b)
			if r.Code != http.StatusBadRequest || r.Message() == "" {
				t.Errorf("%s: got %s, want 400 with a message", name, r)
			}
		}
		w.samC.Do("POST", adminPath(""), `{"name":`).Want(t, http.StatusBadRequest)
		if count(t, w.c.DB, `SELECT count(*) FROM orgs`) != orgs {
			t.Error("a trust was created")
		}
	})
	t.Run("Cognito failing leaves nothing behind", func(t *testing.T) {
		w.c.Cognito.FailWith(errors.New("cognito is down"))
		defer w.c.Cognito.FailWith(nil)
		w.samC.Do("POST", adminPath(""), newTrustBody("SE-5", "new5@elsewhere.example")).Want(t, http.StatusInternalServerError)
		if count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE code = 'SE-5'`) != 0 ||
			count(t, w.c.DB, `SELECT count(*) FROM users WHERE email = 'new5@elsewhere.example'`) != 0 {
			t.Error("something was written")
		}
	})
}

func adminEmailOf(t *testing.T, w *adminWorld, u apitest.User) string {
	t.Helper()
	var e string
	if err := w.c.DB.QueryRow(t.Context(), `SELECT email FROM users WHERE id = $1`, u.ID).Scan(&e); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestAdminGetTrust(t *testing.T) {
	w := newAdminWorld(t)
	t.Run("happy path: the trust, its hospitals and admins, and no patient data", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a1, w.ada, "basic.json")
		w.fx.Episode(t, hq, w.ada)
		r := w.samC.Do("GET", adminOrgPath(w.a1.Org, ""), nil)
		r.Want(t, http.StatusOK)
		var got adminTrust
		r.Decode(t, &got)
		if got.ID != w.a1.Org || len(got.Hospitals) != 1 || got.Hospitals[0].ID != w.a1.ID || len(got.Admins) != 1 ||
			got.Admins[0].UserID != w.ada.ID || got.Admins[0].Status != "active" {
			t.Errorf("got %s", r.Body)
		}
		if got.Hospitals[0].TimeZone == "" || got.Hospitals[0].OwnTimeZone != nil {
			t.Errorf("time zone %+v", got.Hospitals[0])
		}
		for _, leak := range []string{"Patient", "1970-01-01", hq.Name, "episodes"} {
			if strings.Contains(string(r.Body), leak) {
				t.Errorf("%q in %s", leak, r.Body)
			}
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("GET", adminOrgPath(w.a1.Org, ""), nil).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: the trust's own admin gets 403", func(t *testing.T) {
		w.adaC.Do("GET", adminOrgPath(w.a1.Org, ""), nil).Want(t, http.StatusForbidden)
	})
	t.Run("unknown or malformed trust ids get 404", func(t *testing.T) {
		w.samC.Do("GET", adminOrgPath(uuid.New(), ""), nil).Want(t, http.StatusNotFound)
		w.samC.Do("GET", adminPath("/nope"), nil).Want(t, http.StatusNotFound)
	})
}

func hospitalBody(name string) map[string]any {
	return map[string]any{"name": name, "address": "1 High Street\nTown", "phone": "+44 100 000 0001",
		"email": "POA@Hospital.Example", "timeZone": "", "logoUrl": "", "optIns": map[string]any{}}
}

func TestAdminAddHospital(t *testing.T) {
	w := newAdminWorld(t)
	tr := createTrust(t, w, "AH-1", "ah1.admin@elsewhere.example")

	t.Run("happy path: name, address, patients' phone and email, time zone", func(t *testing.T) {
		b := hospitalBody("General Hospital")
		b["optIns"] = map[string]any{"feature.proms": false}
		got := addHospital(t, w, tr.ID, b)
		if len(got.Hospitals) != 1 {
			t.Fatalf("hospitals %+v", got.Hospitals)
		}
		h := got.Hospitals[0]
		if h.Name != "General Hospital" || h.Address != "1 High Street\nTown" || h.Phone != "+44 100 000 0001" ||
			h.Email != "poa@hospital.example" || h.Status != "active" || h.OwnTimeZone != nil || h.TimeZone == "" {
			t.Errorf("hospital %+v", h)
		}
		if h.EffectiveOptIns["feature.proms"] || !h.EffectiveOptIns["instrument.eq5d5l"] || len(h.OptIns) != 1 {
			t.Errorf("opt-ins %+v %+v", h.OptIns, h.EffectiveOptIns)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM hospitals WHERE org_id = $1 AND name = 'General Hospital' AND timezone IS NULL
			AND settings->>'contact.phone' = '+44 100 000 0001' AND settings->>'contact.email' = 'poa@hospital.example'`, tr.ID); n != 1 {
			t.Errorf("rows %d", n)
		}
		// The trust's first admin now has a hospital to work in.
		ann := apitest.User{ID: tr.Admins[0].UserID, Name: "Ann"}
		r := w.c.LoginAs(t, ann).Do("GET", "/api/me", nil)
		if !strings.Contains(string(r.Body), "General Hospital") {
			t.Errorf("me %s", r.Body)
		}
	})
	t.Run("a time zone of its own", func(t *testing.T) {
		b := hospitalBody("Northern Hospital")
		b["timeZone"] = "America/Toronto"
		got := addHospital(t, w, tr.ID, b)
		var h *adminHospital
		for i := range got.Hospitals {
			if got.Hospitals[i].Name == "Northern Hospital" {
				h = &got.Hospitals[i]
			}
		}
		if h == nil || h.TimeZone != "America/Toronto" || h.OwnTimeZone == nil || *h.OwnTimeZone != "America/Toronto" {
			t.Errorf("hospital %+v", h)
		}
	})
	t.Run("a duplicate name in the trust is refused, whatever its case", func(t *testing.T) {
		r := w.samC.Do("POST", adminOrgPath(tr.ID, "/hospitals"), hospitalBody("general HOSPITAL"))
		r.Want(t, http.StatusConflict)
		// Another trust may use the name.
		w.samC.Do("POST", adminOrgPath(w.b1.Org, "/hospitals"), hospitalBody("General Hospital")).Want(t, http.StatusCreated)
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("POST", adminOrgPath(tr.ID, "/hospitals"), hospitalBody("X1")).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: the trust's admin can't add hospitals", func(t *testing.T) {
		w.adaC.Do("POST", adminOrgPath(w.a1.Org, "/hospitals"), hospitalBody("X2")).Want(t, http.StatusForbidden)
	})
	t.Run("an unknown trust gets 404", func(t *testing.T) {
		w.samC.Do("POST", adminOrgPath(uuid.New(), "/hospitals"), hospitalBody("X3")).Want(t, http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		cases := map[string]func(b map[string]any){
			"no name":            func(b map[string]any) { b["name"] = " " },
			"a long name":        func(b map[string]any) { b["name"] = strings.Repeat("h", 201) },
			"no address":         func(b map[string]any) { b["address"] = "" },
			"a long address":     func(b map[string]any) { b["address"] = strings.Repeat("a", 501) },
			"a bad phone":        func(b map[string]any) { b["phone"] = "ring ring" },
			"a bad email":        func(b map[string]any) { b["email"] = "poa@" },
			"an unknown zone":    func(b map[string]any) { b["timeZone"] = "Mars/Olympus" },
			"the server's zone":  func(b map[string]any) { b["timeZone"] = "Local" },
			"a bad logo":         func(b map[string]any) { b["logoUrl"] = "ftp://x.example/logo.png" },
			"a non-opt-in":       func(b map[string]any) { b["optIns"] = map[string]any{"retentionYears": 3} },
			"an opt-in not bool": func(b map[string]any) { b["optIns"] = map[string]any{"feature.files": 1} },
		}
		before := count(t, w.c.DB, `SELECT count(*) FROM hospitals`)
		for name, change := range cases {
			b := hospitalBody("Invalid Hospital")
			change(b)
			if r := w.samC.Do("POST", adminOrgPath(tr.ID, "/hospitals"), b); r.Code != http.StatusBadRequest || r.Message() == "" {
				t.Errorf("%s: got %s, want 400", name, r)
			}
		}
		if count(t, w.c.DB, `SELECT count(*) FROM hospitals`) != before {
			t.Error("a hospital was added")
		}
	})
}

// patch sends a PATCH as the platform admin and decodes the trust.
func patchAdmin(t *testing.T, w *adminWorld, path string, body any, want int) adminTrust {
	t.Helper()
	r := w.samC.Do("PATCH", path, body)
	r.Want(t, want)
	var got adminTrust
	if want == http.StatusOK {
		r.Decode(t, &got)
	}
	return got
}

func TestAdminSuspendTrust(t *testing.T) {
	w := newAdminWorld(t)
	hq := w.fx.PublishedHQ(t, w.a1, w.ada, "basic.json")
	ep := w.fx.Episode(t, hq, w.ada)
	epSession := w.c.AsPatient(t, ep.Link, ep.DateOfBirth) // opened before the suspension
	// Someone in both trusts keeps trust B while A is suspended.
	both := w.fx.User(t, w.a1, "clinician")
	w.fx.TrustMember(t, both, w.b1.Org, "clinician")
	bothC := w.c.LoginAs(t, both)
	episodes := count(t, w.c.DB, `SELECT count(*) FROM episodes WHERE org_id = $1`, w.a1.Org)

	t.Run("suspended: no sign-in, no access, data kept (ORG-03)", func(t *testing.T) {
		got := patchAdmin(t, w, adminOrgPath(w.a1.Org, ""), map[string]any{"status": "suspended"}, http.StatusOK)
		if got.Status != "suspended" {
			t.Errorf("status %q", got.Status)
		}
		// Signed-in members are refused on their next request.
		w.caC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusNotFound)
		w.adaC.Do("GET", orgPath(w.a1.Org, "/staff"), nil).Want(t, http.StatusNotFound)
		// A new sign-in is refused, with a message and no cookie.
		r := w.c.Do("POST", "/api/dev/login", map[string]any{"userId": w.cara.ID})
		r.Want(t, http.StatusForbidden)
		if !strings.Contains(r.Message(), "suspended") || r.Header("Set-Cookie") != "" {
			t.Errorf("got %s %q", r, r.Header("Set-Cookie"))
		}
		// The patient's link is closed too, and so is a session they already have.
		w.c.Do("POST", "/api/p/links/"+ep.Link+"/continue", nil).Want(t, http.StatusForbidden)
		epSession.Do("GET", "/api/p/episodes/"+ep.ID.String(), nil).Want(t, http.StatusForbidden)
		// Someone in another trust still signs in and keeps it, without trust A.
		w.c.Do("POST", "/api/dev/login", map[string]any{"userId": both.ID}).Want(t, http.StatusNoContent)
		me := bothC.Do("GET", "/api/me", nil)
		if strings.Contains(string(me.Body), w.a1.Org.String()) || !strings.Contains(string(me.Body), w.b1.Org.String()) {
			t.Errorf("me %s", me.Body)
		}
		if count(t, w.c.DB, `SELECT count(*) FROM episodes WHERE org_id = $1`, w.a1.Org) != episodes ||
			count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE org_id = $1`, w.a1.Org) != 3 {
			t.Error("data was lost")
		}
	})
	t.Run("a platform admin who is also its staff still signs in, and can re-activate it (verification 1, finding 1)", func(t *testing.T) {
		vera := w.fx.PlatformAdmin(t)
		w.fx.TrustMember(t, vera, w.a1.Org, "admin")
		r := w.c.Do("POST", "/api/dev/login", map[string]any{"userId": vera.ID})
		r.Want(t, http.StatusNoContent)
		veraC := w.c.LoginAs(t, vera)
		me := veraC.Do("GET", "/api/me", nil)
		if strings.Contains(string(me.Body), w.a1.Org.String()) {
			t.Errorf("the suspended trust is still in /api/me: %s", me.Body)
		}
		veraC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusNotFound)
		veraC.Do("PATCH", adminOrgPath(w.a1.Org, ""), map[string]any{"status": "active"}).Want(t, http.StatusOK)
		veraC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusOK)
		if _, err := w.c.DB.Exec(t.Context(), `DELETE FROM memberships WHERE user_id = $1`, vera.ID); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("re-activated: they can again", func(t *testing.T) {
		patchAdmin(t, w, adminOrgPath(w.a1.Org, ""), map[string]any{"status": "active"}, http.StatusOK)
		w.c.Do("POST", "/api/dev/login", map[string]any{"userId": w.cara.ID}).Want(t, http.StatusNoContent)
		w.caC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusOK)
		w.c.Do("POST", "/api/p/links/"+ep.Link+"/continue", nil).Want(t, http.StatusOK)
		epSession.Do("GET", "/api/p/episodes/"+ep.ID.String(), nil).Want(t, http.StatusOK)
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("PATCH", adminOrgPath(w.a1.Org, ""), map[string]any{"status": "suspended"}).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: a trust admin can't suspend their trust", func(t *testing.T) {
		w.adaC.Do("PATCH", adminOrgPath(w.a1.Org, ""), map[string]any{"status": "suspended"}).Want(t, http.StatusForbidden)
		if count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND status = 'active'`, w.a1.Org) != 1 {
			t.Error("suspended")
		}
	})
	t.Run("an unknown trust gets 404", func(t *testing.T) {
		patchAdmin(t, w, adminOrgPath(uuid.New(), ""), map[string]any{"status": "suspended"}, http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, b := range []any{map[string]any{"status": "closed"}, map[string]any{}, map[string]any{"status": ""}, `{"status":`} {
			patchAdmin(t, w, adminOrgPath(w.a1.Org, ""), b, http.StatusBadRequest)
		}
	})
}

func TestAdminSuspendHospital(t *testing.T) {
	w := newAdminWorld(t)
	a2 := w.fx.HospitalIn(t, w.a1.Org)
	hq := w.fx.PublishedHQ(t, w.a1, w.ada, "basic.json")
	w.fx.Episode(t, hq, w.ada)
	w.fx.EpisodeAt(t, hq, a2, w.ada)
	path := adminOrgPath(w.a1.Org, "/hospitals/"+w.a1.ID.String())

	t.Run("suspended: its staff can't sign in or reach it; the trust's other hospitals carry on", func(t *testing.T) {
		got := patchAdmin(t, w, path, map[string]any{"status": "suspended"}, http.StatusOK)
		if h := got.hospital(w.a1.ID); h == nil || h.Status != "suspended" {
			t.Fatalf("hospital %+v", got.Hospitals)
		}
		w.caC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusNotFound)
		w.c.Do("POST", "/api/dev/login", map[string]any{"userId": w.cara.ID}).Want(t, http.StatusForbidden)
		// The whole-trust admin signs in, but can't open it, and its episodes leave "All hospitals".
		w.c.Do("POST", "/api/dev/login", map[string]any{"userId": w.ada.ID}).Want(t, http.StatusNoContent)
		w.adaC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusNotFound)
		w.adaC.Do("GET", a2.Path("/episodes"), nil).Want(t, http.StatusOK)
		all := w.adaC.Do("GET", orgPath(w.a1.Org, "/episodes"), nil)
		all.Want(t, http.StatusOK)
		if ids := episodeIDs(t, all); len(ids) != 1 {
			t.Errorf("all hospitals: %d episodes", len(ids))
		}
		me := w.adaC.Do("GET", "/api/me", nil)
		if strings.Contains(string(me.Body), w.a1.ID.String()) || !strings.Contains(string(me.Body), a2.ID.String()) {
			t.Errorf("me %s", me.Body)
		}
	})
	t.Run("a trust whose only hospital is suspended is left out, and its whole-trust staff can't sign in (verification 1, finding 2)", func(t *testing.T) {
		bAdmin := w.fx.TrustUser(t, w.b1.Org, "admin")
		bC := w.c.LoginAs(t, bAdmin)
		patchAdmin(t, w, adminOrgPath(w.b1.Org, "/hospitals/"+w.b1.ID.String()), map[string]any{"status": "suspended"}, http.StatusOK)
		me := bC.Do("GET", "/api/me", nil)
		me.Want(t, http.StatusOK)
		if strings.Contains(string(me.Body), w.b1.Org.String()) {
			t.Errorf("the trust is listed as if it had no hospitals yet: %s", me.Body)
		}
		r := w.c.Do("POST", "/api/dev/login", map[string]any{"userId": bAdmin.ID})
		r.Want(t, http.StatusForbidden)
		if !strings.Contains(r.Message(), "suspended") {
			t.Errorf("message %q", r.Message())
		}
		patchAdmin(t, w, adminOrgPath(w.b1.Org, "/hospitals/"+w.b1.ID.String()), map[string]any{"status": "active"}, http.StatusOK)
		w.c.Do("POST", "/api/dev/login", map[string]any{"userId": bAdmin.ID}).Want(t, http.StatusNoContent)
		if me := bC.Do("GET", "/api/me", nil); !strings.Contains(string(me.Body), w.b1.ID.String()) {
			t.Errorf("me %s", me.Body)
		}
	})
	t.Run("re-activated", func(t *testing.T) {
		patchAdmin(t, w, path, map[string]any{"status": "active"}, http.StatusOK)
		w.c.Do("POST", "/api/dev/login", map[string]any{"userId": w.cara.ID}).Want(t, http.StatusNoContent)
		w.caC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusOK)
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("PATCH", path, map[string]any{"status": "suspended"}).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role", func(t *testing.T) {
		w.adaC.Do("PATCH", path, map[string]any{"status": "suspended"}).Want(t, http.StatusForbidden)
	})
	t.Run("another trust's hospital under this trust's address gets 404", func(t *testing.T) {
		patchAdmin(t, w, adminOrgPath(w.a1.Org, "/hospitals/"+w.b1.ID.String()), map[string]any{"status": "suspended"}, http.StatusNotFound)
		patchAdmin(t, w, adminOrgPath(w.a1.Org, "/hospitals/nope"), map[string]any{"status": "suspended"}, http.StatusNotFound)
		if count(t, w.c.DB, `SELECT count(*) FROM hospitals WHERE id = $1 AND status = 'active'`, w.b1.ID) != 1 {
			t.Error("trust B's hospital was suspended")
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, b := range []any{map[string]any{"status": "gone"}, map[string]any{}, map[string]any{"optIns": map[string]any{"contact.email": "x"}}} {
			patchAdmin(t, w, path, b, http.StatusBadRequest)
		}
	})
}

func TestAdminOptIns(t *testing.T) {
	w := newAdminWorld(t)
	a2 := w.fx.HospitalIn(t, w.a1.Org)
	trust := adminOrgPath(w.a1.Org, "")
	hosp := func(h apitest.Hospital) string { return adminOrgPath(w.a1.Org, "/hospitals/"+h.ID.String()) }

	t.Run("trust off, with one hospital on (ORG-05)", func(t *testing.T) {
		patchAdmin(t, w, trust, map[string]any{"optIns": map[string]any{"feature.proms": false}}, http.StatusOK)
		got := patchAdmin(t, w, hosp(w.a1), map[string]any{"optIns": map[string]any{"feature.proms": true}}, http.StatusOK)
		if !got.hospital(w.a1.ID).EffectiveOptIns["feature.proms"] || got.hospital(a2.ID).EffectiveOptIns["feature.proms"] ||
			got.EffectiveOptIns["feature.proms"] {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("and the reverse: trust on, one hospital off; the other follows the trust", func(t *testing.T) {
		patchAdmin(t, w, trust, map[string]any{"optIns": map[string]any{"instrument.oxfordKnee": true}}, http.StatusOK)
		got := patchAdmin(t, w, hosp(a2), map[string]any{"optIns": map[string]any{"instrument.oxfordKnee": false}}, http.StatusOK)
		if got.hospital(a2.ID).EffectiveOptIns["instrument.oxfordKnee"] || !got.hospital(w.a1.ID).EffectiveOptIns["instrument.oxfordKnee"] {
			t.Errorf("got %+v", got)
		}
		if v, ok := got.hospital(a2.ID).OptIns["instrument.oxfordKnee"]; !ok || v {
			t.Errorf("a2's own opt-ins %+v", got.hospital(a2.ID).OptIns)
		}
	})
	t.Run("null puts a hospital back to following the trust, and the trust back to the default", func(t *testing.T) {
		got := patchAdmin(t, w, hosp(a2), map[string]any{"optIns": map[string]any{"instrument.oxfordKnee": nil}}, http.StatusOK)
		if _, ok := got.hospital(a2.ID).OptIns["instrument.oxfordKnee"]; ok || !got.hospital(a2.ID).EffectiveOptIns["instrument.oxfordKnee"] {
			t.Errorf("got %+v", got.hospital(a2.ID))
		}
		got = patchAdmin(t, w, trust, map[string]any{"optIns": map[string]any{"instrument.oxfordKnee": nil}}, http.StatusOK)
		if got.EffectiveOptIns["instrument.oxfordKnee"] || got.hospital(a2.ID).EffectiveOptIns["instrument.oxfordKnee"] {
			t.Errorf("got %+v", got)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND settings ? 'instrument.oxfordKnee'`, w.a1.Org); n != 0 {
			t.Error("the key is still stored")
		}
	})
	t.Run("status and opt-ins together", func(t *testing.T) {
		got := patchAdmin(t, w, hosp(a2), map[string]any{"status": "suspended", "optIns": map[string]any{"feature.files": false}}, http.StatusOK)
		if h := got.hospital(a2.ID); h.Status != "suspended" || h.EffectiveOptIns["feature.files"] {
			t.Errorf("got %+v", h)
		}
	})
	t.Run("invalid input: not an opt-in, or not on/off; nothing written", func(t *testing.T) {
		for _, b := range []map[string]any{
			{"optIns": map[string]any{"contact.phone": "0100"}},
			{"optIns": map[string]any{"feature.proms": "on"}},
			{"optIns": map[string]any{"feature.files": true, "nope": true}},
			{"optIns": map[string]any{}},
		} {
			patchAdmin(t, w, trust, b, http.StatusBadRequest)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND settings ? 'feature.files'`, w.a1.Org); n != 0 {
			t.Error("written")
		}
	})
}

func TestAdminInviteTrustAdmin(t *testing.T) {
	w := newAdminWorld(t)
	path := adminOrgPath(w.a1.Org, "/admins")

	t.Run("happy path: another whole-trust admin, invited as S09 does", func(t *testing.T) {
		r := w.samC.Do("POST", path, map[string]any{"email": "Second.Admin@trust-a.example", "name": "Second Admin"})
		r.Want(t, http.StatusCreated)
		var got adminTrust
		r.Decode(t, &got)
		if len(got.Admins) != 2 || got.ExistingAccount {
			t.Errorf("admins %+v", got.Admins)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM memberships m JOIN users u ON u.id = m.user_id
			WHERE u.email = 'second.admin@trust-a.example' AND m.org_id = $1 AND m.hospital_id IS NULL AND m.role = 'admin'`, w.a1.Org); n != 1 {
			t.Errorf("memberships %d", n)
		}
	})
	t.Run("someone who is already an admin of the whole trust gets 409", func(t *testing.T) {
		w.samC.Do("POST", path, map[string]any{"email": adminEmailOf(t, w, w.ada)}).Want(t, http.StatusConflict)
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("POST", path, map[string]any{"email": "x@trust-a.example"}).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: the trust's admin uses the Staff screen instead", func(t *testing.T) {
		w.adaC.Do("POST", path, map[string]any{"email": "y@trust-a.example"}).Want(t, http.StatusForbidden)
	})
	t.Run("an unknown trust gets 404", func(t *testing.T) {
		w.samC.Do("POST", adminOrgPath(uuid.New(), "/admins"), map[string]any{"email": "z@trust-a.example"}).Want(t, http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, b := range []any{map[string]any{"email": "nope"}, map[string]any{}, map[string]any{"email": "a@b.example", "name": strings.Repeat("n", 201)}, `{`} {
			w.samC.Do("POST", path, b).Want(t, http.StatusBadRequest)
		}
	})
}

func TestPlatformAdminSeesNoPatientData(t *testing.T) {
	w := newAdminWorld(t)
	hq := w.fx.PublishedHQ(t, w.a1, w.ada, "basic.json")
	ep := w.fx.Episode(t, hq, w.ada)
	// OPS-06: the platform admin has no membership, so every trust and hospital route is a 404.
	for _, p := range []string{
		orgPath(w.a1.Org, "/episodes"), w.a1.Path("/episodes"), w.a1.Path("/episodes/" + ep.ID.String()),
		w.a1.Path("/patients"), w.a1.Path("/questionnaires"), orgPath(w.a1.Org, "/staff"), orgPath(w.a1.Org, "/settings"),
	} {
		w.samC.Do("GET", p, nil).Want(t, http.StatusNotFound)
	}
	r := w.samC.Do("GET", "/api/me", nil)
	r.Want(t, http.StatusOK)
	var me struct {
		PlatformAdmin bool
		Trusts        []json.RawMessage
	}
	r.Decode(t, &me)
	if !me.PlatformAdmin || len(me.Trusts) != 0 {
		t.Errorf("me %s", r.Body)
	}
}

func TestAdminOptInDefs(t *testing.T) {
	w := newAdminWorld(t)
	t.Run("happy path: the features and instruments, for the New trust form", func(t *testing.T) {
		r := w.samC.Do("GET", "/api/admin/opt-ins", nil)
		r.Want(t, http.StatusOK)
		var defs []struct {
			Key, Label string
			OptIn      bool
			Default    bool
		}
		r.Decode(t, &defs)
		if len(defs) != 7 || defs[0].Key != "feature.assessments" || !defs[0].Default || defs[0].Label == "" {
			t.Errorf("defs %s", r.Body)
		}
		for _, d := range defs {
			if !d.OptIn {
				t.Errorf("%s isn't an opt-in", d.Key)
			}
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("GET", "/api/admin/opt-ins", nil).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role", func(t *testing.T) {
		w.adaC.Do("GET", "/api/admin/opt-ins", nil).Want(t, http.StatusForbidden)
	})
	// Another trust's data and invalid input don't apply: the route takes no trust and no input.
}

// Editing a trust's details at any time (Rahul, 13 Oct, S10 answers 3): name, code, contact and
// logo. Only the fields sent change.
func TestAdminEditTrustDetails(t *testing.T) {
	w := newAdminWorld(t)
	tr := createTrust(t, w, "ED-1", "ed1.admin@elsewhere.example")
	path := adminOrgPath(tr.ID, "")

	t.Run("happy path: each field, trimmed; the rest unchanged", func(t *testing.T) {
		got := patchAdmin(t, w, path, map[string]any{
			"name": " St Elsewhere NHS Trust ", "code": " ed-2 ", "contactName": "Sam Contact",
			"contactEmail": "Sam@Elsewhere.Example", "contactPhone": "", "logoUrl": "",
		}, http.StatusOK)
		if got.Name != "St Elsewhere NHS Trust" || got.Code != "ED-2" || got.ContactName != "Sam Contact" ||
			got.ContactEmail != "sam@elsewhere.example" || got.ContactPhone != "" || got.LogoURL != "" || got.Status != "active" {
			t.Errorf("got %+v", got)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND name = 'St Elsewhere NHS Trust' AND code = 'ED-2'
			AND NOT settings ? 'branding.logoUrl' AND settings->>'feature.proms' = 'true'`, tr.ID); n != 1 {
			t.Error("not stored, or the opt-ins were lost")
		}
		got = patchAdmin(t, w, path, map[string]any{"logoUrl": "https://cdn.example/new.png"}, http.StatusOK)
		if got.LogoURL != "https://cdn.example/new.png" || got.Name != "St Elsewhere NHS Trust" {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("a code another trust has is refused, whatever its case; its own code is fine", func(t *testing.T) {
		var other string
		if err := w.c.DB.QueryRow(t.Context(), `SELECT code FROM orgs WHERE id = $1`, w.a1.Org).Scan(&other); err != nil {
			t.Fatal(err)
		}
		r := w.samC.Do("PATCH", path, map[string]any{"code": strings.ToLower(other)})
		r.Want(t, http.StatusConflict)
		patchAdmin(t, w, path, map[string]any{"code": "ed-2"}, http.StatusOK)
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("PATCH", path, map[string]any{"name": "X"}).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: a trust admin can't rename their trust", func(t *testing.T) {
		w.adaC.Do("PATCH", adminOrgPath(w.a1.Org, ""), map[string]any{"name": "Renamed"}).Want(t, http.StatusForbidden)
		if count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE name = 'Renamed'`) != 0 {
			t.Error("renamed")
		}
	})
	t.Run("an unknown trust gets 404", func(t *testing.T) {
		patchAdmin(t, w, adminOrgPath(uuid.New(), ""), map[string]any{"name": "X"}, http.StatusNotFound)
	})
	t.Run("invalid input: nothing changes", func(t *testing.T) {
		for _, b := range []map[string]any{
			{"name": " "}, {"name": strings.Repeat("n", 201)}, {"code": "A"}, {"code": "A B"}, {"contactName": ""},
			{"contactEmail": "nope"}, {"contactEmail": ""}, {"contactPhone": "call me"}, {"logoUrl": "javascript:x"},
			{"name": "Fine", "code": "!"},
		} {
			patchAdmin(t, w, path, b, http.StatusBadRequest)
		}
		if count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND name = 'St Elsewhere NHS Trust'`, tr.ID) != 1 {
			t.Error("changed")
		}
	})
}

// Editing a hospital's details at any time (S10 answers 3): name, address, the patients' phone
// and email, logo and time zone. "" for phone, email or logo goes back to the trust's.
func TestAdminEditHospitalDetails(t *testing.T) {
	w := newAdminWorld(t)
	tr := createTrust(t, w, "EH-1", "eh1.admin@elsewhere.example")
	got := addHospital(t, w, tr.ID, hospitalBody("East Hospital"))
	east := got.Hospitals[0].ID
	addHospital(t, w, tr.ID, hospitalBody("West Hospital"))
	path := adminOrgPath(tr.ID, "/hospitals/"+east.String())
	find := func(a adminTrust, id uuid.UUID) *adminHospital { return a.hospital(id) }

	t.Run("happy path: each field; the rest unchanged", func(t *testing.T) {
		got := patchAdmin(t, w, path, map[string]any{"name": " East General ", "address": "2 New Road", "timeZone": "America/Toronto",
			"phone": "", "email": "east@hospital.example", "logoUrl": "https://cdn.example/east.png"}, http.StatusOK)
		h := find(got, east)
		if h.Name != "East General" || h.Address != "2 New Road" || h.TimeZone != "America/Toronto" || h.OwnTimeZone == nil ||
			h.Phone != "" || h.Email != "east@hospital.example" || h.LogoURL != "https://cdn.example/east.png" || h.Status != "active" {
			t.Errorf("got %+v", h)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM hospitals WHERE id = $1 AND NOT settings ? 'contact.phone'`, east); n != 1 {
			t.Error("the phone should follow the trust again")
		}
		got = patchAdmin(t, w, path, map[string]any{"timeZone": ""}, http.StatusOK)
		if h := find(got, east); h.OwnTimeZone != nil || h.Name != "East General" {
			t.Errorf("got %+v", h)
		}
	})
	t.Run("another hospital's name in the trust is refused", func(t *testing.T) {
		w.samC.Do("PATCH", path, map[string]any{"name": "west hospital"}).Want(t, http.StatusConflict)
		patchAdmin(t, w, path, map[string]any{"name": "east general"}, http.StatusOK) // its own, in another case
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("PATCH", path, map[string]any{"name": "X"}).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role", func(t *testing.T) {
		w.adaC.Do("PATCH", adminOrgPath(w.a1.Org, "/hospitals/"+w.a1.ID.String()), map[string]any{"name": "Renamed"}).
			Want(t, http.StatusForbidden)
	})
	t.Run("another trust's hospital under this trust's address gets 404", func(t *testing.T) {
		patchAdmin(t, w, adminOrgPath(tr.ID, "/hospitals/"+w.a1.ID.String()), map[string]any{"name": "Stolen"}, http.StatusNotFound)
		if count(t, w.c.DB, `SELECT count(*) FROM hospitals WHERE name = 'Stolen'`) != 0 {
			t.Error("renamed")
		}
	})
	t.Run("invalid input: nothing changes", func(t *testing.T) {
		for _, b := range []map[string]any{
			{"name": ""}, {"address": " "}, {"address": strings.Repeat("a", 501)}, {"timeZone": "Mars/Olympus"},
			{"timeZone": "Local"}, {"phone": "ring"}, {"email": "x@"}, {"logoUrl": "ftp://x.example/a.png"},
			{"name": "Fine", "timeZone": "Nope/Nope"},
		} {
			patchAdmin(t, w, path, b, http.StatusBadRequest)
		}
		if count(t, w.c.DB, `SELECT count(*) FROM hospitals WHERE id = $1 AND name = 'east general'`, east) != 1 {
			t.Error("changed")
		}
	})
}

func TestAdminInviteConflictMessage(t *testing.T) {
	// The console has no list to change access in (S10 test report 1): the message says where.
	w := newAdminWorld(t)
	r := w.samC.Do("POST", adminOrgPath(w.a1.Org, "/admins"), map[string]any{"email": adminEmailOf(t, w, w.ada)})
	r.Want(t, http.StatusConflict)
	if strings.Contains(r.Message(), "list") || !strings.Contains(r.Message(), "Staff") {
		t.Errorf("message %q", r.Message())
	}
}
