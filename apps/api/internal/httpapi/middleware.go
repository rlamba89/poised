package httpapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
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

// recoverPanics answers a handler's panic with the plain-language 500 and logs it with its
// stack. Without it, Go's server drops the connection and the browser gets no response.
//   - The log names the route pattern, never the path: a patient's path holds their link token.
//   - A response that has already started can't become a 500, so it is aborted instead.
//   - http.ErrAbortHandler is passed on: it is the standard way to abort a response on purpose.
func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &startedWriter{ResponseWriter: w}
		defer func() {
			err := recover()
			if err == nil {
				return
			}
			if err == http.ErrAbortHandler {
				panic(err)
			}
			if sw.started {
				log.Printf("panic in %s, after the response started: %v\n%s", r.Pattern, err, debug.Stack())
				panic(http.ErrAbortHandler)
			}
			serverError(w, "panic in "+r.Pattern, fmt.Errorf("%v\n%s", err, debug.Stack()))
		}()
		next.ServeHTTP(sw, r)
	})
}

// startedWriter notes whether the handler has started its response.
type startedWriter struct {
	http.ResponseWriter
	started bool
}

func (s *startedWriter) WriteHeader(code int) {
	s.started = true
	s.ResponseWriter.WriteHeader(code)
}

func (s *startedWriter) Write(b []byte) (int, error) {
	s.started = true
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the real writer.
func (s *startedWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

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
