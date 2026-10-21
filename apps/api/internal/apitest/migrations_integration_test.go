//go:build integration

package apitest_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// oldShape is a database at migration 18, before episode forms (S18), holding one trust's made-up
// data: a published HQ, a patient with an account, and an episode with everything an episode
// has (answers, General notes, a message, a follow-up job, an invite link and a session opened
// with it), plus the account's own sign-in link and its session, which aren't the episode's.
const oldShape = `
INSERT INTO orgs (id, name, code) VALUES ('00000000-0000-0000-0000-00000000000a', 'Trust', 'MIG');
INSERT INTO hospitals (id, org_id, name) VALUES ('00000000-0000-0000-0000-00000000000b', '00000000-0000-0000-0000-00000000000a', 'Hospital');
INSERT INTO users (id, name, email) VALUES ('00000000-0000-0000-0000-00000000000c', 'Cara', 'cara@example.test');
INSERT INTO questionnaires (id, org_id, name, description, created_by)
VALUES ('00000000-0000-0000-0000-00000000000d', '00000000-0000-0000-0000-00000000000a', 'HQ', '', '00000000-0000-0000-0000-00000000000c');
INSERT INTO questionnaire_versions (id, questionnaire_id, version_no, status, updated_by)
VALUES ('00000000-0000-0000-0000-00000000000e', '00000000-0000-0000-0000-00000000000d', 1, 'published', '00000000-0000-0000-0000-00000000000c');
INSERT INTO chapters (id, version_id, position, name, updated_by)
VALUES ('00000000-0000-0000-0000-00000000000f', '00000000-0000-0000-0000-00000000000e', 1, 'About you', '00000000-0000-0000-0000-00000000000c');
INSERT INTO patient_accounts (id, mobile, date_of_birth, first_name_key, last_name_key)
VALUES ('00000000-0000-0000-0000-000000000010', '+447700900001', '1970-01-01', 'sam', 'smith');
INSERT INTO patients (id, org_id, first_name, last_name, date_of_birth, sex, mobile, account_id)
VALUES ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-00000000000a', 'Sam', 'Smith', '1970-01-01', 'unknown',
        '+447700900001', '00000000-0000-0000-0000-000000000010');
INSERT INTO episodes (id, org_id, hospital_id, patient_id, version_id, status, patient_submitted_at, created_by)
VALUES ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-00000000000a', '00000000-0000-0000-0000-00000000000b',
        '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-00000000000e', 'ready_for_review', now(),
        '00000000-0000-0000-0000-00000000000c');
INSERT INTO episode_answers (episode_id, chapter_id, actor, data, revision)
VALUES ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-00000000000f', 'patient', '{"q": 1}', 3);
INSERT INTO episode_events (episode_id, kind, text) VALUES ('00000000-0000-0000-0000-000000000012', 'event', 'Episode created');
INSERT INTO episode_messages (episode_id, kind, channel, created_at) VALUES ('00000000-0000-0000-0000-000000000012', 'invite', 'sms', now());
INSERT INTO jobs (kind, episode_id, run_at) VALUES ('followup', '00000000-0000-0000-0000-000000000012', now());
INSERT INTO login_links (id, token_hash, kind, patient_id, episode_id)
VALUES ('00000000-0000-0000-0000-000000000013', '\x01', 'invite', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000012');
INSERT INTO patient_sessions (link_id, account_id, created_at, last_seen_at)
VALUES ('00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000010', now(), now());
INSERT INTO login_links (id, token_hash, kind, contact, expires_at)
VALUES ('00000000-0000-0000-0000-000000000014', '\x02', 'signin', '+447700900001', now() + interval '1 hour');
INSERT INTO patient_sessions (link_id, account_id, created_at, last_seen_at)
VALUES ('00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000010', now(), now());
`

// S18 reshapes the answers onto episode forms with no data migration (S18 answers 2, Rahul 20
// Oct): the migration removes the made-up episodes, and the seed puts back the demo data. It must
// still run on a database that holds episodes (QA, every developer's), and leave everything else.
func TestEpisodeFormsMigration(t *testing.T) {
	t.Parallel()
	m := apitest.NewMigrations(t, 18)
	if _, err := m.DB.Exec(t.Context(), oldShape); err != nil {
		t.Fatalf("old shape: %v", err)
	}
	m.Up(t)

	count := func(t *testing.T, sql string) int {
		t.Helper()
		var n int
		if err := m.DB.QueryRow(t.Context(), sql).Scan(&n); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}
	t.Run("the episodes and everything of theirs are gone", func(t *testing.T) {
		for _, table := range []string{"episodes", "episode_forms", "episode_answers", "episode_events", "episode_messages", "jobs"} {
			if n := count(t, "SELECT count(*) FROM "+table); n != 0 {
				t.Errorf("%s: %d rows, want 0", table, n)
			}
		}
		if n := count(t, "SELECT count(*) FROM login_links WHERE kind = 'invite'"); n != 0 {
			t.Errorf("invite links: %d, want 0", n)
		}
	})
	t.Run("everything else is kept", func(t *testing.T) {
		for table, want := range map[string]int{"orgs": 1, "hospitals": 1, "users": 1, "questionnaire_versions": 1, "chapters": 1,
			"patients": 1, "patient_accounts": 1, "login_links": 1, "patient_sessions": 1} {
			if n := count(t, "SELECT count(*) FROM "+table); n != want {
				t.Errorf("%s: %d rows, want %d", table, n, want)
			}
		}
	})
	t.Run("the old columns are gone", func(t *testing.T) {
		if n := count(t, `SELECT count(*) FROM information_schema.columns WHERE table_name = 'episodes'
			AND column_name IN ('version_id', 'patient_submitted_at', 'patient_submitted_by')`); n != 0 {
			t.Errorf("episodes still has %d of the columns that moved to its form", n)
		}
		if n := count(t, `SELECT count(*) FROM information_schema.columns WHERE table_name = 'episode_answers'
			AND column_name = 'episode_id'`); n != 0 {
			t.Error("episode_answers is still keyed by the episode")
		}
	})

	// The new shape: an episode, its HQ form, and the answers on the form.
	ep, form := uuid.New(), uuid.New()
	if _, err := m.DB.Exec(t.Context(), `
		INSERT INTO episodes (id, org_id, hospital_id, patient_id, created_by)
		VALUES ($1, '00000000-0000-0000-0000-00000000000a', '00000000-0000-0000-0000-00000000000b',
		        '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-00000000000c')`, ep); err != nil {
		t.Fatalf("episode: %v", err)
	}
	if _, err := m.DB.Exec(t.Context(), `INSERT INTO episode_forms (id, episode_id, version_id, kind)
		VALUES ($1, $2, '00000000-0000-0000-0000-00000000000e', 'hq')`, form, ep); err != nil {
		t.Fatalf("form: %v", err)
	}
	if _, err := m.DB.Exec(t.Context(), `INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data)
		VALUES ($1, '00000000-0000-0000-0000-00000000000f', 'patient', '{}')`, form); err != nil {
		t.Fatalf("answers: %v", err)
	}
	t.Run("a new form is open, not submitted, with no dates", func(t *testing.T) {
		if n := count(t, `SELECT count(*) FROM episode_forms WHERE status = 'open' AND submitted_at IS NULL
			AND submitted_by IS NULL AND due_at IS NULL AND closes_at IS NULL`); n != 1 {
			t.Error("the new form isn't open with no dates")
		}
	})
	refused := map[string]string{
		"a second HQ form": `INSERT INTO episode_forms (episode_id, version_id, kind)
			VALUES ('` + ep.String() + `', '00000000-0000-0000-0000-00000000000e', 'hq')`,
		"an unknown kind": `INSERT INTO episode_forms (episode_id, version_id, kind)
			VALUES ('` + ep.String() + `', '00000000-0000-0000-0000-00000000000e', 'letter')`,
		"submitted with no time":   `UPDATE episode_forms SET status = 'submitted' WHERE id = '` + form.String() + `'`,
		"a submit time while open": `UPDATE episode_forms SET submitted_at = now() WHERE id = '` + form.String() + `'`,
		"an answers row twice": `INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data)
			VALUES ('` + form.String() + `', '00000000-0000-0000-0000-00000000000f', 'patient', '{}')`,
	}
	for name, sql := range refused {
		t.Run("refused: "+name, func(t *testing.T) {
			if _, err := m.DB.Exec(t.Context(), sql); err == nil {
				t.Error("the database allowed it")
			}
		})
	}
	t.Run("an assessment and PROMs can join the HQ (C7, C9)", func(t *testing.T) {
		if _, err := m.DB.Exec(t.Context(), `INSERT INTO episode_forms (episode_id, version_id, kind)
			VALUES ($1, '00000000-0000-0000-0000-00000000000e', 'assessment'),
			       ($1, '00000000-0000-0000-0000-00000000000e', 'proms')`, ep); err != nil {
			t.Error(err)
		}
	})
	t.Run("deleting the episode deletes its forms and answers", func(t *testing.T) {
		if _, err := m.DB.Exec(t.Context(), `DELETE FROM episodes WHERE id = $1`, ep); err != nil {
			t.Fatal(err)
		}
		if n := count(t, "SELECT count(*) FROM episode_forms") + count(t, "SELECT count(*) FROM episode_answers"); n != 0 {
			t.Errorf("%d form and answers rows left", n)
		}
	})
	t.Run("it rolls back, and forward again", func(t *testing.T) {
		m.DownTo(t, 18)
		if n := count(t, `SELECT count(*) FROM information_schema.columns WHERE table_name = 'episode_answers'
			AND column_name = 'episode_id'`); n != 1 {
			t.Error("rolled back, episode_answers isn't keyed by the episode")
		}
		m.Up(t)
	})
}

// S19 replaces the episode's free-text procedure with one from the trust's list (S19 answers 4a,
// Rahul 20 Oct): each trust's distinct texts become its procedures, trimmed and in any case once,
// and its episodes are linked to them. Episodes made since migration 00019 keep their procedure.
func TestProceduresMigration(t *testing.T) {
	t.Parallel()
	m := apitest.NewMigrations(t, 19)
	const (
		trustA, trustB = "00000000-0000-0000-0000-0000000000a0", "00000000-0000-0000-0000-0000000000b0"
		hosA, hosB     = "00000000-0000-0000-0000-0000000000a1", "00000000-0000-0000-0000-0000000000b1"
		user           = "00000000-0000-0000-0000-0000000000c0"
	)
	setup := `
INSERT INTO orgs (id, name, code) VALUES ('` + trustA + `', 'Trust A', 'MA'), ('` + trustB + `', 'Trust B', 'MB');
INSERT INTO hospitals (id, org_id, name) VALUES ('` + hosA + `', '` + trustA + `', 'A1'), ('` + hosB + `', '` + trustB + `', 'B1');
INSERT INTO users (id, name, email) VALUES ('` + user + `', 'Cara', 'cara@example.test');
INSERT INTO patients (id, org_id, first_name, last_name, date_of_birth, sex)
VALUES ('00000000-0000-0000-0000-0000000000d0', '` + trustA + `', 'Sam', 'Smith', '1970-01-01', 'unknown'),
       ('00000000-0000-0000-0000-0000000000d1', '` + trustB + `', 'Jo', 'Jones', '1970-01-01', 'unknown');
INSERT INTO episodes (id, org_id, hospital_id, patient_id, procedure, created_by) VALUES
  ('00000000-0000-0000-0000-0000000000e0', '` + trustA + `', '` + hosA + `', '00000000-0000-0000-0000-0000000000d0', 'Hip replacement', '` + user + `'),
  ('00000000-0000-0000-0000-0000000000e1', '` + trustA + `', '` + hosA + `', '00000000-0000-0000-0000-0000000000d0', ' hip REPLACEMENT ', '` + user + `'),
  ('00000000-0000-0000-0000-0000000000e2', '` + trustA + `', '` + hosA + `', '00000000-0000-0000-0000-0000000000d0', 'Knee scope', '` + user + `'),
  ('00000000-0000-0000-0000-0000000000e3', '` + trustA + `', '` + hosA + `', '00000000-0000-0000-0000-0000000000d0', '  ', '` + user + `'),
  ('00000000-0000-0000-0000-0000000000e4', '` + trustB + `', '` + hosB + `', '00000000-0000-0000-0000-0000000000d1', 'Hip replacement', '` + user + `');
`
	if _, err := m.DB.Exec(t.Context(), setup); err != nil {
		t.Fatalf("old shape: %v", err)
	}
	m.UpTo(t, 20)

	procedures := func(t *testing.T, org string) []string {
		t.Helper()
		rows, err := m.DB.Query(t.Context(), `SELECT name || '|' || code || '|' || specialty || '|' || coalesce(hospital_id::text, '') || '|' ||
			coalesce(default_questionnaire_id::text, '') || '|' || retired || '|' || info_content::text || '|' || proms_plan::text
			FROM procedures WHERE org_id = $1 ORDER BY lower(name)`, org)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatal(err)
			}
			out = append(out, s)
		}
		return out
	}
	t.Run("each trust's distinct texts are its procedures, for every hospital, with no default HQ", func(t *testing.T) {
		if got, want := procedures(t, trustA), []string{"Hip replacement|||||false|[]|[]", "Knee scope|||||false|[]|[]"}; !slices.Equal(got, want) {
			t.Errorf("trust A = %q, want %q", got, want)
		}
		if got, want := procedures(t, trustB), []string{"Hip replacement|||||false|[]|[]"}; !slices.Equal(got, want) {
			t.Errorf("trust B = %q, want %q", got, want)
		}
	})
	t.Run("each episode names its trust's procedure; a blank one names none", func(t *testing.T) {
		rows, err := m.DB.Query(t.Context(), `SELECT e.id::text || '=' || coalesce(p.name, '') || coalesce('@' || p.org_id::text, '')
			FROM episodes e LEFT JOIN procedures p ON p.id = e.procedure_id ORDER BY e.id`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var got []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatal(err)
			}
			got = append(got, s)
		}
		want := []string{
			"00000000-0000-0000-0000-0000000000e0=Hip replacement@" + trustA,
			"00000000-0000-0000-0000-0000000000e1=Hip replacement@" + trustA,
			"00000000-0000-0000-0000-0000000000e2=Knee scope@" + trustA,
			"00000000-0000-0000-0000-0000000000e3=",
			"00000000-0000-0000-0000-0000000000e4=Hip replacement@" + trustB,
		}
		if !slices.Equal(got, want) {
			t.Errorf("episodes = %q, want %q", got, want)
		}
	})
	t.Run("the free text is gone, and the dates start empty", func(t *testing.T) {
		var n int
		if err := m.DB.QueryRow(t.Context(), `SELECT count(*) FROM information_schema.columns
			WHERE table_name = 'episodes' AND column_name = 'procedure'`).Scan(&n); err != nil || n != 0 {
			t.Errorf("episodes.procedure columns = %d (%v), want 0", n, err)
		}
		if err := m.DB.QueryRow(t.Context(), `SELECT count(*) FROM episodes WHERE planned_date IS NOT NULL OR poa_date IS NOT NULL`).Scan(&n); err != nil || n != 0 {
			t.Errorf("episodes with dates = %d (%v), want 0", n, err)
		}
	})
	t.Run("a name is unique in the trust in any case, and the lists must be lists", func(t *testing.T) {
		if _, err := m.DB.Exec(t.Context(), `INSERT INTO procedures (org_id, name) VALUES ($1, 'KNEE SCOPE')`, trustA); err == nil {
			t.Error("a second 'Knee scope' in trust A was allowed")
		}
		if _, err := m.DB.Exec(t.Context(), `INSERT INTO procedures (org_id, name) VALUES ($1, 'Knee scope')`, trustB); err != nil {
			t.Errorf("trust B's own 'Knee scope': %v", err)
		}
		if _, err := m.DB.Exec(t.Context(), `INSERT INTO procedures (org_id, name, proms_plan) VALUES ($1, 'Odd', '{}')`, trustA); err == nil {
			t.Error("a PROMs plan that isn't a list was allowed")
		}
		if _, err := m.DB.Exec(t.Context(), `INSERT INTO procedures (org_id, hospital_id, name) VALUES ($1, $2, 'Elsewhere')`, trustA, hosB); err == nil {
			t.Error("a procedure for another trust's hospital was allowed")
		}
	})
	t.Run("down puts the text back, then up again", func(t *testing.T) {
		m.DownTo(t, 19)
		var text string
		if err := m.DB.QueryRow(t.Context(), `SELECT procedure FROM episodes WHERE id = '00000000-0000-0000-0000-0000000000e1'`).Scan(&text); err != nil || text != "Hip replacement" {
			t.Errorf("procedure after down = %q (%v), want the procedure's name", text, err)
		}
		m.Up(t)
	})
}
