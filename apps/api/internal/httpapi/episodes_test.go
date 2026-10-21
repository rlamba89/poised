package httpapi

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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
	rev := int32(3)
	valid := func() episodeChange {
		return episodeChange{Revision: &rev, Anaesthetic: s(" GA "), Consultant: s(" Dr Who ")}
	}

	in := valid()
	if msg := in.validate(); msg != "" || *in.Anaesthetic != "GA" || *in.Consultant != "Dr Who" {
		t.Errorf("valid change: %q, %q %q; want no problem, trimmed", msg, *in.Anaesthetic, *in.Consultant)
	}
	for name, change := range map[string]func(*episodeChange){
		"no revision (S20)": func(c *episodeChange) { c.Revision = nil },
		"a status (S20)":    func(c *episodeChange) { c.Status = s("poa_complete") },
		"null anaesthetic":  func(c *episodeChange) { c.Anaesthetic = nil },
		"null consultant":   func(c *episodeChange) { c.Consultant = nil },
	} {
		in := valid()
		change(&in)
		if in.validate() == "" {
			t.Errorf("%s: no problem reported", name)
		}
	}
}

// The procedure and its dates (S19): each optional, "" for none; a null in a PATCH clears it too.
func TestReadEpisodeDates(t *testing.T) {
	id := uuid.New()
	d, msg := readEpisodeDates(" "+id.String()+" ", " 2026-11-03 ", "")
	if msg != "" || !d.procedure.Valid || d.procedure.UUID != id || !d.planned.Valid || d.planned.Time.Format(time.DateOnly) != "2026-11-03" || d.poa.Valid {
		t.Errorf("valid: %q %+v", msg, d)
	}
	if d, msg := readEpisodeDates("", "", ""); msg != "" || d != (episodeDates{}) {
		t.Errorf("none: %q %+v", msg, d)
	}
	for name, in := range map[string][3]string{
		"a procedure that isn't an id": {"hip", "", ""},
		"a planned date not a date":    {"", "03/11/2026", ""},
		"a POA date not a date":        {"", "", "2026-02-30"},
		"a date before 1900":           {"", "1899-12-31", ""},
		"a date after 2199":            {"", "", "2200-01-01"},
	} {
		if _, msg := readEpisodeDates(in[0], in[1], in[2]); msg == "" {
			t.Errorf("%s: no problem reported", name)
		}
	}
}

func TestEpisodeDatesChanges(t *testing.T) {
	date := func(s string) pgtype.Date { d, _ := optionalDate(s); return d }
	hip := uuid.NullUUID{UUID: uuid.New(), Valid: true}
	was := episodeDates{procedure: hip, planned: date("2026-11-03")}
	if got := was.changes(was, "Hip"); len(got) != 0 {
		t.Errorf("no change: %q", got)
	}
	knee := uuid.NullUUID{UUID: uuid.New(), Valid: true}
	got := was.changes(episodeDates{procedure: knee, poa: date("2026-10-28")}, "Knee scope")
	want := []string{"Procedure set to Knee scope", "Planned procedure date cleared", "POA appointment date set to 2026-10-28"}
	if !slices.Equal(got, want) {
		t.Errorf("changes = %q, want %q", got, want)
	}
	if got := was.changes(episodeDates{planned: date("2026-11-04")}, ""); !slices.Equal(got, []string{"Procedure cleared", "Planned procedure date set to 2026-11-04"}) {
		t.Errorf("changes = %q", got)
	}
}
