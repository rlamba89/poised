//go:build integration

package main

import (
	"os"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

func TestMain(m *testing.M) {
	if os.Getenv("QA_USERS_RUN_MAIN") == "1" { // TestRefusesOtherStages's child: no database
		os.Exit(m.Run())
	}
	os.Exit(apitest.Main(m))
}

// Only the seed's accounts (fixed ids) get test sign-ins, never accounts testers or journeys made.
func TestSeededStaffAreTheSeedsAccountsOnly(t *testing.T) {
	c, fx := apitest.New(t)
	if _, err := seededStaff(t.Context(), c.DB); err == nil {
		t.Error("no seeded staff: want an error saying to seed first")
	}
	fx.User(t, fx.Hospital(t), "clinician") // a random id, like an invited account
	if _, err := c.DB.Exec(t.Context(), `INSERT INTO users (id, name, email) VALUES
		('00000000-0000-4000-8000-0000000000a3', 'Cara Clinician', 'Cara.Clinician@hospital-a.example')`); err != nil {
		t.Fatal(err)
	}
	got, err := seededStaff(t.Context(), c.DB)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (person{"Cara Clinician", "cara.clinician@hospital-a.example"}) {
		t.Errorf("got %+v", got)
	}
}
