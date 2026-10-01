package episode

import "testing"

func TestCanSetStatus(t *testing.T) {
	tests := []struct {
		current, next string
		want          bool
	}{
		{ReadyForPOA, ReadyForAdmission, true},
		{ReadyForAdmission, OnHold, true},
		{OnHold, POAComplete, true},
		{HQNotComplete, ReadyForAdmission, false}, // the patient hasn't submitted
		{ReadyForReview, POAComplete, false},      // the review isn't complete
		{ReadyForAdmission, ReadyForPOA, false},   // workflow statuses aren't picked by hand
		{ReadyForPOA, HQNotComplete, false},
		{ReadyForReview, ReadyForReview, true}, // saving other fields keeps the status
		{ReadyForPOA, "nonsense", false},
	}
	for _, tt := range tests {
		if got := CanSetStatus(tt.current, tt.next); got != tt.want {
			t.Errorf("CanSetStatus(%s, %s) = %v, want %v", tt.current, tt.next, got, tt.want)
		}
	}
}

func TestNewToken(t *testing.T) {
	a, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewToken()
	if len(a) != 43 || a == b {
		t.Errorf("tokens %q and %q: want two different 43-character tokens", a, b)
	}
}
