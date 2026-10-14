//go:build integration

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// POST /api/auth/callback (S12): the managed login sends the browser back to the app's
// /auth/callback page with a code, which posts it here with its PKCE verifier. The API exchanges
// the code with Cognito (a recording fake), checks the ID token (signed by the test's own key),
// finds the user by Cognito's sub (or, the first time, by the email Cognito verified) and sets
// our own session cookie. The five cases:
//   - happy path: signed in, sub stored, /api/me works;
//   - no sign-in: the route is the sign-in, so the case is "a refused code or token": 401, no cookie;
//   - wrong role: an account whose every trust is suspended, or whose invite expired, is refused
//     (403), and someone Poised doesn't know (no account) too;
//   - another tenant: a token from another pool or for another app client is refused;
//   - invalid input: bad JSON, no code, a malformed verifier.

const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk" // RFC 7636's example: 43 characters

type cognitoWorld struct {
	c     *apitest.Client
	fx    *apitest.Fixtures
	h     apitest.Hospital
	cara  apitest.User
	codes int
}

func newCognitoWorld(t *testing.T) *cognitoWorld {
	c, fx := apitest.NewWith(t, apitest.Options{Cognito: true})
	h := fx.Hospital(t)
	return &cognitoWorld{c: c, fx: fx, h: h, cara: fx.User(t, h, "clinician")}
}

// signIn is the browser coming back from the managed login: a new code for this ID token,
// posted to the callback from a fresh browser, which it returns with the response.
func (w *cognitoWorld) signIn(t *testing.T, idToken string) (*apitest.Client, *apitest.Resp) {
	t.Helper()
	w.codes++
	code := "code-" + uuid.NewString()
	w.c.Cognito.Code(code, verifier, idToken)
	b := w.c.NewBrowser(t)
	return b, b.Do("POST", "/api/auth/callback", map[string]string{"code": code, "verifier": verifier})
}

func signedOut(t *testing.T, b *apitest.Client, r *apitest.Resp) {
	t.Helper()
	if r.Header("Set-Cookie") != "" {
		t.Errorf("a refused sign-in set a cookie: %s", r.Header("Set-Cookie"))
	}
	b.Do("GET", "/api/me", nil).Want(t, http.StatusUnauthorized)
}

func TestCognitoCallback(t *testing.T) {
	t.Parallel()

	t.Run("happy path: signed in by email the first time, then by sub", func(t *testing.T) {
		w := newCognitoWorld(t)
		b, r := w.signIn(t, w.c.IDToken(t, "sub-cara", strings.ToUpper(w.cara.Email), nil))
		r.Want(t, http.StatusNoContent)
		if r.Header("Cache-Control") != "no-store" {
			t.Errorf("Cache-Control = %q", r.Header("Cache-Control"))
		}
		var me struct{ User struct{ ID uuid.UUID } }
		b.Do("GET", "/api/me", nil).Decode(t, &me)
		if me.User.ID != w.cara.ID {
			t.Fatalf("signed in as %v, want Cara", me.User.ID)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE id = $1 AND cognito_sub = 'sub-cara'`, w.cara.ID); n != 1 {
			t.Error("Cognito's sub wasn't stored")
		}
		// Next time the sub finds her, even if her email in the pool changed.
		b2, r := w.signIn(t, w.c.IDToken(t, "sub-cara", "cara.new@test.example", nil))
		r.Want(t, http.StatusNoContent)
		b2.Do("GET", "/api/me", nil).Decode(t, &me)
		if me.User.ID != w.cara.ID {
			t.Errorf("signed in as %v, want Cara", me.User.ID)
		}
	})
	t.Run("a new pool's sub (the pool was re-created) is stored again by the verified email", func(t *testing.T) {
		w := newCognitoWorld(t)
		w.signIn(t, w.c.IDToken(t, "old-sub", w.cara.Email, nil))
		_, r := w.signIn(t, w.c.IDToken(t, "new-sub", w.cara.Email, nil))
		r.Want(t, http.StatusNoContent)
		if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE id = $1 AND cognito_sub = 'new-sub'`, w.cara.ID); n != 1 {
			t.Error("the new sub wasn't stored")
		}
	})
	t.Run("the first sign-in of an invited account accepts its invite", func(t *testing.T) {
		w := newCognitoWorld(t)
		admin := w.c.SignedIn(t, w.fx.TrustUser(t, w.h.Org, "admin"))
		var inv struct{ UserID uuid.UUID }
		admin.Do("POST", "/api/o/"+w.h.Org.String()+"/staff/invites",
			map[string]any{"email": "zoe@test.example", "hospitalId": w.h.ID, "role": "clinician"}).Decode(t, &inv)
		if calls := w.c.Cognito.Calls(); len(calls) != 1 || calls[0].Op != "CreateUser" {
			t.Fatalf("Cognito calls = %+v; the invite must create the pool's user", calls)
		}
		_, r := w.signIn(t, w.c.IDToken(t, "sub-zoe", "zoe@test.example", nil))
		r.Want(t, http.StatusNoContent)
		if n := count(t, w.c.DB, `SELECT count(*) FROM staff_invites WHERE user_id = $1 AND accepted_at IS NOT NULL`, inv.UserID); n != 1 {
			t.Error("the invite wasn't accepted")
		}
	})
	t.Run("no sign-in: a refused code or token signs no one in", func(t *testing.T) {
		w := newCognitoWorld(t)
		good := w.c.IDToken(t, "sub-cara", w.cara.Email, nil)
		b := w.c.NewBrowser(t)
		r := b.Do("POST", "/api/auth/callback", map[string]string{"code": "never-issued", "verifier": verifier})
		refused(t, r, http.StatusUnauthorized)
		signedOut(t, b, r)

		// A code works once.
		w.c.Cognito.Code("once", verifier, good)
		b.Do("POST", "/api/auth/callback", map[string]string{"code": "once", "verifier": verifier}).Want(t, http.StatusNoContent)
		b2 := w.c.NewBrowser(t)
		refused(t, b2.Do("POST", "/api/auth/callback", map[string]string{"code": "once", "verifier": verifier}), http.StatusUnauthorized)

		// Another browser's verifier doesn't match the code's.
		w.c.Cognito.Code("stolen", verifier, good)
		refused(t, b2.Do("POST", "/api/auth/callback", map[string]string{"code": "stolen", "verifier": strings.Repeat("x", 43)}), http.StatusUnauthorized)

		for name, change := range map[string]func(jwt.MapClaims){
			"expired":          func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-48 * time.Hour).Unix() },
			"unverified email": func(c jwt.MapClaims) { c["email_verified"] = false },
			"an access token":  func(c jwt.MapClaims) { c["token_use"] = "access" },
		} {
			b, r := w.signIn(t, w.c.IDToken(t, "sub-cara", w.cara.Email, change))
			if r.Code != http.StatusUnauthorized {
				t.Errorf("%s: %s", name, r)
			}
			signedOut(t, b, r)
		}
	})
	t.Run("wrong role: someone whose every trust is suspended is refused", func(t *testing.T) {
		w := newCognitoWorld(t)
		if _, err := w.c.DB.Exec(t.Context(), `UPDATE orgs SET status = 'suspended' WHERE id = $1`, w.h.Org); err != nil {
			t.Fatal(err)
		}
		b, r := w.signIn(t, w.c.IDToken(t, "sub-cara", w.cara.Email, nil))
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(r.Message(), "suspended") {
			t.Errorf("message = %q", r.Message())
		}
		signedOut(t, b, r)
		// A platform admin still gets in, to re-activate it (S10 verification 1).
		sam := w.fx.PlatformAdmin(t)
		w.fx.Member(t, sam, w.h, "admin")
		w.c.DB.QueryRow(t.Context(), `SELECT email FROM users WHERE id = $1`, sam.ID).Scan(&sam.Email)
		_, r = w.signIn(t, w.c.IDToken(t, "sub-sam", sam.Email, nil))
		r.Want(t, http.StatusNoContent)
	})
	t.Run("wrong role: an invite that expired before the first sign-in is refused", func(t *testing.T) {
		w := newCognitoWorld(t)
		admin := w.c.SignedIn(t, w.fx.TrustUser(t, w.h.Org, "admin"))
		admin.Do("POST", "/api/o/"+w.h.Org.String()+"/staff/invites",
			map[string]any{"email": "late@test.example", "hospitalId": w.h.ID, "role": "clinician"}).Want(t, http.StatusCreated)
		w.c.Clock.Add(8 * 24 * time.Hour)
		b, r := w.signIn(t, w.c.IDToken(t, "sub-late", "late@test.example", nil))
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(r.Message(), "expired") {
			t.Errorf("message = %q", r.Message())
		}
		signedOut(t, b, r)
	})
	t.Run("wrong role: someone with no Poised account is refused", func(t *testing.T) {
		w := newCognitoWorld(t)
		b, r := w.signIn(t, w.c.IDToken(t, "sub-stranger", "stranger@test.example", nil))
		refused(t, r, http.StatusForbidden)
		signedOut(t, b, r)
		if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE lower(email) = 'stranger@test.example'`); n != 0 {
			t.Error("an account was made")
		}
	})
	t.Run("removed from every trust: signed in, with nowhere to work (as the dev login)", func(t *testing.T) {
		w := newCognitoWorld(t)
		if _, err := w.c.DB.Exec(t.Context(), `DELETE FROM memberships WHERE user_id = $1`, w.cara.ID); err != nil {
			t.Fatal(err)
		}
		b, r := w.signIn(t, w.c.IDToken(t, "sub-cara", w.cara.Email, nil))
		r.Want(t, http.StatusNoContent)
		var me struct{ Trusts []any }
		b.Do("GET", "/api/me", nil).Decode(t, &me)
		if len(me.Trusts) != 0 {
			t.Errorf("trusts = %v", me.Trusts)
		}
		b.Do("GET", w.h.Path("/episodes"), nil).Want(t, http.StatusNotFound)
	})
	t.Run("another tenant: tokens from another pool or for another app client are refused", func(t *testing.T) {
		w := newCognitoWorld(t)
		for name, change := range map[string]func(jwt.MapClaims){
			"another pool":       func(c jwt.MapClaims) { c["iss"] = "https://cognito-idp.xx-test-1.amazonaws.com/xx-test-1_Other" },
			"another app client": func(c jwt.MapClaims) { c["aud"] = "anotherclient" },
		} {
			b, r := w.signIn(t, w.c.IDToken(t, "sub-cara", w.cara.Email, change))
			if r.Code != http.StatusUnauthorized {
				t.Errorf("%s: %s", name, r)
			}
			signedOut(t, b, r)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE cognito_sub IS NOT NULL`); n != 0 {
			t.Error("a sub was stored from a refused token")
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		w := newCognitoWorld(t)
		b := w.c.NewBrowser(t)
		for _, body := range []any{
			`{"code":`,
			map[string]string{"verifier": verifier},
			map[string]string{"code": "c", "verifier": "short"},
			map[string]string{"code": "c", "verifier": verifier + "!"},
			map[string]string{"code": strings.Repeat("c", 5000), "verifier": verifier},
		} {
			refused(t, b.Do("POST", "/api/auth/callback", body), http.StatusBadRequest)
		}
		if calls := w.c.Cognito.Calls(); len(calls) != 0 {
			t.Errorf("Cognito was called: %+v", calls)
		}
	})
}

func TestCognitoCallbackOnlyWithAPool(t *testing.T) {
	t.Parallel()
	c, _ := apitest.New(t) // locally: the dev login, no pool
	c.Do("POST", "/api/auth/callback", map[string]string{"code": "c", "verifier": verifier}).Want(t, http.StatusNotFound)
	// And with a pool there's no dev login (it's off with Cognito on AWS; Options.Cognito turns it off).
	w := newCognitoWorld(t)
	w.c.Do("POST", "/api/dev/login", map[string]any{"userId": w.cara.ID}).Want(t, http.StatusNotFound)
}

// The testers' tools (captured messages, the rate-limit reset, the client address, the scheduled
// jobs) exist only where messages are captured (local and QA). Without the dev login (QA, S12)
// they need a signed-in platform admin; with it (local) anyone can be anyone, so they're open.
func TestQAToolsNeedAPlatformAdminWithoutTheDevLogin(t *testing.T) {
	t.Parallel()
	type call struct {
		method, path string
		body         any
	}
	tools := []call{
		{"GET", "/api/qa/messages", nil},
		{"DELETE", "/api/qa/rate-limits", nil},
		{"GET", "/api/qa/client", nil},
		{"GET", "/api/qa/jobs", nil},
		{"POST", "/api/qa/jobs/run", map[string]string{"asOf": time.Now().UTC().Format(time.RFC3339)}},
		{"PUT", "/api/qa/jobs/clock", map[string]int{"offsetMinutes": 0}},
	}
	c, fx := apitest.NewWith(t, apitest.Options{Cognito: true, Capture: true})
	h := fx.Hospital(t)
	admin := c.SignedIn(t, fx.TrustUser(t, h.Org, "admin"))
	sam := c.SignedIn(t, fx.PlatformAdmin(t))

	for _, tool := range tools {
		t.Run(tool.method+" "+tool.path, func(t *testing.T) {
			refused(t, c.Do(tool.method, tool.path, tool.body), http.StatusUnauthorized)  // no sign-in
			refused(t, admin.Do(tool.method, tool.path, tool.body), http.StatusForbidden) // a trust's admin isn't enough
			if r := sam.Do(tool.method, tool.path, tool.body); r.Code >= 300 {
				t.Errorf("platform admin: %s", r)
			}
		})
	}
	t.Run("invalid input still refused for the platform admin", func(t *testing.T) {
		refused(t, sam.Do("POST", "/api/qa/jobs/run", map[string]string{"asOf": "yesterday"}), http.StatusBadRequest)
	})
	t.Run("with the dev login (local), open as before", func(t *testing.T) {
		local, _ := apitest.NewWith(t, apitest.Options{Capture: true})
		for _, tool := range tools {
			if r := local.Do(tool.method, tool.path, tool.body); r.Code >= 300 {
				t.Errorf("%s %s: %s", tool.method, tool.path, r)
			}
		}
	})
}
