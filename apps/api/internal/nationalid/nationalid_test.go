package nationalid_test

import (
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/nationalid"
)

func TestNHSNumber(t *testing.T) {
	nhs, ok := nationalid.Lookup("nhs")
	if !ok {
		t.Fatal("no nhs scheme")
	}
	if nhs.Label != "NHS number" {
		t.Errorf("label = %q", nhs.Label)
	}
	for _, tc := range []struct {
		in, norm string
		valid    bool
	}{
		{"9434765919", "9434765919", true},   // check digit 9: 11 − (299 mod 11)
		{"943 476 5919", "9434765919", true}, // spaces, as printed on letters
		{" 943-476-5919 ", "9434765919", true},
		{"4010232137", "4010232137", true},
		{"9990008000", "9990008000", true},  // remainder 0: the check digit is 0 (11 → 0)
		{"9434765918", "9434765918", false}, // wrong check digit
		{"9434765910", "9434765910", false},
		{"1000000010", "1000000010", false}, // the check digit would be 10: never issued
		{"943476591", "943476591", false},   // 9 digits
		{"94347659190", "94347659190", false},
		{"94347659l9", "94347659l9", false}, // a letter
		{"", "", false},
	} {
		if got := nhs.Normalise(tc.in); got != tc.norm {
			t.Errorf("Normalise(%q) = %q, want %q", tc.in, got, tc.norm)
		}
		if got := nhs.Valid(nhs.Normalise(tc.in)); got != tc.valid {
			t.Errorf("Valid(%q) = %v, want %v", tc.in, got, tc.valid)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, ok := nationalid.Lookup("chi"); ok {
		t.Error("chi isn't built yet")
	}
	if _, ok := nationalid.Lookup(""); ok {
		t.Error("an empty type isn't a scheme")
	}
}
