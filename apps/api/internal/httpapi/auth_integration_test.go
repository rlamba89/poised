//go:build integration

package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
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
			t.Errorf("users = %+v, want Alex, Bea and the viewer", got)
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
	type me struct {
		User      struct{ ID uuid.UUID }
		Hospitals []struct {
			ID    uuid.UUID
			Roles []string
		}
	}

	t.Run("happy path: the user and their hospitals with roles", func(t *testing.T) {
		r := w.author.Do("GET", "/api/me", nil)
		r.Want(t, http.StatusOK)
		var got me
		r.Decode(t, &got)
		if got.User.ID != w.alex.ID || len(got.Hospitals) != 1 || got.Hospitals[0].ID != w.a || len(got.Hospitals[0].Roles) != 2 {
			t.Errorf("me = %+v, want Alex in A as author and publisher", got)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", "/api/me", nil), http.StatusUnauthorized)
	})
	t.Run("role: a user with no hospital gets none", func(t *testing.T) {
		r := w.anon.LoginAs(t, w.fx.User(t, w.a)).Do("GET", "/api/me", nil)
		r.Want(t, http.StatusOK)
		var got me
		r.Decode(t, &got)
		if len(got.Hospitals) != 0 {
			t.Errorf("hospitals = %+v, want none", got.Hospitals)
		}
	})
	t.Run("another hospital: only the user's own", func(t *testing.T) {
		r := w.other.Do("GET", "/api/me", nil)
		r.Want(t, http.StatusOK)
		var got me
		r.Decode(t, &got)
		if len(got.Hospitals) != 1 || got.Hospitals[0].ID != w.b {
			t.Errorf("Bea's hospitals = %+v, want only B", got.Hospitals)
		}
	})
	t.Run("invalid input: a session for a user who no longer exists", func(t *testing.T) {
		gone := w.fx.User(t, w.a)
		c := w.anon.LoginAs(t, gone)
		if _, err := w.db.Exec(t.Context(), "DELETE FROM users WHERE id = $1", gone.ID); err != nil {
			t.Fatal(err)
		}
		refused(t, c.Do("GET", "/api/me", nil), http.StatusUnauthorized)
	})
}
