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

// Auto-triage (plan C6, session S42; ASA-02, A-8, EPI-02, EPI-03, EPI-07). When an HQ is sent (by the
// patient, or staff for them) and the evaluate service's result (a recording fake here) has a rule
// that auto-triages, the episode moves straight on from Ready for review to the rule's target: one
// submit and one auto_triage event naming the rule, an automatic General notes line, and the
// Auto-triaged worklist until a clinician clicks Checked (POST …/triage-checked, S42 answers 1). The
// patient's status message goes at once, or on Checked if the HQ says so (answers 2). If the
// evaluator failed, no rule auto-triages, or a guard stops it, the episode stays Ready for review.

const lowRiskNote = "Auto-triaged by the rule “Low risk”: ASA II, no red or amber flags → Ready for admission"

// triageResult is the evaluator's result when one rule fires with these actions.
func triageResult(asa, rule string, actions ...string) evaluate.Result {
	r := evaluate.Result{Evaluator: "clinical@test", Flags: []evaluate.Flag{}, PatientSets: []evaluate.PatientSet{}, Shown: []uuid.UUID{},
		Rules: []evaluate.FiredRule{{ID: "r1", Name: rule, Line: "Rule fires: " + rule}}}
	if asa != "" {
		r.SuggestedASA = &evaluate.SuggestedASA{Grade: asa}
	}
	for _, a := range actions {
		r.Rules[0].Actions = append(r.Rules[0].Actions, json.RawMessage(a))
	}
	return r
}

const toAdmission = `{"kind": "autoTriage", "target": "ready_for_admission"}`

var lowRisk = triageResult("II", "Low risk", toAdmission)

// statusEventRow is one status history row.
type statusEventRow struct {
	Move, From, To string
	Rule           *string
	User           *uuid.UUID
}

func statusEvents(t *testing.T, w *clinic, id uuid.UUID) []statusEventRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT move, coalesce(from_status, ''), to_status, rule, user_id
		FROM episode_status_events WHERE episode_id = $1 ORDER BY seq`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []statusEventRow
	for rows.Next() {
		var r statusEventRow
		if err := rows.Scan(&r.Move, &r.From, &r.To, &r.Rule, &r.User); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// triageState is the episode's auto-triage columns.
type triageState struct {
	Rule, Summary  *string
	At, CheckedAt  *time.Time
	CheckedBy      *uuid.UUID
	MessagePending bool
}

func readTriage(t *testing.T, w *clinic, id uuid.UUID) triageState {
	t.Helper()
	var s triageState
	if err := w.db.QueryRow(t.Context(), `SELECT triage_rule, triage_summary, triaged_at, triage_checked_at, triage_checked_by, triage_message_pending
		FROM episodes WHERE id = $1`, id).Scan(&s.Rule, &s.Summary, &s.At, &s.CheckedAt, &s.CheckedBy, &s.MessagePending); err != nil {
		t.Fatal(err)
	}
	return s
}

// onAutoTriaged reports whether the episode is on hospital A's Auto-triaged worklist.
func onAutoTriaged(t *testing.T, w *clinic, id uuid.UUID) bool {
	t.Helper()
	return slices.Contains(listIDs(t, w.clinician, at(w.a, "/episodes?worklist=auto_triaged")), id)
}

// notesOf are the General notes lines, each with "auto:" in front when written as automatic with no user.
func notesOf(t *testing.T, w *clinic, id uuid.UUID) []string {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT CASE WHEN automatic AND user_id IS NULL THEN 'auto:' ELSE '' END || text
		FROM episode_events WHERE episode_id = $1 ORDER BY created_at, seq`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func setVersionRules(t *testing.T, w *clinic, version uuid.UUID, rules string) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), `UPDATE questionnaire_versions SET rules = $2::jsonb WHERE id = $1`, version, rules); err != nil {
		t.Fatal(err)
	}
}

// sendForPatient sends a new episode's HQ as Cara, on the patient's behalf, with the evaluator answering res.
func sendForPatient(t *testing.T, w *clinic, res evaluate.Result) apitest.Episode {
	t.Helper()
	e := w.fx.Episode(t, w.hq, w.cara)
	w.anon.Evaluator.Reply(res)
	w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
	return e
}

func TestAutoTriageOnSend(t *testing.T) {
	t.Parallel()
	w := newClinic(t)

	t.Run("low risk, sent by the patient: Ready for admission at once, the rule named", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
		w.anon.Evaluator.Reply(lowRisk)
		p.Do("POST", patientPath(e, "/submit"), nil).Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForAdmission {
			t.Fatalf("status = %s", got.Status)
		}
		ev := statusEvents(t, w, e.ID)
		if len(ev) != 2 || ev[0] != (statusEventRow{Move: "submit", From: episode.HQNotComplete, To: episode.ReadyForReview}) ||
			ev[1].Move != "auto_triage" || ev[1].From != episode.ReadyForReview || ev[1].To != episode.ReadyForAdmission ||
			ev[1].Rule == nil || *ev[1].Rule != "Low risk" || ev[1].User != nil {
			t.Errorf("history = %+v", ev)
		}
		if n := notesOf(t, w, e.ID); !slices.Equal(n, []string{"HQ completed by the patient", "auto:" + lowRiskNote}) {
			t.Errorf("notes = %q", n)
		}
		s := readTriage(t, w, e.ID)
		if s.Rule == nil || *s.Rule != "Low risk" || s.Summary == nil || *s.Summary != "ASA II, no red or amber flags" || s.At == nil ||
			s.CheckedAt != nil || s.MessagePending {
			t.Errorf("triage = %+v", s)
		}
		if !onAutoTriaged(t, w, e.ID) || slices.Contains(listIDs(t, w.clinician, at(w.a, "/episodes?worklist=ready_for_review")), e.ID) {
			t.Error("want it on Auto-triaged, not Ready for review")
		}
		// The episode page gets the stamp's words.
		r := w.clinician.Do("GET", epPath(w.a, e, ""), nil)
		r.Want(t, http.StatusOK)
		var got struct {
			Episode struct {
				TriageRule, TriageSummary  *string
				TriagedAt, TriageCheckedAt *time.Time
				TriageMessagePending       bool
			}
		}
		r.Decode(t, &got)
		if g := got.Episode; g.TriageRule == nil || *g.TriageRule != "Low risk" || g.TriageSummary == nil || g.TriagedAt == nil || g.TriageCheckedAt != nil {
			t.Errorf("episode = %+v", g)
		}
	})

	t.Run("staff sending on the patient's behalf: auto-triaged the same, still automatic", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForAdmission {
			t.Fatalf("status = %s", got.Status)
		}
		ev := statusEvents(t, w, e.ID)
		if len(ev) != 2 || ev[0].Move != "submit" || ev[0].User == nil || *ev[0].User != w.cara.ID ||
			ev[1].Move != "auto_triage" || ev[1].User != nil || ev[1].Rule == nil || *ev[1].Rule != "Low risk" {
			t.Errorf("history = %+v", ev)
		}
		if n := notesOf(t, w, e.ID); !slices.Equal(n, []string{"HQ completed on the patient's behalf", "auto:" + lowRiskNote}) {
			t.Errorf("notes = %q", n)
		}
		if !onAutoTriaged(t, w, e.ID) {
			t.Error("not on Auto-triaged")
		}
	})

	t.Run("the rule's target: Ready for POA, POA complete", func(t *testing.T) {
		for target, label := range map[string]string{episode.ReadyForPOA: "Ready for POA", episode.POAComplete: "POA complete"} {
			e := sendForPatient(t, w, triageResult("I", "Day case", `{"kind": "autoTriage", "target": "`+target+`"}`))
			if got := readEpisode(t, w, e.ID); got.Status != target {
				t.Errorf("status = %s, want %s", got.Status, target)
			}
			if n := lastNote(t, w, e.ID); n != "Auto-triaged by the rule “Day case”: ASA I, no red or amber flags → "+label {
				t.Errorf("note = %q", n)
			}
		}
	})

	t.Run("the evaluator failing: Ready for review, no auto-triage", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		w.anon.Evaluator.Reply(lowRisk)
		w.anon.Evaluator.Fail(errors.New("evaluate: /evaluate: connection refused"))
		w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		w.anon.Evaluator.Fail(nil)
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForReview {
			t.Errorf("status = %s", got.Status)
		}
		for _, ev := range statusEvents(t, w, e.ID) {
			if ev.Move == "auto_triage" {
				t.Errorf("auto-triaged: %+v", ev)
			}
		}
		if n := notesOf(t, w, e.ID); !slices.Equal(n, []string{"HQ completed on the patient's behalf", "auto:" + checksDidntRun}) {
			t.Errorf("notes = %q", n)
		}
		if s := readTriage(t, w, e.ID); s.Rule != nil || onAutoTriaged(t, w, e.ID) {
			t.Errorf("triage = %+v", s)
		}
	})

	t.Run("no rule auto-triages: Ready for review, nothing more", func(t *testing.T) {
		e := sendForPatient(t, w, triageResult("III", "High risk", `{"kind": "task", "title": "Book anaesthetic review", "team": "anaesthetist"}`))
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForReview {
			t.Errorf("status = %s", got.Status)
		}
		// S43: the rule's task is added, with its line; no auto-triage line.
		if n := notesOf(t, w, e.ID); !slices.Equal(n, []string{"HQ completed on the patient's behalf",
			"auto:Task added by the rule “High risk” for the anaesthetist team: Book anaesthetic review"}) {
			t.Errorf("notes = %q", n)
		}
		if len(statusEvents(t, w, e.ID)) != 1 || onAutoTriaged(t, w, e.ID) {
			t.Error("want only the submit, and not on Auto-triaged")
		}
	})

	t.Run("a guard stops it: Ready for review, with a note saying why", func(t *testing.T) {
		amber := triageResult("II", "Low risk", toAdmission)
		amber.Flags = []evaluate.Flag{{Level: "amber", Text: "Are you tired in the day?: Yes"}}
		incomplete := triageResult("II", "Low risk", toAdmission)
		incomplete.Sets = []evaluate.SetResult{{Scores: []evaluate.Score{{QuestionID: "q_stopbang", Incomplete: true}}}}
		required := triageResult("II", "Low risk", toAdmission, `{"kind": "requireAnaesthetistAsa"}`)
		// As the evaluate service sends it since test report 1 (E-2): the rule held, with the reason.
		held := triageResult("II", "Low risk")
		held.Rules[0].Held, held.Rules[0].HeldReason = true, "flag"
		held.Flags = []evaluate.Flag{{Level: "amber", Text: "Are you tired in the day?: Yes"}}
		for res, why := range map[*evaluate.Result]string{
			&amber: "the answers have a red or amber flag", &incomplete: "a score is Incomplete",
			&required: "a rule requires the anaesthetist's ASA grade first", &held: "the answers have a red or amber flag",
		} {
			e := sendForPatient(t, w, *res)
			if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForReview {
				t.Errorf("%s: status = %s", why, got.Status)
			}
			want := []string{"HQ completed on the patient's behalf", "auto:Not auto-triaged by the rule “Low risk”: " + why + ". Review this HQ by hand."}
			if res == &required { // S43: the requirement is recorded first, with its own line
				want = slices.Insert(want, 1, "auto:The rule “Low risk” requires the anaesthetist's ASA grade before Ready for admission")
			}
			if n := notesOf(t, w, e.ID); !slices.Equal(n, want) {
				t.Errorf("notes = %q, want %q", n, want)
			}
			if onAutoTriaged(t, w, e.ID) {
				t.Errorf("%s: on Auto-triaged", why)
			}
		}
	})

	t.Run("on the worklist's count; a cancelled or archived one is in none", func(t *testing.T) {
		before := tabCount(t, w, "auto_triaged")
		cancelled, archived := sendForPatient(t, w, lowRisk), sendForPatient(t, w, lowRisk)
		if n := tabCount(t, w, "auto_triaged"); n != before+2 {
			t.Errorf("count = %d, want %d", n, before+2)
		}
		w.clinician.Do("POST", epPath(w.a, cancelled, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, cancelled.ID)}).
			Want(t, http.StatusNoContent)
		w.author.Do("POST", epPath(w.a, archived, "/archive"), map[string]any{"revision": revisionOf(t, w, archived.ID)}).Want(t, http.StatusNoContent)
		if onAutoTriaged(t, w, cancelled.ID) || onAutoTriaged(t, w, archived.ID) || tabCount(t, w, "auto_triaged") != before {
			t.Error("a cancelled or archived episode is still on Auto-triaged")
		}
	})
}

// tabCount is hospital A's tab count for one worklist.
func tabCount(t *testing.T, w *clinic, key string) int {
	t.Helper()
	for _, l := range counts(t, w.clinician, at(w.a, "/worklists")) {
		if l.Key == key {
			return l.Count
		}
	}
	t.Fatalf("no worklist %q", key)
	return 0
}

// The patient's message (EPI-08) on auto-triage (S42 answers 2): at once by default; held for
// Checked when the HQ says so.
func TestAutoTriageMessages(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	msgs := w.anon.Messages
	setHospitalSettings(t, w, w.a, `{"statusMessages.readyForAdmissionSms": true, "templates.readyForAdmissionSms": "{{hospital}}: {{firstName}}, all done."}`)
	send := func(t *testing.T, last, mobile string) apitest.Episode {
		t.Helper()
		e := statusMsgEpisode(t, w, last, mobile, "")
		setStatus(t, w, e, episode.HQNotComplete)
		w.anon.Evaluator.Reply(lowRisk)
		w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
		return e
	}
	name := hospitalName(t, w, w.a)

	t.Run("by default the message goes at once, noted as automatic", func(t *testing.T) {
		e := send(t, "Quick", "07700 900501")
		if sms := msgs.To("+447700900501"); len(sms) != 2 || sms[1].Body != name+": Sam, all done." {
			t.Errorf("texts = %+v", sms)
		}
		if n := notesOf(t, w, e.ID); n[len(n)-1] != "auto:Ready for admission message sent by text message." {
			t.Errorf("notes = %q", n)
		}
		if s := readTriage(t, w, e.ID); s.MessagePending {
			t.Error("still pending")
		}
	})

	setVersionRules(t, w, w.hq.VersionID, `{"noGradeAutoTriage": true, "messageAfterCheck": true, "rules": [], "testCases": []}`)
	t.Run("the HQ says after Checked: nothing goes until a clinician checks it", func(t *testing.T) {
		e := send(t, "Held", "07700 900502")
		if sms := msgs.To("+447700900502"); len(sms) != 1 { // the invite only
			t.Errorf("texts = %+v", sms)
		}
		if s := readTriage(t, w, e.ID); !s.MessagePending {
			t.Error("not pending")
		}
		w.clinician.Do("POST", epPath(w.a, e, "/triage-checked"), map[string]any{"revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
		if sms := msgs.To("+447700900502"); len(sms) != 2 || sms[1].Body != name+": Sam, all done." {
			t.Errorf("texts after Checked = %+v", sms)
		}
		if n := lastNote(t, w, e.ID); n != "Ready for admission message sent by text message." {
			t.Errorf("note = %q", n)
		}
		if s := readTriage(t, w, e.ID); s.MessagePending {
			t.Error("still pending after Checked")
		}
	})
	t.Run("moved on by hand before Checked: the message no longer fits, so none goes", func(t *testing.T) {
		e := send(t, "Moved", "07700 900503")
		move(t, w, e, episode.OnHold)
		w.clinician.Do("POST", epPath(w.a, e, "/triage-checked"), map[string]any{"revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
		if sms := msgs.To("+447700900503"); len(sms) != 1 {
			t.Errorf("texts = %+v", sms)
		}
	})
	// S42 verification (fixed in S44a): a status change by hand ends the wait, so the message goes once.
	t.Run("On hold, then Ready for admission by hand (which sends it), then Checked: it goes once", func(t *testing.T) {
		e := send(t, "Twice", "07700 900505")
		move(t, w, e, episode.OnHold)
		if s := readTriage(t, w, e.ID); s.MessagePending {
			t.Error("still pending after a change by hand")
		}
		move(t, w, e, episode.ReadyForAdmission)
		if sms := msgs.To("+447700900505"); len(sms) != 2 {
			t.Errorf("texts after Ready for admission = %+v", sms)
		}
		w.clinician.Do("POST", epPath(w.a, e, "/triage-checked"), map[string]any{"revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
		if sms := msgs.To("+447700900505"); len(sms) != 2 {
			t.Errorf("texts after Checked = %+v, want the Ready for admission message once", sms)
		}
	})
	t.Run("re-opened before Checked: none goes, ever", func(t *testing.T) {
		e := send(t, "Reopened", "07700 900504")
		w.author.Do("POST", epPath(w.a, e, "/reopen"), map[string]any{"reason": "Look again", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		if s := readTriage(t, w, e.ID); s.MessagePending || s.Rule != nil {
			t.Errorf("triage = %+v", s)
		}
		if sms := msgs.To("+447700900504"); len(sms) != 1 {
			t.Errorf("texts = %+v", sms)
		}
	})
}

// POST …/episodes/{eid}/triage-checked {revision} (S42 answers 1): any clinician at the hospital
// marks an auto-triaged episode checked; it leaves the Auto-triaged worklist. Audited.
func TestTriageChecked(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	check := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body any) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/triage-checked"), body)
	}
	rev := func(e apitest.Episode) map[string]any { return map[string]any{"revision": revisionOf(t, w, e.ID)} }

	t.Run("happy path: who and when, a note, off the list, audited", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		before := revisionOf(t, w, e.ID)
		a := oneAudit(t, w, "triage.checked", func() *apitest.Resp { return check(w.clinician, w.a, e, rev(e)) }, http.StatusNoContent)
		if !slices.Contains(a.Episodes, e.ID) {
			t.Errorf("audit = %+v", a)
		}
		s := readTriage(t, w, e.ID)
		if s.CheckedBy == nil || *s.CheckedBy != w.cara.ID || s.CheckedAt == nil || s.Rule == nil {
			t.Errorf("triage = %+v", s)
		}
		if revisionOf(t, w, e.ID) != before+1 {
			t.Error("the revision didn't move on")
		}
		ev := readEvents(t, w, e.ID)
		if last := ev[len(ev)-1]; last.Text != "Auto-triage checked" || last.User == nil || *last.User != w.cara.ID {
			t.Errorf("note = %+v", last)
		}
		if onAutoTriaged(t, w, e.ID) {
			t.Error("still on Auto-triaged")
		}
		if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForAdmission {
			t.Errorf("status = %s: Checked changes no status", got.Status)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		check(w.anon, w.a, e, rev(e)).Want(t, http.StatusUnauthorized)
		if readTriage(t, w, e.ID).CheckedAt != nil {
			t.Error("checked")
		}
	})
	t.Run("the role: a clinician at the trust's other hospital can't check this one (404)", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		refused(t, check(w.site2, w.a, e, rev(e)), http.StatusNotFound)
		refused(t, check(w.site2, w.a2, e, rev(e)), http.StatusNotFound)
		if readTriage(t, w, e.ID).CheckedAt != nil {
			t.Error("checked")
		}
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		refused(t, check(w.otherClin, w.a, e, rev(e)), http.StatusNotFound)
		refused(t, check(w.otherClin, w.b, e, rev(e)), http.StatusNotFound)
		if readTriage(t, w, e.ID).CheckedAt != nil {
			t.Error("checked")
		}
	})
	t.Run("invalid input: no revision, or not JSON", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		refused(t, check(w.clinician, w.a, e, map[string]any{}), http.StatusBadRequest)
		refused(t, check(w.clinician, w.a, e, map[string]any{"revision": -1}), http.StatusBadRequest)
		refused(t, check(w.clinician, w.a, e, "nonsense"), http.StatusBadRequest)
		if readTriage(t, w, e.ID).CheckedAt != nil {
			t.Error("checked")
		}
	})
	t.Run("a stale revision: 409 stale", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		r := check(w.clinician, w.a, e, map[string]any{"revision": revisionOf(t, w, e.ID) - 1})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.String(), `"stale"`) || readTriage(t, w, e.ID).CheckedAt != nil {
			t.Errorf("response = %s", r)
		}
	})
	// S42 test report 2 (S44a): a tab opened before someone else checked it says so, and reloads (stale).
	t.Run("someone checked it while it was open: 409 saying so", func(t *testing.T) {
		e := sendForPatient(t, w, lowRisk)
		opened := rev(e)
		check(w.author, w.a, e, rev(e)).Want(t, http.StatusNoContent)
		for _, body := range []map[string]any{opened, rev(e)} { // the stale tab, and a second press with the new revision
			r := check(w.clinician, w.a, e, body)
			refused(t, r, http.StatusConflict)
			if want := "Someone checked this while you had it open."; r.Message() != want || !strings.Contains(r.String(), `"stale"`) {
				t.Errorf("refusal = %s, want %q (stale, so the page reloads)", r, want)
			}
		}
	})
	t.Run("only an auto-triaged episode not yet checked: 409 otherwise", func(t *testing.T) {
		checked := sendForPatient(t, w, lowRisk)
		check(w.clinician, w.a, checked, rev(checked)).Want(t, http.StatusNoContent)
		refused(t, check(w.clinician, w.a, checked, rev(checked)), http.StatusConflict) // twice
		reviewed := sendForPatient(t, w, triageResult("III", "High risk", `{"kind": "flag", "name": "Look"}`))
		refused(t, check(w.clinician, w.a, reviewed, rev(reviewed)), http.StatusConflict) // never auto-triaged
		refused(t, check(w.clinician, w.a, w.ep, rev(w.ep)), http.StatusConflict)         // not sent yet
		cancelled := sendForPatient(t, w, lowRisk)
		w.clinician.Do("POST", epPath(w.a, cancelled, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, cancelled.ID)}).
			Want(t, http.StatusNoContent)
		refused(t, check(w.clinician, w.a, cancelled, rev(cancelled)), http.StatusConflict)
		archived := sendForPatient(t, w, lowRisk)
		w.author.Do("POST", epPath(w.a, archived, "/archive"), map[string]any{"revision": revisionOf(t, w, archived.ID)}).Want(t, http.StatusNoContent)
		refused(t, check(w.clinician, w.a, archived, rev(archived)), http.StatusConflict)
		for _, e := range []apitest.Episode{reviewed, cancelled, archived} {
			if readTriage(t, w, e.ID).CheckedAt != nil {
				t.Errorf("%s checked", e.ID)
			}
		}
	})
}

// Re-open (EPI-07) works for an auto-triaged episode: a super clinician, with a reason, back to
// Ready for review; the auto-triage ends (off the list, no stamp) and the sets can be validated.
func TestReopenAutoTriaged(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := sendForPatient(t, w, lowRisk)
	refused(t, w.clinician.Do("POST", epPath(w.a, e, "/reopen"), map[string]any{"reason": "Look again", "revision": revisionOf(t, w, e.ID)}),
		http.StatusForbidden)
	refused(t, w.author.Do("POST", epPath(w.a, e, "/reopen"), map[string]any{"revision": revisionOf(t, w, e.ID)}), http.StatusBadRequest)
	w.author.Do("POST", epPath(w.a, e, "/reopen"), map[string]any{"reason": "Look again", "revision": revisionOf(t, w, e.ID)}).
		Want(t, http.StatusNoContent)
	if got := readEpisode(t, w, e.ID); got.Status != episode.ReadyForReview {
		t.Fatalf("status = %s", got.Status)
	}
	ev := statusEvents(t, w, e.ID)
	if last := ev[len(ev)-1]; last.Move != "reopen" || last.From != episode.ReadyForAdmission || last.To != episode.ReadyForReview {
		t.Errorf("history = %+v", ev)
	}
	if s := readTriage(t, w, e.ID); s.Rule != nil || s.Summary != nil || s.At != nil {
		t.Errorf("triage = %+v", s)
	}
	if onAutoTriaged(t, w, e.ID) {
		t.Error("still on Auto-triaged")
	}
	refused(t, w.clinician.Do("POST", epPath(w.a, e, "/triage-checked"), map[string]any{"revision": revisionOf(t, w, e.ID)}), http.StatusConflict)
	// Validate works again: the clinician's copy saves while it's Ready for review.
	savedRevision(t, w.clinician.Do("PUT", epPath(w.a, e, "/answers/"+w.hq.Chapters[0].String()),
		map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "validated": true, "revision": 0}))
}
