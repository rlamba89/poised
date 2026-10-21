//go:build integration

package httpapi_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// The status routes (S20): PATCH …/status, POST …/cancel, …/reopen and …/undo. Each gets the five
// cases; on top, every illegal transition → 409, a role without the right → 403, a missing reason
// → 400 and a stale revision → 409. Every hospital member is a clinician at least, so for the
// clinician's moves the "role" case is a clinician at the trust's other hospital (404), as for the
// other episode routes; re-open needs a super clinician, so a clinician gets 403.

// revisionOf is the episode's revision now, as the page would have loaded it.
func revisionOf(t *testing.T, w *clinic, id uuid.UUID) int32 {
	t.Helper()
	var rev int32
	if err := w.db.QueryRow(t.Context(), `SELECT revision FROM episodes WHERE id = $1`, id).Scan(&rev); err != nil {
		t.Fatal(err)
	}
	return rev
}

// statusRow is the part of an episode the status routes change.
type statusRow struct {
	Status, Admitted, Discharged, CancelReason string // dates YYYY-MM-DD or ""
	ReviewCompleted                            bool
	Revision                                   int32
}

func readStatus(t *testing.T, w *clinic, id uuid.UUID) statusRow {
	t.Helper()
	var s statusRow
	err := w.db.QueryRow(t.Context(), `SELECT status, coalesce(admitted_at::text, ''), coalesce(discharged_at::text, ''),
		coalesce(cancel_reason, ''), review_completed_at IS NOT NULL, revision FROM episodes WHERE id = $1`, id).Scan(
		&s.Status, &s.Admitted, &s.Discharged, &s.CancelReason, &s.ReviewCompleted, &s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// historyRow is one row of the status history.
type historyRow struct {
	Move, From, To, Reason string
	User                   *uuid.UUID
	Undone                 bool
}

func readHistory(t *testing.T, w *clinic, id uuid.UUID) []historyRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT move, coalesce(from_status, ''), to_status, reason, user_id, undone_at IS NOT NULL
		FROM episode_status_events WHERE episode_id = $1 ORDER BY seq`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []historyRow
	for rows.Next() {
		var h historyRow
		if err := rows.Scan(&h.Move, &h.From, &h.To, &h.Reason, &h.User, &h.Undone); err != nil {
			t.Fatal(err)
		}
		out = append(out, h)
	}
	return out
}

func moves(h []historyRow) []string {
	var out []string
	for _, r := range h {
		out = append(out, r.Move+":"+r.From+">"+r.To)
	}
	return out
}

func lastNote(t *testing.T, w *clinic, id uuid.UUID) string {
	t.Helper()
	ev := readEvents(t, w, id)
	if len(ev) == 0 {
		return ""
	}
	return ev[len(ev)-1].Text
}

// yesterday is a date that's in the past at any hospital.
func yesterday() string { return time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly) }

func TestChangeStatus(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.ReadyForPOA)
	patch := func(c *apitest.Client, h apitest.Hospital, body map[string]any) *apitest.Resp {
		return c.Do("PATCH", epPath(h, e, "/status"), body)
	}
	body := func(to string) map[string]any {
		return map[string]any{"to": to, "revision": revisionOf(t, w, e.ID)}
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, patch(w.anon, w.a, body(episode.OnHold)), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, patch(w.site2, w.a, body(episode.OnHold)), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, patch(w.otherClin, w.a, body(episode.OnHold)), http.StatusNotFound)
		refused(t, patch(w.otherClin, w.b, body(episode.OnHold)), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		rev := revisionOf(t, w, e.ID)
		for _, bad := range []any{
			map[string]any{"to": episode.OnHold},                 // no revision
			map[string]any{"to": episode.OnHold, "revision": -1}, // not a revision
			map[string]any{"to": "", "revision": rev},
			map[string]any{"to": "bogus", "revision": rev}, // not a status (test report 1: was 409)
			map[string]any{"to": episode.OnHold, "reason": strings.Repeat("x", 1001), "revision": rev},
			map[string]any{"to": episode.Admitted, "revision": rev},                       // no date
			map[string]any{"to": episode.Admitted, "date": "21/10/2026", "revision": rev}, // not a date
			map[string]any{"to": episode.Discharged, "date": "2026-02-30", "revision": rev},
			`{"to":`,
		} {
			refused(t, w.clinician.Do("PATCH", epPath(w.a, e, "/status"), bad), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("PATCH", at(w.a, "/episodes/abc/status"), body(episode.OnHold)), http.StatusNotFound)
	})
	if s := readStatus(t, w, e.ID); s.Status != episode.ReadyForPOA || s.Revision != 0 || len(readHistory(t, w, e.ID)) != 0 {
		t.Fatalf("refusals changed the episode: %+v", s)
	}

	t.Run("happy path: a pick, with an optional reason, recorded in the history and General notes", func(t *testing.T) {
		patch(w.clinician, w.a, body(episode.OnHold)).Want(t, http.StatusNoContent)
		b := body(episode.NotReady)
		b["reason"] = "  Bloods abnormal "
		patch(w.clinician, w.a, b).Want(t, http.StatusNoContent)
		if s := readStatus(t, w, e.ID); s.Status != episode.NotReady || s.Revision != 2 {
			t.Errorf("episode %+v", s)
		}
		h := readHistory(t, w, e.ID)
		if !slices.Equal(moves(h), []string{"set:ready_for_poa>on_hold", "set:on_hold>not_ready"}) || h[0].Reason != "" ||
			h[1].Reason != "Bloods abnormal" || h[1].User == nil || *h[1].User != w.cara.ID {
			t.Errorf("history %+v", h)
		}
		if n := lastNote(t, w, e.ID); n != "Status changed to Not ready for admission: Bloods abnormal" {
			t.Errorf("note %q", n)
		}
	})
	t.Run("happy path: admitted and discharged, each with its date", func(t *testing.T) {
		patch(w.clinician, w.a, body(episode.ReadyForAdmission)).Want(t, http.StatusNoContent)
		b := body(episode.Admitted)
		b["date"] = "2026-01-10"
		patch(w.clinician, w.a, b).Want(t, http.StatusNoContent)
		b = body(episode.Discharged)
		b["date"] = "2026-01-09"
		r := patch(w.clinician, w.a, b)
		refused(t, r, http.StatusBadRequest)
		if !strings.Contains(r.Message(), "before the admission date") {
			t.Errorf("message %q", r.Message())
		}
		b["date"] = time.Now().UTC().AddDate(0, 0, 2).Format(time.DateOnly)
		refused(t, patch(w.clinician, w.a, b), http.StatusBadRequest) // in the future
		b["date"] = "2026-01-12"
		patch(w.clinician, w.a, b).Want(t, http.StatusNoContent)
		if s := readStatus(t, w, e.ID); s.Status != episode.Discharged || s.Admitted != "2026-01-10" || s.Discharged != "2026-01-12" {
			t.Errorf("episode %+v", s)
		}
		if n := lastNote(t, w, e.ID); n != "Status changed to Discharged (discharge date 2026-01-12)" {
			t.Errorf("note %q", n)
		}
	})
	t.Run("an admission date in the future is refused", func(t *testing.T) {
		f := w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, f, episode.ReadyForAdmission)
		r := w.clinician.Do("PATCH", epPath(w.a, f, "/status"), map[string]any{"to": episode.Admitted, "revision": 0,
			"date": time.Now().UTC().AddDate(0, 0, 2).Format(time.DateOnly)})
		refused(t, r, http.StatusBadRequest)
		if s := readStatus(t, w, f.ID); s.Status != episode.ReadyForAdmission || s.Admitted != "" {
			t.Errorf("episode %+v", s)
		}
	})
	t.Run("a stale revision: 409, and nothing changes", func(t *testing.T) {
		f := w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, f, episode.ReadyForPOA)
		w.clinician.Do("PATCH", epPath(w.a, f, "/status"), map[string]any{"to": episode.OnHold, "revision": 0}).Want(t, http.StatusNoContent)
		r := w.clinician.Do("PATCH", epPath(w.a, f, "/status"), map[string]any{"to": episode.POAComplete, "revision": 0}) // the other tab
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "changed since you opened it") || !strings.Contains(string(r.Body), `"code":"stale"`) {
			t.Errorf("response %s", r)
		}
		if s := readStatus(t, w, f.ID); s.Status != episode.OnHold || len(readHistory(t, w, f.ID)) != 1 {
			t.Errorf("episode %+v", s)
		}
	})
}

// Every pair of statuses through PATCH …/status: the clinician's picks, Admit and Discharge are
// applied; every other move (cancel and re-open have their own routes) is a 409 that changes nothing.
func TestChangeStatusEveryPair(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	viaPatch := map[episode.Move]bool{episode.Set: true, episode.Admit: true, episode.Discharge: true}
	for _, from := range episode.Statuses {
		e := w.fx.Episode(t, w.hq, w.cara)
		for _, to := range append(slices.Clone(episode.Statuses), "nonsense") {
			setStatus(t, w, e, from)
			if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET admitted_at = NULL, discharged_at = NULL WHERE id = $1`, e.ID); err != nil {
				t.Fatal(err)
			}
			move := episode.Set
			switch to {
			case episode.Admitted:
				move = episode.Admit
			case episode.Discharged:
				move = episode.Discharge
			}
			_, err := episode.Check(from, to, move, role.Clinician)
			want := http.StatusConflict
			switch {
			case err == nil && viaPatch[move]:
				want = http.StatusNoContent
			case to == "nonsense":
				want = http.StatusBadRequest // not a status at all
			}
			r := w.clinician.Do("PATCH", epPath(w.a, e, "/status"),
				map[string]any{"to": to, "date": yesterday(), "revision": revisionOf(t, w, e.ID)})
			if r.Code != want {
				t.Errorf("%s → %s: %s, want %d", from, to, r, want)
				continue
			}
			got := readStatus(t, w, e.ID).Status
			if want == http.StatusNoContent && got != to || want != http.StatusNoContent && got != from {
				t.Errorf("%s → %s: status %s", from, to, got)
			}
		}
	}
}

func TestCancelEpisode(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	cancel := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body map[string]any) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/cancel"), body)
	}
	withdrew := func(e apitest.Episode) map[string]any {
		return map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, e.ID)}
	}
	e := w.fx.Episode(t, w.hq, w.cara)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, cancel(w.anon, w.a, e, withdrew(e)), http.StatusUnauthorized)
	})
	t.Run("role: a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, cancel(w.site2, w.a, e, withdrew(e)), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, cancel(w.otherClin, w.a, e, withdrew(e)), http.StatusNotFound)
		refused(t, cancel(w.otherClin, w.b, e, withdrew(e)), http.StatusNotFound)
	})
	t.Run("invalid input: a reason from the list is required, and Other needs a sentence", func(t *testing.T) {
		for _, bad := range []map[string]any{
			{"revision": 0},
			{"reason": "", "revision": 0},
			{"reason": "bored", "revision": 0},
			{"reason": "other", "revision": 0},
			{"reason": "other", "details": "   ", "revision": 0},
			{"reason": "other", "details": strings.Repeat("x", 1001), "revision": 0},
			{"reason": "patient_withdrew"}, // no revision
		} {
			refused(t, cancel(w.clinician, w.a, e, bad), http.StatusBadRequest)
		}
	})
	if s := readStatus(t, w, e.ID); s.Status != episode.HQNotComplete || s.CancelReason != "" {
		t.Fatalf("refusals changed the episode: %+v", s)
	}
	t.Run("every reason, from every status before discharged", func(t *testing.T) {
		reasons := []struct{ key, details, recorded string }{
			{"procedure_cancelled", "", "Procedure cancelled"},
			{"patient_withdrew", "", "Patient withdrew"},
			{"moved_hospital", "Leeds General", "Moved to another hospital: Leeds General"},
			{"other", " Emigrated ", "Other: Emigrated"},
		}
		for i, from := range []string{episode.HQNotComplete, episode.ReadyForReview, episode.ReadyForPOA, episode.POAComplete,
			episode.OnHold, episode.NotReady, episode.ReadyForAdmission, episode.Admitted} {
			f := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, f, from)
			r := reasons[i%len(reasons)]
			cancel(w.clinician, w.a, f, map[string]any{"reason": r.key, "details": r.details, "revision": 0}).Want(t, http.StatusNoContent)
			s := readStatus(t, w, f.ID)
			h := readHistory(t, w, f.ID)
			if s.Status != episode.Cancelled || s.CancelReason != r.key || s.Revision != 1 || len(h) != 1 ||
				h[0].Move != "cancel" || h[0].From != from || h[0].Reason != r.recorded {
				t.Errorf("from %s: %+v, history %+v", from, s, h)
			}
			if n := lastNote(t, w, f.ID); n != "Episode cancelled: "+r.recorded {
				t.Errorf("note %q", n)
			}
		}
	})
	t.Run("final: a cancelled or discharged episode can't be cancelled, or moved anywhere", func(t *testing.T) {
		for _, from := range []string{episode.Discharged, episode.Cancelled} {
			f := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, f, from)
			refused(t, cancel(w.clinician, w.a, f, withdrew(f)), http.StatusConflict)
		}
		f := w.fx.Episode(t, w.hq, w.cara)
		cancel(w.clinician, w.a, f, withdrew(f)).Want(t, http.StatusNoContent)
		rev := revisionOf(t, w, f.ID)
		for _, to := range []string{episode.HQNotComplete, episode.ReadyForPOA, episode.OnHold} {
			refused(t, w.clinician.Do("PATCH", epPath(w.a, f, "/status"), map[string]any{"to": to, "revision": rev}), http.StatusConflict)
		}
		refused(t, w.author.Do("POST", epPath(w.a, f, "/reopen"), map[string]any{"reason": "x", "revision": rev}), http.StatusConflict)
		refused(t, w.clinician.Do("POST", epPath(w.a, f, "/undo"), map[string]any{"reason": "x", "revision": rev}), http.StatusConflict)
		refused(t, w.clinician.Do("PATCH", epPath(w.a, f, ""), map[string]any{"nurseAsa": 2, "revision": rev}), http.StatusConflict)
		refused(t, w.clinician.Do("POST", epPath(w.a, f, "/resend-invite"), nil), http.StatusConflict)
		if s := readStatus(t, w, f.ID); s.Status != episode.Cancelled || s.Revision != rev {
			t.Errorf("episode %+v", s)
		}
		// A note can still be added: why it was cancelled, a call with the patient.
		w.clinician.Do("POST", epPath(w.a, f, "/notes"), map[string]string{"text": "Rang the patient"}).Want(t, http.StatusCreated)
	})
	t.Run("a stale revision: 409", func(t *testing.T) {
		f := w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, f, episode.ReadyForPOA)
		w.clinician.Do("PATCH", epPath(w.a, f, "/status"), map[string]any{"to": episode.OnHold, "revision": 0}).Want(t, http.StatusNoContent)
		refused(t, cancel(w.clinician, w.a, f, map[string]any{"reason": "patient_withdrew", "revision": 0}), http.StatusConflict)
		if s := readStatus(t, w, f.ID); s.Status != episode.OnHold {
			t.Errorf("episode %+v", s)
		}
	})
}

// The patient's side of a cancel (S20 answers 5): the invite link and the sessions opened from it
// end, the follow-up stops, and the home shows "cancelled" with no form.
func TestCancelEndsThePatientsSide(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	other := w.fx.EpisodeOf(t, w.hq, w.a, e.PatientID, w.cara, "") // the same patient's other episode
	viaLink := w.anon.AsPatient(t, e.Link, e.DateOfBirth)          // a session opened from the cancelled episode's link
	viaOther := w.anon.AsPatient(t, other.Link, e.DateOfBirth)     // and one from the other episode's
	viaLink.Do("GET", patientPath(e, ""), nil).Want(t, http.StatusOK)
	if _, err := w.db.Exec(t.Context(), `INSERT INTO jobs (kind, episode_id, run_at) VALUES ('followup', $1, now())`, e.ID); err != nil {
		t.Fatal(err)
	}

	w.clinician.Do("POST", epPath(w.a, e, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": 0}).Want(t, http.StatusNoContent)

	t.Run("the link stops working", func(t *testing.T) {
		refused(t, w.anon.NewBrowser(t).Do("POST", linkPath(e, "/continue"), nil), http.StatusNotFound)
		if n := count(t, w.db, `SELECT count(*) FROM login_links WHERE episode_id = $1 AND revoked_at IS NULL`, e.ID); n != 0 {
			t.Errorf("%d live links", n)
		}
		// The other episode's link is untouched.
		w.anon.NewBrowser(t).Do("POST", linkPath(other, "/continue"), nil).Want(t, http.StatusOK)
	})
	t.Run("the session opened from it ends", func(t *testing.T) {
		refused(t, viaLink.Do("GET", "/api/p/home", nil), http.StatusUnauthorized)
	})
	t.Run("the follow-up stops", func(t *testing.T) {
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1 AND done_at IS NULL`, e.ID); n != 0 {
			t.Errorf("%d open jobs", n)
		}
	})
	t.Run("the home shows it cancelled, and its form isn't served or saved", func(t *testing.T) {
		states := map[uuid.UUID]string{}
		for _, h := range readHome(t, viaOther).Episodes {
			states[h.ID] = h.State
		}
		if states[e.ID] != "cancelled" || states[other.ID] != "to_do" {
			t.Errorf("states %v", states)
		}
		r := viaOther.Do("GET", patientPath(e, ""), nil)
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(r.Message(), "cancelled by the hospital") || !strings.Contains(string(r.Body), `"code":"cancelled"`) {
			t.Errorf("response %s", r)
		}
		refused(t, viaOther.Do("PUT", patientPath(e, "/answers/"+w.hq.Chapters[0].String()), map[string]any{"data": map[string]any{}, "revision": 0}), http.StatusForbidden)
		refused(t, viaOther.Do("POST", patientPath(e, "/submit"), nil), http.StatusForbidden)
		refused(t, w.clinician.Do("GET", epPath(w.a, e, "/patient-form"), nil), http.StatusForbidden)
		refused(t, w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil), http.StatusForbidden)
		if s := readStatus(t, w, e.ID); s.Status != episode.Cancelled {
			t.Errorf("episode %+v", s)
		}
	})
}

func TestReopenReview(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	sue := w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician")) // a super clinician of hospital A only
	admin := w.anon.LoginAs(t, w.fx.TrustUser(t, w.a.Org, "admin"))
	reopen := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body map[string]any) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/reopen"), body)
	}
	why := func(e apitest.Episode) map[string]any {
		return map[string]any{"reason": "Missed the cardiac history", "revision": revisionOf(t, w, e.ID)}
	}
	// A reviewed episode: the patient's answers sent, the clinician's validated, the review complete.
	reviewed := func() apitest.Episode {
		e := w.fx.Episode(t, w.hq, w.cara)
		saveAnswers(t, w, e, w.hq.Chapters[0], nil, `{"q_smokes": "o_yes"}`)
		setStatus(t, w, e, episode.ReadyForReview)
		saveAnswers(t, w, e, w.hq.Chapters[0], &w.cara, `{"q_smokes": "o_no"}`)
		if _, err := w.db.Exec(t.Context(), `UPDATE episode_answers SET validated_at = now() WHERE episode_form_id = $1 AND actor = 'clinician'`, e.FormID); err != nil {
			t.Fatal(err)
		}
		w.clinician.Do("POST", epPath(w.a, e, "/complete-review"), nil).Want(t, http.StatusNoContent)
		w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": episode.OnHold, "revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
		return e
	}
	e := reviewed()

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, reopen(w.anon, w.a, e, why(e)), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician can't re-open (403)", func(t *testing.T) {
		r := reopen(w.clinician, w.a, e, why(e))
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(r.Message(), "super clinician") {
			t.Errorf("message %q", r.Message())
		}
		refused(t, reopen(w.site2, w.a, e, why(e)), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, reopen(w.other, w.a, e, why(e)), http.StatusNotFound) // Bea, a super clinician of trust B
		refused(t, reopen(w.other, w.b, e, why(e)), http.StatusNotFound)
	})
	t.Run("invalid input: a reason is required", func(t *testing.T) {
		rev := revisionOf(t, w, e.ID)
		for _, bad := range []any{map[string]any{"revision": rev}, map[string]any{"reason": "  ", "revision": rev},
			map[string]any{"reason": "x"}, `{"reason":`} {
			refused(t, w.author.Do("POST", epPath(w.a, e, "/reopen"), bad), http.StatusBadRequest)
		}
	})
	t.Run("only after the review is complete and before admission", func(t *testing.T) {
		for _, from := range []string{episode.HQNotComplete, episode.ReadyForReview, episode.Admitted, episode.Discharged, episode.Cancelled} {
			f := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, f, from)
			refused(t, reopen(w.author, w.a, f, why(f)), http.StatusConflict)
		}
	})
	if s := readStatus(t, w, e.ID); s.Status != episode.OnHold || !s.ReviewCompleted {
		t.Fatalf("refusals changed the episode: %+v", s)
	}
	t.Run("happy path: back to Ready for review, the stamp cleared, the answers and validations kept and unlocked", func(t *testing.T) {
		reopen(sue, w.a, e, why(e)).Want(t, http.StatusNoContent)
		if s := readStatus(t, w, e.ID); s.Status != episode.ReadyForReview || s.ReviewCompleted {
			t.Errorf("episode %+v", s)
		}
		var data string
		var validated bool
		var rev int32
		if err := w.db.QueryRow(t.Context(), `SELECT data::text, validated_at IS NOT NULL, revision FROM episode_answers
			WHERE episode_form_id = $1 AND actor = 'clinician'`, e.FormID).Scan(&data, &validated, &rev); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(data, "o_no") || !validated {
			t.Errorf("clinician's answers %s, validated %v", data, validated)
		}
		if got := readEpisode(t, w, e.ID); got.Submitted == nil {
			t.Error("the patient's form isn't sent any more: re-opening mustn't ask them again")
		}
		w.clinician.Do("PUT", epPath(w.a, e, "/answers/"+w.hq.Chapters[0].String()),
			map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "validated": true, "revision": rev}).Want(t, http.StatusOK)
		h := readHistory(t, w, e.ID)
		last := h[len(h)-1]
		if last.Move != "reopen" || last.From != episode.OnHold || last.To != episode.ReadyForReview || last.Reason != "Missed the cardiac history" {
			t.Errorf("history %+v", h)
		}
		if n := lastNote(t, w, e.ID); n != "HQ review re-opened: Missed the cardiac history" {
			t.Errorf("note %q", n)
		}
		// The review can be completed again.
		w.clinician.Do("POST", epPath(w.a, e, "/complete-review"), nil).Want(t, http.StatusNoContent)
	})
	t.Run("an admin may re-open too", func(t *testing.T) {
		f := reviewed()
		reopen(admin, w.a, f, why(f)).Want(t, http.StatusNoContent)
	})
	t.Run("a stale revision: 409", func(t *testing.T) {
		f := reviewed()
		rev := revisionOf(t, w, f.ID)
		w.clinician.Do("PATCH", epPath(w.a, f, "/status"), map[string]any{"to": episode.NotReady, "revision": rev}).Want(t, http.StatusNoContent)
		refused(t, reopen(w.author, w.a, f, map[string]any{"reason": "x", "revision": rev}), http.StatusConflict)
	})
}

// Undo (S20 answers 2): any clinician steps back the latest move made by hand, with a reason, and
// again to go further back; the dates of an admission or a discharge go with it. The workflow's
// moves, re-open and cancel stop it.
func TestUndoStatus(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	undo := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body map[string]any) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/undo"), body)
	}
	why := func(e apitest.Episode) map[string]any {
		return map[string]any{"reason": "Pressed by mistake", "revision": revisionOf(t, w, e.ID)}
	}
	set := func(e apitest.Episode, to, date string) {
		t.Helper()
		w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": to, "date": date, "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
	}
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.ReadyForReview)
	w.clinician.Do("POST", epPath(w.a, e, "/complete-review"), nil).Want(t, http.StatusNoContent)
	set(e, episode.OnHold, "")
	set(e, episode.ReadyForAdmission, "")
	set(e, episode.Admitted, "2026-01-10")
	set(e, episode.Discharged, "2026-01-12")

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, undo(w.anon, w.a, e, why(e)), http.StatusUnauthorized)
	})
	t.Run("role: a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, undo(w.site2, w.a, e, why(e)), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, undo(w.otherClin, w.a, e, why(e)), http.StatusNotFound)
		refused(t, undo(w.otherClin, w.b, e, why(e)), http.StatusNotFound)
	})
	t.Run("invalid input: a reason is required", func(t *testing.T) {
		rev := revisionOf(t, w, e.ID)
		for _, bad := range []map[string]any{{"revision": rev}, {"reason": " ", "revision": rev}, {"reason": "x"}} {
			refused(t, undo(w.clinician, w.a, e, bad), http.StatusBadRequest)
		}
	})
	t.Run("a stale revision: 409", func(t *testing.T) {
		refused(t, undo(w.clinician, w.a, e, map[string]any{"reason": "x", "revision": revisionOf(t, w, e.ID) - 1}), http.StatusConflict)
	})
	if s := readStatus(t, w, e.ID); s.Status != episode.Discharged {
		t.Fatalf("refusals changed the episode: %+v", s)
	}
	t.Run("happy path: one step at a time, by any clinician, back to the completed review", func(t *testing.T) {
		for _, step := range []struct{ want, admitted, discharged string }{
			{episode.Admitted, "2026-01-10", ""},
			{episode.ReadyForAdmission, "", ""},
			{episode.OnHold, "", ""},
			{episode.ReadyForPOA, "", ""},
		} {
			undo(w.clinician, w.a, e, why(e)).Want(t, http.StatusNoContent)
			if s := readStatus(t, w, e.ID); s.Status != step.want || s.Admitted != step.admitted || s.Discharged != step.discharged {
				t.Errorf("after undo: %+v, want %+v", s, step)
			}
		}
		if n := lastNote(t, w, e.ID); n != "Status change undone, back to Ready for POA: Pressed by mistake" {
			t.Errorf("note %q", n)
		}
		// The completed review is next: undo stops there (re-open is for that).
		r := undo(w.clinician, w.a, e, why(e))
		refused(t, r, http.StatusConflict)
		h := readHistory(t, w, e.ID)
		var undone, undos int
		for _, row := range h {
			if row.Undone {
				undone++
			}
			if row.Move == "undo" {
				undos++
				if row.Reason != "Pressed by mistake" || row.User == nil || *row.User != w.cara.ID {
					t.Errorf("undo row %+v", row)
				}
			}
		}
		if undone != 4 || undos != 4 {
			t.Errorf("history %+v", h)
		}
	})
	t.Run("after an undo, a new move is the one undone next", func(t *testing.T) {
		set(e, episode.NotReady, "")
		undo(w.clinician, w.a, e, why(e)).Want(t, http.StatusNoContent)
		if s := readStatus(t, w, e.ID); s.Status != episode.ReadyForPOA {
			t.Errorf("episode %+v", s)
		}
	})
	t.Run("nothing to undo: a new episode, or a cancelled one", func(t *testing.T) {
		f := w.fx.Episode(t, w.hq, w.cara)
		refused(t, undo(w.clinician, w.a, f, why(f)), http.StatusConflict)
		w.clinician.Do("POST", epPath(w.a, f, "/cancel"), map[string]any{"reason": "procedure_cancelled", "revision": 0}).Want(t, http.StatusNoContent)
		refused(t, undo(w.clinician, w.a, f, why(f)), http.StatusConflict)
	})
}

// What the episode page gets (S20): the moves this user may make from here, the undo, the status
// history; and the workflow's own moves (create, submit, complete review) are recorded too.
func TestEpisodeStatusView(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	id := newPatientEpisode(t, w, "History", "", "")
	e := apitest.Episode{ID: id}
	type view struct {
		Episode struct {
			Status   string
			Revision int32
		}
		Moves []struct {
			To, Label, Move        string
			NeedsReason, NeedsDate bool
		}
		Undo          *struct{ To, Label string }
		StatusHistory []struct {
			Move       string
			FromStatus *string
			ToStatus   string
			Reason     string
			UserName   *string
			UndoneAt   *time.Time
		}
	}
	read := func(c *apitest.Client) view {
		t.Helper()
		r := c.Do("GET", epPath(w.a, e, ""), nil)
		r.Want(t, http.StatusOK)
		var v view
		r.Decode(t, &v)
		return v
	}
	targets := func(v view) []string {
		var out []string
		for _, m := range v.Moves {
			out = append(out, m.Move+":"+m.To)
		}
		return out
	}

	v := read(w.clinician)
	if !slices.Equal(targets(v), []string{"cancel:cancelled"}) || v.Undo != nil || len(v.StatusHistory) != 1 ||
		v.StatusHistory[0].Move != "create" || v.StatusHistory[0].FromStatus != nil || v.StatusHistory[0].UserName == nil {
		t.Fatalf("new episode: %+v", v)
	}
	w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent) // staff for the patient
	w.clinician.Do("POST", epPath(w.a, e, "/complete-review"), nil).Want(t, http.StatusNoContent)
	v = read(w.clinician)
	if !slices.Equal(targets(v), []string{"set:poa_complete", "set:on_hold", "set:not_ready", "set:ready_for_admission", "cancel:cancelled"}) {
		t.Errorf("a clinician at Ready for POA: %q", targets(v))
	}
	if super := read(w.author); !slices.Contains(targets(super), "reopen:ready_for_review") {
		t.Errorf("a super clinician at Ready for POA: %q", targets(super))
	}
	if v.Undo != nil {
		t.Errorf("undo %+v: completing the review isn't undone (re-open is)", v.Undo)
	}
	w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": episode.ReadyForAdmission, "reason": "All clear", "revision": v.Episode.Revision}).
		Want(t, http.StatusNoContent)
	v = read(w.clinician)
	if v.Undo == nil || v.Undo.To != episode.ReadyForPOA || v.Undo.Label != "Ready for POA" {
		t.Errorf("undo %+v", v.Undo)
	}
	var got []string
	for _, h := range v.StatusHistory {
		got = append(got, h.Move+">"+h.ToStatus)
	}
	if !slices.Equal(got, []string{"create>hq_not_complete", "submit>ready_for_review", "complete_review>ready_for_poa", "set>ready_for_admission"}) {
		t.Errorf("history %q", got)
	}
	if last := v.StatusHistory[3]; last.Reason != "All clear" || last.UserName == nil || *last.FromStatus != episode.ReadyForPOA {
		t.Errorf("last %+v", last)
	}
	if v.Episode.Status != episode.ReadyForAdmission || v.Episode.Revision != 3 {
		t.Errorf("episode %+v", v.Episode)
	}
}

// Two tabs (pending.md §1a, S20 brief): the Nurse ASA saved in one and the Anaesthetist ASA in the
// other, both from the revision they loaded. The second is refused with the stale message, so the
// first isn't lost.
func TestTwoTabsDontOverwriteEachOther(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.ReadyForPOA)
	loaded := revisionOf(t, w, e.ID)
	w.clinician.Do("PATCH", epPath(w.a, e, ""), map[string]any{"nurseAsa": 2, "revision": loaded}).Want(t, http.StatusNoContent)
	r := w.clinician.Do("PATCH", epPath(w.a, e, ""), map[string]any{"anaesthetistAsa": 3, "revision": loaded})
	refused(t, r, http.StatusConflict)
	if !strings.Contains(r.Message(), "changed since you opened it") || !strings.Contains(string(r.Body), `"code":"stale"`) {
		t.Errorf("response %s", r)
	}
	if got := readEpisode(t, w, e.ID); got.NurseAsa == nil || *got.NurseAsa != 2 || got.AnaesthetistAsa != nil {
		t.Errorf("episode %+v", got)
	}
	// After a reload, the second tab's change goes in.
	w.clinician.Do("PATCH", epPath(w.a, e, ""), map[string]any{"anaesthetistAsa": 3, "revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
	if got := readEpisode(t, w, e.ID); got.NurseAsa == nil || *got.NurseAsa != 2 || got.AnaesthetistAsa == nil || *got.AnaesthetistAsa != 3 {
		t.Errorf("episode %+v", got)
	}
}
