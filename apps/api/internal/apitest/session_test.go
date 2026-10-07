//go:build integration

package apitest_test

import (
	"crypto/sha256"
	"net/http"
	"testing"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/auth"
)

// signIn signs in as a new super clinician through the dev login and returns the session cookie.
func signIn(t *testing.T, c *apitest.Client) *http.Cookie {
	t.Helper()
	user := c.User(c.Hospital("Test Hospital"), "super_clinician")
	r := c.Do(http.MethodPost, "/api/dev/login", map[string]any{"userId": user})
	if r.Status != http.StatusNoContent {
		t.Fatalf("login: status = %d: %s", r.Status, r.Body)
	}
	return sessionCookie(t, r)
}

// withCookie sends GET path carrying only the given session cookie.
func withCookie(c *apitest.Client, path string, cookie *http.Cookie) *apitest.Resp {
	return c.Send(http.MethodGet, path, nil, http.Header{"Cookie": {cookie.Name + "=" + cookie.Value}})
}

// The database keeps only a hash of the session id, so its rows can't be used to sign in.
func TestSessionIsStoredAsAHash(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	cookie := signIn(t, c)

	var hash []byte
	if err := c.DB.QueryRow(t.Context(), "SELECT id_hash FROM sessions").Scan(&hash); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte(cookie.Value))
	if string(hash) != string(want[:]) {
		t.Errorf("stored %x, want sha256 of the cookie value", hash)
	}
	if len(cookie.Value) != 43 {
		t.Errorf("session id has %d characters, want 43 (32 random bytes, base64url)", len(cookie.Value))
	}
}

func TestLogoutEndsTheSession(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	cookie := signIn(t, c)

	if r := c.Do(http.MethodPost, "/api/logout", nil); r.Status != http.StatusNoContent {
		t.Fatalf("logout: status = %d", r.Status)
	}
	if r := withCookie(c, "/api/me", cookie); r.Status != http.StatusUnauthorized {
		t.Errorf("old cookie after logout: status = %d, want 401", r.Status)
	}
	var n int
	if err := c.DB.QueryRow(t.Context(), "SELECT count(*) FROM sessions").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("sessions after logout = %d, want 0", n)
	}
}

func TestSessionLimits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		update string // ages the session row
		want   int
	}{
		{"idle past the limit", "UPDATE sessions SET last_seen_at = now() - interval '31 minutes'", http.StatusUnauthorized},
		{"past its maximum age", "UPDATE sessions SET expires_at = now() - interval '1 second'", http.StatusUnauthorized},
		{"idle within the limit", "UPDATE sessions SET last_seen_at = now() - interval '29 minutes'", http.StatusOK},
	} {
		c := apitest.New(t)
		cookie := signIn(t, c)
		if _, err := c.DB.Exec(t.Context(), tc.update); err != nil {
			t.Fatal(err)
		}
		if r := withCookie(c, "/api/me", cookie); r.Status != tc.want {
			t.Errorf("%s: status = %d, want %d: %s", tc.name, r.Status, tc.want, r.Body)
		}
	}
}

// Using the app moves last_seen_at forward, so an active user is not timed out.
func TestActivityKeepsTheSessionAlive(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	cookie := signIn(t, c)
	if _, err := c.DB.Exec(t.Context(), "UPDATE sessions SET last_seen_at = now() - interval '20 minutes'"); err != nil {
		t.Fatal(err)
	}

	if r := withCookie(c, "/api/me", cookie); r.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", r.Status)
	}
	var seen time.Time
	if err := c.DB.QueryRow(t.Context(), "SELECT last_seen_at FROM sessions").Scan(&seen); err != nil {
		t.Fatal(err)
	}
	if time.Since(seen) > time.Minute {
		t.Errorf("last_seen_at = %v, want about now", seen)
	}
}

// A made-up cookie value is refused like an expired one.
func TestUnknownSessionIsRefused(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)

	r := withCookie(c, "/api/me", &http.Cookie{Name: auth.CookieName, Value: "not-a-real-session"})
	if r.Status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", r.Status)
	}
}
