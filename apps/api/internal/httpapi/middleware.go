package httpapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

type ctxKey int

const (
	userKey ctxKey = iota
	placeKey
	patientKey
)

// Memberships reads the caller's memberships, on every request so that removing one takes
// effect at once (STF-04). *db.Queries satisfies it.
type Memberships interface {
	RolesAtHospital(ctx context.Context, arg db.RolesAtHospitalParams) ([]db.RolesAtHospitalRow, error)
	ListOrgMemberships(ctx context.Context, arg db.ListOrgMembershipsParams) ([]db.ListOrgMembershipsRow, error)
}

// place is where a request works, and the caller's role there.
type place struct {
	Org      uuid.UUID
	Hospital uuid.UUID // uuid.Nil on a trust route (/api/o/{oid}/…)
	// Role is the caller's role at Hospital: the higher of their trust-wide role and their role
	// at that hospital. On a trust route it's their trust-wide role, or None if they have none.
	Role role.Role
	// TrustRole is the caller's whole-trust role, or None if they belong to some hospitals only.
	// Changing the trust's shared content needs it (S08 answers, 5).
	TrustRole role.Role
	// Hospitals, on a trust route, are the hospitals the caller is a member of one by one.
	Hospitals []uuid.UUID
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

// requireHospital puts the caller's place in the context for a route under
// /api/o/{oid}/h/{hid}: they must be a member of the whole trust or of that hospital, and the
// hospital must be in the trust. Anyone else gets 404, so trusts and hospitals can't be probed
// (NFR-04). It must run inside requireUser.
func requireHospital(m Memberships, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oid, errO := uuid.Parse(r.PathValue("oid"))
		hid, errH := uuid.Parse(r.PathValue("hid"))
		if errO != nil || errH != nil {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return
		}
		rows, err := m.RolesAtHospital(r.Context(), db.RolesAtHospitalParams{UserID: currentUser(r).ID, OrgID: oid, HospitalID: hid})
		if err != nil {
			serverError(w, "load roles", err)
			return
		}
		p := place{Org: oid, Hospital: hid}
		for _, row := range rows {
			got := role.Highest(row.Role)
			p.Role = max(p.Role, got)
			if !row.HospitalID.Valid {
				p.TrustRole = max(p.TrustRole, got)
			}
		}
		if p.Role == role.None {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), placeKey, p)))
	}
}

// requireOrg is requireHospital for a trust route, /api/o/{oid}/…: the caller must be a member
// of the trust or of one of its hospitals. Non-members get 404.
func requireOrg(m Memberships, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oid, err := uuid.Parse(r.PathValue("oid"))
		if err != nil {
			writeError(w, http.StatusNotFound, "Trust not found.")
			return
		}
		rows, err := m.ListOrgMemberships(r.Context(), db.ListOrgMembershipsParams{UserID: currentUser(r).ID, OrgID: oid})
		if err != nil {
			serverError(w, "load memberships", err)
			return
		}
		if len(rows) == 0 {
			writeError(w, http.StatusNotFound, "Trust not found.")
			return
		}
		p := place{Org: oid}
		for _, row := range rows {
			if !row.HospitalID.Valid {
				p.Role = max(p.Role, role.Highest(row.Role))
				p.TrustRole = p.Role
			} else {
				p.Hospitals = append(p.Hospitals, row.HospitalID.UUID)
			}
		}
		next(w, r.WithContext(context.WithValue(r.Context(), placeKey, p)))
	}
}

func currentUser(r *http.Request) auth.User {
	u, _ := r.Context().Value(userKey).(auth.User)
	return u
}

// withPatient puts the patient's record id (their session's) in the context.
func withPatient(ctx context.Context, patient uuid.UUID) context.Context {
	return context.WithValue(ctx, patientKey, patient)
}

// patientOf is the signed-in patient's record id (requirePatient), or uuid.Nil on other routes.
func patientOf(r *http.Request) uuid.UUID {
	p, _ := r.Context().Value(patientKey).(uuid.UUID)
	return p
}

// here is the request's place; requireHospital or requireOrg has put it there.
func here(r *http.Request) place {
	p, _ := r.Context().Value(placeKey).(place)
	return p
}

// can reports whether the caller's role here is at least need (A-5).
func can(r *http.Request, need role.Role) bool {
	return here(r).Role.AtLeast(need)
}

// canChange reports whether the caller's role is at least need for changing content owned by
// owner: one hospital's own content (owner set) needs that role here; the trust's shared content
// (owner NULL) needs it as a whole-trust role (S08 answers, 5: Rahul, 12 Oct).
func canChange(r *http.Request, owner uuid.NullUUID, need role.Role) bool {
	if owner.Valid {
		return can(r, need)
	}
	return here(r).TrustRole.AtLeast(need)
}

// orgID and hospitalID are the request's trust and hospital, checked by the middleware.
func orgID(r *http.Request) uuid.UUID      { return here(r).Org }
func hospitalID(r *http.Request) uuid.UUID { return here(r).Hospital }

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
