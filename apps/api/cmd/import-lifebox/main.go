// Command import-lifebox loads a questionnaire exported from the Lifebox Author tool
// (tools/export-lifebox-hq.js) as a new draft questionnaire, one chapter per Question Set.
//
//	go run ./cmd/import-lifebox -file ~/Downloads/lifebox-hq-<id>.json
//
// It prints what it could not carry over. Run it again to import another copy.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/lifebox"
)

func main() {
	file := flag.String("file", "", "the JSON file the export snippet downloaded")
	hospital := flag.String("hospital", "00000000-0000-4000-8000-00000000000a", "hospital to import into (default: Hospital A)")
	user := flag.String("user", "00000000-0000-4000-8000-0000000000a1", "user recorded as the author (default: Alex Author)")
	flag.Parse()
	if *file == "" {
		log.Fatal("-file is required")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set (copy .env.example to .env)")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}
	var export lifebox.Export
	if err := json.Unmarshal(raw, &export); err != nil {
		log.Fatalf("read %s: %v", *file, err)
	}
	hid, uid := uuid.MustParse(*hospital), uuid.MustParse(*user)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	lib, err := loadLibrary(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	q, warnings := lifebox.Convert(export, lib)

	var id uuid.UUID
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		dq := db.New(tx)
		name, err := freeName(ctx, dq, hid, q.Name)
		if err != nil {
			return err
		}
		if id, err = dq.CreateQuestionnaire(ctx, db.CreateQuestionnaireParams{HospitalID: hid, Name: name, Description: q.Description, CreatedBy: uid}); err != nil {
			return fmt.Errorf("create questionnaire: %w", err)
		}
		vid, err := dq.CreateVersion(ctx, db.CreateVersionParams{QuestionnaireID: id, VersionNo: 1, UpdatedBy: uid})
		if err != nil {
			return fmt.Errorf("create version: %w", err)
		}
		for _, s := range q.Sets {
			cid, err := dq.CreateChapter(ctx, db.CreateChapterParams{VersionID: vid, Name: s.Name, UpdatedBy: uid})
			if err != nil {
				return fmt.Errorf("create chapter %q: %w", s.Name, err)
			}
			if err := dq.UpdateChapterMeta(ctx, db.UpdateChapterMetaParams{
				ID: cid, Name: s.Name, Description: s.Description, Icon: s.Icon, Audience: "patient", UpdatedBy: uid,
			}); err != nil {
				return fmt.Errorf("chapter %q: %w", s.Name, err)
			}
			content, err := json.Marshal(s.Content)
			if err != nil {
				return err
			}
			if _, err := dq.SaveChapterContent(ctx, db.SaveChapterContentParams{Content: content, UpdatedBy: uid, ID: cid, Revision: 0}); err != nil {
				return fmt.Errorf("save chapter %q: %w", s.Name, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("import: %v", err)
	}
	fmt.Printf("Imported %q: %d Question Sets. Questionnaire id %s\n", q.Name, len(q.Sets), id)
	if len(warnings) > 0 {
		fmt.Printf("\n%d things to check:\n", len(warnings))
		for _, w := range warnings {
			fmt.Println(" -", w)
		}
	}
}

// freeName adds " (2)", " (3)"… if a draft with the name already exists (FRM-04).
func freeName(ctx context.Context, q *db.Queries, hid uuid.UUID, name string) (string, error) {
	candidate := name
	for n := 2; ; n++ {
		taken, err := q.DraftNameExists(ctx, db.DraftNameExistsParams{HospitalID: hid, Name: candidate, ExcludeID: uuid.Nil})
		if err != nil || !taken {
			return candidate, err
		}
		candidate = fmt.Sprintf("%s (%d)", name, n)
	}
}

// library is our code library and categories, loaded once.
type library struct {
	codes map[string]string // "SNOMED|77176002" → description
	cats  map[string]string // id → name
}

func (l library) CodeDisplay(set, code string) (string, bool) {
	d, ok := l.codes[set+"|"+code]
	return d, ok
}

func (l library) CategoryName(id string) (string, bool) {
	n, ok := l.cats[id]
	return n, ok
}

func loadLibrary(ctx context.Context, conn *pgx.Conn) (library, error) {
	lib := library{codes: map[string]string{}, cats: map[string]string{}}
	rows, err := conn.Query(ctx, `SELECT code_set, code, description FROM codes`)
	if err != nil {
		return lib, fmt.Errorf("load codes: %w", err)
	}
	for rows.Next() {
		var set, code, desc string
		if err := rows.Scan(&set, &code, &desc); err != nil {
			return lib, err
		}
		lib.codes[set+"|"+code] = desc
	}
	rows.Close()
	rows, err = conn.Query(ctx, `SELECT id::text, name FROM categories`)
	if err != nil {
		return lib, fmt.Errorf("load categories: %w", err)
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return lib, err
		}
		lib.cats[id] = name
	}
	rows.Close()
	return lib, rows.Err()
}
