package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// Staff sign-in with Cognito and staff invites (plan C1, auth-first Step 4). Cognito says who
// someone is; what they may do comes from memberships.

// StaffSignIn swaps the code from Cognito's managed login for a verified identity.
// *cognito.Client satisfies it; tests pass a fake.
type StaffSignIn interface {
	Exchange(ctx context.Context, code, verifier string) (cognito.Identity, error)
}

// StaffDirectory invites people into the user pool and returns their Cognito subject.
// *cognito.Directory satisfies it; tests pass a fake.
type StaffDirectory interface {
	Invite(ctx context.Context, email, name string) (string, error)
}

// CognitoLogin is what the web app needs to send people to the managed login and back.
type CognitoLogin struct {
	AuthorizeURL      string `json:"authorizeUrl"`
	LogoutURL         string `json:"logoutUrl"`
	ClientID          string `json:"clientId"`
	RedirectURI       string `json:"redirectUri"`
	LogoutRedirectURI string `json:"logoutRedirectUri"`
}

// authConfig tells the sign-in page which ways to sign in exist here. It holds nothing secret.
func (s *server) authConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"cognito": s.login, "devLogin": s.devLoginOn})
}

// cognitoCallback finishes a Cognito sign-in: it checks the code with Cognito, finds the
// invited user, and starts a staff session.
func (s *server) cognitoCallback(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code     string `json:"code"`
		Verifier string `json:"verifier"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	if in.Code == "" || in.Verifier == "" {
		writeError(w, http.StatusBadRequest, "The sign-in link is incomplete. Please sign in again.")
		return
	}
	ctx := r.Context()
	id, err := s.signIn.Exchange(ctx, in.Code, in.Verifier)
	if err != nil {
		log.Printf("cognito sign-in: %v", err)
		writeError(w, http.StatusUnauthorized, "Sign-in didn't work. Please sign in again.")
		return
	}
	user, ok := s.findSignedInUser(w, r, id)
	if !ok {
		return
	}
	n, err := s.q.CountActiveMemberships(ctx, user)
	if err != nil {
		serverError(w, "count memberships", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusForbidden, "Your account has no access to any hospital. Please ask your trust's admin.")
		return
	}
	sid, hash, err := auth.NewSessionID()
	if err != nil {
		serverError(w, "new session", err)
		return
	}
	if err := s.q.CreateSession(ctx, db.CreateSessionParams{IDHash: hash, UserID: user, MaxAgeSeconds: s.sessionMaxAge.Seconds()}); err != nil {
		serverError(w, "create session", err)
		return
	}
	http.SetCookie(w, s.cookie(s.cookieName(), sid, int(s.sessionMaxAge.Seconds())))
	w.WriteHeader(http.StatusNoContent)
}

// findSignedInUser maps a Cognito identity to our user: by subject, or, the first time, by
// email (saving the subject). Nobody else may sign in: accounts come from invites.
func (s *server) findSignedInUser(w http.ResponseWriter, r *http.Request, id cognito.Identity) (uuid.UUID, bool) {
	ctx := r.Context()
	u, err := s.q.GetUserByCognitoSub(ctx, &id.Subject)
	if err == nil {
		return u.ID, true
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		serverError(w, "user by subject", err)
		return uuid.Nil, false
	}
	byEmail, err := s.q.GetUserByEmail(ctx, id.Email)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && byEmail.CognitoSub != nil) {
		// Unknown, or the email belongs to a different Cognito account.
		writeError(w, http.StatusForbidden, "You don't have a Poised account yet. Please ask your trust's admin to invite you.")
		return uuid.Nil, false
	}
	if err != nil {
		serverError(w, "user by email", err)
		return uuid.Nil, false
	}
	if _, err := s.q.SetCognitoSub(ctx, db.SetCognitoSubParams{CognitoSub: &id.Subject, ID: byEmail.ID}); err != nil {
		serverError(w, "save subject", err)
		return uuid.Nil, false
	}
	return byEmail.ID, true
}

// requireTrustAdmin lets through only an admin of the whole trust {oid} (staff and roles are
// trust business, STF-01). Someone outside the trust gets 404; a member who isn't a trust-wide
// admin gets 403. It must run inside requireUser.
func requireTrustAdmin(q db.Querier, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oid, err := uuid.Parse(r.PathValue("oid"))
		if err != nil {
			writeError(w, http.StatusNotFound, "Trust not found.")
			return
		}
		trust, err := q.ListTrustRoles(r.Context(), db.ListTrustRolesParams{UserID: currentUser(r).ID, OrgID: oid})
		if err != nil {
			serverError(w, "load trust roles", err)
			return
		}
		if role.Highest(toRoles(trust)...).AtLeast(role.Admin) {
			next(w, r)
			return
		}
		// Not a trust-wide admin: 403 for members of the trust, 404 for everyone else.
		n, err := q.CountTrustMemberships(r.Context(), db.CountTrustMembershipsParams{UserID: currentUser(r).ID, OrgID: oid})
		if err != nil {
			serverError(w, "count memberships", err)
			return
		}
		if n == 0 {
			writeError(w, http.StatusNotFound, "Trust not found.")
			return
		}
		writeError(w, http.StatusForbidden, "Only the trust's admins can manage staff.")
	}
}

// listStaff lists everyone with access to the trust, one row per membership.
func (s *server) listStaff(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListTrustStaff(r.Context(), orgID(r))
	if err != nil {
		serverError(w, "list staff", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// inviteStaff gives someone a role in the trust or one of its hospitals. A new person is
// created in Cognito, which emails them; someone who already has an account only gains the
// membership (STF-02).
func (s *server) inviteStaff(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email      string        `json:"email"`
		Name       string        `json:"name"`
		Role       role.Role     `json:"role"`
		HospitalID uuid.NullUUID `json:"hospitalId"` // absent: the whole trust
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	if s.directory == nil {
		writeError(w, http.StatusServiceUnavailable, "Inviting staff needs Cognito, which isn't set up here.")
		return
	}
	in.Email, in.Name = strings.ToLower(strings.TrimSpace(in.Email)), strings.TrimSpace(in.Name)
	if a, err := mail.ParseAddress(in.Email); err != nil || a.Address != in.Email {
		writeError(w, http.StatusBadRequest, "Enter a valid email address.")
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "Enter the person's name.")
		return
	}
	if role.Highest(in.Role) == "" {
		writeError(w, http.StatusBadRequest, "Choose a role: clinician, super clinician or admin.")
		return
	}
	ctx, oid := r.Context(), orgID(r)
	if in.HospitalID.Valid {
		ok, err := s.q.HospitalInOrg(ctx, db.HospitalInOrgParams{ID: in.HospitalID.UUID, OrgID: oid})
		if err != nil {
			serverError(w, "check hospital", err)
			return
		}
		if !ok {
			writeError(w, http.StatusBadRequest, "That hospital isn't in this trust.")
			return
		}
	}
	sub, err := s.directory.Invite(ctx, in.Email, in.Name)
	if err != nil {
		serverError(w, "invite in cognito", err)
		return
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	var userID uuid.UUID
	existing, err := q.GetUserByEmail(ctx, in.Email)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		userID, err = q.CreateInvitedUser(ctx, db.CreateInvitedUserParams{Name: in.Name, Email: in.Email, CognitoSub: &sub})
		if err != nil {
			serverError(w, "create user", err)
			return
		}
	case err != nil:
		serverError(w, "user by email", err)
		return
	default:
		userID = existing.ID
		if _, err := q.SetCognitoSub(ctx, db.SetCognitoSubParams{CognitoSub: &sub, ID: userID}); err != nil {
			serverError(w, "save subject", err)
			return
		}
	}
	n, err := q.CreateMembership(ctx, db.CreateMembershipParams{UserID: userID, OrgID: oid, HospitalID: in.HospitalID, Role: string(in.Role)})
	if err != nil {
		serverError(w, "create membership", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "This person already has access there. Change their role instead.")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"userId": userID})
}
