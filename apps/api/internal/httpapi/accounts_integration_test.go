//go:build integration

package httpapi_test

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
)

// Patient accounts across trusts, sign-in links and the patient's home (S15; A-3, PAT-06, PAT-07,
// PAT-10, PAT-11). A person's records in each trust join one account when they prove their date of
// birth through a link sent to a contact; their home then lists every trust's episodes.

var (
	signinLink = regexp.MustCompile(`http://localhost:3000/s/([A-Za-z0-9_-]{43})`)
	verifyLink = regexp.MustCompile(`http://localhost:3000/v/([A-Za-z0-9_-]{43})`)
)

// tokenIn is the token of re's link in the newest message to `to`, failing the test if there's none.
func tokenIn(t *testing.T, msgs *messaging.Recorder, re *regexp.Regexp, to string) string {
	t.Helper()
	sent := msgs.To(to)
	for i := len(sent) - 1; i >= 0; i-- {
		if got := re.FindStringSubmatch(sent[i].Body); got != nil {
			return got[1]
		}
	}
	t.Fatalf("no link to %s in %+v", to, sent)
	return ""
}

// people is a clinic with a published HQ in trust B too, for one person's records in two trusts.
type people struct {
	*clinic
	hqB apitest.HQ
}

func newPeople(t *testing.T) *people {
	t.Helper()
	w := &people{clinic: newClinic(t)}
	w.hqB = w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")
	return w
}

// person is a made-up person's details, the same in every trust that holds them.
type person struct{ first, last, dob, mobile, email string }

func newPerson(dob string) person {
	n := uuid.NewString()[:6]
	return person{first: "Sam", last: "Person " + n, dob: dob, mobile: fmt.Sprintf("+447700%06d", rand.IntN(1_000_000)), email: "sam." + n + "@example.org"}
}

// recordIn adds the person's record in a hospital's trust, with an episode there on hq.
func (w *people) recordIn(t *testing.T, p person, h apitest.Hospital, hq apitest.HQ, by apitest.User) apitest.Episode {
	t.Helper()
	pid := w.fx.Patient(t, h.Org, apitest.PatientRecord{FirstName: p.first, LastName: p.last, DateOfBirth: p.dob, Mobile: p.mobile, Email: p.email})
	return w.fx.EpisodeOf(t, hq, h, pid, by, "")
}

// accountOf is the account a record has joined (uuid.Nil: none yet).
func accountOf(t *testing.T, w *people, patient uuid.UUID) uuid.UUID {
	t.Helper()
	var id *uuid.UUID
	if err := w.db.QueryRow(t.Context(), `SELECT account_id FROM patients WHERE id = $1`, patient).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if id == nil {
		return uuid.Nil
	}
	return *id
}

type accountRow struct{ Mobile, Email, DateOfBirth string }

func readAccount(t *testing.T, w *people, id uuid.UUID) accountRow {
	t.Helper()
	var a accountRow
	if err := w.db.QueryRow(t.Context(), `SELECT mobile, email, to_char(date_of_birth, 'YYYY-MM-DD') FROM patient_accounts WHERE id = $1`,
		id).Scan(&a.Mobile, &a.Email, &a.DateOfBirth); err != nil {
		t.Fatal(err)
	}
	return a
}

// signIn asks for a sign-in link to contact and opens it with the date of birth, in a fresh
// browser, as a returning patient would. It returns the browser, holding the session.
func (w *people) signIn(t *testing.T, contact, to, dateOfBirth string) *apitest.Client {
	t.Helper()
	b := w.anon.NewBrowser(t)
	b.Do("POST", "/api/p/signin", map[string]string{"contact": contact}).Want(t, http.StatusAccepted)
	token := tokenIn(t, w.anon.Messages, signinLink, to)
	b.Do("POST", "/api/p/signin/"+token+"/continue", nil).Want(t, http.StatusOK)
	b.Do("POST", "/api/p/signin/"+token+"/dob", dob(dateOfBirth)).Want(t, http.StatusOK)
	return b
}

type homeEpisode struct {
	ID                uuid.UUID
	PatientName       string
	Hospital          string
	QuestionnaireName string
	Procedure         string
	State             string
}

type home struct {
	FirstName string
	Episodes  []homeEpisode
	Contacts  struct{ Mobile, Email string }
}

func readHome(t *testing.T, b *apitest.Client) home {
	t.Helper()
	r := b.Do("GET", "/api/p/home", nil)
	r.Want(t, http.StatusOK)
	var h home
	r.Decode(t, &h)
	return h
}

func homeIDs(h home) []uuid.UUID {
	var out []uuid.UUID
	for _, e := range h.Episodes {
		out = append(out, e.ID)
	}
	return out
}

func sameIDs(got, want []uuid.UUID) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestAccountLinking(t *testing.T) {
	t.Parallel()
	w := newPeople(t)

	t.Run("the first right date of birth makes an account from the record, and joins it", func(t *testing.T) {
		p := newPerson("1980-05-17")
		e := w.recordIn(t, p, w.a, w.hq, w.cara)
		if accountOf(t, w, e.PatientID) != uuid.Nil {
			t.Fatal("joined before the date of birth")
		}
		w.anon.Do("POST", linkPath(e, "/continue"), nil).Want(t, http.StatusOK)
		w.anon.Do("POST", linkPath(e, "/dob"), dob("1999-01-01")).Want(t, http.StatusUnauthorized)
		if accountOf(t, w, e.PatientID) != uuid.Nil {
			t.Fatal("a wrong date of birth joined an account")
		}
		w.anon.AsPatient(t, e.Link, p.dob)
		acc := accountOf(t, w, e.PatientID)
		if acc == uuid.Nil {
			t.Fatal("no account")
		}
		if got := readAccount(t, w, acc); got != (accountRow{p.mobile, p.email, p.dob}) {
			t.Errorf("account = %+v", got)
		}
		w.anon.AsPatient(t, e.Link, p.dob) // again: the same account
		if accountOf(t, w, e.PatientID) != acc || count(t, w.db, `SELECT count(*) FROM patient_accounts WHERE date_of_birth = $1`, p.dob) != 1 {
			t.Error("a second account")
		}
	})
	t.Run("another trust's record with the same mobile and date of birth joins the same account (PAT-07)", func(t *testing.T) {
		p := newPerson("1981-06-18")
		ea := w.recordIn(t, p, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, ea.Link, p.dob)
		q := p
		q.email = "another." + p.email // trust B has another email
		eb := w.recordIn(t, q, w.b, w.hqB, w.bea)
		w.anon.AsPatient(t, eb.Link, p.dob)
		if a, b := accountOf(t, w, ea.PatientID), accountOf(t, w, eb.PatientID); a == uuid.Nil || a != b {
			t.Errorf("accounts %s and %s", a, b)
		}
	})
	t.Run("the same mobile with another date of birth is someone else: a new account", func(t *testing.T) {
		p := newPerson("1950-01-02")
		parent := w.recordIn(t, p, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, parent.Link, p.dob)
		child := p
		child.dob, child.first = "2015-03-04", "Kim"
		kid := w.recordIn(t, child, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, kid.Link, child.dob)
		if a, b := accountOf(t, w, parent.PatientID), accountOf(t, w, kid.PatientID); a == b {
			t.Error("two people sharing a phone were joined")
		}
	})
	t.Run("opening an invite also joins the person's other records with that contact and date of birth", func(t *testing.T) {
		p := newPerson("1982-07-19")
		eb := w.recordIn(t, p, w.b, w.hqB, w.bea) // never opened
		ea := w.recordIn(t, p, w.a, w.hq, w.cara)
		other := p
		other.mobile, other.email = "+447700900999", "someone.else@example.org" // another contact: not joined
		ec := w.recordIn(t, other, w.a2, w.hq, w.cara)
		w.anon.AsPatient(t, ea.Link, p.dob)
		acc := accountOf(t, w, ea.PatientID)
		if accountOf(t, w, eb.PatientID) != acc {
			t.Error("trust B's record wasn't joined")
		}
		// Trust B's log says its record joined the account, with no hospital (a trust's record).
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'account.join' AND org_id = $1 AND hospital_id IS NULL
			AND actor = 'patient' AND patient_ids = ARRAY[$2::uuid]`, w.b.Org, eb.PatientID); n != 1 {
			t.Errorf("trust B's audit rows: %d", n)
		}
		if accountOf(t, w, ec.PatientID) != uuid.Nil {
			t.Error("a record with other contacts was joined")
		}
	})
	t.Run("twins on one phone (same date of birth and mobile, other first names) stay two accounts (S15 answers 6)", func(t *testing.T) {
		kim := newPerson("2016-04-05")
		kim.first = "Kim"
		lee := kim
		lee.first, lee.last = "Lee", kim.last
		ek := w.recordIn(t, kim, w.a, w.hq, w.cara)
		el := w.recordIn(t, lee, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, ek.Link, kim.dob)
		if accountOf(t, w, el.PatientID) != uuid.Nil {
			t.Fatal("the twin's record was swept into the account")
		}
		w.anon.AsPatient(t, el.Link, lee.dob)
		if a, b := accountOf(t, w, ek.PatientID), accountOf(t, w, el.PatientID); a == uuid.Nil || a == b {
			t.Errorf("twins' accounts %s and %s", a, b)
		}
		h := readHome(t, w.anon.AsPatient(t, el.Link, lee.dob))
		if !sameIDs(homeIDs(h), []uuid.UUID{el.ID}) {
			t.Errorf("Lee's home lists %v", homeIDs(h))
		}
	})
	t.Run("the first name matches ignoring case and spaces; another spelling stays unjoined", func(t *testing.T) {
		p := newPerson("1987-12-24")
		p.first = "Mary Ann"
		ea := w.recordIn(t, p, w.a, w.hq, w.cara)
		same := p
		same.first = " maryann "
		eb := w.recordIn(t, same, w.b, w.hqB, w.bea)
		other := p
		other.first = "Marianne"
		ec := w.recordIn(t, other, w.a2, w.hq, w.cara)
		w.anon.AsPatient(t, ea.Link, p.dob)
		acc := accountOf(t, w, ea.PatientID)
		if accountOf(t, w, eb.PatientID) != acc {
			t.Error("the same name typed differently wasn't joined")
		}
		if accountOf(t, w, ec.PatientID) != uuid.Nil {
			t.Error("another spelling was joined")
		}
		w.anon.AsPatient(t, ec.Link, p.dob) // its own invite: its own account, not Mary Ann's
		if got := accountOf(t, w, ec.PatientID); got == uuid.Nil || got == acc {
			t.Errorf("Marianne's account %s", got)
		}
	})
	t.Run("the same first name, date of birth and mobile with another surname is someone else (S15 answers 8)", func(t *testing.T) {
		p := newPerson("2016-04-05")
		p.first, p.last = "Kim", "TwinRT "+uuid.NewString()[:4]
		other := p
		other.last = "OtherRT " + uuid.NewString()[:4]
		et := w.recordIn(t, p, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, et.Link, p.dob)
		eo := w.recordIn(t, other, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, eo.Link, other.dob)
		if a, b := accountOf(t, w, et.PatientID), accountOf(t, w, eo.PatientID); a == uuid.Nil || a == b {
			t.Fatalf("accounts %s and %s", a, b)
		}
		if h := readHome(t, w.anon.AsPatient(t, et.Link, p.dob)); !sameIDs(homeIDs(h), []uuid.UUID{et.ID}) {
			t.Errorf("Kim TwinRT's home lists %v", homeIDs(h))
		}
	})
	t.Run("the same full name typed with other case and spaces joins; the home greets them as their name is written", func(t *testing.T) {
		p := newPerson("1991-05-06")
		ea := w.recordIn(t, p, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, ea.Link, p.dob)
		odd := p
		odd.first, odd.last = " sam ", strings.ToLower(strings.ReplaceAll(p.last, " ", "  "))
		w.anon.Clock.Add(time.Minute)
		eb := w.recordIn(t, odd, w.b, w.hqB, w.bea)
		w.anon.AsPatient(t, eb.Link, p.dob)
		if accountOf(t, w, eb.PatientID) != accountOf(t, w, ea.PatientID) {
			t.Fatal("not joined")
		}
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET created_at = now() + interval '1 day' WHERE id = $1`, eb.ID); err != nil {
			t.Fatal(err)
		}
		// The greeting takes the name written with a capital; each card shows its hospital's record as written.
		if h := readHome(t, w.anon.AsPatient(t, ea.Link, p.dob)); h.FirstName != "Sam" || h.Episodes[0].PatientName != "sam "+odd.last {
			t.Errorf("greeting %q, newest card %q", h.FirstName, h.Episodes[0].PatientName)
		}
	})
	t.Run("a record joining an existing account is in its trust's log, and takes the account's contacts it lacks", func(t *testing.T) {
		p := newPerson("1988-01-25")
		ea := w.recordIn(t, p, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, ea.Link, p.dob)
		acc := accountOf(t, w, ea.PatientID)
		q := p
		q.email = "" // trust B has only the mobile
		eb := w.recordIn(t, q, w.b, w.hqB, w.bea)
		if _, err := w.db.Exec(t.Context(), `UPDATE patients SET account_id = NULL WHERE id = $1`, eb.PatientID); err != nil {
			t.Fatal(err)
		}
		before := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'account.join' AND org_id = $1 AND $2 = ANY(patient_ids)`, w.b.Org, eb.PatientID)
		w.anon.AsPatient(t, eb.Link, p.dob) // its own invite joins it to the existing account
		if accountOf(t, w, eb.PatientID) != acc {
			t.Fatal("not joined")
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'account.join' AND org_id = $1 AND $2 = ANY(patient_ids)`, w.b.Org, eb.PatientID); n-before != 1 {
			t.Errorf("%d account.join rows", n-before)
		}
		var email string
		if err := w.db.QueryRow(t.Context(), `SELECT email FROM patients WHERE id = $1`, eb.PatientID).Scan(&email); err != nil {
			t.Fatal(err)
		}
		if email != p.email {
			t.Errorf("trust B's record email %q, want the account's %q", email, p.email)
		}
	})
	t.Run("a record with no contacts gets an account of its own", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		w.anon.AsPatient(t, e.Link, e.DateOfBirth)
		acc := accountOf(t, w, e.PatientID)
		if acc == uuid.Nil {
			t.Fatal("no account")
		}
		if got := readAccount(t, w, acc); got.Mobile != "" || got.Email != "" {
			t.Errorf("account = %+v", got)
		}
	})
}

func TestPatientHome(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	p := newPerson("1983-08-20")
	first := w.recordIn(t, p, w.a, w.hq, w.cara)
	w.anon.Clock.Add(time.Minute)
	atA2 := w.fx.EpisodeOf(t, w.hq, w.a2, first.PatientID, w.cara, "")
	w.anon.Clock.Add(time.Minute)
	inB := w.recordIn(t, p, w.b, w.hqB, w.bea)
	// The episodes' times come from the database; make the order certain.
	for i, id := range []uuid.UUID{first.ID, atA2.ID, inB.ID} {
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET created_at = now() - make_interval(days => $2) WHERE id = $1`, id, 10-i); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET procedure = 'Hip replacement' WHERE id = $1`, inB.ID); err != nil {
		t.Fatal(err)
	}

	t.Run("happy path: every trust's episodes, newest first, with the hospital's name; nothing they didn't type", func(t *testing.T) {
		b := w.anon.AsPatient(t, first.Link, p.dob)
		r := b.Do("GET", "/api/p/home", nil)
		r.Want(t, http.StatusOK)
		if r.Header("Cache-Control") != "no-store" {
			t.Errorf("Cache-Control %q", r.Header("Cache-Control"))
		}
		var h home
		r.Decode(t, &h)
		if !sameIDs(homeIDs(h), []uuid.UUID{inB.ID, atA2.ID, first.ID}) {
			t.Fatalf("episodes %v", homeIDs(h))
		}
		if h.Episodes[0].Hospital != hospitalName(t, w.clinic, w.b) || h.Episodes[1].Hospital != hospitalName(t, w.clinic, w.a2) ||
			h.Episodes[2].Hospital != hospitalName(t, w.clinic, w.a) {
			t.Errorf("hospitals %+v", h.Episodes)
		}
		if h.Episodes[0].Procedure != "Hip replacement" || h.Episodes[0].QuestionnaireName == "" || h.Episodes[0].State != "to_do" {
			t.Errorf("episode %+v", h.Episodes[0])
		}
		if h.FirstName != "Sam" {
			t.Errorf("first name %q", h.FirstName)
		}
		// Each card says whose questionnaire it is (S15 answers 6); never their date of birth, contacts or ids.
		for _, e := range h.Episodes {
			if e.PatientName != "Sam "+p.last {
				t.Errorf("patient name %q", e.PatientName)
			}
		}
		for _, hidden := range []string{p.dob, p.mobile, p.email, first.PatientID.String(), inB.PatientID.String()} {
			if strings.Contains(string(r.Body), hidden) {
				t.Errorf("the home holds %q: %s", hidden, r.Body)
			}
		}
		if !strings.HasSuffix(h.Contacts.Mobile, p.mobile[len(p.mobile)-3:]) || !strings.HasSuffix(h.Contacts.Email, "@example.org") ||
			!strings.HasPrefix(h.Contacts.Email, "s") {
			t.Errorf("contacts %+v", h.Contacts)
		}
	})
	t.Run("each episode's state: to do, in progress, sent", func(t *testing.T) {
		b := w.anon.AsPatient(t, first.Link, p.dob)
		b.Do("PUT", patientPath(atA2, "/answers/"+w.hq.Chapters[0].String()), map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 0}).Want(t, http.StatusOK)
		b.Do("POST", patientPath(first, "/submit"), nil).Want(t, http.StatusNoContent)
		h := readHome(t, b)
		states := map[uuid.UUID]string{}
		for _, e := range h.Episodes {
			states[e.ID] = e.State
		}
		if states[inB.ID] != "to_do" || states[atA2.ID] != "in_progress" || states[first.ID] != "sent" {
			t.Errorf("states %v", states)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		refused(t, w.anon.Do("GET", "/api/p/home", nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: a signed-in clinician has no patient session", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", "/api/p/home", nil), http.StatusUnauthorized)
	})
	t.Run("another person's episodes, and an unjoined record's, aren't listed", func(t *testing.T) {
		someone := w.recordIn(t, newPerson(p.dob), w.a, w.hq, w.cara) // same date of birth, other contacts
		w.anon.AsPatient(t, someone.Link, p.dob)
		h := readHome(t, w.anon.AsPatient(t, someone.Link, p.dob))
		if !sameIDs(homeIDs(h), []uuid.UUID{someone.ID}) {
			t.Errorf("their home lists %v", homeIDs(h))
		}
		h = readHome(t, w.anon.AsPatient(t, first.Link, p.dob))
		for _, id := range homeIDs(h) {
			if id == someone.ID {
				t.Error("someone else's episode is listed")
			}
		}
	})
	t.Run("a suspended trust's episodes are left out (ORG-03)", func(t *testing.T) {
		b := w.anon.AsPatient(t, first.Link, p.dob)
		if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET status = 'suspended' WHERE id = $1`, w.b.Org); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `UPDATE orgs SET status = 'active' WHERE id = $1`, w.b.Org)
		for _, id := range homeIDs(readHome(t, b)) {
			if id == inB.ID {
				t.Error("a suspended trust's episode is listed")
			}
		}
	})
	t.Run("the home is in each trust's audit log: a row per hospital it showed", func(t *testing.T) {
		b := w.anon.AsPatient(t, first.Link, p.dob)
		before := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'home.view'`)
		readHome(t, b)
		rows, err := w.db.Query(t.Context(), `SELECT org_id, hospital_id, actor, user_id IS NULL, route, status, patient_ids, episode_ids
			FROM audit_log WHERE action = 'home.view' ORDER BY id DESC LIMIT 3`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		got := map[uuid.UUID][]uuid.UUID{} // hospital → episodes
		for rows.Next() {
			var org, hospital uuid.UUID
			var actor, route string
			var noUser bool
			var status int
			var patients, episodes []uuid.UUID
			if err := rows.Scan(&org, &hospital, &actor, &noUser, &route, &status, &patients, &episodes); err != nil {
				t.Fatal(err)
			}
			if actor != "patient" || !noUser || route != "GET /api/p/home" || status != 200 || len(patients) != 1 {
				t.Errorf("row %s %s %v %s %d %v", org, actor, noUser, route, status, patients)
			}
			got[hospital] = episodes
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'home.view'`); n-before != 3 {
			t.Fatalf("%d rows", n-before)
		}
		if !sameIDs(got[w.a.ID], []uuid.UUID{first.ID}) || !sameIDs(got[w.a2.ID], []uuid.UUID{atA2.ID}) || !sameIDs(got[w.b.ID], []uuid.UUID{inB.ID}) {
			t.Errorf("rows %v", got)
		}
	})
	t.Run("an episode from the home opens with the same session, in either trust", func(t *testing.T) {
		b := w.anon.AsPatient(t, first.Link, p.dob)
		b.Do("GET", patientPath(inB, ""), nil).Want(t, http.StatusOK)
		b.Do("GET", patientPath(atA2, ""), nil).Want(t, http.StatusOK)
	})
}

func TestSignin(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	p := newPerson("1984-09-21")
	ep := w.recordIn(t, p, w.a, w.hq, w.cara)
	w.anon.AsPatient(t, ep.Link, p.dob) // their first visit made their account
	fresh := func() { w.anon.Clock.Add(2 * time.Hour) } // past every rate limit's window

	t.Run("happy path: a link by email, Continue, the date of birth, then the home", func(t *testing.T) {
		fresh()
		b := w.anon.NewBrowser(t)
		r := b.Do("POST", "/api/p/signin", map[string]string{"contact": " " + strings.ToUpper(p.email) + " "})
		r.Want(t, http.StatusAccepted)
		var got struct{ Channel string }
		r.Decode(t, &got)
		if got.Channel != "email" {
			t.Errorf("channel %q", got.Channel)
		}
		sent := w.anon.Messages.To(p.email)
		if len(sent) != 1 || !strings.Contains(sent[0].Body, "15 minutes") || sent[0].Subject == "" {
			t.Fatalf("sent %+v", sent)
		}
		if !strings.Contains(sent[0].Subject, "Preopsis") || !strings.Contains(sent[0].Body, "sign in to Preopsis") {
			t.Errorf("the sign-in email doesn't name Preopsis: %+v", sent[0])
		}
		token := tokenIn(t, w.anon.Messages, signinLink, p.email)
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE kind = 'signin' AND contact = $1 AND patient_id IS NULL
			AND expires_at = created_at + interval '15 minutes'`, p.email); n != 1 {
			t.Errorf("%d sign-in links", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE token_hash::text LIKE '%' || $1 || '%'`, token); n != 0 {
			t.Error("the token is stored")
		}
		c := b.Do("POST", "/api/p/signin/"+token+"/continue", nil)
		c.Want(t, http.StatusOK)
		if c.Header("Set-Cookie") != "" || strings.Contains(string(c.Body), "Sam") {
			t.Errorf("Continue: %s %s", c, c.Header("Set-Cookie"))
		}
		d := b.Do("POST", "/api/p/signin/"+token+"/dob", dob(p.dob))
		d.Want(t, http.StatusOK)
		if d.Header("Cache-Control") != "no-store" {
			t.Errorf("Cache-Control %q", d.Header("Cache-Control"))
		}
		if !sameIDs(homeIDs(readHome(t, b)), []uuid.UUID{ep.ID}) {
			t.Error("home")
		}
		b.Do("GET", patientPath(ep, ""), nil).Want(t, http.StatusOK)
	})
	t.Run("by text message: the mobile as typed, in the deployment's country", func(t *testing.T) {
		fresh()
		typed := "0" + p.mobile[3:7] + " " + p.mobile[7:]
		b := w.signIn(t, typed, p.mobile, p.dob)
		readHome(t, b)
	})
	t.Run("a sign-in link works once", func(t *testing.T) {
		fresh()
		b := w.signIn(t, p.email, p.email, p.dob)
		token := tokenIn(t, w.anon.Messages, signinLink, p.email)
		for _, step := range []string{"/continue", "/dob"} {
			r := w.anon.NewBrowser(t).Do("POST", "/api/p/signin/"+token+step, dob(p.dob))
			refused(t, r, http.StatusNotFound)
			if !strings.Contains(string(r.Body), `"code":"expired"`) {
				t.Errorf("%s: %s", step, r)
			}
		}
		readHome(t, b) // the session it started carries on
	})
	t.Run("a sign-in link expires after 15 minutes; its session doesn't end with it", func(t *testing.T) {
		fresh()
		b := w.signIn(t, p.email, p.email, p.dob)
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email}).Want(t, http.StatusAccepted)
		late := tokenIn(t, w.anon.Messages, signinLink, p.email)
		w.anon.Clock.Add(14 * time.Minute)
		w.anon.NewBrowser(t).Do("POST", "/api/p/signin/"+late+"/continue", nil).Want(t, http.StatusOK)
		readHome(t, b)
		w.anon.Clock.Add(2 * time.Minute)
		refused(t, w.anon.NewBrowser(t).Do("POST", "/api/p/signin/"+late+"/continue", nil), http.StatusNotFound)
		refused(t, w.anon.NewBrowser(t).Do("POST", "/api/p/signin/"+late+"/dob", dob(p.dob)), http.StatusNotFound)
		readHome(t, b) // 16 minutes on, the session (30 minutes idle) is still open
	})
	t.Run("no account: the same answer, and nothing is sent", func(t *testing.T) {
		fresh()
		before := len(w.anon.Messages.Sent())
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": "nobody@example.org"}).Want(t, http.StatusAccepted)
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": "07700 900001"}).Want(t, http.StatusAccepted)
		if len(w.anon.Messages.Sent()) != before {
			t.Error("a message went to a contact with no account")
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE contact IN ('nobody@example.org', '+447700900001')`); n != 0 {
			t.Errorf("%d links", n)
		}
	})
	t.Run("a send that fails gives the same answer as no account (S17: real sending can fail)", func(t *testing.T) {
		fresh()
		w.anon.Messages.FailWith(errors.New("refused"))
		defer w.anon.Messages.FailWith(nil)
		r := w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email})
		r.Want(t, http.StatusAccepted)
		var got struct{ Channel string }
		r.Decode(t, &got)
		if got.Channel != "email" {
			t.Errorf("channel %q", got.Channel)
		}
	})
	t.Run("a record known only to a hospital (never opened) can't be signed in to: invite links do that", func(t *testing.T) {
		fresh()
		q := newPerson("1984-01-01")
		w.recordIn(t, q, w.a, w.hq, w.cara)
		before := len(w.anon.Messages.Sent())
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": q.email}).Want(t, http.StatusAccepted)
		if len(w.anon.Messages.Sent()) != before {
			t.Error("sent")
		}
	})
	t.Run("a wrong date of birth: tries left, and the fifth locks the link", func(t *testing.T) {
		fresh()
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email}).Want(t, http.StatusAccepted)
		token := tokenIn(t, w.anon.Messages, signinLink, p.email)
		b := w.anon.NewBrowser(t)
		for i := 4; i >= 1; i-- {
			r := b.Do("POST", "/api/p/signin/"+token+"/dob", dob("1999-09-09"))
			refused(t, r, http.StatusUnauthorized)
			if !strings.Contains(r.Message(), "left") {
				t.Errorf("message %q", r.Message())
			}
		}
		r := b.Do("POST", "/api/p/signin/"+token+"/dob", dob("1999-09-09"))
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(string(r.Body), `"code":"locked"`) || !strings.Contains(r.Message(), "new sign-in link") {
			t.Errorf("got %s", r)
		}
		refused(t, b.Do("POST", "/api/p/signin/"+token+"/dob", dob(p.dob)), http.StatusForbidden)
		refused(t, b.Do("GET", "/api/p/home", nil), http.StatusUnauthorized)
		if b.Cookie("poised_patient", "/api/p/") != "" {
			t.Error("a session cookie")
		}
	})
	t.Run("two people sharing a mobile: the date of birth decides who is signed in", func(t *testing.T) {
		fresh()
		parent := newPerson("1951-02-03")
		pe := w.recordIn(t, parent, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, pe.Link, parent.dob)
		child := parent
		child.dob, child.first, child.email = "2016-04-05", "Kim", ""
		ke := w.recordIn(t, child, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, ke.Link, child.dob)
		h := readHome(t, w.signIn(t, parent.mobile, parent.mobile, child.dob))
		if !sameIDs(homeIDs(h), []uuid.UUID{ke.ID}) || h.FirstName != "Kim" {
			t.Errorf("the child's home: %+v", h)
		}
		fresh()
		h = readHome(t, w.signIn(t, parent.mobile, parent.mobile, parent.dob))
		if !sameIDs(homeIDs(h), []uuid.UUID{pe.ID}) {
			t.Errorf("the parent's home: %v", homeIDs(h))
		}
	})
	t.Run("twins sharing a mobile: the name is asked for, and picks who is signed in", func(t *testing.T) {
		fresh()
		kim := newPerson("2017-06-07")
		kim.first, kim.email = "Kim", ""
		lee := kim
		lee.first = "Lee"
		ek := w.recordIn(t, kim, w.a, w.hq, w.cara)
		el := w.recordIn(t, lee, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, ek.Link, kim.dob)
		w.anon.AsPatient(t, el.Link, lee.dob)
		b := w.anon.NewBrowser(t)
		b.Do("POST", "/api/p/signin", map[string]string{"contact": kim.mobile}).Want(t, http.StatusAccepted)
		token := tokenIn(t, w.anon.Messages, signinLink, kim.mobile)
		r := b.Do("POST", "/api/p/signin/"+token+"/dob", dob(kim.dob))
		refused(t, r, http.StatusConflict)
		if !strings.Contains(string(r.Body), `"code":"name"`) || !strings.Contains(r.Message(), "first name and surname") ||
			b.Cookie("poised_patient", "/api/p/") != "" {
			t.Fatalf("got %s", r)
		}
		if l := readLink(t, w.clinic, token); l.Attempts != 0 || l.Used != nil {
			t.Errorf("link %+v", l)
		}
		wrong := b.Do("POST", "/api/p/signin/"+token+"/dob", map[string]string{"dateOfBirth": kim.dob, "firstName": "Lee", "lastName": "Someone"})
		refused(t, wrong, http.StatusUnauthorized)
		if !strings.Contains(wrong.Message(), "name don't match") || !strings.Contains(wrong.Message(), "check them") {
			t.Errorf("message %q", wrong.Message())
		}
		if l := readLink(t, w.clinic, token); l.Attempts != 1 {
			t.Errorf("a wrong name isn't a try: %+v", l)
		}
		// A name too long to be anyone's is refused plainly, and isn't a try.
		long := b.Do("POST", "/api/p/signin/"+token+"/dob", map[string]string{"dateOfBirth": kim.dob, "firstName": strings.Repeat("é", 300), "lastName": "X"})
		refused(t, long, http.StatusBadRequest)
		if !strings.Contains(long.Message(), "too long") {
			t.Errorf("message %q", long.Message())
		}
		if l := readLink(t, w.clinic, token); l.Attempts != 1 {
			t.Errorf("attempts %d", l.Attempts)
		}
		b.Do("POST", "/api/p/signin/"+token+"/dob", map[string]string{"dateOfBirth": kim.dob, "firstName": " LEE ",
			"lastName": strings.ToUpper(strings.ReplaceAll(kim.last, " ", "  "))}).Want(t, http.StatusOK)
		h := readHome(t, b)
		if !sameIDs(homeIDs(h), []uuid.UUID{el.ID}) || h.FirstName != "Lee" {
			t.Errorf("Lee's home: %+v", h)
		}
	})
	t.Run("a sign-in link to a contact no account uses any more says it has expired, not a wrong date", func(t *testing.T) {
		fresh()
		q := newPerson("1990-03-04")
		eq := w.recordIn(t, q, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, eq.Link, q.dob)
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": q.email}).Want(t, http.StatusAccepted)
		token := tokenIn(t, w.anon.Messages, signinLink, q.email)
		if _, err := w.db.Exec(t.Context(), `UPDATE patient_accounts SET email = 'moved.on@example.org' WHERE email = $1`, q.email); err != nil {
			t.Fatal(err)
		}
		r := w.anon.NewBrowser(t).Do("POST", "/api/p/signin/"+token+"/dob", dob(q.dob))
		refused(t, r, http.StatusNotFound)
		if !strings.Contains(string(r.Body), `"code":"expired"`) {
			t.Errorf("got %s", r)
		}
		if l := readLink(t, w.clinic, token); l.Attempts != 0 {
			t.Errorf("attempts %d", l.Attempts)
		}
	})
	t.Run("signing in joins the person's records with that contact and date of birth", func(t *testing.T) {
		fresh()
		later := w.recordIn(t, p, w.b, w.hqB, w.bea) // trust B invited them; they never opened it
		h := readHome(t, w.signIn(t, p.email, p.email, p.dob))
		found := false
		for _, id := range homeIDs(h) {
			found = found || id == later.ID
		}
		if !found {
			t.Errorf("trust B's episode isn't on the home: %v", homeIDs(h))
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'account.join' AND org_id = $1
			AND route = 'POST /api/p/signin/{token}/dob' AND $2 = ANY(patient_ids)`, w.b.Org, later.PatientID); n != 1 {
			t.Errorf("trust B's audit rows: %d", n)
		}
	})
	t.Run("invalid input: not a mobile or an email", func(t *testing.T) {
		fresh()
		for _, body := range []any{map[string]string{"contact": ""}, map[string]string{"contact": "hello"}, map[string]string{"contact": "a@b"},
			map[string]string{"contact": "123"}, `{"contact":`, map[string]int{"contact": 1}} {
			refused(t, w.anon.Do("POST", "/api/p/signin", body), http.StatusBadRequest)
		}
		refused(t, w.anon.Do("POST", "/api/p/signin/short/continue", nil), http.StatusNotFound)
		refused(t, w.anon.Do("POST", "/api/p/signin/"+strings.Repeat("A", 43)+"/dob", dob(p.dob)), http.StatusNotFound)
		refused(t, w.anon.Do("POST", "/api/p/signin/"+strings.Repeat("A", 43)+"/continue", nil), http.StatusNotFound)
	})
	t.Run("an invite's link isn't a sign-in link, nor the other way round", func(t *testing.T) {
		fresh()
		refused(t, w.anon.Do("POST", "/api/p/signin/"+ep.Link+"/continue", nil), http.StatusNotFound)
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email}).Want(t, http.StatusAccepted)
		token := tokenIn(t, w.anon.Messages, signinLink, p.email)
		refused(t, w.anon.Do("POST", "/api/p/links/"+token+"/continue", nil), http.StatusNotFound)
	})
	t.Run("a mobile in another country is refused, and nothing is sent (PAT-11)", func(t *testing.T) {
		fresh()
		before := len(w.anon.Messages.Sent())
		r := w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": "+1 202 555 0143"})
		refused(t, r, http.StatusBadRequest)
		if !strings.Contains(r.Message(), "email") {
			t.Errorf("message %q", r.Message())
		}
		if len(w.anon.Messages.Sent()) != before {
			t.Error("sent")
		}
	})
	t.Run("wrong role: a staff sign-in doesn't make a patient session", func(t *testing.T) {
		fresh()
		refused(t, w.clinician.Do("GET", "/api/p/home", nil), http.StatusUnauthorized)
	})
}

func TestSigninRateLimits(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	p := newPerson("1985-10-22")
	ep := w.recordIn(t, p, w.a, w.hq, w.cara)
	w.anon.AsPatient(t, ep.Link, p.dob)

	t.Run("3 links per contact in 15 minutes; the 4th is refused, sends nothing, and says so kindly", func(t *testing.T) {
		for range 3 {
			w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email}).Want(t, http.StatusAccepted)
		}
		r := w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email})
		refused(t, r, http.StatusTooManyRequests)
		if !strings.Contains(r.Message(), "15 minutes") {
			t.Errorf("message %q", r.Message())
		}
		if n := len(w.anon.Messages.To(p.email)); n != 3 {
			t.Errorf("%d messages", n)
		}
		// The same person by text: another contact, its own count.
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.mobile}).Want(t, http.StatusAccepted)
		w.anon.Clock.Add(16 * time.Minute)
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email}).Want(t, http.StatusAccepted)
	})
	t.Run("10 requests an hour from one IP address, whatever the contact", func(t *testing.T) {
		w.anon.Clock.Add(2 * time.Hour)
		for i := range 10 {
			w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": "nobody" + string(rune('a'+i)) + "@example.org"}).Want(t, http.StatusAccepted)
		}
		r := w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email})
		refused(t, r, http.StatusTooManyRequests)
		if !strings.Contains(r.Message(), "hour") {
			t.Errorf("message %q", r.Message())
		}
		w.anon.Clock.Add(61 * time.Minute)
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email}).Want(t, http.StatusAccepted)
	})
	t.Run("in capture mode (local, QA), the captured-messages page can clear the counts for the next test", func(t *testing.T) {
		for range 3 {
			w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": "reset@example.org"}).Want(t, http.StatusAccepted)
		}
		refused(t, w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": "reset@example.org"}), http.StatusTooManyRequests)
		w.anon.Do("DELETE", "/api/qa/rate-limits", nil).Want(t, http.StatusNoContent)
		if n := count(t, w.db, `SELECT count(*) FROM rate_limit_hits`); n != 0 {
			t.Errorf("%d hits left", n)
		}
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": "reset@example.org"}).Want(t, http.StatusAccepted)
	})
}

func TestChangeContact(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	p := newPerson("1986-11-23")
	ea := w.recordIn(t, p, w.a, w.hq, w.cara)
	eb := w.recordIn(t, p, w.b, w.hqB, w.bea)
	session := func() *apitest.Client {
		t.Helper()
		w.anon.Clock.Add(2 * time.Hour) // past the rate limits; a new session
		return w.anon.AsPatient(t, ea.Link, p.dob)
	}
	session()
	acc := accountOf(t, w, ea.PatientID)
	contactsOf := func(patient uuid.UUID) (string, string) {
		t.Helper()
		var m, e string
		if err := w.db.QueryRow(t.Context(), `SELECT mobile, email FROM patients WHERE id = $1`, patient).Scan(&m, &e); err != nil {
			t.Fatal(err)
		}
		return m, e
	}

	t.Run("happy path (email): a link to the new address; once it's opened, the account and every trust's record change", func(t *testing.T) {
		b := session()
		const next = "sam.new@example.org"
		r := b.Do("POST", "/api/p/contact", map[string]string{"contact": "Sam.New@Example.org"})
		r.Want(t, http.StatusAccepted)
		if got := readAccount(t, w, acc); got.Email != p.email {
			t.Fatalf("changed before it was confirmed: %+v", got)
		}
		token := tokenIn(t, w.anon.Messages, verifyLink, next)
		before := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'contact.update'`)
		c := w.anon.NewBrowser(t).Do("POST", "/api/p/contact/"+token+"/confirm", nil) // on another device: no session needed
		c.Want(t, http.StatusOK)
		var got struct{ Channel string }
		c.Decode(t, &got)
		if got.Channel != "email" {
			t.Errorf("channel %q", got.Channel)
		}
		if a := readAccount(t, w, acc); a.Email != next || a.Mobile != p.mobile {
			t.Errorf("account %+v", a)
		}
		for _, e := range []apitest.Episode{ea, eb} {
			if m, em := contactsOf(e.PatientID); em != next || m != p.mobile {
				t.Errorf("record %s: %s %s", e.PatientID, m, em)
			}
		}
		// The old address is told, in case it wasn't them.
		old := w.anon.Messages.To(p.email)
		if len(old) == 0 || !strings.Contains(old[len(old)-1].Body, "changed") {
			t.Errorf("old address got %+v", old)
		}
		// One audit row per trust whose record changed.
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'contact.update'`) - before; n != 2 {
			t.Errorf("%d audit rows", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'contact.update' AND org_id = $1 AND $2 = ANY(patient_ids)`, w.b.Org, eb.PatientID); n != 1 {
			t.Errorf("trust B's row: %d", n)
		}
		// Signing in now uses the new address; the old one gets nothing.
		w.anon.Clock.Add(2 * time.Hour)
		readHome(t, w.signIn(t, next, next, p.dob))
		sentBefore := len(w.anon.Messages.To(p.email))
		w.anon.Do("POST", "/api/p/signin", map[string]string{"contact": p.email}).Want(t, http.StatusAccepted)
		if len(w.anon.Messages.To(p.email)) != sentBefore {
			t.Error("a sign-in link went to the old address")
		}
		// A verification link works once.
		refused(t, w.anon.Do("POST", "/api/p/contact/"+token+"/confirm", nil), http.StatusNotFound)
		p.email = next
	})
	t.Run("happy path (mobile): verified by text; the old number is told", func(t *testing.T) {
		b := session()
		const next = "+447700900777"
		b.Do("POST", "/api/p/contact", map[string]string{"contact": "07700 900777"}).Want(t, http.StatusAccepted)
		token := tokenIn(t, w.anon.Messages, verifyLink, next)
		r := w.anon.Do("POST", "/api/p/contact/"+token+"/confirm", nil)
		r.Want(t, http.StatusOK)
		if a := readAccount(t, w, acc); a.Mobile != next {
			t.Errorf("account %+v", a)
		}
		if m, _ := contactsOf(eb.PatientID); m != next {
			t.Errorf("trust B's record: %s", m)
		}
		if old := w.anon.Messages.To(p.mobile); len(old) == 0 || !strings.Contains(old[len(old)-1].Body, "changed") {
			t.Errorf("old number got %+v", old)
		}
		p.mobile = next
	})
	t.Run("adding a contact where there was none tells the existing one (S15 answers 7)", func(t *testing.T) {
		q := newPerson("1989-02-26")
		q.mobile = ""
		eq := w.recordIn(t, q, w.a, w.hq, w.cara)
		w.anon.Clock.Add(2 * time.Hour)
		b := w.anon.AsPatient(t, eq.Link, q.dob)
		const added = "+447700900888"
		b.Do("POST", "/api/p/contact", map[string]string{"contact": added, "kind": "mobile"}).Want(t, http.StatusAccepted)
		w.anon.Do("POST", "/api/p/contact/"+tokenIn(t, w.anon.Messages, verifyLink, added)+"/confirm", nil).Want(t, http.StatusOK)
		told := w.anon.Messages.To(q.email)
		if len(told) == 0 || !strings.Contains(told[len(told)-1].Body, "added") {
			t.Errorf("the email got %+v", told)
		}
	})
	t.Run("each box takes only its own kind of contact (S15 answers 7)", func(t *testing.T) {
		b := session()
		before := len(w.anon.Messages.Sent())
		r := b.Do("POST", "/api/p/contact", map[string]string{"contact": "07700 900555", "kind": "email"})
		refused(t, r, http.StatusBadRequest)
		if !strings.Contains(r.Message(), "email address") {
			t.Errorf("message %q", r.Message())
		}
		r = b.Do("POST", "/api/p/contact", map[string]string{"contact": "someone@example.org", "kind": "mobile"})
		refused(t, r, http.StatusBadRequest)
		if !strings.Contains(r.Message(), "mobile number") {
			t.Errorf("message %q", r.Message())
		}
		refused(t, b.Do("POST", "/api/p/contact", map[string]string{"contact": "x@example.org", "kind": "fax"}), http.StatusBadRequest)
		if len(w.anon.Messages.Sent()) != before {
			t.Error("sent")
		}
	})
	t.Run("a replaced verification link says it has expired", func(t *testing.T) {
		b := session()
		b.Do("POST", "/api/p/contact", map[string]string{"contact": "one@example.org"}).Want(t, http.StatusAccepted)
		older := tokenIn(t, w.anon.Messages, verifyLink, "one@example.org")
		b.Do("POST", "/api/p/contact", map[string]string{"contact": "two@example.org"}).Want(t, http.StatusAccepted)
		r := w.anon.Do("POST", "/api/p/contact/"+older+"/confirm", nil)
		refused(t, r, http.StatusNotFound)
		if !strings.Contains(string(r.Body), `"code":"expired"`) {
			t.Errorf("got %s", r)
		}
	})
	t.Run("no sign-in: 401 and nothing sent", func(t *testing.T) {
		before := len(w.anon.Messages.Sent())
		refused(t, w.anon.Do("POST", "/api/p/contact", map[string]string{"contact": "x@example.org"}), http.StatusUnauthorized)
		if len(w.anon.Messages.Sent()) != before {
			t.Error("sent")
		}
	})
	t.Run("wrong role: a clinician's sign-in is not a patient session", func(t *testing.T) {
		refused(t, w.clinician.Do("POST", "/api/p/contact", map[string]string{"contact": "x@example.org"}), http.StatusUnauthorized)
	})
	t.Run("invalid input: not a contact, the same one, or a mobile in another country", func(t *testing.T) {
		b := session()
		before := len(w.anon.Messages.Sent())
		for _, c := range []string{"", "hello", p.email, strings.ToUpper(p.email), "+1 202 555 0143"} {
			refused(t, b.Do("POST", "/api/p/contact", map[string]string{"contact": c}), http.StatusBadRequest)
		}
		refused(t, b.Do("POST", "/api/p/contact", `{"contact":`), http.StatusBadRequest)
		if len(w.anon.Messages.Sent()) != before {
			t.Error("sent")
		}
		refused(t, w.anon.Do("POST", "/api/p/contact/short/confirm", nil), http.StatusNotFound)
		refused(t, w.anon.Do("POST", "/api/p/contact/"+strings.Repeat("A", 43)+"/confirm", nil), http.StatusNotFound)
		// A sign-in or invite link isn't a verification link.
		refused(t, w.anon.Do("POST", "/api/p/contact/"+ea.Link+"/confirm", nil), http.StatusNotFound)
	})
	t.Run("a verification link expires after 24 hours, and a newer request replaces it", func(t *testing.T) {
		b := session()
		b.Do("POST", "/api/p/contact", map[string]string{"contact": "late@example.org"}).Want(t, http.StatusAccepted)
		late := tokenIn(t, w.anon.Messages, verifyLink, "late@example.org")
		w.anon.Clock.Add(25 * time.Hour)
		refused(t, w.anon.Do("POST", "/api/p/contact/"+late+"/confirm", nil), http.StatusNotFound)

		b = session()
		b.Do("POST", "/api/p/contact", map[string]string{"contact": "first@example.org"}).Want(t, http.StatusAccepted)
		older := tokenIn(t, w.anon.Messages, verifyLink, "first@example.org")
		b.Do("POST", "/api/p/contact", map[string]string{"contact": "second@example.org"}).Want(t, http.StatusAccepted)
		refused(t, w.anon.Do("POST", "/api/p/contact/"+older+"/confirm", nil), http.StatusNotFound)
		if a := readAccount(t, w, acc); a.Email != p.email {
			t.Errorf("account %+v", a)
		}
	})
	t.Run("rate limited like sign-in links: 3 per contact in 15 minutes", func(t *testing.T) {
		b := session()
		for range 3 {
			b.Do("POST", "/api/p/contact", map[string]string{"contact": "busy@example.org"}).Want(t, http.StatusAccepted)
		}
		refused(t, b.Do("POST", "/api/p/contact", map[string]string{"contact": "busy@example.org"}), http.StatusTooManyRequests)
	})
	t.Run("a contact another account with the same date of birth already uses is refused, nothing changed", func(t *testing.T) {
		twin := newPerson(p.dob)
		twin.last = p.last // the same name and date of birth, other contacts: another account
		te := w.recordIn(t, twin, w.a, w.hq, w.cara)
		w.anon.AsPatient(t, te.Link, twin.dob)
		b := session()
		b.Do("POST", "/api/p/contact", map[string]string{"contact": twin.email}).Want(t, http.StatusAccepted)
		token := tokenIn(t, w.anon.Messages, verifyLink, twin.email)
		refused(t, w.anon.Do("POST", "/api/p/contact/"+token+"/confirm", nil), http.StatusConflict)
		if a := readAccount(t, w, acc); a.Email != p.email {
			t.Errorf("account %+v", a)
		}
	})
}
