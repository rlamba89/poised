package httpapi

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// devUsers lists seeded users for the stub login page.
func (s *server) devUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.q.ListUsers(r.Context())
	if err != nil {
		serverError(w, "list users", err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// devLogin signs in as a seeded user: it starts a session and sets its cookie. Replaced by
// Cognito sign-in (auth plan Step 4) everywhere but local development and QA.
func (s *server) devLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID uuid.UUID `json:"userId"`
	}
	if !readJSON(w, r, 1<<10, &body) {
		return
	}
	u, err := s.q.GetUser(r.Context(), body.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "That user does not exist.")
		return
	}
	if err != nil {
		serverError(w, "get user", err)
		return
	}
	id, hash, err := auth.NewSessionID()
	if err != nil {
		serverError(w, "new session", err)
		return
	}
	err = s.q.CreateSession(r.Context(), db.CreateSessionParams{IDHash: hash, UserID: u.ID, MaxAgeSeconds: s.sessionMaxAge.Seconds()})
	if err != nil {
		serverError(w, "create session", err)
		return
	}
	http.SetCookie(w, s.cookie(s.cookieName(), id, int(s.sessionMaxAge.Seconds())))
	w.WriteHeader(http.StatusNoContent)
}

// logout ends the session in the database, so its cookie stops working even if it was copied,
// then clears the cookie.
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.cookieName()); err == nil {
		if err := s.q.DeleteSession(r.Context(), auth.HashToken(c.Value)); err != nil {
			serverError(w, "delete session", err)
			return
		}
	}
	http.SetCookie(w, s.cookie(s.cookieName(), "", -1))
	w.WriteHeader(http.StatusNoContent)
}

// cookieName is the staff session cookie's name: with the __Host- prefix whenever it is Secure.
func (s *server) cookieName() string {
	if s.secureCookies {
		return auth.SecureCookieName
	}
	return auth.CookieName
}

// patientCookieName is the patient session cookie's name, chosen the same way.
func (s *server) patientCookieName() string {
	if s.secureCookies {
		return auth.SecurePatientCookieName
	}
	return auth.PatientCookieName
}

// cookie is a session cookie with the same attributes when it is set and when it is cleared,
// so the browser treats both as one cookie. maxAge < 0 deletes it.
func (s *server) cookie(name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: "/", MaxAge: maxAge,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookies,
	}
}

// hospitalAccess is one hospital the user can open, its trust, and the highest role that
// applies there (a trust-wide role and a hospital role can both cover it).
type hospitalAccess struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	OrgID   uuid.UUID `json:"orgId"`
	OrgName string    `json:"orgName"`
	Role    role.Role `json:"role"`
}

// me returns the signed-in user and every hospital they can open, by trust.
func (s *server) me(w http.ResponseWriter, r *http.Request) {
	u, err := s.q.GetUser(r.Context(), currentUser(r).ID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	if err != nil {
		serverError(w, "get user", err)
		return
	}
	rows, err := s.q.ListMemberships(r.Context(), u.ID)
	if err != nil {
		serverError(w, "list memberships", err)
		return
	}
	hospitals := []hospitalAccess{}
	at := map[uuid.UUID]int{} // hospital → its index in hospitals
	for _, m := range rows {
		if i, ok := at[m.HospitalID]; ok {
			hospitals[i].Role = role.Highest(hospitals[i].Role, role.Role(m.Role))
			continue
		}
		at[m.HospitalID] = len(hospitals)
		hospitals = append(hospitals, hospitalAccess{
			ID: m.HospitalID, Name: m.HospitalName, OrgID: m.OrgID, OrgName: m.OrgName, Role: role.Role(m.Role),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "hospitals": hospitals})
}
