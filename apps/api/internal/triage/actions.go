package triage

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/rlamba89/poised/apps/api/internal/asarules"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

// The rules' other actions (plan C6, session S43; ASA-03, TSK-01): besides auto-triage (Decide),
// a rule that fires on a send may add a task for a team, add a flag to the episode, require the
// anaesthetist's ASA grade before Ready for admission, or suggest a status. ActionsOf turns the
// result into those writes; internal/httpapi makes them in the send's transaction. A held rule
// (S42) keeps these: only its auto-triage was taken out.

// RuleTask is a task a rule adds (tasks.created_by_rule is the rule's name).
type RuleTask struct{ Rule, Title, Team string }

// RuleFlag is a flag a rule adds to the episode (episode_flags).
type RuleFlag struct{ Rule, Name string }

// Suggestion is a status a rule suggests: shown on the episode, never applied (S43).
type Suggestion struct {
	Rule   string `json:"rule"`
	Status string `json:"status"`
}

// Actions are the writes for one send's fired rules, in the rules' order.
type Actions struct {
	Tasks       []RuleTask
	Flags       []RuleFlag
	RequireASA  string // the first rule requiring the anaesthetist's ASA grade, or ""
	Suggestions []Suggestion
}

// ActionsOf reads the fired rules' actions. The rules' shape was checked when they were saved
// (asarules.Check); anything that check would refuse is left out here too, as a backstop.
func ActionsOf(res evaluate.Result) Actions {
	var out Actions
	for _, r := range res.Rules {
		for _, raw := range r.Actions {
			var a asarules.Action
			if json.Unmarshal(raw, &a) != nil {
				continue
			}
			switch a.Kind {
			case asarules.Task:
				title, problem := episode.TaskInput(a.Title, a.Team)
				if t := (RuleTask{Rule: r.Name, Title: title, Team: a.Team}); problem == "" && !slices.Contains(out.Tasks, t) {
					out.Tasks = append(out.Tasks, t)
				}
			case asarules.Flag:
				name := strings.TrimSpace(a.Name)
				ok := name != "" && !strings.ContainsAny(name, "\r\n") && utf8.RuneCountInString(name) <= asarules.MaxName // as asarules.Check
				if f := (RuleFlag{Rule: r.Name, Name: name}); ok && !slices.Contains(out.Flags, f) {
					out.Flags = append(out.Flags, f)
				}
			case asarules.RequireAnaesthetistASA:
				if out.RequireASA == "" {
					out.RequireASA = r.Name
				}
			case asarules.SuggestStatus:
				if s := (Suggestion{Rule: r.Name, Status: a.Status}); slices.Contains(asarules.Suggestable, a.Status) && !slices.Contains(out.Suggestions, s) {
					out.Suggestions = append(out.Suggestions, s)
				}
			}
		}
	}
	return out
}

// Note is the General notes line when the task is added.
func (t RuleTask) Note() string {
	return "Task added by the rule “" + t.Rule + "” for the " + t.Team + " team: " + t.Title
}

// Note is the General notes line when the flag is added.
func (f RuleFlag) Note() string { return "Flag added by the rule “" + f.Rule + "”: " + f.Name }

// RequireNote is the General notes line when a rule first requires the anaesthetist's ASA grade.
func RequireNote(rule string) string {
	return "The rule “" + rule + "” requires the anaesthetist's ASA grade before Ready for admission"
}

// NeedsAnaesthetistASA is the refusal of a move to Ready for admission while a rule's requirement
// is unmet (S43): the status control shows it, and the API answers it with 409.
const NeedsAnaesthetistASA = "Add the anaesthetist's ASA grade first."
