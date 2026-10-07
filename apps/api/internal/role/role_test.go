package role

import "testing"

func TestAtLeast(t *testing.T) {
	for _, tt := range []struct {
		have, want Role
		ok         bool
	}{
		{Clinician, Clinician, true},
		{Clinician, SuperClinician, false},
		{Clinician, Admin, false},
		{SuperClinician, Clinician, true},
		{SuperClinician, SuperClinician, true},
		{SuperClinician, Admin, false},
		{Admin, Clinician, true},
		{Admin, SuperClinician, true},
		{Admin, Admin, true},
		{"", Clinician, false},
		{"author", Clinician, false}, // an old role name is not on the ladder
	} {
		if got := tt.have.AtLeast(tt.want); got != tt.ok {
			t.Errorf("%q.AtLeast(%q) = %v, want %v", tt.have, tt.want, got, tt.ok)
		}
	}
}

func TestHighest(t *testing.T) {
	if got := Highest(Clinician, Admin, SuperClinician); got != Admin {
		t.Errorf("Highest = %q, want admin", got)
	}
	if got := Highest(); got != "" {
		t.Errorf("Highest() = %q, want empty", got)
	}
	if got := Highest("viewer", Clinician); got != Clinician {
		t.Errorf("Highest ignores roles off the ladder: got %q, want clinician", got)
	}
}
