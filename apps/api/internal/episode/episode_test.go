package episode

import (
	"errors"
	"slices"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/role"
)

// legal is the transition table as agreed (S20 answers 1, 3, 4, Rahul 20 Oct 2026), written out
// again here by hand so the test checks the table rather than copying it.
type legal struct {
	from, to string
	move     Move
	who      role.Role // the least role
	patient  bool      // the patient may make it too
	reason   bool      // a reason is required
	date     bool      // a date is required
}

func agreed() []legal {
	reviewed := []string{ReadyForPOA, POAComplete, OnHold, NotReady, ReadyForAdmission}
	out := []legal{
		{from: HQNotComplete, to: ReadyForReview, move: Submit, who: role.Clinician, patient: true},
		{from: ReadyForReview, to: ReadyForPOA, move: CompleteReview, who: role.Clinician},
		{from: ReadyForAdmission, to: Admitted, move: Admit, who: role.Clinician, date: true},
		{from: Admitted, to: Discharged, move: Discharge, who: role.Clinician, date: true},
	}
	for _, from := range reviewed {
		for _, to := range reviewed {
			if from != to {
				out = append(out, legal{from: from, to: to, move: Set, who: role.Clinician})
			}
		}
		out = append(out, legal{from: from, to: ReadyForReview, move: Reopen, who: role.SuperClinician, reason: true})
	}
	for _, from := range []string{HQNotComplete, ReadyForReview, ReadyForPOA, POAComplete, OnHold, NotReady, ReadyForAdmission, Admitted} {
		out = append(out, legal{from: from, to: Cancelled, move: Cancel, who: role.Clinician, reason: true})
	}
	return out
}

var moves = []Move{Create, Submit, CompleteReview, Set, Admit, Discharge, Cancel, Reopen, Undo, "nonsense"}

// Every pair of statuses, with every move and every role (and the patient): the agreed moves are
// allowed at their least role and above, refused below it (ErrRole); everything else is illegal.
func TestCheckEveryPair(t *testing.T) {
	all := append(slices.Clone(Statuses), "nonsense", "")
	who := []role.Role{role.None, role.Clinician, role.SuperClinician, role.Admin} // role.None: the patient
	checked := 0
	for _, from := range all {
		for _, to := range all {
			for _, m := range moves {
				want := legal{}
				found := false
				for _, l := range agreed() {
					if l.from == from && l.to == to && l.move == m {
						want, found = l, true
					}
				}
				for _, w := range who {
					rule, err := Check(from, to, m, w)
					checked++
					switch {
					case !found:
						if !errors.Is(err, ErrIllegal) {
							t.Errorf("Check(%s → %s, %s, %s) = %v, want ErrIllegal", from, to, m, w, err)
						}
					case w == role.None && !want.patient, w != role.None && !w.AtLeast(want.who):
						if !errors.Is(err, ErrRole) {
							t.Errorf("Check(%s → %s, %s, %s) = %v, want ErrRole", from, to, m, w, err)
						}
					default:
						if err != nil {
							t.Errorf("Check(%s → %s, %s, %s) = %v, want allowed", from, to, m, w, err)
						} else if rule.NeedsReason != want.reason || rule.NeedsDate != want.date || rule.Move != m {
							t.Errorf("Check(%s → %s, %s) = %+v, want reason %v, date %v", from, to, m, rule, want.reason, want.date)
						}
					}
				}
			}
		}
	}
	if checked < 1000 {
		t.Fatalf("checked only %d combinations", checked)
	}
}

func TestEveryStatusHasALabel(t *testing.T) {
	for _, s := range Statuses {
		if Labels[s] == "" {
			t.Errorf("%s has no label", s)
		}
	}
	if len(Labels) != len(Statuses) {
		t.Errorf("%d labels for %d statuses", len(Labels), len(Statuses))
	}
}

// Options are what the status control offers: the moves this role may make from here, in the
// order of the statuses, never the workflow's own (submit, complete review).
func TestOptions(t *testing.T) {
	targets := func(opts []Option) []string {
		var out []string
		for _, o := range opts {
			out = append(out, string(o.Move)+":"+o.To)
		}
		return out
	}
	for _, tt := range []struct {
		from string
		who  role.Role
		want []string
	}{
		{HQNotComplete, role.Clinician, []string{"cancel:cancelled"}},
		{ReadyForReview, role.Admin, []string{"cancel:cancelled"}},
		{OnHold, role.Clinician, []string{"set:ready_for_poa", "set:poa_complete", "set:not_ready", "set:ready_for_admission", "cancel:cancelled"}},
		{OnHold, role.SuperClinician, []string{"reopen:ready_for_review", "set:ready_for_poa", "set:poa_complete", "set:not_ready",
			"set:ready_for_admission", "cancel:cancelled"}},
		{ReadyForAdmission, role.Clinician, []string{"set:ready_for_poa", "set:poa_complete", "set:on_hold", "set:not_ready",
			"admit:admitted", "cancel:cancelled"}},
		{Admitted, role.SuperClinician, []string{"discharge:discharged", "cancel:cancelled"}},
		{Discharged, role.Admin, nil},
		{Cancelled, role.Admin, nil},
		{ReadyForPOA, role.None, nil},
	} {
		if got := targets(Options(tt.from, tt.who)); !slices.Equal(got, tt.want) {
			t.Errorf("Options(%s, %s) = %q, want %q", tt.from, tt.who, got, tt.want)
		}
	}
}

// Undo (S20 answers 2): any clinician steps back the latest move made by hand that isn't undone
// yet, to the status it came from; repeatable, one step at a time. The workflow's moves, re-open,
// cancel and undo itself can't be undone, and the stack stops at them.
func TestUndoTarget(t *testing.T) {
	ev := func(m Move, from, to string) Event { return Event{Move: m, From: from, To: to} }
	for _, tt := range []struct {
		name    string
		current string
		latest  *Event
		want    string
	}{
		{"a pick", OnHold, &Event{Move: Set, From: ReadyForPOA, To: OnHold}, ReadyForPOA},
		{"admitted by mistake", Admitted, ptr(ev(Admit, ReadyForAdmission, Admitted)), ReadyForAdmission},
		{"discharged by mistake", Discharged, ptr(ev(Discharge, Admitted, Discharged)), Admitted},
		{"the review: re-open instead", ReadyForPOA, ptr(ev(CompleteReview, ReadyForReview, ReadyForPOA)), ""},
		{"the patient's submit", ReadyForReview, ptr(ev(Submit, HQNotComplete, ReadyForReview)), ""},
		{"a re-open", ReadyForReview, ptr(ev(Reopen, OnHold, ReadyForReview)), ""},
		{"cancel stays final", Cancelled, ptr(ev(Cancel, OnHold, Cancelled)), ""},
		{"created", HQNotComplete, ptr(ev(Create, "", HQNotComplete)), ""},
		{"no history", OnHold, nil, ""},
		{"history out of step with the status", NotReady, ptr(ev(Set, ReadyForPOA, OnHold)), ""},
	} {
		if got, ok := UndoTarget(tt.current, tt.latest); got != tt.want || ok != (tt.want != "") {
			t.Errorf("%s: UndoTarget = %q, %v; want %q", tt.name, got, ok, tt.want)
		}
	}
}

func ptr(e Event) *Event { return &e }

func TestCancelReasons(t *testing.T) {
	for _, tt := range []struct {
		key, details string
		want         string // the reason as recorded; "" refused
	}{
		{"procedure_cancelled", "", "Procedure cancelled"},
		{"patient_withdrew", "  ", "Patient withdrew"},
		{"moved_hospital", " to Leeds ", "Moved to another hospital: to Leeds"},
		{"other", " Emigrated ", "Other: Emigrated"},
		{"other", "  ", ""}, // "Other" needs a sentence
		{"", "", ""},
		{"nonsense", "x", ""},
	} {
		if got, _ := CancelReason(tt.key, tt.details); got != tt.want {
			t.Errorf("CancelReason(%q, %q) = %q, want %q", tt.key, tt.details, got, tt.want)
		}
	}
}
