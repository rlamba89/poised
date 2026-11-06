//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Assessments (plan C7, session S47; §9): a clinician adds STOP-Bang to an episode (S47 answers 1),
// completes it from what the browser pre-filled, each item with its source, and confirms it:
//
//   - GET  …/episodes/{eid}/assessments: the episode's assessments with the clinician's answers,
//     their sources and the Confirm, the ones still to add, and the trust's observations.precedence;
//   - POST …/episodes/{eid}/assessments {assessment};
//   - PUT  …/episodes/{eid}/assessments/{fid} {data, sources, revision}: save (a Confirm is cleared);
//   - POST …/episodes/{eid}/assessments/{fid}/confirm {data, sources, revision}: save and confirm.

const assessmentsOff = "Assessments aren't turned on at this hospital."

func asmPath(h apitest.Hospital, e apitest.Episode, rest string) string {
	return epPath(h, e, "/assessments"+rest)
}

// addStopBang adds STOP-Bang to the episode as Cara and returns its form's id.
func addStopBang(t *testing.T, w *clinic, e apitest.Episode) uuid.UUID {
	t.Helper()
	r := w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"})
	r.Want(t, http.StatusCreated)
	var out struct{ ID uuid.UUID }
	r.Decode(t, &out)
	return out.ID
}

// stopBangAnswers: S, T, O and P from the HQ, the BMI from an observation, the neck the clinician's.
func stopBangAnswers(revision int) map[string]any {
	return map[string]any{
		"data": map[string]any{"stopbang_snoring": "o_yes", "stopbang_tired": "o_yes", "stopbang_observed": "o_no",
			"stopbang_pressure": "o_yes", "stopbang_bmi": 36.4, "stopbang_neck": "o_no"},
		"sources": map[string]string{"stopbang_snoring": "hq", "stopbang_tired": "hq", "stopbang_observed": "hq",
			"stopbang_pressure": "hq", "stopbang_bmi": "observation", "stopbang_neck": "clinician"},
		"revision": revision,
	}
}

// asmView is GET …/assessments.
type asmView struct {
	Assessments []struct {
		ID            uuid.UUID
		Assessment    string
		Name          string
		ChapterID     uuid.UUID
		Content       json.RawMessage
		Data          map[string]any
		Sources       map[string]string
		Revision      int
		ConfirmedAt   *time.Time
		UpdatedByName *string
	}
	Available     []struct{ ID, Name string }
	ObsPrecedence *bool
}

func listAssessments(t *testing.T, c *apitest.Client, h apitest.Hospital, e apitest.Episode) asmView {
	t.Helper()
	r := c.Do("GET", asmPath(h, e, ""), nil)
	r.Want(t, http.StatusOK)
	var v asmView
	r.Decode(t, &v)
	return v
}

// asmRow is the clinician's answers row of an assessment form.
type asmRow struct {
	Data, Sources map[string]any
	Revision      int
	ValidatedAt   *time.Time
	UpdatedBy     *uuid.UUID
}

func asmRowOf(t *testing.T, w *clinic, form uuid.UUID) (asmRow, bool) {
	t.Helper()
	var r asmRow
	err := w.db.QueryRow(t.Context(), `SELECT data, sources, revision, validated_at, updated_by FROM episode_answers
		WHERE episode_form_id = $1 AND actor = 'clinician'`, form).Scan(&r.Data, &r.Sources, &r.Revision, &r.ValidatedAt, &r.UpdatedBy)
	if err != nil {
		return r, false
	}
	return r, true
}

func asmForms(t *testing.T, w *clinic, e apitest.Episode) int {
	t.Helper()
	return count(t, w.db, `SELECT count(*) FROM episode_forms WHERE episode_id = $1 AND kind = 'assessment'`, e.ID)
}

func TestAddAssessment(t *testing.T) {
	t.Parallel()
	w := newClinic(t)

	t.Run("happy path: STOP-Bang on the episode, as a platform questionnaire's published version", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		var kind, assessment, status string
		var org *uuid.UUID
		if err := w.db.QueryRow(t.Context(), `SELECT f.kind, f.assessment, v.status, q.org_id FROM episode_forms f
			JOIN questionnaire_versions v ON v.id = f.version_id JOIN questionnaires q ON q.id = v.questionnaire_id
			WHERE f.id = $1 AND f.episode_id = $2`, id, e.ID).Scan(&kind, &assessment, &status, &org); err != nil {
			t.Fatal(err)
		}
		if kind != "assessment" || assessment != "stopbang" || status != "published" || org != nil {
			t.Errorf("form: %s %s %s %v", kind, assessment, status, org)
		}
		v := listAssessments(t, w.clinician, w.a, e)
		if len(v.Assessments) != 1 || v.Assessments[0].Name != "STOP-Bang" || v.Assessments[0].Revision != 0 || v.Assessments[0].ConfirmedAt != nil {
			t.Errorf("list: %+v", v.Assessments)
		}
		if len(v.Available) != 0 {
			t.Errorf("STOP-Bang is offered again: %+v", v.Available)
		}
	})
	t.Run("a second episode uses the same version: it's published once", func(t *testing.T) {
		addStopBang(t, w, w.fx.Episode(t, w.hq, w.cara))
		if n := count(t, w.db, `SELECT count(*) FROM questionnaire_versions v JOIN questionnaires q ON q.id = v.questionnaire_id
			WHERE q.assessment = 'stopbang'`); n != 1 {
			t.Errorf("%d versions", n)
		}
	})
	t.Run("changed wording gets a new version; the episodes that have the old one keep it", func(t *testing.T) {
		old := w.fx.Episode(t, w.hq, w.cara)
		oldForm := addStopBang(t, w, old)
		sqlExec(t, w, `UPDATE chapters SET content = '{"pages": []}' WHERE version_id IN (SELECT v.id FROM questionnaire_versions v
			JOIN questionnaires q ON q.id = v.questionnaire_id WHERE q.assessment = 'stopbang')`)
		addStopBang(t, w, w.fx.Episode(t, w.hq, w.cara))
		if n := count(t, w.db, `SELECT count(*) FROM questionnaire_versions v JOIN questionnaires q ON q.id = v.questionnaire_id
			WHERE q.assessment = 'stopbang' AND v.status = 'published'`); n != 1 {
			t.Errorf("%d published versions, want 1", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_forms f JOIN questionnaire_versions v ON v.id = f.version_id
			WHERE f.id = $1 AND v.status = 'retired'`, oldForm); n != 1 {
			t.Error("the old episode's form moved off its version")
		}
	})
	t.Run("twice on one episode: 409, one form", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		addStopBang(t, w, e)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusConflict)
		if n := asmForms(t, w, e); n != 1 {
			t.Errorf("%d forms", n)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, w.anon.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusUnauthorized)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("the role: a clinician at the trust's other hospital can't add one here (404)", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, w.site2.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusNotFound)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, w.otherClin.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", asmPath(w.b, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusNotFound)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("invalid input: an unknown assessment or not JSON is 400", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "dasi"}), http.StatusBadRequest)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, ""), "nope"), http.StatusBadRequest)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("invalid input: a Confirm needs every item answered (verification 1); a save doesn't", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		for _, body := range []map[string]any{
			{"data": map[string]any{}, "sources": map[string]string{}, "revision": 0},
			func() map[string]any {
				b := stopBangAnswers(0)
				delete(b["data"].(map[string]any), "stopbang_neck")
				delete(b["sources"].(map[string]string), "stopbang_neck")
				return b
			}(),
		} {
			r := w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), body)
			refused(t, r, http.StatusBadRequest)
			if !strings.Contains(string(r.Body), "Answer every item before confirming") {
				t.Errorf("message: %s", r.Body)
			}
		}
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
		w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), map[string]any{"data": map[string]any{}, "sources": map[string]string{}, "revision": 0}).
			Want(t, http.StatusOK)
	})
	t.Run("a cancelled or archived episode: 409", func(t *testing.T) {
		for _, status := range []string{"cancelled", "archived"} {
			e := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, e, status)
			refused(t, w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusConflict)
			if asmForms(t, w, e) != 0 {
				t.Error("added")
			}
		}
	})
	t.Run("assessments turned off at the hospital: 403, and the list too", func(t *testing.T) {
		w := newClinic(t)
		sqlExec(t, w, `UPDATE hospitals SET settings = settings || '{"feature.assessments": false}' WHERE id = $1`, w.a.ID)
		e := w.fx.Episode(t, w.hq, w.cara)
		r := w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"})
		refused(t, r, http.StatusForbidden)
		if r.Message() != assessmentsOff {
			t.Errorf("message %q", r.Message())
		}
		refused(t, w.clinician.Do("GET", asmPath(w.a, e, ""), nil), http.StatusForbidden)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
}

func TestListAssessments(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)

	t.Run("happy path: none yet, STOP-Bang to add, and the trust's precedence (on by default)", func(t *testing.T) {
		v := listAssessments(t, w.clinician, w.a, e)
		if len(v.Assessments) != 0 || len(v.Available) != 1 || v.Available[0].ID != "stopbang" || v.Available[0].Name != "STOP-Bang" {
			t.Errorf("%+v", v)
		}
		if v.ObsPrecedence == nil || !*v.ObsPrecedence {
			t.Errorf("obsPrecedence %v, want true", v.ObsPrecedence)
		}
		sqlExec(t, w, `UPDATE orgs SET settings = settings || '{"observations.precedence": false}' WHERE id = $1`, w.a.Org)
		if v := listAssessments(t, w.clinician, w.a, e); v.ObsPrecedence == nil || *v.ObsPrecedence {
			t.Error("the trust's setting is off")
		}
	})
	t.Run("pre-fill sources are reported per item, with the content to fill", func(t *testing.T) {
		id := addStopBang(t, w, e)
		w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)).Want(t, http.StatusOK)
		a := listAssessments(t, w.clinician, w.a, e).Assessments[0]
		if a.Sources["stopbang_bmi"] != "observation" || a.Sources["stopbang_snoring"] != "hq" || a.Sources["stopbang_neck"] != "clinician" {
			t.Errorf("sources %v", a.Sources)
		}
		if a.Data["stopbang_bmi"] != 36.4 || a.Revision != 1 || a.UpdatedByName == nil || len(a.Content) < 100 {
			t.Errorf("%+v", a)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		refused(t, w.anon.Do("GET", asmPath(w.a, e, ""), nil), http.StatusUnauthorized)
	})
	t.Run("another hospital of the trust: 404", func(t *testing.T) {
		refused(t, w.site2.Do("GET", asmPath(w.a, e, ""), nil), http.StatusNotFound)
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		refused(t, w.otherClin.Do("GET", asmPath(w.a, e, ""), nil), http.StatusNotFound)
		refused(t, w.otherClin.Do("GET", asmPath(w.b, e, ""), nil), http.StatusNotFound)
	})
	t.Run("invalid input: a malformed episode id is 404", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/nope/assessments"), nil), http.StatusNotFound)
	})
}

func TestSaveAndConfirmAssessment(t *testing.T) {
	t.Parallel()
	w := newClinic(t)

	t.Run("happy path: save, then confirm (who and when); a change clears the Confirm", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)).Want(t, http.StatusOK)
		row, _ := asmRowOf(t, w, id)
		if row.Revision != 1 || row.ValidatedAt != nil || row.Sources["stopbang_bmi"] != "observation" || row.Data["stopbang_neck"] != "o_no" {
			t.Errorf("saved: %+v", row)
		}
		r := w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(1))
		r.Want(t, http.StatusOK)
		var out struct{ Revision int }
		r.Decode(t, &out)
		row, _ = asmRowOf(t, w, id)
		if out.Revision != 2 || row.Revision != 2 || row.ValidatedAt == nil || row.UpdatedBy == nil || *row.UpdatedBy != w.cara.ID {
			t.Errorf("confirmed: %+v (%d)", row, out.Revision)
		}
		changed := stopBangAnswers(2)
		changed["data"].(map[string]any)["stopbang_neck"] = "o_yes"
		w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), changed).Want(t, http.StatusOK)
		if row, _ = asmRowOf(t, w, id); row.ValidatedAt != nil || row.Data["stopbang_neck"] != "o_yes" {
			t.Errorf("the change kept the Confirm: %+v", row)
		}
		if a := listAssessments(t, w.clinician, w.a, e).Assessments[0]; a.ConfirmedAt != nil {
			t.Error("listed as confirmed")
		}
	})
	t.Run("confirm with no save first", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)).Want(t, http.StatusOK)
		if a := listAssessments(t, w.clinician, w.a, e).Assessments[0]; a.ConfirmedAt == nil || a.UpdatedByName == nil || a.Revision != 1 {
			t.Errorf("%+v", a)
		}
	})
	t.Run("a Confirm stores the patient's age and sex as of that day, source profile (S47 answers 7)", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		b := stopBangAnswers(0)
		b["data"].(map[string]any)["patientAge"] = 62
		b["data"].(map[string]any)["patientSex"] = "male"
		b["sources"].(map[string]string)["patientAge"] = "profile"
		b["sources"].(map[string]string)["patientSex"] = "profile"
		w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), b).Want(t, http.StatusOK)
		row, _ := asmRowOf(t, w, id)
		if row.ValidatedAt == nil || row.Data["patientAge"] != float64(62) || row.Data["patientSex"] != "male" || row.Sources["patientAge"] != "profile" {
			t.Errorf("stored: %+v", row)
		}
		a := listAssessments(t, w.clinician, w.a, e).Assessments[0]
		if a.Data["patientAge"] != float64(62) || a.Sources["patientSex"] != "profile" {
			t.Errorf("listed: %+v", a)
		}
	})
	t.Run("a stale revision is 409, and nothing changes", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)).Want(t, http.StatusOK)
		for _, path := range []string{"", "/confirm"} {
			method := map[string]string{"": "PUT", "/confirm": "POST"}[path]
			r := w.clinician.Do(method, asmPath(w.a, e, "/"+id.String()+path), stopBangAnswers(0))
			refused(t, r, http.StatusConflict)
		}
		if row, _ := asmRowOf(t, w, id); row.Revision != 1 || row.ValidatedAt != nil {
			t.Errorf("%+v", row)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		refused(t, w.anon.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)), http.StatusUnauthorized)
		refused(t, w.anon.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusUnauthorized)
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
	t.Run("the role: another hospital's clinician: 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		refused(t, w.site2.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)), http.StatusNotFound)
		refused(t, w.site2.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusNotFound)
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
	t.Run("another trust's: 404, and another episode's assessment is 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		refused(t, w.otherClin.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", asmPath(w.b, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusNotFound)
		other := w.fx.Episode(t, w.hq, w.cara)
		refused(t, w.clinician.Do("PUT", asmPath(w.a, other, "/"+id.String()), stopBangAnswers(0)), http.StatusNotFound)
		refused(t, w.clinician.Do("PUT", asmPath(w.a, e, "/"+e.FormID.String()), stopBangAnswers(0)), http.StatusNotFound) // its HQ form
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
	t.Run("invalid input: 400", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		bad := func(change func(map[string]any)) map[string]any {
			b := stopBangAnswers(0)
			change(b)
			return b
		}
		for _, body := range []any{
			"nope",
			bad(func(b map[string]any) { delete(b, "revision") }),
			bad(func(b map[string]any) { b["data"].(map[string]any)["stopbang_neck"] = "o_maybe" }),
			bad(func(b map[string]any) { b["data"].(map[string]any)["q_other"] = "o_yes" }),
			bad(func(b map[string]any) { b["data"].(map[string]any)["stopbang_bmi"] = -3 }),
			bad(func(b map[string]any) { b["sources"].(map[string]string)["stopbang_neck"] = "guess" }),
			bad(func(b map[string]any) { delete(b["sources"].(map[string]string), "stopbang_bmi") }),
			bad(func(b map[string]any) { b["sources"].(map[string]string)["stopbang_neck"] = "profile" }), // the age and sex only
			bad(func(b map[string]any) {
				b["data"].(map[string]any)["patientAge"] = 62
				b["sources"].(map[string]string)["patientAge"] = "clinician"
			}),
		} {
			refused(t, w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), body), http.StatusBadRequest)
			refused(t, w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), body), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("PUT", asmPath(w.a, e, "/nope"), stopBangAnswers(0)), http.StatusNotFound)
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
	t.Run("invalid input: a Confirm needs every item answered (verification 1); a save doesn't", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		for _, body := range []map[string]any{
			{"data": map[string]any{}, "sources": map[string]string{}, "revision": 0},
			func() map[string]any {
				b := stopBangAnswers(0)
				delete(b["data"].(map[string]any), "stopbang_neck")
				delete(b["sources"].(map[string]string), "stopbang_neck")
				return b
			}(),
		} {
			r := w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), body)
			refused(t, r, http.StatusBadRequest)
			if !strings.Contains(string(r.Body), "Answer every item before confirming") {
				t.Errorf("message: %s", r.Body)
			}
		}
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
		w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), map[string]any{"data": map[string]any{}, "sources": map[string]string{}, "revision": 0}).
			Want(t, http.StatusOK)
	})
	t.Run("a cancelled or archived episode: 409", func(t *testing.T) {
		for _, status := range []string{"cancelled", "archived"} {
			e := w.fx.Episode(t, w.hq, w.cara)
			id := addStopBang(t, w, e)
			setStatus(t, w, e, status)
			refused(t, w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusConflict)
			if _, ok := asmRowOf(t, w, id); ok {
				t.Error("saved")
			}
		}
	})
	t.Run("turned off at the hospital: 403", func(t *testing.T) {
		w := newClinic(t)
		e := w.fx.Episode(t, w.hq, w.cara)
		id := addStopBang(t, w, e)
		sqlExec(t, w, `UPDATE hospitals SET settings = settings || '{"feature.assessments": false}' WHERE id = $1`, w.a.ID)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusForbidden)
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
}

// Patients never see an assessment (§9): their routes find only the episode's HQ.
func TestPatientNeverSeesAssessments(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	id := addStopBang(t, w, e)
	var chapter uuid.UUID
	if err := w.db.QueryRow(t.Context(), `SELECT c.id FROM episode_forms f JOIN chapters c ON c.version_id = f.version_id WHERE f.id = $1`, id).Scan(&chapter); err != nil {
		t.Fatal(err)
	}
	p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
	r := p.Do("GET", "/api/p/episodes/"+e.ID.String(), nil)
	r.Want(t, http.StatusOK)
	if body := r.String(); strings.Contains(body, "stopbang") || strings.Contains(body, chapter.String()) {
		t.Error("the patient's form holds the assessment")
	}
	refused(t, p.Do("PUT", "/api/p/episodes/"+e.ID.String()+"/answers/"+chapter.String(), map[string]any{"data": map[string]any{"stopbang_neck": "o_yes"}, "revision": 0}), http.StatusNotFound)
	if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE chapter_id = $1`, chapter); n != 0 {
		t.Errorf("%d rows saved", n)
	}
}

func TestEpisodeSaysWhetherAssessmentsAreOn(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	on := func() bool {
		var v struct {
			AssessmentsOn *bool `json:"assessmentsOn"`
		}
		r := w.clinician.Do("GET", epPath(w.a, e, ""), nil)
		r.Want(t, http.StatusOK)
		r.Decode(t, &v)
		if v.AssessmentsOn == nil {
			t.Fatalf("no assessmentsOn: %s", r)
		}
		return *v.AssessmentsOn
	}
	if !on() {
		t.Error("on by default")
	}
	sqlExec(t, w, `UPDATE hospitals SET settings = settings || '{"feature.assessments": false}' WHERE id = $1`, w.a.ID)
	if on() {
		t.Error("off at this hospital")
	}
}

func TestAssessmentsAudited(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	var id uuid.UUID
	for _, c := range []struct {
		action string
		req    func() *apitest.Resp
		want   int
	}{
		{"assessment.add", func() *apitest.Resp {
			r := w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"})
			var out struct{ ID uuid.UUID }
			r.Decode(t, &out)
			id = out.ID
			return r
		}, http.StatusCreated},
		{"assessment.list", func() *apitest.Resp { return w.clinician.Do("GET", asmPath(w.a, e, ""), nil) }, http.StatusOK},
		{"assessment.save", func() *apitest.Resp {
			return w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0))
		}, http.StatusOK},
		{"assessment.confirm", func() *apitest.Resp {
			return w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(1))
		}, http.StatusOK},
	} {
		a := oneAudit(t, w, c.action, c.req, c.want)
		if !slices.Contains(a.Episodes, e.ID) {
			t.Errorf("%s: audit = %+v", c.action, a)
		}
	}
}
