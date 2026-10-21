//go:build integration

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// The episode's tasks (S21, TSK-01): GET and POST …/episodes/{eid}/tasks, PATCH …/tasks/{tid}.
// Each gets the five cases. Every hospital member is a clinician at least, and anyone at the
// hospital may complete any task (S21 answers 1), so the "role" case is a clinician at the trust's
// other hospital (404), as for the other episode routes. Writes send the episode's revision (S20)
// and bump it: a task is part of the episode.

// taskRow is a tasks row as the tests check it.
type taskRow struct {
	ID                            uuid.UUID
	Title, Team                   string
	CreatedBy, DoneBy, ReopenedBy *uuid.UUID
	DoneAt, ReopenedAt            *time.Time
}

func readTasks(t *testing.T, w *clinic, episodeID uuid.UUID) []taskRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT id, title, team, created_by, done_by, reopened_by, done_at, reopened_at
		FROM tasks WHERE episode_id = $1 ORDER BY created_at, id`, episodeID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []taskRow
	for rows.Next() {
		var r taskRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Team, &r.CreatedBy, &r.DoneBy, &r.ReopenedBy, &r.DoneAt, &r.ReopenedAt); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// addTask inserts a task straight into the database.
func addTask(t *testing.T, w *clinic, e apitest.Episode, title, team string, done bool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := w.db.QueryRow(t.Context(), `INSERT INTO tasks (episode_id, title, team, created_by, done_by, done_at)
		VALUES ($1, $2, $3, $4, CASE WHEN $5 THEN $4::uuid END, CASE WHEN $5 THEN now() END) RETURNING id`,
		e.ID, title, team, w.cara.ID, done).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestListTasks(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	first := addTask(t, w, w.ep, "Check bloods", episode.Nurse, true)
	second := addTask(t, w, w.ep, "Book anaesthetic review", episode.Anaesthetist, false)
	addTask(t, w, w.fx.Episode(t, w.hq, w.cara), "Another episode's", episode.Admin, false)
	list := func(c *apitest.Client, h apitest.Hospital) *apitest.Resp {
		return c.Do("GET", epPath(h, w.ep, "/tasks"), nil)
	}

	t.Run("happy path: the episode's tasks, oldest first, with who added and completed them", func(t *testing.T) {
		r := list(w.clinician, w.a)
		r.Want(t, http.StatusOK)
		var got []struct {
			ID                         uuid.UUID
			Title, Team, CreatedByName string
			DoneByName                 *string
			DoneAt                     *time.Time
			CreatedAt                  time.Time
		}
		r.Decode(t, &got)
		if len(got) != 2 || got[0].ID != first || got[1].ID != second || got[0].Title != "Check bloods" || got[0].Team != episode.Nurse ||
			got[0].CreatedByName == "" || got[0].DoneByName == nil || got[0].DoneAt == nil || got[1].DoneAt != nil || got[1].DoneByName != nil {
			t.Errorf("tasks = %+v", got)
		}
	})
	t.Run("happy path: none is an empty list", func(t *testing.T) {
		r := w.clinician.Do("GET", epPath(w.a, w.fx.Episode(t, w.hq, w.cara), "/tasks"), nil)
		r.Want(t, http.StatusOK)
		if s := strings.TrimSpace(string(r.Body)); s != "[]" {
			t.Errorf("body = %s, want []", s)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, list(w.anon, w.a), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, list(w.site2, w.a), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, list(w.otherClin, w.a), http.StatusNotFound)
		refused(t, list(w.otherClin, w.b), http.StatusNotFound)
	})
	t.Run("invalid input: not an episode", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/abc/tasks"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/"+uuid.NewString()+"/tasks"), nil), http.StatusNotFound)
	})
}

func TestCreateTask(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	post := func(c *apitest.Client, h apitest.Hospital, body any) *apitest.Resp {
		return c.Do("POST", epPath(h, w.ep, "/tasks"), body)
	}
	body := func(title, team string) map[string]any {
		return map[string]any{"title": title, "team": team, "revision": revisionOf(t, w, w.ep.ID)}
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, post(w.anon, w.a, body("Check bloods", episode.Nurse)), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, post(w.site2, w.a, body("Check bloods", episode.Nurse)), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, post(w.otherClin, w.a, body("Check bloods", episode.Nurse)), http.StatusNotFound)
		refused(t, post(w.otherClin, w.b, body("Check bloods", episode.Nurse)), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		rev := revisionOf(t, w, w.ep.ID)
		for _, bad := range []any{
			map[string]any{"title": "Check bloods", "team": episode.Nurse}, // no revision
			map[string]any{"title": "Check bloods", "team": episode.Nurse, "revision": -1},
			map[string]any{"title": "  ", "team": episode.Nurse, "revision": rev},
			map[string]any{"title": strings.Repeat("x", 201), "team": episode.Nurse, "revision": rev},
			map[string]any{"title": "Check\nbloods", "team": episode.Nurse, "revision": rev},
			map[string]any{"title": "Check bloods", "team": "surgeon", "revision": rev},
			map[string]any{"title": "Check bloods", "revision": rev}, // no team
			`{"title":`,
		} {
			refused(t, w.clinician.Do("POST", epPath(w.a, w.ep, "/tasks"), bad), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/abc/tasks"), body("Check bloods", episode.Nurse)), http.StatusNotFound)
	})
	t.Run("stale: a revision the episode has moved on from → 409 code stale, and nothing added", func(t *testing.T) {
		b := body("Check bloods", episode.Nurse)
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET revision = revision + 1 WHERE id = $1`, w.ep.ID); err != nil {
			t.Fatal(err)
		}
		r := post(w.clinician, w.a, b)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(string(r.Body), `"code":"stale"`) {
			t.Errorf("body %s, want code stale", r.Body)
		}
	})
	if got := readTasks(t, w, w.ep.ID); len(got) != 0 {
		t.Fatalf("refusals added tasks: %+v", got)
	}

	t.Run("happy path: one for each team, by the person, with the revision bumped and a General notes line", func(t *testing.T) {
		before := revisionOf(t, w, w.ep.ID)
		r := post(w.clinician, w.a, body("  Book anaesthetic review ", episode.Anaesthetist))
		r.Want(t, http.StatusCreated)
		var created struct{ ID uuid.UUID }
		r.Decode(t, &created)
		post(w.clin, w.a, body("Check bloods", episode.Nurse)).Want(t, http.StatusCreated)
		post(w.author, w.a, body("Send the letter", episode.Admin)).Want(t, http.StatusCreated) // a trust-wide super clinician
		got := readTasks(t, w, w.ep.ID)
		if len(got) != 3 || got[0].ID != created.ID || got[0].Title != "Book anaesthetic review" || got[0].Team != episode.Anaesthetist ||
			got[0].CreatedBy == nil || *got[0].CreatedBy != w.cara.ID || got[0].DoneAt != nil || got[2].Team != episode.Admin ||
			*got[2].CreatedBy != w.alex.ID {
			t.Errorf("tasks = %+v", got)
		}
		if rev := revisionOf(t, w, w.ep.ID); rev != before+3 {
			t.Errorf("revision %d, want %d", rev, before+3)
		}
		if n := readEvents(t, w, w.ep.ID); len(n) != 3 || n[0].Text != "Task added for the anaesthetist team: Book anaesthetic review" ||
			n[0].User == nil || *n[0].User != w.cara.ID {
			t.Errorf("notes = %+v", n)
		}
	})
	t.Run("a cancelled episode is read-only: no new task (409)", func(t *testing.T) {
		setStatus(t, w, w.ep, episode.Cancelled)
		refused(t, post(w.clinician, w.a, body("Too late", episode.Nurse)), http.StatusConflict)
		if got := readTasks(t, w, w.ep.ID); len(got) != 3 {
			t.Errorf("tasks = %+v", got)
		}
	})
}

func TestUpdateTask(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	task := addTask(t, w, w.ep, "Book anaesthetic review", episode.Anaesthetist, false)
	otherEp := w.fx.Episode(t, w.hq, w.cara)
	theirs := addTask(t, w, otherEp, "Another episode's", episode.Nurse, false)
	path := func(h apitest.Hospital, id uuid.UUID) string { return epPath(h, w.ep, "/tasks/"+id.String()) }
	body := func(done bool) map[string]any {
		return map[string]any{"done": done, "revision": revisionOf(t, w, w.ep.ID)}
	}
	patch := func(c *apitest.Client, h apitest.Hospital, b any) *apitest.Resp {
		return c.Do("PATCH", path(h, task), b)
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, patch(w.anon, w.a, body(true)), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, patch(w.site2, w.a, body(true)), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, patch(w.otherClin, w.a, body(true)), http.StatusNotFound)
		refused(t, patch(w.otherClin, w.b, body(true)), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		rev := revisionOf(t, w, w.ep.ID)
		for _, bad := range []any{
			map[string]any{"done": true},                   // no revision
			map[string]any{"revision": rev},                // no done
			map[string]any{"done": "yes", "revision": rev}, // not a boolean
			`{"done":`,
		} {
			refused(t, w.clinician.Do("PATCH", path(w.a, task), bad), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("PATCH", epPath(w.a, w.ep, "/tasks/abc"), body(true)), http.StatusNotFound)
		refused(t, w.clinician.Do("PATCH", path(w.a, theirs), body(true)), http.StatusNotFound) // another episode's task
		refused(t, w.clinician.Do("PATCH", path(w.a, uuid.New()), body(true)), http.StatusNotFound)
	})
	t.Run("stale: 409 code stale", func(t *testing.T) {
		b := body(true)
		b["revision"] = b["revision"].(int32) + 1
		r := patch(w.clinician, w.a, b)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(string(r.Body), `"code":"stale"`) {
			t.Errorf("body %s", r.Body)
		}
	})
	if got := readTasks(t, w, w.ep.ID); got[0].DoneAt != nil {
		t.Fatalf("refusals changed the task: %+v", got)
	}

	t.Run("happy path: anyone at the hospital completes it, then re-opens it, each recorded", func(t *testing.T) {
		before := revisionOf(t, w, w.ep.ID)
		patch(w.clin, w.a, body(true)).Want(t, http.StatusNoContent) // not the clinician who added it
		got := readTasks(t, w, w.ep.ID)[0]
		clinID := got.DoneBy
		if got.DoneAt == nil || clinID == nil || *clinID == w.cara.ID || got.ReopenedAt != nil {
			t.Errorf("completed task = %+v", got)
		}
		refused(t, patch(w.clinician, w.a, body(true)), http.StatusConflict) // already done
		patch(w.clinician, w.a, body(false)).Want(t, http.StatusNoContent)
		got = readTasks(t, w, w.ep.ID)[0]
		if got.DoneAt != nil || got.DoneBy != nil || got.ReopenedBy == nil || *got.ReopenedBy != w.cara.ID || got.ReopenedAt == nil {
			t.Errorf("re-opened task = %+v", got)
		}
		refused(t, patch(w.clinician, w.a, body(false)), http.StatusConflict) // already open
		if rev := revisionOf(t, w, w.ep.ID); rev != before+2 {
			t.Errorf("revision %d, want %d (one per change)", rev, before+2)
		}
		n := readEvents(t, w, w.ep.ID)
		if len(n) != 2 || n[0].Text != "Task completed: Book anaesthetic review" || n[1].Text != "Task re-opened: Book anaesthetic review" {
			t.Errorf("notes = %+v", n)
		}
		r := w.clinician.Do("GET", epPath(w.a, w.ep, "/tasks"), nil)
		var view []struct{ ReopenedByName *string }
		r.Decode(t, &view)
		if len(view) != 1 || view[0].ReopenedByName == nil || *view[0].ReopenedByName == "" {
			t.Errorf("tasks view = %s, want who re-opened it", r.Body)
		}
	})
	t.Run("a cancelled episode is read-only: its tasks can't be changed (409)", func(t *testing.T) {
		setStatus(t, w, w.ep, episode.Cancelled)
		refused(t, patch(w.clinician, w.a, body(true)), http.StatusConflict)
		if got := readTasks(t, w, w.ep.ID)[0]; got.DoneAt != nil {
			t.Errorf("task = %+v", got)
		}
	})
}
