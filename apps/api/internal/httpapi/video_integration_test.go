//go:build integration

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// The Video element's tick (S34, INF-02): the patient's "I have watched this video" is saved and
// sent like any answer, and an HQ whose required tick isn't ticked can't be sent. The browser checks
// every required question (plan 2.1); the server holds back only the ticks every patient sees (Go
// never evaluates conditions), so a tick behind a condition is the browser's to check.
// The routes are the patient's save and submit, and staff's on the patient's behalf: their sign-in,
// role and tenancy cases are in TestSavePatientAnswers, TestSubmitPatientHQ and TestCompleteOnBehalf.
func TestVideoTick(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	hq := w.fx.PublishedHQ(t, w.a, w.alex, "video.json")
	videos := hq.Chapters[0]
	ep := w.fx.Episode(t, hq, w.cara)
	p := w.anon.AsPatient(t, ep.Link, ep.DateOfBirth)
	save := func(data map[string]any, revision int) {
		t.Helper()
		savedRevision(t, p.Do("PUT", patientPath(ep, "/answers/"+videos.String()), map[string]any{"data": data, "revision": revision}))
	}
	submitted := func() bool {
		return count(t, w.db, `SELECT count(*) FROM episode_forms WHERE id = $1 AND status = 'submitted'`, ep.FormID) == 1
	}

	t.Run("the patient's JSON keeps the videos", func(t *testing.T) {
		r := p.Do("GET", patientPath(ep, ""), nil)
		r.Want(t, http.StatusOK)
		if !strings.Contains(string(r.Body), `"type":"video"`) || !strings.Contains(string(r.Body), "aqz-KE-bpKQ") {
			t.Errorf("form %s", r.Body)
		}
	})

	t.Run("invalid input: a required tick not ticked (no answers yet) can't be sent", func(t *testing.T) {
		r := p.Do("POST", patientPath(ep, "/submit"), nil)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "“Your anaesthetic”") || strings.Contains(r.Message(), "Fasting") {
			t.Errorf("message %q, want it to name only the required video", r.Message())
		}
		if submitted() || readEpisode(t, w, ep.ID).Status != "hq_not_complete" {
			t.Error("the HQ was sent")
		}
	})

	t.Run("invalid input: unticked again (false), it still can't be sent", func(t *testing.T) {
		save(map[string]any{"q_anaes": false, "q_smokes": "o_yes"}, 0)
		refused(t, p.Do("POST", patientPath(ep, "/submit"), nil), http.StatusConflict)
		if submitted() {
			t.Error("the HQ was sent")
		}
	})

	t.Run("wrong role: staff filling in on the patient's behalf can't send it unticked either", func(t *testing.T) {
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/"+ep.ID.String()+"/patient-form/submit"), nil), http.StatusConflict)
		if submitted() {
			t.Error("the HQ was sent")
		}
	})

	t.Run("happy path: ticked, it's saved and sent; the optional tick and the one behind a condition are the browser's", func(t *testing.T) {
		save(map[string]any{"q_anaes": true, "q_smokes": "o_yes"}, 1)
		if got := patientAnswers(t, w, ep, videos); !strings.Contains(got, `"q_anaes": true`) {
			t.Errorf("answers %s", got)
		}
		p.Do("POST", patientPath(ep, "/submit"), nil).Want(t, http.StatusNoContent)
		if !submitted() || readEpisode(t, w, ep.ID).Status != "ready_for_review" {
			t.Error("the HQ wasn't sent")
		}
		if got := patientAnswers(t, w, ep, videos); !strings.Contains(got, `"q_anaes": true`) {
			t.Errorf("answers after sending %s", got)
		}
	})

	t.Run("a second HQ for the same patient pre-fills their other answers but never the tick (S34 answers 2)", func(t *testing.T) {
		next := w.fx.EpisodeOf(t, hq, w.a, ep.PatientID, w.cara, "")
		r := w.anon.AsPatient(t, next.Link, next.DateOfBirth).Do("GET", patientPath(next, ""), nil)
		r.Want(t, http.StatusOK)
		var got struct{ Answers map[string]map[string]any }
		r.Decode(t, &got)
		set := got.Answers[videos.String()]
		if set["q_smokes"] != "o_yes" {
			t.Errorf("answers %v, want q_smokes pre-filled (so pre-fill ran)", got.Answers)
		}
		if _, ok := set["q_anaes"]; ok {
			t.Errorf("the tick was pre-filled: %v", set)
		}
	})

	t.Run("another HQ without videos is sent as before", func(t *testing.T) {
		other := w.fx.Episode(t, w.hq, w.cara)
		w.anon.AsPatient(t, other.Link, other.DateOfBirth).Do("POST", patientPath(other, "/submit"), nil).Want(t, http.StatusNoContent)
	})
}
