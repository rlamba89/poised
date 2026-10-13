package httpapi

import (
	"strings"
	"testing"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/config"
)

// The episode routes are tested against the database (episodes_integration_test.go); these are
// the input checks on their own. (S08 replaced the fake-querier route tests, testing rule 4.)

func TestPatientDetails(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	s := &server{cfg: config.Config{Country: config.Country{PhoneCountry: "GB", NationalIDType: "nhs"}}}
	valid := func() patientInput {
		return patientInput{FirstName: " Jo ", LastName: "Bloggs", DateOfBirth: "1974-12-11", Sex: "male",
			NationalID: " 943 476 5919 ", Mobile: "07700 900001", Email: " Jo@Example.TEST"}
	}
	d, sex, msg := s.details(valid(), now)
	if msg != "" || d.FirstName != "Jo" || d.NationalID != "9434765919" || d.Mobile != "+447700900001" ||
		d.Email != "jo@example.test" || sex != "male" {
		t.Errorf("valid: %q, %+v; want no problem, cleaned up", msg, d)
	}
	for name, change := range map[string]func(*patientInput){
		"no last name":       func(p *patientInput) { p.LastName = " " },
		"date not a date":    func(p *patientInput) { p.DateOfBirth = "11/12/1974" },
		"born tomorrow":      func(p *patientInput) { p.DateOfBirth = "2026-10-13" },
		"born before 1900":   func(p *patientInput) { p.DateOfBirth = "1899-12-31" },
		"unknown sex value":  func(p *patientInput) { p.Sex = "x" },
		"a bad check digit":  func(p *patientInput) { p.NationalID = "9434765918" },
		"not a phone number": func(p *patientInput) { p.Mobile = "12345" },
		"not an email":       func(p *patientInput) { p.Email = "jo@" },
		"a name that's long": func(p *patientInput) { p.FirstName = strings.Repeat("J", 101) },
	} {
		in := valid()
		change(&in)
		if _, _, msg := s.details(in, now); msg == "" {
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
