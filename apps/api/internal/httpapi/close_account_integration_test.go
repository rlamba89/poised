//go:build integration

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Closing a patient account (S22, EPI-09; answers 5): POST /api/p/account/close, on the patient's
// session. The login goes (the account, its sessions and its sign-in links); every trust's
// records and episodes stay, with a General notes line; a later invite starts a new account.

func TestCloseAccount(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	sam := newPerson("1980-05-17")
	inA := w.recordIn(t, sam, w.a, w.hq, w.cara)
	inB := w.recordIn(t, sam, w.b, w.hqB, w.bea)
	// Sam's first visit (Trust A's invite) makes the account; signing in by email joins Trust B's record.
	w.anon.AsPatient(t, inA.Link, sam.dob)
	b := w.signIn(t, sam.email, sam.email, sam.dob)
	account := accountOf(t, w, inA.PatientID)
	if account == uuid.Nil || accountOf(t, w, inB.PatientID) != account {
		t.Fatalf("set-up: the records aren't joined to one account")
	}
	// Leo: someone else, with their own account, untouched by Sam's closing.
	leo := newPerson("1975-02-03")
	leoEp := w.recordIn(t, leo, w.a, w.hq, w.cara)
	leoBrowser := w.anon.AsPatient(t, leoEp.Link, leo.dob)
	leoAccount := accountOf(t, w, leoEp.PatientID)
	// Before closing, Sam has: an unused sign-in link by text, a link to confirm a new email, and
	// an open follow-up on Trust A's episode.
	w.anon.NewBrowser(t).Do("POST", "/api/p/signin", map[string]string{"contact": sam.mobile}).Want(t, http.StatusAccepted)
	pendingSignin := tokenIn(t, w.anon.Messages, signinLink, sam.mobile)
	b.Do("POST", "/api/p/contact", map[string]string{"contact": "sam.new@example.org", "kind": "email"}).Want(t, http.StatusAccepted)
	if _, err := w.db.Exec(t.Context(), `INSERT INTO jobs (kind, episode_id, run_at) VALUES ('followup', $1, now() + interval '1 day')`, inA.ID); err != nil {
		t.Fatal(err)
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.NewBrowser(t).Do("POST", "/api/p/account/close", nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: a signed-in clinician has no patient session", func(t *testing.T) {
		refused(t, w.clinician.Do("POST", "/api/p/account/close", nil), http.StatusUnauthorized)
	})
	t.Run("invalid: a session that has ended", func(t *testing.T) {
		ended := w.signIn(t, sam.email, sam.email, sam.dob)
		ended.Do("POST", "/api/p/logout", nil).Want(t, http.StatusNoContent)
		refused(t, ended.Do("POST", "/api/p/account/close", nil), http.StatusUnauthorized)
	})
	if n := count(t, w.db, `SELECT count(*) FROM patient_accounts WHERE id = $1`, account); n != 1 {
		t.Fatal("a refusal closed the account")
	}

	t.Run("Sam closes their account", func(t *testing.T) {
		r := b.Do("POST", "/api/p/account/close", nil)
		r.Want(t, http.StatusNoContent)
		if !strings.Contains(r.Header("Set-Cookie"), "Max-Age=0") {
			t.Errorf("the cookie isn't cleared: %q", r.Header("Set-Cookie"))
		}
		// The account's rows go.
		for what, sql := range map[string]string{
			"the account":      `SELECT count(*) FROM patient_accounts WHERE id = $1`,
			"its sessions":     `SELECT count(*) FROM patient_sessions WHERE account_id = $1`,
			"its verify links": `SELECT count(*) FROM login_links WHERE account_id = $1`,
		} {
			if n := count(t, w.db, sql, account); n != 0 {
				t.Errorf("%s: %d rows left", what, n)
			}
		}
		// The records and episodes stay, unlinked; each episode is noted.
		for _, e := range []apitest.Episode{inA, inB} {
			if accountOf(t, w, e.PatientID) != uuid.Nil {
				t.Errorf("record %s is still joined", e.PatientID)
			}
			if n := count(t, w.db, `SELECT count(*) FROM episodes WHERE id = $1`, e.ID); n != 1 {
				t.Errorf("episode %s is gone", e.ID)
			}
			want := "The patient closed their account on " + time.Now().In(london(t)).Format(time.DateOnly) + "."
			if got := lastNote(t, w.clinic, e.ID); got != want {
				t.Errorf("episode %s: General notes %q, want %q", e.ID, got, want)
			}
		}
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1 AND done_at IS NULL`, inA.ID); n != 0 {
			t.Error("the reminders didn't stop")
		}
		// A row in each trust's audit log.
		for _, org := range []uuid.UUID{w.a.Org, w.b.Org} {
			if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'account.close' AND org_id = $1 AND actor = 'patient' AND status = 204`, org); n != 1 {
				t.Errorf("trust %s: %d audit rows", org, n)
			}
		}
	})
	t.Run("the old session, invite links and sign-in link fail", func(t *testing.T) {
		refused(t, b.Do("GET", "/api/p/home", nil), http.StatusUnauthorized)
		refused(t, w.anon.NewBrowser(t).Do("POST", "/api/p/links/"+inA.Link+"/continue", nil), http.StatusNotFound)
		refused(t, w.anon.NewBrowser(t).Do("POST", "/api/p/signin/"+pendingSignin+"/continue", nil), http.StatusNotFound)
		// A new sign-in link isn't even sent: no account signs in with Sam's email now.
		before := len(w.anon.Messages.To(sam.email))
		w.anon.NewBrowser(t).Do("POST", "/api/p/signin", map[string]string{"contact": sam.email}).Want(t, http.StatusAccepted)
		if after := len(w.anon.Messages.To(sam.email)); after != before {
			t.Errorf("a sign-in link went to a closed account's email")
		}
	})
	t.Run("staff still see the patient and the episode", func(t *testing.T) {
		w.clinician.Do("GET", epPath(w.a, inA, ""), nil).Want(t, http.StatusOK)
		w.clinician.Do("GET", at(w.a, "/patients/"+inA.PatientID.String()), nil).Want(t, http.StatusOK)
	})
	t.Run("another patient's account is untouched", func(t *testing.T) {
		if accountOf(t, w, leoEp.PatientID) != leoAccount {
			t.Error("Leo's record left their account")
		}
		if h := readHome(t, leoBrowser); len(h.Episodes) != 1 || h.Episodes[0].ID != leoEp.ID {
			t.Errorf("Leo's home %+v", h)
		}
	})
	t.Run("a new invite starts a new account", func(t *testing.T) {
		w.clinician.Do("POST", epPath(w.a, inA, "/resend-invite"), nil).Want(t, http.StatusOK)
		link := tokenIn(t, w.anon.Messages, inviteLink, sam.mobile)
		p := w.anon.AsPatient(t, link, sam.dob)
		fresh := accountOf(t, w, inA.PatientID)
		if fresh == uuid.Nil || fresh == account {
			t.Errorf("account after the new invite %s (the closed one was %s)", fresh, account)
		}
		if h := readHome(t, p); len(h.Episodes) == 0 {
			t.Errorf("the new account's home is empty")
		}
	})
}

// A sign-in link to a contact another account also uses (twins on one phone, S15) carries on
// when one of them closes their account.
func TestCloseAccountKeepsASharedContactsLinks(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	ann := newPerson("1990-01-01")
	ann.first = "Ann"
	bea := ann // the same phone and email, another person
	bea.first, bea.last, bea.dob = "Bea", ann.last, "1992-02-02"
	annEp := w.recordIn(t, ann, w.a, w.hq, w.cara)
	beaEp := w.recordIn(t, bea, w.a, w.hq, w.cara)
	annBrowser := w.anon.AsPatient(t, annEp.Link, ann.dob)
	w.anon.AsPatient(t, beaEp.Link, bea.dob)
	w.anon.NewBrowser(t).Do("POST", "/api/p/signin", map[string]string{"contact": ann.email}).Want(t, http.StatusAccepted)
	token := tokenIn(t, w.anon.Messages, signinLink, ann.email)

	annBrowser.Do("POST", "/api/p/account/close", nil).Want(t, http.StatusNoContent)
	bb := w.anon.NewBrowser(t)
	bb.Do("POST", "/api/p/signin/"+token+"/continue", nil).Want(t, http.StatusOK)
	bb.Do("POST", "/api/p/signin/"+token+"/dob", dob(bea.dob)).Want(t, http.StatusOK)
	if h := readHome(t, bb); len(h.Episodes) != 1 || h.Episodes[0].ID != beaEp.ID {
		t.Errorf("Bea's home %+v", h)
	}
}

func london(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/London") // the deployment's zone in the tests (config's default)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}
