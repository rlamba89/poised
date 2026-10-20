package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// suspended is the sign-in refusal for someone whose trusts and hospitals are all suspended.
// Both sign-ins (the dev login and Cognito's) check it, in startStaffSession.
const suspended = "Your trust or hospital is suspended in Preopsis at the moment, so you can't sign in. Please contact your trust."

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

// devLogin signs in as any user (seeded or invited) and sets the token cookie. It's local only:
// on AWS staff sign in with Cognito (S12; config refuses DEV_LOGIN with CONFIG_STRICT).
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
	s.startStaffSession(w, r, u.ID, u.Name)
}

// startStaffSession signs the user in, whichever way they proved who they are (the dev login, or
// Cognito's managed login): it refuses someone whose every trust or hospital is suspended, unless
// they're a platform admin, who must still reach the console, e.g. to re-activate it (ORG-03; S10
// verification 1), and an invited account whose invite expired before its first sign-in (STF-01);
// then it sets our own session cookie.
func (s *server) startStaffSession(w http.ResponseWriter, r *http.Request, id uuid.UUID, name string) {
	if ok, err := s.signInAllowed(r.Context(), id); err != nil {
		serverError(w, "sign-in state", err)
		return
	} else if !ok {
		writeError(w, http.StatusForbidden, suspended)
		return
	}
	if err := s.checkInvites(r.Context(), id); errors.Is(err, errInviteExpired) {
		writeError(w, http.StatusForbidden, inviteExpired)
		return
	} else if err != nil {
		serverError(w, "check invites", err)
		return
	}
	token, err := auth.Issue(s.secret, auth.User{ID: id, Name: name}, time.Now())
	if err != nil {
		serverError(w, "issue token", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookies(), MaxAge: int(auth.TTL.Seconds()),
	})
	w.WriteHeader(http.StatusNoContent)
}

const (
	signInAgain = "That sign-in didn't work or has expired. Please sign in again."
	noAccount   = "There's no Preopsis account for %s. Ask your trust's admin to invite you."
)

// pkceVerifier is RFC 7636's code_verifier: 43 to 128 of these characters.
var pkceVerifier = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

// cognitoCallback is POST /api/auth/callback {code, verifier} (S12): the app's /auth/callback
// page posts the code the managed login sent the browser back with, and the PKCE verifier it kept.
// Go exchanges the code with Cognito, checks the ID token, finds the staff member by Cognito's
// sub (or, the first time, by the email Cognito verified, then stores the sub) and sets our own
// session cookie, as the dev login does. Preopsis never creates an account here: accounts come
// only from invites (no self sign-up).
func (s *server) cognitoCallback(w http.ResponseWriter, r *http.Request) {
	var body struct{ Code, Verifier string }
	if !readJSON(w, r, 8<<10, &body) {
		return
	}
	if body.Code == "" || len(body.Code) > 2048 || !pkceVerifier.MatchString(body.Verifier) {
		writeError(w, http.StatusBadRequest, "The sign-in could not be read. Please sign in again.")
		return
	}
	ctx := r.Context()
	raw, err := s.cognito.Exchange(ctx, body.Code, body.Verifier)
	if errors.Is(err, cognito.ErrCode) {
		writeError(w, http.StatusUnauthorized, signInAgain)
		return
	}
	if err != nil {
		serverError(w, "cognito exchange", err)
		return
	}
	id, err := s.idToken.Verify(ctx, raw, s.clock.Now())
	if err != nil {
		log.Printf("cognito callback: %v", err) // never the token itself
		writeError(w, http.StatusUnauthorized, signInAgain)
		return
	}
	user, err := s.q.GetUserByCognitoSub(ctx, &id.Sub)
	if errors.Is(err, pgx.ErrNoRows) {
		var byEmail db.GetUserByEmailRow
		byEmail, err = s.q.GetUserByEmail(ctx, id.Email)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusForbidden, fmt.Sprintf(noAccount, id.Email))
			return
		}
		if err == nil {
			user = db.GetUserByCognitoSubRow(byEmail)
			err = s.q.SetCognitoSub(ctx, db.SetCognitoSubParams{ID: user.ID, Sub: &id.Sub})
		}
	}
	if err != nil {
		serverError(w, "find the user", err)
		return
	}
	s.startStaffSession(w, r, user.ID, user.Name)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.secureCookies(), MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

// secureCookies: the session cookies are Secure when the app is served over https (AWS), so
// they never travel over plain http. Locally (http://localhost) they can't be.
func (s *server) secureCookies() bool { return strings.HasPrefix(s.cfg.Country.AppURL, "https://") }

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
