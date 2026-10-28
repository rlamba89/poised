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
)

// The anaesthetist's ASA grade as a history (plan C6, session S44b; Rahul, 27 Oct): a grade, once
// given, is never changed or cleared. A later grade is added on top, with a reason once a grade
// counts; a grade given by mistake is marked entered in error, with a reason, and the one before it
// counts again. The grade that counts is the latest not entered in error.
//
//   - POST …/episodes/{eid}/anaesthetist-asa {grade, reason, revision}
//   - POST …/episodes/{eid}/anaesthetist-asa/{gid}/entered-in-error {reason, revision}
//
// The episode PATCH no longer takes the grade.

const (
	gradeNeedsReason = "Say why the grade is changed."
	onlyGradeMsg     = "Add the correct grade first, or move the episode back from Ready for admission."
)

// gradeRow is an anaesthetist_asa_grades row.
type gradeRow struct {
	ID          uuid.UUID
	Grade       int
	GivenBy     *uuid.UUID
	Reason      *string
	ErrorBy     *uuid.UUID
	ErrorAt     *time.Time
	ErrorReason *string
}

func gradesOf(t *testing.T, w *clinic, id uuid.UUID) []gradeRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT id, grade, given_by, reason, error_by, error_at, error_reason
		FROM anaesthetist_asa_grades WHERE episode_id = $1 ORDER BY given_at, id`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []gradeRow
	for rows.Next() {
		var g gradeRow
		if err := rows.Scan(&g.ID, &g.Grade, &g.GivenBy, &g.Reason, &g.ErrorBy, &g.ErrorAt, &g.ErrorReason); err != nil {
			t.Fatal(err)
		}
		out = append(out, g)
	}
	return out
}

// gradesView is GET …/episodes/{eid}'s grade and history.
type gradesView struct {
	Episode struct {
		AnaesthetistAsa *int
		Revision        int32
	}
	AnaesthetistAsaGrades []struct {
		ID                       uuid.UUID
		Grade                    int
		GivenAt                  time.Time
		GivenByName, Reason      *string
		ErrorAt                  *time.Time
		ErrorByName, ErrorReason *string
	}
}

func gradesJSON(t *testing.T, w *clinic, e apitest.Episode) gradesView {
	t.Helper()
	r := w.clinician.Do("GET", epPath(w.a, e, ""), nil)
	r.Want(t, http.StatusOK)
	var v gradesView
	r.Decode(t, &v)
	return v
}

func addGrade(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body any) *apitest.Resp {
	return c.Do("POST", epPath(h, e, "/anaesthetist-asa"), body)
}

func markGrade(c *apitest.Client, h apitest.Hospital, e apitest.Episode, gid uuid.UUID, body any) *apitest.Resp {
	return c.Do("POST", epPath(h, e, "/anaesthetist-asa/"+gid.String()+"/entered-in-error"), body)
}

// grade adds a grade as Cara, from the episode's current revision.
func grade(t *testing.T, w *clinic, e apitest.Episode, g int, reason string) uuid.UUID {
	t.Helper()
	r := addGrade(w.clinician, w.a, e, map[string]any{"grade": g, "reason": reason, "revision": revisionOf(t, w, e.ID)})
	r.Want(t, http.StatusCreated)
	var out struct{ ID uuid.UUID }
	r.Decode(t, &out)
	return out.ID
}

// markInError marks a grade entered in error as Cara, from the current revision.
func markInError(t *testing.T, w *clinic, e apitest.Episode, gid uuid.UUID, reason string) *apitest.Resp {
	t.Helper()
	return markGrade(w.clinician, w.a, e, gid, map[string]any{"reason": reason, "revision": revisionOf(t, w, e.ID)})
}

func countingOf(t *testing.T, w *clinic, e apitest.Episode) *int {
	t.Helper()
	return readEpisode(t, w, e.ID).AnaesthetistAsa
}

func TestAddAnaesthetistGrade(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	body := func(e apitest.Episode, g int, reason string) map[string]any {
		return map[string]any{"grade": g, "reason": reason, "revision": revisionOf(t, w, e.ID)}
	}
	ready := func() apitest.Episode {
		e := w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, e, episode.ReadyForPOA)
		return e
	}
	none := func(t *testing.T, e apitest.Episode) {
		t.Helper()
		if g := gradesOf(t, w, e.ID); len(g) != 0 {
			t.Errorf("grades = %+v, want none", g)
		}
	}

	t.Run("happy path: a first grade needs no reason; a re-grade needs one; both kept, the latest counts; audited", func(t *testing.T) {
		e := ready()
		before := revisionOf(t, w, e.ID)
		a := oneAudit(t, w, "asa.add", func() *apitest.Resp { return addGrade(w.clinician, w.a, e, body(e, 3, "")) }, http.StatusCreated)
		if !slices.Contains(a.Episodes, e.ID) {
			t.Errorf("audit = %+v", a)
		}
		if revisionOf(t, w, e.ID) != before+1 {
			t.Error("the revision didn't move on")
		}
		if c := countingOf(t, w, e); c == nil || *c != 3 {
			t.Errorf("counting = %v, want 3", c)
		}

		r := addGrade(w.clinician, w.a, e, body(e, 4, "  "))
		refused(t, r, http.StatusUnprocessableEntity)
		if !strings.Contains(r.Message(), gradeNeedsReason) {
			t.Errorf("refusal = %s", r)
		}
		addGrade(w.clinician, w.a, e, body(e, 4, " New echo result ")).Want(t, http.StatusCreated)

		g := gradesOf(t, w, e.ID)
		if len(g) != 2 || g[0].Grade != 3 || g[0].Reason != nil || g[0].GivenBy == nil || *g[0].GivenBy != w.cara.ID ||
			g[1].Grade != 4 || g[1].Reason == nil || *g[1].Reason != "New echo result" || g[1].ErrorAt != nil {
			t.Errorf("grades = %+v", g)
		}
		if c := countingOf(t, w, e); c == nil || *c != 4 {
			t.Errorf("counting = %v, want 4", c)
		}
		v := gradesJSON(t, w, e)
		if v.Episode.AnaesthetistAsa == nil || *v.Episode.AnaesthetistAsa != 4 || len(v.AnaesthetistAsaGrades) != 2 {
			t.Fatalf("view = %+v", v)
		}
		if h := v.AnaesthetistAsaGrades[1]; h.Grade != 4 || h.GivenByName == nil || *h.GivenByName != w.cara.Name || h.Reason == nil || h.GivenAt.IsZero() {
			t.Errorf("history = %+v", v.AnaesthetistAsaGrades)
		}
		var notes []string
		for _, ev := range readEvents(t, w, e.ID) {
			notes = append(notes, ev.Text)
		}
		want := []string{"Anaesthetist ASA grade set to 3", "Anaesthetist ASA grade changed from 3 to 4: New echo result"}
		if !slices.Equal(notes, want) {
			t.Errorf("notes = %q, want %q", notes, want)
		}
	})
	t.Run("the same grade again, with a reason, is a confirmation (test report 1, exploring note 1)", func(t *testing.T) {
		e := ready()
		grade(t, w, e, 3, "")
		grade(t, w, e, 3, "Second anaesthetist agrees")
		ev := readEvents(t, w, e.ID)
		if last := ev[len(ev)-1]; last.Text != "Anaesthetist ASA grade confirmed as 3: Second anaesthetist agrees" {
			t.Errorf("note = %q", last.Text)
		}
		if g := gradesOf(t, w, e.ID); len(g) != 2 {
			t.Errorf("grades = %+v", g)
		}
	})
	t.Run("every grade after the episode's first needs a reason, even after the only one was entered in error (verification 1)", func(t *testing.T) {
		e := ready()
		first := grade(t, w, e, 5, "")
		markInError(t, w, e, first, "Wrong patient").Want(t, http.StatusNoContent)
		r := addGrade(w.clinician, w.a, e, body(e, 2, ""))
		refused(t, r, http.StatusUnprocessableEntity)
		if !strings.Contains(r.Message(), gradeNeedsReason) {
			t.Errorf("refusal = %s", r)
		}
		grade(t, w, e, 2, "The correct patient's notes")
		if c := countingOf(t, w, e); c == nil || *c != 2 {
			t.Errorf("counting = %v, want 2", c)
		}
		ev := readEvents(t, w, e.ID)
		if last := ev[len(ev)-1]; last.Text != "Anaesthetist ASA grade set to 2: The correct patient's notes" {
			t.Errorf("note = %q", last.Text) // nothing counted, so "set to", with the reason
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := ready()
		refused(t, addGrade(w.anon, w.a, e, body(e, 3, "")), http.StatusUnauthorized)
		none(t, e)
	})
	t.Run("the role: a clinician at the trust's other hospital can't add one here (404)", func(t *testing.T) {
		e := ready()
		refused(t, addGrade(w.site2, w.a, e, body(e, 3, "")), http.StatusNotFound)
		refused(t, addGrade(w.site2, w.a2, e, body(e, 3, "")), http.StatusNotFound)
		none(t, e)
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e := ready()
		refused(t, addGrade(w.otherClin, w.a, e, body(e, 3, "")), http.StatusNotFound)
		refused(t, addGrade(w.otherClin, w.b, e, body(e, 3, "")), http.StatusNotFound)
		none(t, e)
	})
	t.Run("invalid input: no grade, out of range, no revision, a reason too long, not JSON", func(t *testing.T) {
		e := ready()
		rev := revisionOf(t, w, e.ID)
		for _, bad := range []any{
			map[string]any{"revision": rev},
			map[string]any{"grade": 0, "revision": rev},
			map[string]any{"grade": 7, "revision": rev},
			map[string]any{"grade": "III", "revision": rev},
			map[string]any{"grade": 3},
			map[string]any{"grade": 3, "revision": -1},
			map[string]any{"grade": 3, "reason": strings.Repeat("x", 1001), "revision": rev},
			"nonsense",
		} {
			refused(t, addGrade(w.clinician, w.a, e, bad), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/abc/anaesthetist-asa"), body(e, 3, "")), http.StatusNotFound)
		none(t, e)
	})
	t.Run("a stale view (a grade added in another tab meanwhile): 409 stale, nothing added", func(t *testing.T) {
		e := ready()
		loaded := revisionOf(t, w, e.ID)
		grade(t, w, e, 3, "") // the other tab
		r := addGrade(w.clinician, w.a, e, map[string]any{"grade": 4, "reason": "Seen", "revision": loaded})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.String(), `"stale"`) {
			t.Errorf("response = %s", r)
		}
		if g := gradesOf(t, w, e.ID); len(g) != 1 {
			t.Errorf("grades = %+v", g)
		}
	})
	t.Run("a cancelled or archived episode: 409", func(t *testing.T) {
		for _, status := range []string{episode.Cancelled, episode.Archived} {
			e := ready()
			setStatus(t, w, e, status)
			refused(t, addGrade(w.clinician, w.a, e, body(e, 3, "")), http.StatusConflict)
			none(t, e)
		}
	})
	t.Run("the episode PATCH refuses the grade (400) and changes nothing", func(t *testing.T) {
		e := ready()
		grade(t, w, e, 3, "")
		rev := revisionOf(t, w, e.ID)
		for _, bad := range []map[string]any{
			{"anaesthetistAsa": 4, "revision": rev},
			{"anaesthetistAsa": 0, "revision": rev},
			{"anaesthetistAsa": nil, "revision": rev},
			{"nurseAsa": 2, "anaesthetistAsa": 4, "revision": rev},
		} {
			r := w.clinician.Do("PATCH", epPath(w.a, e, ""), bad)
			refused(t, r, http.StatusBadRequest)
			if !strings.Contains(r.Message(), "anaesthetist's ASA grade") {
				t.Errorf("refusal = %s", r)
			}
		}
		if got := readEpisode(t, w, e.ID); got.NurseAsa != nil || got.AnaesthetistAsa == nil || *got.AnaesthetistAsa != 3 {
			t.Errorf("episode = %+v", got)
		}
		if revisionOf(t, w, e.ID) != rev {
			t.Error("the revision moved on")
		}
	})
	t.Run("no edit and no delete: no route, and the database refuses a change", func(t *testing.T) {
		e := ready()
		gid := grade(t, w, e, 3, "")
		path := epPath(w.a, e, "/anaesthetist-asa/"+gid.String())
		for _, method := range []string{"PATCH", "PUT", "DELETE"} {
			for _, p := range []string{path, epPath(w.a, e, "/anaesthetist-asa")} {
				if r := w.clinician.Do(method, p, map[string]any{"grade": 4, "revision": revisionOf(t, w, e.ID)}); r.Code != http.StatusNotFound && r.Code != http.StatusMethodNotAllowed {
					t.Errorf("%s %s = %s", method, p, r)
				}
			}
		}
		for _, change := range []string{
			`UPDATE anaesthetist_asa_grades SET grade = 4 WHERE id = $1`,
			`UPDATE anaesthetist_asa_grades SET reason = 'Changed' WHERE id = $1`,
			`UPDATE anaesthetist_asa_grades SET given_by = NULL WHERE id = $1`,
		} {
			if _, err := w.db.Exec(t.Context(), change, gid); err == nil || !strings.Contains(err.Error(), "can't be changed") {
				t.Errorf("%s: err = %v", change, err)
			}
		}
		markInError(t, w, e, gid, "Wrong patient").Want(t, http.StatusNoContent)
		if _, err := w.db.Exec(t.Context(), `UPDATE anaesthetist_asa_grades SET error_by = NULL, error_at = NULL, error_reason = NULL WHERE id = $1`, gid); err == nil {
			t.Error("an entered-in-error mark was taken off")
		}
		if g := gradesOf(t, w, e.ID); len(g) != 1 || g[0].Grade != 3 || g[0].ErrorAt == nil {
			t.Errorf("grades = %+v", g)
		}
		// Nor a delete (verification 1, finding 3), though an episode's own deletion still takes its grades.
		if _, err := w.db.Exec(t.Context(), `DELETE FROM anaesthetist_asa_grades WHERE id = $1`, gid); err == nil || !strings.Contains(err.Error(), "can't be deleted") {
			t.Errorf("delete: err = %v", err)
		}
		if g := gradesOf(t, w, e.ID); len(g) != 1 {
			t.Errorf("grades = %+v", g)
		}
		gone := ready()
		grade(t, w, gone, 3, "")
		if _, err := w.db.Exec(t.Context(), `DELETE FROM login_links WHERE episode_id = $1`, gone.ID); err != nil { // no cascade there
			t.Fatal(err)
		}
		if _, err := w.db.Exec(t.Context(), `DELETE FROM episodes WHERE id = $1`, gone.ID); err != nil {
			t.Fatalf("deleting the episode: %v", err)
		}
		if g := gradesOf(t, w, gone.ID); len(g) != 0 {
			t.Errorf("grades left = %+v", g)
		}
	})
}

func TestMarkAnaesthetistGradeInError(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	body := func(e apitest.Episode) map[string]any {
		return map[string]any{"reason": "Wrong patient's notes", "revision": revisionOf(t, w, e.ID)}
	}
	graded := func(grades ...int) (apitest.Episode, []uuid.UUID) {
		e := w.fx.Episode(t, w.hq, w.cara)
		setStatus(t, w, e, episode.ReadyForPOA)
		var ids []uuid.UUID
		for i, g := range grades {
			reason := ""
			if i > 0 {
				reason = "Re-graded"
			}
			ids = append(ids, grade(t, w, e, g, reason))
		}
		return e, ids
	}
	unmarked := func(t *testing.T, e apitest.Episode) {
		t.Helper()
		for _, g := range gradesOf(t, w, e.ID) {
			if g.ErrorAt != nil {
				t.Errorf("marked: %+v", g)
			}
		}
	}

	t.Run("happy path: kept, struck through with who, when and why; the one before counts again; audited", func(t *testing.T) {
		e, ids := graded(3, 4)
		before := revisionOf(t, w, e.ID)
		a := oneAudit(t, w, "asa.error", func() *apitest.Resp { return markGrade(w.clinician, w.a, e, ids[1], body(e)) }, http.StatusNoContent)
		if !slices.Contains(a.Episodes, e.ID) {
			t.Errorf("audit = %+v", a)
		}
		if revisionOf(t, w, e.ID) != before+1 {
			t.Error("the revision didn't move on")
		}
		g := gradesOf(t, w, e.ID)
		if len(g) != 2 || g[1].ErrorBy == nil || *g[1].ErrorBy != w.cara.ID || g[1].ErrorAt == nil || g[1].ErrorReason == nil ||
			*g[1].ErrorReason != "Wrong patient's notes" || g[1].Grade != 4 || g[0].ErrorAt != nil {
			t.Errorf("grades = %+v", g)
		}
		if c := countingOf(t, w, e); c == nil || *c != 3 {
			t.Errorf("counting = %v, want 3 again", c)
		}
		v := gradesJSON(t, w, e)
		if h := v.AnaesthetistAsaGrades[1]; h.ErrorAt == nil || h.ErrorByName == nil || *h.ErrorByName != w.cara.Name || h.ErrorReason == nil {
			t.Errorf("history = %+v", v.AnaesthetistAsaGrades)
		}
		ev := readEvents(t, w, e.ID)
		if last := ev[len(ev)-1]; last.Text != "Anaesthetist ASA grade 4 marked entered in error: Wrong patient's notes" || last.User == nil || *last.User != w.cara.ID {
			t.Errorf("note = %+v", last)
		}
		// An earlier grade, not the one that counts, can be marked too: the latest still counts.
		e2, ids2 := graded(2, 3)
		markInError(t, w, e2, ids2[0], "Typed in the wrong box").Want(t, http.StatusNoContent)
		if c := countingOf(t, w, e2); c == nil || *c != 3 {
			t.Errorf("counting = %v, want 3", c)
		}
	})
	t.Run("once marked, it can't be unmarked or marked again: 409", func(t *testing.T) {
		e, ids := graded(3)
		markInError(t, w, e, ids[0], "Wrong patient").Want(t, http.StatusNoContent)
		r := markInError(t, w, e, ids[0], "Again")
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "already marked entered in error") {
			t.Errorf("refusal = %s", r)
		}
		if g := gradesOf(t, w, e.ID); *g[0].ErrorReason != "Wrong patient" {
			t.Errorf("grades = %+v", g)
		}
		if c := countingOf(t, w, e); c != nil {
			t.Errorf("counting = %v, want none", *c)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e, ids := graded(3)
		refused(t, markGrade(w.anon, w.a, e, ids[0], body(e)), http.StatusUnauthorized)
		unmarked(t, e)
	})
	t.Run("the role: a clinician at the trust's other hospital can't mark one here (404)", func(t *testing.T) {
		e, ids := graded(3)
		refused(t, markGrade(w.site2, w.a, e, ids[0], body(e)), http.StatusNotFound)
		refused(t, markGrade(w.site2, w.a2, e, ids[0], body(e)), http.StatusNotFound)
		unmarked(t, e)
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e, ids := graded(3)
		refused(t, markGrade(w.otherClin, w.a, e, ids[0], body(e)), http.StatusNotFound)
		refused(t, markGrade(w.otherClin, w.b, e, ids[0], body(e)), http.StatusNotFound)
		unmarked(t, e)
	})
	t.Run("invalid input: no reason, too long, no revision, not JSON; another episode's grade or none: 404", func(t *testing.T) {
		e, ids := graded(3)
		rev := revisionOf(t, w, e.ID)
		for _, bad := range []any{
			map[string]any{"revision": rev},
			map[string]any{"reason": "   ", "revision": rev},
			map[string]any{"reason": strings.Repeat("x", 1001), "revision": rev},
			map[string]any{"reason": "Seen"},
			map[string]any{"reason": "Seen", "revision": -1},
			"nonsense",
		} {
			refused(t, markGrade(w.clinician, w.a, e, ids[0], bad), http.StatusBadRequest)
		}
		other, _ := graded(2)
		refused(t, markGrade(w.clinician, w.a, other, ids[0], body(other)), http.StatusNotFound)
		refused(t, markGrade(w.clinician, w.a, e, uuid.New(), body(e)), http.StatusNotFound)
		refused(t, w.clinician.Do("POST", epPath(w.a, e, "/anaesthetist-asa/abc/entered-in-error"), body(e)), http.StatusNotFound)
		unmarked(t, e)
	})
	t.Run("a stale revision: 409 stale", func(t *testing.T) {
		e, ids := graded(3)
		r := markGrade(w.clinician, w.a, e, ids[0], map[string]any{"reason": "Seen", "revision": revisionOf(t, w, e.ID) - 1})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.String(), `"stale"`) {
			t.Errorf("response = %s", r)
		}
		unmarked(t, e)
	})
	t.Run("a cancelled or archived episode: 409", func(t *testing.T) {
		for _, status := range []string{episode.Cancelled, episode.Archived} {
			e, ids := graded(3)
			setStatus(t, w, e, status)
			refused(t, markGrade(w.clinician, w.a, e, ids[0], body(e)), http.StatusConflict)
			unmarked(t, e)
		}
	})
}

// S43's requirement reads the grade that counts. While a rule requires the grade and the episode is
// at Ready for admission or later, the only counting grade can't be marked entered in error, so the
// grade can't go missing there: Admit can't be reached without one, and a cancelled episode's Undo
// back to Ready for admission keeps working.
func TestTheRequiredGradeCantGoMissing(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	to := func(e apitest.Episode, status string) *apitest.Resp {
		body := map[string]any{"to": status, "revision": revisionOf(t, w, e.ID)}
		if status == episode.Admitted || status == episode.Discharged {
			body["date"] = yesterday()
		}
		return w.clinician.Do("PATCH", epPath(w.a, e, "/status"), body)
	}
	atAdmission := func() (apitest.Episode, uuid.UUID) {
		e := sendForPatient(t, w, highRisk)
		setStatus(t, w, e, episode.ReadyForPOA)
		gid := grade(t, w, e, 3, "")
		to(e, episode.ReadyForAdmission).Want(t, http.StatusNoContent)
		return e, gid
	}
	refusedOnly := func(t *testing.T, e apitest.Episode, gid uuid.UUID) {
		t.Helper()
		r := markInError(t, w, e, gid, "Wrong patient")
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), onlyGradeMsg) {
			t.Errorf("refusal = %s", r)
		}
		if c := countingOf(t, w, e); c == nil {
			t.Error("the grade went missing")
		}
	}

	t.Run("at Ready for admission: the only grade can't be marked; Admit works", func(t *testing.T) {
		e, gid := atAdmission()
		refusedOnly(t, e, gid)
		to(e, episode.Admitted).Want(t, http.StatusNoContent)
		refusedOnly(t, e, gid) // admitted
		to(e, episode.Discharged).Want(t, http.StatusNoContent)
		refusedOnly(t, e, gid) // discharged
	})
	t.Run("with an earlier grade to fall back on, the latest can be marked; then the earlier can't", func(t *testing.T) {
		e, first := atAdmission()
		latest := grade(t, w, e, 4, "New echo result")
		markInError(t, w, e, latest, "Wrong patient").Want(t, http.StatusNoContent)
		if c := countingOf(t, w, e); c == nil || *c != 3 {
			t.Errorf("counting = %v", c)
		}
		refusedOnly(t, e, first)
	})
	t.Run("admitted, then undone: the Undo back to Ready for admission still works (the grade stayed)", func(t *testing.T) {
		e, gid := atAdmission()
		to(e, episode.Admitted).Want(t, http.StatusNoContent)
		refusedOnly(t, e, gid)
		if u := getEpisodeJSON(t, w, e).Undo; u == nil || u.To != episode.ReadyForAdmission || u.Blocked != "" {
			t.Errorf("undo = %+v", u)
		}
		w.clinician.Do("POST", epPath(w.a, e, "/undo"), map[string]any{"reason": "Admitted by mistake", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForAdmission {
			t.Errorf("status = %s", got.Status)
		}
	})
	t.Run("cancelled: the grade is kept and can't be marked (read-only)", func(t *testing.T) {
		e, gid := atAdmission()
		w.clinician.Do("POST", epPath(w.a, e, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		refused(t, markInError(t, w, e, gid, "Wrong patient"), http.StatusConflict)
		if c := countingOf(t, w, e); c == nil || *c != 3 {
			t.Errorf("counting = %v", c)
		}
	})
	t.Run("before Ready for admission it can be marked; then the move there is refused until a grade counts", func(t *testing.T) {
		e := sendForPatient(t, w, highRisk)
		setStatus(t, w, e, episode.ReadyForPOA)
		gid := grade(t, w, e, 3, "")
		markInError(t, w, e, gid, "Wrong patient").Want(t, http.StatusNoContent)
		r := to(e, episode.ReadyForAdmission)
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), needsGrade) {
			t.Errorf("refusal = %s", r)
		}
		grade(t, w, e, 2, "The correct grade")
		to(e, episode.ReadyForAdmission).Want(t, http.StatusNoContent)
	})
	t.Run("no rule requires it: the only grade can be marked at Ready for admission", func(t *testing.T) {
		e := sendForPatient(t, w, triageResult("III", "Look", reviewFlag))
		setStatus(t, w, e, episode.ReadyForAdmission)
		gid := grade(t, w, e, 3, "")
		markInError(t, w, e, gid, "Wrong patient").Want(t, http.StatusNoContent)
	})
}
