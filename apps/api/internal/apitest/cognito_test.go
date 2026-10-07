//go:build integration

package apitest_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

// fakeSignIn stands in for Cognito's token endpoint: each code names one identity.
type fakeSignIn map[string]cognito.Identity

func (f fakeSignIn) Exchange(_ context.Context, code, verifier string) (cognito.Identity, error) {
	if id, ok := f[code]; ok && verifier == "the-verifier" {
		return id, nil
	}
	return cognito.Identity{}, errors.New("invalid_grant")
}

// fakeDirectory records invites; an email invited twice keeps its first subject, as Cognito does.
type fakeDirectory struct{ subs map[string]string }

func (f *fakeDirectory) Invite(_ context.Context, email, name string) (string, error) {
	if sub, ok := f.subs[email]; ok {
		return sub, nil
	}
	f.subs[email] = "sub-" + uuid.NewString()
	return f.subs[email], nil
}

var testLogin = &httpapi.CognitoLogin{
	AuthorizeURL: "https://pool.example/oauth2/authorize", LogoutURL: "https://pool.example/logout",
	ClientID: "client", RedirectURI: "http://localhost:3000/auth/callback", LogoutRedirectURI: "http://localhost:3000/login",
}

func withCognito(t *testing.T, signIn fakeSignIn, dir *fakeDirectory) *apitest.Client {
	t.Helper()
	cfg := httpapi.Config{DevLogin: true, SignIn: signIn, Login: testLogin}
	if dir != nil {
		cfg.Directory = dir
	}
	return apitest.NewWith(t, cfg)
}

func TestAuthConfig(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		c    *apitest.Client
		want []string
	}{
		{"no Cognito", apitest.New(t), []string{`"cognito":null`, `"devLogin":true`}},
		{"Cognito, no dev login", apitest.NewWith(t, httpapi.Config{SignIn: fakeSignIn{}, Login: testLogin}), []string{`"clientId":"client"`, `"authorizeUrl":"https://pool.example/oauth2/authorize"`, `"devLogin":false`}},
	} {
		r := tc.c.Do(http.MethodGet, "/api/auth/config", nil)
		for _, w := range tc.want {
			if r.Status != http.StatusOK || !strings.Contains(string(r.Body), w) {
				t.Errorf("%s: %d %s, want %s", tc.name, r.Status, r.Body, w)
			}
		}
	}
}

func TestCognitoCallback(t *testing.T) {
	t.Parallel()
	signIn := fakeSignIn{}
	c := withCognito(t, signIn, nil)
	h := c.Hospital("H")
	callback := func(code string) *apitest.Resp {
		return c.NewBrowser().Do(http.MethodPost, "/api/auth/callback", map[string]string{"code": code, "verifier": "the-verifier"})
	}

	// Invited: the subject is already on the user.
	invited := c.User(h, "clinician")
	if _, err := c.DB.Exec(t.Context(), "UPDATE users SET cognito_sub = 'sub-invited' WHERE id = $1", invited); err != nil {
		t.Fatal(err)
	}
	signIn["c-invited"] = cognito.Identity{Subject: "sub-invited", Email: "whatever@x.example"}
	r := callback("c-invited")
	if r.Status != http.StatusNoContent {
		t.Fatalf("invited: status = %d: %s", r.Status, r.Body)
	}
	if cookie := sessionCookie(t, r); cookie.Value == "" {
		t.Error("invited: no session cookie")
	}

	// Known by email only (first sign-in): matched case-insensitively, and the subject is saved.
	byEmail := c.User(h, "clinician")
	var email string
	if err := c.DB.QueryRow(t.Context(), "SELECT email FROM users WHERE id = $1", byEmail).Scan(&email); err != nil {
		t.Fatal(err)
	}
	signIn["c-email"] = cognito.Identity{Subject: "sub-new", Email: strings.ToUpper(email)}
	if r := callback("c-email"); r.Status != http.StatusNoContent {
		t.Errorf("by email: status = %d: %s", r.Status, r.Body)
	}
	var sub string
	if err := c.DB.QueryRow(t.Context(), "SELECT coalesce(cognito_sub, '') FROM users WHERE id = $1", byEmail).Scan(&sub); err != nil {
		t.Fatal(err)
	}
	if sub != "sub-new" {
		t.Errorf("cognito_sub = %q, want sub-new", sub)
	}

	// Refused: unknown person, no access, a suspended trust, a code Cognito rejects, bad input.
	signIn["c-stranger"] = cognito.Identity{Subject: "sub-stranger", Email: "stranger@x.example"}
	noAccess := c.User(h, "clinician")
	if _, err := c.DB.Exec(t.Context(), "UPDATE users SET cognito_sub = 'sub-noaccess' WHERE id = $1", noAccess); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(t.Context(), "DELETE FROM memberships WHERE user_id = $1", noAccess); err != nil {
		t.Fatal(err)
	}
	signIn["c-noaccess"] = cognito.Identity{Subject: "sub-noaccess", Email: "noaccess@x.example"}
	suspended := c.User(c.Hospital("Suspended"), "admin")
	if _, err := c.DB.Exec(t.Context(), `UPDATE users SET cognito_sub = 'sub-suspended' WHERE id = $1;`, suspended); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(t.Context(), `UPDATE orgs SET status = 'suspended' WHERE id = (SELECT org_id FROM memberships WHERE user_id = $1)`, suspended); err != nil {
		t.Fatal(err)
	}
	signIn["c-suspended"] = cognito.Identity{Subject: "sub-suspended", Email: "suspended@x.example"}
	for _, tc := range []struct {
		code string
		want int
	}{
		{"c-stranger", http.StatusForbidden},
		{"c-noaccess", http.StatusForbidden},
		{"c-suspended", http.StatusForbidden},
		{"rejected-by-cognito", http.StatusUnauthorized},
	} {
		r := callback(tc.code)
		if r.Status != tc.want {
			t.Errorf("%s: status = %d, want %d: %s", tc.code, r.Status, tc.want, r.Body)
		}
		if len(r.Header.Values("Set-Cookie")) != 0 {
			t.Errorf("%s: set a cookie", tc.code)
		}
	}
	if r := c.Do(http.MethodPost, "/api/auth/callback", map[string]string{"code": ""}); r.Status != http.StatusBadRequest {
		t.Errorf("empty code: status = %d, want 400", r.Status)
	}

	// Without Cognito the route doesn't exist.
	if r := apitest.New(t).Do(http.MethodPost, "/api/auth/callback", map[string]string{"code": "x", "verifier": "y"}); r.Status != http.StatusNotFound {
		t.Errorf("no Cognito: status = %d, want 404", r.Status)
	}
}

// Staff invites and the staff list belong to a trust-wide admin (STF-01, STF-02).
func TestInviteStaff(t *testing.T) {
	t.Parallel()
	dir := &fakeDirectory{subs: map[string]string{}}
	c := withCognito(t, fakeSignIn{}, dir)
	org := c.Org("Trust")
	h := c.HospitalIn(org, "H")
	other := c.Hospital("Other")
	path := "/api/o/" + org.String() + "/staff"
	c.LoginAs(c.TrustUser(org, "admin"))

	invite := func(body map[string]any) *apitest.Resp { return c.Do(http.MethodPost, path+"/invites", body) }
	r := invite(map[string]any{"email": " New.Person@Example.org ", "name": "New Person", "role": "clinician", "hospitalId": h})
	if r.Status != http.StatusCreated {
		t.Fatalf("invite: status = %d: %s", r.Status, r.Body)
	}
	var userID uuid.UUID
	var sub string
	if err := c.DB.QueryRow(t.Context(), "SELECT id, cognito_sub FROM users WHERE email = 'new.person@example.org'").Scan(&userID, &sub); err != nil {
		t.Fatalf("invited user row: %v", err)
	}
	if sub != dir.subs["new.person@example.org"] {
		t.Errorf("cognito_sub = %q, want the directory's", sub)
	}

	// The same person again, for the whole trust: one more membership, still one user (STF-02).
	if r := invite(map[string]any{"email": "new.person@example.org", "name": "New Person", "role": "super_clinician"}); r.Status != http.StatusCreated {
		t.Errorf("second scope: status = %d: %s", r.Status, r.Body)
	}
	var users, memberships int
	if err := c.DB.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM users WHERE email = 'new.person@example.org'),
		(SELECT count(*) FROM memberships WHERE user_id = $1)`, userID).Scan(&users, &memberships); err != nil {
		t.Fatal(err)
	}
	if users != 1 || memberships != 2 {
		t.Errorf("users %d, memberships %d; want 1 and 2", users, memberships)
	}
	if r := invite(map[string]any{"email": "new.person@example.org", "name": "New Person", "role": "admin", "hospitalId": h}); r.Status != http.StatusConflict {
		t.Errorf("same scope again: status = %d, want 409", r.Status)
	}

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"no email", map[string]any{"email": "nope", "name": "X", "role": "clinician"}},
		{"no name", map[string]any{"email": "a@b.example", "name": " ", "role": "clinician"}},
		{"old role", map[string]any{"email": "a@b.example", "name": "X", "role": "viewer"}},
		{"hospital of another trust", map[string]any{"email": "a@b.example", "name": "X", "role": "clinician", "hospitalId": other}},
	} {
		if r := invite(tc.body); r.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, r.Status)
		}
	}

	r = c.Do(http.MethodGet, path, nil)
	if r.Status != http.StatusOK || !strings.Contains(string(r.Body), "new.person@example.org") {
		t.Errorf("staff list: %d %s", r.Status, r.Body)
	}

	// Not a trust-wide admin: refused. Another trust: not found.
	for _, tc := range []struct {
		name string
		user uuid.UUID
		want int
	}{
		{"hospital admin", c.User(h, "admin"), http.StatusForbidden},
		{"trust super clinician", c.TrustUser(org, "super_clinician"), http.StatusForbidden},
		{"admin of another trust", c.TrustUser(c.OrgOf(other), "admin"), http.StatusNotFound},
	} {
		c.LoginAs(tc.user)
		if r := c.Do(http.MethodGet, path, nil); r.Status != tc.want {
			t.Errorf("%s, list: status = %d, want %d", tc.name, r.Status, tc.want)
		}
		if r := invite(map[string]any{"email": "z@z.example", "name": "Z", "role": "clinician"}); r.Status != tc.want {
			t.Errorf("%s, invite: status = %d, want %d", tc.name, r.Status, tc.want)
		}
	}
}

func TestInviteNeedsCognito(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	org := c.Org("Trust")
	c.LoginAs(c.TrustUser(org, "admin"))
	r := c.Do(http.MethodPost, "/api/o/"+org.String()+"/staff/invites", map[string]any{"email": "a@b.example", "name": "A", "role": "clinician"})
	if r.Status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", r.Status)
	}
}
