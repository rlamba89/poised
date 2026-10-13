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

// Hospital is a hospital and its trust. Its routes are under Path.
type Hospital struct {
	Org uuid.UUID
	ID  uuid.UUID
}

// Path is a route of the hospital: /api/o/{org}/h/{id} + p.
func (h Hospital) Path(p string) string {
	return "/api/o/" + h.Org.String() + "/h/" + h.ID.String() + p
}

// Org adds a trust.
func (f *Fixtures) Org(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	suffix := randomHex(3)
	f.scan(t, &id, `INSERT INTO orgs (name, code) VALUES ($1, $2) RETURNING id`, "Trust "+suffix, "T"+suffix)
	return id
}

// Hospital adds a trust with one hospital: another hospital is also another trust.
func (f *Fixtures) Hospital(t *testing.T) Hospital {
	t.Helper()
	return f.HospitalIn(t, f.Org(t))
}

// HospitalIn adds a hospital to a trust.
func (f *Fixtures) HospitalIn(t *testing.T, org uuid.UUID) Hospital {
	t.Helper()
	h := Hospital{Org: org}
	f.scan(t, &h.ID, `INSERT INTO hospitals (org_id, name) VALUES ($1, $2) RETURNING id`, org, "Hospital "+randomHex(3))
	return h
}

// User adds a user with this role at the hospital, or with no membership when role is "".
func (f *Fixtures) User(t *testing.T, h Hospital, role string) User {
	t.Helper()
	suffix := randomHex(3)
	u := User{Name: "User " + suffix}
	f.scan(t, &u.ID, `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id`, u.Name, suffix+"@test.example")
	if role != "" {
		f.Member(t, u, h, role)
	}
	return u
}

// TrustUser adds a user with this role in the whole trust.
func (f *Fixtures) TrustUser(t *testing.T, org uuid.UUID, role string) User {
	t.Helper()
	u := f.User(t, Hospital{}, "")
	f.TrustMember(t, u, org, role)
	return u
}

// PlatformAdmin adds a superadmin (A-21): a platform_admins row and no memberships.
func (f *Fixtures) PlatformAdmin(t *testing.T) User {
	t.Helper()
	u := f.User(t, Hospital{}, "")
	f.exec(t, `INSERT INTO platform_admins (user_id) VALUES ($1)`, u.ID)
	return u
}

// Member gives u a role at one hospital.
func (f *Fixtures) Member(t *testing.T, u User, h Hospital, role string) {
	t.Helper()
	f.exec(t, `INSERT INTO memberships (user_id, org_id, hospital_id, role) VALUES ($1, $2, $3, $4)`, u.ID, h.Org, h.ID, role)
}

// TrustMember gives u a role in the whole trust.
func (f *Fixtures) TrustMember(t *testing.T, u User, org uuid.UUID, role string) {
	t.Helper()
	f.exec(t, `INSERT INTO memberships (user_id, org_id, role) VALUES ($1, $2, $3)`, u.ID, org, role)
}

// HQ is a questionnaire with one version, made from a file in testdata/. It's the trust's, so
// every hospital of the trust uses it; Hospital is where Episode puts its episodes.
type HQ struct {
	Hospital  Hospital
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

// DraftHQ adds a questionnaire whose only version, 1, is a draft.
func (f *Fixtures) DraftHQ(t *testing.T, hospital Hospital, by User, file string) HQ {
	t.Helper()
	return f.hq(t, hospital, by, "draft", file)
}

// PublishedHQ adds a questionnaire whose only version, 1, is published.
func (f *Fixtures) PublishedHQ(t *testing.T, hospital Hospital, by User, file string) HQ {
	t.Helper()
	return f.hq(t, hospital, by, "published", file)
}

// hq reads testdata/<file>: {"name", "description", "chapters": [{"name", "audience", "content"}]},
// where each content is a Question Set's SurveyJS JSON.
func (f *Fixtures) hq(t *testing.T, hospital Hospital, by User, status, file string) HQ {
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
	f.scan(t, &hq.ID, `INSERT INTO questionnaires (org_id, name, description, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id`, hospital.Org, hq.Name, in.Description, by.ID)
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

// Episode adds a made-up patient in the HQ's trust and an episode on the HQ, which should be
// published, at the HQ's hospital.
func (f *Fixtures) Episode(t *testing.T, hq HQ, by User) Episode {
	t.Helper()
	return f.EpisodeAt(t, hq, hq.Hospital, by)
}

// EpisodeAt is Episode at another hospital of the HQ's trust.
func (f *Fixtures) EpisodeAt(t *testing.T, hq HQ, h Hospital, by User) Episode {
	t.Helper()
	token, err := episode.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	e := Episode{Token: token}
	f.scan(t, &e.PatientID, `INSERT INTO patients (org_id, first_name, last_name, date_of_birth, sex)
		VALUES ($1, 'Test', $2, '1970-01-01', 'unknown') RETURNING id`, h.Org, "Patient "+randomHex(3))
	f.scan(t, &e.ID, `INSERT INTO episodes (org_id, hospital_id, patient_id, version_id, patient_token, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, h.Org, h.ID, e.PatientID, hq.VersionID, token, by.ID)
	return e
}

// PatientRecord is a made-up patient's details, for Patient. Dates are YYYY-MM-DD; a NationalID
// is an NHS number (the only scheme built), and the mobile is in E.164.
type PatientRecord struct {
	FirstName, LastName, DateOfBirth, NationalID, Mobile, Email string
}

// Patient adds a trust's patient record, with no episodes.
func (f *Fixtures) Patient(t *testing.T, org uuid.UUID, p PatientRecord) uuid.UUID {
	t.Helper()
	var id, typ *string
	if p.NationalID != "" {
		nhs := "nhs"
		id, typ = &p.NationalID, &nhs
	}
	var pid uuid.UUID
	f.scan(t, &pid, `INSERT INTO patients (org_id, first_name, last_name, date_of_birth, sex, national_id, national_id_type, mobile, email)
		VALUES ($1, $2, $3, $4, 'unknown', $5, $6, $7, $8) RETURNING id`,
		org, p.FirstName, p.LastName, p.DateOfBirth, id, typ, p.Mobile, p.Email)
	return pid
}

// EpisodeOf adds an episode on the HQ for an existing patient at a hospital of the HQ's trust,
// with a hospital number ("" for none).
func (f *Fixtures) EpisodeOf(t *testing.T, hq HQ, h Hospital, patient uuid.UUID, by User, hospitalNumber string) Episode {
	t.Helper()
	token, err := episode.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	e := Episode{Token: token, PatientID: patient}
	f.scan(t, &e.ID, `INSERT INTO episodes (org_id, hospital_id, patient_id, version_id, patient_token, created_by, hospital_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, h.Org, h.ID, patient, hq.VersionID, token, by.ID, hospitalNumber)
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
