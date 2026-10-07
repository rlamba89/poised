//go:build integration

package apitest_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

// The dev login is off unless the config turns it on (docs/plans/f3-lambda-entrypoints.md).
func TestDevLoginIsOffByDefault(t *testing.T) {
	t.Parallel()
	c := apitest.NewWith(t, httpapi.Config{})
	user := c.User(c.Hospital("Test Hospital"), "super_clinician")

	if r := c.Do(http.MethodGet, "/api/dev/users", nil); r.Status != http.StatusNotFound {
		t.Errorf("GET /api/dev/users: status = %d, want 404", r.Status)
	}
	if r := c.Do(http.MethodPost, "/api/dev/login", map[string]any{"userId": user}); r.Status != http.StatusNotFound {
		t.Errorf("POST /api/dev/login: status = %d, want 404", r.Status)
	}
}

func TestDevLoginWhenOn(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)

	if r := c.Do(http.MethodGet, "/api/dev/users", nil); r.Status != http.StatusOK {
		t.Errorf("GET /api/dev/users: status = %d, want 200", r.Status)
	}
}

func TestSessionCookieFlags(t *testing.T) {
	t.Parallel()
	for _, secure := range []bool{false, true} {
		c := apitest.NewWith(t, httpapi.Config{DevLogin: true, SecureCookies: secure})
		user := c.User(c.Hospital("Test Hospital"), "super_clinician")

		login := c.Do(http.MethodPost, "/api/dev/login", map[string]any{"userId": user})
		if login.Status != http.StatusNoContent {
			t.Fatalf("login: status = %d: %s", login.Status, login.Body)
		}
		set := sessionCookie(t, login)
		if !set.HttpOnly || set.SameSite != http.SameSiteLaxMode || set.Secure != secure || set.Path != "/" {
			t.Errorf("secure=%v: login cookie = %+v, want HttpOnly, SameSite=Lax, Secure=%v, Path=/", secure, set, secure)
		}
		// A Secure cookie takes the __Host- prefix: the browser then refuses it from a subdomain.
		wantName := auth.CookieName
		if secure {
			wantName = auth.SecureCookieName
		}
		if set.Name != wantName {
			t.Errorf("secure=%v: cookie name = %q, want %q", secure, set.Name, wantName)
		}

		logout := c.Do(http.MethodPost, "/api/logout", nil)
		cleared := sessionCookie(t, logout)
		if cleared.MaxAge >= 0 || !cleared.HttpOnly || cleared.SameSite != http.SameSiteLaxMode || cleared.Secure != secure {
			t.Errorf("secure=%v: logout cookie = %+v, want expired, HttpOnly, SameSite=Lax, Secure=%v", secure, cleared, secure)
		}
	}
}

// Browsers send Sec-Fetch-Site and Origin; a write from another site is refused (CSRF).
func TestCrossSiteWritesAreRefused(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		header http.Header
		want   int
	}{
		{"write from another site", http.MethodPost, "/api/logout", http.Header{"Sec-Fetch-Site": {"cross-site"}}, http.StatusForbidden},
		{"write with a foreign Origin", http.MethodPost, "/api/logout", http.Header{"Origin": {"https://evil.example"}}, http.StatusForbidden},
		{"write from our own page", http.MethodPost, "/api/logout", http.Header{"Sec-Fetch-Site": {"same-origin"}}, http.StatusNoContent},
		{"write with no browser headers", http.MethodPost, "/api/logout", nil, http.StatusNoContent},
		{"read from another site", http.MethodGet, "/api/health", http.Header{"Sec-Fetch-Site": {"cross-site"}}, http.StatusOK},
	} {
		r := c.Send(tc.method, tc.path, nil, tc.header)
		if r.Status != tc.want {
			t.Errorf("%s: status = %d, want %d: %s", tc.name, r.Status, tc.want, r.Body)
		}
		if tc.want == http.StatusForbidden && !strings.Contains(string(r.Body), `"error"`) {
			t.Errorf("%s: body = %s, want the JSON error shape", tc.name, r.Body)
		}
	}
}

// Behind the web app's /api proxy the API's Host is not the page's, so a browser that sends
// only Origin needs the app's origin to be trusted (APP_ORIGIN).
func TestTheAppOriginIsTrusted(t *testing.T) {
	t.Parallel()
	c := apitest.NewWith(t, httpapi.Config{AppOrigin: "http://localhost:3000"})

	if r := c.Send(http.MethodPost, "/api/logout", nil, http.Header{"Origin": {"http://localhost:3000"}}); r.Status != http.StatusNoContent {
		t.Errorf("Origin of the app: status = %d, want 204: %s", r.Status, r.Body)
	}
	if r := c.Send(http.MethodPost, "/api/logout", nil, http.Header{"Origin": {"https://evil.example"}}); r.Status != http.StatusForbidden {
		t.Errorf("another Origin: status = %d, want 403", r.Status)
	}
}

func TestAMalformedAppOriginIsRefused(t *testing.T) {
	t.Parallel()
	if _, err := httpapi.NewRouter(nil, httpapi.Config{AppOrigin: "localhost:3000/path"}); err == nil {
		t.Error("NewRouter accepted a malformed AppOrigin")
	}
}

// sessionCookie returns the session cookie the response sets, failing if there isn't one.
func sessionCookie(t *testing.T, r *apitest.Resp) *http.Cookie {
	t.Helper()
	for _, line := range r.Header.Values("Set-Cookie") {
		c, err := http.ParseSetCookie(line)
		if err != nil {
			t.Fatalf("parse Set-Cookie %q: %v", line, err)
		}
		if c.Name == auth.CookieName || c.Name == auth.SecureCookieName {
			return c
		}
	}
	t.Fatalf("no session cookie in %v", r.Header.Values("Set-Cookie"))
	return nil
}
