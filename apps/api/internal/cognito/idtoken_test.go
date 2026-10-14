package cognito_test

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
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
)

const (
	issuer = "https://cognito-idp.ca-central-1.amazonaws.com/ca-central-1_AbCdEf123"
	client = "1example23456789client"
)

var now = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// claims are a good ID token's, as Cognito issues them after the managed login.
func claims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "5f1c-sub", "email": "cara@trust.example", "email_verified": true,
		"iss": issuer, "aud": client, "token_use": "id",
		"iat": now.Add(-time.Minute).Unix(), "exp": now.Add(time.Hour).Unix(),
	}
}

func sign(t *testing.T, k *rsa.PrivateKey, kid string, c jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(k)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func verifier(k *rsa.PrivateKey) cognito.Verifier {
	return cognito.Verifier{Issuer: issuer, ClientID: client, Keys: cognito.StaticKeys{"key-1": &k.PublicKey}}
}

func TestVerifyAGoodIDToken(t *testing.T) {
	k := newKey(t)
	id, err := verifier(k).Verify(context.Background(), sign(t, k, "key-1", claims()), now)
	if err != nil {
		t.Fatal(err)
	}
	if id != (cognito.Identity{Sub: "5f1c-sub", Email: "cara@trust.example"}) {
		t.Errorf("identity = %+v", id)
	}
}

func TestVerifyLowerCasesTheEmail(t *testing.T) {
	k := newKey(t)
	c := claims()
	c["email"] = "Cara@Trust.Example"
	id, err := verifier(k).Verify(context.Background(), sign(t, k, "key-1", c), now)
	if err != nil || id.Email != "cara@trust.example" {
		t.Fatalf("got %+v, %v", id, err)
	}
}

func TestVerifyRefuses(t *testing.T) {
	k, other := newKey(t), newKey(t)
	cases := map[string]func() string{
		"another key's signature": func() string { return sign(t, other, "key-1", claims()) },
		"an unknown key id":       func() string { return sign(t, k, "key-2", claims()) },
		"another pool":            func() string { c := claims(); c["iss"] = issuer + "x"; return sign(t, k, "key-1", c) },
		"another app client":      func() string { c := claims(); c["aud"] = "someoneelse"; return sign(t, k, "key-1", c) },
		"an access token":         func() string { c := claims(); c["token_use"] = "access"; return sign(t, k, "key-1", c) },
		"expired":                 func() string { c := claims(); c["exp"] = now.Add(-time.Second).Unix(); return sign(t, k, "key-1", c) },
		"no expiry":               func() string { c := claims(); delete(c, "exp"); return sign(t, k, "key-1", c) },
		"an unverified email":     func() string { c := claims(); c["email_verified"] = false; return sign(t, k, "key-1", c) },
		"no email":                func() string { c := claims(); delete(c, "email"); return sign(t, k, "key-1", c) },
		"no subject":              func() string { c := claims(); delete(c, "sub"); return sign(t, k, "key-1", c) },
		"HS256 with the public key as secret": func() string {
			tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims())
			tok.Header["kid"] = "key-1"
			s, _ := tok.SignedString(k.PublicKey.N.Bytes())
			return s
		},
		"alg none": func() string {
			tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims())
			tok.Header["kid"] = "key-1"
			s, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
			return s
		},
		"not a token": func() string { return "not.a.token" },
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if id, err := verifier(k).Verify(context.Background(), token(), now); err == nil {
				t.Errorf("accepted: %+v", id)
			}
		})
	}
}

// jwksServer serves k's public key as a JWKS, as the pool does at <issuer>/.well-known/jwks.json.
func jwksServer(t *testing.T, kid string, k *rsa.PrivateKey, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	b64 := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": kid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": b64(k.PublicKey.N.Bytes()), "e": b64(big.NewInt(int64(k.PublicKey.E)).Bytes()),
		}}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestJWKSFetchesTheKeysOnceAndCachesThem(t *testing.T) {
	k := newKey(t)
	var hits atomic.Int32
	srv := jwksServer(t, "key-1", k, &hits)
	v := cognito.Verifier{Issuer: issuer, ClientID: client, Keys: &cognito.JWKS{URL: srv.URL}}
	for range 3 {
		if _, err := v.Verify(context.Background(), sign(t, k, "key-1", claims()), now); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 1 {
		t.Errorf("fetched the keys %d times; want once", hits.Load())
	}
	// An unknown key id (e.g. the pool rotated its keys) fetches again, but not on every request.
	for range 3 {
		if _, err := v.Verify(context.Background(), sign(t, k, "key-9", claims()), now); err == nil {
			t.Fatal("accepted an unknown key")
		}
	}
	if hits.Load() > 2 {
		t.Errorf("fetched the keys %d times; unknown key ids mustn't fetch every time", hits.Load())
	}
}

func TestJWKSURLIsTheIssuers(t *testing.T) {
	if got := cognito.JWKSURL(issuer); got != issuer+"/.well-known/jwks.json" || !strings.HasPrefix(got, "https://") {
		t.Errorf("got %q", got)
	}
}
