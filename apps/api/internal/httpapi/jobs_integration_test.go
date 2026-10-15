//go:build integration

package httpapi_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Scheduled jobs (S16): each episode's follow-up sends reminders on the hospital's days after the
// first invite (2 and 5 by default) and marks it "not responding" on day 7, run here through the
// capture-mode route testers use, POST /api/qa/jobs/run {asOf}. The messages go to the
// recording fake. The QA routes have no sign-in (like /api/qa/messages), so of the five usual
// cases only the happy path and invalid input apply; they exist only with MESSAGING=capture,
// the only mode until S17 (S17's CDK check refuses capture outside QA).

type jobsSummary struct {
	AsOf                                     time.Time
	Jobs, Reminders, NotResponding, Failed int
}

func runJobs(t *testing.T, c *apitest.Client, asOf time.Time) jobsSummary {
	t.Helper()
	r := c.Do("POST", "/api/qa/jobs/run", map[string]string{"asOf": asOf.Format(time.RFC3339Nano)})
	r.Want(t, http.StatusOK)
	var s jobsSummary
	r.Decode(t, &s)
	if !s.AsOf.Equal(asOf) {
		t.Errorf("asOf %v, want %v", s.AsOf, asOf)
	}
	return s
}

// followup is the episode's follow-up state.
type followup struct {
	InvitedAt, NotRespondingAt *time.Time
	RemindersSent              int
}

func readFollowup(t *testing.T, w *clinic, id uuid.UUID) followup {
	t.Helper()
	var f followup
	if err := w.db.QueryRow(t.Context(), `SELECT invited_at, reminders_sent, not_responding_at FROM episodes WHERE id = $1`, id).
		Scan(&f.InvitedAt, &f.RemindersSent, &f.NotRespondingAt); err != nil {
		t.Fatal(err)
	}
	return f
}

// dayOf is day n after the episode's first invite, in the hospital's zone (the deployment's
// default here), as the reminders count it.
func dayOf(t *testing.T, w *clinic, id uuid.UUID, n int) time.Time {
	t.Helper()
	return invitedAt(t, w, id).In(mustZone(t, "Europe/London")).AddDate(0, 0, n)
}

// invitedAt is when the episode's follow-up started, failing the test if it hasn't.
func invitedAt(t *testing.T, w *clinic, id uuid.UUID) time.Time {
	t.Helper()
	f := readFollowup(t, w, id)
	if f.InvitedAt == nil {
		t.Fatal("no invited_at")
	}
	return *f.InvitedAt
}

func lastEvent(t *testing.T, w *clinic, id uuid.UUID) eventRow {
	t.Helper()
	ev := readEvents(t, w, id)
	return ev[len(ev)-1]
}

func TestFollowupReminders(t *testing.T) {
	w := newClinic(t)
	msgs := w.anon.Messages
	hospital := hospitalName(t, w, w.a)
	id := newPatientEpisode(t, w, "Reminded", "07700 900301", "reminded@example.org")
	mobile, email := "+447700900301", "reminded@example.org"
	invited := invitedAt(t, w, id)
	firstLink := linkIn(t, msgs.To(mobile)[0])
	day := func(n int) time.Time { return dayOf(t, w, id, n) }

	t.Run("the first invite starts the follow-up", func(t *testing.T) {
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1 AND kind = 'followup' AND done_at IS NULL AND run_at = $2`, id, invited); n != 1 {
			t.Errorf("follow-up jobs: %d", n)
		}
		if time.Since(invited) > time.Minute || time.Since(invited) < 0 {
			t.Errorf("invited_at %v", invited)
		}
	})
	t.Run("before day 2: nothing", func(t *testing.T) {
		s := runJobs(t, w.anon, day(2).Add(-time.Minute))
		if s.Jobs != 1 || s.Reminders != 0 || len(msgs.To(mobile)) != 1 {
			t.Errorf("summary %+v, sent %+v", s, msgs.To(mobile))
		}
	})
	var reminderLink string
	t.Run("day 2: reminder 1 by text and email, with a new link; the invite's link still works", func(t *testing.T) {
		s := runJobs(t, w.anon, day(2))
		if s.Jobs != 1 || s.Reminders != 1 || s.NotResponding != 0 || s.Failed != 0 {
			t.Errorf("summary %+v", s)
		}
		sms, mail := msgs.To(mobile), msgs.To(email)
		if len(sms) != 2 || len(mail) != 2 {
			t.Fatalf("sent %+v", msgs.Sent())
		}
		reminderLink = linkIn(t, sms[1])
		if want := "Hello Sam, a reminder from " + hospital + ": please fill in your health questionnaire: http://localhost:3000/i/" + reminderLink; sms[1].Body != want {
			t.Errorf("sms %q, want %q", sms[1].Body, want)
		}
		if linkIn(t, mail[1]) != reminderLink || !strings.Contains(mail[1].Body, "This is a reminder from "+hospital) ||
			mail[1].Subject != hospital+": a reminder about your health questionnaire" {
			t.Errorf("email %+v", mail[1])
		}
		if reminderLink == firstLink {
			t.Error("the reminder reused the invite's token")
		}
		if f := readFollowup(t, w, id); f.RemindersSent != 1 || f.NotRespondingAt != nil {
			t.Errorf("follow-up %+v", f)
		}
		ev := lastEvent(t, w, id)
		if ev.Text != "Day 2 reminder sent by text message and email." || ev.User != nil {
			t.Errorf("event %+v", ev)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_events WHERE episode_id = $1 AND automatic`, id); n != 1 {
			t.Errorf("automatic events: %d", n)
		}
		// Both links open the HQ: the patient may use the invite they already have.
		w.anon.AsPatient(t, firstLink, "1980-05-17").Do("GET", "/api/p/episodes/"+id.String(), nil).Want(t, http.StatusOK)
		w.anon.AsPatient(t, reminderLink, "1980-05-17").Do("GET", "/api/p/episodes/"+id.String(), nil).Want(t, http.StatusOK)
	})
	t.Run("the same time again: nothing is sent twice", func(t *testing.T) {
		s := runJobs(t, w.anon, day(2))
		if s.Jobs != 0 || len(msgs.To(mobile)) != 2 {
			t.Errorf("summary %+v, sent %d", s, len(msgs.To(mobile)))
		}
		s = runJobs(t, w.anon, day(4)) // or later, before day 5
		if s.Jobs != 0 || len(msgs.To(mobile)) != 2 {
			t.Errorf("summary %+v, sent %d", s, len(msgs.To(mobile)))
		}
	})
	t.Run("the staff page shows the follow-up and who wrote the note", func(t *testing.T) {
		r := w.clinician.Do("GET", at(w.a, "/episodes/"+id.String()), nil)
		r.Want(t, http.StatusOK)
		var got struct {
			Episode struct {
				InvitedAt       *time.Time
				RemindersSent   int
				NotRespondingAt *time.Time
			}
			Events []struct {
				Text      string
				UserName  *string
				Automatic bool
			}
		}
		r.Decode(t, &got)
		if got.Episode.InvitedAt == nil || got.Episode.RemindersSent != 1 || got.Episode.NotRespondingAt != nil {
			t.Errorf("episode %+v", got.Episode)
		}
		last := got.Events[len(got.Events)-1]
		if !last.Automatic || last.UserName != nil || got.Events[0].Automatic {
			t.Errorf("events %+v", got.Events)
		}
	})
	t.Run("day 5: reminder 2", func(t *testing.T) {
		s := runJobs(t, w.anon, day(5))
		if s.Reminders != 1 || len(msgs.To(mobile)) != 3 || len(msgs.To(email)) != 3 {
			t.Errorf("summary %+v, sent %+v", s, msgs.Sent())
		}
		if ev := lastEvent(t, w, id); ev.Text != "Day 5 reminder sent by text message and email." {
			t.Errorf("event %+v", ev)
		}
	})
	t.Run("day 7: not responding, once; no message; the follow-up is finished", func(t *testing.T) {
		s := runJobs(t, w.anon, day(7))
		if s.Jobs != 1 || s.Reminders != 0 || s.NotResponding != 1 || len(msgs.To(mobile)) != 3 {
			t.Errorf("summary %+v, sent %d", s, len(msgs.To(mobile)))
		}
		f := readFollowup(t, w, id)
		if f.NotRespondingAt == nil || !f.NotRespondingAt.Equal(day(7)) || f.RemindersSent != 2 {
			t.Errorf("follow-up %+v", f)
		}
		if ev := lastEvent(t, w, id); ev.Text != "Not responding: the HQ hasn't been sent 7 days after the invite." {
			t.Errorf("event %+v", ev)
		}
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1 AND done_at IS NOT NULL`, id); n != 1 {
			t.Errorf("finished jobs: %d", n)
		}
		if s := runJobs(t, w.anon, day(30)); s.Jobs != 0 {
			t.Errorf("after it finished: %+v", s)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_events WHERE episode_id = $1 AND text LIKE 'Not responding%'`, id); n != 1 {
			t.Errorf("not responding notes: %d", n)
		}
	})
}

func TestFollowupStops(t *testing.T) {
	w := newClinic(t)
	msgs := w.anon.Messages

	t.Run("submitting stops reminders and not responding", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Submits", "07700 900311", "")
		day := func(n int) time.Time { return dayOf(t, w, id, n) }
		token := linkIn(t, msgs.To("+447700900311")[0])
		runJobs(t, w.anon, day(2))
		if len(msgs.To("+447700900311")) != 2 {
			t.Fatalf("no day 2 reminder: %+v", msgs.Sent())
		}
		p := w.anon.AsPatient(t, token, "1980-05-17")
		p.Do("POST", "/api/p/episodes/"+id.String()+"/submit", nil).Want(t, http.StatusNoContent)
		s := runJobs(t, w.anon, day(7))
		if s.Jobs != 1 || s.Reminders != 0 || s.NotResponding != 0 || len(msgs.To("+447700900311")) != 2 {
			t.Errorf("summary %+v, sent %d", s, len(msgs.To("+447700900311")))
		}
		if f := readFollowup(t, w, id); f.NotRespondingAt != nil || f.RemindersSent != 1 {
			t.Errorf("follow-up %+v", f)
		}
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1 AND done_at IS NOT NULL`, id); n != 1 {
			t.Errorf("finished jobs: %d", n)
		}
	})
	t.Run("starting stops reminders, but not responding still comes on day 7", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Starts", "07700 900312", "")
		day := func(n int) time.Time { return dayOf(t, w, id, n) }
		token := linkIn(t, msgs.To("+447700900312")[0])
		p := w.anon.AsPatient(t, token, "1980-05-17")
		p.Do("PUT", "/api/p/episodes/"+id.String()+"/answers/"+w.hq.Chapters[0].String(),
			map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 0}).Want(t, http.StatusOK)
		if s := runJobs(t, w.anon, day(5)); s.Reminders != 0 || len(msgs.To("+447700900312")) != 1 {
			t.Errorf("summary %+v, sent %d", s, len(msgs.To("+447700900312")))
		}
		if s := runJobs(t, w.anon, day(7)); s.NotResponding != 1 {
			t.Errorf("summary %+v", s)
		}
		if f := readFollowup(t, w, id); f.NotRespondingAt == nil || f.RemindersSent != 0 {
			t.Errorf("follow-up %+v", f)
		}
	})
	t.Run("answers staff entered for the patient count as started", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Behalf", "07700 900313", "")
		w.clinician.Do("PUT", at(w.a, "/episodes/"+id.String()+"/patient-form/answers/"+w.hq.Chapters[0].String()),
			map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 0}).Want(t, http.StatusOK)
		if s := runJobs(t, w.anon, dayOf(t, w, id, 2)); s.Reminders != 0 || len(msgs.To("+447700900313")) != 1 {
			t.Errorf("summary %+v", s)
		}
	})
	t.Run("a resend doesn't restart the follow-up", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Resent", "07700 900315", "")
		invited := invitedAt(t, w, id)
		w.clinician.Do("POST", at(w.a, "/episodes/"+id.String()+"/resend-invite"), nil).Want(t, http.StatusOK)
		if got := invitedAt(t, w, id); !got.Equal(invited) {
			t.Errorf("invited_at moved from %v to %v", invited, got)
		}
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1`, id); n != 1 {
			t.Errorf("jobs: %d", n)
		}
	})
	t.Run("an episode whose invite never went has no follow-up", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Uncontactable", "", "")
		if f := readFollowup(t, w, id); f.InvitedAt != nil {
			t.Errorf("follow-up %+v", f)
		}
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1`, id); n != 0 {
			t.Errorf("jobs: %d", n)
		}
	})
	t.Run("a suspended hospital's patients get nothing until it's active again", func(t *testing.T) {
		id := newPatientEpisode(t, w, "Suspended", "07700 900314", "")
		day := func(n int) time.Time { return dayOf(t, w, id, n) }
		if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET status = 'suspended' WHERE id = $1`, w.a.ID); err != nil {
			t.Fatal(err)
		}
		runJobs(t, w.anon, day(2))
		if len(msgs.To("+447700900314")) != 1 || readFollowup(t, w, id).RemindersSent != 0 {
			t.Errorf("sent while suspended: %+v", msgs.To("+447700900314"))
		}
		if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET status = 'active' WHERE id = $1`, w.a.ID); err != nil {
			t.Fatal(err)
		}
		runJobs(t, w.anon, day(2).Add(time.Hour)) // it looks again an hour later
		if len(msgs.To("+447700900314")) != 2 {
			t.Errorf("not sent once active: %+v", msgs.To("+447700900314"))
		}
	})
}

func TestFollowupHospitalSettings(t *testing.T) {
	w := newClinic(t)
	msgs := w.anon.Messages
	// The hospital's own schedule and reminder text (ORG-04), in its own time zone.
	if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET timezone = 'Pacific/Auckland', settings =
		'{"reminders.days": [1, 3], "reminders.notRespondingDay": 4, "templates.reminderSms": "{{hospital}} reminds {{firstName}}: {{link}}"}'
		WHERE id = $1`, w.a.ID); err != nil {
		t.Fatal(err)
	}
	id := newPatientEpisode(t, w, "Scheduled", "07700 900321", "")
	day := func(n int) time.Time {
		return invitedAt(t, w, id).In(mustZone(t, "Pacific/Auckland")).AddDate(0, 0, n)
	}
	if s := runJobs(t, w.anon, day(1)); s.Reminders != 1 {
		t.Errorf("day 1: %+v", s)
	}
	sms := msgs.To("+447700900321")
	if len(sms) != 2 || !strings.HasPrefix(sms[1].Body, hospitalName(t, w, w.a)+" reminds Sam: http://localhost:3000/i/") {
		t.Errorf("sms %+v", sms)
	}
	if s := runJobs(t, w.anon, day(4)); s.Reminders != 1 || s.NotResponding != 1 {
		t.Errorf("day 4: %+v", s)
	}
	if ev := lastEvent(t, w, id); ev.Text != "Not responding: the HQ hasn't been sent 4 days after the invite." {
		t.Errorf("event %+v", ev)
	}
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestFollowupRunnersAtOnce(t *testing.T) {
	w := newClinic(t)
	msgs := w.anon.Messages
	var ids []uuid.UUID
	for i := range 5 {
		ids = append(ids, newPatientEpisode(t, w, "Concurrent"+string(rune('A'+i)), "07700 90033"+string(rune('0'+i)), ""))
	}
	asOf := dayOf(t, w, ids[len(ids)-1], 2).Add(time.Minute)
	before := len(msgs.Sent())
	var wg sync.WaitGroup
	sums := make([]jobsSummary, 4)
	for i := range sums {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sums[i] = runJobs(t, w.anon, asOf)
		}()
	}
	wg.Wait()
	total := 0
	for _, s := range sums {
		total += s.Reminders
	}
	if sent := len(msgs.Sent()) - before; sent != 5 || total != 5 {
		t.Errorf("4 runs at once sent %d reminders (summaries %+v), want one each for 5 episodes", sent, sums)
	}
	for _, id := range ids {
		if f := readFollowup(t, w, id); f.RemindersSent != 1 {
			t.Errorf("episode %s: %+v", id, f)
		}
	}
}

func TestQAJobsRoutes(t *testing.T) {
	t.Parallel()
	c, _ := apitest.New(t)

	t.Run("invalid input: 400 with a message", func(t *testing.T) {
		for _, body := range []any{nil, `{"asOf":`, map[string]string{}, map[string]string{"asOf": "tomorrow"},
			map[string]string{"asOf": "2026-10-15T10:00"}, map[string]int{"asOf": 5}} {
			refused(t, c.Do("POST", "/api/qa/jobs/run", body), http.StatusBadRequest)
		}
		for _, body := range []any{nil, `{"offsetMinutes":`, map[string]int{"offsetMinutes": -1},
			map[string]int{"offsetMinutes": 60*24*90 + 1}, map[string]string{"offsetMinutes": "60"}, map[string]any{}} {
			refused(t, c.Do("PUT", "/api/qa/jobs/clock", body), http.StatusBadRequest)
		}
	})
	t.Run("nothing due: an empty run", func(t *testing.T) {
		if s := runJobs(t, c, time.Now()); s.Jobs != 0 {
			t.Errorf("summary %+v", s)
		}
	})
	t.Run("the timer's clock: read, moved ahead, set back", func(t *testing.T) {
		type clock struct {
			OffsetMinutes            int
			Timer                    bool
			LastTickAt, LastTickAsOf *time.Time
		}
		get := func() clock {
			t.Helper()
			r := c.Do("GET", "/api/qa/jobs", nil)
			r.Want(t, http.StatusOK)
			var got clock
			r.Decode(t, &got)
			return got
		}
		if got := get(); got.OffsetMinutes != 0 || !got.Timer || got.LastTickAt != nil {
			t.Errorf("start %+v", got)
		}
		c.Do("PUT", "/api/qa/jobs/clock", map[string]int{"offsetMinutes": 2 * 24 * 60}).Want(t, http.StatusNoContent)
		if got := get(); got.OffsetMinutes != 2880 {
			t.Errorf("after moving %+v", got)
		}
		c.Do("PUT", "/api/qa/jobs/clock", map[string]int{"offsetMinutes": 0}).Want(t, http.StatusNoContent)
		if got := get(); got.OffsetMinutes != 0 {
			t.Errorf("after setting back %+v", got)
		}
	})
}

func TestQAJobsTimerOff(t *testing.T) {
	t.Parallel()
	c, _ := apitest.NewWith(t, apitest.Options{Env: map[string]string{"JOBS_TIMER": "false"}})
	r := c.Do("GET", "/api/qa/jobs", nil)
	r.Want(t, http.StatusOK)
	var got struct{ Timer bool }
	r.Decode(t, &got)
	if got.Timer {
		t.Error("JOBS_TIMER=false, but the page says the timer is on")
	}
}

// tryDob sends a date of birth on an invite link from a fresh browser, as the link's page does.
func tryDob(t *testing.T, c *apitest.Client, token, date string) *apitest.Resp {
	t.Helper()
	b := c.NewBrowser(t)
	return b.Do("POST", "/api/p/links/"+token+"/dob", map[string]string{"dateOfBirth": date})
}

// S14's lock, extended to reminders (S16 test report 1, X-1): five wrong dates of birth lock the
// episode, whichever of its links they were typed on, until staff resend; a locked episode gets
// no reminders.
func TestEpisodeDobLock(t *testing.T) {
	w := newClinic(t)
	msgs := w.anon.Messages
	const right, wrong = "1980-05-17", "1980-05-18"

	// lockedEpisode is an episode with its invite and day 2 reminder links, both working.
	lockedEpisode := func(t *testing.T, last, mobile string) (uuid.UUID, string, string) {
		t.Helper()
		id := newPatientEpisode(t, w, last, mobile, "")
		e164 := "+44" + strings.ReplaceAll(mobile, " ", "")[1:]
		runJobs(t, w.anon, dayOf(t, w, id, 2))
		sms := msgs.To(e164)
		if len(sms) != 2 {
			t.Fatalf("sent %+v", sms)
		}
		return id, linkIn(t, sms[0]), linkIn(t, sms[1])
	}
	staffInvite := func(t *testing.T, id uuid.UUID) (lockedAt *time.Time) {
		t.Helper()
		r := w.clinician.Do("GET", at(w.a, "/episodes/"+id.String()), nil)
		r.Want(t, http.StatusOK)
		var got struct{ Invite *struct{ LockedAt *time.Time } }
		r.Decode(t, &got)
		if got.Invite == nil {
			t.Fatal("no invite")
		}
		return got.Invite.LockedAt
	}

	t.Run("the tries are the episode's: wrong dates on two links lock both at the fifth", func(t *testing.T) {
		id, invite, reminder := lockedEpisode(t, "LockShared", "07700 900341")
		for i, want := range []string{"4 tries", "3 tries", "2 tries"} {
			r := tryDob(t, w.anon, invite, wrong)
			if r.Code != http.StatusUnauthorized || !strings.Contains(r.Message(), "You have "+want+" left") {
				t.Fatalf("try %d on the invite: %s", i+1, r)
			}
		}
		if r := tryDob(t, w.anon, reminder, wrong); r.Code != http.StatusUnauthorized || !strings.Contains(r.Message(), "You have 1 try left") {
			t.Fatalf("4th try, on the reminder's link: %s", r)
		}
		r := tryDob(t, w.anon, reminder, wrong)
		r.Want(t, http.StatusForbidden)
		// Every link of the episode is locked now: even the right date is refused on each.
		for _, token := range []string{invite, reminder} {
			if r := tryDob(t, w.anon, token, right); r.Code != http.StatusForbidden || !strings.Contains(string(r.Body), `"locked"`) {
				t.Errorf("right date on a locked link: %s", r)
			}
			if r := w.anon.NewBrowser(t).Do("POST", "/api/p/links/"+token+"/continue", nil); r.Code != http.StatusForbidden {
				t.Errorf("continue on a locked link: %s", r)
			}
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1 AND revoked_at IS NULL AND locked_at IS NULL`, id); n != 0 {
			t.Errorf("unlocked live links: %d", n)
		}
		if staffInvite(t, id) == nil {
			t.Error("staff aren't told it's locked")
		}
	})
	t.Run("a right date on any link starts the count again", func(t *testing.T) {
		_, invite, reminder := lockedEpisode(t, "LockReset", "07700 900342")
		for range 3 {
			tryDob(t, w.anon, invite, wrong).Want(t, http.StatusUnauthorized)
		}
		tryDob(t, w.anon, reminder, right).Want(t, http.StatusOK)
		if r := tryDob(t, w.anon, invite, wrong); !strings.Contains(r.Message(), "You have 4 tries left") {
			t.Errorf("after a right date: %s", r)
		}
	})
	t.Run("another episode's tries don't count", func(t *testing.T) {
		_, invite, _ := lockedEpisode(t, "LockOther", "07700 900343")
		_, other, _ := lockedEpisode(t, "LockOther2", "07700 900344")
		for range 4 {
			tryDob(t, w.anon, other, wrong).Want(t, http.StatusUnauthorized)
		}
		if r := tryDob(t, w.anon, invite, wrong); !strings.Contains(r.Message(), "You have 4 tries left") {
			t.Errorf("another episode's tries counted: %s", r)
		}
	})
	t.Run("the lock ends a session opened from the episode's other link", func(t *testing.T) {
		id, invite, reminder := lockedEpisode(t, "LockSession", "07700 900345")
		p := w.anon.AsPatient(t, invite, right)
		p.Do("GET", "/api/p/episodes/"+id.String(), nil).Want(t, http.StatusOK)
		for range 5 {
			tryDob(t, w.anon, reminder, wrong)
		}
		p.Do("GET", "/api/p/episodes/"+id.String(), nil).Want(t, http.StatusUnauthorized)
	})
	t.Run("a locked episode gets no reminders, only a note; not responding still comes", func(t *testing.T) {
		id, invite, _ := lockedEpisode(t, "LockNoRemind", "07700 900346")
		for range 5 {
			tryDob(t, w.anon, invite, wrong)
		}
		s := runJobs(t, w.anon, dayOf(t, w, id, 5))
		if len(msgs.To("+447700900346")) != 2 {
			t.Errorf("a reminder went while locked: summary %+v, sent %+v", s, msgs.To("+447700900346"))
		}
		if ev := lastEvent(t, w, id); ev.Text != "Day 5 reminder not sent: the patient's link is locked after 5 wrong dates of birth. Resend the invite to send them a new link." {
			t.Errorf("event %+v", ev)
		}
		if f := readFollowup(t, w, id); f.RemindersSent != 2 {
			t.Errorf("follow-up %+v", f)
		}
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1 AND locked_at IS NULL AND revoked_at IS NULL`, id); n != 0 {
			t.Errorf("a working link was made while locked: %d", n)
		}
		runJobs(t, w.anon, dayOf(t, w, id, 7))
		if f := readFollowup(t, w, id); f.NotRespondingAt == nil {
			t.Errorf("day 7: %+v", f)
		}
	})
	t.Run("only staff's Resend clears the lock; the later reminder days carry on", func(t *testing.T) {
		id := newPatientEpisode(t, w, "LockResend", "07700 900347", "")
		invite := linkIn(t, msgs.To("+447700900347")[0])
		for range 5 {
			tryDob(t, w.anon, invite, wrong)
		}
		runJobs(t, w.anon, dayOf(t, w, id, 2)) // withheld
		if len(msgs.To("+447700900347")) != 1 {
			t.Fatalf("sent %+v", msgs.To("+447700900347"))
		}
		w.clinician.Do("POST", at(w.a, "/episodes/"+id.String()+"/resend-invite"), nil).Want(t, http.StatusOK)
		if staffInvite(t, id) != nil {
			t.Error("still shown locked after Resend")
		}
		resent := linkIn(t, msgs.To("+447700900347")[1])
		if r := tryDob(t, w.anon, resent, wrong); !strings.Contains(r.Message(), "You have 4 tries left") {
			t.Errorf("after Resend: %s", r)
		}
		tryDob(t, w.anon, invite, right).Want(t, http.StatusNotFound) // replaced by the Resend
		s := runJobs(t, w.anon, dayOf(t, w, id, 5))
		if s.Reminders != 1 || len(msgs.To("+447700900347")) != 3 {
			t.Errorf("day 5 after Resend: %+v, sent %+v", s, msgs.To("+447700900347"))
		}
	})
}

// On AWS (S17) the jobs Lambda's schedule runs the jobs, not a timer in the API: the page says so,
// with the schedule's interval (JOBS_SCHEDULE_MINUTES, set by infra from the country entry).
func TestQAJobsOnASchedule(t *testing.T) {
	t.Parallel()
	c, _ := apitest.NewWith(t, apitest.Options{Env: map[string]string{"JOBS_SCHEDULE_MINUTES": "15", "JOBS_TIMER": "false"}})
	r := c.Do("GET", "/api/qa/jobs", nil)
	r.Want(t, http.StatusOK)
	var got struct {
		Timer           bool
		Runner          string
		IntervalSeconds int
	}
	r.Decode(t, &got)
	if !got.Timer || got.Runner != "schedule" || got.IntervalSeconds != 900 {
		t.Errorf("got %+v", got)
	}
	local, _ := apitest.New(t) // and locally it's the timer
	r = local.Do("GET", "/api/qa/jobs", nil)
	r.Decode(t, &got)
	if got.Runner != "timer" || got.IntervalSeconds != 60 {
		t.Errorf("local %+v", got)
	}
}
