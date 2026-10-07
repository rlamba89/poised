package cognito

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	clientID    = "test-client"
	redirectURI = "http://localhost:3000/auth/callback"
	kid         = "key-1"
)

// fakeCognito serves a JWKS and a token endpoint, like a user pool and its domain.
type fakeCognito struct {
	key     *rsa.PrivateKey
	srv     *httptest.Server
	idToken string            // what the token endpoint returns
	form    map[string]string // what the token endpoint was sent
}

func newFakeCognito(t *testing.T) *fakeCognito {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeCognito{key: key}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /pool/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": kid,
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		f.form = map[string]string{}
		for k := range r.PostForm {
			f.form[k] = r.PostForm.Get(k)
		}
		json.NewEncoder(w).Encode(map[string]string{"id_token": f.idToken, "access_token": "x", "token_type": "Bearer"})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeCognito) issuer() string { return f.srv.URL + "/pool" }

func (f *fakeCognito) client() *Client {
	return New(f.srv.URL, clientID, redirectURI, f.issuer(), f.srv.Client())
}

// sign makes an ID token; change edits the claims first.
func (f *fakeCognito) sign(t *testing.T, key *rsa.PrivateKey, change func(jwt.MapClaims)) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": "sub-123", "email": "Ada@Example.org", "email_verified": true,
		"iss": f.issuer(), "aud": clientID, "token_use": "id",
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}
	if change != nil {
		change(claims)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestExchange(t *testing.T) {
	f := newFakeCognito(t)
	f.idToken = f.sign(t, f.key, nil)

	id, err := f.client().Exchange(context.Background(), "the-code", "the-verifier")
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "sub-123" || id.Email != "ada@example.org" {
		t.Errorf("identity = %+v, want sub-123 and the email in lower case", id)
	}
	want := map[string]string{
		"grant_type": "authorization_code", "client_id": clientID, "code": "the-code",
		"code_verifier": "the-verifier", "redirect_uri": redirectURI,
	}
	for k, v := range want {
		if f.form[k] != v {
			t.Errorf("token request %s = %q, want %q", k, f.form[k], v)
		}
	}
}

func TestExchangeRefusesBadTokens(t *testing.T) {
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := newFakeCognito(t)
	for _, tc := range []struct {
		name  string
		token func() string
	}{
		{"expired", func() string {
			return f.sign(t, f.key, func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() })
		}},
		{"another client", func() string { return f.sign(t, f.key, func(c jwt.MapClaims) { c["aud"] = "someone-else" }) }},
		{"another issuer", func() string {
			return f.sign(t, f.key, func(c jwt.MapClaims) { c["iss"] = "https://evil.example/pool" })
		}},
		{"an access token", func() string { return f.sign(t, f.key, func(c jwt.MapClaims) { c["token_use"] = "access" }) }},
		{"email not verified", func() string { return f.sign(t, f.key, func(c jwt.MapClaims) { c["email_verified"] = false }) }},
		{"no subject", func() string { return f.sign(t, f.key, func(c jwt.MapClaims) { delete(c, "sub") }) }},
		{"signed by another key", func() string { return f.sign(t, other, nil) }},
		{"HS256 with the client id as secret", func() string {
			s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "aud": clientID, "iss": f.issuer(), "token_use": "id", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(clientID))
			return s
		}},
		{"not a token", func() string { return "nonsense" }},
	} {
		f.idToken = tc.token()
		if _, err := f.client().Exchange(context.Background(), "code", "verifier"); err == nil {
			t.Errorf("%s: accepted", tc.name)
		}
	}
}

func TestExchangeReportsTokenEndpointErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	_, err := New(srv.URL, clientID, redirectURI, srv.URL+"/pool", srv.Client()).Exchange(context.Background(), "used-code", "v")
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("err = %v, want the token endpoint's invalid_grant", err)
	}
}
