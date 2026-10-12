package role

import "testing"

// A-5: one ordered role, clinician < super clinician < admin, so a permission check is a
// single comparison.
func TestLadder(t *testing.T) {
	if !(Clinician < SuperClinician && SuperClinician < Admin) {
		t.Fatal("the ladder is out of order")
	}
	if None.AtLeast(Clinician) {
		t.Error("no role can do what a clinician does")
	}
	for _, tt := range []struct {
		have, need Role
		want       bool
	}{
		{Clinician, Clinician, true},
		{Clinician, SuperClinician, false}, // a clinician can't publish
		{SuperClinician, SuperClinician, true},
		{SuperClinician, Admin, false}, // a super clinician can't manage staff
		{Admin, Clinician, true},
		{Admin, SuperClinician, true},
	} {
		if got := tt.have.AtLeast(tt.need); got != tt.want {
			t.Errorf("%s.AtLeast(%s) = %v, want %v", tt.have, tt.need, got, tt.want)
		}
	}
}

func TestParse(t *testing.T) {
	for s, want := range map[string]Role{"clinician": Clinician, "super_clinician": SuperClinician, "admin": Admin} {
		got, ok := Parse(s)
		if !ok || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", s, got, ok, want)
		}
		if got.String() != s {
			t.Errorf("%v.String() = %q, want %q", got, got.String(), s)
		}
	}
	// Today's old roles are gone (A-5): the migration maps them.
	for _, s := range []string{"", "viewer", "author", "publisher", "hospital_admin", "Admin"} {
		if got, ok := Parse(s); ok || got != None {
			t.Errorf("Parse(%q) = %v, %v; want None, false", s, got, ok)
		}
	}
}

// Someone with a trust-wide role and a role at one hospital has the higher of the two there.
func TestHighest(t *testing.T) {
	if got := Highest("clinician", "admin"); got != Admin {
		t.Errorf("Highest(clinician, admin) = %s, want admin", got)
	}
	if got := Highest("super_clinician"); got != SuperClinician {
		t.Errorf("Highest(super_clinician) = %s", got)
	}
	if got := Highest(); got != None {
		t.Errorf("Highest() = %s, want none", got)
	}
	if got := Highest("viewer"); got != None {
		t.Errorf("an unknown role counts for nothing, got %s", got)
	}
}
