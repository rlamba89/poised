package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// The patient's session (plan C2, A-4, PAT-04): after their link and date of birth, a signed,
// HTTP-only cookie names their session, a patient_sessions row (never their patient record: S14
// answers 4). The row is what counts: Sign out, a resend or a lock ends it there. The cookie also
// expires PatientIdle after the last request: each request issues it again with a fresh expiry.

// PatientCookieName is the patient's session cookie. It's sent only to the patient API (/api/p).
const PatientCookieName = "poised_patient"

// PatientIdle is the idle timeout: the session ends this long after the patient's last request.
const PatientIdle = 30 * time.Minute

// patientAudience marks a patient's session, so it's never taken for a staff sign-in (Verify
// refuses any token with an audience) or the other way round.
const patientAudience = "patient"

// IssuePatient signs a cookie for the patient session (its patient_sessions id), valid for
// PatientIdle from now.
func IssuePatient(secret []byte, session uuid.UUID, now time.Time) (string, error) {
	c := jwt.RegisteredClaims{
		Subject:   session.String(),
		Audience:  jwt.ClaimStrings{patientAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(PatientIdle)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret)
}

// VerifyPatient checks a patient's session cookie and returns the session's id. The caller
// still checks the session in the database.
func VerifyPatient(secret []byte, token string, now time.Time) (uuid.UUID, error) {
	var c jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(patientAudience),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("verify patient session: %w", err)
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, errors.New("verify patient session: subject is not a session id")
	}
	return id, nil
}

// NewLinkToken returns a random, URL-safe token for a patient's link: 32 bytes, 43 characters.
// Only its hash (HashLinkToken) is stored, so the database alone can't open anyone's link.
func NewLinkToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashLinkToken is what login_links stores for a token: its SHA-256. A plain hash is enough:
// the token is 32 random bytes, so it can't be guessed from its hash.
func HashLinkToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
