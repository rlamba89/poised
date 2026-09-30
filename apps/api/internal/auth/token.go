// Package auth issues and verifies the dev login token. The token carries only
// who the user is; hospitals and roles always come from the memberships table.
// Later, the same Verify step also accepts a real IdP token.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CookieName is the httpOnly cookie that holds the token.
const CookieName = "sj_token"

// TTL is how long a dev login lasts.
const TTL = 12 * time.Hour

// User is the identity inside a verified token.
type User struct {
	ID   uuid.UUID
	Name string
}

type claims struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

// Issue signs an HS256 token with sub and name, valid for TTL from now.
func Issue(secret []byte, u User, now time.Time) (string, error) {
	c := claims{
		Name: u.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret)
}

// Verify checks the signature, algorithm and expiry, and returns the user.
func Verify(secret []byte, token string, now time.Time) (User, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return User{}, fmt.Errorf("verify token: %w", err)
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return User{}, errors.New("verify token: subject is not a user id")
	}
	return User{ID: id, Name: c.Name}, nil
}
