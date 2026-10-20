//go:build integration

package httpapi_test

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
)

// Patient invites (S14; PAT-03, PAT-05, PAT-11). The texts and emails go to the recording fake;
// the links in them are opened through the patient's routes.

var inviteLink = regexp.MustCompile(`http://localhost:3000/i/([A-Za-z0-9_-]{43})`)

// linkIn is the token in a message's link, failing the test if there's none.
func linkIn(t *testing.T, m messaging.Message) string {
	t.Helper()
	got := inviteLink.FindStringSubmatch(m.Body)
	if got == nil {
		t.Fatalf("no link in %q", m.Body)
	}
	return got[1]
}

// newPatientEpisode creates an episode on the New episode page's route for a typed patient, and
// returns its id. contacts are the patient's mobile and email ("" for none).
func newPatientEpisode(t *testing.T, w *clinic, last, mobile, email string) uuid.UUID {
	t.Helper()
	r := w.clinician.Do("POST", at(w.a, "/episodes"), map[string]any{
		"versionId": w.hq.VersionID, "procedure": "Knee",
		"patient": map[string]string{"firstName": "Sam", "lastName": last, "dateOfBirth": "1980-05-17", "sex": "female",
			"mobile": mobile, "email": email},
	})
	r.Want(t, http.StatusCreated)
	var got struct{ ID uuid.UUID }
	r.Decode(t, &got)
	return got.ID
}

func hospitalName(t *testing.T, w *clinic, h apitest.Hospital) string {
	t.Helper()
	var name string
	if err := w.db.QueryRow(t.Context(), `SELECT name FROM hospitals WHERE id = $1`, h.ID).Scan(&name); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestCreateEpisodeSendsTheInvite(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	msgs := w.anon.Messages
	hospital := hospitalName(t, w, w.a)

	t.Run("happy path: a text and an email, each with the link; only its hash is stored", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Invited", "07700 900123", "Sam.Invited@Example.org")
		sms, email := msgs.To("+447700900123"), msgs.To("sam.invited@example.org")
		if len(sms) != 1 || len(email) != 1 || sms[0].Channel != "sms" || email[0].Channel != "email" {
			t.Fatalf("sent %+v", msgs.Sent())
		}
		token := linkIn(t, sms[0])
		// PAT-03's wording, with the hospital's name from its record.
		if want := hospital + ": please complete your pre-op health questionnaire: http://localhost:3000/i/" + token; sms[0].Body != want {
			t.Errorf("sms %q, want %q", sms[0].Body, want)
		}
		if linkIn(t, email[0]) != token || !strings.Contains(email[0].Body, "Hello Sam") || email[0].Subject != hospital+": your health questionnaire" {
			t.Errorf("email %+v", email[0])
		}
		hash := sha256.Sum256([]byte(token))
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1 AND kind = 'invite'
			AND token_hash = $2 AND revoked_at IS NULL AND expires_at IS NULL AND created_by IS NOT NULL`, id, hash[:]); n != 1 {
			t.Errorf("links for the token's hash: %d", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE position($1::bytea IN token_hash) > 0`, []byte(token)); n != 0 {
			t.Error("the token itself is stored")
		}
		ev := readEvents(t, w, id)
		if len(ev) != 2 || ev[0].Text != "Episode created" || ev[1].Text != "Invite sent by text message and email." || ev[1].User == nil {
			t.Errorf("events %+v", ev)
		}
		// The link opens the episode with the patient's date of birth.
		w.anon.AsPatient(t, token, "1980-05-17").Do("GET", "/api/p/episodes/"+id.String(), nil).Want(t, http.StatusOK)
	})
	t.Run("the hospital's own template is used", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET settings = '{"templates.inviteSms": "{{hospital}} for {{firstName}}: {{link}}"}' WHERE id = $1`, w.a.ID); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `UPDATE hospitals SET settings = '{}' WHERE id = $1`, w.a.ID)
		newPatientEpisode(t, w, "Templated", "07700 900124", "")
		sms := msgs.To("+447700900124")
		if len(sms) != 1 || !strings.HasPrefix(sms[0].Body, hospital+" for Sam: http://localhost:3000/i/") {
			t.Errorf("sms %+v", sms)
		}
	})
	t.Run("only an email when there's no mobile", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Emailed", "", "emailed@example.org")
		if len(msgs.To("emailed@example.org")) != 1 {
			t.Errorf("sent %+v", msgs.Sent())
		}
		if ev := readEvents(t, w, id); ev[len(ev)-1].Text != "Invite sent by email." {
			t.Errorf("events %+v", ev)
		}
	})
	t.Run("PAT-11: no text to a mobile outside the deployment's country; the email still goes", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Abroad", "+1 202 555 0123", "abroad@example.org")
		if len(msgs.To("+12025550123")) != 0 || len(msgs.To("abroad@example.org")) != 1 {
			t.Errorf("sent %+v", msgs.Sent())
		}
		if ev := readEvents(t, w, id); ev[len(ev)-1].Text != "Invite sent by email. No text message: texts go only to mobile numbers in this country." {
			t.Errorf("events %+v", ev)
		}
	})
	t.Run("no mobile or email: nothing sent, no link, and the notes say so", func(t *testing.T) {
		before := len(msgs.Sent())
		id := newPatientEpisode(t, w, "Unreachable", "", "")
		if len(msgs.Sent()) != before || count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1`, id) != 0 {
			t.Error("an invite was made")
		}
		if ev := readEvents(t, w, id); ev[len(ev)-1].Text != "No invite sent. The patient has no mobile or email." {
			t.Errorf("events %+v", ev)
		}
	})
	t.Run("a message that can't be sent doesn't stop the episode; the notes say so", func(t *testing.T) {
		msgs.FailWith(errors.New("provider down"))
		defer msgs.FailWith(nil)
		id := newPatientEpisode(t, w, "Failed", "07700 900125", "failed@example.org")
		if e := readEpisode(t, w, id); e.Status != "hq_not_complete" {
			t.Errorf("episode %+v", e)
		}
		if ev := readEvents(t, w, id); ev[len(ev)-1].Text != "No invite sent. The text message couldn't be sent. The email couldn't be sent." {
			t.Errorf("events %+v", ev)
		}
		// Nothing went, so the page mustn't say "Last sent": no working link (S14 verification 1).
		var got struct{ Invite *struct{ SentAt string } }
		r := w.clinician.Do("GET", at(w.a, "/episodes/"+id.String()), nil)
		r.Decode(t, &got)
		if got.Invite != nil {
			t.Errorf("invite %s, want null (Send invite offered)", r.Body)
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1 AND revoked_at IS NULL`, id); n != 0 {
			t.Errorf("%d working links, though no message went", n)
		}
	})
}

func TestResendInvite(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	msgs := w.anon.Messages
	id := newPatientEpisode(t, w, "Resent", "07700 900321", "resent@example.org")
	first := linkIn(t, msgs.To("+447700900321")[0])
	path := at(w.a, "/episodes/"+id.String()+"/resend-invite")

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("POST", path, nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician at the trust's other hospital gets 404", func(t *testing.T) {
		refused(t, w.site2.Do("POST", at(w.a2, "/episodes/"+id.String()+"/resend-invite"), nil), http.StatusNotFound)
		refused(t, w.site2.Do("POST", path, nil), http.StatusNotFound)
	})
	t.Run("another trust: 404", func(t *testing.T) {
		refused(t, w.otherClin.Do("POST", path, nil), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", at(w.b, "/episodes/"+id.String()+"/resend-invite"), nil), http.StatusNotFound)
	})
	t.Run("invalid input: no such episode", func(t *testing.T) {
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/abc/resend-invite"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/"+uuid.NewString()+"/resend-invite"), nil), http.StatusNotFound)
	})
	if len(msgs.Sent()) != 2 {
		t.Fatalf("refusals sent messages: %+v", msgs.Sent())
	}
	t.Run("happy path: a new link by both; the old one stops working", func(t *testing.T) {
		r := w.clinician.Do("POST", path, nil)
		r.Want(t, http.StatusOK)
		if !strings.Contains(r.Message()+string(r.Body), "Invite sent again by text message and email.") {
			t.Errorf("got %s", r)
		}
		sms, email := msgs.To("+447700900321"), msgs.To("resent@example.org")
		if len(sms) != 2 || len(email) != 2 {
			t.Fatalf("sent %+v", msgs.Sent())
		}
		second := linkIn(t, sms[1])
		if second == first || linkIn(t, email[1]) != second {
			t.Error("the resend didn't send a new link to both")
		}
		refused(t, w.anon.Do("POST", "/api/p/links/"+first+"/continue", nil), http.StatusNotFound)
		w.anon.AsPatient(t, second, "1980-05-17")
		if ev := readEvents(t, w, id); ev[len(ev)-1].Text != "Invite sent again by text message and email." || ev[len(ev)-1].User == nil {
			t.Errorf("events %+v", ev)
		}
	})
	t.Run("a locked link: the resend gives the patient a new one that works (PAT-05)", func(t *testing.T) {
		locked := linkIn(t, msgs.To("+447700900321")[1])
		for range 5 {
			w.anon.Do("POST", "/api/p/links/"+locked+"/dob", dob("1999-01-01"))
		}
		refused(t, w.anon.Do("POST", "/api/p/links/"+locked+"/continue", nil), http.StatusForbidden)
		var e struct{ Invite struct{ LockedAt *string } }
		r := w.clinician.Do("GET", at(w.a, "/episodes/"+id.String()), nil)
		r.Decode(t, &e)
		if e.Invite.LockedAt == nil {
			t.Errorf("staff don't see the lock: %s", r.Body)
		}
		w.clinician.Do("POST", path, nil).Want(t, http.StatusOK)
		fresh := linkIn(t, msgs.To("+447700900321")[2])
		w.anon.AsPatient(t, fresh, "1980-05-17")
		r = w.clinician.Do("GET", at(w.a, "/episodes/"+id.String()), nil)
		e.Invite.LockedAt = nil
		r.Decode(t, &e)
		if e.Invite.LockedAt != nil {
			t.Errorf("still shown as locked: %s", r.Body)
		}
	})
	t.Run("a resend whose messages all fail keeps the old link, its session and Last sent (verification 1)", func(t *testing.T) {
		current := linkIn(t, msgs.To("+447700900321")[len(msgs.To("+447700900321"))-1])
		session := w.anon.AsPatient(t, current, "1980-05-17")
		before := w.clinician.Do("GET", at(w.a, "/episodes/"+id.String()), nil)
		var was struct{ Invite struct{ SentAt string } }
		before.Decode(t, &was)
		msgs.FailWith(errors.New("provider down"))
		r := w.clinician.Do("POST", path, nil)
		msgs.FailWith(nil)
		refused(t, r, http.StatusBadGateway)
		w.anon.Do("POST", "/api/p/links/"+current+"/continue", nil).Want(t, http.StatusOK)
		session.Do("GET", "/api/p/episodes/"+id.String(), nil).Want(t, http.StatusOK)
		var now struct{ Invite struct{ SentAt string } }
		w.clinician.Do("GET", at(w.a, "/episodes/"+id.String()), nil).Decode(t, &now)
		if now.Invite.SentAt != was.Invite.SentAt {
			t.Errorf("Last sent moved from %s to %s though nothing went", was.Invite.SentAt, now.Invite.SentAt)
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1 AND revoked_at IS NULL`, id); n != 1 {
			t.Errorf("working links = %d, want the old one only", n)
		}
		if ev := readEvents(t, w, id); ev[len(ev)-1].Text != "No invite sent. The text message couldn't be sent. The email couldn't be sent." {
			t.Errorf("events %+v", ev)
		}
	})
	t.Run("the resend waits for the episode's lock, so two at once can't leave two working links", func(t *testing.T) {
		// NO KEY UPDATE: it blocks the resend's FOR UPDATE on the episode, but not the foreign-key
		// checks of the new link's insert, so only the episode lock makes the request wait.
		r := meanwhile(t, w.db, `SELECT id FROM episodes WHERE id = $1 FOR NO KEY UPDATE`, []any{id}, func() *apitest.Resp {
			return w.clinician.Do("POST", path, nil)
		})
		r.Want(t, http.StatusOK)
		done := make(chan *apitest.Resp, 4)
		for range 4 {
			go func() { done <- w.clinician.Do("POST", path, nil) }()
		}
		for range 4 {
			(<-done).Want(t, http.StatusOK)
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1 AND revoked_at IS NULL`, id); n != 1 {
			t.Fatalf("working links = %d, want 1", n)
		}
		sms := msgs.To("+447700900321")
		newest := 0
		for _, m := range sms[len(sms)-4:] { // the one link left works; the other three don't
			if w.anon.Do("POST", "/api/p/links/"+linkIn(t, m)+"/continue", nil).Code == http.StatusOK {
				newest++
			}
		}
		if newest != 1 {
			t.Errorf("%d of the last 4 links work, want 1", newest)
		}
	})
	t.Run("no mobile or email: 409, nothing sent", func(t *testing.T) {
		none := newPatientEpisode(t, w, "Nobody", "", "")
		before := len(msgs.Sent())
		r := w.clinician.Do("POST", at(w.a, "/episodes/"+none.String()+"/resend-invite"), nil)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "no mobile or email") || len(msgs.Sent()) != before {
			t.Errorf("got %s, sent %d", r, len(msgs.Sent())-before)
		}
	})
	t.Run("once the answers are sent: 409, nothing sent", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `WITH f AS (UPDATE episode_forms SET status = 'submitted', submitted_at = now() WHERE episode_id = $1 AND kind = 'hq')
			UPDATE episodes SET status = 'ready_for_review' WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		before := len(msgs.Sent())
		refused(t, w.clinician.Do("POST", path, nil), http.StatusConflict)
		if len(msgs.Sent()) != before {
			t.Error("sent after submit")
		}
	})
}

func TestCapturedMessages(t *testing.T) {
	t.Parallel()
	c, fx := apitest.NewWith(t, apitest.Options{Capture: true}) // MESSAGING=capture, as make dev runs
	h := fx.Hospital(t)
	cara := fx.User(t, h, "clinician")
	hq := fx.PublishedHQ(t, h, fx.TrustUser(t, h.Org, "super_clinician"), "workflow.json")
	staff := c.LoginAs(t, cara)
	staff.Do("POST", h.Path("/episodes"), map[string]any{
		"versionId": hq.VersionID,
		"patient": map[string]string{"firstName": "Sam", "lastName": "Captured", "dateOfBirth": "1980-05-17", "sex": "female",
			"mobile": "07700 900456", "email": "captured@example.org"},
	}).Want(t, http.StatusCreated)

	type message struct{ Channel, Recipient, Subject, Body, SentAt string }
	list := func(to string) []message {
		t.Helper()
		r := c.Do("GET", "/api/qa/messages?to="+to, nil)
		r.Want(t, http.StatusOK)
		var got []message
		r.Decode(t, &got)
		return got
	}
	t.Run("stored, not sent, and readable by recipient", func(t *testing.T) {
		if n := count(t, c.DB, `SELECT count(*) FROM sent_messages`); n != 2 {
			t.Errorf("sent_messages: %d", n)
		}
		sms := list("%2B447700900456")
		if len(sms) != 1 || sms[0].Channel != "sms" || !inviteLink.MatchString(sms[0].Body) || sms[0].SentAt == "" {
			t.Errorf("sms %+v", sms)
		}
		if email := list("captured@example.org"); len(email) != 1 || email[0].Channel != "email" || email[0].Subject == "" {
			t.Errorf("email %+v", email)
		}
		if all := list(""); len(all) != 2 || all[0].Channel != "email" { // newest first
			t.Errorf("all %+v", all)
		}
		if none := list("nobody@example.org"); len(none) != 0 {
			t.Errorf("none %+v", none)
		}
	})
}

// GET /api/qa/client (capture mode, like the other QA routes; no sign-in): the address the rate
// limits count this caller by, and the X-Forwarded-For the API got, so a tester can check on
// QA that it's the browser's own address and not a proxy's (S06, FORWARDED_HOPS).
func TestQAClientAddress(t *testing.T) {
	t.Parallel()
	c, _ := apitest.NewWith(t, apitest.Options{Capture: true})
	r := c.Do("GET", "/api/qa/client", nil)
	r.Want(t, http.StatusOK)
	var got struct{ Address, ForwardedFor string }
	r.Decode(t, &got)
	if got.Address != "127.0.0.1" || got.ForwardedFor != "" {
		t.Errorf("got %+v; want the test client's own address and no header", got)
	}
	if cc := r.Header("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control %q", cc)
	}
}
