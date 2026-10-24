//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// Pre-fill from another trust (S26, changed in S27; Rahul, 24 Oct): answers a patient gave trust B
// are always copied into their HQ at trust A, with no trust setting and no question first, by the
// patient or by staff for them. They fill only what trust A's own forms didn't ask, match by key
// only for a platform library key (else by exact wording), and trust A keeps no trust B id.
// Reading trust B's answers writes an audit row under trust B too. A set holding them opens on a
// card naming trust B's hospital, with Start fresh (TestStartFresh).

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
	bHQ       apitest.HQ
	bHospital string // trust B's hospital's name
}

func newCrossTrustWorld(t *testing.T) *crossTrustWorld {
	t.Helper()
	w := &crossTrustWorld{prefillWorld: newPrefillWorld(t)}
	w.bHQ = w.fx.PublishedHQ(t, w.b, w.bea, "crosstrust-b.json")
	if err := w.db.QueryRow(t.Context(), `SELECT name FROM hospitals WHERE id = $1`, w.b.ID).Scan(&w.bHospital); err != nil {
		t.Fatal(err)
	}
	return w
}

// person makes a crossTrust: trust B's HQ answered (smoking Yes, which is o_no there; medicines;
// diabetes) and sent; trust A's record joined to the same account. A's own earlier answers, if
// any, are the caller's.
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

// crossForm is the patient's HQ as GET returns it, with where each pre-filled answer came from.
type crossForm struct {
	prefillForm
	Prefill map[string]struct {
		Answers map[string]struct {
			FromForm      uuid.UUID
			MatchedBy     string
			Questionnaire string
			OtherTrust    bool
			Hospital      string
			Value         json.RawMessage
		}
		Choice *string
	}
	// S26's question, which S27 removed: it must never come back.
	PrefillConsent json.RawMessage
}

func getCrossForm(t *testing.T, c *apitest.Client, path string) crossForm {
	t.Helper()
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	var f crossForm
	r.Decode(t, &f)
	return f
}

// prefillReads counts the prefill.read rows under trust B that name its source episode.
func (w *crossTrustWorld) prefillReads(t *testing.T, x crossTrust) int {
	t.Helper()
	return count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND action = 'prefill.read' AND $2 = ANY(episode_ids)`, w.b.Org, x.inB.ID)
}

func TestPrefillCrossTrust(t *testing.T) {
	t.Parallel()
	w := newCrossTrustWorld(t)
	again, healthAgain := w.later.Chapters[0], w.later.Chapters[1]

	t.Run("happy path: copied at the first open, with no setting and no question; by wording, marked as another trust's, with no trust B id", func(t *testing.T) {
		x := w.person(t, "Yes")
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if f.PrefillConsent != nil {
			t.Errorf("a consent question: %s", f.PrefillConsent)
		}
		// Not smoking: q_smokes is trust B's own key, not a platform library key, and it's worded
		// differently there ("Do you smoke now?"), so it never matches (S26, after verification 1).
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Ibuprofen"}]}` {
			t.Errorf("About you again %s", got)
		}
		if got := answersJSON(t, f.Answers[healthAgain.String()]); got != `{"pf_diabetes":true}` {
			t.Errorf("Health again %s", got)
		}
		meds := f.Prefill[again.String()].Answers["q_meds"]
		if meds.FromForm != uuid.Nil || !meds.OtherTrust || meds.Hospital != w.bHospital || meds.Questionnaire != w.bHQ.Name || meds.MatchedBy != "wording" {
			t.Errorf("q_meds %+v", meds)
		}
		if d := f.Prefill[healthAgain.String()].Answers["pf_diabetes"]; d.MatchedBy != "wording" || !d.OtherTrust {
			t.Errorf("pf_diabetes %+v", d)
		}
		if f.Revisions[again.String()] != 1 {
			t.Errorf("revision %d, want 1 for a pre-filled set", f.Revisions[again.String()])
		}
		// No trust B id in trust A's rows, nor in what trust A's staff get (S26 verification 1, finding 2).
		for _, id := range []uuid.UUID{x.inB.FormID, x.inB.ID, x.inB.PatientID, w.b.Org, w.b.ID, w.bHQ.ID, w.bHQ.VersionID} {
			if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_form_id = $1 AND (prefill::text LIKE '%' || $2 || '%' OR data::text LIKE '%' || $2 || '%')`,
				x.inA.FormID, id.String()); n != 0 {
				t.Errorf("trust A's answers hold trust B's id %s", id)
			}
			if r := w.clinician.Do("GET", epPath(w.a, x.inA, "/hq"), nil); strings.Contains(string(r.Body), id.String()) {
				t.Errorf("trust A's staff get trust B's id %s", id)
			}
		}
		// Audited under trust B too: the patient, its record and the source episode, at its hospital.
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND hospital_id = $2 AND action = 'prefill.read' AND actor = 'patient'
			AND patient_ids = ARRAY[$3::uuid] AND episode_ids = ARRAY[$4::uuid]`, w.b.Org, w.b.ID, x.inB.PatientID, x.inB.ID); n != 1 {
			t.Errorf("%d prefill.read rows under trust B", n)
		}
		// Trust B's admin reads it as a read, with no "(they agreed)": nobody is asked now.
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
		if got := labels["prefill.read"]; got != "Patient's answers read to pre-fill their HQ at another trust" {
			t.Errorf("trust B's audit label %q", got)
		}
		// Trust A's own row names only trust A's records.
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND ($2 = ANY(episode_ids) OR $3 = ANY(patient_ids))`, w.a.Org, x.inB.ID, x.inB.PatientID); n != 0 {
			t.Errorf("%d trust A rows name trust B's records", n)
		}

		// A second open reads trust B no more: every set has its row.
		before := w.prefillReads(t, x)
		getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if after := w.prefillReads(t, x); after != before {
			t.Errorf("prefill.read rows %d → %d", before, after)
		}
		// Another episode at trust A: copied again.
		next := w.fx.EpisodeOf(t, w.later, w.a, x.inA.PatientID, w.cara, "")
		if f := getCrossForm(t, x.patient, patientPath(next, "")); !f.Prefill[again.String()].Answers["q_meds"].OtherTrust {
			t.Errorf("next episode: prefill %+v", f.Prefill)
		}
	})

	t.Run("a trust's old crossTrustPrefill value changes nothing", func(t *testing.T) {
		for _, org := range []uuid.UUID{w.a.Org, w.b.Org} {
			if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET settings = settings || '{"crossTrustPrefill": false}' WHERE id = $1`, org); err != nil {
				t.Fatal(err)
			}
			defer w.db.Exec(t.Context(), `UPDATE orgs SET settings = settings - 'crossTrustPrefill' WHERE id = $1`, org)
		}
		x := w.person(t, "OldSetting")
		if f := getCrossForm(t, x.patient, patientPath(x.inA, "")); answersJSON(t, f.Answers[healthAgain.String()]) != `{"pf_diabetes":true}` {
			t.Errorf("Health again %v", f.Answers)
		}
	})

	t.Run("a platform library key matches across trusts though reworded, its choice mapped by option text", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `INSERT INTO library_questions (org_id, key, title, element)
			VALUES (NULL, 'q_smokes', 'Do you smoke?', '{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?"}')`); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `DELETE FROM library_questions WHERE org_id IS NULL AND key = 'q_smokes'`)
		x := w.person(t, "Library")
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		// "Yes" at trust B is o_no there, and o_yes ("Yes") here: by its text, not its value.
		if got := answersJSON(t, f.Answers[again.String()]); got != `{"q_meds":[{"q_med_name":"Ibuprofen"}],"q_smokes":"o_yes"}` {
			t.Errorf("About you again %s", got)
		}
		if s := f.Prefill[again.String()].Answers["q_smokes"]; s.MatchedBy != "key" || !s.OtherTrust || s.FromForm != uuid.Nil {
			t.Errorf("q_smokes %+v", s)
		}
	})

	t.Run("the form's own trust first, and nothing the patient already typed is overwritten", func(t *testing.T) {
		x := w.person(t, "Own")
		// An earlier sent HQ at trust A too, whose smoking answer the clinician made "No".
		own := w.fx.EpisodeOf(t, w.hq, w.a, x.inA.PatientID, w.cara, "")
		w.answeredBy(t, own, x.patient, "o_no")
		// Typed (another tab) before the form was first opened.
		savedRevision(t, x.patient.Do("PUT", patientPath(x.inA, "/answers/"+healthAgain.String()), map[string]any{"data": map[string]any{"pf_diabetes": false}, "revision": 0}))
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
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

	t.Run("staff filling in for the patient get another trust's answers too, audited under it as them", func(t *testing.T) {
		x := w.person(t, "Behalf")
		f := getCrossForm(t, w.clinician, epPath(w.a, x.inA, "/patient-form"))
		if got := answersJSON(t, f.Answers[healthAgain.String()]); got != `{"pf_diabetes":true}` {
			t.Errorf("on behalf: Health again %s", got)
		}
		if m := f.Prefill[again.String()].Answers["q_meds"]; !m.OtherTrust || m.Hospital != w.bHospital || m.FromForm != uuid.Nil {
			t.Errorf("on behalf: q_meds %+v", m)
		}
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE org_id = $1 AND hospital_id = $2 AND action = 'prefill.read' AND actor = 'staff'
			AND user_id = $3 AND episode_ids = ARRAY[$4::uuid]`, w.b.Org, w.b.ID, w.cara.ID, x.inB.ID); n != 1 {
			t.Errorf("%d prefill.read rows under trust B for Cara", n)
		}
	})

	t.Run("on behalf, a record that hasn't joined the person's account finds no other trust's answers (S27 test report 1, F1)", func(t *testing.T) {
		// Other trusts' records are found only through the account, which a record joins when the
		// patient proves their date of birth on its link (S15): Go never matches records across
		// trusts by typed details alone.
		x := w.person(t, "Unjoined")
		if _, err := w.db.Exec(t.Context(), `UPDATE patients SET account_id = NULL WHERE id = $1`, x.inA.PatientID); err != nil {
			t.Fatal(err)
		}
		f := getCrossForm(t, w.clinician, epPath(w.a, x.inA, "/patient-form"))
		if len(f.Answers) != 0 || w.prefillReads(t, x) != 0 {
			t.Errorf("unjoined, on behalf: %v", f.Answers)
		}
	})

	t.Run("a suspended trust's answers aren't used", func(t *testing.T) {
		x := w.person(t, "Suspended")
		if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET status = 'suspended' WHERE id = $1`, w.b.Org); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `UPDATE orgs SET status = 'active' WHERE id = $1`, w.b.Org)
		if f := getCrossForm(t, x.patient, patientPath(x.inA, "")); len(f.Answers) != 0 || w.prefillReads(t, x) != 0 {
			t.Errorf("suspended trust B: %v", f.Answers)
		}
	})

	t.Run("S26's consent route is gone", func(t *testing.T) {
		x := w.person(t, "Route")
		r := x.patient.Do("POST", patientPath(x.inA, "/prefill-consent"), map[string]any{"orgId": w.b.Org, "agree": true})
		if r.Code != http.StatusNotFound && r.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST …/prefill-consent: %d", r.Code)
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

// Start fresh (S27 answers 1–2): the opening card of a Question Set holding another trust's answers
// has a third button, which clears every pre-filled answer in that set (any trust's), so it opens
// empty, and is recorded like the other two: POST …/prefill/{cid} {choice: start_fresh}, by the
// patient or staff for them. Only while nothing in the set has changed, and only once.
func TestStartFresh(t *testing.T) {
	t.Parallel()
	w := newCrossTrustWorld(t)
	again, healthAgain := w.later.Chapters[0], w.later.Chapters[1]
	fresh := map[string]string{"choice": "start_fresh"}
	row := func(e apitest.Episode, set uuid.UUID) (data string, revision int, choice string, by *uuid.UUID, kept bool) {
		t.Helper()
		var c *string
		if err := w.db.QueryRow(t.Context(), `SELECT data::text, revision, prefill_choice, prefill_choice_by, prefill IS NOT NULL AND prefill <> '{}'
			FROM episode_answers WHERE episode_form_id = $1 AND chapter_id = $2 AND actor = 'patient'`, e.FormID, set).Scan(&data, &revision, &c, &by, &kept); err != nil {
			t.Fatal(err)
		}
		if c != nil {
			choice = *c
		}
		return
	}
	x := w.person(t, "Fresh")
	getCrossForm(t, x.patient, patientPath(x.inA, "")) // pre-fills both sets from trust B
	path := patientPath(x.inA, "/prefill/"+again.String())

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("POST", path, fresh), http.StatusUnauthorized)
		refused(t, w.anon.Do("POST", epPath(w.a, x.inA, "/patient-form/prefill/"+again.String()), fresh), http.StatusUnauthorized)
	})
	t.Run("wrong role: another patient's session, and staff at the trust's other hospital, get 404", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		someoneElse := w.anon.NewBrowser(t).AsPatient(t, other.Link, other.DateOfBirth)
		refused(t, someoneElse.Do("POST", path, fresh), http.StatusNotFound)
		refused(t, w.site2.Do("POST", at(w.a2, "/episodes/"+x.inA.ID.String()+"/patient-form/prefill/"+again.String()), fresh), http.StatusNotFound)
	})
	t.Run("another trust: 404, trust B's staff included", func(t *testing.T) {
		refused(t, w.otherClin.Do("POST", epPath(w.a, x.inA, "/patient-form/prefill/"+again.String()), fresh), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", at(w.b, "/episodes/"+x.inA.ID.String()+"/patient-form/prefill/"+again.String()), fresh), http.StatusNotFound)
	})
	t.Run("invalid input: only for another trust's answers, only while unchanged, and not after another button", func(t *testing.T) {
		// A set pre-filled from the form's own trust only.
		own := w.person(t, "OwnOnly")
		earlier := w.fx.EpisodeOf(t, w.hq, w.a, own.inA.PatientID, w.cara, "")
		w.answeredBy(t, earlier, own.patient, "o_no")
		getCrossForm(t, own.patient, patientPath(own.inA, ""))
		r := own.patient.Do("POST", patientPath(own.inA, "/prefill/"+again.String()), fresh)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "another hospital") {
			t.Errorf("same trust only: %q", r.Message())
		}
		// Changed since the card showed (another tab saved).
		y := w.person(t, "Changed")
		f := getCrossForm(t, y.patient, patientPath(y.inA, ""))
		savedRevision(t, y.patient.Do("PUT", patientPath(y.inA, "/answers/"+healthAgain.String()), map[string]any{"data": map[string]any{"pf_diabetes": false}, "revision": f.Revisions[healthAgain.String()]}))
		r = y.patient.Do("POST", patientPath(y.inA, "/prefill/"+healthAgain.String()), fresh)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "reload") {
			t.Errorf("changed: %q", r.Message())
		}
		if data, _, choice, _, _ := row(y.inA, healthAgain); data != `{"pf_diabetes": false}` || choice != "" {
			t.Errorf("changed set %s %q", data, choice)
		}
		// After Check each answer (another tab): the first button stays.
		y.patient.Do("POST", patientPath(y.inA, "/prefill/"+again.String()), map[string]string{"choice": "check_each"}).Want(t, http.StatusNoContent)
		refused(t, y.patient.Do("POST", patientPath(y.inA, "/prefill/"+again.String()), fresh), http.StatusConflict)
		if data, _, choice, _, _ := row(y.inA, again); choice != "check_each" || data == "{}" {
			t.Errorf("after Check each: %s %q", data, choice)
		}
		bad := x.patient.Do("POST", path, map[string]string{"choice": "start fresh"})
		refused(t, bad, http.StatusBadRequest)
		if !strings.Contains(bad.Message(), "Start fresh") { // every button named (verification 1, finding 4)
			t.Errorf("unknown choice: %q", bad.Message())
		}
	})
	t.Run("happy path: the patient's Start fresh empties the set, is recorded, and the set isn't filled again", func(t *testing.T) {
		x.patient.Do("POST", path, fresh).Want(t, http.StatusNoContent)
		data, revision, choice, by, kept := row(x.inA, again)
		// The answers go and the revision moves on, so a tab still showing them gets the 409; the map
		// stays (what was copied, for the clinician's line).
		if data != "{}" || revision != 2 || choice != "start_fresh" || by != nil || !kept {
			t.Errorf("after Start fresh: %s rev %d %q by %v map kept %v", data, revision, choice, by, kept)
		}
		f := getCrossForm(t, x.patient, patientPath(x.inA, ""))
		if len(f.Answers[again.String()]) != 0 || f.Revisions[again.String()] != 2 {
			t.Errorf("filled again: %v rev %d", f.Answers[again.String()], f.Revisions[again.String()])
		}
		if p := f.Prefill[again.String()]; p.Choice == nil || *p.Choice != "start_fresh" {
			t.Errorf("prefill %+v", p)
		}
		// The other set keeps its answers.
		if got := answersJSON(t, f.Answers[healthAgain.String()]); got != `{"pf_diabetes":true}` {
			t.Errorf("Health again %s", got)
		}
		if got := homeState(t, x.patient, x.inA.ID); got != "in_progress" {
			t.Errorf("home state %q", got)
		}
		// Pressed again (another tab): nothing more happens.
		x.patient.Do("POST", path, fresh).Want(t, http.StatusNoContent)
		if _, revision, _, _, _ := row(x.inA, again); revision != 2 {
			t.Errorf("revision %d after a second press", revision)
		}
		// A tab that still shows the answers can't save them back.
		refused(t, x.patient.Do("PUT", patientPath(x.inA, "/answers/"+again.String()), map[string]any{"data": map[string]any{"q_meds": []any{}}, "revision": 1}), http.StatusConflict)
		if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'link.prefill' AND $1 = ANY(episode_ids) AND status = 204`, x.inA.ID); n != 2 {
			t.Errorf("%d link.prefill rows", n)
		}
	})
	t.Run("after Start fresh, a stale card's other buttons are refused, so an empty set is never sent as checked (verification 1, finding 1)", func(t *testing.T) {
		y := w.person(t, "FreshStale")
		getCrossForm(t, y.patient, patientPath(y.inA, ""))
		y.patient.Do("POST", patientPath(y.inA, "/prefill/"+again.String()), fresh).Want(t, http.StatusNoContent) // one tab
		for _, other := range []string{"still_correct", "check_each"} { // the card still open in another tab
			r := y.patient.Do("POST", patientPath(y.inA, "/prefill/"+again.String()), map[string]string{"choice": other})
			refused(t, r, http.StatusConflict)
			if !strings.Contains(r.Message(), "reload") {
				t.Errorf("%s after Start fresh: %q", other, r.Message())
			}
		}
		// Staff on the patient's behalf, with the card open after the patient started fresh.
		z := w.person(t, "FreshStaleBehalf")
		getCrossForm(t, w.clinician, epPath(w.a, z.inA, "/patient-form"))
		z.patient.Do("POST", patientPath(z.inA, "/prefill/"+healthAgain.String()), fresh).Want(t, http.StatusNoContent)
		refused(t, w.clinician.Do("POST", epPath(w.a, z.inA, "/patient-form/prefill/"+healthAgain.String()), map[string]string{"choice": "still_correct"}), http.StatusConflict)
		if _, _, choice, _, _ := row(z.inA, healthAgain); choice != "start_fresh" {
			t.Errorf("choice %q", choice)
		}
	})
	t.Run("happy path: staff for the patient, marked as theirs", func(t *testing.T) {
		z := w.person(t, "FreshBehalf")
		getCrossForm(t, w.clinician, epPath(w.a, z.inA, "/patient-form"))
		w.clinician.Do("POST", epPath(w.a, z.inA, "/patient-form/prefill/"+healthAgain.String()), fresh).Want(t, http.StatusNoContent)
		if data, _, choice, by, _ := row(z.inA, healthAgain); data != "{}" || choice != "start_fresh" || by == nil || *by != w.cara.ID {
			t.Errorf("on behalf: %s %q by %v", data, choice, by)
		}
	})
	t.Run("once sent: 409", func(t *testing.T) {
		z := w.person(t, "FreshSent")
		getCrossForm(t, z.patient, patientPath(z.inA, ""))
		if _, err := w.db.Exec(t.Context(), `UPDATE episode_forms SET status = 'submitted', submitted_at = now() WHERE id = $1`, z.inA.FormID); err != nil {
			t.Fatal(err)
		}
		refused(t, z.patient.Do("POST", patientPath(z.inA, "/prefill/"+again.String()), fresh), http.StatusConflict)
		if data, _, _, _, _ := row(z.inA, again); data == "{}" {
			t.Error("cleared after sending")
		}
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
