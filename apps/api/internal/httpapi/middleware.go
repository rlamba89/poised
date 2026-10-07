package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

type ctxKey int

const (
	userKey ctxKey = iota
	roleKey
)

// RoleLister loads the roles that apply to a user in one hospital of one trust. *db.Queries satisfies it.
type RoleLister interface {
	ListRolesAt(ctx context.Context, arg db.ListRolesAtParams) ([]string, error)
}

// SessionStore finds a live session and records that it was used. *db.Queries satisfies it.
type SessionStore interface {
	GetSession(ctx context.Context, arg db.GetSessionParams) (db.GetSessionRow, error)
	TouchSession(ctx context.Context, idHash []byte) error
}

// touchEvery limits last_seen_at writes to one a minute per session.
const touchEvery = time.Minute

// requireUser rejects requests without a live session and puts the user in the context. A session
// is live before its maximum age and while each request comes within idle of the one before.
func requireUser(sessions SessionStore, cookieName string, idle time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Please sign in.")
			return
		}
		hash := auth.HashToken(c.Value)
		row, err := sessions.GetSession(r.Context(), db.GetSessionParams{IDHash: hash, IdleSeconds: idle.Seconds()})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "Your session has expired. Please sign in again.")
			return
		}
		if err != nil {
			serverError(w, "load session", err)
			return
		}
		if time.Since(row.LastSeenAt) > touchEvery {
			if err := sessions.TouchSession(r.Context(), hash); err != nil {
				log.Printf("touch session: %v", err) // the request can still go ahead
			}
		}
		u := auth.User{ID: row.UserID, Name: row.Name}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	}
}

// requireMembership checks the caller may open hospital {hid} of trust {oid} (NFR-04): a
// membership for that hospital or the whole trust, the hospital in that trust, and the trust
// active. It puts the highest role that applies in the context. Anyone else gets 404, so
// trusts and hospitals can't be probed. It must run inside requireUser.
func requireMembership(roles RoleLister, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oid, err1 := uuid.Parse(r.PathValue("oid"))
		hid, err2 := uuid.Parse(r.PathValue("hid"))
		if err1 != nil || err2 != nil {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return
		}
		got, err := roles.ListRolesAt(r.Context(), db.ListRolesAtParams{UserID: currentUser(r).ID, OrgID: oid, HospitalID: hid})
		if err != nil {
			serverError(w, "load roles", err)
			return
		}
		best := role.Highest(toRoles(got)...)
		if best == "" {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), roleKey, best)))
	}
}

func toRoles(names []string) []role.Role {
	rs := make([]role.Role, len(names))
	for i, n := range names {
		rs[i] = role.Role(n)
	}
	return rs
}

func currentUser(r *http.Request) auth.User {
	u, _ := r.Context().Value(userKey).(auth.User)
	return u
}

// atLeast reports whether the caller's role in this hospital is want or above (requireMembership set it).
func atLeast(r *http.Request, want role.Role) bool {
	have, _ := r.Context().Value(roleKey).(role.Role)
	return have.AtLeast(want)
}

// orgID is the {oid} path value; requireMembership has already validated it.
func orgID(r *http.Request) uuid.UUID {
	return uuid.MustParse(r.PathValue("oid"))
}

// hospitalID is the {hid} path value; requireMembership has already validated it.
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
