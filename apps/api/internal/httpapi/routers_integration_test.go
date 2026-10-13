//go:build integration

package httpapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

// The staff and patient routers (plan F3 Step 1): on AWS each runs as its own Lambda (D-4),
// so the patient's function can't serve a staff route or the dev login, and the staff
// function can't serve a patient's link. Locally, NewRouter mounts both.

// staffRoutes is one request per staff route group; each is answered (not 404) by NewRouter.
var staffRoutes = [][2]string{
	{"GET", "/api/me"},
	{"GET", "/api/codes?q=asthma"},
	{"GET", "/api/categories"},
	{"POST", "/api/logout"},
	{"GET", "/api/dev/users"},
	{"POST", "/api/dev/login"},
}

func TestPatientRouter(t *testing.T) {
	t.Parallel()
	p, fx := apitest.NewWith(t, apitest.Options{Router: httpapi.NewPatientRouter}) // DEV_LOGIN is on
	h := fx.Hospital(t)
	alex := fx.User(t, h, "super_clinician")
	hq := fx.PublishedHQ(t, h, alex, "workflow.json")
	ep := fx.Episode(t, hq, alex)

	t.Run("serves the patient's link and their session", func(t *testing.T) {
		b := p.AsPatient(t, ep.Link, ep.DateOfBirth)
		b.Do("GET", patientPath(ep, ""), nil).Want(t, http.StatusOK)
		b.Do("GET", "/api/p/home", nil).Want(t, http.StatusOK)
		p.Do("POST", "/api/p/signin", map[string]string{"contact": "nobody@example.org"}).Want(t, http.StatusAccepted)
		p.Do("POST", "/api/p/logout", nil).Want(t, http.StatusNoContent)
	})
	t.Run("serves health and config", func(t *testing.T) {
		p.Do("GET", "/api/health", nil).Want(t, http.StatusOK)
		p.Do("GET", "/api/config", nil).Want(t, http.StatusOK)
	})
	t.Run("404 for every staff route, the dev login included", func(t *testing.T) {
		for _, r := range staffRoutes {
			p.Do(r[0], r[1], map[string]any{"userId": alex.ID}).Want(t, http.StatusNotFound)
		}
		p.Do("GET", at(h, "/episodes"), nil).Want(t, http.StatusNotFound)
		p.Do("GET", at(h, "/questionnaires"), nil).Want(t, http.StatusNotFound)
		p.Do("GET", at(h, "/patients?q=Patient"), nil).Want(t, http.StatusNotFound)
		p.Do("GET", at(h, "/patients/"+ep.PatientID.String()), nil).Want(t, http.StatusNotFound)
		p.Do("GET", "/api/o/"+h.Org.String()+"/episodes", nil).Want(t, http.StatusNotFound)
		p.Do("GET", "/api/o/"+h.Org.String()+"/staff", nil).Want(t, http.StatusNotFound)
		p.Do("POST", "/api/o/"+h.Org.String()+"/staff/invites", map[string]any{"email": "a@b.example"}).Want(t, http.StatusNotFound)
		p.Do("GET", "/api/o/"+h.Org.String()+"/settings", nil).Want(t, http.StatusNotFound)
		p.Do("GET", "/api/o/"+h.Org.String()+"/audit", nil).Want(t, http.StatusNotFound)
		p.Do("GET", "/api/admin/orgs", nil).Want(t, http.StatusNotFound)
		p.Do("POST", "/api/admin/orgs", map[string]any{"name": "x"}).Want(t, http.StatusNotFound)
	})
}

func TestStaffRouter(t *testing.T) {
	t.Parallel()
	s, fx := apitest.NewWith(t, apitest.Options{Router: httpapi.NewStaffRouter})
	h := fx.Hospital(t)
	alex := fx.User(t, h, "super_clinician")
	hq := fx.PublishedHQ(t, h, alex, "workflow.json")
	ep := fx.Episode(t, hq, alex)
	signed := s.LoginAs(t, alex) // through the staff router's dev login

	t.Run("serves the staff routes", func(t *testing.T) {
		signed.Do("GET", "/api/me", nil).Want(t, http.StatusOK)
		signed.Do("GET", at(h, "/episodes"), nil).Want(t, http.StatusOK)
		signed.Do("GET", at(h, "/questionnaires"), nil).Want(t, http.StatusOK)
		signed.Do("GET", "/api/o/"+h.Org.String()+"/episodes", nil).Want(t, http.StatusOK)
		signed.Do("GET", "/api/o/"+h.Org.String()+"/staff", nil).Want(t, http.StatusForbidden) // served; Alex isn't an admin
		signed.Do("GET", "/api/o/"+h.Org.String()+"/settings", nil).Want(t, http.StatusForbidden)
		signed.Do("GET", "/api/o/"+h.Org.String()+"/audit", nil).Want(t, http.StatusForbidden)
		signed.Do("GET", "/api/admin/orgs", nil).Want(t, http.StatusForbidden) // served; Alex isn't a platform admin
	})
	t.Run("serves health and config", func(t *testing.T) {
		s.Do("GET", "/api/health", nil).Want(t, http.StatusOK)
		s.Do("GET", "/api/config", nil).Want(t, http.StatusOK)
	})
	t.Run("404 for the patient's link, signed in or not", func(t *testing.T) {
		for _, c := range []*apitest.Client{s, signed} {
			c.Do("POST", "/api/p/links/"+ep.Link+"/continue", nil).Want(t, http.StatusNotFound)
			c.Do("POST", "/api/p/links/"+ep.Link+"/dob", map[string]string{"dateOfBirth": ep.DateOfBirth}).Want(t, http.StatusNotFound)
			c.Do("GET", patientPath(ep, ""), nil).Want(t, http.StatusNotFound)
			c.Do("PUT", patientPath(ep, "/answers/"+hq.Chapters[0].String()), map[string]any{"data": map[string]any{}}).Want(t, http.StatusNotFound)
			c.Do("POST", patientPath(ep, "/submit"), nil).Want(t, http.StatusNotFound)
			c.Do("POST", "/api/p/signin", map[string]string{"contact": "a@b.example"}).Want(t, http.StatusNotFound)
			c.Do("GET", "/api/p/home", nil).Want(t, http.StatusNotFound)
			c.Do("POST", "/api/p/contact", map[string]string{"contact": "a@b.example"}).Want(t, http.StatusNotFound)
		}
		if n := count(t, s.DB, `SELECT count(*) FROM episode_answers WHERE episode_id = $1`, ep.ID); n != 0 {
			t.Errorf("%d answers rows written through the staff router", n)
		}
	})
}

func TestDevLoginOffUnlessEnabled(t *testing.T) {
	t.Parallel()
	for name, router := range map[string]func(httpapi.Deps) http.Handler{
		"NewRouter": httpapi.NewRouter, "NewStaffRouter": httpapi.NewStaffRouter,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c, fx := apitest.NewWith(t, apitest.Options{Router: router, Env: map[string]string{"DEV_LOGIN": ""}})
			alex := fx.User(t, fx.Hospital(t), "super_clinician")
			c.Do("POST", "/api/dev/login", map[string]any{"userId": alex.ID}).Want(t, http.StatusNotFound)
			c.Do("GET", "/api/dev/users", nil).Want(t, http.StatusNotFound)
			// The rest of the staff API is still there, waiting for a real sign-in.
			c.Do("GET", "/api/me", nil).Want(t, http.StatusUnauthorized)
		})
	}
}

// GET /api/config: the deployment's public per-country values (A-20). It's public, because the
// sign-in page and the patient's link need it before anyone signs in, so the five cases become:
//   - happy path: a made-up country's values come back, and nothing else;
//   - no sign-in: the same answer;
//   - wrong role / another trust: a signed-in user gets the same answer; there's no trust data;
//   - invalid input: another method is refused, and query parameters change nothing.
func TestConfig(t *testing.T) {
	t.Parallel()
	country := map[string]string{
		"APP_URL": "https://ca.example.test", "DEFAULT_TIME_ZONE": "America/Toronto",
		"DEFAULT_LOCALE": "fr-CA", "PHONE_COUNTRY": "CA", "CONFIG_STRICT": "true",
		"NATIONAL_ID_TYPE": "nhs",     // the only scheme built so far
		"MESSAGING":        "capture", // strict needs it set (S14 verification 1)
	}
	c, fx := apitest.NewWith(t, apitest.Options{Env: country})
	want := map[string]any{"locale": "fr-CA", "timeZone": "America/Toronto",
		"nationalId": map[string]any{"type": "nhs", "label": "NHS number"}}
	check := func(t *testing.T, r *apitest.Resp) {
		t.Helper()
		r.Want(t, http.StatusOK)
		var got map[string]any
		r.Decode(t, &got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("config = %v, want exactly %v", got, want)
		}
	}

	t.Run("happy path, with no sign-in: the deployment's values and nothing else", func(t *testing.T) {
		check(t, c.Do("GET", "/api/config", nil))
	})
	t.Run("signed in: the same answer", func(t *testing.T) {
		check(t, c.LoginAs(t, fx.User(t, fx.Hospital(t), "clinician")).Do("GET", "/api/config", nil))
	})
	t.Run("invalid input", func(t *testing.T) {
		c.Do("POST", "/api/config", map[string]any{"locale": "en-US"}).Want(t, http.StatusMethodNotAllowed)
		check(t, c.Do("GET", "/api/config?locale=en-US", nil))
	})
	t.Run("the local defaults when nothing is set", func(t *testing.T) {
		d, _ := apitest.New(t)
		r := d.Do("GET", "/api/config", nil)
		r.Want(t, http.StatusOK)
		var got struct {
			Locale, TimeZone string
			NationalID       struct{ Type string }
		}
		r.Decode(t, &got)
		if got.Locale != "en-GB" || got.TimeZone != "Europe/London" || got.NationalID.Type != "nhs" {
			t.Errorf("config = %v, want the local UK defaults", got)
		}
	})
}
