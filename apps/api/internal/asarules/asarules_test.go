package asarules

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDefaultsPassTheirOwnCheck(t *testing.T) {
	set, problem := Check(Defaults())
	if problem != "" {
		t.Fatalf("the defaults are refused: %s", problem)
	}
	if !set.NoGradeAutoTriage || len(set.Rules) != 2 {
		t.Fatalf("defaults = %+v, want the two rules and no-grade auto-triage on", set)
	}
	low, high := set.Rules[0], set.Rules[1]
	if low.When.ASA.Op != "<=" || low.When.ASA.Grade != 2 || !low.When.NoRedFlags || low.Then[0].Kind != AutoTriage || low.Then[0].Target != "ready_for_admission" {
		t.Errorf("low risk = %+v", low)
	}
	// S44c (Rahul, 28 Oct): a High STOP-Bang holds back the Low risk auto-triage.
	if nb := low.When.NotBand; nb == nil || nb.Score != "stopbang_score" || len(nb.Bands) != 1 || nb.Bands[0] != "b_high" {
		t.Errorf("low risk's notBand = %+v, want STOP-Bang isn't High risk", nb)
	}
	if high.When.ASA.Op != ">=" || high.When.ASA.Grade != 3 || len(high.Then) != 3 ||
		high.Then[0].Title != "Book anaesthetic review" || high.Then[0].Team != "anaesthetist" || high.Then[2].Kind != RequireAnaesthetistASA {
		t.Errorf("high risk = %+v", high)
	}
}

// A copy of the defaults with one change, as JSON.
func variant(t *testing.T, change func(m map[string]any)) json.RawMessage {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(Defaults(), &m); err != nil {
		t.Fatal(err)
	}
	change(m)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func rule(m map[string]any, i int) map[string]any { return m["rules"].([]any)[i].(map[string]any) }
func when(m map[string]any, i int) map[string]any { return rule(m, i)["when"].(map[string]any) }
func action(m map[string]any, i, j int) map[string]any {
	return rule(m, i)["then"].([]any)[j].(map[string]any)
}

func TestCheckRefusesABadShapeInPlainWords(t *testing.T) {
	long := strings.Repeat("x", 201)
	for _, c := range []struct {
		name   string
		raw    json.RawMessage
		saying string
	}{
		{"not JSON", json.RawMessage(`{"rules": [`), "could not be read"},
		{"an unknown field", variant(t, func(m map[string]any) { m["extra"] = 1 }), "could not be read"},
		{"no rules list", json.RawMessage(`{"noGradeAutoTriage": true}`), "could not be read"},
		{"a rule with no name", variant(t, func(m map[string]any) { rule(m, 0)["name"] = "  " }), "Give each rule a name."},
		{"two rules with one name", variant(t, func(m map[string]any) { rule(m, 1)["name"] = "low RISK" }), "Two rules are called"},
		{"a bad id", variant(t, func(m map[string]any) { rule(m, 0)["id"] = "a b" }), "could not be read"},
		{"two rules with one id", variant(t, func(m map[string]any) { rule(m, 1)["id"] = "low_risk" }), "could not be read"},
		{"no ASA comparison", variant(t, func(m map[string]any) { delete(when(m, 0), "asa") }), "how the suggested ASA is compared"},
		{"an unknown comparison", variant(t, func(m map[string]any) { when(m, 0)["asa"] = map[string]any{"op": "<", "grade": 2} }), "how the suggested ASA is compared"},
		{"a grade past VI", variant(t, func(m map[string]any) { when(m, 0)["asa"] = map[string]any{"op": "<=", "grade": 7} }), "I to VI"},
		{"no actions", variant(t, func(m map[string]any) { rule(m, 0)["then"] = []any{} }), "add at least one action"},
		{"an unknown action", variant(t, func(m map[string]any) { action(m, 0, 0)["kind"] = "email" }), "isn't one Preopsis knows"},
		{"auto-triage to Admitted", variant(t, func(m map[string]any) { action(m, 0, 0)["target"] = "admitted" }), "Ready for admission, Ready for POA or POA complete"},
		{"auto-triage twice", variant(t, func(m map[string]any) {
			rule(m, 0)["then"] = []any{action(m, 0, 0), map[string]any{"kind": "autoTriage", "target": "ready_for_poa"}}
		}), "auto-triage only once"},
		{"a task with no title", variant(t, func(m map[string]any) { action(m, 1, 0)["title"] = " " }), "Rule “High risk”: Enter the task."},
		{"a task too long", variant(t, func(m map[string]any) { action(m, 1, 0)["title"] = long }), "too long"},
		{"a task for no team", variant(t, func(m map[string]any) { action(m, 1, 0)["team"] = "porters" }), "Choose the team"},
		{"a flag with no name", variant(t, func(m map[string]any) { action(m, 1, 1)["name"] = "" }), "name the flag"},
		{"a suggested status not after review", variant(t, func(m map[string]any) {
			rule(m, 0)["then"] = []any{map[string]any{"kind": "suggestStatus", "status": "cancelled"}}
		}), "can suggest"},
		{"a band with no bands", variant(t, func(m map[string]any) { when(m, 0)["band"] = map[string]any{"score": "q_bmi", "bands": []any{}} }), "the score and at least one of its bands"},
		{"not in a band, with no bands", variant(t, func(m map[string]any) {
			when(m, 0)["notBand"] = map[string]any{"score": "stopbang_score", "bands": []any{}}
		}), "the score and at least one of its bands"},
		{"not in a band, with no score", variant(t, func(m map[string]any) { when(m, 0)["notBand"] = map[string]any{"score": " ", "bands": []any{"b_high"}} }), "the score and at least one of its bands"},
		{"a code with no code", variant(t, func(m map[string]any) {
			when(m, 0)["code"] = map[string]any{"set": "ICD10", "code": "", "display": "x"}
		}), "choose the code"},
		{"no procedures", variant(t, func(m map[string]any) { when(m, 0)["procedures"] = []any{} }), "at least one procedure"},
		{"a procedure that isn't an id", variant(t, func(m map[string]any) { when(m, 0)["procedures"] = []any{"hip"} }), "could not be read"},
		{"a test case with no name", variant(t, func(m map[string]any) {
			m["testCases"] = []any{map[string]any{"id": "t1", "name": "", "patient": map[string]any{"name": "Sam"}, "answers": map[string]any{}, "expected": []any{}}}
		}), "Give each test case a name."},
		{"two test cases with one name", variant(t, func(m map[string]any) {
			tc := func(id string) map[string]any {
				return map[string]any{"id": id, "name": "Heart", "patient": map[string]any{"name": "Sam"}, "answers": map[string]any{}, "expected": []any{}}
			}
			m["testCases"] = []any{tc("t1"), tc("t2")}
		}), "Two test cases are called “Heart”"},
		{"a test case aged 200", variant(t, func(m map[string]any) {
			m["testCases"] = []any{map[string]any{"id": "t1", "name": "Old", "patient": map[string]any{"name": "Sam", "age": 200}, "answers": map[string]any{}, "expected": []any{}}}
		}), "age"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, problem := Check(c.raw)
			if !strings.Contains(problem, c.saying) {
				t.Errorf("problem = %q, want it to say %q", problem, c.saying)
			}
		})
	}
}

func TestCheckKeepsEveryKindOfWhenAndAction(t *testing.T) {
	proc := uuid.New()
	raw := variant(t, func(m map[string]any) {
		w := when(m, 0)
		w["band"] = map[string]any{"score": "q_stopbang", "bands": []any{"b_high"}}
		w["notBand"] = map[string]any{"score": "q_other", "bands": []any{"b_low"}}
		w["code"] = map[string]any{"set": "ICD10", "code": "I50.9", "display": "Heart failure"}
		w["procedures"] = []any{proc.String()}
		rule(m, 0)["then"] = []any{
			map[string]any{"kind": "autoTriage", "target": "ready_for_poa"},
			map[string]any{"kind": "suggestStatus", "status": "on_hold"},
			map[string]any{"kind": "flag", "name": "  Sleep apnoea "},
			map[string]any{"kind": "task", "title": " Call the patient ", "team": "nurse"},
		}
		m["noGradeAutoTriage"] = false
		m["testCases"] = []any{map[string]any{"id": "t1", "name": "Fit", "patient": map[string]any{"name": "Sam", "age": 40, "sex": "female"},
			"procedureId": proc.String(), "answers": map[string]any{"q_smokes": "no"}, "expected": []any{"No suggested ASA"}}}
	})
	set, problem := Check(raw)
	if problem != "" {
		t.Fatal(problem)
	}
	r := set.Rules[0]
	if r.When.Band.Score != "q_stopbang" || r.When.NotBand == nil || r.When.NotBand.Score != "q_other" || r.When.Code.Code != "I50.9" || r.When.Procedures[0] != proc {
		t.Errorf("when = %+v", r.When)
	}
	if r.Then[2].Name != "Sleep apnoea" || r.Then[3].Title != "Call the patient" {
		t.Errorf("names and titles are stored trimmed: %+v", r.Then)
	}
	if set.NoGradeAutoTriage {
		t.Error("the no-grade setting was lost")
	}
	if ids := set.ProcedureIDs(); len(ids) != 1 || ids[0] != proc {
		t.Errorf("procedure ids = %v, want the rule's and the test case's one, once", ids)
	}
	// What's stored is the checked set, trimmed.
	var back RuleSet
	if err := json.Unmarshal(set.JSON(), &back); err != nil || back.Rules[0].Then[2].Name != "Sleep apnoea" {
		t.Errorf("stored JSON = %s (%v)", set.JSON(), err)
	}
}

func TestCheckLimitsTheNumberOfRules(t *testing.T) {
	raw := variant(t, func(m map[string]any) {
		var rules []any
		for i := range MaxRules + 1 {
			r := map[string]any{"id": "r" + strings.Repeat("x", i%3) + string(rune('a'+i%26)) + string(rune('a'+i/26)), "name": "Rule " + string(rune('a'+i%26)) + string(rune('a'+i/26)),
				"when": map[string]any{"asa": map[string]any{"op": "=", "grade": 1}}, "then": []any{map[string]any{"kind": "requireAnaesthetistAsa"}}}
			rules = append(rules, r)
		}
		m["rules"] = rules
	})
	if _, problem := Check(raw); !strings.Contains(problem, "at most") {
		t.Errorf("problem = %q", problem)
	}
}

// S42 (answers 2): each HQ says when an auto-triaged patient's status message goes. Off by default
// (straight away); on, it waits for a clinician's Checked. A set stored before S42 has no such
// field and counts as off; anything but true or false is refused.
func TestCheckKeepsWhenTheMessageGoes(t *testing.T) {
	set, _ := Check(Defaults())
	if set.MessageAfterCheck {
		t.Error("the defaults send the message straight away")
	}
	if !strings.Contains(string(Defaults()), `"messageAfterCheck": false`) {
		t.Error("the defaults name the setting")
	}
	old := variant(t, func(m map[string]any) { delete(m, "messageAfterCheck") })
	if set, problem := Check(old); problem != "" || set.MessageAfterCheck {
		t.Errorf("a set from before S42 = %+v, %q; want accepted, off", set, problem)
	}
	on := variant(t, func(m map[string]any) { m["messageAfterCheck"] = true })
	set, problem := Check(on)
	if problem != "" || !set.MessageAfterCheck {
		t.Fatalf("on = %+v, %q", set, problem)
	}
	if !strings.Contains(string(set.JSON()), `"messageAfterCheck":true`) {
		t.Errorf("stored = %s", set.JSON())
	}
	if _, problem := Check(variant(t, func(m map[string]any) { m["messageAfterCheck"] = "yes" })); problem == "" {
		t.Error("a setting that isn't true or false was accepted")
	}
}
