//go:build integration

package apitest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// Fixtures insert rows straight into the test's database, so a test sets up only what it
// needs without going through other routes. Every name gets a random suffix.
type Fixtures struct {
	db *pgxpool.Pool
}

// User is a signed-in person, for LoginAs and the created_by columns.
type User struct {
	ID   uuid.UUID
	Name string
}

// HQ is a questionnaire with one version, made from a file in testdata/.
type HQ struct {
	Hospital  uuid.UUID
	ID        uuid.UUID // the questionnaire
	Name      string
	VersionID uuid.UUID
	Chapters  []uuid.UUID // the Question Sets, in order
}

// Episode is a patient's episode on a published HQ, still waiting for the patient.
type Episode struct {
	ID        uuid.UUID
	PatientID uuid.UUID
	Token     string // the patient's link: /api/p/{Token}
}

func (f *Fixtures) Hospital(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	f.scan(t, &id, `INSERT INTO hospitals (name) VALUES ($1) RETURNING id`, "Hospital "+randomHex(3))
	return id
}

// User adds a user with these roles in the hospital. With no roles, they belong nowhere.
func (f *Fixtures) User(t *testing.T, hospital uuid.UUID, roles ...string) User {
	t.Helper()
	suffix := randomHex(3)
	u := User{Name: "User " + suffix}
	f.scan(t, &u.ID, `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id`, u.Name, suffix+"@test.example")
	for _, role := range roles {
		f.exec(t, `INSERT INTO memberships (user_id, hospital_id, role) VALUES ($1, $2, $3)`, u.ID, hospital, role)
	}
	return u
}

// DraftHQ adds a questionnaire whose only version, 1, is a draft.
func (f *Fixtures) DraftHQ(t *testing.T, hospital uuid.UUID, by User, file string) HQ {
	t.Helper()
	return f.hq(t, hospital, by, "draft", file)
}

// PublishedHQ adds a questionnaire whose only version, 1, is published.
func (f *Fixtures) PublishedHQ(t *testing.T, hospital uuid.UUID, by User, file string) HQ {
	t.Helper()
	return f.hq(t, hospital, by, "published", file)
}

// hq reads testdata/<file>: {"name", "description", "chapters": [{"name", "audience", "content"}]},
// where each content is a Question Set's SurveyJS JSON.
func (f *Fixtures) hq(t *testing.T, hospital uuid.UUID, by User, status, file string) HQ {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(module, "internal", "apitest", "testdata", file))
	if err != nil {
		t.Fatalf("apitest: %v", err)
	}
	var in struct {
		Name, Description string
		Chapters          []struct {
			Name, Audience string
			Content        json.RawMessage
		}
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		t.Fatalf("apitest: %s: %v", file, err)
	}
	hq := HQ{Hospital: hospital, Name: in.Name + " " + randomHex(3)}
	f.scan(t, &hq.ID, `INSERT INTO questionnaires (hospital_id, name, description, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id`, hospital, hq.Name, in.Description, by.ID)
	f.scan(t, &hq.VersionID, `INSERT INTO questionnaire_versions (questionnaire_id, version_no, status, updated_by)
		VALUES ($1, 1, $2, $3) RETURNING id`, hq.ID, status, by.ID)
	for i, ch := range in.Chapters {
		var id uuid.UUID
		f.scan(t, &id, `INSERT INTO chapters (version_id, position, name, audience, content, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, hq.VersionID, i+1, ch.Name, ch.Audience, ch.Content, by.ID)
		hq.Chapters = append(hq.Chapters, id)
	}
	return hq
}

// Episode adds a made-up patient in the HQ's hospital and an episode on the HQ, which
// should be published.
func (f *Fixtures) Episode(t *testing.T, hq HQ, by User) Episode {
	t.Helper()
	token, err := episode.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	e := Episode{Token: token}
	f.scan(t, &e.PatientID, `INSERT INTO patients (hospital_id, first_name, last_name, date_of_birth, sex)
		VALUES ($1, 'Test', $2, '1970-01-01', 'unknown') RETURNING id`, hq.Hospital, "Patient "+randomHex(3))
	f.scan(t, &e.ID, `INSERT INTO episodes (hospital_id, patient_id, version_id, patient_token, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`, hq.Hospital, e.PatientID, hq.VersionID, token, by.ID)
	return e
}

func (f *Fixtures) scan(t *testing.T, dest any, sql string, args ...any) {
	t.Helper()
	if err := f.db.QueryRow(context.Background(), sql, args...).Scan(dest); err != nil {
		t.Fatalf("apitest: %v\n%s", err, sql)
	}
}

func (f *Fixtures) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("apitest: %v\n%s", err, sql)
	}
}
