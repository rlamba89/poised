//go:build integration

package main

import (
	"context"
	"os"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/content"
)

func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }

// The pipeline runs the seed after every deploy to QA (plan F4 Step 2), so it must be safe to run
// twice: the second run must succeed and leave exactly the same rows, and it must put back what a
// tester changed in the demo data.
func TestSeedTwice(t *testing.T) {
	c, _ := apitest.New(t)
	ctx := context.Background()

	if err := seed(ctx, c.DB, "../../db/seed/codes"); err != nil {
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
	// …changes a platform library question, and adds one the seed doesn't have (S24)…
	if _, err := c.DB.Exec(ctx, `UPDATE library_questions SET title = 'Changed', tags = '{}' WHERE key = 'smoker' AND org_id IS NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(ctx, `INSERT INTO library_questions (key, title, element) VALUES ('q_gone', 'Gone', '{"type":"text","name":"q_gone"}')`); err != nil {
		t.Fatal(err)
	}
	// …retires a starter question and an example video, and renames another (S33)…
	if _, err := c.DB.Exec(ctx, `UPDATE library_questions SET retired = true WHERE key = 'smoker' AND org_id IS NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(ctx, `UPDATE content_items SET status = 'retired' WHERE id = $1`, exampleVideos[0].id); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(ctx, `UPDATE content_items SET title = 'Changed' WHERE id = $1`, exampleVideos[1].id); err != nil {
		t.Fatal(err)
	}
	// …and retires a demo procedure and clears its default HQ (S19).
	if _, err := c.DB.Exec(ctx, `UPDATE procedures SET retired = true, default_questionnaire_id = NULL WHERE name = 'Total hip replacement'`); err != nil {
		t.Fatal(err)
	}

	if err := seed(ctx, c.DB, "../../db/seed/codes"); err != nil {
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

	// S24: the platform's starter library is back as seeded: the 11 questions Rahul agreed, no others.
	var platform int
	var smoker string
	if err := c.DB.QueryRow(ctx, `SELECT count(*), max(title) FILTER (WHERE key = 'smoker') FROM library_questions WHERE org_id IS NULL`).Scan(&platform, &smoker); err != nil {
		t.Fatal(err)
	}
	if platform != 11 || smoker != "Do you smoke?" {
		t.Errorf("the platform library has %d questions (smoker: %q), want 11 as seeded", platform, smoker)
	}

	// S33: a retired starter question stays retired: the seed adds new ones, but a platform admin's
	// retirement is a real decision, not test data.
	var smokerRetired bool
	if err := c.DB.QueryRow(ctx, `SELECT retired FROM library_questions WHERE key = 'smoker' AND org_id IS NULL`).Scan(&smokerRetired); err != nil {
		t.Fatal(err)
	}
	if !smokerRetired {
		t.Error("the seed brought back a retired starter question")
	}
	// S33: the example videos are back as seeded: Preopsis's, active, YouTube or Vimeo links,
	// titled "Example: …", each with a length and a checklist tag.
	var examples, active, named int
	if err := c.DB.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE status = 'active' AND org_id IS NULL AND kind = 'video' AND length_seconds > 0),
			count(*) FILTER (WHERE title LIKE 'Example: %')
		FROM content_items`).Scan(&examples, &active, &named); err != nil {
		t.Fatal(err)
	}
	if examples != len(exampleVideos) || active != examples || named != examples || examples < 2 {
		t.Errorf("content items: %d, %d active platform videos, %d titled as examples; want the %d examples", examples, active, named, len(exampleVideos))
	}
	for _, v := range exampleVideos {
		if _, ok := content.CheckVideo(v.url); !ok {
			t.Errorf("example %q: %s isn't a YouTube or Vimeo link", v.title, v.url)
		}
	}

	// S19: each trust has a published demo HQ and a few procedures, most with it as their default
	// HQ, one for Hospital A1 only, none retired.
	var trustWide, a1Only, withDefault, retired, ofOtherTrust int
	if err := c.DB.QueryRow(ctx, `SELECT count(*) FILTER (WHERE p.hospital_id IS NULL), count(*) FILTER (WHERE h.name = 'Hospital A1'),
			count(*) FILTER (WHERE v.status = 'published'), count(*) FILTER (WHERE p.retired), count(*) FILTER (WHERE q.org_id <> p.org_id)
		FROM procedures p JOIN orgs o ON o.id = p.org_id LEFT JOIN hospitals h ON h.id = p.hospital_id
		LEFT JOIN questionnaires q ON q.id = p.default_questionnaire_id
		LEFT JOIN questionnaire_versions v ON v.questionnaire_id = q.id
		WHERE o.code = 'TA'`).Scan(&trustWide, &a1Only, &withDefault, &retired, &ofOtherTrust); err != nil {
		t.Fatal(err)
	}
	if trustWide < 3 || a1Only != 1 || withDefault < 3 || withDefault == trustWide+a1Only || retired != 0 || ofOtherTrust != 0 {
		t.Errorf("Trust A's procedures: %d trust-wide, %d A1-only, %d with a published default HQ, %d retired, %d with another trust's HQ",
			trustWide, a1Only, withDefault, retired, ofOtherTrust)
	}
	var bProcedures int
	if err := c.DB.QueryRow(ctx, `SELECT count(*) FROM procedures p JOIN orgs o ON o.id = p.org_id
		JOIN questionnaires q ON q.id = p.default_questionnaire_id AND q.org_id = p.org_id WHERE o.code = 'TB'`).Scan(&bProcedures); err != nil {
		t.Fatal(err)
	}
	if bProcedures < 1 {
		t.Errorf("Trust B has %d procedures with its own default HQ, want some", bProcedures)
	}
}

func counts(t *testing.T, c *apitest.Client) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{"categories", "codes", "orgs", "hospitals", "users", "memberships", "platform_admins", "patients",
		"questionnaires", "questionnaire_versions", "chapters", "procedures", "library_questions", "content_items"} {
		var n int
		if err := c.DB.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		out[table] = n
	}
	return out
}
