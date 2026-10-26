package asaguide

import (
	"encoding/csv"
	"os"
	"slices"
	"testing"
)

func TestChoicesFor(t *testing.T) {
	// A code whose grade depends on control: both examples, lowest grade first.
	got := ChoicesFor("ICD10", "E11")
	var adults []Choice
	for _, c := range got {
		if c.Group == "Adults" {
			adults = append(adults, c)
		}
	}
	want := []Choice{
		{Grade: "II", Example: "Well-controlled diabetes", Group: "Adults"},
		{Grade: "III", Example: "Poorly controlled diabetes", Group: "Adults"},
	}
	if !slices.Equal(adults, want) {
		t.Errorf("E11 adults' choices = %+v, want %+v", adults, want)
	}
	for i := 1; i < len(got); i++ {
		if rank(got[i-1].Grade) > rank(got[i].Grade) {
			t.Errorf("E11's choices aren't in grade order: %+v", got)
		}
	}
	// One example: one choice.
	if got := ChoicesFor("SNOMED", "13645005"); !slices.Equal(got, []Choice{{Grade: "III", Example: "COPD", Group: "Adults"}}) {
		t.Errorf("COPD = %+v", got)
	}
	// Not in the guide, another code set, or a code that differs in case: none, never nil (JSON []).
	for _, c := range [][2]string{{"SNOMED", "49727002"}, {"SNOMED", "E11"}, {"ICD10", "e11"}} {
		if got := ChoicesFor(c[0], c[1]); got == nil || len(got) != 0 {
			t.Errorf("%v = %#v, want an empty list", c, got)
		}
	}
	// The caller can't change the guide through the list it gets.
	ChoicesFor("ICD10", "E11")[0].Grade = "VI"
	if ChoicesFor("ICD10", "E11")[0].Grade == "VI" {
		t.Error("ChoicesFor returned the guide's own slice")
	}
}

// The file's shape: every example has an id, a grade I–VI, words, a known group and codes, and
// every code is one the seed loads (db/seed/codes/codes.csv), so no example is unreachable.
func TestGuideFile(t *testing.T) {
	g, err := load()
	if err != nil {
		t.Fatal(err)
	}
	seeded := seededCodes(t)
	ids := map[string]bool{}
	for _, e := range g.Examples {
		if e.ID == "" || ids[e.ID] {
			t.Errorf("example %q: missing or repeated id", e.ID)
		}
		ids[e.ID] = true
		if rank(e.Grade) == 0 {
			t.Errorf("%s: grade %q isn't I–VI", e.ID, e.Grade)
		}
		if e.Example == "" || !slices.Contains(groups, e.Group) || len(e.Codes) == 0 {
			t.Errorf("%s: needs words, a group (%v) and codes", e.ID, groups)
		}
		for _, c := range e.Codes {
			if !seeded[c] {
				t.Errorf("%s: %v isn't in the seeded codes", e.ID, c)
			}
		}
	}
}

func seededCodes(t *testing.T) map[[2]string]bool {
	t.Helper()
	f, err := os.Open("../../db/seed/codes/codes.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	sets := map[string]string{ // the incumbent's code set ids (db/seed/codes/code_sets.csv)
		"2773268e-5d3c-4eaa-957c-c861a05effdc": "SNOMED",
		"85a84ce7-ae57-4781-a40e-4d15ab99133e": "ICD10",
	}
	out := map[[2]string]bool{}
	for _, r := range rows[1:] {
		out[[2]string{sets[r[2]], r[1]}] = true
	}
	return out
}
