// Command seed loads the incumbent's categories and codes from db/seed/codes/*.csv,
// plus the demo trusts, hospitals and users. It is safe to run more than once.
package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Fixed ids keep the seed idempotent and the demo data recognisable.
const (
	unassignedCategoryID = "00000000-0000-4000-8000-000000000001"

	trustA     = "00000000-0000-4000-8000-0000000000fa"
	trustB     = "00000000-0000-4000-8000-0000000000fb"
	hospitalA1 = "00000000-0000-4000-8000-00000000000a" // was "Hospital A" before S08
	hospitalA2 = "00000000-0000-4000-8000-00000000002a"
	hospitalB1 = "00000000-0000-4000-8000-00000000000b" // was "Hospital B"
)

// The demo trusts (S08): Trust A with Hospitals A1 and A2, Trust B with Hospital B1.
// Each run puts them back to active, with these contacts and addresses and no settings of their own
// (Preopsis's defaults), so each test round starts the same (S10).
var demoTrusts = []struct {
	id, name, code string
	contact        [3]string   // name, email, phone
	hospitals      [][2]string // id, name
}{
	{trustA, "Trust A", "TA", [3]string{"Ada Admin", "ada.admin@trust-a.example", ""},
		[][2]string{{hospitalA1, "Hospital A1"}, {hospitalA2, "Hospital A2"}}},
	{trustB, "Trust B", "TB", [3]string{"Max Multi", "max.multi@trust-b.example", ""},
		[][2]string{{hospitalB1, "Hospital B1"}}},
}

// Each demo user and their memberships: a hospital of "" is the whole trust.
var demoUsers = []struct {
	id, name, email string
	memberships     []membership
	platformAdmin   bool
}{
	{"00000000-0000-4000-8000-0000000000a4", "Ada Admin", "ada.admin@trust-a.example",
		[]membership{{trustA, "", "admin"}}, false},
	{"00000000-0000-4000-8000-0000000000a1", "Alex Author", "alex.author@hospital-a.example",
		[]membership{{trustA, "", "super_clinician"}}, false},
	{"00000000-0000-4000-8000-0000000000a3", "Cara Clinician", "cara.clinician@hospital-a.example",
		[]membership{{trustA, hospitalA1, "clinician"}}, false},
	// A super clinician of one hospital only: sees the trust's questionnaires but can't change
	// them; only whole-trust staff can (S08 answers, 5).
	{"00000000-0000-4000-8000-0000000000a6", "Sue Super", "sue.super@trust-a.example",
		[]membership{{trustA, hospitalA1, "super_clinician"}}, false},
	{"00000000-0000-4000-8000-0000000000a2", "Val Clinician", "val.clinician@trust-a.example",
		[]membership{{trustA, hospitalA2, "clinician"}}, false},
	{"00000000-0000-4000-8000-0000000000b1", "Bea Author", "bea.author@hospital-b.example",
		[]membership{{trustB, "", "super_clinician"}}, false},
	{"00000000-0000-4000-8000-0000000000d1", "Max Multi", "max.multi@trust-b.example",
		[]membership{{trustA, hospitalA2, "clinician"}, {trustB, "", "admin"}}, false},
	{"00000000-0000-4000-8000-0000000000e1", "Sam Superadmin", "sam.superadmin@poised.example", nil, true},
}

type membership struct{ trust, hospital, role string }

// Made-up patients only (NFR-03), all Trust A's records, with no episodes: no hospital finds them
// by searching, but typing their details on the New episode page links to them silently (S13).
// NHS numbers are made up in the 999 range with valid check digits; mobiles are Ofcom's drama
// range (07700 900xxx). Pat has no NHS number, so name + date of birth + contact matches her.
var demoPatients = []struct{ id, first, last, dob, sex, nhs, mobile, email string }{
	{"00000000-0000-4000-8000-0000000000c1", "Jo", "Bloggs", "1974-12-11", "male", "9990001006", "+447700900001", "jo.bloggs@example.test"},
	{"00000000-0000-4000-8000-0000000000c2", "Pat", "Example", "1961-03-02", "female", "", "+447700900002", "pat.example@example.test"},
	{"00000000-0000-4000-8000-0000000000c3", "Robin", "Sample", "1990-07-21", "other", "9990001014", "", ""},
}

// The demo HQ (S19), one published copy per trust, so the demo procedures have a default HQ and
// a tester can create an episode without authoring one first. Made up, like the e2e journeys' HQ.
//
//go:embed demo-hq.json
var demoHQ []byte

// The demo follow-up HQ (S25), Trust A's only: it shares questions with the Demo pre-op HQ (by
// ID, and one by wording), so a tester sees pre-fill on a patient's second episode.
//
//go:embed demo-followup-hq.json
var demoFollowUpHQ []byte

// The platform's starter question library (S24): seeded into every deployment, like the codes.
//
//go:embed library.json
var libraryJSON []byte

// libraryItem is one platform library question: its key is its element's name.
type libraryItem struct {
	key, title string
	tags       []string
	element    json.RawMessage
}

// starterLibrary reads library.json. An item's title is its element's (a group's, for "yes/no +
// details").
func starterLibrary() ([]libraryItem, error) {
	var file struct {
		Questions []struct {
			Tags    []string
			Element json.RawMessage
		}
	}
	if err := json.Unmarshal(libraryJSON, &file); err != nil {
		return nil, fmt.Errorf("library.json: %w", err)
	}
	var out []libraryItem
	for _, q := range file.Questions {
		var el struct{ Name, Title string }
		if err := json.Unmarshal(q.Element, &el); err != nil || el.Name == "" {
			return nil, fmt.Errorf("library.json: an element without a name: %s", q.Element)
		}
		out = append(out, libraryItem{key: el.Name, title: el.Title, tags: q.Tags, element: q.Element})
	}
	return out, nil
}

// demoHQs are each trust's copy: the questionnaire, its version 1, and its Question Sets' ids
// (the third and later digits of the last group count up from chapterBase), made by its author.
var demoHQs = []struct{ trust, questionnaire, version, chapterBase, author string }{
	{trustA, "00000000-0000-4000-8000-000000000101", "00000000-0000-4000-8000-000000000102", "00000000-0000-4000-8000-00000000011", "00000000-0000-4000-8000-0000000000a1"},
	{trustB, "00000000-0000-4000-8000-000000000201", "00000000-0000-4000-8000-000000000202", "00000000-0000-4000-8000-00000000021", "00000000-0000-4000-8000-0000000000b1"},
}

// demoFollowUps are the follow-up HQ's copies (Trust A only), with fixed ids as demoHQs.
var demoFollowUps = []struct{ trust, questionnaire, version, chapterBase, author string }{
	{trustA, "00000000-0000-4000-8000-000000000301", "00000000-0000-4000-8000-000000000302", "00000000-0000-4000-8000-00000000031", "00000000-0000-4000-8000-0000000000a1"},
}

// demoProcedures are each trust's procedure list (S19): made-up codes, a hospital of "" is every
// hospital of the trust, and withHQ gives it the trust's demo HQ as its default. One has no
// default HQ, so it falls back to the hospital's default HQ (Settings) on a new episode.
var demoProcedures = []struct {
	id, trust, hospital, name, code, specialty string
	withHQ                                     bool
}{
	{"00000000-0000-4000-8000-000000000301", trustA, "", "Total hip replacement", "HIP-01", "Orthopaedics", true},
	{"00000000-0000-4000-8000-000000000302", trustA, "", "Knee arthroscopy", "KNEE-02", "Orthopaedics", true},
	{"00000000-0000-4000-8000-000000000303", trustA, "", "Laparoscopic cholecystectomy", "", "General surgery", false},
	{"00000000-0000-4000-8000-000000000304", trustA, hospitalA1, "Cataract surgery", "EYE-01", "Ophthalmology", true},
	{"00000000-0000-4000-8000-000000000305", trustB, "", "Total hip replacement", "HIP-01", "Orthopaedics", true},
	{"00000000-0000-4000-8000-000000000306", trustB, "", "Inguinal hernia repair", "", "General surgery", true},
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set (copy .env.example to .env)")
	}
	dir := "db/seed/codes"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	if err := seed(ctx, conn, dir); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

// seed loads everything in one transaction. It is safe to run again (the pipeline runs it after
// every deploy to QA): each run leaves the same rows and puts the demo data back as seeded.
func seed(ctx context.Context, db interface {
	Begin(context.Context) (pgx.Tx, error)
}, dir string) error {
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		if err := seedCategories(ctx, tx, dir); err != nil {
			return err
		}
		if err := seedCodes(ctx, tx, dir); err != nil {
			return err
		}
		if err := seedLibrary(ctx, tx); err != nil {
			return err
		}
		if err := seedContent(ctx, tx); err != nil {
			return err
		}
		return seedDemo(ctx, tx)
	})
}

func seedCategories(ctx context.Context, tx pgx.Tx, dir string) error {
	rows, err := readCSV(filepath.Join(dir, "code_categories.csv"))
	if err != nil {
		return err
	}
	rows = append(rows, map[string]string{"id": unassignedCategoryID, "name": "Unassigned"})
	for _, r := range rows {
		_, err := tx.Exec(ctx, `
			INSERT INTO categories (id, name, note_only) VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, note_only = EXCLUDED.note_only`,
			r["id"], r["name"], r["id"] == unassignedCategoryID)
		if err != nil {
			return fmt.Errorf("category %s: %w", r["name"], err)
		}
	}
	fmt.Printf("categories: %d\n", len(rows))
	return nil
}

// seedCodes maps incumbent columns: synonym → description, fsn → full_name,
// code_set_id → code set name. Every code is active.
func seedCodes(ctx context.Context, tx pgx.Tx, dir string) error {
	sets, err := readCSV(filepath.Join(dir, "code_sets.csv"))
	if err != nil {
		return err
	}
	setName := map[string]string{}
	for _, s := range sets {
		setName[s["id"]] = s["name"]
	}
	rows, err := readCSV(filepath.Join(dir, "codes.csv"))
	if err != nil {
		return err
	}
	for _, r := range rows {
		set, ok := setName[r["code_set_id"]]
		if !ok {
			return fmt.Errorf("code %s: unknown code set %s", r["code"], r["code_set_id"])
		}
		var category any // empty in some incumbent rows
		if r["code_category_id"] != "" {
			category = r["code_category_id"]
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO codes (id, code_set, code, description, full_name, category_id, billable, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
			ON CONFLICT (code_set, code) DO UPDATE SET
				description = EXCLUDED.description, full_name = EXCLUDED.full_name,
				category_id = EXCLUDED.category_id, billable = EXCLUDED.billable`,
			r["id"], set, r["code"], oneLine(r["synonym"]), oneLine(r["fsn"]), category, r["billable"] == "true")
		if err != nil {
			return fmt.Errorf("code %s %s: %w", set, r["code"], err)
		}
	}
	fmt.Printf("codes: %d\n", len(rows))
	return nil
}

// seedLibrary puts the platform's starter library back as library.json has it: each question
// added or updated by its key, and any other platform question removed. Questionnaires it was
// inserted into keep their copies. A trust's own library is left alone, and so is a starter
// question a platform admin retired (S33): that's a decision, not test data.
func seedLibrary(ctx context.Context, tx pgx.Tx) error {
	items, err := starterLibrary()
	if err != nil {
		return err
	}
	keys := make([]string, len(items))
	for i, it := range items {
		keys[i] = it.key
		tags := it.tags
		if tags == nil {
			tags = []string{}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO library_questions (key, title, element, tags) VALUES ($1, $2, $3, $4)
			ON CONFLICT (key) WHERE org_id IS NULL DO UPDATE SET title = EXCLUDED.title, element = EXCLUDED.element, tags = EXCLUDED.tags`,
			it.key, it.title, it.element, tags); err != nil {
			return fmt.Errorf("library %s: %w", it.key, err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM library_questions WHERE org_id IS NULL AND NOT (key = ANY ($1))`, keys); err != nil {
		return fmt.Errorf("library: %w", err)
	}
	fmt.Printf("library questions: %d\n", len(items))
	return nil
}

// exampleVideos are the content library's made-up examples (S33 answers 1): Preopsis's, so every
// trust sees them, all pointing to one public test film (the Blender Foundation's "Big Buck
// Bunny"), so the video picker (S34) can be tried. Real ones replace them before go-live. Each run
// puts them back as seeded; other items are left alone.
var exampleVideos = []struct {
	id, title, url string
	seconds        int
	tags           []string
}{
	{"00000000-0000-4000-8000-000000000501", "Example: Your anaesthetic explained", exampleVideoURL, 635, []string{"The anaesthetic", "Example"}},
	{"00000000-0000-4000-8000-000000000502", "Example: Fasting before your operation", exampleVideoURL, 635, []string{"Fasting", "Example"}},
	{"00000000-0000-4000-8000-000000000503", "Example: Getting ready for a knee replacement", exampleVideoURL, 635, []string{"Knee replacement", "Prehab", "Example"}},
}

const exampleVideoURL = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"

// seedContent puts the example videos back as seeded.
func seedContent(ctx context.Context, tx pgx.Tx) error {
	for _, v := range exampleVideos {
		if _, err := tx.Exec(ctx, `INSERT INTO content_items (id, org_id, kind, title, url, length_seconds, tags)
			VALUES ($1, NULL, 'video', $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET org_id = NULL, kind = 'video', title = EXCLUDED.title, url = EXCLUDED.url,
				length_seconds = EXCLUDED.length_seconds, tags = EXCLUDED.tags, status = 'active', updated_at = now()`,
			v.id, v.title, v.url, v.seconds, v.tags); err != nil {
			return fmt.Errorf("example video %s: %w", v.title, err)
		}
	}
	fmt.Printf("example videos: %d\n", len(exampleVideos))
	return nil
}

func seedDemo(ctx context.Context, tx pgx.Tx) error {
	// Every test round starts with no sign-in rate limits counted (S15): a manual tester or a
	// journey run from this machine shares one IP address's 10 an hour.
	if _, err := tx.Exec(ctx, `DELETE FROM rate_limit_hits`); err != nil {
		return fmt.Errorf("rate limits: %w", err)
	}
	// …and with the jobs timer's clock back on real time (S16), so a tester's "move the clock
	// ahead" doesn't send reminders on the next round's new episodes.
	if _, err := tx.Exec(ctx, `UPDATE qa_jobs_clock SET offset_minutes = 0`); err != nil {
		return fmt.Errorf("jobs clock: %w", err)
	}
	orgs := map[string]string{} // the fixed trust id → its id in this database
	for _, tr := range demoTrusts {
		id, err := seedTrust(ctx, tx, tr.id, tr.name, tr.code, tr.hospitals[0][0])
		if err != nil {
			return fmt.Errorf("trust %s: %w", tr.name, err)
		}
		orgs[tr.id] = id
		if _, err := tx.Exec(ctx, `UPDATE orgs SET status = 'active', settings = '{}', contact_name = $2, contact_email = $3, contact_phone = $4
			WHERE id = $1`, id, tr.contact[0], tr.contact[1], tr.contact[2]); err != nil {
			return fmt.Errorf("trust %s: %w", tr.name, err)
		}
		for _, h := range tr.hospitals {
			if _, err := tx.Exec(ctx, `INSERT INTO hospitals (id, org_id, name, address) VALUES ($1, $2, $3, $4)
				ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, address = EXCLUDED.address, status = 'active',
					timezone = NULL, settings = '{}'`,
				h[0], id, h[1], "1 Example Road, "+h[1]); err != nil {
				return fmt.Errorf("hospital %s: %w", h[1], err)
			}
		}
	}
	for _, u := range demoUsers {
		if _, err := tx.Exec(ctx, `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email`, u.id, u.name, u.email); err != nil {
			return fmt.Errorf("user %s: %w", u.name, err)
		}
		// The demo users' memberships are exactly these, whatever they were before.
		if _, err := tx.Exec(ctx, `DELETE FROM memberships WHERE user_id = $1`, u.id); err != nil {
			return fmt.Errorf("memberships %s: %w", u.name, err)
		}
		for _, m := range u.memberships {
			var hospital any // NULL: the whole trust
			if m.hospital != "" {
				hospital = m.hospital
			}
			if _, err := tx.Exec(ctx, `INSERT INTO memberships (user_id, org_id, hospital_id, role) VALUES ($1, $2, $3, $4)`,
				u.id, orgs[m.trust], hospital, m.role); err != nil {
				return fmt.Errorf("membership %s: %w", u.name, err)
			}
		}
		if u.platformAdmin {
			if _, err := tx.Exec(ctx, `INSERT INTO platform_admins (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, u.id); err != nil {
				return fmt.Errorf("platform admin %s: %w", u.name, err)
			}
		}
	}
	for _, p := range demoPatients {
		var nhs, typ any // NULL: no NHS number
		if p.nhs != "" {
			nhs, typ = p.nhs, "nhs"
		}
		// Each run puts the demo records back as they were seeded.
		if _, err := tx.Exec(ctx, `INSERT INTO patients (id, org_id, first_name, last_name, date_of_birth, sex, national_id,
				national_id_type, mobile, email)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO UPDATE SET first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name,
				date_of_birth = EXCLUDED.date_of_birth, sex = EXCLUDED.sex, national_id = EXCLUDED.national_id,
				national_id_type = EXCLUDED.national_id_type, mobile = EXCLUDED.mobile, email = EXCLUDED.email`,
			p.id, orgs[trustA], p.first, p.last, p.dob, p.sex, nhs, typ, p.mobile, p.email); err != nil {
			return fmt.Errorf("patient %s: %w", p.last, err)
		}
	}
	if err := seedHQs(ctx, tx, orgs); err != nil {
		return err
	}
	if err := seedProcedures(ctx, tx, orgs); err != nil {
		return err
	}
	fmt.Printf("trusts: %d, users: %d, patients: %d, demo HQs: %d, procedures: %d\n",
		len(demoTrusts), len(demoUsers), len(demoPatients), len(demoHQs)+len(demoFollowUps), len(demoProcedures))
	return nil
}

// seedHQs adds each trust's published demo HQ, and Trust A's follow-up HQ, the first time. Later
// runs leave them as they are: a published version can't change, and a tester may have made a new
// version of one.
func seedHQs(ctx context.Context, tx pgx.Tx, orgs map[string]string) error {
	for _, h := range []struct {
		file   string
		raw    []byte
		copies []struct{ trust, questionnaire, version, chapterBase, author string }
	}{{"demo-hq.json", demoHQ, demoHQs}, {"demo-followup-hq.json", demoFollowUpHQ, demoFollowUps}} {
		var hq struct {
			Name, Description string
			Chapters          []struct {
				Name, Audience string
				Content        json.RawMessage
			}
		}
		if err := json.Unmarshal(h.raw, &hq); err != nil {
			return fmt.Errorf("%s: %w", h.file, err)
		}
		if len(hq.Chapters) > 9 {
			return fmt.Errorf("%s: %d Question Sets, at most 9", h.file, len(hq.Chapters))
		}
		for _, d := range h.copies {
			if _, err := tx.Exec(ctx, `INSERT INTO questionnaires (id, org_id, name, description, created_by) VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (id) DO NOTHING`, d.questionnaire, orgs[d.trust], hq.Name, hq.Description, d.author); err != nil {
				return fmt.Errorf("%s: %w", h.file, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO questionnaire_versions (id, questionnaire_id, version_no, status, updated_by)
				VALUES ($1, $2, 1, 'published', $3) ON CONFLICT (id) DO NOTHING`, d.version, d.questionnaire, d.author); err != nil {
				return fmt.Errorf("%s version: %w", h.file, err)
			}
			for i, ch := range hq.Chapters {
				if _, err := tx.Exec(ctx, `INSERT INTO chapters (id, version_id, position, name, audience, content, updated_by)
					VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO NOTHING`,
					fmt.Sprintf("%s%d", d.chapterBase, i+1), d.version, i+1, ch.Name, ch.Audience, ch.Content, d.author); err != nil {
					return fmt.Errorf("%s set %s: %w", h.file, ch.Name, err)
				}
			}
		}
	}
	return nil
}

// seedProcedures puts each demo procedure back as seeded: in use, with its fields and default HQ.
// A name a tester has since given another procedure of the trust is left alone (names are unique).
func seedProcedures(ctx context.Context, tx pgx.Tx, orgs map[string]string) error {
	hqOf := map[string]string{}
	for _, d := range demoHQs {
		hqOf[d.trust] = d.questionnaire
	}
	for _, p := range demoProcedures {
		var hospital, hq any // NULL: every hospital; no default HQ
		if p.hospital != "" {
			hospital = p.hospital
		}
		if p.withHQ {
			hq = hqOf[p.trust]
		}
		// ON CONFLICT DO NOTHING covers both the id and a name already taken.
		if _, err := tx.Exec(ctx, `INSERT INTO procedures (id, org_id, hospital_id, name, code, specialty, default_questionnaire_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`,
			p.id, orgs[p.trust], hospital, p.name, p.code, p.specialty, hq); err != nil {
			return fmt.Errorf("procedure %s: %w", p.name, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE procedures SET hospital_id = $3, name = $4, code = $5, specialty = $6,
				default_questionnaire_id = $7, retired = false, info_content = '[]', proms_plan = '[]', updated_at = now()
			WHERE id = $1 AND org_id = $2 AND NOT EXISTS (SELECT 1 FROM procedures o WHERE o.org_id = $2 AND lower(o.name) = lower($4) AND o.id <> $1)`,
			p.id, orgs[p.trust], hospital, p.name, p.code, p.specialty, hq); err != nil {
			return fmt.Errorf("procedure %s: %w", p.name, err)
		}
	}
	return nil
}

// seedTrust names the trust that holds the demo hospital firstHospital and returns its id. On a
// database from before S08, migration 00006 put that hospital in a trust of its own with a new
// id, and that trust (with its questionnaires, patients and episodes) becomes the demo trust.
// Otherwise the trust is created with the fixed id.
func seedTrust(ctx context.Context, tx pgx.Tx, fixedID, name, code, firstHospital string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT org_id::text FROM hospitals WHERE id = $1`, firstHospital).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		id = fixedID
	} else if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO orgs (id, name, code) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, code = EXCLUDED.code`, id, name, code)
	return id, err
}

// readCSV returns each row keyed by header name.
func readCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%s: empty", path)
	}
	var rows []map[string]string
	for _, rec := range records[1:] {
		row := map[string]string{}
		for i, h := range records[0] {
			row[h] = rec[i]
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// oneLine collapses line breaks and repeated spaces inside a field. One incumbent
// row (ICD10 F17.1) has a line break in its description.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
