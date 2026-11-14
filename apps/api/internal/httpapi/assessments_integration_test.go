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

// stopBangAnswers: S, T, O and P from the HQ, the BMI from an observation, the neck the
// clinician's, the age and sex the profile's (the browser sends them on every save, S47 answers 7).
func stopBangAnswers(revision int) map[string]any {
	return map[string]any{
		"data": map[string]any{"stopbang_snoring": "o_yes", "stopbang_tired": "o_yes", "stopbang_observed": "o_no",
			"stopbang_pressure": "o_yes", "stopbang_bmi": 36.4, "stopbang_neck": "o_no", "patientAge": 62, "patientSex": "male"},
		"sources": map[string]string{"stopbang_snoring": "hq", "stopbang_tired": "hq", "stopbang_observed": "hq",
			"stopbang_pressure": "hq", "stopbang_bmi": "observation", "stopbang_neck": "clinician", "patientAge": "profile", "patientSex": "profile"},
		"revision": revision,
	}
}

// recordAs sets the episode's patient's record: the sex, and a date of birth that makes them `age`
// today, a month clear of a birthday so no time zone changes it. A "profile" age or sex must be the
// record's (S48 verification 1).
func recordAs(t *testing.T, w *clinic, e *apitest.Episode, age int, sex string) {
	t.Helper()
	e.DateOfBirth = time.Now().AddDate(-age, 0, -30).Format("2006-01-02")
	sqlExec(t, w, `UPDATE patients SET sex = $2, date_of_birth = $3 WHERE id = $1`, e.PatientID, sex, e.DateOfBirth)
}

// asmEpisode is an episode whose patient is 62 and male, the age and sex the tests' answers send
// from the profile (stopBangAnswers).
func asmEpisode(t *testing.T, w *clinic) apitest.Episode {
	t.Helper()
	e := w.fx.Episode(t, w.hq, w.cara)
	recordAs(t, w, &e, 62, "male")
	return e
}

// addAssessment adds an assessment to the episode as Cara and returns its form's id.
func addAssessment(t *testing.T, w *clinic, e apitest.Episode, id string) uuid.UUID {
	t.Helper()
	r := w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": id})
	r.Want(t, http.StatusCreated)
	var out struct{ ID uuid.UUID }
	r.Decode(t, &out)
	return out.ID
}

// answers builds a save's body: each key's value and source.
func answers(revision int, items map[string][2]any) map[string]any {
	data, sources := map[string]any{}, map[string]string{}
	for k, v := range items {
		data[k], sources[k] = v[0], v[1].(string)
	}
	return map[string]any{"data": data, "sources": sources, "revision": revision}
}

// dasiAnswers: the twelve items, each from the HQ, Yes to those given.
func dasiAnswers(revision int, yes ...string) map[string]any {
	items := map[string][2]any{}
	for _, k := range []string{"dasi_selfcare", "dasi_walk_indoors", "dasi_walk_block", "dasi_stairs", "dasi_run", "dasi_light_work",
		"dasi_moderate_work", "dasi_heavy_work", "dasi_yardwork", "dasi_sexual", "dasi_moderate_recreation", "dasi_strenuous_sports"} {
		v := "o_no"
		if slices.Contains(yes, k) {
			v = "o_yes"
		}
		items[k] = [2]any{v, "hq"}
	}
	return answers(revision, items)
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
		e := asmEpisode(t, w)
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
		if slices.ContainsFunc(v.Available, func(a struct{ ID, Name string }) bool { return a.ID == "stopbang" }) {
			t.Errorf("STOP-Bang is offered again: %+v", v.Available)
		}
	})
	t.Run("a second episode uses the same version: it's published once", func(t *testing.T) {
		addStopBang(t, w, asmEpisode(t, w))
		if n := count(t, w.db, `SELECT count(*) FROM questionnaire_versions v JOIN questionnaires q ON q.id = v.questionnaire_id
			WHERE q.assessment = 'stopbang'`); n != 1 {
			t.Errorf("%d versions", n)
		}
	})
	t.Run("changed wording gets a new version; the episodes that have the old one keep it", func(t *testing.T) {
		old := asmEpisode(t, w)
		oldForm := addStopBang(t, w, old)
		sqlExec(t, w, `UPDATE chapters SET content = '{"pages": []}' WHERE version_id IN (SELECT v.id FROM questionnaire_versions v
			JOIN questionnaires q ON q.id = v.questionnaire_id WHERE q.assessment = 'stopbang')`)
		addStopBang(t, w, asmEpisode(t, w))
		if n := count(t, w.db, `SELECT count(*) FROM questionnaire_versions v JOIN questionnaires q ON q.id = v.questionnaire_id
			WHERE q.assessment = 'stopbang' AND v.status = 'published'`); n != 1 {
			t.Errorf("%d published versions, want 1", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_forms f JOIN questionnaire_versions v ON v.id = f.version_id
			WHERE f.id = $1 AND v.status = 'retired'`, oldForm); n != 1 {
			t.Error("the old episode's form moved off its version")
		}
	})
	t.Run("every assessment (S48, S49): each its own platform questionnaire, all eight on one episode", func(t *testing.T) {
		e := asmEpisode(t, w)
		for _, id := range []string{"stopbang", "dasi", "prisma7", "apfel", "must", "morse", "vte", "rockwood"} {
			addAssessment(t, w, e, id)
		}
		v := listAssessments(t, w.clinician, w.a, e)
		var names []string
		for _, a := range v.Assessments {
			names = append(names, a.Name)
		}
		if strings.Join(names, ",") != "STOP-Bang,DASI,PRISMA-7,Apfel PONV,MUST,Morse Fall Scale,VTE risk,Clinical Frailty Scale" || len(v.Available) != 0 {
			t.Errorf("listed %v, still to add %+v", names, v.Available)
		}
		if n := count(t, w.db, `SELECT count(DISTINCT q.id) FROM episode_forms f JOIN questionnaire_versions v ON v.id = f.version_id
			JOIN questionnaires q ON q.id = v.questionnaire_id WHERE f.episode_id = $1 AND q.org_id IS NULL`, e.ID); n != 8 {
			t.Errorf("%d platform questionnaires, want 8", n)
		}
	})
	t.Run("a version with the same content is reused, not only the latest (S47 verification 4: two builds at once)", func(t *testing.T) {
		w := newClinic(t)
		first := addAssessment(t, w, asmEpisode(t, w), "dasi")
		// Another build (an overlapping deploy) published different wording since.
		sqlExec(t, w, `UPDATE questionnaire_versions SET status = 'retired' WHERE id = (SELECT version_id FROM episode_forms WHERE id = $1)`, first)
		sqlExec(t, w, `INSERT INTO questionnaire_versions (questionnaire_id, version_no, status)
			SELECT questionnaire_id, 2, 'published' FROM questionnaire_versions WHERE id = (SELECT version_id FROM episode_forms WHERE id = $1)`, first)
		sqlExec(t, w, `INSERT INTO chapters (version_id, position, name, audience, content)
			SELECT v.id, 1, 'DASI', 'clinician', '{"pages": []}' FROM questionnaire_versions v JOIN questionnaires q ON q.id = v.questionnaire_id
			WHERE q.assessment = 'dasi' AND v.version_no = 2`)
		second := addAssessment(t, w, asmEpisode(t, w), "dasi")
		if n := count(t, w.db, `SELECT count(DISTINCT version_id) FROM episode_forms WHERE id IN ($1, $2)`, first, second); n != 1 {
			t.Error("this build's wording was published again")
		}
		if n := count(t, w.db, `SELECT count(*) FROM questionnaire_versions v JOIN questionnaires q ON q.id = v.questionnaire_id
			WHERE q.assessment = 'dasi'`); n != 2 {
			t.Errorf("%d versions, want 2", n)
		}
	})
	t.Run("a renamed assessment's questionnaire takes the new name with its new version (S47 verification 4)", func(t *testing.T) {
		w := newClinic(t)
		addAssessment(t, w, asmEpisode(t, w), "prisma7")
		sqlExec(t, w, `UPDATE questionnaires SET name = 'PRISMA 7 (old name)' WHERE assessment = 'prisma7'`)
		sqlExec(t, w, `UPDATE chapters SET content = '{"pages": []}' WHERE version_id IN (SELECT v.id FROM questionnaire_versions v
			JOIN questionnaires q ON q.id = v.questionnaire_id WHERE q.assessment = 'prisma7')`)
		e := asmEpisode(t, w)
		addAssessment(t, w, e, "prisma7")
		if a := listAssessments(t, w.clinician, w.a, e).Assessments[0]; a.Name != "PRISMA-7" {
			t.Errorf("name %q", a.Name)
		}
	})
	t.Run("each form keeps its own version's name: a rename shows on new forms only (S48 verification 1, Rahul 7 Nov)", func(t *testing.T) {
		w := newClinic(t)
		old := asmEpisode(t, w)
		addAssessment(t, w, old, "prisma7")
		// Version 1 as it would be had it been published under an older name, with other content.
		sqlExec(t, w, `UPDATE questionnaires SET name = 'PRISMA 7 (old name)' WHERE assessment = 'prisma7'`)
		sqlExec(t, w, `UPDATE chapters SET name = 'PRISMA 7 (old name)', content = '{"pages": []}' WHERE version_id IN (SELECT v.id
			FROM questionnaire_versions v JOIN questionnaires q ON q.id = v.questionnaire_id WHERE q.assessment = 'prisma7')`)
		e := asmEpisode(t, w)
		addAssessment(t, w, e, "prisma7")
		if a := listAssessments(t, w.clinician, w.a, old).Assessments[0]; a.Name != "PRISMA 7 (old name)" {
			t.Errorf("the old episode's name %q, want its own version's", a.Name)
		}
		if a := listAssessments(t, w.clinician, w.a, e).Assessments[0]; a.Name != "PRISMA-7" {
			t.Errorf("the new episode's name %q", a.Name)
		}
	})
	t.Run("twice on one episode: 409, one form", func(t *testing.T) {
		e := asmEpisode(t, w)
		addStopBang(t, w, e)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusConflict)
		if n := asmForms(t, w, e); n != 1 {
			t.Errorf("%d forms", n)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := asmEpisode(t, w)
		refused(t, w.anon.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusUnauthorized)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("the role: a clinician at the trust's other hospital can't add one here (404)", func(t *testing.T) {
		e := asmEpisode(t, w)
		refused(t, w.site2.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusNotFound)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e := asmEpisode(t, w)
		refused(t, w.otherClin.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", asmPath(w.b, e, ""), map[string]string{"assessment": "stopbang"}), http.StatusNotFound)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("invalid input: an unknown assessment or not JSON is 400", func(t *testing.T) {
		e := asmEpisode(t, w)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, ""), map[string]string{"assessment": "mmse"}), http.StatusBadRequest)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, ""), "nope"), http.StatusBadRequest)
		if asmForms(t, w, e) != 0 {
			t.Error("added")
		}
	})
	t.Run("invalid input: a Confirm needs every item answered (verification 1); a save doesn't", func(t *testing.T) {
		e := asmEpisode(t, w)
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
			e := asmEpisode(t, w)
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
		e := asmEpisode(t, w)
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
	e := asmEpisode(t, w)

	t.Run("happy path: none yet, STOP-Bang to add, and the trust's precedence (on by default)", func(t *testing.T) {
		v := listAssessments(t, w.clinician, w.a, e)
		var offered []string
		for _, a := range v.Available {
			offered = append(offered, a.ID+":"+a.Name)
		}
		if len(v.Assessments) != 0 || strings.Join(offered, ",") != "stopbang:STOP-Bang,dasi:DASI,prisma7:PRISMA-7,apfel:Apfel PONV,"+
			"must:MUST,morse:Morse Fall Scale,vte:VTE risk,rockwood:Clinical Frailty Scale" {
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
	t.Run("pre-fill sources per item for Apfel (S48): the HQ's two, the profile's sex, the clinician's opioids", func(t *testing.T) {
		e := asmEpisode(t, w)
		recordAs(t, w, &e, 62, "female")
		id := addAssessment(t, w, e, "apfel")
		w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), answers(0, map[string][2]any{
			"apfel_smoking": {"o_no", "hq"}, "apfel_ponv": {"o_yes", "hq"}, "apfel_opioids": {"o_yes", "clinician"}, "patientSex": {"female", "profile"},
		})).Want(t, http.StatusOK)
		a := listAssessments(t, w.clinician, w.a, e).Assessments[0]
		if a.Assessment != "apfel" || a.Sources["apfel_smoking"] != "hq" || a.Sources["apfel_opioids"] != "clinician" || a.Sources["patientSex"] != "profile" {
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
		e := asmEpisode(t, w)
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
		e := asmEpisode(t, w)
		id := addStopBang(t, w, e)
		w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)).Want(t, http.StatusOK)
		if a := listAssessments(t, w.clinician, w.a, e).Assessments[0]; a.ConfirmedAt == nil || a.UpdatedByName == nil || a.Revision != 1 {
			t.Errorf("%+v", a)
		}
	})
	t.Run("a Confirm stores the patient's age and sex as of that day, source profile (S47 answers 7)", func(t *testing.T) {
		e := asmEpisode(t, w)
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
	t.Run("an age or sex that isn't the record's is 400, and nothing is stored (S48 verification 1)", func(t *testing.T) {
		e := asmEpisode(t, w)
		recordAs(t, w, &e, 62, "other")
		id := addStopBang(t, w, e)
		r := w.clinician.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)) // says male
		refused(t, r, http.StatusBadRequest)
		if r.Message() != "Sex doesn't match the patient's record. Reload the page and try again." {
			t.Errorf("message %q", r.Message())
		}
		recordAs(t, w, &e, 63, "male")
		r = w.clinician.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)) // says 62
		refused(t, r, http.StatusBadRequest)
		if r.Message() != "Age doesn't match the patient's record (63 today). Reload the page and try again." {
			t.Errorf("message %q", r.Message())
		}
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("stored")
		}
	})
	t.Run("Confirm DASI, PRISMA-7 and Apfel (S48): every item, the age and sex where each names them", func(t *testing.T) {
		e := asmEpisode(t, w)
		dasi := addAssessment(t, w, e, "dasi")
		w.clinician.Do("POST", asmPath(w.a, e, "/"+dasi.String()+"/confirm"), dasiAnswers(0, "dasi_selfcare", "dasi_run")).Want(t, http.StatusOK)
		recordAs(t, w, &e, 87, "female") // the age and sex PRISMA-7 and Apfel send below
		prisma := addAssessment(t, w, e, "prisma7")
		five := map[string][2]any{}
		for _, k := range []string{"prisma7_limit", "prisma7_help", "prisma7_home", "prisma7_count_on", "prisma7_aid"} {
			five[k] = [2]any{"o_no", "hq"}
		}
		r := w.clinician.Do("POST", asmPath(w.a, e, "/"+prisma.String()+"/confirm"), answers(0, five))
		refused(t, r, http.StatusBadRequest)
		if r.Message() != "Answer every item before confirming: Age has no answer." {
			t.Errorf("message %q", r.Message())
		}
		five["patientAge"], five["patientSex"] = [2]any{87, "profile"}, [2]any{"female", "profile"}
		w.clinician.Do("POST", asmPath(w.a, e, "/"+prisma.String()+"/confirm"), answers(0, five)).Want(t, http.StatusOK)
		apfel := addAssessment(t, w, e, "apfel")
		three := map[string][2]any{"apfel_smoking": {"o_no", "hq"}, "apfel_ponv": {"o_yes", "hq"}, "patientSex": {"female", "profile"}}
		r = w.clinician.Do("POST", asmPath(w.a, e, "/"+apfel.String()+"/confirm"), answers(0, three))
		refused(t, r, http.StatusBadRequest)
		if r.Message() != "Answer every item before confirming: Opioids after surgery has no answer." {
			t.Errorf("message %q", r.Message())
		}
		three["apfel_opioids"] = [2]any{"o_yes", "clinician"}
		w.clinician.Do("POST", asmPath(w.a, e, "/"+apfel.String()+"/confirm"), answers(0, three)).Want(t, http.StatusOK)
		for _, f := range []uuid.UUID{dasi, prisma, apfel} {
			if row, _ := asmRowOf(t, w, f); row.ValidatedAt == nil {
				t.Errorf("%s not confirmed", f)
			}
		}
		// Apfel names no age: one sent is refused; an observation fills only a number item.
		three["patientAge"] = [2]any{40, "profile"}
		refused(t, w.clinician.Do("PUT", asmPath(w.a, e, "/"+apfel.String()), answers(2, three)), http.StatusBadRequest)
		delete(three, "patientAge")
		three["apfel_ponv"] = [2]any{"o_yes", "observation"}
		refused(t, w.clinician.Do("PUT", asmPath(w.a, e, "/"+apfel.String()), answers(2, three)), http.StatusBadRequest)
	})
	t.Run("Confirm MUST, VTE and the Clinical Frailty Scale (S49): the observations' numbers, the HQ's 0 kg, the age VTE names", func(t *testing.T) {
		e := asmEpisode(t, w) // 62, male
		must := addAssessment(t, w, e, "must")
		four := map[string][2]any{"must_bmi": {19.5, "observation"}, "must_weight": {64, "observation"}, "must_wtloss_kg": {0, "hq"}}
		r := w.clinician.Do("POST", asmPath(w.a, e, "/"+must.String()+"/confirm"), answers(0, four))
		refused(t, r, http.StatusBadRequest)
		if r.Message() != "Answer every item before confirming: Acute disease effect has no answer." {
			t.Errorf("message %q", r.Message())
		}
		four["must_acute"] = [2]any{"o_no", "clinician"}
		w.clinician.Do("POST", asmPath(w.a, e, "/"+must.String()+"/confirm"), answers(0, four)).Want(t, http.StatusOK)
		if row, _ := asmRowOf(t, w, must); row.ValidatedAt == nil || row.Sources["must_weight"] != "observation" || row.Data["must_wtloss_kg"] != float64(0) {
			t.Errorf("MUST row %+v", row)
		}

		vte := addAssessment(t, w, e, "vte")
		var content struct {
			Pages []struct{ Elements []struct{ Name, Type string; Elements []struct{ Name, Type string } } }
		}
		for _, a := range listAssessments(t, w.clinician, w.a, e).Assessments {
			if a.Assessment == "vte" {
				_ = json.Unmarshal(a.Content, &content)
			}
		}
		items := map[string][2]any{"vte_bmi": {31.2, "observation"}}
		for _, el := range content.Pages[0].Elements {
			for _, q := range append([]struct{ Name, Type string }{{el.Name, el.Type}}, el.Elements...) {
				if q.Type == "radiogroup" {
					items[q.Name] = [2]any{"o_no", "clinician"}
				}
			}
		}
		if len(items) != 29 {
			t.Fatalf("%d VTE items, want 29", len(items))
		}
		r = w.clinician.Do("POST", asmPath(w.a, e, "/"+vte.String()+"/confirm"), answers(0, items))
		refused(t, r, http.StatusBadRequest)
		if r.Message() != "Answer every item before confirming: Age has no answer." {
			t.Errorf("message %q", r.Message())
		}
		items["patientAge"] = [2]any{62, "profile"}
		w.clinician.Do("POST", asmPath(w.a, e, "/"+vte.String()+"/confirm"), answers(0, items)).Want(t, http.StatusOK)

		cfs := addAssessment(t, w, e, "rockwood")
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, "/"+cfs.String()+"/confirm"), answers(0, map[string][2]any{})), http.StatusBadRequest)
		refused(t, w.clinician.Do("POST", asmPath(w.a, e, "/"+cfs.String()+"/confirm"),
			answers(0, map[string][2]any{"cfs_level": {"o_cfs_5", "clinician"}, "patientAge": {62, "profile"}})), http.StatusBadRequest) // it names no age
		w.clinician.Do("POST", asmPath(w.a, e, "/"+cfs.String()+"/confirm"), answers(0, map[string][2]any{"cfs_level": {"o_cfs_5", "clinician"}})).Want(t, http.StatusOK)
		for _, f := range []uuid.UUID{vte, cfs} {
			if row, _ := asmRowOf(t, w, f); row.ValidatedAt == nil {
				t.Errorf("%s not confirmed", f)
			}
		}
	})
	t.Run("a stale revision is 409, and nothing changes", func(t *testing.T) {
		e := asmEpisode(t, w)
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
		e := asmEpisode(t, w)
		id := addStopBang(t, w, e)
		refused(t, w.anon.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)), http.StatusUnauthorized)
		refused(t, w.anon.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusUnauthorized)
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
	t.Run("the role: another hospital's clinician: 404", func(t *testing.T) {
		e := asmEpisode(t, w)
		id := addStopBang(t, w, e)
		refused(t, w.site2.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)), http.StatusNotFound)
		refused(t, w.site2.Do("POST", asmPath(w.a, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusNotFound)
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
	t.Run("another trust's: 404, and another episode's assessment is 404", func(t *testing.T) {
		e := asmEpisode(t, w)
		id := addStopBang(t, w, e)
		refused(t, w.otherClin.Do("PUT", asmPath(w.a, e, "/"+id.String()), stopBangAnswers(0)), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", asmPath(w.b, e, "/"+id.String()+"/confirm"), stopBangAnswers(0)), http.StatusNotFound)
		other := asmEpisode(t, w)
		refused(t, w.clinician.Do("PUT", asmPath(w.a, other, "/"+id.String()), stopBangAnswers(0)), http.StatusNotFound)
		refused(t, w.clinician.Do("PUT", asmPath(w.a, e, "/"+e.FormID.String()), stopBangAnswers(0)), http.StatusNotFound) // its HQ form
		if _, ok := asmRowOf(t, w, id); ok {
			t.Error("saved")
		}
	})
	t.Run("invalid input: 400", func(t *testing.T) {
		e := asmEpisode(t, w)
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
		e := asmEpisode(t, w)
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
			e := asmEpisode(t, w)
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
		e := asmEpisode(t, w)
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
	e := asmEpisode(t, w)
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
	e := asmEpisode(t, w)
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
	e := asmEpisode(t, w)
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
