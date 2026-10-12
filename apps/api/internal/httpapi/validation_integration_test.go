//go:build integration

package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The clinician validates each Question Set, then completes the HQ review (plan-workflow.md
// Step 4). Two rules are the browser's, not the server's (decisions.md, 8 Oct): it pre-fills the
// clinician's answers from the patient's, and it enables "Complete HQ review" once every visible
// set is validated. S03's journey J4 covers them. These tests cover what the server enforces.

// clinicianRow is the clinician's answers row for a Question Set; Data is "" if there isn't one.
type clinicianRow struct {
	Data      string
	By        *uuid.UUID
	Validated *time.Time
	Revision  int
}

func readClinicianAnswers(t *testing.T, w *clinic, e apitest.Episode, chapter uuid.UUID) clinicianRow {
	t.Helper()
	var c clinicianRow
	err := w.db.QueryRow(t.Context(), `SELECT data::text, updated_by, validated_at, revision FROM episode_answers
		WHERE episode_id = $1 AND chapter_id = $2 AND actor = 'clinician'`, e.ID, chapter).Scan(&c.Data, &c.By, &c.Validated, &c.Revision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return c
}

// savedRevision checks a save's 200 and returns the revision it reports.
func savedRevision(t *testing.T, r *apitest.Resp) int {
	t.Helper()
	r.Want(t, http.StatusOK)
	var got struct{ Revision *int }
	r.Decode(t, &got)
	if got.Revision == nil {
		t.Fatalf("the save returned no revision: %s", r)
	}
	return *got.Revision
}

func TestEpisodeHQ(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, nurse := w.hq.Chapters[0], w.hq.Chapters[2]
	setStatus(t, w, w.ep, "ready_for_review")
	saveAnswers(t, w, w.ep, about, nil, `{"q_smokes": "o_yes"}`)
	saveAnswers(t, w, w.ep, about, &w.cara, `{"q_smokes": "o_no"}`)
	get := func(c *apitest.Client, h apitest.Hospital) *apitest.Resp {
		return c.Do("GET", epPath(h, w.ep, "/hq"), nil)
	}

	t.Run("happy path: every Question Set in full, with both actors' answers", func(t *testing.T) {
		r := get(w.clinician, w.a)
		r.Want(t, http.StatusOK)
		var got struct {
			Chapters []struct {
				ID       uuid.UUID
				Audience string
			}
			Answers []struct {
				ChapterID            uuid.UUID
				Actor, UpdatedByName string
				Data                 json.RawMessage
				Revision             *int
			}
		}
		r.Decode(t, &got)
		if len(got.Chapters) != 3 || got.Chapters[2].ID != nurse || got.Chapters[2].Audience != "clinician" {
			t.Errorf("chapters = %+v, want all 3, the clinician set last", got.Chapters)
		}
		// Clinicians see the clinician-only content the patient doesn't.
		for _, s := range []string{"q_secret_note", "q_secret_ecg", "q_on_secret_page", "clinicalOutputs", "testCases"} {
			if !strings.Contains(string(r.Body), s) {
				t.Errorf("the clinician's HQ is missing %q", s)
			}
		}
		actors := map[string]string{}
		for _, a := range got.Answers {
			if a.ChapterID == about {
				actors[a.Actor] = string(a.Data)
			}
			// Each row's revision, which a save sends back (S02a). Rows saved before S02a are at 0.
			if a.Revision == nil || *a.Revision != 0 {
				t.Errorf("answers row %+v: want revision 0", a)
			}
		}
		if len(got.Answers) != 2 || actors["patient"] != `{"q_smokes":"o_yes"}` || actors["clinician"] != `{"q_smokes":"o_no"}` {
			t.Errorf("answers = %+v, want the patient's and the clinician's", got.Answers)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, get(w.anon, w.a), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, get(w.site2, w.a), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, get(w.otherClin, w.a), http.StatusNotFound)
		refused(t, get(w.otherClin, w.b), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/abc/hq"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/"+uuid.NewString()+"/hq"), nil), http.StatusNotFound)
	})
}

func TestSaveClinicianAnswers(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	setStatus(t, w, w.ep, "ready_for_review")
	const patientSaid = `{"q_heart": true, "q_smokes": "o_yes"}`
	saveAnswers(t, w, w.ep, about, nil, patientSaid)
	save := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, chapter uuid.UUID, body any) *apitest.Resp {
		return c.Do("PUT", epPath(h, e, "/answers/"+chapter.String()), body)
	}
	// The browser pre-fills from the patient's answers; the clinician corrects one. Each save
	// sends the revision it started from: 0 when the clinician has no row yet.
	correctedData := map[string]any{"q_smokes": "o_no", "q_heart": true}
	corrected := map[string]any{"data": correctedData, "revision": 0}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, save(w.anon, w.a, w.ep, about, corrected), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, save(w.site2, w.a, w.ep, about, corrected), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, save(w.otherClin, w.a, w.ep, about, corrected), http.StatusNotFound)
		refused(t, save(w.otherClin, w.b, w.ep, about, corrected), http.StatusNotFound)
		theirs := w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")
		refused(t, save(w.clinician, w.a, w.ep, theirs.Chapters[0], corrected), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, save(w.clinician, w.a, w.ep, uuid.New(), corrected), http.StatusNotFound)
		refused(t, w.clinician.Do("PUT", epPath(w.a, w.ep, "/answers/abc"), corrected), http.StatusNotFound)
		refused(t, w.clinician.Do("PUT", at(w.a, "/episodes/abc/answers/"+about.String()), corrected), http.StatusNotFound)
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data":`), http.StatusBadRequest)
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": [1]}`), http.StatusBadRequest)
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": {}, "validated": "yes", "revision": 0}`), http.StatusBadRequest)
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": {}}`), http.StatusBadRequest) // no revision
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": {}, "revision": null}`), http.StatusBadRequest)
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": {}, "revision": "0"}`), http.StatusBadRequest)
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": {}, "revision": 1.5}`), http.StatusBadRequest)
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": {}, "revision": -1}`), http.StatusBadRequest)
	})
	t.Run("before the patient submits", func(t *testing.T) {
		waiting := w.fx.Episode(t, w.hq, w.cara)
		r := save(w.clinician, w.a, waiting, about, corrected)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "hasn't sent") {
			t.Errorf("message = %q", r.Message())
		}
	})
	if n := count(t, w.db, "SELECT count(*) FROM episode_answers WHERE actor = 'clinician'"); n != 0 {
		t.Fatalf("refusals saved %d clinician rows", n)
	}

	t.Run("happy path: the first write makes the clinician's own row, at revision 1", func(t *testing.T) {
		if rev := savedRevision(t, save(w.clinician, w.a, w.ep, about, corrected)); rev != 1 {
			t.Errorf("revision = %d, want 1", rev)
		}
		c := readClinicianAnswers(t, w, w.ep, about)
		if c.Data != `{"q_heart": true, "q_smokes": "o_no"}` || c.By == nil || *c.By != w.cara.ID || c.Validated != nil || c.Revision != 1 {
			t.Errorf("clinician row = %+v, want the corrected answers by Cara, not validated, revision 1", c)
		}
		if got := patientAnswers(t, w, w.ep, about); got != patientSaid {
			t.Errorf("the patient's answers changed: %s", got)
		}
	})
	t.Run("happy path: validating stamps the set, at the next revision", func(t *testing.T) {
		start := time.Now().Add(-time.Minute)
		if rev := savedRevision(t, save(w.clinician, w.a, w.ep, about, map[string]any{"data": correctedData, "validated": true, "revision": 1})); rev != 2 {
			t.Errorf("revision = %d, want 2", rev)
		}
		c := readClinicianAnswers(t, w, w.ep, about)
		if c.Validated == nil || c.Validated.Before(start) || c.By == nil || *c.By != w.cara.ID || c.Revision != 2 {
			t.Errorf("clinician row = %+v, want validated now by Cara, revision 2", c)
		}
	})
	t.Run("happy path: a clinician Question Set, which has no patient row", func(t *testing.T) {
		savedRevision(t, save(w.clinician, w.a, w.ep, nurse, map[string]any{"data": map[string]string{"q_secret_bp": "120/80"}, "validated": true, "revision": 0}))
		if c := readClinicianAnswers(t, w, w.ep, nurse); c.Validated == nil || c.Revision != 1 {
			t.Errorf("clinician set = %+v, want validated, revision 1", c)
		}
	})
	t.Run("a later save without validated clears the stamp", func(t *testing.T) {
		savedRevision(t, save(w.clinician, w.a, w.ep, about, map[string]any{"data": correctedData, "revision": 2}))
		if c := readClinicianAnswers(t, w, w.ep, about); c.Validated != nil || c.Revision != 3 {
			t.Errorf("clinician row = %+v, want the stamp cleared, revision 3", c)
		}
	})
	t.Run("a stale revision is refused, and nothing is written", func(t *testing.T) {
		before := readClinicianAnswers(t, w, w.ep, about)
		for _, body := range []map[string]any{
			{"data": map[string]string{"q_smokes": "o_yes"}, "revision": 2},                    // older
			{"data": map[string]string{"q_smokes": "o_yes"}, "revision": 4},                    // newer than the row
			{"data": map[string]string{"q_smokes": "o_yes"}, "revision": 0, "validated": true}, // "first" save, but there is a row
		} {
			r := save(w.clinician, w.a, w.ep, about, body)
			refused(t, r, http.StatusConflict)
			if !strings.Contains(r.Message(), "changed somewhere else") {
				t.Errorf("message = %q", r.Message())
			}
		}
		if after := readClinicianAnswers(t, w, w.ep, about); !reflect.DeepEqual(after, before) {
			t.Errorf("clinician row = %+v, want unchanged %+v", after, before)
		}
		// A revision for a row that doesn't exist yet: the health set has no clinician row.
		refused(t, save(w.clinician, w.a, w.ep, health, map[string]any{"data": map[string]bool{"q_diabetes": true}, "revision": 3}), http.StatusConflict)
		if c := readClinicianAnswers(t, w, w.ep, health); c.Data != "" {
			t.Errorf("a stale save made a row: %+v", c)
		}
	})
	t.Run("X-1: a stale tab's Validate doesn't overwrite a newer answer", func(t *testing.T) {
		// Both tabs opened the set at the current revision.
		opened := readClinicianAnswers(t, w, w.ep, about).Revision
		// Tab A changes smoking to Yes, and it autosaves.
		savedRevision(t, save(w.clinician, w.a, w.ep, about, map[string]any{"data": map[string]any{"q_smokes": "o_yes", "q_heart": true}, "revision": opened}))
		// Tab B, still showing No, validates.
		r := save(w.clinician, w.a, w.ep, about, map[string]any{"data": correctedData, "validated": true, "revision": opened})
		refused(t, r, http.StatusConflict)
		if c := readClinicianAnswers(t, w, w.ep, about); c.Data != `{"q_heart": true, "q_smokes": "o_yes"}` || c.Validated != nil || c.Revision != opened+1 {
			t.Errorf("clinician row = %+v, want tab A's answer, unvalidated, revision %d", c, opened+1)
		}
	})
	t.Run("two first saves at once: one wins, the other gets a 409", func(t *testing.T) {
		// The other tab's first save is in, but not committed, when this one arrives.
		r := meanwhile(t, w.db, `INSERT INTO episode_answers (episode_id, chapter_id, actor, data, updated_by, revision)
			VALUES ($1, $2, 'clinician', '{"q_diabetes": true}', $3, 1)`, []any{w.ep.ID, health, w.cara.ID},
			func() *apitest.Resp {
				return save(w.clinician, w.a, w.ep, health, map[string]any{"data": map[string]bool{"q_diabetes": false}, "revision": 0})
			})
		refused(t, r, http.StatusConflict)
		if c := readClinicianAnswers(t, w, w.ep, health); c.Data != `{"q_diabetes": true}` || c.Revision != 1 {
			t.Errorf("clinician row = %+v, want the first save's, revision 1", c)
		}
	})
	t.Run("read-only once the review is complete", func(t *testing.T) {
		before := readClinicianAnswers(t, w, w.ep, nurse)
		if before.Validated == nil {
			t.Fatal("the clinician set wasn't validated by the earlier case")
		}
		setStatus(t, w, w.ep, "ready_for_poa")
		r := save(w.clinician, w.a, w.ep, nurse, map[string]any{"data": map[string]string{"q_secret_bp": "changed"}, "validated": true, "revision": before.Revision})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "already complete") {
			t.Errorf("message = %q", r.Message())
		}
		if after := readClinicianAnswers(t, w, w.ep, nurse); !reflect.DeepEqual(after, before) {
			t.Errorf("clinician row changed after the review: %+v", after)
		}
	})
}

func TestCompleteReview(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	setStatus(t, w, w.ep, "ready_for_review")
	complete := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/complete-review"), nil)
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, complete(w.anon, w.a, w.ep), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, complete(w.site2, w.a, w.ep), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, complete(w.otherClin, w.a, w.ep), http.StatusNotFound)
		refused(t, complete(w.otherClin, w.b, w.ep), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/abc/complete-review"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/"+uuid.NewString()+"/complete-review"), nil), http.StatusNotFound)
	})
	t.Run("before the patient submits", func(t *testing.T) {
		waiting := w.fx.Episode(t, w.hq, w.cara)
		refused(t, complete(w.clinician, w.a, waiting), http.StatusConflict)
		if e := readEpisode(t, w, waiting.ID); e.Status != "hq_not_complete" || e.ReviewCompleted != nil {
			t.Errorf("episode = %+v", e)
		}
	})
	if e := readEpisode(t, w, w.ep.ID); e.Status != "ready_for_review" || e.ReviewCompleted != nil {
		t.Fatalf("refusals changed the episode: %+v", e)
	}
	if ev := readEvents(t, w, w.ep.ID); len(ev) != 0 {
		t.Fatalf("refusals wrote events: %+v", ev)
	}

	t.Run("happy path: Ready for POA, stamped, with an event", func(t *testing.T) {
		complete(w.clinician, w.a, w.ep).Want(t, http.StatusNoContent)
		e := readEpisode(t, w, w.ep.ID)
		if e.Status != "ready_for_poa" || e.ReviewCompleted == nil || e.ReviewCompletedBy == nil || *e.ReviewCompletedBy != w.cara.ID {
			t.Errorf("episode = %+v, want Ready for POA, completed by Cara", e)
		}
		ev := readEvents(t, w, w.ep.ID)
		if len(ev) != 1 || ev[0].Text != "HQ review completed by "+w.cara.Name || ev[0].User == nil || *ev[0].User != w.cara.ID {
			t.Errorf("events = %+v, want 'HQ review completed by %s'", ev, w.cara.Name)
		}
	})
	t.Run("only once", func(t *testing.T) {
		refused(t, complete(w.clinician, w.a, w.ep), http.StatusConflict)
		if ev := readEvents(t, w, w.ep.ID); len(ev) != 1 {
			t.Errorf("events = %+v, want still 1", ev)
		}
	})
}
