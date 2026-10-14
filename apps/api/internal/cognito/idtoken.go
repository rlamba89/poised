package cognito

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Identity is who an ID token says signed in: Cognito's user (sub) and their email, which
// Cognito has verified (accounts are created by AdminCreateUser with the email marked verified,
// there's no self sign-up, and the app client can't change the email).
type Identity struct {
	Sub   string
	Email string // lower case
}

// Keys gives the public key that signed a token, by its key id ("kid").
type Keys interface {
	Key(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

// Verifier checks an ID token from the staff user pool (S12): signed (RS256) by one of the pool's
// keys, issued by the pool, for our app client, an ID token (not an access token), unexpired,
// with a verified email.
type Verifier struct {
	Issuer   string // config.Cognito.Issuer()
	ClientID string
	Keys     Keys // the pool's JWKS; tests use StaticKeys
}

type idClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	TokenUse      string `json:"token_use"`
	jwt.RegisteredClaims
}

// Verify returns the token's identity, or an error saying why it was refused.
func (v Verifier) Verify(ctx context.Context, raw string, now time.Time) (Identity, error) {
	var c idClaims
	_, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("no key id")
		}
		return v.Keys.Key(ctx, kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(v.Issuer),
		jwt.WithAudience(v.ClientID),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	switch {
	case err != nil:
		return Identity{}, fmt.Errorf("id token: %w", err)
	case c.TokenUse != "id":
		return Identity{}, fmt.Errorf("id token: token_use is %q, not id", c.TokenUse)
	case c.Subject == "":
		return Identity{}, errors.New("id token: no subject")
	case c.Email == "" || !c.EmailVerified:
		return Identity{}, errors.New("id token: no verified email")
	}
	return Identity{Sub: c.Subject, Email: strings.ToLower(c.Email)}, nil
}

// StaticKeys is a fixed key set, for tests.
type StaticKeys map[string]*rsa.PublicKey

func (s StaticKeys) Key(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if k, ok := s[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown key id %q", kid)
}

// JWKSURL is where a pool publishes its public keys.
func JWKSURL(issuer string) string { return issuer + "/.well-known/jwks.json" }

// refetchAfter limits how often an unknown key id fetches the keys again, so tokens with made-up
// key ids can't make every request call Cognito.
const refetchAfter = time.Minute

// JWKS fetches the pool's public keys on first use and keeps them for the life of the process
// (a Lambda instance). An unknown key id fetches them again, at most once a minute, in case the
// pool rotated its keys.
type JWKS struct {
	URL  string
	HTTP *http.Client // nil: a client with a 5-second timeout

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

func (j *JWKS) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if k, ok := j.keys[kid]; ok {
		return k, nil
	}
	if j.keys != nil && time.Since(j.fetched) < refetchAfter {
		return nil, fmt.Errorf("unknown key id %q", kid)
	}
	keys, err := j.fetch(ctx)
	if err != nil {
		return nil, err
	}
	j.keys, j.fetched = keys, time.Now()
	if k, ok := keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown key id %q", kid)
}

func (j *JWKS) fetch(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	hc := j.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.URL, nil)
	if err != nil {
		return nil, err
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch the pool's keys: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch the pool's keys: %s", res.Status)
	}
	var set struct {
		Keys []struct{ Kid, Kty, N, E string }
	}
	if err := json.NewDecoder(res.Body).Decode(&set); err != nil {
		return nil, fmt.Errorf("read the pool's keys: %w", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, errN := base64.RawURLEncoding.DecodeString(k.N)
		e, errE := base64.RawURLEncoding.DecodeString(k.E)
		if errN != nil || errE != nil || len(e) > 4 {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	return keys, nil
}
