package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// suspended is the sign-in refusal for someone whose trusts and hospitals are all suspended.
// S12's Cognito sign-in must call signInAllowed too.
const suspended = "Your trust or hospital is suspended in Poised at the moment, so you can't sign in. Please contact your trust."

// signInAllowed reports whether the user may sign in as far as suspension goes (ORG-03): a
// platform admin always may; anyone else needs at least one usable membership, or none at all
// (they then see "ask an admin to invite you"). Suspended trusts and hospitals stay out of
// /api/me and every trust and hospital route either way.
func (s *server) signInAllowed(ctx context.Context, user uuid.UUID) (bool, error) {
	admin, err := s.q.IsPlatformAdmin(ctx, user)
	if err != nil || admin {
		return admin, err
	}
	st, err := s.q.SignInState(ctx, user)
	if err != nil {
		return false, err
	}
	return st.Total == 0 || st.Active > 0, nil
}

// devUsers lists seeded users for the stub login page.
func (s *server) devUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.q.ListUsers(r.Context())
	if err != nil {
		serverError(w, "list users", err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// devLogin signs in as any user (seeded or invited) and sets the token cookie. Cognito replaces
// it outside local development (S12).
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
	// Someone whose every trust or hospital is suspended can't sign in (ORG-03), unless they're a
	// platform admin: they must still reach the console, e.g. to re-activate it (S10 verification 1).
	if ok, err := s.signInAllowed(r.Context(), u.ID); err != nil {
		serverError(w, "sign-in state", err)
		return
	} else if !ok {
		writeError(w, http.StatusForbidden, suspended)
		return
	}
	// An invited account's first sign-in must come before its invite expires (STF-01).
	if err := s.checkInvites(r.Context(), u.ID); errors.Is(err, errInviteExpired) {
		writeError(w, http.StatusForbidden, inviteExpired)
		return
	} else if err != nil {
		serverError(w, "check invites", err)
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

// meHospital is a hospital the user can work in, with their role there: the higher of their
// trust-wide role and their role at that hospital.
type meHospital struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	TimeZone string    `json:"timeZone"` // the hospital's, or the deployment's default (A-20)
	Role     string    `json:"role"`
}

// meTrust is a trust the user belongs to. Role is their trust-wide role, or null when they
// belong only to some of its hospitals ("All hospitals" is for trust-wide staff, STF-03).
type meTrust struct {
	ID        uuid.UUID    `json:"id"`
	Name      string       `json:"name"`
	Code      string       `json:"code"`
	Role      *string      `json:"role"`
	Hospitals []meHospital `json:"hospitals"`
}

// me returns the signed-in user, whether they're a platform admin, and their trusts with the
// hospitals they can work in, by name. The web app's hospital picker and header use it.
func (s *server) me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u, err := s.q.GetUser(ctx, currentUser(r).ID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	if err != nil {
		serverError(w, "get user", err)
		return
	}
	admin, err := s.q.IsPlatformAdmin(ctx, u.ID)
	if err != nil {
		serverError(w, "platform admin", err)
		return
	}
	rows, err := s.q.ListMyHospitals(ctx, u.ID)
	if err != nil {
		serverError(w, "list hospitals", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "platformAdmin": admin, "trusts": s.myTrusts(rows)})
}

// myTrusts folds the membership rows (ordered by trust, then hospital) into trusts and their
// hospitals, keeping the highest role where two memberships reach the same hospital.
func (s *server) myTrusts(rows []db.ListMyHospitalsRow) []meTrust {
	trusts := []meTrust{}
	for _, m := range rows {
		if n := len(trusts); n == 0 || trusts[n-1].ID != m.OrgID {
			trusts = append(trusts, meTrust{ID: m.OrgID, Name: m.OrgName, Code: m.OrgCode, Hospitals: []meHospital{}})
		}
		t := &trusts[len(trusts)-1]
		if m.TrustWide {
			higher := role.Highest(m.Role)
			if t.Role != nil {
				higher = max(higher, role.Highest(*t.Role))
			}
			name := higher.String()
			t.Role = &name
		}
		if !m.HospitalID.Valid { // a trust with no hospitals yet
			continue
		}
		if n := len(t.Hospitals); n == 0 || t.Hospitals[n-1].ID != m.HospitalID.UUID {
			tz := s.cfg.Country.TimeZone
			if m.Timezone != nil {
				tz = *m.Timezone
			}
			t.Hospitals = append(t.Hospitals, meHospital{ID: m.HospitalID.UUID, Name: *m.HospitalName, TimeZone: tz})
		}
		h := &t.Hospitals[len(t.Hospitals)-1]
		h.Role = role.Highest(h.Role, m.Role).String()
	}
	return trusts
}
