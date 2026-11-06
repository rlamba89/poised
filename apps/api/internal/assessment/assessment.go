// Package assessment holds the platform's assessments (plan C7, session S47; §9): validated
// instruments a clinician completes on an episode, such as STOP-Bang. Each is locked SurveyJS JSON
// written from packages/clinical (src/assessments.ts, built from the ready-made score in ready.ts),
// so the editor, the HQ and the assessment share one wording, scoring and set of keys; the Vitest
// test assessments.test.ts fails if this copy differs.
//
// Go stores each as a platform questionnaire (no trust; migration 00038) and checks the shape of
// what a clinician saves. It never works out a score: the browser does (packages/clinical).
package assessment

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
)

//go:embed *.json
var files embed.FS

// Def is one assessment: its id (the questionnaire's `assessment`), name and locked content.
type Def struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Content json.RawMessage `json:"-"`
}

// All are the assessments a clinician can add, in the order they're offered.
var All = []Def{load("stopbang", "STOP-Bang")}

func load(id, name string) Def {
	b, err := files.ReadFile(id + ".json")
	if err != nil {
		panic(err)
	}
	return Def{ID: id, Name: name, Content: bytes.TrimSpace(b)}
}

// Lookup finds an assessment by its id.
func Lookup(id string) (Def, bool) {
	i := slices.IndexFunc(All, func(d Def) bool { return d.ID == id })
	if i < 0 {
		return Def{}, false
	}
	return All[i], true
}

// Sources are where a stored answer may come from: the HQ, an observation, or the clinician. The
// patient's age and sex come from their record (source profile) and only they may.
var Sources = []string{"hq", "observation", "clinician"}

// profileItems are the patient's age and sex as of Confirm, stored with the confirmed answers so a
// later correction to the record never moves a confirmed score (S47 answers 7). They aren't
// questions: the score reads them as {patientAge} and {patientSex}, so they're allowed only where
// the content names them. The sex is one the score can use; Other and Not known leave it missing.
var profileItems = map[string]element{
	"patientAge": {Name: "patientAge", Title: "Age", Min: ptr(0), Max: ptr(150)},
	"patientSex": {Name: "patientSex", Title: "Sex", Choices: []json.RawMessage{json.RawMessage(`{"value":"male"}`), json.RawMessage(`{"value":"female"}`)}},
}

func ptr(f float64) *float64 { return &f }

// element is the part of a SurveyJS element the checks read.
type element struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Title    string            `json:"title"`
	Min      *float64          `json:"min"`
	Max      *float64          `json:"max"`
	Choices  []json.RawMessage `json:"choices"`
	Elements []element         `json:"elements"`
}

// questionList is the content's questions in order: not its Groups, Calculations or texts.
func questionList(content json.RawMessage) []element {
	var doc struct {
		Pages []struct {
			Elements []element `json:"elements"`
		} `json:"pages"`
	}
	var out []element
	if json.Unmarshal(content, &doc) != nil {
		return out
	}
	var walk func([]element)
	walk = func(els []element) {
		for _, el := range els {
			switch el.Type {
			case "panel":
				walk(el.Elements)
			case "expression", "html":
			default:
				out = append(out, el)
			}
		}
	}
	for _, p := range doc.Pages {
		walk(p.Elements)
	}
	return out
}

// questions are the content's questions by name.
func questions(content json.RawMessage) map[string]element {
	out := map[string]element{}
	for _, el := range questionList(content) {
		out[el.Name] = el
	}
	return out
}

// Unanswered names the first question with no answer, or "" when every one has one: a Confirm
// needs them all (the browser's check too; Go only reads the JSON, it doesn't evaluate).
func Unanswered(content json.RawMessage, data map[string]json.RawMessage) string {
	for _, el := range questionList(content) {
		if raw, ok := data[el.Name]; !ok || string(bytes.TrimSpace(raw)) == "null" {
			return label(el)
		}
	}
	return ""
}

// label is how a problem names a question: its first words.
func label(el element) string {
	switch el.Name {
	case "stopbang_bmi":
		return "BMI"
	case "stopbang_neck":
		return "Neck"
	}
	if len([]rune(el.Title)) > 40 {
		return string([]rune(el.Title)[:40]) + "…"
	}
	return el.Title
}

// Check returns a plain-language problem with a clinician's answers and their sources, or "":
// every answer is one of the content's questions, an option it offers or a number within its
// limits, and each has exactly one source.
func Check(content json.RawMessage, data map[string]json.RawMessage, sources map[string]string) string {
	qs := questions(content)
	for name, raw := range data {
		el, ok := qs[name]
		profile, isProfile := profileItems[name]
		if isProfile && !ok && bytes.Contains(content, []byte("{"+name+"}")) {
			el, ok = profile, true
		}
		if !ok {
			return "An answer doesn't belong to this assessment."
		}
		if !fits(el, raw) || (isProfile && len(el.Choices) == 0 && !isWhole(raw)) {
			if el.Min != nil && el.Max != nil && len(el.Choices) == 0 && isNumber(raw) {
				return fmt.Sprintf("%s: a number from %s to %s.", label(el), num(*el.Min), num(*el.Max))
			}
			return fmt.Sprintf("%s: that answer isn't one this assessment takes.", label(el))
		}
		if isProfile != (sources[name] == "profile") || (!isProfile && !slices.Contains(Sources, sources[name])) {
			return "Say where each answer came from."
		}
	}
	for name := range sources {
		if _, ok := data[name]; !ok {
			return "Say where each answer came from."
		}
	}
	return ""
}

// isNumber says whether an answer is a JSON number.
func isNumber(raw json.RawMessage) bool {
	var n float64
	return json.Unmarshal(raw, &n) == nil
}

// isWhole says whether an answer is a whole number (an age in years).
func isWhole(raw json.RawMessage) bool {
	var n float64
	return json.Unmarshal(raw, &n) == nil && n == math.Trunc(n)
}

// num writes a limit as a person would: 5, 150, 2.5.
func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// fits says whether an answer is one of the element's options, or a number within its limits.
func fits(el element, raw json.RawMessage) bool {
	if len(el.Choices) > 0 {
		var v string
		if json.Unmarshal(raw, &v) != nil {
			return false
		}
		for _, c := range el.Choices {
			var choice struct {
				Value json.RawMessage `json:"value"`
			}
			var s string
			if json.Unmarshal(c, &choice) == nil && json.Unmarshal(choice.Value, &s) == nil && s == v {
				return true
			}
		}
		return false
	}
	var n float64
	if json.Unmarshal(raw, &n) != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return false
	}
	return (el.Min == nil || n >= *el.Min) && (el.Max == nil || n <= *el.Max)
}
