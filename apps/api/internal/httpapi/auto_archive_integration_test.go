//go:build integration

package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// The daily auto-archive (S22, EPI-05; answers 3 and 4), run through the testers' "run the jobs
// as of" (POST /api/qa/jobs/run), which runs every hospital's auto-archive as of then, and as the
// job's own schedule, 02:00 at each hospital. The decision is episode.ArchiveDue (unit-tested);
// these check the job archives what it says, with the rule in the history, and nothing else.

// lastActivity is the episode's last_activity_at.
func lastActivity(t *testing.T, w *clinic, id uuid.UUID) time.Time {
	t.Helper()
	var at time.Time
	if err := w.db.QueryRow(t.Context(), `SELECT last_activity_at FROM episodes WHERE id = $1`, id).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

func setActivity(t *testing.T, w *clinic, id uuid.UUID, at time.Time) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET last_activity_at = $2 WHERE id = $1`, id, at); err != nil {
		t.Fatal(err)
	}
}

// discharged makes e Discharged on date (YYYY-MM-DD).
func discharged(t *testing.T, w *clinic, e apitest.Episode, date string) {
	t.Helper()
	setStatus(t, w, e, episode.Discharged)
	if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET admitted_at = $2::date - 2, discharged_at = $2 WHERE id = $1`, e.ID, date); err != nil {
		t.Fatal(err)
	}
}

// atNoon is noon UTC on date + days: the same date at Hospital A, in the deployment's zone.
func atNoon(date string, days int) time.Time {
	d, err := time.Parse(time.DateOnly, date)
	if err != nil {
		panic(err)
	}
	return d.AddDate(0, 0, days).Add(12 * time.Hour)
}

func TestAutoArchive(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	// Hospital A2 overrides the days after discharge (7) and the months (2); A keeps the defaults.
	if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET settings = '{"autoArchive.afterDischargeDays": 7, "autoArchive.inactiveMonths": 2}' WHERE id = $1`, w.a2.ID); err != nil {
		t.Fatal(err)
	}
	const day0 = "2026-03-02"
	atA := w.fx.Episode(t, w.hq, w.cara)
	discharged(t, w, atA, day0)
	atA2 := w.fx.EpisodeAt(t, w.hq, w.a2, w.cara)
	discharged(t, w, atA2, day0)

	t.Run("29 days after discharge: only the hospital with 7 days archives", func(t *testing.T) {
		s := runJobs(t, w.anon, atNoon(day0, 29))
		if got := readStatus(t, w, atA.ID).Status; got != episode.Discharged {
			t.Errorf("Hospital A's (30 days): %s", got)
		}
		if got := readStatus(t, w, atA2.ID).Status; got != episode.Archived || archivedFrom(t, w, atA2.ID) != episode.Discharged {
			t.Errorf("Hospital A2's (7 days): %s", got)
		}
		if s.Archived != 1 || s.AutoArchives < 2 || s.Failed != 0 {
			t.Errorf("summary %+v", s)
		}
		h := readHistory(t, w, atA2.ID)
		if last := h[len(h)-1]; last.Move != "archive" || last.From != episode.Discharged || last.User != nil {
			t.Errorf("history %+v", last)
		}
		var rule string
		if err := w.db.QueryRow(t.Context(), `SELECT rule FROM episode_status_events WHERE episode_id = $1 AND move = 'archive'`, atA2.ID).Scan(&rule); err != nil || rule != "Auto-archive: 7 days after discharge" {
			t.Errorf("rule %q (%v)", rule, err)
		}
		ev := readEvents(t, w, atA2.ID)
		if last := ev[len(ev)-1]; last.Text != "Episode archived (Auto-archive: 7 days after discharge)" || last.User != nil {
			t.Errorf("General notes %+v", last)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_events WHERE episode_id = $1 AND automatic`, atA2.ID); n != 1 {
			t.Errorf("%d automatic notes", n)
		}
	})
	t.Run("31 days after discharge: archived, and its history names the rule", func(t *testing.T) {
		runJobs(t, w.anon, atNoon(day0, 31))
		if got := readStatus(t, w, atA.ID).Status; got != episode.Archived {
			t.Fatalf("Hospital A's: %s", got)
		}
		var rule string
		if err := w.db.QueryRow(t.Context(), `SELECT rule FROM episode_status_events WHERE episode_id = $1 AND move = 'archive'`, atA.ID).Scan(&rule); err != nil || rule != "Auto-archive: 30 days after discharge" {
			t.Errorf("rule %q (%v)", rule, err)
		}
		// The staff page's history shows it as automatic, with the rule.
		r := w.clinician.Do("GET", epPath(w.a, atA, ""), nil)
		r.Want(t, http.StatusOK)
		var v struct {
			StatusHistory []struct {
				Move     string
				Rule     *string
				UserName *string
			}
		}
		r.Decode(t, &v)
		if last := v.StatusHistory[len(v.StatusHistory)-1]; last.Move != "archive" || last.Rule == nil || *last.Rule != rule || last.UserName != nil {
			t.Errorf("history on the page %+v", last)
		}
	})
	t.Run("un-archived, it isn't archived again the next night: its days count from the un-archive", func(t *testing.T) {
		w.author.Do("POST", epPath(w.a, atA, "/unarchive"), withRevision(t, w, atA.ID, map[string]any{"reason": "Made-up: a follow-up visit"})).Want(t, http.StatusNoContent)
		runJobs(t, w.anon, time.Now().AddDate(0, 0, 1))
		if got := readStatus(t, w, atA.ID).Status; got != episode.Discharged {
			t.Errorf("the next night: %s", got)
		}
		runJobs(t, w.anon, time.Now().AddDate(0, 0, 31))
		if got := readStatus(t, w, atA.ID).Status; got != episode.Archived {
			t.Errorf("31 days after the un-archive: %s", got)
		}
	})

	t.Run("pre-op: untouched for 6 months is archived; one with a note last week isn't", func(t *testing.T) {
		long := time.Now().AddDate(0, -7, 0)
		untouched, noted, admitted, cancelled := w.fx.Episode(t, w.hq, w.cara), w.fx.Episode(t, w.hq, w.cara),
			w.fx.Episode(t, w.hq, w.cara), w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, untouched, episode.OnHold)
		setStatus(t, w, admitted, episode.Admitted)
		setStatus(t, w, cancelled, episode.Cancelled)
		for _, e := range []apitest.Episode{untouched, noted, admitted, cancelled} {
			setActivity(t, w, e.ID, long)
		}
		// A note today: written through the route, it moves the episode's last activity.
		w.clinician.Do("POST", epPath(w.a, noted, "/notes"), map[string]any{"text": "Made-up: rang the patient"}).Want(t, http.StatusCreated)
		if got := lastActivity(t, w, noted.ID); time.Since(got) > time.Minute {
			t.Fatalf("the note left the last activity at %s", got)
		}
		s := runJobs(t, w.anon, time.Now().AddDate(0, 0, 7)) // so the note was a week ago
		for _, c := range []struct {
			e    apitest.Episode
			want string
		}{{untouched, episode.Archived}, {noted, episode.HQNotComplete}, {admitted, episode.Admitted}, {cancelled, episode.Archived}} {
			if got := readStatus(t, w, c.e.ID).Status; got != c.want {
				t.Errorf("%s: %s, want %s", c.e.ID, got, c.want)
			}
		}
		if archivedFrom(t, w, cancelled.ID) != episode.Cancelled || s.Failed != 0 {
			t.Errorf("cancelled archived from %q; summary %+v", archivedFrom(t, w, cancelled.ID), s)
		}
		var rule string
		if err := w.db.QueryRow(t.Context(), `SELECT rule FROM episode_status_events WHERE episode_id = $1 AND move = 'archive'`, untouched.ID).Scan(&rule); err != nil || rule != "Auto-archive: 6 months without activity" {
			t.Errorf("rule %q (%v)", rule, err)
		}
	})
	t.Run("the hospital's own months: 2 at A2", func(t *testing.T) {
		e := w.fx.EpisodeAt(t, w.hq, w.a2, w.cara)
		setActivity(t, w, e.ID, time.Now().AddDate(0, -3, 0))
		atA := w.fx.Episode(t, w.hq, w.cara)
		setActivity(t, w, atA.ID, time.Now().AddDate(0, -3, 0))
		runJobs(t, w.anon, time.Now())
		if got := readStatus(t, w, e.ID).Status; got != episode.Archived {
			t.Errorf("A2's, 3 months: %s", got)
		}
		if got := readStatus(t, w, atA.ID).Status; got != episode.HQNotComplete {
			t.Errorf("A's, 3 months: %s", got)
		}
	})
	t.Run("nothing is archived while the hospital is suspended", func(t *testing.T) {
		e := w.fx.EpisodeAt(t, w.hq, w.a2, w.cara)
		setActivity(t, w, e.ID, time.Now().AddDate(-1, 0, 0))
		if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET status = 'suspended' WHERE id = $1`, w.a2.ID); err != nil {
			t.Fatal(err)
		}
		runJobs(t, w.anon, time.Now())
		if got := readStatus(t, w, e.ID).Status; got != episode.HQNotComplete {
			t.Errorf("suspended: %s", got)
		}
		if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET status = 'active' WHERE id = $1`, w.a2.ID); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("an archived episode's page was stale: its next write is refused", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, e, episode.OnHold)
		rev := revisionOf(t, w, e.ID)
		setActivity(t, w, e.ID, time.Now().AddDate(-1, 0, 0))
		runJobs(t, w.anon, time.Now())
		refused(t, w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": episode.NotReady, "revision": rev}), http.StatusConflict)
	})
}

// The job's own schedule (S22 answers 4): every hospital gets one auto-archive job, first due at
// its next 02:00, then again at the next 02:00 after each run, in its own time zone.
func TestAutoArchiveSchedule(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET timezone = 'Pacific/Kiritimati' WHERE id = $1`, w.a2.ID); err != nil {
		t.Fatal(err)
	}
	runAt := func(h apitest.Hospital) time.Time {
		t.Helper()
		var at time.Time
		if err := w.db.QueryRow(t.Context(), `SELECT run_at FROM jobs WHERE kind = 'auto_archive' AND hospital_id = $1 AND done_at IS NULL`, h.ID).Scan(&at); err != nil {
			t.Fatalf("hospital %s's job: %v", h.ID, err)
		}
		return at
	}
	asOf := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	runJobs(t, w.anon, asOf)
	london, _ := time.LoadLocation("Europe/London")
	kiritimati, _ := time.LoadLocation("Pacific/Kiritimati")
	if got, want := runAt(w.a), episode.NextArchiveRun(asOf, london); !got.Equal(want) {
		t.Errorf("Hospital A (the deployment's zone): next run %s, want %s", got, want)
	}
	if got, want := runAt(w.a2), episode.NextArchiveRun(asOf, kiritimati); !got.Equal(want) {
		t.Errorf("Hospital A2 (its own zone): next run %s, want %s", got, want)
	}
	if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE kind = 'auto_archive' AND done_at IS NULL`); n != count(t, w.db, `SELECT count(*) FROM hospitals`) {
		t.Errorf("%d auto-archive jobs", n)
	}
	// A new hospital gets its job at the next run; running again adds no second job.
	h := w.fx.HospitalIn(t, w.a.Org)
	runJobs(t, w.anon, asOf)
	runAt(h)
	if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE kind = 'auto_archive' AND hospital_id = $1`, w.a.ID); n != 1 {
		t.Errorf("Hospital A has %d auto-archive jobs", n)
	}
}

// last_activity_at (S22 answers 3): every write to the episode moves it forward: answers (the
// patient's and the clinician's), status, notes, tasks and messages sent.
func TestLastActivity(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	long := time.Now().AddDate(-1, 0, 0)
	moved := func(t *testing.T, id uuid.UUID, what string) {
		t.Helper()
		if got := lastActivity(t, w, id); time.Since(got) > time.Minute {
			t.Errorf("%s left the last activity at %s", what, got)
		}
	}
	e := w.fx.Episode(t, w.hq, w.cara)
	set := w.hq.Chapters[0].String()

	setActivity(t, w, e.ID, long)
	p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
	p.Do("PUT", patientPath(e, "/answers/"+set), map[string]any{"data": map[string]any{}, "revision": 0}).Want(t, http.StatusOK)
	moved(t, e.ID, "the patient's answers")

	setActivity(t, w, e.ID, long)
	w.clinician.Do("POST", epPath(w.a, e, "/notes"), map[string]any{"text": "x"}).Want(t, http.StatusCreated)
	moved(t, e.ID, "a note")

	setActivity(t, w, e.ID, long)
	w.clinician.Do("POST", epPath(w.a, e, "/tasks"), map[string]any{"title": "Call", "team": "nurse", "revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusCreated)
	moved(t, e.ID, "a task")

	invited := newPatientEpisode(t, w, "Activity", "+447700900931", "") // a patient with a mobile, invited
	setActivity(t, w, invited, long)
	w.clinician.Do("POST", at(w.a, "/episodes/"+invited.String()+"/resend-invite"), nil).Want(t, http.StatusOK)
	moved(t, invited, "a message sent")

	f := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, f, episode.ReadyForReview)
	setActivity(t, w, f.ID, long)
	w.clinician.Do("PUT", epPath(w.a, f, "/answers/"+set), map[string]any{"data": map[string]any{}, "revision": 0}).Want(t, http.StatusOK)
	moved(t, f.ID, "the clinician's answers")

	setStatus(t, w, f, episode.ReadyForPOA)
	setActivity(t, w, f.ID, long)
	w.clinician.Do("PATCH", epPath(w.a, f, "/status"), map[string]any{"to": episode.OnHold, "revision": revisionOf(t, w, f.ID)}).Want(t, http.StatusNoContent)
	moved(t, f.ID, "a status change")

	// Another episode's write leaves this one's alone.
	setActivity(t, w, e.ID, long)
	w.clinician.Do("POST", epPath(w.a, f, "/notes"), map[string]any{"text": "x"}).Want(t, http.StatusCreated)
	if got := lastActivity(t, w, e.ID); got.Sub(long).Abs() > time.Second {
		t.Errorf("another episode's note moved this one's last activity to %s", got)
	}
}

// Test report 1, F-2: a run "as of" a later time is one moment. The auto-archive decides first, on
// the activity there has been, so an idle episode due both a reminder and the archive is archived
// (and not reminded); and what the jobs write (notes, so activity) carries the run's time, not
// the real clock's.
func TestJobRunsUseTheirOwnTime(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	idle := newPatientEpisode(t, w, "Idle", "", "idle@example.test") // invited: its reminders are due from day 2
	sent := len(w.anon.Messages.To("idle@example.test"))
	asOf := time.Now().AddDate(0, 6, 0).Add(time.Minute)
	runJobs(t, w.anon, asOf)
	if got := readStatus(t, w, idle).Status; got != episode.Archived {
		t.Errorf("the idle episode is %s; want archived before any reminder made it look active", got)
	}
	if n := len(w.anon.Messages.To("idle@example.test")); n != sent {
		t.Errorf("%d messages sent to an episode archived in the same run", n-sent)
	}

	reminded := newPatientEpisode(t, w, "Reminded", "", "reminded@example.test")
	day2 := time.Now().AddDate(0, 0, 2).Add(time.Hour)
	runJobs(t, w.anon, day2)
	var at time.Time
	if err := w.db.QueryRow(t.Context(), `SELECT created_at FROM episode_events WHERE episode_id = $1 AND automatic
		ORDER BY created_at DESC LIMIT 1`, reminded).Scan(&at); err != nil {
		t.Fatalf("no reminder note: %v", err)
	}
	if at.Sub(day2).Abs() > time.Second {
		t.Errorf("the reminder note is dated %s, want the run's time %s", at, day2)
	}
	if got := lastActivity(t, w, reminded); got.Sub(day2).Abs() > time.Second {
		t.Errorf("last activity %s, want the run's time %s", got, day2)
	}
	var archivedAt time.Time
	if err := w.db.QueryRow(t.Context(), `SELECT created_at FROM episode_status_events WHERE episode_id = $1 AND move = 'archive'`, idle).Scan(&archivedAt); err != nil {
		t.Fatal(err)
	}
	if archivedAt.Sub(asOf).Abs() > time.Second {
		t.Errorf("archived at %s in the history, want the run's time %s", archivedAt, asOf)
	}
}
