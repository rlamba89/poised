package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/keys"
)

// The seeded Demo pre-op HQ and the e2e journeys' questionnaires (e2e/data) are published into the
// same local trusts, so they must agree on every key they share: since S23 a key published with
// different answers in one trust blocks publishing that question anywhere in it (test report 3,
// R-05, where old test HQs had done exactly that).
func TestSeedAndE2EQuestionnairesAgreeOnKeys(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "..", "e2e", "data", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no e2e data files: %v", err)
	}
	walk := func(name string, raw []byte) []keys.Placed {
		var hq struct {
			Chapters []struct {
				Name, Audience string
				Content        json.RawMessage
			}
		}
		if err := json.Unmarshal(raw, &hq); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var sets []keys.Set
		for _, c := range hq.Chapters {
			sets = append(sets, keys.Set{Name: c.Name, Audience: c.Audience, Content: c.Content})
		}
		qs, err := keys.WalkSets(sets)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return qs
	}
	all := map[string][]keys.Placed{"demo-hq.json": walk("demo-hq.json", demoHQ)}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		all[filepath.Base(f)] = walk(f, raw)
	}
	for name, qs := range all {
		published := map[string][]keys.Published{}
		for other, oqs := range all {
			if other == name {
				continue
			}
			for _, q := range oqs {
				published[q.Name] = append(published[q.Name], keys.Published{Questionnaire: other, VersionNo: 1, Question: q.Question})
			}
		}
		for _, p := range keys.LockProblems(qs, published) {
			t.Errorf("%s: %s · %s", name, p.Set, p.Message)
		}
	}
}
