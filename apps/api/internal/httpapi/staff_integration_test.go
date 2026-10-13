//go:build integration

package httpapi_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
)

// Staff, invites and removing access (plan C1, session S09; STF-01, STF-02, STF-04). Only the
// trust's whole-trust admins manage staff. A person can hold several memberships in one trust (the
// whole trust and/or some of its hospitals, each with a role: S09 answers, Rahul 13 Oct); each is
// changed or removed on its own at /staff/{uid}/access/{scope}, where scope is "trust" or a
// hospital id. Trust A has hospitals A1 and A2; trust B has B1.

type staffWorld struct {
	c          *apitest.Client // not signed in
	fx         *apitest.Fixtures
	a1, a2, b1 apitest.Hospital

	ada, sup, cara, hosAdmin, bea apitest.User
	adaC, supC, caraC, hosAdminC  *apitest.Client
	beaC                          *apitest.Client
}

func newStaffWorld(t *testing.T) *staffWorld {
	t.Helper()
	c, fx := apitest.New(t)
	w := &staffWorld{c: c, fx: fx, a1: fx.Hospital(t), b1: fx.Hospital(t)}
	w.a2 = fx.HospitalIn(t, w.a1.Org)
	w.ada = fx.TrustUser(t, w.a1.Org, "admin")           // trust A's admin
	w.sup = fx.TrustUser(t, w.a1.Org, "super_clinician") // the whole trust, but not an admin
	w.cara = fx.User(t, w.a1, "clinician")               // A1 only
	w.hosAdmin = fx.User(t, w.a1, "admin")               // an admin of A1 only, not the whole trust
	w.bea = fx.TrustUser(t, w.b1.Org, "admin")           // trust B's admin
	w.adaC, w.supC, w.caraC = c.LoginAs(t, w.ada), c.LoginAs(t, w.sup), c.LoginAs(t, w.cara)
	w.hosAdminC, w.beaC = c.LoginAs(t, w.hosAdmin), c.LoginAs(t, w.bea)
	return w
}

func staffPath(org uuid.UUID, p string) string { return "/api/o/" + org.String() + "/staff" + p }

// staffRow is one person in the staff list, with each of their memberships in the trust.
type staffRow struct {
	UserID          uuid.UUID
	Name, Email     string
	Access          []staffAccessRow
	Status          string
	InviteExpiresAt *time.Time
	ExistingAccount bool
}

type staffAccessRow struct {
	HospitalID   *uuid.UUID // nil: the whole trust
	HospitalName *string
	Role         string
}

// accessIs reports whether the row's memberships are exactly these (hospital, role) pairs, in any
// order, with the whole trust (uuid.Nil) first: a hospital of uuid.Nil is the whole trust.
func accessIs(r *staffRow, want ...any) bool {
	if r == nil || len(r.Access) != len(want)/2 {
		return false
	}
	for i, a := range r.Access {
		if i > 0 && a.HospitalID == nil {
			return false // the whole trust comes first
		}
		if a.HospitalID != nil && (a.HospitalName == nil || *a.HospitalName == "") {
			return false
		}
		found := false
		for j := 0; j < len(want); j += 2 {
			h, role := want[j].(uuid.UUID), want[j+1].(string)
			found = found || ((h == uuid.Nil) == (a.HospitalID == nil) && (a.HospitalID == nil || *a.HospitalID == h) && a.Role == role)
		}
		if !found {
			return false
		}
	}
	return true
}

// accessPath is one membership's route: scope is "trust" or a hospital id.
func accessPath(org, user uuid.UUID, scope string) string {
	return staffPath(org, "/"+user.String()+"/access/"+scope)
}

func listStaff(t *testing.T, c *apitest.Client, org uuid.UUID) []staffRow {
	t.Helper()
	r := c.Do("GET", staffPath(org, ""), nil)
	r.Want(t, http.StatusOK)
	var rows []staffRow
	r.Decode(t, &rows)
	return rows
}

func rowOf(rows []staffRow, id uuid.UUID) *staffRow {
	for i := range rows {
		if rows[i].UserID == id {
			return &rows[i]
		}
	}
	return nil
}

func invite(email string, hospital any, role string) map[string]any {
	return map[string]any{"email": email, "hospitalId": hospital, "role": role}
}

// userByEmail is the id of the one account with this email (any case), failing if there isn't one.
func userByEmail(t *testing.T, w *staffWorld, email string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.c.DB.QueryRow(t.Context(), `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&id); err != nil {
		t.Fatalf("no account for %s: %v", email, err)
	}
	return id
}

// loginByID signs in through the dev login and returns the client and the response.
func loginByID(t *testing.T, w *staffWorld, id uuid.UUID) *apitest.Resp {
	t.Helper()
	return w.c.Do("POST", "/api/dev/login", map[string]any{"userId": id})
}

func memberships(t *testing.T, w *staffWorld, user, org uuid.UUID) int {
	t.Helper()
	return count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2`, user, org)
}

func TestListStaff(t *testing.T) {
	t.Parallel()
	w := newStaffWorld(t)

	t.Run("an admin of the whole trust sees everyone with access, with their scope and role", func(t *testing.T) {
		w.fx.Member(t, w.cara, w.a2, "super_clinician") // Cara at two hospitals: still one row
		rows := listStaff(t, w.adaC, w.a1.Org)
		if len(rows) != 4 {
			t.Fatalf("got %d rows, want 4 (Ada, Sup, Cara, the A1 admin): %+v", len(rows), rows)
		}
		cara := rowOf(rows, w.cara.ID)
		if !accessIs(cara, w.a1.ID, "clinician", w.a2.ID, "super_clinician") || cara.Status != "active" ||
			cara.InviteExpiresAt != nil || cara.Email == "" {
			t.Errorf("Cara's row = %+v, want A1 clinician and A2 super clinician", cara)
		}
		if ada := rowOf(rows, w.ada.ID); !accessIs(ada, uuid.Nil, "admin") {
			t.Errorf("Ada's row = %+v, want admin of the whole trust", ada)
		}
		if rowOf(rows, w.bea.ID) != nil {
			t.Error("trust B's admin is listed in trust A")
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.c.Do("GET", staffPath(w.a1.Org, ""), nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: a super clinician, a clinician, an admin of one hospital only", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.supC, w.caraC, w.hosAdminC} {
			refused(t, c.Do("GET", staffPath(w.a1.Org, ""), nil), http.StatusForbidden)
		}
	})
	t.Run("another trust's staff is 404", func(t *testing.T) {
		refused(t, w.beaC.Do("GET", staffPath(w.a1.Org, ""), nil), http.StatusNotFound)
		refused(t, w.adaC.Do("GET", staffPath(w.b1.Org, ""), nil), http.StatusNotFound)
	})
	t.Run("invalid input: a trust id that isn't one", func(t *testing.T) {
		refused(t, w.adaC.Do("GET", "/api/o/abc/staff", nil), http.StatusNotFound)
	})
}

func TestInviteStaff(t *testing.T) {
	t.Parallel()

	t.Run("a new email gets an account, a membership and a 7-day invite", func(t *testing.T) {
		w := newStaffWorld(t)
		now := w.c.Clock.Now()
		body := invite("  New.Person@Hospital.Example ", w.a1.ID, "clinician")
		body["name"] = "Nia New"
		r := w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), body)
		r.Want(t, http.StatusCreated)
		var got staffRow
		r.Decode(t, &got)
		if got.Email != "new.person@hospital.example" || got.Name != "Nia New" || !accessIs(&got, w.a1.ID, "clinician") ||
			got.Status != "invited" || got.ExistingAccount {
			t.Errorf("response = %+v", got)
		}
		if got.InviteExpiresAt == nil || !got.InviteExpiresAt.Equal(now.Add(7*24*time.Hour).Truncate(time.Microsecond)) {
			t.Errorf("inviteExpiresAt = %v, want 7 days from %v", got.InviteExpiresAt, now)
		}
		id := userByEmail(t, w, "new.person@hospital.example")
		if id != got.UserID {
			t.Errorf("userId %s isn't the account %s", got.UserID, id)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2 AND hospital_id = $3 AND role = 'clinician'`,
			id, w.a1.Org, w.a1.ID); n != 1 {
			t.Errorf("%d memberships at A1 as clinician, want 1", n)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM staff_invites WHERE user_id = $1 AND org_id = $2 AND hospital_id = $3
			AND role = 'clinician' AND invited_by = $4 AND accepted_at IS NULL AND expires_at = created_at + interval '7 days'`,
			id, w.a1.Org, w.a1.ID, w.ada.ID); n != 1 {
			t.Errorf("%d open invites, want 1", n)
		}
		if calls := w.c.Cognito.Calls(); !slices.Equal(calls, []cognito.Call{{Op: "CreateUser", Email: "new.person@hospital.example"}}) {
			t.Errorf("Cognito calls = %+v", calls)
		}

		// Locally, they're in the dev login list at once, and signing in shows only A1.
		var users []struct{ ID uuid.UUID }
		w.c.Do("GET", "/api/dev/users", nil).Decode(t, &users)
		if !slices.ContainsFunc(users, func(u struct{ ID uuid.UUID }) bool { return u.ID == id }) {
			t.Error("the invited person isn't in the dev login list")
		}
		nia := w.c.LoginAs(t, apitest.User{ID: id, Name: "Nia New"})
		var me struct {
			Trusts []struct {
				ID        uuid.UUID
				Role      *string
				Hospitals []struct{ ID uuid.UUID }
			}
		}
		nia.Do("GET", "/api/me", nil).Decode(t, &me)
		if len(me.Trusts) != 1 || me.Trusts[0].Role != nil || len(me.Trusts[0].Hospitals) != 1 || me.Trusts[0].Hospitals[0].ID != w.a1.ID {
			t.Errorf("/api/me = %+v, want only A1", me)
		}
		nia.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusOK)
		refused(t, nia.Do("GET", w.a2.Path("/episodes"), nil), http.StatusNotFound)
		if n := count(t, w.c.DB, `SELECT count(*) FROM staff_invites WHERE user_id = $1 AND accepted_at IS NULL`, id); n != 0 {
			t.Error("the first sign-in didn't accept the invite")
		}
		if r := rowOf(listStaff(t, w.adaC, w.a1.Org), id); r == nil || r.Status != "active" || r.InviteExpiresAt != nil {
			t.Errorf("after sign-in the row is %+v, want active", r)
		}
	})

	t.Run("the whole trust: hospitalId null; a new name defaults to the email", func(t *testing.T) {
		w := newStaffWorld(t)
		r := w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("whole@trust.example", nil, "super_clinician"))
		r.Want(t, http.StatusCreated)
		var got staffRow
		r.Decode(t, &got)
		if !accessIs(&got, uuid.Nil, "super_clinician") || got.Name != "whole@trust.example" {
			t.Errorf("response = %+v", got)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2 AND hospital_id IS NULL`,
			got.UserID, w.a1.Org); n != 1 {
			t.Error("no whole-trust membership")
		}
	})

	t.Run("STF-02: an existing email only adds a membership", func(t *testing.T) {
		w := newStaffWorld(t)
		beaEmail := emailOf(t, w, w.bea.ID)
		users := count(t, w.c.DB, `SELECT count(*) FROM users`)
		r := w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite(upper(beaEmail), w.a2.ID, "clinician"))
		r.Want(t, http.StatusCreated)
		var got staffRow
		r.Decode(t, &got)
		if got.UserID != w.bea.ID || !got.ExistingAccount || got.Status != "active" || !accessIs(&got, w.a2.ID, "clinician") {
			t.Errorf("response = %+v, want Bea's existing account, active", got)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM users`); n != users {
			t.Errorf("users went from %d to %d: a second account", users, n)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM staff_invites WHERE user_id = $1 AND accepted_at IS NOT NULL`, w.bea.ID); n != 1 {
			t.Error("the invite of an existing account isn't recorded as accepted")
		}
		if calls := w.c.Cognito.Calls(); len(calls) != 0 {
			t.Errorf("Cognito was asked %+v for an existing account", calls)
		}
		w.beaC.Do("GET", w.a2.Path("/episodes"), nil).Want(t, http.StatusOK) // the new membership
		w.beaC.Do("GET", w.b1.Path("/episodes"), nil).Want(t, http.StatusOK) // and her own trust still
		refused(t, w.beaC.Do("GET", w.a1.Path("/episodes"), nil), http.StatusNotFound)
	})

	t.Run("someone with a working account who already has that access is refused: change it instead", func(t *testing.T) {
		w := newStaffWorld(t)
		email := emailOf(t, w, w.cara.ID)
		r := w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("  "+upper(email)+" ", w.a1.ID, "admin"))
		refused(t, r, http.StatusConflict)
		caraUnchanged(t, w)
	})

	t.Run("inviting someone already in the trust to another hospital adds a membership (STF-02)", func(t *testing.T) {
		w := newStaffWorld(t)
		r := w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite(emailOf(t, w, w.cara.ID), w.a2.ID, "super_clinician"))
		r.Want(t, http.StatusCreated)
		var got staffRow
		r.Decode(t, &got)
		if !got.ExistingAccount || got.Status != "active" || !accessIs(&got, w.a1.ID, "clinician", w.a2.ID, "super_clinician") {
			t.Errorf("response = %+v, want A1 clinician and A2 super clinician", got)
		}
		if memberships(t, w, w.cara.ID, w.a1.Org) != 2 {
			t.Error("not two memberships")
		}
		w.caraC.Do("GET", w.a2.Path("/episodes"), nil).Want(t, http.StatusOK)
		w.caraC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusOK)
		// …and the whole trust too, on top of them
		w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite(emailOf(t, w, w.cara.ID), nil, "clinician")).Want(t, http.StatusCreated)
		if memberships(t, w, w.cara.ID, w.a1.Org) != 3 {
			t.Error("not three memberships")
		}
	})

	t.Run("inviting someone who hasn't signed in yet sends the invite again, keeping their access", func(t *testing.T) {
		w := newStaffWorld(t)
		w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("again@x.example", w.a1.ID, "clinician")).Want(t, http.StatusCreated)
		w.c.Clock.Add(8 * 24 * time.Hour) // the first invite has expired
		r := w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("again@x.example", w.a2.ID, "super_clinician"))
		r.Want(t, http.StatusCreated)
		var got staffRow
		r.Decode(t, &got)
		if got.Status != "invited" || !accessIs(&got, w.a1.ID, "clinician", w.a2.ID, "super_clinician") {
			t.Errorf("response = %+v", got)
		}
		id := userByEmail(t, w, "again@x.example")
		if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE lower(email) = 'again@x.example'`); n != 1 {
			t.Errorf("%d accounts", n)
		}
		want := []cognito.Call{{Op: "CreateUser", Email: "again@x.example"}, {Op: "ResendInvite", Email: "again@x.example"}}
		if calls := w.c.Cognito.Calls(); !slices.Equal(calls, want) {
			t.Errorf("Cognito calls = %+v, want %+v", calls, want)
		}
		loginByID(t, w, id).Want(t, http.StatusNoContent) // the new invite is open
	})

	t.Run("resending to the same access (Resend invite) changes nothing else; a new role there is taken", func(t *testing.T) {
		w := newStaffWorld(t)
		w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("resend@x.example", w.a1.ID, "clinician")).Want(t, http.StatusCreated)
		w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("resend@x.example", w.a1.ID, "clinician")).Want(t, http.StatusCreated)
		id := userByEmail(t, w, "resend@x.example")
		if memberships(t, w, id, w.a1.Org) != 1 {
			t.Error("resending added a membership")
		}
		r := w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("resend@x.example", w.a1.ID, "admin"))
		r.Want(t, http.StatusCreated)
		var got staffRow
		r.Decode(t, &got)
		if !accessIs(&got, w.a1.ID, "admin") {
			t.Errorf("response = %+v, want A1 admin", got)
		}
		if n := len(w.c.Cognito.Calls()); n != 3 {
			t.Errorf("%d Cognito calls, want CreateUser + 2 ResendInvite", n)
		}
	})

	t.Run("no sign-in", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.c.Do("POST", staffPath(w.a1.Org, "/invites"), invite("a@b.example", w.a1.ID, "clinician")), http.StatusUnauthorized)
		noInvites(t, w)
	})

	t.Run("wrong role: a super clinician can't invite; nor can an admin of one hospital", func(t *testing.T) {
		w := newStaffWorld(t)
		for _, c := range []*apitest.Client{w.supC, w.caraC, w.hosAdminC} {
			r := c.Do("POST", staffPath(w.a1.Org, "/invites"), invite("a@b.example", w.a1.ID, "clinician"))
			refused(t, r, http.StatusForbidden)
		}
		// the role is checked before the body is read
		refused(t, w.supC.Do("POST", staffPath(w.a1.Org, "/invites"), `{"email":`), http.StatusForbidden)
		noInvites(t, w)
	})

	t.Run("another trust: its URL is 404, and its hospital in the body is refused", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.beaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("a@b.example", w.a1.ID, "clinician")), http.StatusNotFound)
		refused(t, w.adaC.Do("POST", staffPath(w.b1.Org, "/invites"), invite("a@b.example", w.b1.ID, "clinician")), http.StatusNotFound)
		refused(t, w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("a@b.example", w.b1.ID, "clinician")), http.StatusBadRequest)
		noInvites(t, w)
	})

	t.Run("invalid input", func(t *testing.T) {
		w := newStaffWorld(t)
		long := make([]byte, 201)
		for i := range long {
			long[i] = 'x'
		}
		for _, body := range []any{
			invite("", w.a1.ID, "clinician"),
			invite("not-an-email", w.a1.ID, "clinician"),
			invite("two@at@x.example", w.a1.ID, "clinician"),
			invite("Ann <ann@x.example>", w.a1.ID, "clinician"),
			invite("ann@localhost", w.a1.ID, "clinician"),
			invite("ann@x.example", w.a1.ID, "viewer"),
			invite("ann@x.example", w.a1.ID, ""),
			invite("ann@x.example", "abc", "clinician"),
			invite("ann@x.example", uuid.New(), "clinician"),              // no such hospital
			map[string]any{"email": "ann@x.example", "role": "clinician"}, // no scope given
			map[string]any{"email": "ann@x.example", "hospitalId": nil, "role": "clinician", "name": string(long)},
			`{"email":`,
		} {
			refused(t, w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), body), http.StatusBadRequest)
		}
		noInvites(t, w)
	})

	t.Run("when Cognito fails nothing is written", func(t *testing.T) {
		w := newStaffWorld(t)
		w.c.Cognito.FailWith(errors.New("cognito is down"))
		refused(t, w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("down@x.example", w.a1.ID, "clinician")), http.StatusInternalServerError)
		noInvites(t, w)
		if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE email = 'down@x.example'`); n != 0 {
			t.Error("the account was kept")
		}
	})
}

// noInvites checks that a refused invite wrote nothing.
func noInvites(t *testing.T, w *staffWorld) {
	t.Helper()
	if n := count(t, w.c.DB, `SELECT count(*) FROM staff_invites`); n != 0 {
		t.Errorf("%d invites written", n)
	}
	if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE email LIKE '%x.example' OR email LIKE '%b.example'`); n != 0 {
		t.Errorf("%d accounts written", n)
	}
	if calls := w.c.Cognito.Calls(); len(calls) != 0 {
		t.Errorf("Cognito calls = %+v", calls)
	}
}

func emailOf(t *testing.T, w *staffWorld, user uuid.UUID) string {
	t.Helper()
	var email string
	if err := w.c.DB.QueryRow(t.Context(), `SELECT email FROM users WHERE id = $1`, user).Scan(&email); err != nil {
		t.Fatal(err)
	}
	return email
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

// STF-01: an invite expires after 7 days. The first sign-in must come before then.
func TestStaffInviteExpires(t *testing.T) {
	t.Parallel()

	t.Run("signing in after 7 days is refused, and the list says the invite expired", func(t *testing.T) {
		w := newStaffWorld(t)
		w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("late@x.example", w.a1.ID, "clinician")).Want(t, http.StatusCreated)
		id := userByEmail(t, w, "late@x.example")
		w.c.Clock.Add(7*24*time.Hour + time.Minute)
		r := loginByID(t, w, id)
		refused(t, r, http.StatusForbidden)
		if r.Header("Set-Cookie") != "" {
			t.Error("a session cookie was set")
		}
		if row := rowOf(listStaff(t, w.adaC, w.a1.Org), id); row == nil || row.Status != "expired" {
			t.Errorf("row = %+v, want expired", row)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM staff_invites WHERE user_id = $1 AND accepted_at IS NULL`, id); n != 1 {
			t.Error("the expired invite was accepted")
		}
	})

	t.Run("signing in within 7 days accepts the invite for good", func(t *testing.T) {
		w := newStaffWorld(t)
		w.adaC.Do("POST", staffPath(w.a1.Org, "/invites"), invite("prompt@x.example", w.a1.ID, "clinician")).Want(t, http.StatusCreated)
		id := userByEmail(t, w, "prompt@x.example")
		w.c.Clock.Add(6 * 24 * time.Hour)
		loginByID(t, w, id).Want(t, http.StatusNoContent)
		w.c.Clock.Add(30 * 24 * time.Hour)
		loginByID(t, w, id).Want(t, http.StatusNoContent) // accepted: the deadline no longer applies
	})

	t.Run("staff who were never invited (seeded) sign in as before", func(t *testing.T) {
		w := newStaffWorld(t)
		w.c.Clock.Add(365 * 24 * time.Hour)
		loginByID(t, w, w.cara.ID).Want(t, http.StatusNoContent)
	})
}

func access(hospital any, role string) map[string]any {
	return map[string]any{"hospitalId": hospital, "role": role}
}

func TestUpdateStaffAccess(t *testing.T) {
	t.Parallel()

	t.Run("changing the role changes what they can do on their next request", func(t *testing.T) {
		w := newStaffWorld(t)
		newHQ := map[string]string{"name": "Cara's HQ", "description": "x"}
		refused(t, w.caraC.Do("POST", w.a1.Path("/questionnaires"), newHQ), http.StatusForbidden)

		r := w.adaC.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), access(nil, "super_clinician"))
		r.Want(t, http.StatusOK)
		var got staffRow
		r.Decode(t, &got)
		if got.UserID != w.cara.ID || !accessIs(&got, uuid.Nil, "super_clinician") || got.Status != "active" {
			t.Errorf("response = %+v", got)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2 AND hospital_id IS NULL AND role = 'super_clinician'`,
			w.cara.ID, w.a1.Org); n != 1 || memberships(t, w, w.cara.ID, w.a1.Org) != 1 {
			t.Error("the membership isn't one whole-trust super clinician")
		}
		w.caraC.Do("POST", w.a1.Path("/questionnaires"), newHQ).Want(t, http.StatusCreated)
		w.caraC.Do("GET", w.a2.Path("/episodes"), nil).Want(t, http.StatusOK) // the whole trust now
	})

	t.Run("changing one membership's hospital moves only that one", func(t *testing.T) {
		w := newStaffWorld(t)
		third := w.fx.HospitalIn(t, w.a1.Org)
		w.fx.Member(t, w.cara, w.a2, "clinician") // Cara: A1 and A2
		r := w.adaC.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), access(third.ID, "clinician"))
		r.Want(t, http.StatusOK)
		var got staffRow
		r.Decode(t, &got)
		if len(got.Access) != 2 {
			t.Errorf("response = %+v, want two memberships", got)
		}
		refused(t, w.caraC.Do("GET", w.a1.Path("/episodes"), nil), http.StatusNotFound)
		w.caraC.Do("GET", w.a2.Path("/episodes"), nil).Want(t, http.StatusOK)
		w.caraC.Do("GET", third.Path("/episodes"), nil).Want(t, http.StatusOK)
	})

	t.Run("moving onto access they already have is refused", func(t *testing.T) {
		w := newStaffWorld(t)
		w.fx.Member(t, w.cara, w.a2, "super_clinician")
		refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), access(w.a2.ID, "clinician")), http.StatusConflict)
		if memberships(t, w, w.cara.ID, w.a1.Org) != 2 ||
			count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND hospital_id = $2 AND role = 'super_clinician'`, w.cara.ID, w.a2.ID) != 1 {
			t.Error("Cara's memberships changed")
		}
	})

	t.Run("the last whole-trust admin can't be changed; with another admin they can", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, w.ada.ID, "trust"), access(nil, "super_clinician")), http.StatusConflict)
		refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, w.ada.ID, "trust"), access(w.a1.ID, "admin")), http.StatusConflict)
		if n := count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND hospital_id IS NULL AND role = 'admin'`, w.ada.ID); n != 1 {
			t.Error("Ada stopped being the trust's admin")
		}
		w.adaC.Do("PATCH", accessPath(w.a1.Org, w.sup.ID, "trust"), access(nil, "admin")).Want(t, http.StatusOK)
		w.adaC.Do("PATCH", accessPath(w.a1.Org, w.ada.ID, "trust"), access(nil, "clinician")).Want(t, http.StatusOK)
		refused(t, w.adaC.Do("GET", staffPath(w.a1.Org, ""), nil), http.StatusForbidden) // her next request
	})

	t.Run("no sign-in", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.c.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), access(nil, "admin")), http.StatusUnauthorized)
		caraUnchanged(t, w)
	})

	t.Run("wrong role: not even to promote themselves", func(t *testing.T) {
		w := newStaffWorld(t)
		for _, c := range []*apitest.Client{w.supC, w.caraC, w.hosAdminC} {
			refused(t, c.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), access(nil, "admin")), http.StatusForbidden)
		}
		caraUnchanged(t, w)
	})

	t.Run("another trust: its URL is 404, its staff are 404, and its hospital is refused", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.beaC.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), access(nil, "admin")), http.StatusNotFound)
		refused(t, w.adaC.Do("PATCH", accessPath(w.b1.Org, w.bea.ID, "trust"), access(nil, "clinician")), http.StatusNotFound)
		refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, w.bea.ID, "trust"), access(nil, "clinician")), http.StatusNotFound)
		refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), access(w.b1.ID, "clinician")), http.StatusBadRequest)
		caraUnchanged(t, w)
		if n := count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2 AND role = 'admin' AND hospital_id IS NULL`, w.bea.ID, w.b1.Org); n != 1 {
			t.Error("Bea's membership changed")
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		w := newStaffWorld(t)
		for _, body := range []any{
			access(w.a2.ID, "viewer"),
			access(w.a2.ID, ""),
			access("abc", "clinician"),
			access(uuid.New(), "clinician"),
			map[string]any{"role": "admin"}, // no scope: never read as the whole trust
			`{"role":`,
		} {
			refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), body), http.StatusBadRequest)
		}
		// a person or a membership that isn't there
		refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, uuid.New(), "trust"), access(nil, "clinician")), http.StatusNotFound)
		refused(t, w.adaC.Do("PATCH", staffPath(w.a1.Org, "/abc/access/trust"), access(nil, "clinician")), http.StatusNotFound)
		for _, scope := range []string{"trust", w.a2.ID.String(), "abc"} { // Cara has only A1
			refused(t, w.adaC.Do("PATCH", accessPath(w.a1.Org, w.cara.ID, scope), access(nil, "clinician")), http.StatusNotFound)
		}
		caraUnchanged(t, w)
	})
}

func caraUnchanged(t *testing.T, w *staffWorld) {
	t.Helper()
	if n := count(t, w.c.DB, `SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2 AND hospital_id = $3 AND role = 'clinician'`,
		w.cara.ID, w.a1.Org, w.a1.ID); n != 1 || memberships(t, w, w.cara.ID, w.a1.Org) != 1 {
		t.Error("Cara's membership changed")
	}
}

func TestRemoveStaffAccess(t *testing.T) {
	t.Parallel()

	t.Run("removing one membership keeps the others", func(t *testing.T) {
		w := newStaffWorld(t)
		w.fx.Member(t, w.cara, w.a2, "clinician")
		w.adaC.Do("DELETE", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), nil).Want(t, http.StatusNoContent)
		refused(t, w.caraC.Do("GET", w.a1.Path("/episodes"), nil), http.StatusNotFound) // her next request
		w.caraC.Do("GET", w.a2.Path("/episodes"), nil).Want(t, http.StatusOK)
		if r := rowOf(listStaff(t, w.adaC, w.a1.Org), w.cara.ID); !accessIs(r, w.a2.ID, "clinician") {
			t.Errorf("row = %+v, want A2 only", r)
		}
	})

	t.Run("removing the last membership ends their access to the trust (STF-04)", func(t *testing.T) {
		w := newStaffWorld(t)
		w.adaC.Do("DELETE", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), nil).Want(t, http.StatusNoContent)
		refused(t, w.caraC.Do("GET", w.a1.Path("/episodes"), nil), http.StatusNotFound)
		if rowOf(listStaff(t, w.adaC, w.a1.Org), w.cara.ID) != nil {
			t.Error("Cara is still listed")
		}
	})

	t.Run("the last whole-trust admin's membership can't be removed", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.adaC.Do("DELETE", accessPath(w.a1.Org, w.ada.ID, "trust"), nil), http.StatusConflict)
		if memberships(t, w, w.ada.ID, w.a1.Org) != 1 {
			t.Error("Ada was removed")
		}
		// but another of her memberships can go
		w.fx.Member(t, w.ada, w.a1, "clinician")
		w.adaC.Do("DELETE", accessPath(w.a1.Org, w.ada.ID, w.a1.ID.String()), nil).Want(t, http.StatusNoContent)
	})

	t.Run("no sign-in", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.c.Do("DELETE", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), nil), http.StatusUnauthorized)
		caraUnchanged(t, w)
	})

	t.Run("wrong role", func(t *testing.T) {
		w := newStaffWorld(t)
		for _, c := range []*apitest.Client{w.supC, w.caraC, w.hosAdminC} {
			refused(t, c.Do("DELETE", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), nil), http.StatusForbidden)
		}
		caraUnchanged(t, w)
	})

	t.Run("another trust", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.beaC.Do("DELETE", accessPath(w.a1.Org, w.cara.ID, w.a1.ID.String()), nil), http.StatusNotFound)
		refused(t, w.adaC.Do("DELETE", accessPath(w.b1.Org, w.bea.ID, "trust"), nil), http.StatusNotFound)
		refused(t, w.adaC.Do("DELETE", accessPath(w.a1.Org, w.bea.ID, "trust"), nil), http.StatusNotFound)
		refused(t, w.adaC.Do("DELETE", accessPath(w.a1.Org, w.cara.ID, w.b1.ID.String()), nil), http.StatusNotFound)
		caraUnchanged(t, w)
		if memberships(t, w, w.bea.ID, w.b1.Org) != 1 {
			t.Error("Bea's membership changed")
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		w := newStaffWorld(t)
		for _, scope := range []string{"abc", "trust", w.a2.ID.String()} {
			refused(t, w.adaC.Do("DELETE", accessPath(w.a1.Org, w.cara.ID, scope), nil), http.StatusNotFound)
		}
		refused(t, w.adaC.Do("DELETE", staffPath(w.a1.Org, "/abc/access/trust"), nil), http.StatusNotFound)
		caraUnchanged(t, w)
	})
}

func TestRemoveStaff(t *testing.T) {
	t.Parallel()

	t.Run("STF-04: removing someone refuses their next request", func(t *testing.T) {
		w := newStaffWorld(t)
		w.caraC.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusOK)
		w.adaC.Do("DELETE", staffPath(w.a1.Org, "/"+w.cara.ID.String()), nil).Want(t, http.StatusNoContent)
		if memberships(t, w, w.cara.ID, w.a1.Org) != 0 {
			t.Error("the membership is still there")
		}
		refused(t, w.caraC.Do("GET", w.a1.Path("/episodes"), nil), http.StatusNotFound)
		var me struct{ Trusts []any }
		w.caraC.Do("GET", "/api/me", nil).Decode(t, &me)
		if len(me.Trusts) != 0 {
			t.Errorf("/api/me still lists %d trusts", len(me.Trusts))
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM users WHERE id = $1`, w.cara.ID); n != 1 {
			t.Error("the account was deleted: past actions must keep their author")
		}
		if rowOf(listStaff(t, w.adaC, w.a1.Org), w.cara.ID) != nil {
			t.Error("Cara is still listed")
		}
	})

	t.Run("all their memberships in this trust go, and only this trust's", func(t *testing.T) {
		w := newStaffWorld(t)
		w.fx.Member(t, w.bea, w.a2, "clinician")
		w.fx.Member(t, w.bea, w.a1, "clinician")
		w.adaC.Do("DELETE", staffPath(w.a1.Org, "/"+w.bea.ID.String()), nil).Want(t, http.StatusNoContent)
		refused(t, w.beaC.Do("GET", w.a2.Path("/episodes"), nil), http.StatusNotFound)
		w.beaC.Do("GET", w.b1.Path("/episodes"), nil).Want(t, http.StatusOK)
	})

	t.Run("the last whole-trust admin can't be removed; with another admin they can remove themselves", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.adaC.Do("DELETE", staffPath(w.a1.Org, "/"+w.ada.ID.String()), nil), http.StatusConflict)
		if memberships(t, w, w.ada.ID, w.a1.Org) != 1 {
			t.Error("Ada was removed")
		}
		w.fx.TrustMember(t, w.cara, w.a1.Org, "admin")
		w.adaC.Do("DELETE", staffPath(w.a1.Org, "/"+w.ada.ID.String()), nil).Want(t, http.StatusNoContent)
		refused(t, w.adaC.Do("GET", staffPath(w.a1.Org, ""), nil), http.StatusNotFound)
	})

	t.Run("no sign-in", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.c.Do("DELETE", staffPath(w.a1.Org, "/"+w.cara.ID.String()), nil), http.StatusUnauthorized)
		caraUnchanged(t, w)
	})

	t.Run("wrong role", func(t *testing.T) {
		w := newStaffWorld(t)
		for _, c := range []*apitest.Client{w.supC, w.caraC, w.hosAdminC} {
			refused(t, c.Do("DELETE", staffPath(w.a1.Org, "/"+w.cara.ID.String()), nil), http.StatusForbidden)
		}
		caraUnchanged(t, w)
	})

	t.Run("another trust", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.beaC.Do("DELETE", staffPath(w.a1.Org, "/"+w.cara.ID.String()), nil), http.StatusNotFound)
		refused(t, w.adaC.Do("DELETE", staffPath(w.b1.Org, "/"+w.bea.ID.String()), nil), http.StatusNotFound)
		refused(t, w.adaC.Do("DELETE", staffPath(w.a1.Org, "/"+w.bea.ID.String()), nil), http.StatusNotFound)
		caraUnchanged(t, w)
		if memberships(t, w, w.bea.ID, w.b1.Org) != 1 {
			t.Error("Bea's membership changed")
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		w := newStaffWorld(t)
		refused(t, w.adaC.Do("DELETE", staffPath(w.a1.Org, "/abc"), nil), http.StatusNotFound)
		refused(t, w.adaC.Do("DELETE", staffPath(w.a1.Org, "/"+uuid.NewString()), nil), http.StatusNotFound)
		caraUnchanged(t, w)
	})
}
