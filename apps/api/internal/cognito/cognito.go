// Package cognito signs staff in with Amazon Cognito (decision A-12). After Cognito's managed
// login, the browser brings back a one-time code; Exchange swaps it for an ID token and checks
// that token before trusting who it names. Roles never come from Cognito: they are in our
// memberships table.
package cognito

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Identity is the person an ID token names: Cognito's stable user id and their verified email.
type Identity struct {
	Subject string
	Email   string // lower case
}

// Client talks to one user pool through one app client.
type Client struct {
	domain      string // the managed login domain, e.g. https://<prefix>.auth.eu-west-2.amazoncognito.com
	clientID    string
	redirectURI string
	issuer      string // https://cognito-idp.<region>.amazonaws.com/<pool id>
	http        *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey // the pool's signing keys, by kid
	refreshed time.Time
}

// New returns a Client. The app client is public (no secret): PKCE protects the code.
func New(domain, clientID, redirectURI, issuer string, httpClient *http.Client) *Client {
	return &Client{domain: strings.TrimRight(domain, "/"), clientID: clientID, redirectURI: redirectURI, issuer: issuer, http: httpClient}
}

// Exchange swaps an authorization code and its PKCE verifier for the signed-in person's identity.
func (c *Client) Exchange(ctx context.Context, code, verifier string) (Identity, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {c.clientID},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {c.redirectURI},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.domain+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return Identity{}, fmt.Errorf("token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.http.Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("token request: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return Identity{}, fmt.Errorf("token response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return Identity{}, fmt.Errorf("token endpoint: %s: %s", res.Status, body)
	}
	var tokens struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokens); err != nil {
		return Identity{}, fmt.Errorf("token response: %w", err)
	}
	return c.verify(ctx, tokens.IDToken)
}

type idClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	TokenUse      string `json:"token_use"`
	jwt.RegisteredClaims
}

// verify checks the ID token's signature (RS256, the pool's keys), issuer, audience, use and
// expiry, and that it names a subject with a verified email.
func (c *Client) verify(ctx context.Context, token string) (Identity, error) {
	var cl idClaims
	_, err := jwt.ParseWithClaims(token, &cl, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return c.key(ctx, kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(c.issuer),
		jwt.WithAudience(c.clientID),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Identity{}, fmt.Errorf("id token: %w", err)
	}
	switch {
	case cl.TokenUse != "id":
		return Identity{}, errors.New("id token: token_use is not id")
	case cl.Subject == "":
		return Identity{}, errors.New("id token: no subject")
	case cl.Email == "" || !cl.EmailVerified:
		return Identity{}, errors.New("id token: no verified email")
	}
	return Identity{Subject: cl.Subject, Email: strings.ToLower(cl.Email)}, nil
}

// key returns the pool's signing key kid, fetching the key set when it is unknown (Cognito
// rotates keys rarely). Fetches are at most one a minute.
func (c *Client) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k, ok := c.keys[kid]; ok {
		return k, nil
	}
	if time.Since(c.refreshed) < time.Minute && c.keys != nil {
		return nil, fmt.Errorf("unknown signing key %q", kid)
	}
	keys, err := c.fetchKeys(ctx)
	if err != nil {
		return nil, err
	}
	c.keys, c.refreshed = keys, time.Now()
	if k, ok := keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown signing key %q", kid)
}

func (c *Client) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return nil, fmt.Errorf("jwks: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwks: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks: %s", res.Status)
	}
	var set struct {
		Keys []struct{ Kty, Kid, N, E string } `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&set); err != nil {
		return nil, fmt.Errorf("jwks: %w", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("jwks: key %q is not base64url", k.Kid)
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	return keys, nil
}
