package triage

import (
	"encoding/json"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

func fired(name string, actions ...string) evaluate.FiredRule {
	r := evaluate.FiredRule{ID: name, Name: name}
	for _, a := range actions {
		r.Actions = append(r.Actions, json.RawMessage(a))
	}
	return r
}

const toAdmission = `{"kind": "autoTriage", "target": "ready_for_admission"}`

func result(asa string, rules ...evaluate.FiredRule) evaluate.Result {
	r := evaluate.Result{Evaluator: "test", Flags: []evaluate.Flag{}, Rules: rules}
	if asa != "" {
		r.SuggestedASA = &evaluate.SuggestedASA{Grade: asa}
	}
	return r
}

// Which rule auto-triages, where to, and the words for the stamp and the note (S42).
func TestDecide(t *testing.T) {
	for _, tt := range []struct {
		name    string
		res     evaluate.Result
		want    Decision
		blocked string // the guard that stopped it, if one did
	}{
		{name: "no rule fires: nothing", res: result("II")},
		{name: "rules fire, none auto-triages: nothing", res: result("III", fired("High risk", `{"kind": "task", "title": "Book", "team": "anaesthetist"}`))},
		{name: "low risk: Ready for admission, named", res: result("II", fired("Low risk", toAdmission)),
			want: Decision{Rule: "Low risk", Target: "ready_for_admission", Summary: "ASA II, no red or amber flags"}},
		{name: "no ASA grade (counted as ASA I by the rules)", res: result("", fired("Low risk", toAdmission)),
			want: Decision{Rule: "Low risk", Target: "ready_for_admission", Summary: "no ASA grade, no red or amber flags"}},
		{name: "an emergency grade keeps its E", res: func() evaluate.Result {
			r := result("I", fired("Low risk", toAdmission))
			r.SuggestedASA.Emergency = true
			return r
		}(), want: Decision{Rule: "Low risk", Target: "ready_for_admission", Summary: "ASA IE, no red or amber flags"}},
		{name: "the first rule that auto-triages wins", res: result("I",
			fired("Flag only", `{"kind": "flag", "name": "Look"}`),
			fired("Day case", `{"kind": "autoTriage", "target": "ready_for_poa"}`),
			fired("Low risk", toAdmission)),
			want: Decision{Rule: "Day case", Target: "ready_for_poa", Summary: "ASA I, no red or amber flags"}},
		{name: "POA complete", res: result("II", fired("Quick", `{"kind": "autoTriage", "target": "poa_complete"}`)),
			want: Decision{Rule: "Quick", Target: "poa_complete", Summary: "ASA II, no red or amber flags"}},
		{name: "a held rule (no grade, the HQ's setting off) has no auto-triage left: nothing",
			res: func() evaluate.Result { r := result("", fired("Low risk")); r.Rules[0].Held = true; return r }()},
		{name: "a target that isn't one of the rules' targets: nothing", res: result("II", fired("Odd", `{"kind": "autoTriage", "target": "admitted"}`))},
		// The guards (S42 answers 2): they can't be switched off.
		{name: "a red flag stops it", res: func() evaluate.Result {
			r := result("II", fired("Low risk", toAdmission))
			r.Flags = []evaluate.Flag{{Level: "red", Text: "Chest pain?: Yes"}}
			return r
		}(), want: Decision{Rule: "Low risk"}, blocked: "the answers have a red or amber flag"},
		{name: "an amber flag stops it", res: func() evaluate.Result {
			r := result("II", fired("Low risk", toAdmission))
			r.Flags = []evaluate.Flag{{Level: "amber", Text: "Tired?: Yes"}}
			return r
		}(), want: Decision{Rule: "Low risk"}, blocked: "the answers have a red or amber flag"},
		{name: "an Incomplete score stops it", res: func() evaluate.Result {
			r := result("II", fired("Low risk", toAdmission))
			r.Sets = []evaluate.SetResult{{Scores: []evaluate.Score{{QuestionID: "q_a", Incomplete: false}}}, {Scores: []evaluate.Score{{QuestionID: "q_stopbang", Incomplete: true}}}}
			return r
		}(), want: Decision{Rule: "Low risk"}, blocked: "a score is Incomplete"},
		{name: "the anaesthetist's ASA required (S43) stops Ready for admission", res: result("II",
			fired("Low risk", toAdmission), fired("Diabetes", `{"kind": "requireAnaesthetistAsa"}`)),
			want: Decision{Rule: "Low risk"}, blocked: "a rule requires the anaesthetist's ASA grade first"},
		{name: "but not Ready for POA", res: result("II",
			fired("Day case", `{"kind": "autoTriage", "target": "ready_for_poa"}`), fired("Diabetes", `{"kind": "requireAnaesthetistAsa"}`)),
			want: Decision{Rule: "Day case", Target: "ready_for_poa", Summary: "ASA II, no red or amber flags"}},
		// The evaluator applies the same guards (packages/clinical firedRules, test report 1, E-2): a
		// rule held for one has no auto-triage left, and its reason names the guard.
		{name: "held by the evaluator for a flag", res: func() evaluate.Result {
			r := result("II", fired("Low risk"))
			r.Rules[0].Held, r.Rules[0].HeldReason = true, "flag"
			return r
		}(), want: Decision{Rule: "Low risk"}, blocked: "the answers have a red or amber flag"},
		{name: "held by the evaluator for an Incomplete score", res: func() evaluate.Result {
			r := result("II", fired("Low risk"))
			r.Rules[0].Held, r.Rules[0].HeldReason = true, "incomplete"
			return r
		}(), want: Decision{Rule: "Low risk"}, blocked: "a score is Incomplete"},
		{name: "held by the evaluator for the anaesthetist's ASA", res: func() evaluate.Result {
			r := result("II", fired("Low risk"), fired("Diabetes", `{"kind": "requireAnaesthetistAsa"}`))
			r.Rules[0].Held, r.Rules[0].HeldReason = true, "anaesthetistAsa"
			return r
		}(), want: Decision{Rule: "Low risk"}, blocked: "a rule requires the anaesthetist's ASA grade first"},
		{name: "a held rule for Ready for admission, then one for Ready for POA: the second moves it, as the Preview says", res: func() evaluate.Result {
			r := result("II", fired("Low risk"), fired("Day case", `{"kind": "autoTriage", "target": "ready_for_poa"}`), fired("Diabetes", `{"kind": "requireAnaesthetistAsa"}`))
			r.Rules[0].Held, r.Rules[0].HeldReason = true, "anaesthetistAsa"
			return r
		}(), want: Decision{Rule: "Day case", Target: "ready_for_poa", Summary: "ASA II, no red or amber flags"}},
		{name: "held for no grade (the HQ's setting): nothing, as before", res: func() evaluate.Result {
			r := result("", fired("Low risk"))
			r.Rules[0].Held, r.Rules[0].HeldReason = true, "noGrade"
			return r
		}()},
		{name: "an action that can't be read is skipped", res: result("II", fired("Broken", `not json`), fired("Low risk", toAdmission)),
			want: Decision{Rule: "Low risk", Target: "ready_for_admission", Summary: "ASA II, no red or amber flags"}},
	} {
		got := Decide(tt.res)
		tt.want.Blocked = tt.blocked
		if got != tt.want {
			t.Errorf("%s: Decide = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestDecisionWords(t *testing.T) {
	d := Decision{Rule: "Low risk", Target: "ready_for_admission", Summary: "ASA II, no red or amber flags"}
	if got := d.Note(); got != "Auto-triaged by the rule “Low risk”: ASA II, no red or amber flags → Ready for admission" {
		t.Errorf("note = %q", got)
	}
	b := Decision{Rule: "Low risk", Blocked: "a score is Incomplete"}
	if got := b.Note(); got != "Not auto-triaged by the rule “Low risk”: a score is Incomplete. Review this HQ by hand." {
		t.Errorf("blocked note = %q", got)
	}
	if !d.Moves() || b.Moves() || (Decision{}).Moves() {
		t.Error("only a decision with a target moves the episode")
	}
}
