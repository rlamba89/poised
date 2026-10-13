package patient_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/nationalid"
	"github.com/rlamba89/poised/apps/api/internal/patient"
)

func TestMobile(t *testing.T) {
	// Made-up numbers from Ofcom's drama range (07700 900xxx), as every test uses.
	for _, tc := range []struct{ in, country, want string }{
		{"", "GB", ""},
		{"   ", "GB", ""},
		{"07700 900123", "GB", "+447700900123"},    // the national format, in the deployment's country
		{"+44 7700 900123", "GB", "+447700900123"}, // already international
		{"+44 (0)7700 900123", "GB", "+447700900123"},
		{"0044 7700 900123", "GB", "+447700900123"},
		{"+1 416 555 0199", "GB", "+14165550199"}, // another country's, typed internationally
		{"(416) 555-0199", "CA", "+14165550199"},  // a made-up second deployment's national format
	} {
		got, ok := patient.Mobile(tc.in, tc.country)
		if !ok || got != tc.want {
			t.Errorf("Mobile(%q, %s) = %q, %v; want %q", tc.in, tc.country, got, ok, tc.want)
		}
	}
	for _, bad := range []string{"12345", "abc", "077009001234567", "+44"} {
		if got, ok := patient.Mobile(bad, "GB"); ok {
			t.Errorf("Mobile(%q) = %q, want refused", bad, got)
		}
	}
}

func TestEmail(t *testing.T) {
	for in, want := range map[string]string{"": "", " Ada@Example.TEST ": "ada@example.test", "a.b+c@x.example": "a.b+c@x.example"} {
		if got, ok := patient.Email(in); !ok || got != want {
			t.Errorf("Email(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"ada", "ada@", "@example.test", "Ada <ada@example.test>", "ada@example.test, bob@example.test", "ada @example.test"} {
		if got, ok := patient.Email(bad); ok {
			t.Errorf("Email(%q) = %q, want refused", bad, got)
		}
	}
}

func TestNameKey(t *testing.T) {
	if patient.NameKey(" Mary  Ann ") != patient.NameKey("maryann") || patient.NameKey("O'Neil") != "o'neil" {
		t.Error("names compare ignoring case and spaces")
	}
	if patient.NameKey("Smith") == patient.NameKey("Smyth") {
		t.Error("different names must differ")
	}
}

var (
	birth   = time.Date(1980, 5, 17, 0, 0, 0, 0, time.UTC)
	nhsNo   = "9434765919"
	another = "4010232137"
)

func rec(first, last string, dob time.Time, id, mobile, email string) patient.Record {
	return patient.Record{ID: uuid.New(), FirstName: first, LastName: last, DateOfBirth: dob, NationalID: id, Mobile: mobile, Email: email}
}

func TestDecide(t *testing.T) {
	typed := patient.Details{FirstName: "Sam", LastName: "Smith", DateOfBirth: birth}
	with := func(f func(*patient.Details)) patient.Details { d := typed; f(&d); return d }

	t.Run("no match: create", func(t *testing.T) {
		d := patient.Decide(typed, nil, nil, false)
		if d.Outcome != patient.Create {
			t.Errorf("outcome = %v", d.Outcome)
		}
	})
	t.Run("the national ID and date of birth and surname agree: link silently", func(t *testing.T) {
		r := rec("Samantha", " SMITH", birth, nhsNo, "", "") // first names may differ (Sam, Samantha)
		d := patient.Decide(with(func(d *patient.Details) { d.NationalID = nhsNo }), &r, nil, false)
		if d.Outcome != patient.Link || d.Record.ID != r.ID {
			t.Errorf("decision = %+v, want a link to %s", d, r.ID)
		}
	})
	t.Run("the national ID matches but the date of birth differs: mismatch, even with Create anyway", func(t *testing.T) {
		r := rec("Sam", "Smith", birth.AddDate(0, 0, 1), nhsNo, "", "")
		d := patient.Decide(with(func(d *patient.Details) { d.NationalID = nhsNo }), &r, nil, true)
		if d.Outcome != patient.Mismatch {
			t.Errorf("outcome = %v, want Mismatch", d.Outcome)
		}
	})
	t.Run("the national ID matches but the surname differs: mismatch", func(t *testing.T) {
		r := rec("Sam", "Jones", birth, nhsNo, "", "")
		if d := patient.Decide(with(func(d *patient.Details) { d.NationalID = nhsNo }), &r, nil, false); d.Outcome != patient.Mismatch {
			t.Errorf("outcome = %v, want Mismatch", d.Outcome)
		}
	})
	t.Run("same name and birth and the same mobile: link silently", func(t *testing.T) {
		other := rec("Sam", "Smith", birth, "", "+447700900001", "")
		same := rec(" sam", "SMITH ", birth, "", "+447700900002", "")
		d := patient.Decide(with(func(d *patient.Details) { d.Mobile = "+447700900002" }), nil, []patient.Record{other, same}, false)
		if d.Outcome != patient.Link || d.Record.ID != same.ID {
			t.Errorf("decision = %+v, want a link to %s", d, same.ID)
		}
	})
	t.Run("same name and birth and the same email: link silently, even to a record with a national ID", func(t *testing.T) {
		r := rec("Sam", "Smith", birth, nhsNo, "", "sam@example.test")
		d := patient.Decide(with(func(d *patient.Details) { d.Email = "sam@example.test" }), nil, []patient.Record{r}, false)
		if d.Outcome != patient.Link || d.Record.ID != r.ID {
			t.Errorf("decision = %+v, want a link to %s", d, r.ID)
		}
	})
	t.Run("same name and birth, different contacts: a possible duplicate", func(t *testing.T) {
		r := rec("Sam", "Smith", birth, "", "+447700900001", "sam@example.test")
		d := patient.Decide(with(func(d *patient.Details) { d.Mobile = "+447700900009"; d.Email = "other@example.test" }), nil, []patient.Record{r}, false)
		if d.Outcome != patient.PossibleDuplicate || len(d.Matches) != 1 || d.Matches[0].ID != r.ID {
			t.Errorf("decision = %+v, want a possible duplicate of %s", d, r.ID)
		}
	})
	t.Run("same name and birth and no contacts typed: a possible duplicate (empty never matches empty)", func(t *testing.T) {
		r := rec("Sam", "Smith", birth, "", "", "")
		if d := patient.Decide(typed, nil, []patient.Record{r}, false); d.Outcome != patient.PossibleDuplicate {
			t.Errorf("outcome = %v, want PossibleDuplicate", d.Outcome)
		}
	})
	t.Run("a possible duplicate with Create anyway: create", func(t *testing.T) {
		r := rec("Sam", "Smith", birth, "", "", "")
		if d := patient.Decide(typed, nil, []patient.Record{r}, true); d.Outcome != patient.Create {
			t.Errorf("outcome = %v, want Create", d.Outcome)
		}
	})
	t.Run("a different first name or surname on the same day is someone else", func(t *testing.T) {
		rs := []patient.Record{rec("Sally", "Smith", birth, "", "+447700900002", ""), rec("Sam", "Smyth", birth, "", "+447700900002", "")}
		if d := patient.Decide(with(func(d *patient.Details) { d.Mobile = "+447700900002" }), nil, rs, false); d.Outcome != patient.Create {
			t.Errorf("outcome = %v, want Create", d.Outcome)
		}
	})
	t.Run("a typed national ID the trust doesn't have: records with another national ID are someone else", func(t *testing.T) {
		r := rec("Sam", "Smith", birth, another, "+447700900002", "")
		if d := patient.Decide(with(func(d *patient.Details) { d.NationalID = nhsNo; d.Mobile = "+447700900002" }), nil, []patient.Record{r}, false); d.Outcome != patient.Create {
			t.Errorf("outcome = %v, want Create", d.Outcome)
		}
	})
	t.Run("a typed national ID the trust doesn't have: a record without one is matched by name, birth and contact", func(t *testing.T) {
		r := rec("Sam", "Smith", birth, "", "+447700900002", "")
		d := patient.Decide(with(func(d *patient.Details) { d.NationalID = nhsNo; d.Mobile = "+447700900002" }), nil, []patient.Record{r}, false)
		if d.Outcome != patient.Link || d.Record.ID != r.ID {
			t.Errorf("decision = %+v, want a link to %s", d, r.ID)
		}
		if d := patient.Decide(with(func(d *patient.Details) { d.NationalID = nhsNo }), nil, []patient.Record{r}, false); d.Outcome != patient.PossibleDuplicate {
			t.Errorf("without a contact: outcome = %v, want PossibleDuplicate", d.Outcome)
		}
	})
}

func TestParseSearch(t *testing.T) {
	nhs, _ := nationalid.Lookup("nhs")
	ok := func(q, dob string) patient.Search {
		t.Helper()
		s, msg := patient.ParseSearch(q, dob, nhs)
		if msg != "" {
			t.Fatalf("ParseSearch(%q, %q): %s", q, dob, msg)
		}
		return s
	}
	if s := ok("943 476 5919", ""); s.NationalID != nhsNo || s.Term != "943 476 5919" {
		t.Errorf("a valid NHS number: %+v", s)
	}
	if s := ok("9434765918", ""); s.NationalID != "" || s.Term != "9434765918" {
		t.Errorf("a bad check digit is only a hospital number or surname: %+v", s)
	}
	if s := ok(" smi ", ""); s.Term != "smi" || s.NationalID != "" || s.Birth != nil {
		t.Errorf("3 letters of a surname: %+v", s)
	}
	if s := ok("Li", "1980-05-17"); s.Term != "Li" || s.Birth == nil || !s.Birth.Equal(birth) {
		t.Errorf("a short surname with a date of birth: %+v", s)
	}
	for _, bad := range [][2]string{
		{"", ""}, {"sm", ""}, {"  ab ", ""}, // too broad
		{"", "1980-05-17"},             // a date of birth alone lists too many
		{"Smith", "17/05/1980"},        // not a date
		{strings.Repeat("x", 101), ""}, // too long
	} {
		if _, msg := patient.ParseSearch(bad[0], bad[1], nhs); msg == "" {
			t.Errorf("ParseSearch(%q, %q) accepted", bad[0], bad[1])
		}
	}
}
