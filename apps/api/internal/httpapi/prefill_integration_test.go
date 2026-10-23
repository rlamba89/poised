//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// Pre-fill (S25; PRE-01–03, PRE-06): opening an HQ form fills each patient Question Set with no
// answers yet from the patient's earlier submitted HQs in the same trust, and stores it as the
// patient's answers. The fixtures: workflow.json is the earlier HQ, prefill.json the later one.

// prefillWorld is the clinic plus trust A's later HQ (prefill.json).
type prefillWorld struct {
	*clinic
	later apitest.HQ
}

func newPrefillWorld(t *testing.T) *prefillWorld {
	t.Helper()
	w := &prefillWorld{clinic: newClinic(t)}
	w.later = w.fx.PublishedHQ(t, w.a, w.alex, "prefill.json")
	return w
}

// answered gives episode e (on the Workflow HQ) the patient's answers, sends it, and saves the
// clinician's validated copy of About you and Nurse checks, as a reviewed episode would have.
// It returns the form's submit time.
func (w *prefillWorld) answered(t *testing.T, e apitest.Episode, smokes string) time.Time {
	t.Helper()
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	p := w.anon.NewBrowser(t).AsPatient(t, e.Link, e.DateOfBirth)
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+about.String()), map[string]any{
		"data": map[string]any{"q_smokes": "o_no", "q_heart": true, "q_meds": []any{map[string]any{"q_med_name": "Aspirin"}}}, "revision": 0}))
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+health.String()), map[string]any{
		"data": map[string]any{"q_diabetes": false}, "revision": 0}))
	p.Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
	savedRevision(t, w.clinician.Do("PUT", epPath(w.a, e, "/answers/"+about.String()), map[string]any{"data": map[string]any{
		"q_smokes": smokes, "q_heart": true, "q_secret_note": "SECRET note",
		"q_meds": []any{map[string]any{"q_med_name": "Aspirin", "q_secret_dose_check": "SECRET ok"}}}, "validated": true, "revision": 0}))
	savedRevision(t, w.clinician.Do("PUT", epPath(w.a, e, "/answers/"+nurse.String()), map[string]any{
		"data": map[string]any{"q_secret_bp": "120/80"}, "validated": true, "revision": 0}))
	var at time.Time
	if err := w.db.QueryRow(t.Context(), `SELECT submitted_at FROM episode_forms WHERE id = $1`, e.FormID).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// prefillForm is the patient's HQ as GET returns it, the parts pre-fill adds.
type prefillForm struct {
	Answers   map[string]map[string]json.RawMessage
	Revisions map[string]int
	Prefill   map[string]struct {
		Answers map[string]struct {
			FromForm   uuid.UUID
			AnsweredAt time.Time
			Value      json.RawMessage
			MatchedBy  string
		}
		Choice   *string
		ChoiceAt *time.Time
	}
}

func getForm(t *testing.T, c *apitest.Client, path string) prefillForm {
	t.Helper()
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	var f prefillForm
	r.Decode(t, &f)
	return f
}

// homeState is the patient's home's state for an episode (to_do, in_progress…).
func homeState(t *testing.T, c *apitest.Client, episode uuid.UUID) string {
	t.Helper()
	r := c.Do("GET", "/api/p/home", nil)
	r.Want(t, http.StatusOK)
	var home struct {
		Episodes []struct {
			ID    uuid.UUID
			State string
		}
	}
	r.Decode(t, &home)
	for _, e := range home.Episodes {
		if e.ID == episode {
			return e.State
		}
	}
	return ""
}

func answersJSON(t *testing.T, m map[string]json.RawMessage) string {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPrefill(t *testing.T) {
	t.Parallel()
	w := newPrefillWorld(t)
	again, healthAgain, clinic := w.later.Chapters[0], w.later.Chapters[1], w.later.Chapters[2]
	first := w.ep
	submitted := w.answered(t, first, "o_yes") // the clinician changed "No" to "Yes"
	later := w.fx.EpisodeOf(t, w.later, w.a, first.PatientID, w.cara, "")
	patient := w.anon.NewBrowser(t).AsPatient(t, later.Link, later.DateOfBirth)

	t.Run("happy path: the same trust's answers, the clinician's copy first, by key and by wording", func(t *testing.T) {
		f := getForm(t, patient, patientPath(later, ""))
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Aspirin"}],"q_smokes":"o_yes"}` {
			t.Errorf("About you again %s", got) // not q_heart (Always ask fresh), nor the clinician-only note or dose check
		}
		if got := answersJSON(t, f.Answers[healthAgain.String()]); got != `{"pf_diabetes":false}` {
			t.Errorf("Health again %s", got) // not the nurse's blood pressure
		}
		if _, ok := f.Answers[clinic.String()]; ok {
			t.Error("the clinician set was sent to the patient")
		}
		if f.Revisions[again.String()] != 1 || f.Revisions[healthAgain.String()] != 1 { // S26: a pre-filled set starts at 1
			t.Errorf("revisions %v", f.Revisions)
		}
		smokes := f.Prefill[again.String()].Answers["q_smokes"]
		if smokes.FromForm != first.FormID || !smokes.AnsweredAt.Equal(submitted) || string(smokes.Value) != `"o_yes"` || smokes.MatchedBy != "key" {
			t.Errorf("q_smokes %+v", smokes)
		}
		if d := f.Prefill[healthAgain.String()].Answers["pf_diabetes"]; d.MatchedBy != "wording" || string(d.Value) != "false" {
			t.Errorf("pf_diabetes %+v", d)
		}
		if len(f.Prefill[again.String()].Answers) != 2 || f.Prefill[again.String()].Choice != nil {
			t.Errorf("prefill %+v", f.Prefill[again.String()])
		}
		// Stored as the patient's answers, revision 1 (S26), with where each came from.
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_form_id = $1 AND actor = 'patient' AND revision = 1
			AND prefill ? 'q_smokes' AND updated_by IS NULL`, later.FormID); n != 1 {
			t.Errorf("%d pre-filled rows with q_smokes", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_form_id = $1 AND chapter_id = $2`, later.FormID, clinic); n != 0 {
			t.Error("the clinician set was pre-filled")
		}
		// A pre-filled set doesn't count as started: reminders keep going, and the home says To do.
		if got := homeState(t, patient, later.ID); got != "to_do" {
			t.Errorf("home state %q", got)
		}
		if f, err := db.New(w.db).GetFollowup(t.Context(), later.ID); err != nil || f.Started {
			t.Errorf("reminders see it started: %v %v", f.Started, err)
		}
	})

	t.Run("a second open doesn't pre-fill again, and nothing the patient typed is overwritten", func(t *testing.T) {
		savedRevision(t, patient.Do("PUT", patientPath(later, "/answers/"+again.String()), map[string]any{
			"data": map[string]any{"q_smokes": "o_no", "pf_new": "Nothing"}, "revision": 1}))
		// Even with a newer earlier form that would fill it differently.
		newer := w.fx.EpisodeOf(t, w.hq, w.a, first.PatientID, w.cara, "")
		w.answered(t, newer, "o_yes")
		f := getForm(t, patient, patientPath(later, ""))
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"pf_new":"Nothing","q_smokes":"o_no"}` {
			t.Errorf("About you again %s", got)
		}
		if f.Revisions[again.String()] != 2 {
			t.Errorf("revision %d", f.Revisions[again.String()])
		}
		// The prefill map stays as it was, so the change shows.
		if v := f.Prefill[again.String()].Answers["q_smokes"]; string(v.Value) != `"o_yes"` || v.FromForm != first.FormID {
			t.Errorf("q_smokes %+v", v)
		}
		if got := homeState(t, patient, later.ID); got != "in_progress" {
			t.Errorf("home state %q", got)
		}
	})

	t.Run("two opens at once: the second sends what the first stored", func(t *testing.T) {
		e := w.fx.EpisodeOf(t, w.later, w.a, first.PatientID, w.cara, "")
		p := w.anon.NewBrowser(t).AsPatient(t, e.Link, e.DateOfBirth)
		// The first open's rows are written, not yet committed, when the second reads none.
		r := meanwhile(t, w.db, `INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data, prefill, revision)
			VALUES ($1, $2, 'patient', '{"q_smokes":"o_yes"}', '{"q_smokes":{"value":"o_yes"}}', 1)`, []any{e.FormID, again},
			func() *apitest.Resp { return p.Do("GET", patientPath(e, ""), nil) })
		r.Want(t, http.StatusOK)
		var f prefillForm
		r.Decode(t, &f)
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_smokes":"o_yes"}` {
			t.Errorf("About you again %s, want the first open's", got)
		}
	})

	t.Run("a set answered before the first open isn't pre-filled; the others are", func(t *testing.T) {
		e := w.fx.EpisodeOf(t, w.later, w.a, first.PatientID, w.cara, "")
		p := w.anon.NewBrowser(t).AsPatient(t, e.Link, e.DateOfBirth)
		savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+healthAgain.String()), map[string]any{"data": map[string]any{"q_secret_bp": "130/85"}, "revision": 0}))
		f := getForm(t, p, patientPath(e, ""))
		if got := answersJSON(t, f.Answers[healthAgain.String()]); got != `{"q_secret_bp":"130/85"}` {
			t.Errorf("Health again %s", got)
		}
		if _, ok := f.Prefill[healthAgain.String()]; ok {
			t.Error("prefill on a set the patient answered")
		}
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Aspirin"}],"q_smokes":"o_yes"}` {
			t.Errorf("About you again %s", got)
		}
	})

	t.Run("the newest submitted form first, whatever its episode's status", func(t *testing.T) {
		p := w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "New", LastName: "Est", DateOfBirth: "1981-02-03"})
		older := w.fx.EpisodeOf(t, w.hq, w.a, p, w.cara, "")
		w.answered(t, older, "o_no")
		newest := w.fx.EpisodeOf(t, w.hq, w.a, p, w.cara, "")
		w.answered(t, newest, "o_yes")
		// Cancelled after review: answers never expire (A-9).
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET status = 'cancelled' WHERE id = $1`, newest.ID); err != nil {
			t.Fatal(err)
		}
		e := w.fx.EpisodeOf(t, w.later, w.a, p, w.cara, "")
		f := getForm(t, w.anon.NewBrowser(t).AsPatient(t, e.Link, e.DateOfBirth), patientPath(e, ""))
		if v := f.Prefill[again.String()].Answers["q_smokes"]; string(v.Value) != `"o_yes"` || v.FromForm != newest.FormID {
			t.Errorf("q_smokes %+v", v)
		}
		// The earlier forms' episodes are in the open's audit row: their answers were read.
		var episodes []uuid.UUID
		if err := w.db.QueryRow(t.Context(), `SELECT episode_ids FROM audit_log WHERE action = 'link.view' ORDER BY id DESC LIMIT 1`).Scan(&episodes); err != nil {
			t.Fatal(err)
		}
		if len(episodes) != 3 {
			t.Errorf("audit episodes %v, want this one and the two earlier", episodes)
		}
	})

	t.Run("records joined to the patient's account count; without an account, only the same record", func(t *testing.T) {
		other := w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Twin", LastName: "Record", DateOfBirth: "1970-01-01"})
		earlier := w.fx.EpisodeOf(t, w.hq, w.a, other, w.cara, "")
		w.answered(t, earlier, "o_yes")
		mine := w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Twin", LastName: "Record", DateOfBirth: "1970-01-01"})
		unlinked := w.fx.EpisodeOf(t, w.later, w.a, mine, w.cara, "")
		p := w.anon.NewBrowser(t).AsPatient(t, unlinked.Link, unlinked.DateOfBirth) // mine joins an account; other doesn't
		f := getForm(t, p, patientPath(unlinked, ""))
		if len(f.Prefill) != 0 {
			t.Errorf("pre-filled from a record not joined: %+v", f.Prefill)
		}
		if _, err := w.db.Exec(t.Context(), `UPDATE patients SET account_id = (SELECT account_id FROM patients WHERE id = $1) WHERE id = $2`, mine, other); err != nil {
			t.Fatal(err)
		}
		joined := w.fx.EpisodeOf(t, w.later, w.a, mine, w.cara, "")
		f = getForm(t, p, patientPath(joined, ""))
		if v := f.Prefill[again.String()].Answers["q_smokes"]; v.FromForm != earlier.FormID {
			t.Errorf("q_smokes %+v, want from the joined record's form", v)
		}
	})

	t.Run("another trust's answers are never used", func(t *testing.T) {
		bHQ := w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")
		inB := w.fx.EpisodeOf(t, bHQ, w.b, w.fx.Patient(t, w.b.Org, apitest.PatientRecord{FirstName: "Cross", LastName: "Trust", DateOfBirth: "1960-06-06"}), w.bea, "")
		pb := w.anon.NewBrowser(t).AsPatient(t, inB.Link, inB.DateOfBirth)
		savedRevision(t, pb.Do("PUT", patientPath(inB, "/answers/"+bHQ.Chapters[0].String()), map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 0}))
		pb.Do("POST", patientPath(inB, "/submit"), nil).Want(t, http.StatusNoContent)
		inA := w.fx.EpisodeOf(t, w.later, w.a, w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Cross", LastName: "Trust", DateOfBirth: "1960-06-06"}), w.cara, "")
		// The same person's account holds both trusts' records (S15).
		if _, err := w.db.Exec(t.Context(), `UPDATE patients SET account_id = (SELECT account_id FROM patients WHERE id = $1) WHERE id = $2`, inB.PatientID, inA.PatientID); err != nil {
			t.Fatal(err)
		}
		f := getForm(t, pb, patientPath(inA, ""))
		if len(f.Prefill) != 0 || len(f.Answers) != 0 {
			t.Errorf("pre-filled from trust B: %+v %+v", f.Prefill, f.Answers)
		}
	})

	t.Run("staff filling in for the patient get the same pre-fill", func(t *testing.T) {
		e := w.fx.EpisodeOf(t, w.later, w.a, first.PatientID, w.cara, "")
		f := getForm(t, w.clinician, epPath(w.a, e, "/patient-form"))
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Aspirin"}],"q_smokes":"o_yes"}` {
			t.Errorf("About you again %s", got)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_form_id = $1 AND updated_by IS NULL AND revision = 1`, e.FormID); n != 2 {
			t.Errorf("%d pre-filled rows", n)
		}
	})

	t.Run("a submitted form is never pre-filled", func(t *testing.T) {
		e := w.fx.EpisodeOf(t, w.later, w.a, first.PatientID, w.cara, "")
		if _, err := w.db.Exec(t.Context(), `UPDATE episode_forms SET status = 'submitted', submitted_at = now() WHERE id = $1`, e.FormID); err != nil {
			t.Fatal(err)
		}
		getForm(t, w.clinician, epPath(w.a, e, "/patient-form"))
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_form_id = $1`, e.FormID); n != 0 {
			t.Errorf("%d rows", n)
		}
	})
}

// The opening card's button (PRE-02, S25 answers 5): POST …/prefill/{cid}, by the patient or by
// staff for them, recorded once per pre-filled set.
func TestPrefillChoice(t *testing.T) {
	t.Parallel()
	w := newPrefillWorld(t)
	again, healthAgain, clinic := w.later.Chapters[0], w.later.Chapters[1], w.later.Chapters[2]
	w.answered(t, w.ep, "o_yes")
	e := w.fx.EpisodeOf(t, w.later, w.a, w.ep.PatientID, w.cara, "")
	patient := w.anon.NewBrowser(t).AsPatient(t, e.Link, e.DateOfBirth)
	getForm(t, patient, patientPath(e, "")) // pre-fills both sets
	still := map[string]string{"choice": "still_correct"}
	check := map[string]string{"choice": "check_each"}
	choice := func(set uuid.UUID) (string, *uuid.UUID, bool) {
		t.Helper()
		var c *string
		var by *uuid.UUID
		var at *time.Time
		if err := w.db.QueryRow(t.Context(), `SELECT prefill_choice, prefill_choice_by, prefill_choice_at FROM episode_answers
			WHERE episode_form_id = $1 AND chapter_id = $2 AND actor = 'patient'`, e.FormID, set).Scan(&c, &by, &at); err != nil {
			t.Fatal(err)
		}
		if c == nil {
			return "", by, at != nil
		}
		return *c, by, at != nil
	}
	staffPath := func(rest string) string { return epPath(w.a, e, "/patient-form/prefill/"+rest) }

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("POST", patientPath(e, "/prefill/"+again.String()), still), http.StatusUnauthorized)
		refused(t, w.anon.Do("POST", staffPath(again.String()), still), http.StatusUnauthorized)
	})
	t.Run("wrong role: another patient's session, and staff at the trust's other hospital, get 404", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		someoneElse := w.anon.NewBrowser(t).AsPatient(t, other.Link, other.DateOfBirth)
		refused(t, someoneElse.Do("POST", patientPath(e, "/prefill/"+again.String()), still), http.StatusNotFound)
		refused(t, w.site2.Do("POST", at(w.a2, "/episodes/"+e.ID.String()+"/patient-form/prefill/"+again.String()), still), http.StatusNotFound)
	})
	t.Run("another trust: 404", func(t *testing.T) {
		refused(t, w.otherClin.Do("POST", staffPath(again.String()), still), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", at(w.b, "/episodes/"+e.ID.String()+"/patient-form/prefill/"+again.String()), still), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, patient.Do("POST", patientPath(e, "/prefill/"+again.String()), map[string]string{"choice": "maybe"}), http.StatusBadRequest)
		refused(t, patient.Do("POST", patientPath(e, "/prefill/"+again.String()), `{"choice":`), http.StatusBadRequest)
		refused(t, patient.Do("POST", patientPath(e, "/prefill/abc"), still), http.StatusNotFound)
		refused(t, patient.Do("POST", patientPath(e, "/prefill/"+clinic.String()), still), http.StatusNotFound) // the clinician's set
		refused(t, patient.Do("POST", patientPath(e, "/prefill/"+w.hq.Chapters[0].String()), still), http.StatusNotFound)
		// A set with nothing pre-filled in it.
		bare := w.fx.Episode(t, w.later, w.cara)
		p := w.anon.NewBrowser(t).AsPatient(t, bare.Link, bare.DateOfBirth)
		getForm(t, p, patientPath(bare, ""))
		refused(t, p.Do("POST", patientPath(bare, "/prefill/"+again.String()), still), http.StatusConflict)
		if c, _, _ := choice(again); c != "" {
			t.Errorf("choice %q after refusals", c)
		}
	})
	t.Run("happy path: the patient's button is recorded once, the first kept", func(t *testing.T) {
		patient.Do("POST", patientPath(e, "/prefill/"+again.String()), still).Want(t, http.StatusNoContent)
		if c, by, when := choice(again); c != "still_correct" || by != nil || !when {
			t.Errorf("choice %q by %v at %v", c, by, when)
		}
		// Confirming counts as starting (test report 1, found 1): the home says In progress, and the
		// reminders stop, as after a save.
		if got := homeState(t, patient, e.ID); got != "in_progress" {
			t.Errorf("home state %q after confirming", got)
		}
		if f, err := db.New(w.db).GetFollowup(t.Context(), e.ID); err != nil || !f.Started {
			t.Errorf("reminders see it not started after confirming: %v %v", f.Started, err)
		}
		patient.Do("POST", patientPath(e, "/prefill/"+again.String()), check).Want(t, http.StatusNoContent) // another tab
		if c, _, _ := choice(again); c != "still_correct" {
			t.Errorf("choice %q, want the first", c)
		}
		f := getForm(t, patient, patientPath(e, ""))
		if p := f.Prefill[again.String()]; p.Choice == nil || *p.Choice != "still_correct" || p.ChoiceAt == nil {
			t.Errorf("prefill %+v", p)
		}
		// Recording the button isn't an answer change: the next save's revision is still the
		// pre-filled row's, 1 (S26).
		savedRevision(t, patient.Do("PUT", patientPath(e, "/answers/"+again.String()), map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 1}))
	})
	t.Run("happy path: staff for the patient, marked as theirs", func(t *testing.T) {
		w.clinician.Do("POST", staffPath(healthAgain.String()), check).Want(t, http.StatusNoContent)
		if c, by, _ := choice(healthAgain); c != "check_each" || by == nil || *by != w.cara.ID {
			t.Errorf("choice %q by %v", c, by)
		}
	})
	t.Run("audited, one row each", func(t *testing.T) {
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'link.prefill' AND $1 = ANY(episode_ids)`, e.ID); n < 2 {
			t.Errorf("%d link.prefill rows", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'behalf.prefill' AND $1 = ANY(episode_ids) AND user_id = $2`, e.ID, w.cara.ID); n != 1 {
			t.Errorf("%d behalf.prefill rows", n)
		}
	})
	t.Run("once sent: 409", func(t *testing.T) {
		patient.Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		refused(t, w.clinician.Do("POST", staffPath(healthAgain.String()), still), http.StatusConflict)
	})
}
