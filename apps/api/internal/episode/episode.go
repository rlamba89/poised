// Package episode holds the episode workflow's rules (docs/plan-workflow.md): statuses and
// the patient's link token.
package episode

import (
	"crypto/rand"
	"encoding/base64"
)

// Statuses are Lifebox's Full HQ states (statemodel.go) with readable keys, in workflow order.
const (
	HQNotComplete     = "hq_not_complete"     // Lifebox "primary"
	ReadyForReview    = "ready_for_review"    // "review": the patient has submitted (triage)
	ReadyForPOA       = "ready_for_poa"       // "triageready": the clinician completed the HQ review
	POAComplete       = "poa_complete"        // "pending"
	OnHold            = "on_hold"             // "onhold"
	NotReady          = "not_ready"           // "notready"
	ReadyForAdmission = "ready_for_admission" // "ready"
)

// Labels are the names shown to clinicians, as in Lifebox.
var Labels = map[string]string{
	HQNotComplete:     "HQ not complete",
	ReadyForReview:    "Ready for review",
	ReadyForPOA:       "Ready for POA",
	POAComplete:       "POA complete",
	OnHold:            "On hold",
	NotReady:          "Not ready for admission",
	ReadyForAdmission: "Ready for admission",
}

// manual are the statuses a clinician picks from the status dropdown. The first three are set
// by the workflow itself: creating the episode, the patient submitting, completing the review.
var manual = map[string]bool{POAComplete: true, OnHold: true, NotReady: true, ReadyForAdmission: true}

// CanSetStatus reports whether a clinician may move an episode from current to next by hand:
// only to a manual status, and only once the HQ review is complete.
func CanSetStatus(current, next string) bool {
	if current == next {
		return true
	}
	return manual[next] && (current == ReadyForPOA || manual[current])
}

// NewToken returns a random, URL-safe token for the patient's link (32 bytes).
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
