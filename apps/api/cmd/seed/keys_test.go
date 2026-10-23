package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/keys"
	"github.com/rlamba89/poised/apps/api/internal/prefill"
)

// The seeded Demo pre-op HQ, the follow-up HQ (S25) and the e2e journeys' questionnaires (e2e/data)
// are published into the same local trusts, so they must agree on every key they share: since S23 a key published with
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
	all := map[string][]keys.Placed{"demo-hq.json": walk("demo-hq.json", demoHQ), "demo-followup-hq.json": walk("demo-followup-hq.json", demoFollowUpHQ)}
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

// The platform's starter library (S24 answers 1): each item is a question (or, for "yes/no +
// details", a group of a Yes/No question and its details) whose key is its ID; no key is used twice
// across the library, and the seeded Demo pre-op HQ asks every library question it has exactly as
// the library does, so its answers pre-fill wherever the library's are inserted (S25).
func TestStarterLibrary(t *testing.T) {
	items, err := starterLibrary()
	if err != nil {
		t.Fatal(err)
	}
	var keysOf []string
	library := map[string][]keys.Published{}
	seen := map[string]bool{}
	for _, it := range items {
		keysOf = append(keysOf, it.key)
		qs, err := keys.Element(it.element)
		if err != nil || len(qs) == 0 {
			t.Fatalf("%s: %v", it.key, err)
		}
		if it.title == "" {
			t.Errorf("%s has no title", it.key)
		}
		for _, q := range qs {
			if seen[q.Name] {
				t.Errorf("the key %s is used twice in the library", q.Name)
			}
			seen[q.Name] = true
			library[q.Name] = append(library[q.Name], keys.Published{Questionnaire: "Library", VersionNo: 1, Question: q})
		}
	}
	for _, want := range []string{"smoker", "alcohol_units", "snoring", "ponv_history", "height", "weight", "pregnant", "allergies", "regular_medicines"} {
		if !seen[want] {
			t.Errorf("no %s in the library", want)
		}
	}
	for _, want := range []string{"anaesthetic_problems", "family_anaesthetic_problems"} {
		if !seen[want] || !seen[want+"_details"] {
			t.Errorf("no %s with its details in the library", want)
		}
	}
	if len(items) != 11 {
		t.Errorf("%d library items %v, want 11", len(items), keysOf)
	}
	for _, k := range []string{"weight", "pregnant"} {
		if q := library[k][0].Question; !q.AlwaysFresh {
			t.Errorf("%s isn't Always ask fresh", k)
		}
	}

	var hq struct {
		Chapters []struct {
			Name, Audience string
			Content        json.RawMessage
		}
	}
	if err := json.Unmarshal(demoHQ, &hq); err != nil {
		t.Fatal(err)
	}
	var sets []keys.Set
	for _, c := range hq.Chapters {
		sets = append(sets, keys.Set{Name: c.Name, Audience: c.Audience, Content: c.Content})
	}
	demo, err := keys.WalkSets(sets)
	if err != nil {
		t.Fatal(err)
	}
	shared := 0
	for _, q := range demo {
		if seen[q.Name] {
			shared++
		}
	}
	if shared < 8 {
		t.Errorf("the Demo pre-op HQ shares %d keys with the library, want most of it", shared)
	}
	for _, p := range keys.LockProblems(demo, library) {
		t.Errorf("demo-hq.json differs from the library: %s · %s", p.Set, p.Message)
	}
	if p := keys.Duplicates(demo); len(p) > 0 {
		t.Errorf("demo-hq.json: %v", p)
	}
}

// The follow-up HQ (S25) is for seeing pre-fill: a patient who sent the Demo pre-op HQ gets most of
// its questions filled, by ID and one by wording, and never weight or pregnancy (Always ask fresh),
// the nurse's blood pressure (a clinician set) or its new question.
func TestFollowUpHQPrefillsFromTheDemoHQ(t *testing.T) {
	chapters := func(raw []byte, prefix string) []prefill.Chapter {
		var hq struct {
			Chapters []struct {
				Audience string
				Content  json.RawMessage
			}
		}
		if err := json.Unmarshal(raw, &hq); err != nil {
			t.Fatal(err)
		}
		var out []prefill.Chapter
		for i, c := range hq.Chapters {
			out = append(out, prefill.Chapter{ID: prefix + string(rune('1'+i)), Audience: c.Audience, Content: c.Content})
		}
		return out
	}
	demo, followUp := chapters(demoHQ, "demo"), chapters(demoFollowUpHQ, "fu")
	src := prefill.Source{FormID: "f1", SubmittedAt: time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC), Chapters: demo,
		Patient: map[string]json.RawMessage{
			"demo1": json.RawMessage(`{"smoker":"ex","alcohol_units":4,"height":172,"weight":80}`),
			"demo2": json.RawMessage(`{"q_diabetes":"o_no","snoring":"yes","pregnant":"no","ponv_history":"no","anaesthetic_problems":"yes",` +
				`"anaesthetic_problems_details":"Sick for a day","allergies":[{"allergies_what":"Penicillin","allergies_reaction":"Rash"}]}`),
		},
		Clinician: map[string]json.RawMessage{"demo3": json.RawMessage(`{"q_bp":"120/80"}`)},
	}
	got, err := prefill.Fill(followUp, map[string]bool{"fu1": true, "fu2": true, "fu3": true}, []prefill.Source{src})
	if err != nil {
		t.Fatal(err)
	}
	names := func(set string) []string {
		var out []string
		for k := range got[set].Prefill {
			out = append(out, k)
		}
		slices.Sort(out)
		return out
	}
	if n := names("fu1"); !reflect.DeepEqual(n, []string{"alcohol_units", "height", "smoker"}) {
		t.Errorf("About you: %v", n)
	}
	if n := names("fu2"); !reflect.DeepEqual(n, []string{"allergies", "anaesthetic_problems", "anaesthetic_problems_details", "fu_diabetes", "ponv_history", "snoring"}) {
		t.Errorf("Your health: %v", n)
	}
	if e := got["fu2"].Prefill["fu_diabetes"]; e.MatchedBy != "wording" || string(e.Value) != `"n"` {
		t.Errorf("fu_diabetes %+v", e)
	}
	if _, ok := got["fu3"]; ok {
		t.Error("the nurse's set was pre-filled")
	}
}

// Every element the patient sees in the seeded HQs has a title of its own: an untitled one shows
// its ID ("profile", S25 test report 1, found 2).
func TestSeededHQsHaveTitles(t *testing.T) {
	for name, raw := range map[string][]byte{"demo-hq.json": demoHQ, "demo-followup-hq.json": demoFollowUpHQ} {
		var hq struct{ Chapters []struct{ Content json.RawMessage } }
		if err := json.Unmarshal(raw, &hq); err != nil {
			t.Fatal(err)
		}
		for _, c := range hq.Chapters {
			qs, err := keys.Walk(c.Content)
			if err != nil {
				t.Fatal(err)
			}
			for _, q := range qs {
				if q.Title == "" && q.Type != "html" {
					t.Errorf("%s: %s (%s) has no title", name, q.Name, q.Type)
				}
			}
		}
	}
}
