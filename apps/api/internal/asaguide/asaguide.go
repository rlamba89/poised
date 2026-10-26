// Package asaguide is the ASA guide (plan C6, session S39; ASA-01): the ASA Physical Status
// Classification's own published examples, each with the codes it covers (guide.json).
//
// No standard maps a diagnosis code to an ASA grade: the grade depends on severity and control
// (well-controlled diabetes is II, poorly controlled is III). So a code gives the author choices,
// never a silent grade: GET /api/codes sends each code's examples, and the editor fills in the
// grade when there's one and asks the author to pick when there are several (S39 answers 1).
// The grade and the example's words are copied into the disclosure, so a published HQ never
// changes when this file does. Go only serves the file; packages/clinical works out the suggested
// ASA in the browser.
package asaguide

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

//go:embed guide.json
var guideJSON []byte

// Choice is one example a code can take its grade from, as the code search sends it.
type Choice struct {
	Grade   string `json:"grade"`   // I–VI, as packages/clinical's AsaGrade
	Example string `json:"example"` // the ASA's words, e.g. "Poorly controlled diabetes"
	Group   string `json:"group"`   // whose example it is: Adults, Children or Pregnancy
}

type example struct {
	ID      string      `json:"id"`
	Grade   string      `json:"grade"`
	Group   string      `json:"group"`
	Example string      `json:"example"`
	Codes   [][2]string `json:"codes"` // [code set, code]
}

type guide struct {
	Examples []example `json:"examples"`
}

var (
	grades = []string{"I", "II", "III", "IV", "V", "VI"}
	groups = []string{"Adults", "Children", "Pregnancy"}
	byCode = mustIndex()
)

// rank is a grade's number (I is 1), or 0 for anything else.
func rank(grade string) int { return slices.Index(grades, grade) + 1 }

func load() (guide, error) {
	var g guide
	if err := json.Unmarshal(guideJSON, &g); err != nil {
		return g, fmt.Errorf("asaguide: guide.json: %w", err)
	}
	return g, nil
}

func mustIndex() map[[2]string][]Choice {
	g, err := load()
	if err != nil {
		panic(err)
	}
	out := map[[2]string][]Choice{}
	for _, e := range g.Examples {
		for _, c := range e.Codes {
			out[c] = append(out[c], Choice{Grade: e.Grade, Example: e.Example, Group: e.Group})
		}
	}
	for _, list := range out {
		// Lowest grade first; within a grade, the file's order (adults before children).
		slices.SortStableFunc(list, func(a, b Choice) int { return rank(a.Grade) - rank(b.Grade) })
	}
	return out
}

// ChoicesFor is the examples a code can take its grade from, lowest grade first: none (an empty
// list, never nil) for a code the guide doesn't list. Codes match exactly, as the codes table holds them.
func ChoicesFor(codeSet, code string) []Choice {
	return append([]Choice{}, byCode[[2]string{codeSet, code}]...)
}
