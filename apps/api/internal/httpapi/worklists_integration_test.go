//go:build integration

package httpapi_test

import (
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// The worklists (S21, EPI-02): GET …/episodes?worklist=<key> (a hospital's, and a trust's "All
// hospitals"), and their counts, GET …/worklists. Each worklist must return exactly its episodes:
// the set-up has, for each rule, episodes just inside it and just outside it.

// worklistWorld is hospital A (Pacific/Kiritimati, UTC+14) and A2 (Pacific/Pago_Pago, UTC-11), so
// their dates are always different, and "today" must be each hospital's own. A2 sets its own
// "procedure within" days: 7 (A has the platform's 14).
type worklistWorld struct {
	*clinic
	want       map[string][]uuid.UUID // A's episodes in each worklist
	inA2       map[string][]uuid.UUID // A2's
	trustAdmin *apitest.Client        // a whole-trust admin: sees both hospitals
}

func newWorklistWorld(t *testing.T) *worklistWorld {
	t.Helper()
	w := &worklistWorld{clinic: newClinic(t), want: map[string][]uuid.UUID{}, inA2: map[string][]uuid.UUID{}}
	w.trustAdmin = w.anon.LoginAs(t, w.fx.TrustUser(t, w.a.Org, "admin"))
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := w.db.Exec(t.Context(), sql, args...); err != nil {
			t.Fatalf("%v\n%s", err, sql)
		}
	}
	exec(`UPDATE hospitals SET timezone = 'Pacific/Kiritimati' WHERE id = $1`, w.a.ID)
	exec(`UPDATE hospitals SET timezone = 'Pacific/Pago_Pago', settings = '{"worklists.soonDays": 7}' WHERE id = $1`, w.a2.ID)
	today := func(zone string) time.Time {
		loc, err := time.LoadLocation(zone)
		if err != nil {
			t.Fatal(err)
		}
		y, m, d := w.anon.Clock.Now().In(loc).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	todayA, todayA2 := today("Pacific/Kiritimati"), today("Pacific/Pago_Pago")

	// ep makes an episode at hospital h in a status, with a planned date (days from today there,
	// or none), then adds it to the worklists it should be in.
	ep := func(h apitest.Hospital, status string, planned *int, in ...string) apitest.Episode {
		t.Helper()
		e := w.fx.EpisodeAt(t, w.hq, h, w.cara)
		setStatus(t, w.clinic, e, status)
		if planned != nil {
			base := todayA
			if h.ID == w.a2.ID {
				base = todayA2
			}
			exec(`UPDATE episodes SET planned_date = $2 WHERE id = $1`, e.ID, base.AddDate(0, 0, *planned).Format(time.DateOnly))
		}
		into := w.want
		if h.ID == w.a2.ID {
			into = w.inA2
		}
		for _, k := range in {
			into[k] = append(into[k], e.ID)
		}
		return e
	}
	days := func(n int) *int { return &n }

	// Ready for review: the status alone.
	ep(w.a, episode.ReadyForReview, nil, "ready_for_review")
	ep(w.a, episode.ReadyForPOA, nil) // just after: the review is complete

	// Not responding: S16's flag while the HQ waits for the patient; not once they've sent it.
	nr := ep(w.a, episode.HQNotComplete, nil, "not_responding")
	exec(`UPDATE episodes SET not_responding_at = now() WHERE id = $1`, nr.ID)
	sent := ep(w.a, episode.ReadyForReview, nil, "ready_for_review")
	exec(`UPDATE episodes SET not_responding_at = now() WHERE id = $1`, sent.ID)
	// (w.ep, made by newClinic: waiting, not flagged, so in none.)

	// Anaesthetic review and open tasks: an open anaesthetist task is in both; a done one is in
	// neither; an open nurse task is only in Open tasks; a cancelled episode's open task in none.
	anaes := ep(w.a, episode.ReadyForPOA, nil, "anaesthetic_review", "open_tasks")
	addTask(t, w.clinic, anaes, "Book anaesthetic review", episode.Anaesthetist, false)
	addTask(t, w.clinic, anaes, "Done already", episode.Anaesthetist, true)
	nurse := ep(w.a, episode.POAComplete, nil, "open_tasks")
	addTask(t, w.clinic, nurse, "Book anaesthetic review", episode.Anaesthetist, true)
	addTask(t, w.clinic, nurse, "Check bloods", episode.Nurse, false)
	done := ep(w.a, episode.POAComplete, nil)
	addTask(t, w.clinic, done, "All done", episode.Admin, true)
	cancelled := ep(w.a, episode.Cancelled, days(1))
	addTask(t, w.clinic, cancelled, "Book anaesthetic review", episode.Anaesthetist, false)

	// Procedure within 14 days (A) and not Ready for admission: today and day 14 are in; day 15,
	// yesterday and no date are out; so are Ready for admission and the statuses after it.
	ep(w.a, episode.NotReady, days(0), "procedure_soon")
	ep(w.a, episode.HQNotComplete, days(14), "procedure_soon")
	ep(w.a, episode.ReadyForPOA, days(15))
	ep(w.a, episode.ReadyForPOA, days(-1))
	ep(w.a, episode.ReadyForAdmission, days(3))
	ep(w.a, episode.Admitted, days(3))
	ep(w.a, episode.Discharged, days(3))

	// On hold: the status alone, here with a date inside the 14 days too.
	ep(w.a, episode.OnHold, days(5), "on_hold", "procedure_soon")

	// A2: its own 7 days (day 7 in, day 8 out, though A's 14 would take it), a review and a task.
	ep(w.a2, episode.ReadyForPOA, days(7), "procedure_soon")
	ep(w.a2, episode.ReadyForPOA, days(8))
	a2task := ep(w.a2, episode.ReadyForReview, nil, "ready_for_review", "open_tasks")
	addTask(t, w.clinic, a2task, "Call the patient", episode.Admin, false)

	// Trust B: in every list it could be, never in A's.
	theirs := w.fx.Episode(t, w.fx.PublishedHQ(t, w.b, w.bea, "basic.json"), w.bob)
	setStatus(t, w.clinic, theirs, episode.OnHold)
	return w
}

// listIDs is the ids a list route returns, sorted.
func listIDs(t *testing.T, c *apitest.Client, path string) []uuid.UUID {
	t.Helper()
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	var got []struct{ ID uuid.UUID }
	r.Decode(t, &got)
	out := []uuid.UUID{}
	for _, e := range got {
		out = append(out, e.ID)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	return out
}

func sorted(ids ...[]uuid.UUID) []uuid.UUID {
	out := []uuid.UUID{}
	for _, l := range ids {
		out = append(out, l...)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	return out
}

type worklistCount struct {
	Key, Label, Help string
	Count            int
}

func counts(t *testing.T, c *apitest.Client, path string) []worklistCount {
	t.Helper()
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	var got []worklistCount
	r.Decode(t, &got)
	return got
}

var worklistKeys = []string{"ready_for_review", "not_responding", "anaesthetic_review", "procedure_soon", "on_hold", "open_tasks"}

func TestWorklists(t *testing.T) {
	t.Parallel()
	w := newWorklistWorld(t)

	t.Run("happy path: each worklist returns exactly its episodes, at one hospital", func(t *testing.T) {
		for _, k := range worklistKeys {
			if got, want := listIDs(t, w.clinician, at(w.a, "/episodes?worklist="+k)), sorted(w.want[k]); !slices.Equal(got, want) {
				t.Errorf("%s at A = %v, want %v", k, got, want)
			}
			if got, want := listIDs(t, w.site2, at(w.a2, "/episodes?worklist="+k)), sorted(w.inA2[k]); !slices.Equal(got, want) {
				t.Errorf("%s at A2 = %v, want %v", k, got, want)
			}
		}
	})
	t.Run("happy path: the counts are the lists' lengths, in the agreed order, with the hospital's days", func(t *testing.T) {
		got := counts(t, w.clinician, at(w.a, "/worklists"))
		var keys []string
		for i, c := range got {
			keys = append(keys, c.Key)
			if c.Count != len(w.want[c.Key]) || c.Label == "" || c.Help == "" {
				t.Errorf("count %d = %+v, want %d", i, c, len(w.want[c.Key]))
			}
		}
		if !slices.Equal(keys, worklistKeys) {
			t.Errorf("worklists = %v, want %v", keys, worklistKeys)
		}
		if got[3].Label != "Procedure within 14 days" {
			t.Errorf("A's soon worklist = %q", got[3].Label)
		}
		if a2 := counts(t, w.site2, at(w.a2, "/worklists")); a2[3].Label != "Procedure within 7 days" || a2[3].Count != 1 || a2[0].Count != 1 {
			t.Errorf("A2's counts = %+v", a2)
		}
	})
	t.Run("happy path: a trust-wide member's All hospitals has both hospitals' episodes", func(t *testing.T) {
		all := counts(t, w.trustAdmin, "/api/o/"+w.a.Org.String()+"/worklists")
		for i, k := range worklistKeys {
			want := sorted(w.want[k], w.inA2[k])
			if got := listIDs(t, w.trustAdmin, "/api/o/"+w.a.Org.String()+"/episodes?worklist="+k); !slices.Equal(got, want) {
				t.Errorf("%s across the trust = %v, want %v", k, got, want)
			}
			if all[i].Key != k || all[i].Count != len(want) {
				t.Errorf("count %+v, want %s %d", all[i], k, len(want))
			}
		}
		if all[3].Label != "Procedure soon" {
			t.Errorf("several hospitals' soon worklist = %q, want no one number of days", all[3].Label)
		}
	})
	t.Run("happy path: a hospital-only clinician's All hospitals list is still only their hospital", func(t *testing.T) {
		for _, k := range worklistKeys {
			if got := listIDs(t, w.clinician, "/api/o/"+w.a.Org.String()+"/episodes?worklist="+k); !slices.Equal(got, sorted(w.want[k])) {
				t.Errorf("Cara's %s across the trust = %v, want only A's", k, got)
			}
		}
		for i, c := range counts(t, w.clinician, "/api/o/"+w.a.Org.String()+"/worklists") {
			if c.Count != len(w.want[worklistKeys[i]]) {
				t.Errorf("Cara's count %+v, want only A's", c)
			}
		}
	})
	t.Run("happy path: a worklist and a status filter together", func(t *testing.T) {
		got := listIDs(t, w.clinician, at(w.a, "/episodes?worklist=procedure_soon&status=on_hold"))
		if !slices.Equal(got, sorted(w.want["on_hold"])) {
			t.Errorf("on hold and soon = %v, want %v", got, w.want["on_hold"])
		}
	})
	t.Run("happy path: a trust's own days apply where a hospital sets none", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET settings = '{"worklists.soonDays": 20}' WHERE id = $1`, w.a.Org); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `UPDATE orgs SET settings = '{}' WHERE id = $1`, w.a.Org)
		got := counts(t, w.clinician, at(w.a, "/worklists"))
		if got[3].Label != "Procedure within 20 days" || got[3].Count != len(w.want["procedure_soon"])+1 { // day 15 comes in
			t.Errorf("soon with the trust's 20 days = %+v", got[3])
		}
		if a2 := counts(t, w.site2, at(w.a2, "/worklists")); a2[3].Label != "Procedure within 7 days" {
			t.Errorf("A2 keeps its own: %+v", a2[3])
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", at(w.a, "/episodes?worklist=on_hold"), nil), http.StatusUnauthorized)
		refused(t, w.anon.Do("GET", at(w.a, "/worklists"), nil), http.StatusUnauthorized)
		refused(t, w.anon.Do("GET", "/api/o/"+w.a.Org.String()+"/worklists", nil), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, w.site2.Do("GET", at(w.a, "/episodes?worklist=on_hold"), nil), http.StatusNotFound)
		refused(t, w.site2.Do("GET", at(w.a, "/worklists"), nil), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.otherClin.Do("GET", at(w.a, "/worklists"), nil), http.StatusNotFound)
		refused(t, w.otherClin.Do("GET", "/api/o/"+w.a.Org.String()+"/worklists", nil), http.StatusNotFound)
		refused(t, w.otherClin.Do("GET", at(w.a, "/episodes?worklist=on_hold"), nil), http.StatusNotFound)
		// B's own lists never hold A's episodes.
		bc := counts(t, w.otherClin, at(w.b, "/worklists"))
		got := map[string]int{}
		for _, c := range bc {
			got[c.Key] = c.Count
		}
		want := map[string]int{"ready_for_review": 0, "not_responding": 0, "anaesthetic_review": 0, "procedure_soon": 0, "on_hold": 1, "open_tasks": 0}
		if !maps.Equal(got, want) {
			t.Errorf("B's counts = %v, want %v", got, want)
		}
	})
	t.Run("invalid input: an unknown worklist", func(t *testing.T) {
		for _, k := range []string{"nonsense", "auto_triaged", "On_hold"} {
			refused(t, w.clinician.Do("GET", at(w.a, "/episodes?worklist="+k), nil), http.StatusBadRequest)
			refused(t, w.clinician.Do("GET", "/api/o/"+w.a.Org.String()+"/episodes?worklist="+k, nil), http.StatusBadRequest)
		}
		for _, p := range badIDs(w.a, "/worklists") {
			refused(t, w.clinician.Do("GET", p, nil), http.StatusNotFound)
		}
	})
}

// The counts change as episodes move: completing the last open task takes the episode out of
// Open tasks, and a status move takes it out of On hold.
func TestWorklistsFollowChanges(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.OnHold)
	countOf := func(key string) int {
		for _, c := range counts(t, w.clinician, at(w.a, "/worklists")) {
			if c.Key == key {
				return c.Count
			}
		}
		t.Fatalf("no %s", key)
		return 0
	}
	r := w.clinician.Do("POST", epPath(w.a, e, "/tasks"), map[string]any{"title": "Book review", "team": "anaesthetist", "revision": revisionOf(t, w, e.ID)})
	r.Want(t, http.StatusCreated)
	var task struct{ ID uuid.UUID }
	r.Decode(t, &task)
	if countOf("open_tasks") != 1 || countOf("anaesthetic_review") != 1 || countOf("on_hold") != 1 {
		t.Fatalf("after adding a task: %+v", counts(t, w.clinician, at(w.a, "/worklists")))
	}
	w.clinician.Do("PATCH", epPath(w.a, e, "/tasks/"+task.ID.String()), map[string]any{"done": true, "revision": revisionOf(t, w, e.ID)}).
		Want(t, http.StatusNoContent)
	w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": episode.NotReady, "revision": revisionOf(t, w, e.ID)}).
		Want(t, http.StatusNoContent)
	for _, k := range []string{"open_tasks", "anaesthetic_review", "on_hold"} {
		if n := countOf(k); n != 0 {
			t.Errorf("%s = %d after the task was done and the status moved, want 0", k, n)
		}
	}
	if got := listIDs(t, w.clinician, at(w.a, "/episodes?worklist=open_tasks")); len(got) != 0 {
		t.Errorf("open tasks = %v", got)
	}
}
