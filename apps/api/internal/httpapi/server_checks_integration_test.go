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

// The server's own checks on send and on Complete review (plan C6, session S44; D-1, §10). The
// browser already stops both; these hold for a request sent straight to the API. Each check asks
// the evaluate service (a recording fake here), and if it can't be reached, falls back to today's
// behaviour (the browser's check). No new routes: the five cases of both submit routes and of
// Complete review are in patient_, behalf_ and validation_integration_test.go.

// validateAll stamps a validated clinician's copy of every Question Set of the episode's HQ, as the
// clinician does on Validate, so Complete review's check passes.
func validateAll(t *testing.T, w *clinic, e apitest.Episode) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), `INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data, updated_by, validated_at)
		SELECT f.id, c.id, 'clinician', '{}', $2::uuid, now() FROM episode_forms f JOIN chapters c ON c.version_id = f.version_id
		WHERE f.episode_id = $1 AND f.kind = 'hq'
		ON CONFLICT (episode_form_id, chapter_id, actor) DO UPDATE SET validated_at = now()`, e.ID, w.cara.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSendRefusedWhileASetIsIncomplete(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health := w.hq.Chapters[0], w.hq.Chapters[1]
	incomplete := func(sets ...uuid.UUID) evaluate.Result {
		res := evaluate.Result{Evaluator: evaluate.FakeVersion, Shown: w.hq.Chapters, PatientSets: []evaluate.PatientSet{}}
		for _, s := range []uuid.UUID{about, health} {
			res.PatientSets = append(res.PatientSets, evaluate.PatientSet{ID: s, Complete: !containsID(sets, s)})
		}
		return res
	}
	e := w.fx.Episode(t, w.hq, w.cara)
	p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+about.String()), map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 0}))
	unchanged := func(t *testing.T, e apitest.Episode) {
		t.Helper()
		if got := readEpisode(t, w, e.ID); got.Status != "hq_not_complete" || got.Submitted != nil {
			t.Errorf("episode = %+v, want HQ not complete, not sent", got)
		}
		if ev := readEvents(t, w, e.ID); len(ev) != 0 {
			t.Errorf("events = %+v", ev)
		}
		if rows := formOutputs(t, w, e.FormID); len(rows) != 0 {
			t.Errorf("form_outputs = %+v", rows)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_status_events WHERE episode_id = $1 AND move = 'submit'`, e.ID); n != 0 {
			t.Errorf("submit moves = %d", n)
		}
	}

	t.Run("an incomplete set: 409 naming it, nothing frozen, the status unchanged", func(t *testing.T) {
		w.anon.Evaluator.Reply(incomplete(health))
		r := p.Do("POST", patientPath(e, "/submit"), nil)
		refused(t, r, http.StatusConflict)
		if want := "Please finish “Your health” before sending your answers."; r.Message() != want {
			t.Errorf("message = %q, want %q", r.Message(), want)
		}
		unchanged(t, e)
		// Not frozen: the patient can carry on.
		savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+health.String()), map[string]any{"data": map[string]any{"q_diabetes": "o_no"}, "revision": 0}))
	})
	t.Run("two incomplete sets: both named", func(t *testing.T) {
		w.anon.Evaluator.Reply(incomplete(about, health))
		r := p.Do("POST", patientPath(e, "/submit"), nil)
		refused(t, r, http.StatusConflict)
		if want := "Please finish “About you” and “Your health” before sending your answers."; r.Message() != want {
			t.Errorf("message = %q, want %q", r.Message(), want)
		}
		unchanged(t, e)
	})
	t.Run("staff sending for the patient: refused the same", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		w.anon.Evaluator.Reply(incomplete(about))
		r := w.clinician.Do("POST", epPath(w.a, other, "/patient-form/submit"), nil)
		refused(t, r, http.StatusConflict)
		if want := "Please finish “About you” before sending your answers."; r.Message() != want {
			t.Errorf("message = %q, want %q", r.Message(), want)
		}
		unchanged(t, other)
	})
	t.Run("complete: sent as today, with one evaluator call stored", func(t *testing.T) {
		w.anon.Evaluator.Reply(incomplete())
		before := len(w.anon.Evaluator.Inputs())
		p.Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		if n := len(w.anon.Evaluator.Inputs()) - before; n != 1 {
			t.Errorf("evaluator calls for one send = %d, want 1", n)
		}
		if got := readEpisode(t, w, e.ID); got.Status != "ready_for_review" || got.Submitted == nil {
			t.Errorf("episode = %+v", got)
		}
		if rows := formOutputs(t, w, e.FormID); len(rows) != 1 || rows[0].Error != nil {
			t.Errorf("form_outputs = %+v", rows)
		}
		if ev := readEvents(t, w, e.ID); len(ev) != 1 || ev[0].Text != "HQ completed by the patient" {
			t.Errorf("events = %+v", ev)
		}
	})
	t.Run("a set the patient's form doesn't show isn't asked for", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		// Only About you is shown to this patient (Your health's condition hides it), and it's complete.
		w.anon.Evaluator.Reply(evaluate.Result{Evaluator: evaluate.FakeVersion, PatientSets: []evaluate.PatientSet{{ID: about, Complete: true}}})
		w.clinician.Do("POST", epPath(w.a, other, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
	})
}

func containsID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// The send's fallback when the evaluator fails is TestSubmitWhenTheEvaluatorFails (S41): accepted,
// Ready for review, the note and the log line.

func TestCompleteReviewChecksEverySetShown(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	e := w.fx.Episode(t, w.hq, w.cara)
	if _, err := w.db.Exec(t.Context(), `UPDATE patients SET date_of_birth = '1960-03-04', sex = 'male' WHERE id = $1`, e.PatientID); err != nil {
		t.Fatal(err)
	}
	saveAnswers(t, w, e, about, nil, `{"q_smokes": "o_yes"}`)
	saveAnswers(t, w, e, health, nil, `{"q_diabetes": "o_no"}`)
	setStatus(t, w, e, "ready_for_review")
	// Sent on 1 June 2020: the patient was 60 that day, whatever today is.
	if _, err := w.db.Exec(t.Context(), `UPDATE episode_forms SET submitted_at = '2020-06-01 12:00:00+00' WHERE id = $1`, e.FormID); err != nil {
		t.Fatal(err)
	}
	complete := func() *apitest.Resp { return w.clinician.Do("POST", epPath(w.a, e, "/complete-review"), nil) }
	unchanged := func(t *testing.T) {
		t.Helper()
		if got := readEpisode(t, w, e.ID); got.Status != "ready_for_review" || got.ReviewCompleted != nil {
			t.Errorf("episode = %+v, want Ready for review", got)
		}
		if ev := readEvents(t, w, e.ID); len(ev) != 0 {
			t.Errorf("events = %+v", ev)
		}
	}

	t.Run("no set validated: 409 naming them all", func(t *testing.T) {
		r := complete()
		refused(t, r, http.StatusConflict)
		if want := "Validate every Question Set first: “About you”, “Your health” and “Nurse checks” aren't validated."; r.Message() != want {
			t.Errorf("message = %q, want %q", r.Message(), want)
		}
		unchanged(t)
	})
	t.Run("the evaluator gets every set, the answers that count now and the patient on the day they sent", func(t *testing.T) {
		saveAnswers(t, w, e, about, &w.cara, `{"q_smokes": "o_no"}`) // the clinician's copy, started
		refused(t, complete(), http.StatusConflict)
		calls := w.anon.Evaluator.ShownInputs()
		in := calls[len(calls)-1]
		if in.FormID != e.FormID || len(in.Sets) != 3 || in.Sets[2].ID != nurse || in.Sets[2].Audience != "clinician" ||
			!strings.Contains(string(in.Sets[0].Content), "SECRET nurse note") {
			t.Errorf("input = %+v", in)
		}
		// The clinician's copy where there is one, else the patient's.
		if len(in.Answers) != 2 || !strings.Contains(string(in.Answers[about]), `"o_no"`) || !strings.Contains(string(in.Answers[health]), `"q_diabetes": "o_no"`) {
			t.Errorf("answers = %v", in.Answers)
		}
		if in.Patient != (evaluate.Patient{Age: 60, Sex: "male"}) {
			t.Errorf("patient = %+v, want 60, male", in.Patient)
		}
		if len(w.anon.Evaluator.Inputs()) != 0 {
			t.Error("Complete review stored or re-ran a send's result")
		}
	})
	t.Run("one shown set not validated: 409 naming it", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `UPDATE episode_answers SET validated_at = now() WHERE episode_form_id = $1 AND actor = 'clinician'`, e.FormID); err != nil {
			t.Fatal(err)
		}
		saveAnswers(t, w, e, health, &w.cara, `{"q_diabetes": "o_no"}`) // started, not validated
		r := complete()
		refused(t, r, http.StatusConflict)
		if want := "Validate every Question Set first: “Your health” and “Nurse checks” aren't validated."; r.Message() != want {
			t.Errorf("message = %q, want %q", r.Message(), want)
		}
		if _, err := w.db.Exec(t.Context(), `UPDATE episode_answers SET validated_at = now() WHERE episode_form_id = $1 AND actor = 'clinician'`, e.FormID); err != nil {
			t.Fatal(err)
		}
		r = complete()
		refused(t, r, http.StatusConflict)
		if want := "Validate every Question Set first: “Nurse checks” isn't validated."; r.Message() != want {
			t.Errorf("message = %q, want %q", r.Message(), want)
		}
		unchanged(t)
	})
	t.Run("a set its condition hides doesn't need validating", func(t *testing.T) {
		w.anon.Evaluator.Hide(nurse)
		t.Cleanup(func() { w.anon.Evaluator.Hide() })
		complete().Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, e.ID); got.Status != "ready_for_poa" || got.ReviewCompleted == nil {
			t.Errorf("episode = %+v", got)
		}
		if ev := readEvents(t, w, e.ID); len(ev) != 1 || ev[0].Text != "HQ review completed by "+w.cara.Name {
			t.Errorf("events = %+v", ev)
		}
	})
	t.Run("every shown set validated: as today", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, other, "ready_for_review")
		validateAll(t, w, other)
		w.clinician.Do("POST", epPath(w.a, other, "/complete-review"), nil).Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, other.ID); got.Status != "ready_for_poa" {
			t.Errorf("status = %s", got.Status)
		}
	})
}

// The failing call reads the server's log, so it isn't parallel.
func TestCompleteReviewWhenTheEvaluatorFails(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	saveAnswers(t, w, e, w.hq.Chapters[0], nil, `{"q_smokes": "o_yes", "q_meds": [{"q_med_name": "Wonderdrug"}]}`)
	setStatus(t, w, e, "ready_for_review")
	w.anon.Evaluator.Fail(errors.New("evaluate: /shown: dial tcp 127.0.0.1:8090: connect: connection refused"))

	t.Run("allowed as today, with no set validated", func(t *testing.T) {
		w.clinician.Do("POST", epPath(w.a, e, "/complete-review"), nil).Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, e.ID); got.Status != "ready_for_poa" {
			t.Errorf("status = %s", got.Status)
		}
	})
	t.Run("one EVALUATE_FAILED line, naming the form and no answer", func(t *testing.T) {
		var lines []string
		for _, l := range strings.Split(logged.String(), "\n") {
			if strings.Contains(l, evaluate.FailedMarker) {
				lines = append(lines, l)
			}
		}
		if len(lines) != 1 || !strings.Contains(lines[0], evaluate.FailedMarker+" complete review, form "+e.FormID.String()+":") {
			t.Fatalf("EVALUATE_FAILED lines = %q", lines)
		}
		for _, secret := range []string{"o_yes", "Wonderdrug", "q_smokes"} {
			if strings.Contains(logged.String(), secret) {
				t.Errorf("the log holds an answer (%s)", secret)
			}
		}
	})
	t.Run("nothing stored and no note: only the send's checks write those", func(t *testing.T) {
		if rows := formOutputs(t, w, e.FormID); len(rows) != 0 {
			t.Errorf("form_outputs = %+v", rows)
		}
		if _, all := automaticNotes(t, w, e.ID); all != 0 {
			t.Errorf("checks-didn't-run notes = %d", all)
		}
	})
}

// S44a (Rahul, 27 Oct): on send, the frozen patient answers are only what the patient's form showed
// at that moment. The evaluate service (a fake here) says which answers that is; Go drops the rest,
// with where they were pre-filled from. If the service can't be reached, every answer is kept.
func TestSendKeepsOnlyWhatTheFormShowed(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health := w.hq.Chapters[0], w.hq.Chapters[1]
	type row struct{ data, prefill string }
	patientRows := func(t *testing.T, e apitest.Episode) map[uuid.UUID]row {
		t.Helper()
		rows, err := w.db.Query(t.Context(), `SELECT chapter_id, data::text, coalesce(prefill::text, '') FROM episode_answers
			WHERE episode_form_id = $1 AND actor = 'patient'`, e.FormID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[uuid.UUID]row{}
		for rows.Next() {
			var id uuid.UUID
			var r row
			if err := rows.Scan(&id, &r.data, &r.prefill); err != nil {
				t.Fatal(err)
			}
			out[id] = r
		}
		return out
	}
	// The patient answers both sets; About you's medicines answer was pre-filled from an earlier HQ.
	answer := func(t *testing.T, e apitest.Episode) {
		t.Helper()
		p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
		savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+about.String()), map[string]any{"data": map[string]any{"q_smokes": "o_yes", "q_meds": "Aspirin"}, "revision": 0}))
		savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+health.String()), map[string]any{"data": map[string]any{"q_diabetes": "o_no"}, "revision": 0}))
		if _, err := w.db.Exec(t.Context(), `UPDATE episode_answers SET prefill = CASE chapter_id
				WHEN $2 THEN '{"q_smokes": {"value": "o_yes"}, "q_meds": {"value": "Aspirin"}}'::jsonb ELSE '{"q_diabetes": {"value": "o_no"}}'::jsonb END
			WHERE episode_form_id = $1 AND actor = 'patient'`, e.FormID, about); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("a hidden question's answer and a hidden set's are dropped; what was shown is kept", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		answer(t, e)
		w.anon.Evaluator.HideAnswers(about, "q_meds") // a condition hid the medicines question
		w.anon.Evaluator.HideAnswers(health)          // and the whole of Your health
		t.Cleanup(w.anon.Evaluator.ShowAnswers)
		w.anon.AsPatient(t, e.Link, e.DateOfBirth).Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		got := patientRows(t, e)
		if !sameJSONText(t, json.RawMessage(got[about].data), `{"q_smokes": "o_yes"}`) || !sameJSONText(t, json.RawMessage(got[about].prefill), `{"q_smokes": {"value": "o_yes"}}`) {
			t.Errorf("About you = %+v, want only the smoking answer and its pre-fill", got[about])
		}
		if !sameJSONText(t, json.RawMessage(got[health].data), `{}`) || !sameJSONText(t, json.RawMessage(got[health].prefill), `{}`) {
			t.Errorf("Your health = %+v, want nothing", got[health])
		}
		// Verification 1, finding 6: the stored result keeps no second copy of the answers sent.
		if rows := formOutputs(t, w, e.FormID); len(rows) != 1 || strings.Contains(string(rows[0].Result), `"sent"`) || !strings.Contains(string(rows[0].Result), `"evaluator"`) {
			t.Errorf("form_outputs = %+v, want the result without the answers sent", rows)
		}
		if got := readEpisode(t, w, e.ID); got.Status != "ready_for_review" {
			t.Errorf("status = %s", got.Status)
		}
	})
	t.Run("staff sending for the patient: the same", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		answer(t, e)
		w.anon.Evaluator.HideAnswers(health)
		t.Cleanup(w.anon.Evaluator.ShowAnswers)
		w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		got := patientRows(t, e)
		if !sameJSONText(t, json.RawMessage(got[about].data), `{"q_smokes": "o_yes", "q_meds": "Aspirin"}`) || !sameJSONText(t, json.RawMessage(got[health].data), `{}`) {
			t.Errorf("rows = %+v", got)
		}
	})
	// Test report 1, F1: a set staff filled in whose answers are all dropped holds nothing anyone
	// entered, so it no longer names who entered it; a set that keeps answers still does.
	t.Run("filled in by staff: a set left with no answers names no one who entered them", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		for set, data := range map[uuid.UUID]map[string]any{about: {"q_smokes": "o_yes", "q_meds": "Aspirin"}, health: {"q_diabetes": "o_no"}} {
			savedRevision(t, w.clinician.Do("PUT", epPath(w.a, e, "/patient-form/answers/"+set.String()), map[string]any{"data": data, "revision": 0}))
		}
		w.anon.Evaluator.HideAnswers(about, "q_meds")
		w.anon.Evaluator.HideAnswers(health)
		t.Cleanup(w.anon.Evaluator.ShowAnswers)
		w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		by := func(set uuid.UUID) *uuid.UUID {
			var id *uuid.UUID
			if err := w.db.QueryRow(t.Context(), `SELECT updated_by FROM episode_answers WHERE episode_form_id = $1 AND chapter_id = $2 AND actor = 'patient'`,
				e.FormID, set).Scan(&id); err != nil {
				t.Fatal(err)
			}
			return id
		}
		if got := by(health); got != nil {
			t.Errorf("Your health (no answers left) entered by %v, want no one", *got)
		}
		if got := by(about); got == nil || *got != w.cara.ID {
			t.Errorf("About you (answers kept) entered by %v, want Cara", got)
		}
	})
	t.Run("everything shown: nothing changes", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		answer(t, e)
		before := patientRows(t, e)
		w.anon.AsPatient(t, e.Link, e.DateOfBirth).Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		if got := patientRows(t, e); !sameRows(before, got) {
			t.Errorf("rows = %+v, want %+v", got, before)
		}
	})
	t.Run("the evaluator fails: every answer is kept (S41's fail-safe)", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		answer(t, e)
		before := patientRows(t, e)
		w.anon.Evaluator.HideAnswers(health)
		w.anon.Evaluator.Fail(errors.New("down"))
		t.Cleanup(func() { w.anon.Evaluator.Fail(nil); w.anon.Evaluator.ShowAnswers() })
		w.anon.AsPatient(t, e.Link, e.DateOfBirth).Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		if got := patientRows(t, e); !sameRows(before, got) {
			t.Errorf("rows = %+v, want %+v", got, before)
		}
	})
	t.Run("a reply naming no set keeps the answers (a service from before S44a)", func(t *testing.T) { // last: Reply stays
		e := w.fx.Episode(t, w.hq, w.cara)
		answer(t, e)
		before := patientRows(t, e)
		w.anon.Evaluator.Reply(evaluate.Result{Evaluator: evaluate.FakeVersion, Sent: map[uuid.UUID]json.RawMessage{}})
		w.anon.AsPatient(t, e.Link, e.DateOfBirth).Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		if got := patientRows(t, e); !sameRows(before, got) {
			t.Errorf("rows = %+v, want %+v", got, before)
		}
	})
}

func sameRows[K comparable, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// S44 verification (fixed in S44a): a save that clears a validation waits for a Complete review in
// progress, which holds the episode's row, and is then refused, so the review can't complete over a
// set no longer validated. The save takes the episode's row itself.
func TestClinicianSaveWaitsForCompleteReview(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about := w.hq.Chapters[0]
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, "ready_for_review")
	validateAll(t, w, e)
	// The review completing, holding the episode's row (as completeReview's LockEpisodeStatus does).
	r := meanwhile(t, w.db, `UPDATE episodes SET status = 'ready_for_poa', revision = revision + 1 WHERE id = $1`, []any{e.ID}, func() *apitest.Resp {
		return w.clinician.Do("PUT", epPath(w.a, e, "/answers/"+about.String()), map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 0})
	})
	refused(t, r, http.StatusConflict)
	// S44c (S44a verification 3): the review completed while the save waited, so it says that, not
	// "changed somewhere else" (the status is read again after no row).
	if want := "The HQ review is already complete, so the answers can't be changed."; r.Message() != want {
		t.Errorf("message = %q, want %q", r.Message(), want)
	}
	if n := count(t, w.db, `SELECT count(*) FROM episode_answers a JOIN episode_forms f ON f.id = a.episode_form_id
		WHERE f.episode_id = $1 AND a.actor = 'clinician' AND a.validated_at IS NULL`, e.ID); n != 0 {
		t.Errorf("%d sets lost their validation after the review completed", n)
	}
	// And the other order: a save in progress holds the row, so Complete review waits for it, then
	// sees the set isn't validated.
	e2 := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e2, "ready_for_review")
	validateAll(t, w, e2)
	tx, err := w.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	if _, err := tx.Exec(t.Context(), `UPDATE episode_answers a SET validated_at = NULL FROM episode_forms f
		WHERE f.id = a.episode_form_id AND f.episode_id = $1 AND a.chapter_id = $2 AND a.actor = 'clinician'`, e2.ID, about); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(t.Context(), `SELECT 1 FROM episodes WHERE id = $1 FOR NO KEY UPDATE`, e2.ID); err != nil {
		t.Fatal(err)
	}
	done := make(chan *apitest.Resp, 1)
	go func() { done <- w.clinician.Do("POST", epPath(w.a, e2, "/complete-review"), nil) }()
	time.Sleep(100 * time.Millisecond)
	select {
	case r := <-done:
		t.Fatalf("Complete review didn't wait for the save: %s", r)
	default:
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	r = <-done
	refused(t, r, http.StatusConflict)
	if want := "Validate every Question Set first: “About you” isn't validated."; r.Message() != want {
		t.Errorf("message = %q, want %q", r.Message(), want)
	}
}
