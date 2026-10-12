//go:build integration

package httpapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The dev sign-in stub (replaced by Cognito in C1) and /api/me. The stub's routes need no
// sign-in and no role, so those cases check what an unsigned caller gets; /api/me has all five.

func TestDevUsers(t *testing.T) {
	t.Parallel()
	w := newCast(t)

	t.Run("happy path, with no sign-in: every user, for the stub login page", func(t *testing.T) {
		r := w.anon.Do("GET", "/api/dev/users", nil)
		r.Want(t, http.StatusOK)
		var got []struct{ ID uuid.UUID }
		r.Decode(t, &got)
		ids := map[uuid.UUID]bool{}
		for _, u := range got {
			ids[u.ID] = true
		}
		if len(got) != 3 || !ids[w.alex.ID] || !ids[w.bea.ID] {
			t.Errorf("users = %+v, want Alex, Bea and the clinician", got)
		}
	})
}

func TestDevLogin(t *testing.T) {
	t.Parallel()
	w := newCast(t)

	t.Run("happy path: signs in", func(t *testing.T) {
		// LoginAs signs in through this route and fails the test unless it answers 204.
		w.anon.LoginAs(t, w.alex).Do("GET", "/api/me", nil).Want(t, http.StatusOK)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.anon.Do("POST", "/api/dev/login", map[string]any{"userId": uuid.New()}), http.StatusNotFound)
		refused(t, w.anon.Do("POST", "/api/dev/login", `{"userId": "abc"}`), http.StatusBadRequest)
		refused(t, w.anon.Do("POST", "/api/dev/login", `{"userId":`), http.StatusBadRequest)
	})
	t.Run("a refused sign-in leaves the caller signed out", func(t *testing.T) {
		w.anon.Do("POST", "/api/dev/login", map[string]any{"userId": uuid.New()}).Want(t, http.StatusNotFound)
		refused(t, w.anon.Do("GET", "/api/me", nil), http.StatusUnauthorized)
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()
	w := newCast(t)

	t.Run("happy path: the session cookie is cleared", func(t *testing.T) {
		alex := w.anon.LoginAs(t, w.alex)
		alex.Do("POST", "/api/logout", nil).Want(t, http.StatusNoContent)
		refused(t, alex.Do("GET", "/api/me", nil), http.StatusUnauthorized)
	})
	t.Run("no sign-in: harmless", func(t *testing.T) {
		w.anon.Do("POST", "/api/logout", nil).Want(t, http.StatusNoContent)
	})
	t.Run("another user's session is untouched", func(t *testing.T) {
		w.anon.LoginAs(t, w.alex).Do("POST", "/api/logout", nil).Want(t, http.StatusNoContent)
		w.author.Do("GET", "/api/me", nil).Want(t, http.StatusOK)
	})
}

func TestMe(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	type hospital struct {
		ID       uuid.UUID
		Name     string
		TimeZone string
		Role     string
	}
	type trust struct {
		ID        uuid.UUID
		Name      string
		Role      *string
		Hospitals []hospital
	}
	type me struct {
		User          struct{ ID uuid.UUID }
		PlatformAdmin bool
		Trusts        []trust
	}
	get := func(t *testing.T, c *apitest.Client) me {
		t.Helper()
		r := c.Do("GET", "/api/me", nil)
		r.Want(t, http.StatusOK)
		var got me
		r.Decode(t, &got)
		return got
	}

	t.Run("happy path: each trust with the hospitals the user can work in, and their role in each", func(t *testing.T) {
		// Max: a trust-wide clinician in A who is also admin at A's second hospital, and a
		// clinician at B only.
		a2 := w.fx.HospitalIn(t, w.a.Org)
		maxU := w.fx.TrustUser(t, w.a.Org, "clinician")
		w.fx.Member(t, maxU, a2, "admin")
		w.fx.Member(t, maxU, w.b, "clinician")
		if _, err := w.db.Exec(t.Context(), "UPDATE hospitals SET timezone = 'America/Toronto' WHERE id = $1", a2.ID); err != nil {
			t.Fatal(err)
		}
		got := get(t, w.anon.LoginAs(t, maxU))
		if got.User.ID != maxU.ID || got.PlatformAdmin || len(got.Trusts) != 2 {
			t.Fatalf("me = %+v, want Max in two trusts", got)
		}
		roles := map[uuid.UUID]string{}
		zones := map[uuid.UUID]string{}
		trustRole := map[uuid.UUID]string{}
		for _, tr := range got.Trusts {
			if tr.Role != nil {
				trustRole[tr.ID] = *tr.Role
			}
			for _, h := range tr.Hospitals {
				roles[h.ID], zones[h.ID] = h.Role, h.TimeZone
			}
		}
		want := map[uuid.UUID]string{w.a.ID: "clinician", a2.ID: "admin", w.b.ID: "clinician"}
		if !reflect.DeepEqual(roles, want) {
			t.Errorf("roles = %v, want %v (the higher of trust-wide and hospital roles)", roles, want)
		}
		if !reflect.DeepEqual(trustRole, map[uuid.UUID]string{w.a.Org: "clinician"}) {
			t.Errorf("trust-wide roles = %v, want clinician in A only (B's is one hospital)", trustRole)
		}
		var cfg struct{ TimeZone string }
		w.anon.Do("GET", "/api/config", nil).Decode(t, &cfg)
		if zones[a2.ID] != "America/Toronto" || zones[w.a.ID] != cfg.TimeZone || cfg.TimeZone == "" {
			t.Errorf("time zones = %v, want A2's own and the deployment's %q for the others", zones, cfg.TimeZone)
		}
	})
	t.Run("happy path: a platform admin is marked", func(t *testing.T) {
		sam := w.fx.User(t, w.a, "")
		if _, err := w.db.Exec(t.Context(), "INSERT INTO platform_admins (user_id) VALUES ($1)", sam.ID); err != nil {
			t.Fatal(err)
		}
		if got := get(t, w.anon.LoginAs(t, sam)); !got.PlatformAdmin || len(got.Trusts) != 0 {
			t.Errorf("me = %+v, want a platform admin in no trust", got)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", "/api/me", nil), http.StatusUnauthorized)
	})
	t.Run("role: a user with no membership gets no trust", func(t *testing.T) {
		if got := get(t, w.anon.LoginAs(t, w.fx.User(t, w.a, ""))); len(got.Trusts) != 0 || got.Trusts == nil {
			t.Errorf("trusts = %+v, want an empty list", got.Trusts)
		}
	})
	t.Run("another trust: only the user's own", func(t *testing.T) {
		got := get(t, w.other)
		if len(got.Trusts) != 1 || got.Trusts[0].ID != w.b.Org || got.Trusts[0].Role == nil || *got.Trusts[0].Role != "super_clinician" ||
			len(got.Trusts[0].Hospitals) != 1 || got.Trusts[0].Hospitals[0].ID != w.b.ID || got.Trusts[0].Hospitals[0].Role != "super_clinician" {
			t.Errorf("Bea's trusts = %+v, want only trust B, as its super clinician", got.Trusts)
		}
	})
	t.Run("invalid input: a session for a user who no longer exists", func(t *testing.T) {
		gone := w.fx.User(t, w.a, "")
		c := w.anon.LoginAs(t, gone)
		if _, err := w.db.Exec(t.Context(), "DELETE FROM users WHERE id = $1", gone.ID); err != nil {
			t.Fatal(err)
		}
		refused(t, c.Do("GET", "/api/me", nil), http.StatusUnauthorized)
	})
}
