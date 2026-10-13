//go:build integration

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Staff completing the HQ on the patient's behalf (PAT-09), e.g. on the phone: the patient's own
// form at …/episodes/{eid}/patient-form. The answers are the patient's, marked as entered by the
// staff member (episode_answers.updated_by on the patient's row; episodes.patient_submitted_by).

func TestCompleteOnBehalf(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health := w.hq.Chapters[0], w.hq.Chapters[1]
	form := func(rest string) string { return at(w.a, "/episodes/"+w.ep.ID.String()+"/patient-form"+rest) }
	answers := map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 0}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", form(""), nil), http.StatusUnauthorized)
		refused(t, w.anon.Do("PUT", form("/answers/"+about.String()), answers), http.StatusUnauthorized)
		refused(t, w.anon.Do("POST", form("/submit"), nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician at the trust's other hospital gets 404", func(t *testing.T) {
		otherSite := at(w.a2, "/episodes/"+w.ep.ID.String()+"/patient-form")
		refused(t, w.site2.Do("GET", otherSite, nil), http.StatusNotFound)
		refused(t, w.site2.Do("PUT", otherSite+"/answers/"+about.String(), answers), http.StatusNotFound)
		refused(t, w.site2.Do("POST", otherSite+"/submit", nil), http.StatusNotFound)
	})
	t.Run("another trust: 404", func(t *testing.T) {
		refused(t, w.otherClin.Do("GET", form(""), nil), http.StatusNotFound)
		refused(t, w.otherClin.Do("POST", at(w.b, "/episodes/"+w.ep.ID.String()+"/patient-form/submit"), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/abc/patient-form"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/"+uuid.NewString()+"/patient-form"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("PUT", form("/answers/"+w.hq.Chapters[2].String()), answers), http.StatusNotFound) // the clinician's set
		refused(t, w.clinician.Do("PUT", form("/answers/"+about.String()), `{"data": {}}`), http.StatusBadRequest)
	})
	if n := count(t, w.db, `SELECT count(*) FROM episode_answers`); n != 0 {
		t.Fatalf("refusals saved %d rows", n)
	}

	t.Run("happy path: the patient's form, their answers marked as entered by the clinician", func(t *testing.T) {
		r := w.clinician.Do("GET", form(""), nil)
		r.Want(t, http.StatusOK)
		var got struct {
			Patient  struct{ FirstName string }
			Chapters []struct{ ID uuid.UUID }
		}
		r.Decode(t, &got)
		if got.Patient.FirstName != "Test" || len(got.Chapters) != 2 || strings.Contains(string(r.Body), "q_secret") {
			t.Errorf("form %s", r.Body)
		}
		savedRevision(t, w.clinician.Do("PUT", form("/answers/"+about.String()), answers))
		var by *uuid.UUID
		if err := w.db.QueryRow(t.Context(), `SELECT updated_by FROM episode_answers WHERE episode_id = $1 AND chapter_id = $2
			AND actor = 'patient'`, w.ep.ID, about).Scan(&by); err != nil {
			t.Fatal(err)
		}
		if by == nil || *by != w.cara.ID {
			t.Errorf("entered by %v, want Cara", by)
		}
	})
	t.Run("the patient adding to it later keeps the mark", func(t *testing.T) {
		p := w.anon.AsPatient(t, w.ep.Link, w.ep.DateOfBirth)
		savedRevision(t, p.Do("PUT", patientPath(w.ep, "/answers/"+about.String()), map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 1}))
		savedRevision(t, p.Do("PUT", patientPath(w.ep, "/answers/"+health.String()), map[string]any{"data": map[string]any{"q_diabetes": "o_no"}, "revision": 0}))
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_id = $1 AND chapter_id = $2 AND updated_by = $3`,
			w.ep.ID, about, w.cara.ID); n != 1 {
			t.Error("the patient's save cleared the mark")
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_id = $1 AND chapter_id = $2 AND updated_by IS NULL`,
			w.ep.ID, health); n != 1 {
			t.Error("the patient's own set is marked")
		}
	})
	t.Run("the clinician's HQ view names who entered them", func(t *testing.T) {
		r := w.clinician.Do("GET", at(w.a, "/episodes/"+w.ep.ID.String()+"/hq"), nil)
		r.Want(t, http.StatusOK)
		var got struct {
			Answers []struct {
				ChapterID     uuid.UUID
				Actor         string
				UpdatedByName *string
			}
		}
		r.Decode(t, &got)
		names := map[uuid.UUID]string{}
		for _, a := range got.Answers {
			if a.Actor == "patient" && a.UpdatedByName != nil {
				names[a.ChapterID] = *a.UpdatedByName
			}
		}
		if names[about] != w.cara.Name || names[health] != "" {
			t.Errorf("entered by %v", names)
		}
	})
	t.Run("sending it: Ready for review, sent by the clinician, and the notes say so", func(t *testing.T) {
		w.clinician.Do("POST", form("/submit"), nil).Want(t, http.StatusNoContent)
		if e := readEpisode(t, w, w.ep.ID); e.Status != "ready_for_review" || e.Submitted == nil {
			t.Errorf("episode %+v", e)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episodes WHERE id = $1 AND patient_submitted_by = $2`, w.ep.ID, w.cara.ID); n != 1 {
			t.Error("not marked as sent by Cara")
		}
		ev := readEvents(t, w, w.ep.ID)
		if last := ev[len(ev)-1]; last.Text != "HQ completed on the patient's behalf" || last.User == nil || *last.User != w.cara.ID {
			t.Errorf("events %+v", ev)
		}
		r := w.clinician.Do("GET", at(w.a, "/episodes/"+w.ep.ID.String()), nil)
		if !strings.Contains(string(r.Body), `"patientSubmittedByName":"`+w.cara.Name+`"`) {
			t.Errorf("episode %s", r.Body)
		}
	})
	t.Run("frozen once sent, from either side", func(t *testing.T) {
		refused(t, w.clinician.Do("PUT", form("/answers/"+about.String()), map[string]any{"data": map[string]any{}, "revision": 2}), http.StatusConflict)
		refused(t, w.clinician.Do("POST", form("/submit"), nil), http.StatusConflict)
	})
}
