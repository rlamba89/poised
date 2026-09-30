// Command seed loads Lifebox's categories and codes from db/seed/lifebox/*.csv,
// plus the demo hospitals and users. It is safe to run more than once.
package main

import (
	"context"
	"encoding/csv"
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

	hospitalA = "00000000-0000-4000-8000-00000000000a"
	hospitalB = "00000000-0000-4000-8000-00000000000b"
)

var demoUsers = []struct{ id, name, email, hospital, role string }{
	{"00000000-0000-4000-8000-0000000000a1", "Alex Author", "alex.author@hospital-a.example", hospitalA, "author"},
	{"00000000-0000-4000-8000-0000000000a2", "Val Viewer", "val.viewer@hospital-a.example", hospitalA, "viewer"},
	{"00000000-0000-4000-8000-0000000000b1", "Bea Author", "bea.author@hospital-b.example", hospitalB, "author"},
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
	for id, name := range map[string]string{hospitalA: "Hospital A", hospitalB: "Hospital B"} {
		if _, err := tx.Exec(ctx, `INSERT INTO hospitals (id, name) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`, id, name); err != nil {
			return fmt.Errorf("hospital %s: %w", name, err)
		}
	}
	for _, u := range demoUsers {
		if _, err := tx.Exec(ctx, `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email`, u.id, u.name, u.email); err != nil {
			return fmt.Errorf("user %s: %w", u.name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO memberships (user_id, hospital_id, role) VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`, u.id, u.hospital, u.role); err != nil {
			return fmt.Errorf("membership %s: %w", u.name, err)
		}
	}
	fmt.Printf("hospitals: 2, users: %d\n", len(demoUsers))
	return nil
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
