//go:build integration

package httpapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

// The evaluate service on send and on publish (plan C6, session S41; D-1). The server asks the
// evaluate service (a recording fake here) what a sent HQ's answers give, and stores it in
// form_outputs; it acts on nothing yet (S42). If the call fails, the send still works, the error is
// stored and the General notes say the checks didn't run. Publishing asks the service for the
// browser's publish checks, and is refused while it finds problems or can't be reached. No new
// routes: the five cases of both submit routes and publish are in patient_, behalf_ and
// questionnaires_integration_test.go.

// outputRow is a form_outputs row.
type outputRow struct {
	Result  json.RawMessage
	Version *string
	Error   *string
	At      time.Time
}

func formOutputs(t *testing.T, w *clinic, form uuid.UUID) []outputRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT result, evaluator_version, error, created_at FROM form_outputs
		WHERE episode_form_id = $1 ORDER BY id`, form)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []outputRow
	for rows.Next() {
		var o outputRow
		if err := rows.Scan(&o.Result, &o.Version, &o.Error, &o.At); err != nil {
			t.Fatal(err)
		}
		out = append(out, o)
	}
	return out
}

const checksDidntRun = "The automatic checks didn't run. Review this HQ by hand."

// automaticNotes counts the episode's "checks didn't run" lines written as automatic (shown
// "Automatic", like a reminder's; test report 1, F-1), and all of them.
func automaticNotes(t *testing.T, w *clinic, episode uuid.UUID) (automatic, all int) {
	t.Helper()
	all = count(t, w.db, `SELECT count(*) FROM episode_events WHERE episode_id = $1 AND text = $2`, episode, checksDidntRun)
	automatic = count(t, w.db, `SELECT count(*) FROM episode_events WHERE episode_id = $1 AND text = $2
		AND automatic AND user_id IS NULL AND kind = 'event'`, episode, checksDidntRun)
	return automatic, all
}

func TestSubmitStoresWhatTheAnswersGive(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	rules := `{"noGradeAutoTriage": true, "rules": [{"id": "r", "name": "R", "when": {"asa": {"op": ">=", "grade": 1}}, "then": [{"kind": "flag", "name": "Look"}]}], "testCases": []}`
	if _, err := w.db.Exec(t.Context(), `UPDATE questionnaire_versions SET rules = $2::jsonb WHERE id = $1`, w.hq.VersionID, rules); err != nil {
		t.Fatal(err)
	}
	e := w.fx.Episode(t, w.hq, w.cara)
	if _, err := w.db.Exec(t.Context(), `UPDATE patients SET date_of_birth = '1960-03-04', sex = 'female' WHERE id = $1`, e.PatientID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET procedure_id = $2 WHERE id = $1`, e.ID, w.hip); err != nil {
		t.Fatal(err)
	}
	p := w.anon.AsPatient(t, e.Link, "1960-03-04")
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+about.String()), map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 0}))
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+health.String()), map[string]any{"data": map[string]any{"q_diabetes": true}, "revision": 0}))
	result := evaluate.Result{Evaluator: "clinical@abc123", Shown: []uuid.UUID{about, health, nurse},
		SuggestedASA: &evaluate.SuggestedASA{Grade: "II"}, Flags: []evaluate.Flag{}, Rules: []evaluate.FiredRule{}, PatientSets: []evaluate.PatientSet{}}
	result.Raw = json.RawMessage(`{"evaluator": "clinical@abc123", "shown": [], "suggestedAsa": {"grade": "II", "emergency": false}}`)
	w.anon.Evaluator.Reply(result)

	t.Run("happy path: the evaluator gets the version, the patient's answers and the patient context", func(t *testing.T) {
		p.Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		calls := w.anon.Evaluator.Inputs()
		if len(calls) != 1 {
			t.Fatalf("calls = %d", len(calls))
		}
		in := calls[0]
		if in.FormID != e.FormID {
			t.Errorf("form = %s, want %s", in.FormID, e.FormID)
		}
		if len(in.Sets) != 3 || in.Sets[0].ID != about || in.Sets[0].Name != "About you" || in.Sets[0].Audience != "patient" ||
			in.Sets[2].ID != nurse || in.Sets[2].Audience != "clinician" {
			t.Errorf("sets = %+v", in.Sets)
		}
		// Whole: the service decides what counts (a clinician-only score uses the patient's answers).
		if !strings.Contains(string(in.Sets[0].Content), "SECRET nurse note") {
			t.Error("the evaluator got the set without its clinician-only parts")
		}
		if len(in.Answers) != 2 || !strings.Contains(string(in.Answers[about]), `"q_smokes": "o_yes"`) || !strings.Contains(string(in.Answers[health]), `"q_diabetes": true`) {
			t.Errorf("answers = %v", in.Answers)
		}
		if !sameJSONText(t, in.Rules, rules) {
			t.Errorf("rules = %s", in.Rules)
		}
		now := w.anon.Clock.Now()
		age := now.Year() - 1960
		if now.Before(time.Date(now.Year(), 3, 4, 0, 0, 0, 0, now.Location())) {
			age--
		}
		if in.Patient != (evaluate.Patient{Age: age, Sex: "female"}) {
			t.Errorf("patient = %+v, want age %d, female", in.Patient, age)
		}
		if in.ProcedureID == nil || *in.ProcedureID != w.hip {
			t.Errorf("procedure = %v", in.ProcedureID)
		}
	})
	t.Run("the result is stored with its evaluator's version", func(t *testing.T) {
		rows := formOutputs(t, w, e.FormID)
		if len(rows) != 1 || rows[0].Error != nil || rows[0].Version == nil || *rows[0].Version != "clinical@abc123" ||
			!sameJSONText(t, rows[0].Result, string(result.Raw)) {
			t.Fatalf("form_outputs = %+v", rows)
		}
	})
	t.Run("it changes nothing else: Ready for review, as before (S42 acts on it)", func(t *testing.T) {
		if got := readEpisode(t, w, e.ID); got.Status != "ready_for_review" {
			t.Errorf("status = %s", got.Status)
		}
		ev := readEvents(t, w, e.ID)
		if len(ev) != 1 || ev[0].Text != "HQ completed by the patient" {
			t.Errorf("events = %+v", ev)
		}
	})
	t.Run("a refused send doesn't call the evaluator", func(t *testing.T) {
		refused(t, p.Do("POST", patientPath(e, "/submit"), nil), http.StatusConflict)
		if n := len(w.anon.Evaluator.Inputs()); n != 1 {
			t.Errorf("calls = %d", n)
		}
		if rows := formOutputs(t, w, e.FormID); len(rows) != 1 {
			t.Errorf("form_outputs = %d rows", len(rows))
		}
	})
	t.Run("staff sending for the patient: the same", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		w.anon.Evaluator.Reply(evaluate.Result{Evaluator: "clinical@def456"})
		w.clinician.Do("POST", epPath(w.a, other, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		calls := w.anon.Evaluator.Inputs()
		if len(calls) != 2 || calls[1].FormID != other.FormID || len(calls[1].Answers) != 0 || calls[1].ProcedureID != nil {
			t.Fatalf("calls = %+v", calls)
		}
		if in := calls[1]; in.Patient.Sex != "unknown" || len(in.Sets) != 3 {
			t.Errorf("input = %+v", in)
		}
		rows := formOutputs(t, w, other.FormID)
		if len(rows) != 1 || rows[0].Version == nil || *rows[0].Version != "clinical@def456" || rows[0].Error != nil {
			t.Errorf("form_outputs = %+v", rows)
		}
		if got := readEpisode(t, w, other.ID); got.Status != "ready_for_review" {
			t.Errorf("status = %s", got.Status)
		}
	})
}

// The failing call is the one test here that isn't parallel: it reads the server's log.
func TestSubmitWhenTheEvaluatorFails(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+w.hq.Chapters[0].String()), map[string]any{"data": map[string]any{"q_smokes": "o_yes", "q_meds": []any{map[string]any{"q_med_name": "Wonderdrug"}}}, "revision": 0}))
	w.anon.Evaluator.Fail(errors.New("evaluate: /evaluate: dial tcp 127.0.0.1:8090: connect: connection refused"))

	t.Run("the send still works for the patient", func(t *testing.T) {
		p.Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, e.ID); got.Status != "ready_for_review" || got.Submitted == nil {
			t.Errorf("episode = %+v, want Ready for review", got)
		}
	})
	t.Run("the error is stored, with no result", func(t *testing.T) {
		rows := formOutputs(t, w, e.FormID)
		if len(rows) != 1 || rows[0].Result != nil || rows[0].Version != nil || rows[0].Error == nil || !strings.Contains(*rows[0].Error, "connection refused") {
			t.Fatalf("form_outputs = %+v", rows)
		}
	})
	t.Run("the General notes say to review it by hand", func(t *testing.T) {
		ev := readEvents(t, w, e.ID)
		if len(ev) != 2 || ev[0].Text != "HQ completed by the patient" || ev[1].Text != checksDidntRun || ev[1].Kind != "event" || ev[1].User != nil {
			t.Errorf("events = %+v", ev)
		}
		if auto, all := automaticNotes(t, w, e.ID); auto != 1 || all != 1 {
			t.Errorf("the note isn't written as automatic: %d of %d", auto, all)
		}
	})
	t.Run("one EVALUATE_FAILED line, naming the form and no answer", func(t *testing.T) {
		var lines []string
		for _, l := range strings.Split(logged.String(), "\n") {
			if strings.Contains(l, "EVALUATE_FAILED") {
				lines = append(lines, l)
			}
		}
		// The alarm (S45) counts the lines holding the marker, as OPS-05's DELIVERY_FAILED.
		if len(lines) != 1 || !strings.Contains(lines[0], evaluate.FailedMarker+" form "+e.FormID.String()+":") {
			t.Fatalf("EVALUATE_FAILED lines = %q", lines)
		}
		for _, secret := range []string{"o_yes", "Wonderdrug", "q_smokes"} {
			if strings.Contains(logged.String(), secret) {
				t.Errorf("the log holds an answer (%s): %s", secret, logged.String())
			}
		}
	})
	t.Run("staff sending for the patient: the same", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		w.clinician.Do("POST", epPath(w.a, other, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, other.ID); got.Status != "ready_for_review" {
			t.Errorf("status = %s", got.Status)
		}
		if rows := formOutputs(t, w, other.FormID); len(rows) != 1 || rows[0].Error == nil {
			t.Errorf("form_outputs = %+v", rows)
		}
		if ev := readEvents(t, w, other.ID); len(ev) != 2 || ev[1].Text != checksDidntRun {
			t.Errorf("events = %+v", ev)
		}
		// Automatic, not the staff member's nor the patient's (F-1: it showed "Patient").
		if auto, all := automaticNotes(t, w, other.ID); auto != 1 || all != 1 {
			t.Errorf("the note isn't written as automatic: %d of %d", auto, all)
		}
	})
}

func sameJSONText(t *testing.T, a json.RawMessage, b string) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &y); err != nil {
		t.Fatalf("%s: %v", b, err)
	}
	xa, _ := json.Marshal(x)
	ya, _ := json.Marshal(y)
	return bytes.Equal(xa, ya)
}

func TestPublishAsksTheEvaluator(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	publish := func(hq apitest.HQ) *apitest.Resp {
		return w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/publish"), map[string]any{"versionId": hq.VersionID})
	}

	t.Run("happy path: it sends every set with its audience, and the version's rules", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		rules := `{"noGradeAutoTriage": false, "rules": [], "testCases": []}`
		if _, err := w.db.Exec(t.Context(), `UPDATE questionnaire_versions SET rules = $2::jsonb WHERE id = $1`, hq.VersionID, rules); err != nil {
			t.Fatal(err)
		}
		publish(hq).Want(t, http.StatusNoContent)
		calls := w.anon.Evaluator.Published()
		in := calls[len(calls)-1]
		if len(in.Sets) != 3 || in.Sets[0].Name != "About you" || in.Sets[0].Audience != "patient" || in.Sets[2].Audience != "clinician" ||
			!strings.Contains(string(in.Sets[0].Content), "q_smokes") {
			t.Errorf("sets = %+v", in.Sets)
		}
		if !sameJSONText(t, in.Rules, rules) {
			t.Errorf("rules = %s", in.Rules)
		}
		if s := versionStatus(t, w.db, hq.VersionID); s != "published" {
			t.Errorf("status = %s", s)
		}
	})
	// S41 verification (fixed in S44a): a rules save in progress when Publish is pressed is waited
	// for, and what's checked is what's published: the version is re-read inside the transaction.
	t.Run("a rules save in progress: Publish waits for it and checks the rules as saved", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		saved := `{"noGradeAutoTriage": true, "rules": [], "testCases": [{"id": "t", "name": "Saved meanwhile", "patient": {"name": "S"}, "answers": {}, "expected": []}]}`
		r := meanwhile(t, w.db, `UPDATE questionnaire_versions SET rules = $2::jsonb, rules_revision = rules_revision + 1 WHERE id = $1`,
			[]any{hq.VersionID, saved}, func() *apitest.Resp { return publish(hq) })
		r.Want(t, http.StatusNoContent)
		calls := w.anon.Evaluator.Published()
		if in := calls[len(calls)-1]; !sameJSONText(t, in.Rules, saved) {
			t.Errorf("checked rules = %s, want the ones saved meanwhile", in.Rules)
		}
	})
	t.Run("a Question Set save while Publish holds the version waits, then is refused", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		var rev int32
		if err := w.db.QueryRow(t.Context(), `SELECT revision FROM chapters WHERE id = $1`, hq.Chapters[0]).Scan(&rev); err != nil {
			t.Fatal(err)
		}
		r := meanwhile(t, w.db, `UPDATE questionnaire_versions SET status = 'published' WHERE id = $1`, []any{hq.VersionID}, func() *apitest.Resp {
			return w.author.Do("PUT", at(w.a, "/chapters/"+hq.Chapters[0].String()+"/content"),
				map[string]any{"content": map[string]any{"pages": []any{}}, "revision": rev})
		})
		refused(t, r, http.StatusConflict)
		if want := "Only draft versions can be changed."; r.Message() != want { // verification 1, finding 4: not "someone else changed it"
			t.Errorf("message = %q, want %q", r.Message(), want)
		}
		var content string
		if err := w.db.QueryRow(t.Context(), `SELECT content::text FROM chapters WHERE id = $1`, hq.Chapters[0]).Scan(&content); err != nil || !strings.Contains(content, "q_smokes") {
			t.Errorf("the published set's content changed: %s (%v)", content, err)
		}
	})
	// Verification 1, finding 3: every other Question Set change waits for a Publish in progress too.
	t.Run("a set's audience, name, deletion, a new set or a new order while Publish holds the version: wait, then refused", func(t *testing.T) {
		for _, c := range []struct {
			name string
			req  func(hq apitest.HQ) *apitest.Resp
		}{
			{"audience", func(hq apitest.HQ) *apitest.Resp {
				return w.author.Do("PATCH", at(w.a, "/chapters/"+hq.Chapters[2].String()), map[string]any{"audience": "patient"})
			}},
			{"delete", func(hq apitest.HQ) *apitest.Resp { return w.author.Do("DELETE", at(w.a, "/chapters/"+hq.Chapters[2].String()), nil) }},
			{"add", func(hq apitest.HQ) *apitest.Resp {
				return w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/chapters"), map[string]any{"name": "Late set"})
			}},
			{"reorder", func(hq apitest.HQ) *apitest.Resp {
				return w.author.Do("PUT", at(w.a, "/questionnaires/"+hq.ID.String()+"/chapter-order"), []uuid.UUID{hq.Chapters[2], hq.Chapters[1], hq.Chapters[0]})
			}},
		} {
			t.Run(c.name, func(t *testing.T) {
				hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
				r := meanwhile(t, w.db, `UPDATE questionnaire_versions SET status = 'published' WHERE id = $1`, []any{hq.VersionID}, func() *apitest.Resp { return c.req(hq) })
				refused(t, r, http.StatusConflict)
				if want := "Only draft versions can be changed."; r.Message() != want {
					t.Errorf("message = %q, want %q", r.Message(), want)
				}
				var sets, nurse int
				if err := w.db.QueryRow(t.Context(), `SELECT count(*), count(*) FILTER (WHERE audience = 'clinician' AND position = 3) FROM chapters WHERE version_id = $1`,
					hq.VersionID).Scan(&sets, &nurse); err != nil || sets != 3 || nurse != 1 {
					t.Errorf("the published version's sets changed: %d sets, %d clinician set last (%v)", sets, nurse, err)
				}
			})
		}
	})
	// Verification 2, finding 1: a reorder and a content save in progress (which holds the version
	// FOR SHARE while it updates its set) used to deadlock, the reorder's touch waiting on the save's
	// SHARE and the save waiting on the set the reorder had moved. The reorder now waits instead.
	t.Run("a reorder while a Question Set save holds the version: it waits, then works", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		tx, err := w.db.Begin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(t.Context())
		if _, err := tx.Exec(t.Context(), `SELECT 1 FROM questionnaire_versions WHERE id = $1 FOR SHARE`, hq.VersionID); err != nil {
			t.Fatal(err)
		}
		done := make(chan *apitest.Resp, 1)
		go func() {
			done <- w.author.Do("PUT", at(w.a, "/questionnaires/"+hq.ID.String()+"/chapter-order"), []uuid.UUID{hq.Chapters[2], hq.Chapters[1], hq.Chapters[0]})
		}()
		const waiting = `SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`
		for deadline := time.Now().Add(5 * time.Second); count(t, w.db, waiting) == 0; time.Sleep(5 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatal("the reorder never waited for the version")
			}
		}
		// The save's own update of its set, then it commits.
		if _, err := tx.Exec(t.Context(), `UPDATE chapters SET revision = revision + 1 WHERE id = $1`, hq.Chapters[0]); err != nil {
			t.Fatalf("the save's update: %v", err)
		}
		if err := tx.Commit(t.Context()); err != nil {
			t.Fatal(err)
		}
		(<-done).Want(t, http.StatusNoContent)
		var first uuid.UUID
		if err := w.db.QueryRow(t.Context(), `SELECT id FROM chapters WHERE version_id = $1 ORDER BY position LIMIT 1`, hq.VersionID).Scan(&first); err != nil || first != hq.Chapters[2] {
			t.Errorf("first set = %v (%v), want the reordered one", first, err)
		}
	})
	t.Run("a version with no rules sends null", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json") // the same keys, answered the same way
		publish(hq).Want(t, http.StatusNoContent)
		calls := w.anon.Evaluator.Published()
		sent, err := json.Marshal(calls[len(calls)-1]) // as it goes to the service
		if err != nil || !strings.Contains(string(sent), `"rules":null`) {
			t.Errorf("sent %s (%v)", sent, err)
		}
	})
}

// Its own server, as the fake's answers apply to every publish on it.
func TestPublishRefusedByTheEvaluator(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	publish := func(hq apitest.HQ) *apitest.Resp {
		return w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/publish"), map[string]any{"versionId": hq.VersionID})
	}

	t.Run("a failing test case blocks publishing through the API too", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		w.anon.Evaluator.SetProblems(evaluate.Problem{Set: "Rules", Message: "Test case “Heart failure is ASA II” fails"})
		r := publish(hq)
		r.Want(t, http.StatusUnprocessableEntity)
		var got struct {
			Error    string
			Problems []struct{ Set, Message string }
		}
		r.Decode(t, &got)
		if got.Error != "This version can't be published until these problems are fixed." || len(got.Problems) != 1 ||
			got.Problems[0].Set != "Rules" || got.Problems[0].Message != "Test case “Heart failure is ASA II” fails" {
			t.Errorf("refusal = %+v", got)
		}
		if s := versionStatus(t, w.db, hq.VersionID); s != "draft" {
			t.Errorf("status = %s, want still draft", s)
		}
		w.anon.Evaluator.SetProblems()
	})
	t.Run("the evaluator's problems and the key problems are listed together", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		dup := `{"pages": [{"name": "p", "elements": [{"type": "text", "name": "q_dup"}, {"type": "text", "name": "q_dup"}]}]}`
		if _, err := w.db.Exec(t.Context(), `UPDATE chapters SET content = $2::jsonb WHERE id = $1`, hq.Chapters[0], dup); err != nil {
			t.Fatal(err)
		}
		w.anon.Evaluator.SetProblems(evaluate.Problem{Set: "About you", Message: "Test case “X” fails"})
		r := publish(hq)
		r.Want(t, http.StatusUnprocessableEntity)
		var got struct {
			Problems []struct{ Set, Message string }
		}
		r.Decode(t, &got)
		if len(got.Problems) < 2 || got.Problems[0].Message != "Test case “X” fails" {
			t.Errorf("problems = %+v", got.Problems)
		}
		w.anon.Evaluator.SetProblems()
	})
	t.Run("the evaluator can't be reached: refused, nothing published unchecked", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		w.anon.Evaluator.Fail(errors.New("evaluate: /publish-problems: status 502"))
		r := publish(hq)
		r.Want(t, http.StatusServiceUnavailable)
		if msg := r.Message(); msg != "The checks couldn't run. Try again in a minute." {
			t.Errorf("message = %q", msg)
		}
		if s := versionStatus(t, w.db, hq.VersionID); s != "draft" {
			t.Errorf("status = %s, want still draft", s)
		}
		w.anon.Evaluator.Fail(nil)
		publish(hq).Want(t, http.StatusNoContent)
	})
}
