//go:build integration

package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
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
}

func readClinicianAnswers(t *testing.T, w *clinic, e apitest.Episode, chapter uuid.UUID) clinicianRow {
	t.Helper()
	var c clinicianRow
	err := w.db.QueryRow(t.Context(), `SELECT data::text, updated_by, validated_at FROM episode_answers
		WHERE episode_id = $1 AND chapter_id = $2 AND actor = 'clinician'`, e.ID, chapter).Scan(&c.Data, &c.By, &c.Validated)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return c
}

func TestEpisodeHQ(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, nurse := w.hq.Chapters[0], w.hq.Chapters[2]
	setStatus(t, w, w.ep, "ready_for_review")
	saveAnswers(t, w, w.ep, about, nil, `{"q_smokes": "o_yes"}`)
	saveAnswers(t, w, w.ep, about, &w.cara, `{"q_smokes": "o_no"}`)
	get := func(c *apitest.Client, h uuid.UUID) *apitest.Resp { return c.Do("GET", epPath(h, w.ep, "/hq"), nil) }

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
		}
		if len(got.Answers) != 2 || actors["patient"] != `{"q_smokes":"o_yes"}` || actors["clinician"] != `{"q_smokes":"o_no"}` {
			t.Errorf("answers = %+v, want the patient's and the clinician's", got.Answers)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, get(w.anon, w.a), http.StatusUnauthorized)
	})
	t.Run("wrong role: an author", func(t *testing.T) {
		refused(t, get(w.author, w.a), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
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
	about, nurse := w.hq.Chapters[0], w.hq.Chapters[2]
	setStatus(t, w, w.ep, "ready_for_review")
	const patientSaid = `{"q_heart": true, "q_smokes": "o_yes"}`
	saveAnswers(t, w, w.ep, about, nil, patientSaid)
	save := func(c *apitest.Client, h uuid.UUID, e apitest.Episode, chapter uuid.UUID, body any) *apitest.Resp {
		return c.Do("PUT", epPath(h, e, "/answers/"+chapter.String()), body)
	}
	// The browser pre-fills from the patient's answers; the clinician corrects one.
	corrected := map[string]any{"data": map[string]any{"q_smokes": "o_no", "q_heart": true}}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, save(w.anon, w.a, w.ep, about, corrected), http.StatusUnauthorized)
	})
	t.Run("wrong role: an author", func(t *testing.T) {
		refused(t, save(w.author, w.a, w.ep, about, corrected), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
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
		refused(t, save(w.clinician, w.a, w.ep, about, `{"data": {}, "validated": "yes"}`), http.StatusBadRequest)
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

	t.Run("happy path: the first write makes the clinician's own row", func(t *testing.T) {
		save(w.clinician, w.a, w.ep, about, corrected).Want(t, http.StatusNoContent)
		c := readClinicianAnswers(t, w, w.ep, about)
		if c.Data != `{"q_heart": true, "q_smokes": "o_no"}` || c.By == nil || *c.By != w.cara.ID || c.Validated != nil {
			t.Errorf("clinician row = %+v, want the corrected answers by Cara, not validated", c)
		}
		if got := patientAnswers(t, w, w.ep, about); got != patientSaid {
			t.Errorf("the patient's answers changed: %s", got)
		}
	})
	t.Run("happy path: validating stamps the set", func(t *testing.T) {
		start := time.Now().Add(-time.Minute)
		save(w.clinician, w.a, w.ep, about, map[string]any{"data": corrected["data"], "validated": true}).Want(t, http.StatusNoContent)
		c := readClinicianAnswers(t, w, w.ep, about)
		if c.Validated == nil || c.Validated.Before(start) || c.By == nil || *c.By != w.cara.ID {
			t.Errorf("clinician row = %+v, want validated now by Cara", c)
		}
	})
	t.Run("happy path: a clinician Question Set, which has no patient row", func(t *testing.T) {
		save(w.clinician, w.a, w.ep, nurse, map[string]any{"data": map[string]string{"q_secret_bp": "120/80"}, "validated": true}).Want(t, http.StatusNoContent)
		if c := readClinicianAnswers(t, w, w.ep, nurse); c.Validated == nil {
			t.Errorf("clinician set = %+v, want validated", c)
		}
	})
	t.Run("a later save without validated clears the stamp", func(t *testing.T) {
		save(w.clinician, w.a, w.ep, about, corrected).Want(t, http.StatusNoContent)
		if c := readClinicianAnswers(t, w, w.ep, about); c.Validated != nil {
			t.Errorf("clinician row = %+v, want the stamp cleared", c)
		}
	})
	t.Run("read-only once the review is complete", func(t *testing.T) {
		before := readClinicianAnswers(t, w, w.ep, nurse)
		if before.Validated == nil {
			t.Fatal("the clinician set wasn't validated by the earlier case")
		}
		setStatus(t, w, w.ep, "ready_for_poa")
		r := save(w.clinician, w.a, w.ep, nurse, map[string]any{"data": map[string]string{"q_secret_bp": "changed"}, "validated": true})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "already complete") {
			t.Errorf("message = %q", r.Message())
		}
		if after := readClinicianAnswers(t, w, w.ep, nurse); after.Data != before.Data || after.Validated == nil || !after.Validated.Equal(*before.Validated) {
			t.Errorf("clinician row changed after the review: %+v", after)
		}
	})
}

func TestCompleteReview(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	setStatus(t, w, w.ep, "ready_for_review")
	complete := func(c *apitest.Client, h uuid.UUID, e apitest.Episode) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/complete-review"), nil)
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, complete(w.anon, w.a, w.ep), http.StatusUnauthorized)
	})
	t.Run("wrong role: an author", func(t *testing.T) {
		refused(t, complete(w.author, w.a, w.ep), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
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
