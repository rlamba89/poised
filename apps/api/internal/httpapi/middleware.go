package httpapi

import (
	"context"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

type ctxKey int

const (
	userKey ctxKey = iota
	rolesKey
)

// RoleLister loads a user's roles in one hospital. *db.Queries satisfies it.
type RoleLister interface {
	ListRolesInHospital(ctx context.Context, arg db.ListRolesInHospitalParams) ([]string, error)
}

// requireUser rejects requests without a valid token cookie and puts the user in the context.
func requireUser(secret []byte, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.CookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Please sign in.")
			return
		}
		u, err := auth.Verify(secret, c.Value, time.Now())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Your session has expired. Please sign in again.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	}
}

// requireHospital checks the caller is a member of the {hid} hospital (NFR-04) and
// puts their roles there in the context. Non-members get 404 so hospitals can't be probed.
// It must run inside requireUser.
func requireHospital(roles RoleLister, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hid, err := uuid.Parse(r.PathValue("hid"))
		if err != nil {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return
		}
		got, err := roles.ListRolesInHospital(r.Context(), db.ListRolesInHospitalParams{UserID: currentUser(r).ID, HospitalID: hid})
		if err != nil {
			serverError(w, "load roles", err)
			return
		}
		if len(got) == 0 {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), rolesKey, got)))
	}
}

func currentUser(r *http.Request) auth.User {
	u, _ := r.Context().Value(userKey).(auth.User)
	return u
}

func hasRole(r *http.Request, role string) bool {
	roles, _ := r.Context().Value(rolesKey).([]string)
	return slices.Contains(roles, role)
}

// hospitalID is the {hid} path value; requireHospital has already validated it.
func hospitalID(r *http.Request) uuid.UUID {
	return uuid.MustParse(r.PathValue("hid"))
}

// pathID parses a uuid path value, writing a 404 with notFound when it isn't one.
func pathID(w http.ResponseWriter, r *http.Request, name, notFound string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusNotFound, notFound)
		return uuid.Nil, false
	}
	return id, true
}

func serverError(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
}
