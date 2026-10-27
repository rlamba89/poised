// Package triage decides, from what a sent HQ's answers give (the evaluate service's result, S41),
// whether the episode is auto-triaged (plan C6, session S42; ASA-02, A-8): which rule moves it on,
// to where, and the words for the stamp and the General notes. Go never evaluates SurveyJS: the
// service worked out which rules fire; this only reads the result.
//
// The first rule that fires with an auto-triage action wins. Whatever the rules say, the guards
// below always apply and can't be switched off (S42 answers 2): the server's check ran (the caller
// only asks when it did), no red or amber flag, no score Incomplete, and no rule requiring the
// anaesthetist's ASA grade before Ready for admission (S43). The rules' own test cases passed at
// publish (S41's server check), so a published version's rules are tested.
package triage

import (
	"encoding/json"
	"slices"

	"github.com/rlamba89/poised/apps/api/internal/asarules"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

// Decision is what the rules decided. Rule is "" when no rule auto-triages; Blocked says which
// guard stopped the rule that would have (Target is then "").
type Decision struct {
	Rule    string // the rule's name, as the status history and the stamp name it
	Target  string // the status it moves to: one of asarules.Targets
	Summary string // "ASA II, no red or amber flags": the stamp's words
	Blocked string // the guard that stopped it, in words
}

// Moves reports whether the episode is moved on.
func (d Decision) Moves() bool { return d.Target != "" }

// Note is the General notes line: the move, or why the rule didn't move it.
func (d Decision) Note() string {
	if d.Blocked != "" {
		return "Not auto-triaged by the rule “" + d.Rule + "”: " + d.Blocked + ". Review this HQ by hand."
	}
	return "Auto-triaged by the rule “" + d.Rule + "”: " + d.Summary + " → " + episode.Labels[d.Target]
}

// guards are the evaluator's reasons for holding a rule's auto-triage that are S42's guards (not
// "noGrade", the HQ's own setting), in the words of the General notes line.
var guards = map[string]string{
	"flag":            "the answers have a red or amber flag",
	"incomplete":      "a score is Incomplete",
	"anaesthetistAsa": AnaesthetistASAGuard,
}

// AnaesthetistASAGuard is the guard's words when a rule requires the anaesthetist's ASA grade
// before Ready for admission (S43).
const AnaesthetistASAGuard = "a rule requires the anaesthetist's ASA grade first"

// Decide reads the result of a send whose checks ran. The evaluator applies the guards itself
// (packages/clinical's firedRules holds the rule's auto-triage, with the reason), so the Preview and
// the test cases say what a send does; Go checks them again, in case it didn't.
func Decide(res evaluate.Result) Decision {
	var d, held Decision // held: the first rule the evaluator held for a guard
	required := false    // a rule requires the anaesthetist's ASA (S43)
	for _, r := range res.Rules {
		if why, ok := guards[r.HeldReason]; ok && r.Held && held.Rule == "" {
			held = Decision{Rule: r.Name, Blocked: why}
		}
		for _, raw := range r.Actions {
			var a asarules.Action
			if json.Unmarshal(raw, &a) != nil {
				continue
			}
			switch {
			case a.Kind == asarules.RequireAnaesthetistASA:
				required = true
			case a.Kind == asarules.AutoTriage && d.Rule == "" && slices.Contains(asarules.Targets, a.Target):
				d.Rule, d.Target = r.Name, a.Target
			}
		}
	}
	if d.Rule == "" {
		return held // the zero Decision when nothing was held either
	}
	switch {
	case len(res.Flags) > 0:
		d.Blocked = "the answers have a red or amber flag"
	case incomplete(res):
		d.Blocked = "a score is Incomplete"
	case required && d.Target == episode.ReadyForAdmission:
		d.Blocked = AnaesthetistASAGuard
	}
	if d.Blocked != "" {
		return Decision{Rule: d.Rule, Blocked: d.Blocked}
	}
	asa := "no ASA grade"
	if g := res.SuggestedASA; g != nil {
		asa = "ASA " + g.Grade
		if g.Emergency {
			asa += "E"
		}
	}
	d.Summary = asa + ", no red or amber flags"
	return d
}

func incomplete(res evaluate.Result) bool {
	for _, s := range res.Sets {
		for _, sc := range s.Scores {
			if sc.Incomplete {
				return true
			}
		}
	}
	return false
}
