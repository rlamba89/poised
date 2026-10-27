package triage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

// The rules' other actions (S43; ASA-03, TSK-01): a sent HQ's result turned into the writes Go makes.
func TestActions(t *testing.T) {
	const (
		task    = `{"kind": "task", "title": "Book anaesthetic review", "team": "anaesthetist"}`
		flag    = `{"kind": "flag", "name": "Anaesthetic review"}`
		require = `{"kind": "requireAnaesthetistAsa"}`
	)
	for _, tt := range []struct {
		name string
		res  evaluate.Result
		want Actions
	}{
		{name: "no rule fires: nothing", res: result("II")},
		{name: "auto-triage alone: nothing here (Decide's)", res: result("II", fired("Low risk", toAdmission))},
		{name: "high risk: the task, the flag and the requirement, each naming the rule",
			res: result("III", fired("High risk", task, flag, require)),
			want: Actions{
				Tasks:      []RuleTask{{Rule: "High risk", Title: "Book anaesthetic review", Team: "anaesthetist"}},
				Flags:      []RuleFlag{{Rule: "High risk", Name: "Anaesthetic review"}},
				RequireASA: "High risk",
			}},
		{name: "a suggested status", res: result("I", fired("Fit", `{"kind": "suggestStatus", "status": "ready_for_poa"}`)),
			want: Actions{Suggestions: []Suggestion{{Rule: "Fit", Status: "ready_for_poa"}}}},
		{name: "a held rule keeps its other actions (only its auto-triage was taken out)", res: func() evaluate.Result {
			r := result("III", fired("High risk", task))
			r.Rules[0].Held, r.Rules[0].HeldReason = true, "anaesthetistAsa"
			return r
		}(), want: Actions{Tasks: []RuleTask{{Rule: "High risk", Title: "Book anaesthetic review", Team: "anaesthetist"}}}},
		{name: "every rule that fires, in order; the first rule's name for the requirement",
			res: result("IV",
				fired("High risk", task, require),
				fired("Very high", `{"kind": "task", "title": "Call the consultant", "team": "nurse"}`, flag, require,
					`{"kind": "suggestStatus", "status": "on_hold"}`)),
			want: Actions{
				Tasks: []RuleTask{
					{Rule: "High risk", Title: "Book anaesthetic review", Team: "anaesthetist"},
					{Rule: "Very high", Title: "Call the consultant", Team: "nurse"},
				},
				Flags:       []RuleFlag{{Rule: "Very high", Name: "Anaesthetic review"}},
				RequireASA:  "High risk",
				Suggestions: []Suggestion{{Rule: "Very high", Status: "on_hold"}},
			}},
		{name: "the same action twice in one rule: once", res: result("III", fired("High risk", task, task, flag, flag)),
			want: Actions{
				Tasks: []RuleTask{{Rule: "High risk", Title: "Book anaesthetic review", Team: "anaesthetist"}},
				Flags: []RuleFlag{{Rule: "High risk", Name: "Anaesthetic review"}},
			}},
		{name: "what the rules' check would refuse is left out (a backstop)", res: result("III", fired("Odd",
			`{"kind": "task", "title": "  ", "team": "anaesthetist"}`,
			`{"kind": "task", "title": "Ring", "team": "porters"}`,
			`{"kind": "flag", "name": ""}`,
			`{"kind": "flag", "name": "Two\nlines"}`,
			`{"kind": "flag", "name": "`+strings.Repeat("x", 101)+`"}`, // verification 1, finding 3: Check's limits
			`{"kind": "suggestStatus", "status": "admitted"}`,
			`{"kind": "unknown"}`,
			`not json`))},
		{name: "titles and names trimmed", res: result("III", fired("High risk",
			`{"kind": "task", "title": " Book ", "team": "admin"}`, `{"kind": "flag", "name": " Look "}`)),
			want: Actions{Tasks: []RuleTask{{Rule: "High risk", Title: "Book", Team: "admin"}}, Flags: []RuleFlag{{Rule: "High risk", Name: "Look"}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActionsOf(tt.res); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ActionsOf = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

// The General notes lines for the rules' writes.
func TestActionNotes(t *testing.T) {
	for got, want := range map[string]string{
		RuleTask{Rule: "High risk", Title: "Book anaesthetic review", Team: "anaesthetist"}.Note(): "Task added by the rule “High risk” for the anaesthetist team: Book anaesthetic review",
		RuleFlag{Rule: "High risk", Name: "Anaesthetic review"}.Note():                             "Flag added by the rule “High risk”: Anaesthetic review",
		RequireNote("High risk"): "The rule “High risk” requires the anaesthetist's ASA grade before Ready for admission",
	} {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
