package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/role"
)

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"ann@hospital.example":        "ann@hospital.example",
		"  Ann.Lee@Hospital.Example ": "ann.lee@hospital.example",
		"a+tag@nhs.net":               "a+tag@nhs.net",
	} {
		if got, msg := normalizeEmail(in); got != want || msg != "" {
			t.Errorf("normalizeEmail(%q) = %q, %q; want %q", in, got, msg, want)
		}
	}
	for _, in := range []string{"", "   ", "ann", "ann@", "@x.org", "ann@localhost", "ann@x.", "ann@.org",
		"a@b@x.org", "Ann <ann@x.org>", "ann @x.org", "ann@x.org,bob@x.org"} {
		if got, msg := normalizeEmail(in); msg == "" {
			t.Errorf("normalizeEmail(%q) = %q, accepted", in, got)
		}
	}
}

func TestStaffAccessParse(t *testing.T) {
	id := uuid.New()
	quoted, _ := json.Marshal(id)
	h, r, msg := staffAccess{HospitalID: quoted, Role: "super_clinician"}.parse()
	if msg != "" || !h.Valid || h.UUID != id || r != role.SuperClinician {
		t.Errorf("one hospital: %v %v %q", h, r, msg)
	}
	h, r, msg = staffAccess{HospitalID: json.RawMessage("null"), Role: "admin"}.parse()
	if msg != "" || h.Valid || r != role.Admin {
		t.Errorf("the whole trust: %v %v %q", h, r, msg)
	}
	for name, a := range map[string]staffAccess{
		"no scope":       {Role: "admin"},
		"a bad id":       {HospitalID: json.RawMessage(`"abc"`), Role: "admin"},
		"a number":       {HospitalID: json.RawMessage(`1`), Role: "admin"},
		"no role":        {HospitalID: json.RawMessage("null")},
		"an old role":    {HospitalID: json.RawMessage("null"), Role: "viewer"},
		"the wrong case": {HospitalID: json.RawMessage("null"), Role: "Admin"},
	} {
		if _, _, msg := a.parse(); msg == "" {
			t.Errorf("%s: accepted", name)
		}
	}
}
