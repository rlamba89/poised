package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// Staff, invites and removing access (plan C1, session S09; STF-01, STF-02, STF-04).
//
//   - Only the trust's whole-trust admins manage its staff. An admin of one hospital can't.
//   - A person can hold several memberships in a trust (STF-02; S09 answers, Rahul 13 Oct): the
//     whole trust and/or some of its hospitals, each with a role. Each is changed or removed on
//     its own (/staff/{uid}/access/{scope}, scope "trust" or a hospital id), or all of them at
//     once (/staff/{uid}). A change takes effect on the person's next request, because every
//     request re-reads the memberships (requireHospital, requireOrg).
//   - Inviting adds the membership at once. A new email also gets an account (in Cognito too),
//     which must sign in for the first time within inviteLife; an existing account only gets the
//     membership (STF-02), including someone already in the trust who gets another hospital.

// inviteLife is how long a new account has for its first sign-in (STF-01). Cognito's temporary
// password lasts as long (S12).
const inviteLife = 7 * 24 * time.Hour

const (
	staffRole       = "Only the trust's admins can manage staff."
	staffNotFound   = "That person isn't on this trust's staff."
	accessNotFound  = "That person doesn't have that access."
	lastAdmin       = "The trust needs at least one admin for the whole trust. Make someone else an admin first."
	inviteExpired   = "Your invitation has expired. Ask your trust's admin to invite you again."
	alreadyHas      = "%s already has that access. Change it in the list instead."
	hospitalNotHere = "Choose the whole trust or one of this trust's hospitals."
)

// staffRow is one person in the staff list, with each of their memberships in the trust.
type staffRow struct {
	UserID uuid.UUID        `json:"userId"`
	Name   string           `json:"name"`
	Email  string           `json:"email"`
	Access []staffAccessRow `json:"access"` // the whole trust first, then by hospital name
	// Status is "active", or while the account waits for its first sign-in, "invited" until
	// InviteExpiresAt and "expired" after it.
	Status          string     `json:"status"`
	InviteExpiresAt *time.Time `json:"inviteExpiresAt"`
}

// staffAccessRow is one membership: the whole trust (hospitalId null) or one hospital, and a role.
type staffAccessRow struct {
	HospitalID   *uuid.UUID `json:"hospitalId"`
	HospitalName *string    `json:"hospitalName"`
	Role         string     `json:"role"`
}

// staffRowsOf folds the membership rows (ordered by person) into one row per person.
func (s *server) staffRowsOf(rows []db.ListStaffRow) []staffRow {
	out := []staffRow{}
	for _, r := range rows {
		if n := len(out); n == 0 || out[n-1].UserID != r.UserID {
			row := staffRow{UserID: r.UserID, Name: r.Name, Email: r.Email, Access: []staffAccessRow{}}
			row.Status, row.InviteExpiresAt = s.inviteStatus(r.InviteExpiresAt)
			out = append(out, row)
		}
		a := staffAccessRow{HospitalName: r.HospitalName, Role: r.Role}
		if r.HospitalID.Valid {
			a.HospitalID = &r.HospitalID.UUID
		}
		out[len(out)-1].Access = append(out[len(out)-1].Access, a)
	}
	return out
}

// inviteStatus is an account's status from its latest open invite's deadline (none: it has
// signed in, or was seeded): "active", or "invited" until the deadline and "expired" after it.
func (s *server) inviteStatus(expires pgtype.Timestamptz) (string, *time.Time) {
	if !expires.Valid {
		return "active", nil
	}
	t := expires.Time
	if !s.clock.Now().Before(t) {
		return "expired", &t
	}
	return "invited", &t
}

// staffAccess is what a person can do in the trust: the whole trust (hospitalId null) or one
// hospital, and a role. Both must be given: a missing hospitalId is never read as the whole trust.
type staffAccess struct {
	HospitalID json.RawMessage `json:"hospitalId"`
	Role       string          `json:"role"`
}

// parse checks the shape; whether the hospital is the trust's is checked against the database.
func (a staffAccess) parse() (hospital uuid.NullUUID, r role.Role, msg string) {
	r, ok := role.Parse(a.Role)
	if !ok {
		return hospital, r, "Choose a role: clinician, super clinician or admin."
	}
	if hospital, ok = parseHospitalID(a.HospitalID); !ok {
		return hospital, r, hospitalNotHere
	}
	return hospital, r, ""
}

// parseHospitalID reads a body's "hospitalId": null is the whole trust, a uuid one hospital.
// A missing one isn't either: it's never read as the whole trust.
func parseHospitalID(raw json.RawMessage) (hospital uuid.NullUUID, ok bool) {
	switch strings.TrimSpace(string(raw)) {
	case "":
		return hospital, false
	case "null":
		return hospital, true
	}
	if err := json.Unmarshal(raw, &hospital.UUID); err != nil {
		return hospital, false
	}
	hospital.Valid = true
	return hospital, true
}

type inviteInput struct {
	Email string `json:"email"`
	Name  string `json:"name"` // optional: the email until the person's name is known
	staffAccess
}

// normalizeEmail trims and lower-cases an email address, or says why it isn't one. Only a bare
// address is accepted (not "Ann <ann@x.org>"), and its domain must have a dot.
func normalizeEmail(s string) (string, string) {
	e := strings.ToLower(strings.TrimSpace(s))
	const bad = "Enter a valid email address, like name@hospital.org."
	if e == "" {
		return "", "Enter the person's email address."
	}
	a, err := mail.ParseAddress(e)
	if err != nil || a.Address != e || len(e) > 254 {
		return "", bad
	}
	at := strings.LastIndex(e, "@")
	if domain := e[at+1:]; !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", bad
	}
	return e, ""
}

func (in *inviteInput) validate() (hospital uuid.NullUUID, r role.Role, msg string) {
	if in.Email, msg = normalizeEmail(in.Email); msg != "" {
		return hospital, r, msg
	}
	if in.Name = strings.TrimSpace(in.Name); len([]rune(in.Name)) > 200 {
		return hospital, r, "The name must be 200 characters or fewer."
	}
	if in.Name == "" {
		in.Name = in.Email
	}
	return in.staffAccess.parse()
}

// staffAdmin answers 403 unless the caller is an admin of the whole trust.
func staffAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !here(r).TrustRole.AtLeast(role.Admin) {
		writeError(w, http.StatusForbidden, staffRole)
		return false
	}
	return true
}

func (s *server) listStaff(w http.ResponseWriter, r *http.Request) {
	if !staffAdmin(w, r) {
		return
	}
	rows, err := s.q.ListStaff(r.Context(), orgID(r))
	if err != nil {
		serverError(w, "list staff", err)
		return
	}
	writeJSON(w, http.StatusOK, s.staffRowsOf(rows))
}

// staffTx runs fn in a transaction holding the trust's staff lock, then commits. fn writes its
// own refusal and returns false to roll back.
func (s *server) staffTx(w http.ResponseWriter, r *http.Request, fn func(q *db.Queries) bool) bool {
	return s.orgTx(w, r, orgID(r), fn)
}

// orgTx is staffTx for a trust named by the caller (the platform admin's routes have no place).
func (s *server) orgTx(w http.ResponseWriter, r *http.Request, org uuid.UUID, fn func(q *db.Queries) bool) bool {
	return s.inTx(w, r, func(q *db.Queries) bool {
		if err := q.LockOrgStaff(r.Context(), org); err != nil {
			serverError(w, "lock staff", err)
			return false
		}
		return fn(q)
	})
}

// inTx runs fn in a transaction, then commits. fn writes its own refusal and returns false to
// roll back.
func (s *server) inTx(w http.ResponseWriter, r *http.Request, fn func(q *db.Queries) bool) bool {
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return false
	}
	defer tx.Rollback(ctx)
	if !fn(db.New(tx)) {
		return false
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return false
	}
	return true
}

// checkHospital answers 400 unless the hospital (if any) is one of this trust's.
func checkHospital(w http.ResponseWriter, r *http.Request, q *db.Queries, hospital uuid.NullUUID) bool {
	if !hospital.Valid {
		return true
	}
	ok, err := q.HospitalInOrg(r.Context(), db.HospitalInOrgParams{OrgID: orgID(r), ID: hospital.UUID})
	if err != nil {
		serverError(w, "check hospital", err)
		return false
	}
	if !ok {
		writeError(w, http.StatusBadRequest, hospitalNotHere)
	}
	return ok
}

// membershipRole is the user's role in one membership of the trust, or "" when they don't hold it.
func membershipRole(ctx context.Context, q *db.Queries, user, org uuid.UUID, hospital uuid.NullUUID) (string, error) {
	got, err := q.MembershipRole(ctx, db.MembershipRoleParams{UserID: user, OrgID: org, HospitalID: hospital})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return got, err
}

// staffMember reads one person's row after a change, in the transaction.
func (s *server) staffMember(ctx context.Context, q *db.Queries, org, user uuid.UUID) (staffRow, error) {
	rows, err := q.GetStaffMember(ctx, db.GetStaffMemberParams{OrgID: org, UserID: user})
	if err != nil {
		return staffRow{}, err
	}
	if len(rows) == 0 {
		return staffRow{}, pgx.ErrNoRows
	}
	list := make([]db.ListStaffRow, len(rows))
	for i, row := range rows {
		list[i] = db.ListStaffRow(row)
	}
	return s.staffRowsOf(list)[0], nil
}

// inviteStaff invites someone by email to the whole trust or one hospital, with a role (STF-01).
func (s *server) inviteStaff(w http.ResponseWriter, r *http.Request) {
	if !staffAdmin(w, r) {
		return
	}
	var in inviteInput
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	hospital, rl, msg := in.validate()
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx, org := r.Context(), orgID(r)

	var out struct {
		staffRow
		ExistingAccount bool `json:"existingAccount"`
	}
	ok := s.staffTx(w, r, func(q *db.Queries) bool {
		if !checkHospital(w, r, q, hospital) {
			return false
		}
		user, existing, ok := s.inviteTo(w, r, q, org, hospital, in.Email, in.Name, rl, alreadyHas)
		if !ok {
			return false
		}
		var err error
		if out.staffRow, err = s.staffMember(ctx, q, org, user); err != nil {
			serverError(w, "read staff", err)
			return false
		}
		out.ExistingAccount = existing
		return true
	})
	if ok {
		writeJSON(w, http.StatusCreated, out)
	}
}

// inviteTo invites email to the trust's whole trust (hospital not Valid) or one of its hospitals,
// with a role, inside the caller's transaction: S09's rules, used by the Staff screen and by the
// platform admin's new trust (S10). It writes its own refusal or error and returns ok false.
//   - A new email gets an account (in Cognito too), which must sign in within inviteLife.
//   - An account that has signed in (or was seeded) only gets the membership (STF-02).
//   - An account still waiting for its first sign-in gets the access and a fresh invitation.
//   - Access a working account already holds is refused (409) with conflict, a format taking the
//     person's name.
//
// Cognito is called last, so its failure rolls everything back.
func (s *server) inviteTo(w http.ResponseWriter, r *http.Request, q *db.Queries, org uuid.UUID, hospital uuid.NullUUID,
	email, name string, rl role.Role, conflict string) (user uuid.UUID, existing, ok bool) {
	ctx, now := r.Context(), s.clock.Now()
	account, created, err := findOrCreateAccount(ctx, q, email, name)
	if err != nil {
		serverError(w, "find or create the account", err)
		return user, false, false
	}
	existing = !created
	waiting := !existing // a new account waits for its first sign-in
	if existing {
		if waiting, err = q.AwaitingFirstSignIn(ctx, account.ID); err != nil {
			serverError(w, "invite state", err)
			return user, false, false
		}
	}
	held, err := membershipRole(ctx, q, account.ID, org, hospital)
	if err != nil {
		serverError(w, "membership", err)
		return user, false, false
	}
	switch {
	case held != "" && !waiting: // a working account: change that access instead
		writeError(w, http.StatusConflict, fmt.Sprintf(conflict, account.Name))
		return user, false, false
	case held != "": // not signed in yet: send the invite again, with this role there
		err = q.UpdateMembership(ctx, db.UpdateMembershipParams{UserID: account.ID, OrgID: org, HospitalID: hospital, NewHospitalID: hospital, Role: rl.String()})
	default: // another membership, on top of any they have (STF-02)
		err = q.AddMembership(ctx, db.AddMembershipParams{UserID: account.ID, OrgID: org, HospitalID: hospital, Role: rl.String()})
	}
	if err != nil {
		serverError(w, "add membership", err)
		return user, false, false
	}
	accepted := pgtype.Timestamptz{Time: now, Valid: !waiting}
	if err := q.CreateStaffInvite(ctx, db.CreateStaffInviteParams{
		OrgID: org, HospitalID: hospital, UserID: account.ID, Role: rl.String(),
		InvitedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
		CreatedAt: now, ExpiresAt: now.Add(inviteLife), AcceptedAt: accepted,
	}); err != nil {
		serverError(w, "create invite", err)
		return user, false, false
	}
	// Last, so a failure rolls everything back. If the commit then fails, the Cognito account
	// is left without ours; inviting again resends to it (S12 handles "already exists").
	switch {
	case !existing:
		err = s.cognito.CreateUser(ctx, email)
	case waiting:
		err = s.cognito.ResendInvite(ctx, account.Email)
	}
	if err != nil {
		serverError(w, "cognito invite", err)
		return user, false, false
	}
	return account.ID, existing, true
}

// findOrCreateAccount returns the account with this email (any case), creating it when there's
// none; created says which.
func findOrCreateAccount(ctx context.Context, q *db.Queries, email, name string) (db.GetUserByEmailRow, bool, error) {
	user, err := q.GetUserByEmail(ctx, email)
	if !errors.Is(err, pgx.ErrNoRows) {
		return user, false, err
	}
	ids, err := q.CreateUserIfNew(ctx, db.CreateUserIfNewParams{Name: name, Email: email})
	if err != nil {
		return user, false, err
	}
	if len(ids) == 0 { // another invite created it just now
		user, err = q.GetUserByEmail(ctx, email)
		return user, false, err
	}
	return db.GetUserByEmailRow{ID: ids[0], Name: name, Email: email}, true, nil
}

// staffTarget is the {uid} of a staff route, answering 404 itself when it isn't a uuid.
func staffTarget(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathID(w, r, "uid", staffNotFound)
}

// isTrustAdmin reports whether a membership is a whole-trust admin's.
func isTrustAdmin(hospital uuid.NullUUID, r string) bool {
	return !hospital.Valid && r == role.Admin.String()
}

// keepsAnAdmin answers 409 when a change would leave the trust without a whole-trust admin: the
// user's membership being changed or removed is one (losing), and nobody else is one.
func keepsAnAdmin(w http.ResponseWriter, r *http.Request, q *db.Queries, user uuid.UUID, losing bool) bool {
	if !losing {
		return true
	}
	others, err := q.CountOtherTrustAdmins(r.Context(), db.CountOtherTrustAdminsParams{OrgID: orgID(r), UserID: user})
	if err != nil {
		serverError(w, "count admins", err)
		return false
	}
	if others == 0 {
		writeError(w, http.StatusConflict, lastAdmin)
		return false
	}
	return true
}

// currentAccess reads the user's memberships in the trust, answering 404 when there are none.
func currentAccess(w http.ResponseWriter, r *http.Request, q *db.Queries, user uuid.UUID) ([]db.UserOrgMembershipsRow, bool) {
	rows, err := q.UserOrgMemberships(r.Context(), db.UserOrgMembershipsParams{UserID: user, OrgID: orgID(r)})
	if err != nil {
		serverError(w, "memberships", err)
		return nil, false
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, staffNotFound)
		return nil, false
	}
	return rows, true
}

// staffScope is the {scope} of a membership route: "trust" (the whole trust) or a hospital id.
// Anything else answers 404 itself.
func staffScope(w http.ResponseWriter, r *http.Request) (uuid.NullUUID, bool) {
	raw := r.PathValue("scope")
	if raw == "trust" {
		return uuid.NullUUID{}, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, accessNotFound)
		return uuid.NullUUID{}, false
	}
	return uuid.NullUUID{UUID: id, Valid: true}, true
}

// heldRole reads the role of the membership a route names, answering 404 when the person
// doesn't hold it here.
func heldRole(w http.ResponseWriter, r *http.Request, q *db.Queries, user uuid.UUID, scope uuid.NullUUID) (string, bool) {
	held, err := membershipRole(r.Context(), q, user, orgID(r), scope)
	if err != nil {
		serverError(w, "membership", err)
		return "", false
	}
	if held == "" {
		writeError(w, http.StatusNotFound, accessNotFound)
		return "", false
	}
	return held, true
}

// updateStaffAccess changes one of someone's memberships: its scope (the whole trust or another
// hospital), its role, or both. Their other memberships stay.
func (s *server) updateStaffAccess(w http.ResponseWriter, r *http.Request) {
	if !staffAdmin(w, r) {
		return
	}
	user, ok := staffTarget(w, r)
	if !ok {
		return
	}
	scope, ok := staffScope(w, r)
	if !ok {
		return
	}
	var in staffAccess
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	hospital, rl, msg := in.parse()
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	var out staffRow
	ok = s.staffTx(w, r, func(q *db.Queries) bool {
		held, ok := heldRole(w, r, q, user, scope)
		if !ok || !checkHospital(w, r, q, hospital) {
			return false
		}
		if hospital != scope { // moving onto access they may already have
			other, err := membershipRole(ctx, q, user, orgID(r), hospital)
			if err != nil {
				serverError(w, "membership", err)
				return false
			}
			if other != "" {
				writeError(w, http.StatusConflict, "They already have that access. Change or remove that one instead.")
				return false
			}
		}
		if !keepsAnAdmin(w, r, q, user, isTrustAdmin(scope, held) && !isTrustAdmin(hospital, rl.String())) {
			return false
		}
		if err := q.UpdateMembership(ctx, db.UpdateMembershipParams{
			UserID: user, OrgID: orgID(r), HospitalID: scope, NewHospitalID: hospital, Role: rl.String(),
		}); err != nil {
			serverError(w, "change membership", err)
			return false
		}
		var err error
		if out, err = s.staffMember(ctx, q, orgID(r), user); err != nil {
			serverError(w, "read staff", err)
			return false
		}
		return true
	})
	if ok {
		writeJSON(w, http.StatusOK, out)
	}
}

// removeStaffAccess removes one of someone's memberships. Removing their last one ends their
// access to the trust (STF-04).
func (s *server) removeStaffAccess(w http.ResponseWriter, r *http.Request) {
	if !staffAdmin(w, r) {
		return
	}
	user, ok := staffTarget(w, r)
	if !ok {
		return
	}
	scope, ok := staffScope(w, r)
	if !ok {
		return
	}
	ok = s.staffTx(w, r, func(q *db.Queries) bool {
		held, ok := heldRole(w, r, q, user, scope)
		if !ok || !keepsAnAdmin(w, r, q, user, isTrustAdmin(scope, held)) {
			return false
		}
		if err := q.DeleteMembership(r.Context(), db.DeleteMembershipParams{UserID: user, OrgID: orgID(r), HospitalID: scope}); err != nil {
			serverError(w, "remove membership", err)
			return false
		}
		return true
	})
	if ok {
		w.WriteHeader(http.StatusNoContent)
	}
}

// removeStaff removes all of someone's access to the trust (STF-04). Their account stays: past
// actions keep their author, and they may belong to other trusts.
func (s *server) removeStaff(w http.ResponseWriter, r *http.Request) {
	if !staffAdmin(w, r) {
		return
	}
	user, ok := staffTarget(w, r)
	if !ok {
		return
	}
	ok = s.staffTx(w, r, func(q *db.Queries) bool {
		current, ok := currentAccess(w, r, q, user)
		if !ok {
			return false
		}
		losing := false
		for _, m := range current {
			losing = losing || isTrustAdmin(m.HospitalID, m.Role)
		}
		if !keepsAnAdmin(w, r, q, user, losing) {
			return false
		}
		if err := q.DeleteUserOrgMemberships(r.Context(), db.DeleteUserOrgMembershipsParams{UserID: user, OrgID: orgID(r)}); err != nil {
			serverError(w, "remove membership", err)
			return false
		}
		return true
	})
	if ok {
		w.WriteHeader(http.StatusNoContent)
	}
}

// errInviteExpired is checkInvites' refusal: the account never signed in, and every invite it
// had has expired.
var errInviteExpired = errors.New("invite expired")

// checkInvites runs at sign-in. An account waiting for its first sign-in may sign in while one
// of its invites is open, which accepts them all; once all have expired it can't. Accounts that
// have signed in before, or were never invited (seeded), have nothing waiting. S12's Cognito
// sign-in calls it too.
func (s *server) checkInvites(ctx context.Context, user uuid.UUID) error {
	now := s.clock.Now()
	st, err := s.q.InviteState(ctx, db.InviteStateParams{UserID: user, Now: now})
	if err != nil {
		return err
	}
	switch {
	case st.Waiting == 0:
		return nil
	case st.Open == 0:
		return errInviteExpired
	}
	return s.q.AcceptInvites(ctx, db.AcceptInvitesParams{UserID: user, Now: pgtype.Timestamptz{Time: now, Valid: true}})
}
