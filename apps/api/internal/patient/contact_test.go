package patient_test

import (
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/patient"
)

func TestContact(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		email    bool
	}{
		{" Sam@Example.ORG ", "sam@example.org", true},
		{"07700 900123", "+447700900123", false},
		{"+44 7700 900123", "+447700900123", false},
	} {
		got, email, msg := patient.Contact(tc.in, "GB")
		if msg != "" || got != tc.want || email != tc.email {
			t.Errorf("Contact(%q) = %q, %v, %q; want %q, %v", tc.in, got, email, msg, tc.want, tc.email)
		}
	}
	for _, bad := range []string{"", "  ", "hello", "a@b", "sam@", "123", "+1 202 555 0143", "07700 900123 or sam@example.org"} {
		if got, _, msg := patient.Contact(bad, "GB"); msg == "" {
			t.Errorf("Contact(%q) = %q, want refused", bad, got)
		}
	}
	// PAT-11: a mobile in another country is refused with its own message, pointing to email.
	if _, _, msg := patient.Contact("+1 202 555 0143", "GB"); msg != patient.NotThisCountry {
		t.Errorf("another country's mobile: %q", msg)
	}
	// The same number is fine in a deployment of that country (A-20: config, not code).
	if got, _, msg := patient.Contact("(202) 555-0143", "US"); msg != "" || got != "+12025550143" {
		t.Errorf("a US deployment: %q %q", got, msg)
	}
}

func TestMask(t *testing.T) {
	for in, want := range map[string]string{
		"+447700900123":       "•••123",
		"":                    "",
		"+44123":              "•••123",
	} {
		if got := patient.MaskMobile(in); got != want {
			t.Errorf("MaskMobile(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{
		"sam.jones@example.org": "s•••@example.org",
		"s@example.org":         "s•••@example.org",
		"":                      "",
	} {
		if got := patient.MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
