// Package episode holds the episode workflow's rules (docs/plan-workflow.md, plan C3): its
// statuses, and which moves between them are allowed, to whom (S20).
// The patient's link is internal/auth's (NewLinkToken, S14).
package episode

import (
	"errors"
	"strings"

	"github.com/rlamba89/poised/apps/api/internal/role"
)

// Statuses are Lifebox's Full HQ states (statemodel.go) with readable keys, then the ones after
// the POA (saas-requirements §5.2). `recovery` (C9) and `archived` (S22) aren't built yet.
const (
	HQNotComplete     = "hq_not_complete"     // Lifebox "primary"
	ReadyForReview    = "ready_for_review"    // "review": the patient has submitted (triage)
	ReadyForPOA       = "ready_for_poa"       // "triageready": the clinician completed the HQ review
	POAComplete       = "poa_complete"        // "pending"
	OnHold            = "on_hold"             // "onhold"
	NotReady          = "not_ready"           // "notready"
	ReadyForAdmission = "ready_for_admission" // "ready"
	Admitted          = "admitted"            // with the admission date (EPI-01)
	Discharged        = "discharged"          // with the discharge date
	Cancelled         = "cancelled"           // with a reason; final (S20 answers 3)
)

// Statuses in workflow order: the status control lists its options in this order.
var Statuses = []string{HQNotComplete, ReadyForReview, ReadyForPOA, POAComplete, OnHold, NotReady, ReadyForAdmission,
	Admitted, Discharged, Cancelled}

// Labels are the names shown to clinicians, as in Lifebox.
var Labels = map[string]string{
	HQNotComplete:     "HQ not complete",
	ReadyForReview:    "Ready for review",
	ReadyForPOA:       "Ready for POA",
	POAComplete:       "POA complete",
	OnHold:            "On hold",
	NotReady:          "Not ready for admission",
	ReadyForAdmission: "Ready for admission",
	Admitted:          "Admitted",
	Discharged:        "Discharged",
	Cancelled:         "Cancelled",
}

// Move is the kind of a status change. Each is recorded with the change (EPI-03).
type Move string

const (
	Create         Move = "create"          // the episode is made: its first status
	Submit         Move = "submit"          // the patient sends the HQ, or staff for them (PAT-09)
	CompleteReview Move = "complete_review" // the clinician completes the HQ review
	Set            Move = "set"             // a clinician picks a status by hand
	Admit          Move = "admit"           // with the admission date
	Discharge      Move = "discharge"       // with the discharge date
	Cancel         Move = "cancel"          // with a reason from the list; final
	Reopen         Move = "reopen"          // re-opens the HQ review (EPI-07), with a reason
	Undo           Move = "undo"            // steps back a move made by hand, with a reason
)

// Rule is what an allowed move needs.
type Rule struct {
	Move        Move
	Who         role.Role // the least role that may make it (the A-5 ladder)
	Patient     bool      // the patient may make it too (their submit)
	NeedsReason bool
	NeedsDate   bool
}

var (
	// ErrIllegal: the move isn't in the table (a 409: the episode isn't where the move starts).
	ErrIllegal = errors.New("episode: the status can't be changed to that now")
	// ErrRole: the move is in the table, but needs a higher role (a 403).
	ErrRole = errors.New("episode: your role can't make that change")
)

// reviewed are the statuses once the HQ review is complete and before admission, which a
// clinician moves between by hand.
var reviewed = []string{ReadyForPOA, POAComplete, OnHold, NotReady, ReadyForAdmission}

type pair struct{ from, to string }

// allowed is the transition table (S20 answers 1, 3, 4: Rahul, 20 Oct 2026), from → to.
// Undo isn't in it: its target depends on the history (UndoTarget).
var allowed = func() map[pair]Rule {
	t := map[pair]Rule{
		{HQNotComplete, ReadyForReview}: {Move: Submit, Who: role.Clinician, Patient: true},
		{ReadyForReview, ReadyForPOA}:   {Move: CompleteReview, Who: role.Clinician},
		{ReadyForAdmission, Admitted}:   {Move: Admit, Who: role.Clinician, NeedsDate: true},
		{Admitted, Discharged}:          {Move: Discharge, Who: role.Clinician, NeedsDate: true},
	}
	for _, from := range reviewed {
		for _, to := range reviewed {
			if from != to {
				t[pair{from, to}] = Rule{Move: Set, Who: role.Clinician}
			}
		}
		t[pair{from, ReadyForReview}] = Rule{Move: Reopen, Who: role.SuperClinician, NeedsReason: true}
	}
	// Any status before discharged can be cancelled.
	for _, from := range []string{HQNotComplete, ReadyForReview, ReadyForPOA, POAComplete, OnHold, NotReady, ReadyForAdmission, Admitted} {
		t[pair{from, Cancelled}] = Rule{Move: Cancel, Who: role.Clinician, NeedsReason: true}
	}
	return t
}()

// Check says whether someone with role who (role.None: the patient) may make move from → to,
// and what it needs. ErrIllegal: not a move in the table; ErrRole: their role is too low.
func Check(from, to string, move Move, who role.Role) (Rule, error) {
	r, ok := allowed[pair{from, to}]
	if !ok || r.Move != move {
		return Rule{}, ErrIllegal
	}
	if who == role.None && !r.Patient || who != role.None && !who.AtLeast(r.Who) {
		return Rule{}, ErrRole
	}
	return r, nil
}

// Option is a move the status control offers.
type Option struct {
	To          string `json:"to"`
	Label       string `json:"label"`
	Move        Move   `json:"move"`
	NeedsReason bool   `json:"needsReason"`
	NeedsDate   bool   `json:"needsDate"`
}

// Options are the moves someone with role who may make by hand from here, in status order. The
// workflow's own (the submit, completing the review) have their own buttons, so aren't listed.
func Options(from string, who role.Role) []Option {
	var out []Option
	for _, to := range Statuses {
		r, ok := allowed[pair{from, to}]
		if !ok || r.Move == Submit || r.Move == CompleteReview {
			continue
		}
		if _, err := Check(from, to, r.Move, who); err == nil {
			out = append(out, Option{To: to, Label: Labels[to], Move: r.Move, NeedsReason: r.NeedsReason, NeedsDate: r.NeedsDate})
		}
	}
	return out
}

// Event is a recorded status change, as UndoTarget needs it.
type Event struct {
	Move     Move
	From, To string
}

// undoable are the moves made by hand that an undo can step back (S20 answers 2). Not the
// workflow's (re-open undoes completing the review), not re-open (complete the review again),
// not cancel (final), and not undo itself.
var undoable = map[Move]bool{Set: true, Admit: true, Discharge: true}

// UndoTarget is where an undo goes from current: back to where latest came from. latest is the
// episode's newest change that isn't an undo and hasn't been undone (nil: none). Any clinician
// may undo, with a reason, and again to go further back, one step at a time (S20 answers 2).
func UndoTarget(current string, latest *Event) (string, bool) {
	if latest == nil || !undoable[latest.Move] || latest.To != current || latest.From == "" {
		return "", false
	}
	return latest.From, true
}

// cancelReasons are the fixed list (S20 answers 3); "other" needs a sentence.
var cancelReasons = map[string]string{
	"procedure_cancelled": "Procedure cancelled",
	"patient_withdrew":    "Patient withdrew",
	"moved_hospital":      "Moved to another hospital",
	"other":               "Other",
}

// CancelReason is a cancellation's reason as recorded ("Patient withdrew", "Other: Emigrated"),
// from the list's key and any details, or a plain-language problem.
func CancelReason(key, details string) (string, string) {
	label, ok := cancelReasons[key]
	details = strings.TrimSpace(details)
	switch {
	case !ok:
		return "", "Choose why the episode is cancelled."
	case key == "other" && details == "":
		return "", "Say why the episode is cancelled."
	case details == "":
		return label, ""
	}
	return label + ": " + details, ""
}
