// Package auth holds sign-in sessions. A session id is 32 random bytes kept only in the
// browser's cookie; the database stores its SHA-256 hash, so a copied sessions row can't be
// used to sign in. The session says only who the user is: hospitals and roles always come
// from the memberships table.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CookieName is the staff session cookie on plain-HTTP local development.
const CookieName = "poised_session"

// SecureCookieName is the session cookie everywhere else. The __Host- prefix makes the browser
// accept it only when Secure, with Path=/ and no Domain, so a subdomain can't set or replace it.
const SecureCookieName = "__Host-" + CookieName

// PatientCookieName is the patient session cookie: a separate cookie, so one browser can hold a
// staff and a patient session at once, and neither is ever taken for the other.
const PatientCookieName = "poised_patient"

// SecurePatientCookieName is PatientCookieName with the __Host- prefix, for Secure cookies.
const SecurePatientCookieName = "__Host-" + PatientCookieName

// Default session limits for staff: signed out 8 hours after signing in, or after 30 minutes
// without a request, whichever comes first.
const (
	DefaultMaxAge = 8 * time.Hour
	DefaultIdle   = 30 * time.Minute
)

// User is the signed-in person a session belongs to.
type User struct {
	ID   uuid.UUID
	Name string
}

// NewSessionID returns a new random session id for the cookie and the hash to store for it.
func NewSessionID() (id string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("new session id: %w", err)
	}
	id = base64.RawURLEncoding.EncodeToString(b)
	return id, HashToken(id), nil
}

// HashToken returns the SHA-256 the database keeps in place of a secret token: a session id
// or a patient link.
func HashToken(id string) []byte {
	sum := sha256.Sum256([]byte(id))
	return sum[:]
}
