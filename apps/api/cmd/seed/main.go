// Command seed loads Lifebox's categories and codes from db/seed/lifebox/*.csv,
// plus the demo trusts, hospitals and users. It is safe to run more than once.
package main

import (
	"context"
	"encoding/csv"
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
// (Poised's defaults), so each test round starts the same (S10).
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

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set (copy .env.example to .env)")
	}
	dir := "db/seed/lifebox"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if err := seedCategories(ctx, tx, dir); err != nil {
			return err
		}
		if err := seedCodes(ctx, tx, dir); err != nil {
			return err
		}
		return seedDemo(ctx, tx)
	})
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
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

// seedCodes maps Lifebox columns: synonym → description, fsn → full_name,
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
		var category any // empty in some Lifebox rows
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
	fmt.Printf("trusts: %d, users: %d, patients: %d\n", len(demoTrusts), len(demoUsers), len(demoPatients))
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

// oneLine collapses line breaks and repeated spaces inside a field. One Lifebox
// row (ICD10 F17.1) has a line break in its description.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
