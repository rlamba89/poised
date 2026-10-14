//go:build integration

package main

import (
	"context"
	"os"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }

// The pipeline runs the seed after every deploy to QA (plan F4 Step 2), so it must be safe to run
// twice: the second run must succeed and leave exactly the same rows, and it must put back what a
// tester changed in the demo data.
func TestSeedTwice(t *testing.T) {
	c, _ := apitest.New(t)
	ctx := context.Background()

	if err := seed(ctx, c.DB, "../../db/seed/lifebox"); err != nil {
		t.Fatalf("first run: %v", err)
	}
	first := counts(t, c)
	for table, n := range first {
		if n == 0 {
			t.Errorf("the first run left %s empty", table)
		}
	}

	// A tester suspends Trust A and renames a demo user between runs.
	if _, err := c.DB.Exec(ctx, `UPDATE orgs SET status = 'suspended' WHERE code = 'TA'`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(ctx, `UPDATE users SET name = 'Changed' WHERE email = 'cara.clinician@hospital-a.example'`); err != nil {
		t.Fatal(err)
	}

	if err := seed(ctx, c.DB, "../../db/seed/lifebox"); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second := counts(t, c)
	for table, n := range first {
		if second[table] != n {
			t.Errorf("%s: %d rows after the first run, %d after the second", table, n, second[table])
		}
	}

	var status, name string
	if err := c.DB.QueryRow(ctx, `SELECT status FROM orgs WHERE code = 'TA'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Errorf("Trust A is %q after the second run, want active", status)
	}
	if err := c.DB.QueryRow(ctx, `SELECT name FROM users WHERE email = 'cara.clinician@hospital-a.example'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Cara Clinician" {
		t.Errorf("Cara is named %q after the second run", name)
	}
}

func counts(t *testing.T, c *apitest.Client) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{"categories", "codes", "orgs", "hospitals", "users", "memberships", "platform_admins", "patients"} {
		var n int
		if err := c.DB.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		out[table] = n
	}
	return out
}
