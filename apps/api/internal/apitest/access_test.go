//go:build integration

package apitest_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

type route struct{ method, path string }

// hospitalRoutes is every route under one hospital (router.go). Path ids are made up: the
// sign-in and membership checks run before any handler looks them up.
func hospitalRoutes(base string) []route {
	id := uuid.NewString()
	return []route{
		{"GET", base + "/questionnaires"},
		{"POST", base + "/questionnaires"},
		{"GET", base + "/questionnaires/" + id},
		{"DELETE", base + "/questionnaires/" + id},
		{"POST", base + "/questionnaires/" + id + "/chapters"},
		{"PUT", base + "/questionnaires/" + id + "/chapter-order"},
		{"POST", base + "/questionnaires/" + id + "/publish"},
		{"POST", base + "/questionnaires/" + id + "/versions"},
		{"GET", base + "/chapters/" + id},
		{"PATCH", base + "/chapters/" + id},
		{"DELETE", base + "/chapters/" + id},
		{"PUT", base + "/chapters/" + id + "/content"},
		{"GET", base + "/option-lists"},
		{"POST", base + "/option-lists"},
		{"PUT", base + "/option-lists/" + id},
		{"DELETE", base + "/option-lists/" + id},
		{"GET", base + "/patients"},
		{"POST", base + "/patients"},
		{"GET", base + "/published-hqs"},
		{"GET", base + "/episodes"},
		{"POST", base + "/episodes"},
		{"GET", base + "/episodes/" + id},
		{"PATCH", base + "/episodes/" + id},
		{"POST", base + "/episodes/" + id + "/notes"},
		{"GET", base + "/episodes/" + id + "/hq"},
		{"PUT", base + "/episodes/" + id + "/answers/" + id},
		{"POST", base + "/episodes/" + id + "/complete-review"},
		{"POST", base + "/episodes/" + id + "/patient-link"},
	}
}

func TestEveryStaffRouteNeedsSignIn(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	hospital := c.Hospital("Test Hospital")

	routes := append(hospitalRoutes(c.HospitalPath(hospital)),
		route{"GET", "/api/me"}, route{"GET", "/api/codes?set=SNOMED&q=a"}, route{"GET", "/api/categories"},
		route{"GET", "/api/o/" + c.OrgOf(hospital).String() + "/staff"},
		route{"POST", "/api/o/" + c.OrgOf(hospital).String() + "/staff/invites"})
	for _, rt := range routes {
		if r := c.Do(rt.method, rt.path, nil); r.Status != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401", rt.method, rt.path, r.Status)
		}
	}
}

// Another trust's hospitals answer 404, as if they didn't exist, on every route (NFR-04).
func TestAnotherTrustIsNotFound(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	mine, theirs := c.Hospital("Mine"), c.Hospital("Theirs")
	c.LoginAs(c.TrustUser(c.OrgOf(mine), "admin"))

	for _, rt := range hospitalRoutes(c.HospitalPath(theirs)) {
		if r := c.Do(rt.method, rt.path, nil); r.Status != http.StatusNotFound {
			t.Errorf("%s %s: status = %d, want 404", rt.method, rt.path, r.Status)
		}
	}
}

// A hospital is reached only through its own trust, even by someone in both trusts.
func TestAHospitalOnlyUnderItsOwnTrust(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	a, b := c.Hospital("A"), c.Hospital("B")
	user := c.User(a, "clinician")
	c.Member(user, c.OrgOf(b), &b, "clinician")
	c.LoginAs(user)

	for _, tc := range []struct {
		name, path string
		want       int
	}{
		{"A under its trust", c.HospitalPath(a) + "/episodes", http.StatusOK},
		{"B under its trust", c.HospitalPath(b) + "/episodes", http.StatusOK},
		{"B under A's trust", "/api/o/" + c.OrgOf(a).String() + "/h/" + b.String() + "/episodes", http.StatusNotFound},
		{"trust id not a uuid", "/api/o/abc/h/" + a.String() + "/episodes", http.StatusNotFound},
		{"hospital id not a uuid", "/api/o/" + c.OrgOf(a).String() + "/h/abc/episodes", http.StatusNotFound},
	} {
		if r := c.Do(http.MethodGet, tc.path, nil); r.Status != tc.want {
			t.Errorf("%s: status = %d, want %d", tc.name, r.Status, tc.want)
		}
	}
}

// A hospital-scoped member sees only their hospital; a trust-wide member sees every hospital in the trust.
func TestHospitalAndTrustScope(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	org := c.Org("Trust")
	h1, h2 := c.HospitalIn(org, "H1"), c.HospitalIn(org, "H2")

	c.LoginAs(c.User(h1, "clinician"))
	if r := c.Do(http.MethodGet, c.HospitalPath(h1)+"/episodes", nil); r.Status != http.StatusOK {
		t.Errorf("hospital member, own hospital: status = %d, want 200", r.Status)
	}
	if r := c.Do(http.MethodGet, c.HospitalPath(h2)+"/episodes", nil); r.Status != http.StatusNotFound {
		t.Errorf("hospital member, other hospital in the trust: status = %d, want 404", r.Status)
	}

	c.LoginAs(c.TrustUser(org, "clinician"))
	for _, h := range []uuid.UUID{h1, h2} {
		if r := c.Do(http.MethodGet, c.HospitalPath(h)+"/episodes", nil); r.Status != http.StatusOK {
			t.Errorf("trust member, %s: status = %d, want 200", h, r.Status)
		}
	}
}

// The ladder: a clinician can't author; a super clinician authors and still sees episodes.
func TestRoleLadder(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	h := c.Hospital("H")
	newQ := map[string]any{"name": "Pre-op " + uuid.NewString(), "description": "Test"}

	c.LoginAs(c.User(h, "clinician"))
	if r := c.Do(http.MethodPost, c.HospitalPath(h)+"/questionnaires", newQ); r.Status != http.StatusForbidden {
		t.Errorf("clinician creates a questionnaire: status = %d, want 403", r.Status)
	}
	if r := c.Do(http.MethodGet, c.HospitalPath(h)+"/episodes", nil); r.Status != http.StatusOK {
		t.Errorf("clinician lists episodes: status = %d, want 200", r.Status)
	}

	c.LoginAs(c.User(h, "super_clinician"))
	r := c.Do(http.MethodPost, c.HospitalPath(h)+"/questionnaires", newQ)
	if r.Status != http.StatusCreated {
		t.Fatalf("super clinician creates a questionnaire: status = %d, want 201: %s", r.Status, r.Body)
	}
	if r := c.Do(http.MethodGet, c.HospitalPath(h)+"/episodes", nil); r.Status != http.StatusOK {
		t.Errorf("super clinician lists episodes: status = %d, want 200", r.Status)
	}
	var created struct{ ID string }
	r.Decode(t, &created)
	qPath := c.HospitalPath(h) + "/questionnaires/" + created.ID

	// Deleting a draft: its creator or an admin, not another super clinician.
	c.LoginAs(c.User(h, "super_clinician"))
	if r := c.Do(http.MethodDelete, qPath, nil); r.Status != http.StatusForbidden {
		t.Errorf("another super clinician deletes the draft: status = %d, want 403", r.Status)
	}
	c.LoginAs(c.TrustUser(c.OrgOf(h), "admin"))
	if r := c.Do(http.MethodDelete, qPath, nil); r.Status != http.StatusNoContent {
		t.Errorf("trust admin deletes the draft: status = %d, want 204: %s", r.Status, r.Body)
	}
	var n int
	if err := c.DB.QueryRow(t.Context(), "SELECT count(*) FROM questionnaires WHERE id = $1", created.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("questionnaire rows after delete = %d, want 0", n)
	}
}

// Removing a membership or suspending the trust ends access on the very next request (STF-04).
func TestAccessEndsAtOnce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, sql string }{
		{"membership removed", "DELETE FROM memberships"},
		{"trust suspended", "UPDATE orgs SET status = 'suspended'"},
	} {
		c := apitest.New(t)
		h := c.Hospital("H")
		c.LoginAs(c.User(h, "clinician"))
		if r := c.Do(http.MethodGet, c.HospitalPath(h)+"/episodes", nil); r.Status != http.StatusOK {
			t.Fatalf("%s: before: status = %d, want 200", tc.name, r.Status)
		}
		path := c.HospitalPath(h) + "/episodes" // read before the change: OrgOf still works
		if _, err := c.DB.Exec(t.Context(), tc.sql); err != nil {
			t.Fatal(err)
		}
		if r := c.Do(http.MethodGet, path, nil); r.Status != http.StatusNotFound {
			t.Errorf("%s: after: status = %d, want 404", tc.name, r.Status)
		}
		var me struct{ Hospitals []struct{ ID string } }
		r := c.Do(http.MethodGet, "/api/me", nil)
		r.Decode(t, &me)
		if len(me.Hospitals) != 0 {
			t.Errorf("%s: /api/me still lists %+v", tc.name, me.Hospitals)
		}
	}
}

// /api/me lists every hospital the user can open, with its trust and the role that applies there.
func TestMeListsTrustsAndHospitals(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	org := c.Org("Northern Trust")
	h1, h2 := c.HospitalIn(org, "North General"), c.HospitalIn(org, "North Royal")
	other := c.Hospital("Elsewhere")

	user := c.TrustUser(org, "clinician")
	c.Member(user, org, &h2, "super_clinician") // higher in one hospital
	c.Member(user, c.OrgOf(other), &other, "admin")
	c.LoginAs(user)

	r := c.Do(http.MethodGet, "/api/me", nil)
	if r.Status != http.StatusOK {
		t.Fatalf("status = %d: %s", r.Status, r.Body)
	}
	var me struct {
		Hospitals []struct{ ID, Name, OrgID, OrgName, Role string }
	}
	r.Decode(t, &me)
	got := map[string]string{}
	for _, h := range me.Hospitals {
		got[h.ID] = h.OrgID + " " + h.Role
	}
	want := map[string]string{
		h1.String():    org.String() + " clinician",
		h2.String():    org.String() + " super_clinician",
		other.String(): c.OrgOf(other).String() + " admin",
	}
	if len(got) != len(want) {
		t.Fatalf("hospitals = %+v, want %d", me.Hospitals, len(want))
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("hospital %s = %q, want %q", id, got[id], w)
		}
	}
}
