package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
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

// devLogin signs in as a seeded user and sets the token cookie. Replaced by SSO later.
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
	token, err := auth.Issue(s.secret, auth.User{ID: u.ID, Name: u.Name}, time.Now())
	if err != nil {
		serverError(w, "issue token", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(auth.TTL.Seconds()),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

type hospitalRoles struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Roles []string  `json:"roles"`
}

// me returns the signed-in user and their hospitals with roles.
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
	hospitals := []hospitalRoles{}
	for _, m := range rows {
		if n := len(hospitals); n > 0 && hospitals[n-1].ID == m.HospitalID {
			hospitals[n-1].Roles = append(hospitals[n-1].Roles, m.Role)
			continue
		}
		hospitals = append(hospitals, hospitalRoles{ID: m.HospitalID, Name: m.HospitalName, Roles: []string{m.Role}})
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "hospitals": hospitals})
}
