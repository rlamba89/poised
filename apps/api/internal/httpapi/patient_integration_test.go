//go:build integration

package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The patient's link has no sign-in: the token is the key. So the five cases become:
//   - happy path: with no sign-in;
//   - "no sign-in": nothing to refuse; a signed-in clinician using the link gets no more than
//     the patient does (the "wrong role" case below);
//   - another hospital: a link can't reach another episode's Question Sets or answers;
//   - invalid input: a bad or unknown token is 404, and so is a Question Set the patient doesn't get.

func patientPath(e apitest.Episode, rest string) string { return "/api/p/" + e.Token + rest }

// patientAnswers is the patient's answers row for a Question Set, or "" if there isn't one.
func patientAnswers(t *testing.T, w *clinic, e apitest.Episode, chapter uuid.UUID) string {
	t.Helper()
	var data string
	err := w.db.QueryRow(t.Context(), `SELECT data::text FROM episode_answers
		WHERE episode_id = $1 AND chapter_id = $2 AND actor = 'patient'`, e.ID, chapter).Scan(&data)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return data
}

func TestPatientHQ(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	saveAnswers(t, w, w.ep, about, nil, `{"q_smokes": "o_yes"}`)
	saveAnswers(t, w, w.ep, nurse, &w.cara, `{"q_secret_bp": "SECRET 120/80"}`)
	other := w.fx.Episode(t, w.hq, w.cara)
	saveAnswers(t, w, other, health, nil, `{"q_diabetes": true}`)

	type hq struct {
		Patient           struct{ FirstName, LastName string }
		QuestionnaireName string
		Submitted         bool
		Chapters          []struct {
			ID      uuid.UUID
			Name    string
			Content json.RawMessage
		}
		Answers map[uuid.UUID]json.RawMessage
	}
	check := func(t *testing.T, r *apitest.Resp) hq {
		t.Helper()
		r.Want(t, http.StatusOK)
		// Nothing for clinicians reaches the patient (VEW-01): no clinician-only element, page or
		// set, no disclosure (clinicalOutputs), no test case, and no clinician answers.
		for _, secret := range []string{"SECRET", "clinicianOnly", "clinicalOutputs", "testCases", "q_secret", "q_on_secret_page"} {
			if strings.Contains(string(r.Body), secret) {
				t.Errorf("the patient JSON has %q: %s", secret, r.Body)
			}
		}
		var got hq
		r.Decode(t, &got)
		return got
	}

	t.Run("happy path: the patient's Question Sets, stripped, and their answers", func(t *testing.T) {
		got := check(t, w.anon.Do("GET", patientPath(w.ep, ""), nil))
		if len(got.Chapters) != 2 || got.Chapters[0].ID != about || got.Chapters[1].ID != health {
			t.Fatalf("chapters = %+v, want the 2 patient sets in order", got.Chapters)
		}
		content := string(got.Chapters[0].Content)
		for _, kept := range []string{"q_smokes", "g_heart", "q_heart", "q_meds", "q_med_name", "Yes"} {
			if !strings.Contains(content, kept) {
				t.Errorf("patient content lost %q: %s", kept, content)
			}
		}
		if got.QuestionnaireName != w.hq.Name || got.Patient.FirstName != "Test" || got.Submitted {
			t.Errorf("hq = %+v", got)
		}
		if len(got.Answers) != 1 || string(got.Answers[about]) != `{"q_smokes":"o_yes"}` {
			t.Errorf("answers = %s, want only the patient's own answer", got.Answers)
		}
	})
	t.Run("wrong role: a signed-in clinician gets only what the patient gets", func(t *testing.T) {
		got := check(t, w.clinician.Do("GET", patientPath(w.ep, ""), nil))
		if len(got.Chapters) != 2 {
			t.Errorf("chapters = %d, want 2", len(got.Chapters))
		}
	})
	t.Run("another episode: its answers aren't shown", func(t *testing.T) {
		got := check(t, w.anon.Do("GET", patientPath(other, ""), nil))
		if len(got.Answers) != 1 || got.Answers[health] == nil {
			t.Errorf("answers = %s, want only this episode's", got.Answers)
		}
	})
	t.Run("invalid input: a bad token is 404", func(t *testing.T) {
		refused(t, w.anon.Do("GET", "/api/p/short", nil), http.StatusNotFound)
		refused(t, w.anon.Do("GET", "/api/p/"+strings.Repeat("A", 43), nil), http.StatusNotFound) // well formed, unknown
		refused(t, w.anon.Do("GET", "/api/p/"+strings.Repeat("!", 43), nil), http.StatusNotFound)
	})
}

func TestSavePatientAnswers(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	save := func(c *apitest.Client, e apitest.Episode, chapter uuid.UUID, body any) *apitest.Resp {
		return c.Do("PUT", patientPath(e, "/answers/"+chapter.String()), body)
	}
	answers := map[string]any{"data": map[string]any{"q_smokes": "o_no", "q_heart": true}}

	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.anon.Do("PUT", "/api/p/short/answers/"+about.String(), answers), http.StatusNotFound)
		refused(t, w.anon.Do("PUT", "/api/p/"+strings.Repeat("A", 43)+"/answers/"+about.String(), answers), http.StatusNotFound)
		refused(t, w.anon.Do("PUT", patientPath(w.ep, "/answers/abc"), answers), http.StatusNotFound)
		refused(t, save(w.anon, w.ep, uuid.New(), answers), http.StatusNotFound)
		refused(t, save(w.anon, w.ep, about, `{"data":`), http.StatusBadRequest)
		refused(t, save(w.anon, w.ep, about, `{"data": ["q_smokes"]}`), http.StatusBadRequest)
	})
	t.Run("wrong role: the clinician Question Set is not the patient's", func(t *testing.T) {
		refused(t, save(w.anon, w.ep, nurse, map[string]any{"data": map[string]string{"q_secret_bp": "1"}}), http.StatusNotFound)
	})
	t.Run("another hospital: a Question Set of another HQ", func(t *testing.T) {
		theirs := w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")
		refused(t, save(w.anon, w.ep, theirs.Chapters[0], answers), http.StatusNotFound)
	})
	if n := count(t, w.db, "SELECT count(*) FROM episode_answers"); n != 0 {
		t.Fatalf("refusals saved %d answer rows", n)
	}

	t.Run("happy path: saved, and saved again on the next autosave", func(t *testing.T) {
		save(w.anon, w.ep, about, answers).Want(t, http.StatusNoContent)
		save(w.anon, w.ep, about, map[string]any{"data": map[string]any{"q_smokes": "o_yes"}}).Want(t, http.StatusNoContent)
		if got := patientAnswers(t, w, w.ep, about); got != `{"q_smokes": "o_yes"}` {
			t.Errorf("answers = %s, want the second save", got)
		}
	})
	t.Run("answers to questions the patient doesn't get are dropped", func(t *testing.T) {
		save(w.anon, w.ep, about, map[string]any{"data": map[string]any{
			"q_smokes": "o_yes", "q_smokes-Comment": "Pipe", "q_heart": false,
			"q_secret_note": "injected", "q_secret_ecg": "injected", "q_on_secret_page": "injected", "q_made_up": 1,
		}}).Want(t, http.StatusNoContent)
		var got map[string]any
		if err := json.Unmarshal([]byte(patientAnswers(t, w, w.ep, about)), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 || got["q_smokes"] != "o_yes" || got["q_smokes-Comment"] != "Pipe" || got["q_heart"] != false {
			t.Errorf("answers = %v, want q_smokes, q_smokes-Comment and q_heart only", got)
		}
	})
	t.Run("frozen after submit", func(t *testing.T) {
		w.anon.Do("POST", patientPath(w.ep, "/submit"), nil).Want(t, http.StatusNoContent)
		refused(t, save(w.anon, w.ep, about, answers), http.StatusConflict)
		refused(t, save(w.anon, w.ep, health, map[string]any{"data": map[string]bool{"q_diabetes": true}}), http.StatusConflict)
		if got := patientAnswers(t, w, w.ep, health); got != "" {
			t.Errorf("a Question Set was saved after submit: %s", got)
		}
		if got := patientAnswers(t, w, w.ep, about); !strings.Contains(got, `"q_smokes-Comment": "Pipe"`) {
			t.Errorf("answers changed after submit: %s", got)
		}
	})
}

func TestSubmitPatientHQ(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	other := w.fx.Episode(t, w.hq, w.cara)
	submit := func(e apitest.Episode) *apitest.Resp { return w.anon.Do("POST", patientPath(e, "/submit"), nil) }

	t.Run("invalid input: a bad token is 404", func(t *testing.T) {
		refused(t, w.anon.Do("POST", "/api/p/short/submit", nil), http.StatusNotFound)
		refused(t, w.anon.Do("POST", "/api/p/"+strings.Repeat("A", 43)+"/submit", nil), http.StatusNotFound)
	})
	if e := readEpisode(t, w, w.ep.ID); e.Status != "hq_not_complete" || e.Submitted != nil {
		t.Fatalf("refusals changed the episode: %+v", e)
	}
	t.Run("happy path: Ready for review, with an event from the patient", func(t *testing.T) {
		submit(w.ep).Want(t, http.StatusNoContent)
		e := readEpisode(t, w, w.ep.ID)
		if e.Status != "ready_for_review" || e.Submitted == nil {
			t.Errorf("episode = %+v, want Ready for review with the submit time", e)
		}
		ev := readEvents(t, w, w.ep.ID)
		if len(ev) != 1 || ev[0].Kind != "event" || ev[0].Text != "HQ completed by the patient" || ev[0].User != nil {
			t.Errorf("events = %+v, want 'HQ completed by the patient' with no user", ev)
		}
	})
	t.Run("wrong role: a signed-in clinician using the link submits as the patient", func(t *testing.T) {
		viaStaff := w.fx.Episode(t, w.hq, w.cara)
		w.clinician.Do("POST", patientPath(viaStaff, "/submit"), nil).Want(t, http.StatusNoContent)
		ev := readEvents(t, w, viaStaff.ID)
		if len(ev) != 1 || ev[0].Text != "HQ completed by the patient" || ev[0].User != nil {
			t.Errorf("events = %+v, want 'HQ completed by the patient' with no user", ev)
		}
	})
	t.Run("another episode isn't touched", func(t *testing.T) {
		if e := readEpisode(t, w, other.ID); e.Status != "hq_not_complete" || e.Submitted != nil {
			t.Errorf("the other episode = %+v", e)
		}
	})
	t.Run("a second submit is refused and writes nothing", func(t *testing.T) {
		refused(t, submit(w.ep), http.StatusConflict)
		if ev := readEvents(t, w, w.ep.ID); len(ev) != 1 {
			t.Errorf("events = %+v, want still 1", ev)
		}
	})
	t.Run("the link still opens, as submitted", func(t *testing.T) {
		r := w.anon.Do("GET", patientPath(w.ep, ""), nil)
		r.Want(t, http.StatusOK)
		var got struct{ Submitted bool }
		r.Decode(t, &got)
		if !got.Submitted {
			t.Error("submitted = false after submit")
		}
	})
}
