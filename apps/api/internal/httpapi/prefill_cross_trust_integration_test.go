//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// Pre-fill from another trust (S26; §6 point 7, ORG-04's crossTrustPrefill): answers a patient
// gave trust B are offered at trust A only when both trusts have the setting on, and used only once
// the patient has said Yes, asked once per trust (patient_prefill_consents). Never on the patient's
// behalf. Reading trust B's answers writes an audit row under trust B too.

// crossTrust is one person with a sent, answered HQ at trust B (crosstrust-b.json) and a record at
// trust A joined to the same account, with an open episode on trust A's later HQ (prefill.json).
type crossTrust struct {
	inB     apitest.Episode // the sent HQ at trust B
	inA     apitest.Episode // the HQ to fill in at trust A
	patient *apitest.Client // the person's session (their account's)
}

// crossTrustWorld is the pre-fill world plus trust B's HQ.
type crossTrustWorld struct {
	*prefillWorld
	bHQ apitest.HQ
}

func newCrossTrustWorld(t *testing.T) *crossTrustWorld {
	t.Helper()
	w := &crossTrustWorld{prefillWorld: newPrefillWorld(t)}
	w.bHQ = w.fx.PublishedHQ(t, w.b, w.bea, "crosstrust-b.json")
	return w
}

// setCrossTrust switches a trust's crossTrustPrefill setting.
func (w *crossTrustWorld) setCrossTrust(t *testing.T, org uuid.UUID, on bool) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET settings = settings || jsonb_build_object('crossTrustPrefill', $2::boolean) WHERE id = $1`, org, on); err != nil {
		t.Fatal(err)
	}
}

// person makes a crossTrust: trust B's HQ answered (smoking Yes, which is o_no there; medicines;
// diabetes) and sent; trust A's
// record joined to the same account. A's own earlier answers, if any, are the caller's.
func (w *crossTrustWorld) person(t *testing.T, last string) crossTrust {
	t.Helper()
	rec := apitest.PatientRecord{FirstName: "Cross", LastName: last, DateOfBirth: "1960-06-06"}
	inB := w.fx.EpisodeOf(t, w.bHQ, w.b, w.fx.Patient(t, w.b.Org, rec), w.bea, "")
	p := w.anon.NewBrowser(t).AsPatient(t, inB.Link, inB.DateOfBirth)
	savedRevision(t, p.Do("PUT", patientPath(inB, "/answers/"+w.bHQ.Chapters[0].String()), map[string]any{
		"data": map[string]any{"q_smokes": "o_no", "q_meds": []any{map[string]any{"q_med_name": "Ibuprofen"}}}, "revision": 0}))
	savedRevision(t, p.Do("PUT", patientPath(inB, "/answers/"+w.bHQ.Chapters[1].String()), map[string]any{
		"data": map[string]any{"q_diabetes": true}, "revision": 0}))
	p.Do("POST", patientPath(inB, "/submit"), nil).Want(t, http.StatusNoContent)
	aRec := w.fx.Patient(t, w.a.Org, rec)
	if _, err := w.db.Exec(t.Context(), `UPDATE patients SET account_id = (SELECT account_id FROM patients WHERE id = $1) WHERE id = $2`, inB.PatientID, aRec); err != nil {
		t.Fatal(err)
	}
	return crossTrust{inB: inB, inA: w.fx.EpisodeOf(t, w.later, w.a, aRec, w.cara, ""), patient: p}
}

// crossForm is the patient's HQ as GET returns it, with the consent questions.
type crossForm struct {
	prefillForm
	PrefillConsent []struct {
		OrgID       uuid.UUID
		Hospital    string
		SubmittedAt time.Time
	}
	Prefill map[string]struct {
		Answers map[string]struct {
			FromForm      uuid.UUID
			MatchedBy     string
			Questionnaire string
			OtherTrust    bool
			Hospital      string
			Value         json.RawMessage
		}
	}
}

func getCrossForm(t *testing.T, c *apitest.Client, path string) crossForm {
	t.Helper()
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	var f crossForm
	r.Decode(t, &f)
	return f
}

func consent(c *apitest.Client, e apitest.Episode, org uuid.UUID, agree bool) *apitest.Resp {
	return c.Do("POST", patientPath(e, "/prefill-consent"), map[string]any{"orgId": org, "agree": agree})
}

func TestPrefillCrossTrust(t *testing.T) {
	t.Parallel()
	w := newCrossTrustWorld(t)
	again, healthAgain := w.later.Chapters[0], w.later.Chapters[1]
	var bHospital string
	if err := w.db.QueryRow(t.Context(), `SELECT name FROM hospitals WHERE id = $1`, w.b.ID).Scan(&bHospital); err != nil {
		t.Fatal(err)
	}

	t.Run("not offered unless both trusts have the setting on", func(t *testing.T) {
		x := w.person(t, "Settings")
		for _, c := range []struct{ a, b bool }{{false, false}, {true, false}, {false, true}} {
			w.setCrossTrust(t, w.a.Org, c.a)
			w.setCrossTrust(t, w.b.Org, c.b)
			f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
			if len(f.PrefillConsent) != 0 || len(f.Answers) != 0 {
				t.Errorf("A %v, B %v: consent %+v, answers %v", c.a, c.b, f.PrefillConsent, f.Answers)
			}
		}
		// Both on: the question names trust B's hospital and when the answers were given, and nothing
		// is filled in until it's answered.
		w.setCrossTrust(t, w.a.Org, true)
		w.setCrossTrust(t, w.b.Org, true)
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if len(f.PrefillConsent) != 1 || f.PrefillConsent[0].OrgID != w.b.Org || f.PrefillConsent[0].Hospital != bHospital || f.PrefillConsent[0].SubmittedAt.IsZero() {
			t.Fatalf("consent %+v", f.PrefillConsent)
		}
		if len(f.Answers) != 0 || count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_form_id = $1`, x.inA.FormID) != 0 {
			t.Errorf("filled before the patient answered: %v", f.Answers)
		}
		// Reading the question reads no answers of trust B's: no audit row under trust B.
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND action = 'prefill.read'`, w.b.Org); n != 0 {
			t.Errorf("%d audit rows under trust B before the patient agreed", n)
		}
	})

	t.Run("happy path: Yes fills from trust B by wording, marked as another trust's, with no trust B id; its own colliding key never matches", func(t *testing.T) {
		w.setCrossTrust(t, w.a.Org, true)
		w.setCrossTrust(t, w.b.Org, true)
		x := w.person(t, "Yes")
		getCrossForm(t, x.patient, patientPath(x.inA, ""))
		consent(x.patient, x.inA, w.b.Org, true).Want(t, http.StatusNoContent)
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if len(f.PrefillConsent) != 0 {
			t.Errorf("asked again: %+v", f.PrefillConsent)
		}
		// Not smoking: q_smokes is trust B's own key, not a platform library key, and it's worded
		// differently there ("Do you smoke now?"), so it never matches (answers, after verification 1).
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Ibuprofen"}]}` {
			t.Errorf("About you again %s", got)
		}
		if got := answersJSON(t, f.Answers[healthAgain.String()]); got != `{"pf_diabetes":true}` {
			t.Errorf("Health again %s", got)
		}
		meds := f.Prefill[again.String()].Answers["q_meds"]
		if meds.FromForm != uuid.Nil || !meds.OtherTrust || meds.Hospital != bHospital || meds.Questionnaire != w.bHQ.Name || meds.MatchedBy != "wording" {
			t.Errorf("q_meds %+v", meds)
		}
		// No trust B id in trust A's rows, nor in what trust A's staff get (verification 1, finding 2).
		for _, id := range []uuid.UUID{x.inB.FormID, x.inB.ID, x.inB.PatientID, w.b.Org, w.b.ID, w.bHQ.ID, w.bHQ.VersionID} {
			if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_form_id = $1 AND (prefill::text LIKE '%' || $2 || '%' OR data::text LIKE '%' || $2 || '%')`,
				x.inA.FormID, id.String()); n != 0 {
				t.Errorf("trust A's answers hold trust B's id %s", id)
			}
			if r := w.clinician.Do("GET", epPath(w.a, x.inA, "/hq"), nil); strings.Contains(string(r.Body), id.String()) {
				t.Errorf("trust A's staff get trust B's id %s", id)
			}
		}
		if d := f.Prefill[healthAgain.String()].Answers["pf_diabetes"]; d.MatchedBy != "wording" || !d.OtherTrust {
			t.Errorf("pf_diabetes %+v", d)
		}
		if f.Revisions[again.String()] != 1 {
			t.Errorf("revision %d, want 1 for a pre-filled set", f.Revisions[again.String()])
		}
		if n := count(t, w.db, `SELECT count(*) FROM patient_prefill_consents c JOIN patients p ON p.account_id = c.account_id
			WHERE p.id = $1 AND c.org_id = $2 AND c.agreed AND c.episode_id = $3`, x.inA.PatientID, w.b.Org, x.inA.ID); n != 1 {
			t.Errorf("%d consent rows", n)
		}
		// Audited under trust B too: the patient, its record and the source episode, at its hospital.
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND hospital_id = $2 AND action = 'prefill.read' AND actor = 'patient'
			AND patient_ids = ARRAY[$3::uuid] AND episode_ids = ARRAY[$4::uuid]`, w.b.Org, w.b.ID, x.inB.PatientID, x.inB.ID); n != 1 {
			t.Errorf("%d prefill.read rows under trust B", n)
		}
		// Trust B's admin reads it as a read, not as a use: Trust A's own answers may have covered
		// every question (test report 1, found 4).
		bAdmin := w.anon.NewBrowser(t).LoginAs(t, w.fx.TrustUser(t, w.b.Org, "admin"))
		r := bAdmin.Do("GET", "/api/o/"+w.b.Org.String()+"/audit", nil)
		r.Want(t, http.StatusOK)
		var log struct {
			Rows []struct{ Action, ActionLabel string }
		}
		r.Decode(t, &log)
		labels := map[string]string{}
		for _, row := range log.Rows {
			labels[row.Action] = row.ActionLabel
		}
		if got := labels["prefill.read"]; got != "Patient's answers read to pre-fill their HQ at another trust (they agreed)" {
			t.Errorf("trust B's audit label %q", got)
		}
		// Trust A's own row names only trust A's records.
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND ($2 = ANY(episode_ids) OR $3 = ANY(patient_ids))`, w.a.Org, x.inB.ID, x.inB.PatientID); n != 0 {
			t.Errorf("%d trust A rows name trust B's records", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND action = 'link.prefill_consent' AND $2 = ANY(episode_ids)`, w.a.Org, x.inA.ID); n != 1 {
			t.Errorf("%d consent audit rows", n)
		}

		// A second open reads trust B no more: every set has its row.
		before := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND action = 'prefill.read'`, w.b.Org)
		getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if after := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND action = 'prefill.read'`, w.b.Org); after != before {
			t.Errorf("prefill.read rows %d → %d", before, after)
		}

		// Another episode at trust A: used again without asking (asked once per trust).
		next := w.fx.EpisodeOf(t, w.later, w.a, x.inA.PatientID, w.cara, "")
		f = getCrossForm(t, x.patient, patientPath(next, ""))
		if len(f.PrefillConsent) != 0 || !f.Prefill[again.String()].Answers["q_meds"].OtherTrust {
			t.Errorf("next episode: consent %+v prefill %+v", f.PrefillConsent, f.Prefill)
		}

		// Switching either trust's setting off stops new offers; what was copied stays.
		w.setCrossTrust(t, w.b.Org, false)
		third := w.fx.EpisodeOf(t, w.later, w.a, x.inA.PatientID, w.cara, "")
		if f := getCrossForm(t, x.patient, patientPath(third, "")); len(f.Answers) != 0 || len(f.PrefillConsent) != 0 {
			t.Errorf("trust B off: %v %+v", f.Answers, f.PrefillConsent)
		}
		if f := getCrossForm(t, x.patient, patientPath(x.inA, "")); answersJSON(t, f.Answers[again.String()]) != `{"q_meds":[{"q_med_name":"Ibuprofen"}]}` {
			t.Errorf("copied answers changed: %v", f.Answers)
		}
		w.setCrossTrust(t, w.b.Org, true)
	})

	t.Run("a platform library key matches across trusts though reworded, its choice mapped by option text", func(t *testing.T) {
		w.setCrossTrust(t, w.a.Org, true)
		w.setCrossTrust(t, w.b.Org, true)
		if _, err := w.db.Exec(t.Context(), `INSERT INTO library_questions (org_id, key, title, element)
			VALUES (NULL, 'q_smokes', 'Do you smoke?', '{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?"}')`); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `DELETE FROM library_questions WHERE org_id IS NULL AND key = 'q_smokes'`)
		x := w.person(t, "Library")
		consent(x.patient, x.inA, w.b.Org, true).Want(t, http.StatusNoContent)
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		// "Yes" at trust B is o_no there, and o_yes ("Yes") here: by its text, not its value.
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Ibuprofen"}],"q_smokes":"o_yes"}` {
			t.Errorf("About you again %s", got)
		}
		if s := f.Prefill[again.String()].Answers["q_smokes"]; s.MatchedBy != "key" || !s.OtherTrust || s.FromForm != uuid.Nil {
			t.Errorf("q_smokes %+v", s)
		}
	})

	t.Run("No, thanks is remembered: nothing from trust B, and not asked again here or at another episode", func(t *testing.T) {
		w.setCrossTrust(t, w.a.Org, true)
		w.setCrossTrust(t, w.b.Org, true)
		x := w.person(t, "No")
		consent(x.patient, x.inA, w.b.Org, false).Want(t, http.StatusNoContent)
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if len(f.PrefillConsent) != 0 || len(f.Answers) != 0 {
			t.Errorf("after No: consent %+v answers %v", f.PrefillConsent, f.Answers)
		}
		next := w.fx.EpisodeOf(t, w.later, w.a, x.inA.PatientID, w.cara, "")
		if f := getCrossForm(t, x.patient, patientPath(next, "")); len(f.PrefillConsent) != 0 || len(f.Answers) != 0 {
			t.Errorf("another episode: consent %+v answers %v", f.PrefillConsent, f.Answers)
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND action = 'prefill.read' AND $2 = ANY(episode_ids)`, w.b.Org, x.inB.ID); n != 0 {
			t.Errorf("%d prefill.read rows after No", n)
		}
		// The choice can't be changed (S26 answers 3).
		refused(t, consent(x.patient, x.inA, w.b.Org, true), http.StatusConflict)
	})

	t.Run("the form's own trust first, and nothing the patient already typed is overwritten", func(t *testing.T) {
		w.setCrossTrust(t, w.a.Org, true)
		w.setCrossTrust(t, w.b.Org, true)
		x := w.person(t, "Own")
		// An earlier sent HQ at trust A too, whose smoking answer the clinician made "No".
		own := w.fx.EpisodeOf(t, w.hq, w.a, x.inA.PatientID, w.cara, "")
		w.answeredBy(t, own, x.patient, "o_no")
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if len(f.PrefillConsent) != 1 {
			t.Fatalf("consent %+v", f.PrefillConsent)
		}
		// Typed (another tab) before answering the question.
		savedRevision(t, x.patient.Do("PUT", patientPath(x.inA, "/answers/"+healthAgain.String()), map[string]any{"data": map[string]any{"pf_diabetes": false}, "revision": 0}))
		consent(x.patient, x.inA, w.b.Org, true).Want(t, http.StatusNoContent)
		f = getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if got := answersJSON(t, f.Answers[healthAgain.String()]); got != `{"pf_diabetes":false}` || f.Revisions[healthAgain.String()] != 1 {
			t.Errorf("Health again %s rev %d", got, f.Revisions[healthAgain.String()])
		}
		if _, ok := f.Prefill[healthAgain.String()]; ok {
			t.Error("prefill on a set the patient typed in")
		}
		// Smoking and medicines from trust A's own form (it asked them), not trust B's.
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Aspirin"}],"q_smokes":"o_no"}` {
			t.Errorf("About you again %s", got)
		}
		if s := f.Prefill[again.String()].Answers["q_smokes"]; s.FromForm != own.FormID || s.OtherTrust || s.Hospital != "" || s.Questionnaire != w.hq.Name {
			t.Errorf("q_smokes %+v", s)
		}
	})

	t.Run("staff filling in for the patient never use another trust's answers, even after a Yes", func(t *testing.T) {
		w.setCrossTrust(t, w.a.Org, true)
		w.setCrossTrust(t, w.b.Org, true)
		x := w.person(t, "Behalf")
		consent(x.patient, x.inA, w.b.Org, true).Want(t, http.StatusNoContent)
		e := w.fx.EpisodeOf(t, w.later, w.a, x.inA.PatientID, w.cara, "")
		f := getCrossForm(t, w.clinician, epPath(w.a, e, "/patient-form"))
		if len(f.Answers) != 0 || len(f.PrefillConsent) != 0 {
			t.Errorf("on behalf: answers %v consent %+v", f.Answers, f.PrefillConsent)
		}
	})

	t.Run("another trust's unreadable settings: skipped, the form still opens", func(t *testing.T) {
		w.setCrossTrust(t, w.a.Org, true)
		x := w.person(t, "Unreadable")
		if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET settings = '[]' WHERE id = $1`, w.b.Org); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `UPDATE orgs SET settings = '{}' WHERE id = $1`, w.b.Org)
		if f := getCrossForm(t, x.patient, patientPath(x.inA, "")); len(f.PrefillConsent) != 0 || len(f.Answers) != 0 {
			t.Errorf("unreadable trust B: %+v %v", f.PrefillConsent, f.Answers)
		}
	})

	t.Run("a suspended trust's answers aren't offered", func(t *testing.T) {
		w.setCrossTrust(t, w.a.Org, true)
		w.setCrossTrust(t, w.b.Org, true)
		x := w.person(t, "Suspended")
		if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET status = 'suspended' WHERE id = $1`, w.b.Org); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `UPDATE orgs SET status = 'active' WHERE id = $1`, w.b.Org)
		if f := getCrossForm(t, x.patient, patientPath(x.inA, "")); len(f.PrefillConsent) != 0 || len(f.Answers) != 0 {
			t.Errorf("suspended trust B: %+v %v", f.PrefillConsent, f.Answers)
		}
	})
}

// answeredBy is answered for a patient who already has a session (their account's).
func (w *crossTrustWorld) answeredBy(t *testing.T, e apitest.Episode, p *apitest.Client, smokes string) {
	t.Helper()
	about, health := w.hq.Chapters[0], w.hq.Chapters[1]
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+about.String()), map[string]any{
		"data": map[string]any{"q_smokes": "o_yes", "q_meds": []any{map[string]any{"q_med_name": "Aspirin"}}}, "revision": 0}))
	savedRevision(t, p.Do("PUT", patientPath(e, "/answers/"+health.String()), map[string]any{"data": map[string]any{"q_diabetes": true}, "revision": 0}))
	p.Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
	savedRevision(t, w.clinician.Do("PUT", epPath(w.a, e, "/answers/"+about.String()), map[string]any{"data": map[string]any{
		"q_smokes": smokes, "q_meds": []any{map[string]any{"q_med_name": "Aspirin"}}}, "validated": true, "revision": 0}))
}

// POST /api/p/episodes/{eid}/prefill-consent {orgId, agree}: the patient's answer to "Use the
// answers you gave …?", once per account and source trust.
func TestPrefillConsentRoute(t *testing.T) {
	t.Parallel()
	w := newCrossTrustWorld(t)
	w.setCrossTrust(t, w.a.Org, true)
	w.setCrossTrust(t, w.b.Org, true)
	x := w.person(t, "Route")
	decided := func() int {
		return count(t, w.db, `SELECT count(*) FROM patient_prefill_consents c JOIN patients p ON p.account_id = c.account_id WHERE p.id = $1`, x.inA.PatientID)
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, consent(w.anon, x.inA, w.b.Org, true), http.StatusUnauthorized)
	})
	t.Run("wrong role: staff have no patient session", func(t *testing.T) {
		refused(t, consent(w.clinician, x.inA, w.b.Org, true), http.StatusUnauthorized)
	})
	t.Run("another patient's episode: 404", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		someoneElse := w.anon.NewBrowser(t).AsPatient(t, other.Link, other.DateOfBirth)
		refused(t, consent(someoneElse, x.inA, w.b.Org, true), http.StatusNotFound)
		// Nor can the person's own session name an episode that isn't theirs.
		refused(t, consent(x.patient, other, w.b.Org, true), http.StatusNotFound)
		if decided() != 0 {
			t.Error("recorded")
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		path := patientPath(x.inA, "/prefill-consent")
		refused(t, x.patient.Do("POST", path, `{"orgId":`), http.StatusBadRequest)
		bad := x.patient.Do("POST", path, map[string]any{"orgId": "abc", "agree": true})
		refused(t, bad, http.StatusBadRequest)
		if !strings.Contains(bad.Message(), "reload the page") { // not "Choose Yes or No" (test report 1, found 2)
			t.Errorf("bad orgId: %q", bad.Message())
		}
		none := x.patient.Do("POST", path, map[string]any{"orgId": w.b.Org}) // no answer
		refused(t, none, http.StatusBadRequest)
		if !strings.Contains(none.Message(), "Yes, use them") {
			t.Errorf("no answer: %q", none.Message())
		}
		// Not on offer: the form's own trust, or a trust the person has no answers at.
		refused(t, consent(x.patient, x.inA, w.a.Org, true), http.StatusConflict)
		refused(t, consent(x.patient, x.inA, uuid.New(), true), http.StatusConflict)
		if decided() != 0 {
			t.Error("recorded")
		}
	})
	t.Run("happy path: recorded once, audited; then neither changed nor asked again", func(t *testing.T) {
		consent(x.patient, x.inA, w.b.Org, false).Want(t, http.StatusNoContent)
		if n := count(t, w.db, `SELECT count(*) FROM patient_prefill_consents c JOIN patients p ON p.account_id = c.account_id
			WHERE p.id = $1 AND c.org_id = $2 AND NOT c.agreed AND c.decided_at IS NOT NULL`, x.inA.PatientID, w.b.Org); n != 1 {
			t.Errorf("%d consent rows", n)
		}
		refused(t, consent(x.patient, x.inA, w.b.Org, false), http.StatusConflict)
		refused(t, consent(x.patient, x.inA, w.b.Org, true), http.StatusConflict)
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'link.prefill_consent' AND org_id = $1 AND $2 = ANY(episode_ids) AND status = 204`, w.a.Org, x.inA.ID); n != 1 {
			t.Errorf("%d audit rows", n)
		}
	})
	t.Run("once the HQ is sent: 409", func(t *testing.T) {
		y := w.person(t, "Sent")
		if _, err := w.db.Exec(t.Context(), `UPDATE episode_forms SET status = 'submitted', submitted_at = now() WHERE id = $1`, y.inA.FormID); err != nil {
			t.Fatal(err)
		}
		refused(t, consent(y.patient, y.inA, w.b.Org, true), http.StatusConflict)
	})
}

// A pre-filled set starts at revision 1, so a tab that loaded the set before it was pre-filled
// (it sends 0, "no row yet") gets the stale 409 instead of overwriting it, and the prefill map keeps
// describing the row (S25 verify, for S26). It still isn't "started" until the patient saves.
func TestPrefillRevision(t *testing.T) {
	t.Parallel()
	w := newPrefillWorld(t)
	again := w.later.Chapters[0]
	e := w.fx.EpisodeOf(t, w.later, w.a, w.ep.PatientID, w.cara, "")
	stale := w.anon.NewBrowser(t).AsPatient(t, e.Link, e.DateOfBirth)
	f := getForm(t, stale, patientPath(e, "")) // nothing to fill yet: no row
	if len(f.Answers) != 0 {
		t.Fatalf("answers %v", f.Answers)
	}
	// An earlier HQ is sent meanwhile; another open pre-fills.
	w.answered(t, w.ep, "o_yes")
	fresh := w.anon.NewBrowser(t).AsPatient(t, e.Link, e.DateOfBirth)
	f = getForm(t, fresh, patientPath(e, ""))
	if f.Revisions[again.String()] != 1 {
		t.Fatalf("pre-filled revision %d, want 1", f.Revisions[again.String()])
	}
	if got, err := db.New(w.db).GetFollowup(t.Context(), e.ID); err != nil || got.Started {
		t.Errorf("untouched pre-fill counts as started: %v %v", got.Started, err)
	}
	refused(t, stale.Do("PUT", patientPath(e, "/answers/"+again.String()), map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 0}), http.StatusConflict)
	if got := answersJSON(t, getForm(t, fresh, patientPath(e, "")).Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Aspirin"}],"q_smokes":"o_yes"}` {
		t.Errorf("overwritten: %s", got)
	}
	if rev := savedRevision(t, fresh.Do("PUT", patientPath(e, "/answers/"+again.String()), map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 1})); rev != 2 {
		t.Errorf("revision %d after the first save", rev)
	}
	if got, err := db.New(w.db).GetFollowup(t.Context(), e.ID); err != nil || !got.Started {
		t.Errorf("saved pre-fill not started: %v %v", got.Started, err)
	}
}
