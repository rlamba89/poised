package httpapi

import (
	"testing"
	"time"
)

// The episode routes are tested against the database (episodes_integration_test.go); these are
// the input checks on their own. (S08 replaced the fake-querier route tests, testing rule 4.)

func TestPatientInputValidate(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	valid := func() patientInput {
		return patientInput{FirstName: " Jo ", LastName: "Bloggs", DateOfBirth: "1974-12-11", Sex: "male"}
	}
	in := valid()
	if _, msg := in.validate(now); msg != "" || in.FirstName != "Jo" {
		t.Errorf("valid: %q, first name %q; want no problem, trimmed", msg, in.FirstName)
	}
	for name, change := range map[string]func(*patientInput){
		"no last name":      func(p *patientInput) { p.LastName = " " },
		"date not a date":   func(p *patientInput) { p.DateOfBirth = "11/12/1974" },
		"born tomorrow":     func(p *patientInput) { p.DateOfBirth = "2026-10-13" },
		"born before 1900":  func(p *patientInput) { p.DateOfBirth = "1899-12-31" },
		"unknown sex value": func(p *patientInput) { p.Sex = "x" },
	} {
		in := valid()
		change(&in)
		if _, msg := in.validate(now); msg == "" {
			t.Errorf("%s: no problem reported", name)
		}
	}
}

// A PATCH sends only what it changes. A text field sent as null is refused, not a crash (found
// by S01's code review: the handler dereferenced the nil pointer and dropped the connection).
func TestEpisodeChangeValidate(t *testing.T) {
	s := func(v string) *string { return &v }
	valid := func() episodeChange {
		return episodeChange{Status: s("ready_for_poa"), Procedure: s(" Hip replacement "), Anaesthetic: s(" GA "), Consultant: s(" Dr Who ")}
	}

	in := valid()
	if msg := in.validate(); msg != "" || *in.Procedure != "Hip replacement" || *in.Anaesthetic != "GA" || *in.Consultant != "Dr Who" {
		t.Errorf("valid change: %q, %q %q %q; want no problem, trimmed", msg, *in.Procedure, *in.Anaesthetic, *in.Consultant)
	}
	for name, change := range map[string]func(*episodeChange){
		"null status":      func(c *episodeChange) { c.Status = nil },
		"null procedure":   func(c *episodeChange) { c.Procedure = nil },
		"null anaesthetic": func(c *episodeChange) { c.Anaesthetic = nil },
		"null consultant":  func(c *episodeChange) { c.Consultant = nil },
	} {
		in := valid()
		change(&in)
		if in.validate() == "" {
			t.Errorf("%s: no problem reported", name)
		}
	}
}
