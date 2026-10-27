//go:build integration

package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

// The HQ rules' other actions (plan C6, session S43; ASA-02, ASA-03, TSK-01, EPI-02). When an HQ is
// sent and the evaluate service's result (a recording fake here) has a rule that fires, Go makes its
// writes in the send's transaction: a task for a team naming the rule, a flag on the episode, the
// anaesthetist's ASA grade required before Ready for admission, and the statuses it suggests. A
// clinician clears a flag with a reason (POST …/flags/{fid}/clear, S43 answers 1).

const (
	bookReview  = `{"kind": "task", "title": "Book anaesthetic review", "team": "anaesthetist"}`
	reviewFlag  = `{"kind": "flag", "name": "Anaesthetic review"}`
	requireASA  = `{"kind": "requireAnaesthetistAsa"}`
	needsGrade  = "Add the anaesthetist's ASA grade first."
	highRiskReq = "The rule “High risk” requires the anaesthetist's ASA grade before Ready for admission"
)

var highRisk = triageResult("III", "High risk", bookReview, reviewFlag, requireASA)

// ruleTaskRow is a tasks row with its maker.
type ruleTaskRow struct {
	Title, Team string
	Rule        *string
	CreatedBy   *uuid.UUID
	Done        bool
}

func ruleTasks(t *testing.T, w *clinic, id uuid.UUID) []ruleTaskRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT title, team, created_by_rule, created_by, done_at IS NOT NULL
		FROM tasks WHERE episode_id = $1 ORDER BY created_at, id`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []ruleTaskRow
	for rows.Next() {
		var r ruleTaskRow
		if err := rows.Scan(&r.Title, &r.Team, &r.Rule, &r.CreatedBy, &r.Done); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// flagRow is an episode_flags row.
type flagRow struct {
	ID            uuid.UUID
	Name, Rule    string
	ClearedBy     *uuid.UUID
	ClearedAt     *time.Time
	ClearedReason *string
}

func flagsOf(t *testing.T, w *clinic, id uuid.UUID) []flagRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT id, name, rule, cleared_by, cleared_at, cleared_reason
		FROM episode_flags WHERE episode_id = $1 ORDER BY created_at, id`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []flagRow
	for rows.Next() {
		var r flagRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Rule, &r.ClearedBy, &r.ClearedAt, &r.ClearedReason); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func requiredBy(t *testing.T, w *clinic, id uuid.UUID) *string {
	t.Helper()
	var rule *string
	if err := w.db.QueryRow(t.Context(), `SELECT anaesthetist_asa_required_by FROM episodes WHERE id = $1`, id).Scan(&rule); err != nil {
		t.Fatal(err)
	}
	return rule
}

// sendAgain puts a sent episode back to waiting for the patient, as if its HQ was re-opened and
// sent again (no route does that yet: "running rules again later" isn't built), then sends it.
func sendAgain(t *testing.T, w *clinic, e apitest.Episode, res evaluate.Result) {
	t.Helper()
	setStatus(t, w, e, episode.HQNotComplete)
	w.anon.Evaluator.Reply(res)
	w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
}

// episodeJSON is GET …/episodes/{eid}'s part S43 adds.
type episodeJSON struct {
	Episode struct {
		Status                    string
		AnaesthetistAsaRequiredBy *string
		RuleSuggestions           []struct{ Rule, Status string }
	}
	Flags []struct {
		ID                           uuid.UUID
		Name, Rule                   string
		CreatedAt                    time.Time
		ClearedAt                    *time.Time
		ClearedByName, ClearedReason *string
	}
	Moves []struct {
		To, Move, Blocked string
	}
	Undo *struct{ To, Blocked string }
}

func getEpisodeJSON(t *testing.T, w *clinic, e apitest.Episode) episodeJSON {
	t.Helper()
	r := w.clinician.Do("GET", epPath(w.a, e, ""), nil)
	r.Want(t, http.StatusOK)
	var v episodeJSON
	r.Decode(t, &v)
	return v
}

func TestRuleActionsOnSend(t *testing.T) {
	t.Parallel()
	w := newClinic(t)

	t.Run("high risk, sent by the patient: the task, the flag, the requirement; Ready for review", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
		w.anon.Evaluator.Reply(highRisk)
		p.Do("POST", "/api/p/episodes/"+e.ID.String()+"/submit", nil).Want(t, http.StatusNoContent)

		tasks := ruleTasks(t, w, e.ID)
		if len(tasks) != 1 || tasks[0].Title != "Book anaesthetic review" || tasks[0].Team != "anaesthetist" ||
			tasks[0].Rule == nil || *tasks[0].Rule != "High risk" || tasks[0].CreatedBy != nil {
			t.Errorf("tasks = %+v", tasks)
		}
		flags := flagsOf(t, w, e.ID)
		if len(flags) != 1 || flags[0].Name != "Anaesthetic review" || flags[0].Rule != "High risk" || flags[0].ClearedAt != nil {
			t.Errorf("flags = %+v", flags)
		}
		if r := requiredBy(t, w, e.ID); r == nil || *r != "High risk" {
			t.Errorf("required by %v", r)
		}
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForReview {
			t.Errorf("status = %s", got.Status)
		}
		notes := notesOf(t, w, e.ID)
		for _, want := range []string{
			"auto:Task added by the rule “High risk” for the anaesthetist team: Book anaesthetic review",
			"auto:Flag added by the rule “High risk”: Anaesthetic review",
			"auto:" + highRiskReq,
		} {
			if !slices.Contains(notes, want) {
				t.Errorf("notes %q lack %q", notes, want)
			}
		}
		// Anaesthetic review needed (EPI-02): any open task for the anaesthetist team.
		if !slices.Contains(listIDs(t, w.clinician, at(w.a, "/episodes?worklist=anaesthetic_review")), e.ID) {
			t.Error("not on Anaesthetic review needed")
		}
		// The episode's JSON and the list's row carry the flag; the Tasks panel names the rule.
		v := getEpisodeJSON(t, w, e)
		if len(v.Flags) != 1 || v.Flags[0].Name != "Anaesthetic review" || v.Flags[0].Rule != "High risk" || v.Flags[0].ClearedAt != nil {
			t.Errorf("flags = %+v", v.Flags)
		}
		if r := v.Episode.AnaesthetistAsaRequiredBy; r == nil || *r != "High risk" {
			t.Errorf("anaesthetistAsaRequiredBy = %v", r)
		}
		var list []struct {
			ID    uuid.UUID
			Flags []string
		}
		lr := w.clinician.Do("GET", at(w.a, "/episodes"), nil)
		lr.Want(t, http.StatusOK)
		lr.Decode(t, &list)
		for _, row := range list {
			if row.ID == e.ID && !slices.Equal(row.Flags, []string{"Anaesthetic review"}) {
				t.Errorf("list row flags = %q", row.Flags)
			}
			if row.ID != e.ID && row.Flags == nil {
				t.Errorf("a row without flags has null, not []")
			}
		}
		var tl []struct{ CreatedByRule, CreatedByName *string }
		tr := w.clinician.Do("GET", epPath(w.a, e, "/tasks"), nil)
		tr.Want(t, http.StatusOK)
		tr.Decode(t, &tl)
		if len(tl) != 1 || tl[0].CreatedByRule == nil || *tl[0].CreatedByRule != "High risk" || tl[0].CreatedByName != nil {
			t.Errorf("tasks JSON = %+v", tl)
		}
	})
	t.Run("sent by staff on the patient's behalf: the same", func(t *testing.T) {
		e := sendForPatient(t, w, highRisk)
		if len(ruleTasks(t, w, e.ID)) != 1 || len(flagsOf(t, w, e.ID)) != 1 || requiredBy(t, w, e.ID) == nil {
			t.Error("the rule's actions weren't made")
		}
	})
	t.Run("with Low risk to Ready for admission too: Ready for review instead (S42's guard)", func(t *testing.T) {
		res := highRisk
		res.Rules = append([]evaluate.FiredRule{{ID: "low", Name: "Low risk", Actions: []json.RawMessage{json.RawMessage(toAdmission)}}}, highRisk.Rules...)
		e := sendForPatient(t, w, res)
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForReview {
			t.Errorf("status = %s", got.Status)
		}
		if requiredBy(t, w, e.ID) == nil || len(ruleTasks(t, w, e.ID)) != 1 {
			t.Error("the high-risk rule's actions weren't made")
		}
	})
	t.Run("a requirement standing from an earlier send holds a later Low risk too", func(t *testing.T) {
		e := sendForPatient(t, w, highRisk)
		sendAgain(t, w, e, lowRisk)
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForReview {
			t.Errorf("status = %s", got.Status)
		}
		want := "auto:Not auto-triaged by the rule “Low risk”: a rule requires the anaesthetist's ASA grade first. Review this HQ by hand."
		if !slices.Contains(notesOf(t, w, e.ID), want) {
			t.Errorf("notes %q", notesOf(t, w, e.ID))
		}
	})
	t.Run("one rule, the same task title for two teams: both tasks (verification 1, finding 1)", func(t *testing.T) {
		e := sendForPatient(t, w, triageResult("III", "High risk", `{"kind": "task", "title": "Book review", "team": "nurse"}`,
			`{"kind": "task", "title": "Book review", "team": "anaesthetist"}`))
		// One send's tasks share created_at, so their order is the ids' (random): compare the teams as a set.
		tasks := ruleTasks(t, w, e.ID)
		var teams []string
		for _, task := range tasks {
			teams = append(teams, task.Team)
		}
		slices.Sort(teams)
		if !slices.Equal(teams, []string{"anaesthetist", "nurse"}) {
			t.Errorf("tasks = %+v", tasks)
		}
		notes := strings.Join(notesOf(t, w, e.ID), "\n")
		for _, team := range []string{"nurse", "anaesthetist"} {
			if !strings.Contains(notes, "Task added by the rule “High risk” for the "+team+" team: Book review") {
				t.Errorf("no note for the %s team: %s", team, notes)
			}
		}
		if !slices.Contains(listIDs(t, w.clinician, at(w.a, "/episodes?worklist=anaesthetic_review")), e.ID) {
			t.Error("not on Anaesthetic review needed")
		}
		sendAgain(t, w, e, triageResult("III", "High risk", `{"kind": "task", "title": "Book review", "team": "nurse"}`,
			`{"kind": "task", "title": "Book review", "team": "anaesthetist"}`))
		if n := len(ruleTasks(t, w, e.ID)); n != 2 {
			t.Errorf("after a second send: %d tasks", n)
		}
	})
	t.Run("two rules adding the same flag name: two flags, the list row names it once (finding 4)", func(t *testing.T) {
		res := highRisk
		res.Rules = append(append([]evaluate.FiredRule{}, highRisk.Rules...),
			evaluate.FiredRule{ID: "airway", Name: "Airway", Actions: []json.RawMessage{json.RawMessage(reviewFlag)}})
		e := sendForPatient(t, w, res)
		if n := len(flagsOf(t, w, e.ID)); n != 2 {
			t.Errorf("%d flags", n)
		}
		var list []struct {
			ID    uuid.UUID
			Flags []string
		}
		lr := w.clinician.Do("GET", at(w.a, "/episodes"), nil)
		lr.Decode(t, &list)
		for _, row := range list {
			if row.ID == e.ID && !slices.Equal(row.Flags, []string{"Anaesthetic review"}) {
				t.Errorf("list row flags = %q", row.Flags)
			}
		}
	})
	t.Run("a suggested status: shown with the episode, nothing moves", func(t *testing.T) {
		e := sendForPatient(t, w, triageResult("I", "Fit", `{"kind": "suggestStatus", "status": "ready_for_poa"}`))
		v := getEpisodeJSON(t, w, e)
		if v.Episode.Status != episode.ReadyForReview || len(v.Episode.RuleSuggestions) != 1 ||
			v.Episode.RuleSuggestions[0].Rule != "Fit" || v.Episode.RuleSuggestions[0].Status != episode.ReadyForPOA {
			t.Errorf("episode = %+v", v.Episode)
		}
		if len(ruleTasks(t, w, e.ID)) != 0 || len(flagsOf(t, w, e.ID)) != 0 || requiredBy(t, w, e.ID) != nil {
			t.Error("a suggestion made other writes")
		}
	})
	t.Run("the fake failing: none of them", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		w.anon.Evaluator.Fail(errors.New("down"))
		w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		w.anon.Evaluator.Fail(nil)
		if len(ruleTasks(t, w, e.ID)) != 0 || len(flagsOf(t, w, e.ID)) != 0 || requiredBy(t, w, e.ID) != nil {
			t.Error("actions made without a result")
		}
		if v := getEpisodeJSON(t, w, e); len(v.Flags) != 0 || v.Episode.RuleSuggestions == nil || len(v.Episode.RuleSuggestions) != 0 {
			t.Errorf("episode JSON = %+v (flags and suggestions are [] when none)", v)
		}
		if r := w.clinician.Do("GET", epPath(w.a, e, ""), nil); !strings.Contains(r.String(), `"flags":[]`) { // finding 6: not null
			t.Errorf("flags not [] in %s", r)
		}
	})
	t.Run("sent again: the rule's open task and standing flag aren't added twice; a cleared flag and a done task come back", func(t *testing.T) {
		e := sendForPatient(t, w, highRisk)
		sendAgain(t, w, e, highRisk)
		if n, f := len(ruleTasks(t, w, e.ID)), len(flagsOf(t, w, e.ID)); n != 1 || f != 1 {
			t.Fatalf("after a second send: %d tasks, %d flags", n, f)
		}
		if n := strings.Count(strings.Join(notesOf(t, w, e.ID), "\n"), highRiskReq); n != 1 {
			t.Errorf("the requirement's note written %d times", n)
		}
		// The task done and the flag cleared, a third send makes them again.
		if _, err := w.db.Exec(t.Context(), `UPDATE tasks SET done_by = $2, done_at = now() WHERE episode_id = $1`, e.ID, w.cara.ID); err != nil {
			t.Fatal(err)
		}
		fid := flagsOf(t, w, e.ID)[0].ID
		w.clinician.Do("POST", epPath(w.a, e, "/flags/"+fid.String()+"/clear"), map[string]any{"reason": "Seen", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		sendAgain(t, w, e, highRisk)
		tasks, flags := ruleTasks(t, w, e.ID), flagsOf(t, w, e.ID)
		if len(tasks) != 2 || !tasks[0].Done || tasks[1].Done {
			t.Errorf("tasks = %+v", tasks)
		}
		if len(flags) != 2 || flags[0].ClearedAt == nil || flags[1].ClearedAt != nil {
			t.Errorf("flags = %+v", flags)
		}
	})
}

// While a rule's requirement is unmet, a move to Ready for admission is refused; saving the
// anaesthetist's grade meets it (S43).
func TestReadyForAdmissionNeedsTheAnaesthetistsASA(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	to := func(e apitest.Episode, status string) *apitest.Resp {
		return w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": status, "revision": revisionOf(t, w, e.ID)})
	}
	grade := func(e apitest.Episode, field string, g int) {
		w.clinician.Do("PATCH", epPath(w.a, e, ""), map[string]any{field: g, "revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
	}
	blocked := func(v episodeJSON) string {
		for _, m := range v.Moves {
			if m.To == episode.ReadyForAdmission {
				return m.Blocked
			}
		}
		return "(not offered)"
	}

	e := sendForPatient(t, w, highRisk)
	setStatus(t, w, e, episode.ReadyForPOA)
	if b := blocked(getEpisodeJSON(t, w, e)); b != needsGrade {
		t.Errorf("the status control's Ready for admission: blocked %q", b)
	}
	r := to(e, episode.ReadyForAdmission)
	refused(t, r, http.StatusConflict)
	if !strings.Contains(r.String(), needsGrade) {
		t.Errorf("refusal = %s", r)
	}
	to(e, episode.OnHold).Want(t, http.StatusNoContent) // other moves are fine
	grade(e, "nurseAsa", 3)                             // the nurse's grade doesn't meet it
	refused(t, to(e, episode.ReadyForAdmission), http.StatusConflict)

	grade(e, "anaesthetistAsa", 3)
	if b := blocked(getEpisodeJSON(t, w, e)); b != "" {
		t.Errorf("met, still blocked %q", b)
	}
	to(e, episode.ReadyForAdmission).Want(t, http.StatusNoContent)
	if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForAdmission {
		t.Errorf("status = %s", got.Status)
	}

	t.Run("the grade cleared again: unmet again", func(t *testing.T) {
		e := sendForPatient(t, w, highRisk)
		setStatus(t, w, e, episode.ReadyForPOA)
		grade(e, "anaesthetistAsa", 4)
		grade(e, "anaesthetistAsa", 0)
		refused(t, to(e, episode.ReadyForAdmission), http.StatusConflict)
	})
	t.Run("an undo back to Ready for admission is refused too while unmet", func(t *testing.T) {
		e := sendForPatient(t, w, highRisk)
		setStatus(t, w, e, episode.ReadyForPOA)
		grade(e, "anaesthetistAsa", 3)
		to(e, episode.ReadyForAdmission).Want(t, http.StatusNoContent)
		to(e, episode.OnHold).Want(t, http.StatusNoContent)
		grade(e, "anaesthetistAsa", 0)
		// The Undo is offered with why it's refused for now, as the dropdown's move is (verification 1, finding 2).
		if u := getEpisodeJSON(t, w, e).Undo; u == nil || u.To != episode.ReadyForAdmission || u.Blocked != needsGrade {
			t.Errorf("undo = %+v", u)
		}
		r := w.clinician.Do("POST", epPath(w.a, e, "/undo"), map[string]any{"reason": "Back", "revision": revisionOf(t, w, e.ID)})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.String(), needsGrade) {
			t.Errorf("refusal = %s", r)
		}
	})
	t.Run("no rule requires it: Ready for admission as before", func(t *testing.T) {
		e := sendForPatient(t, w, triageResult("III", "Look", reviewFlag))
		setStatus(t, w, e, episode.ReadyForPOA)
		if b := blocked(getEpisodeJSON(t, w, e)); b != "" {
			t.Errorf("blocked %q", b)
		}
		to(e, episode.ReadyForAdmission).Want(t, http.StatusNoContent)
	})
	t.Run("the fake failing: no requirement", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		w.anon.Evaluator.Fail(errors.New("down"))
		w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		w.anon.Evaluator.Fail(nil)
		setStatus(t, w, e, episode.ReadyForPOA)
		to(e, episode.ReadyForAdmission).Want(t, http.StatusNoContent)
	})
}

// POST …/episodes/{eid}/flags/{fid}/clear {reason, revision} (S43 answers 1): any clinician at the
// hospital clears a rule's flag with a short reason; who, when and why are recorded. Audited.
func TestClearFlag(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	clear := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, fid uuid.UUID, body any) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/flags/"+fid.String()+"/clear"), body)
	}
	body := func(e apitest.Episode) map[string]any {
		return map[string]any{"reason": "Reviewed with the anaesthetist", "revision": revisionOf(t, w, e.ID)}
	}
	flagged := func() (apitest.Episode, uuid.UUID) {
		e := sendForPatient(t, w, highRisk)
		return e, flagsOf(t, w, e.ID)[0].ID
	}
	standing := func(t *testing.T, e apitest.Episode) {
		t.Helper()
		if f := flagsOf(t, w, e.ID); f[0].ClearedAt != nil {
			t.Errorf("cleared: %+v", f)
		}
	}

	t.Run("happy path: who, when and why; a note; off the list; audited", func(t *testing.T) {
		e, fid := flagged()
		before := revisionOf(t, w, e.ID)
		a := oneAudit(t, w, "flag.clear", func() *apitest.Resp { return clear(w.clinician, w.a, e, fid, body(e)) }, http.StatusNoContent)
		if !slices.Contains(a.Episodes, e.ID) {
			t.Errorf("audit = %+v", a)
		}
		f := flagsOf(t, w, e.ID)[0]
		if f.ClearedBy == nil || *f.ClearedBy != w.cara.ID || f.ClearedAt == nil || f.ClearedReason == nil || *f.ClearedReason != "Reviewed with the anaesthetist" {
			t.Errorf("flag = %+v", f)
		}
		if revisionOf(t, w, e.ID) != before+1 {
			t.Error("the revision didn't move on")
		}
		ev := readEvents(t, w, e.ID)
		if last := ev[len(ev)-1]; last.Text != "Flag cleared: Anaesthetic review: Reviewed with the anaesthetist" || last.User == nil || *last.User != w.cara.ID {
			t.Errorf("note = %+v", last)
		}
		v := getEpisodeJSON(t, w, e)
		if len(v.Flags) != 1 || v.Flags[0].ClearedAt == nil || v.Flags[0].ClearedByName == nil || *v.Flags[0].ClearedByName != w.cara.Name ||
			v.Flags[0].ClearedReason == nil {
			t.Errorf("flags JSON = %+v", v.Flags)
		}
		var list []struct {
			ID    uuid.UUID
			Flags []string
		}
		lr := w.clinician.Do("GET", at(w.a, "/episodes"), nil)
		lr.Decode(t, &list)
		for _, row := range list {
			if row.ID == e.ID && len(row.Flags) != 0 {
				t.Errorf("a cleared flag still in the list: %q", row.Flags)
			}
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e, fid := flagged()
		refused(t, clear(w.anon, w.a, e, fid, body(e)), http.StatusUnauthorized)
		standing(t, e)
	})
	t.Run("the role: a clinician at the trust's other hospital can't clear this one (404)", func(t *testing.T) {
		e, fid := flagged()
		refused(t, clear(w.site2, w.a, e, fid, body(e)), http.StatusNotFound)
		refused(t, clear(w.site2, w.a2, e, fid, body(e)), http.StatusNotFound)
		standing(t, e)
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e, fid := flagged()
		refused(t, clear(w.otherClin, w.a, e, fid, body(e)), http.StatusNotFound)
		refused(t, clear(w.otherClin, w.b, e, fid, body(e)), http.StatusNotFound)
		standing(t, e)
	})
	t.Run("invalid input: no reason, too long, no revision, not JSON; another episode's flag or none: 404", func(t *testing.T) {
		e, fid := flagged()
		rev := revisionOf(t, w, e.ID)
		for _, bad := range []any{
			map[string]any{"revision": rev},
			map[string]any{"reason": "   ", "revision": rev},
			map[string]any{"reason": strings.Repeat("x", 1001), "revision": rev},
			map[string]any{"reason": "Seen"},
			map[string]any{"reason": "Seen", "revision": -1},
			"nonsense",
		} {
			refused(t, clear(w.clinician, w.a, e, fid, bad), http.StatusBadRequest)
		}
		other, _ := flagged()
		refused(t, clear(w.clinician, w.a, other, fid, body(other)), http.StatusNotFound) // not that episode's flag
		refused(t, clear(w.clinician, w.a, e, uuid.New(), body(e)), http.StatusNotFound)
		refused(t, w.clinician.Do("POST", epPath(w.a, e, "/flags/abc/clear"), body(e)), http.StatusNotFound)
		standing(t, e)
	})
	t.Run("twice: 409", func(t *testing.T) {
		e, fid := flagged()
		clear(w.clinician, w.a, e, fid, body(e)).Want(t, http.StatusNoContent)
		r := clear(w.clinician, w.a, e, fid, body(e))
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.String(), "This flag is already cleared.") {
			t.Errorf("refusal = %s", r)
		}
	})
	t.Run("a stale revision: 409 stale", func(t *testing.T) {
		e, fid := flagged()
		r := clear(w.clinician, w.a, e, fid, map[string]any{"reason": "Seen", "revision": revisionOf(t, w, e.ID) - 1})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.String(), `"stale"`) {
			t.Errorf("response = %s", r)
		}
		standing(t, e)
	})
	t.Run("an archived or cancelled episode's flag: 409", func(t *testing.T) {
		e, fid := flagged()
		w.author.Do("POST", epPath(w.a, e, "/archive"), map[string]any{"revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
		refused(t, clear(w.clinician, w.a, e, fid, body(e)), http.StatusConflict)
		c, cfid := flagged()
		w.clinician.Do("POST", epPath(w.a, c, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, c.ID)}).
			Want(t, http.StatusNoContent)
		refused(t, clear(w.clinician, w.a, c, cfid, body(c)), http.StatusConflict)
		standing(t, e)
		standing(t, c)
	})
}
