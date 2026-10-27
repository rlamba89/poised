// Package asarules holds an HQ's ASA rules (plan C6, session S40; ASA-02, ASA-03): the platform's
// default rules (defaults.json), and the check of a rule list's shape before it's stored with a
// questionnaire version (questionnaire_versions.rules).
//
// Go only stores and checks the rules; it never decides which fire. packages/clinical (rules.ts)
// works that out from the answers, in the browser and, from S41, in the evaluate service. So the
// check here is about shape alone: known actions, a target and a status from the allowed lists, a
// team from episode.Teams and a task title as episode.TaskInput checks it.
//
// The stored JSON is one object: the rules in order, the HQ's setting for patients whose answers
// give no ASA grade (S40 answers 2: counted as ASA I and auto-triaged, unless an author turns it
// off), when an auto-triaged patient's message goes (S42 answers 2: at once unless the author says
// after Checked), and the whole-HQ test cases (S40 answers 4), which check which rules fire.
package asarules

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/episode"
)

//go:embed defaults.json
var defaultsJSON []byte

// Defaults is the platform's default rule set, given to every new HQ and new version of an HQ
// that had none (S40 answers 1). A copy, so callers can't change it.
func Defaults() json.RawMessage { return bytes.Clone(defaultsJSON) }

// The action kinds (ASA-03's fixed list).
const (
	AutoTriage             = "autoTriage"
	Task                   = "task"
	Flag                   = "flag"
	RequireAnaesthetistASA = "requireAnaesthetistAsa"
	SuggestStatus          = "suggestStatus"
)

// Targets are where auto-triage may move an episode (A-8: Ready for admission by default).
var Targets = []string{episode.ReadyForAdmission, episode.ReadyForPOA, episode.POAComplete}

// Suggestable are the statuses a rule may suggest: those after the HQ review, before admission.
var Suggestable = []string{episode.ReadyForPOA, episode.POAComplete, episode.OnHold, episode.NotReady, episode.ReadyForAdmission}

// Limits on the stored set.
const (
	MaxRules     = 50
	MaxTestCases = 100
	MaxName      = 100
	MaxBytes     = 512 << 10
)

type ASA struct {
	Op    string `json:"op"`    // ">=", "<=" or "="
	Grade int    `json:"grade"` // 1–6 (I–VI)
}

type Band struct {
	Score string   `json:"score"` // the Calculation's key
	Bands []string `json:"bands"` // band ids; the rule fires when the score is in one of them
}

type Code struct {
	Set     string `json:"set"` // SNOMED or ICD10, as packages/clinical's CodeSet
	Code    string `json:"code"`
	Display string `json:"display"`
}

type When struct {
	ASA        *ASA        `json:"asa"`
	NoRedFlags bool        `json:"noRedFlags,omitempty"`
	Band       *Band       `json:"band,omitempty"`
	Code       *Code       `json:"code,omitempty"`
	Procedures []uuid.UUID `json:"procedures,omitempty"` // the episode's procedure is one of these
}

// Action is one of the kinds above; only the fields of its kind are kept.
type Action struct {
	Kind   string `json:"kind"`
	Target string `json:"target,omitempty"` // autoTriage
	Title  string `json:"title,omitempty"`  // task
	Team   string `json:"team,omitempty"`   // task
	Name   string `json:"name,omitempty"`   // flag
	Status string `json:"status,omitempty"` // suggestStatus
}

type Rule struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	When When     `json:"when"`
	Then []Action `json:"then"`
}

// Patient is a test case's made-up patient (PRV-08).
type Patient struct {
	Name string   `json:"name"`
	Age  *float64 `json:"age,omitempty"`
	Sex  string   `json:"sex,omitempty"`
}

// TestCase is a whole-HQ test case: answers by question key, and the lines expected
// (the suggested ASA and the rules that fire, as packages/clinical writes them).
type TestCase struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Patient     Patient         `json:"patient"`
	ProcedureID *uuid.UUID      `json:"procedureId,omitempty"`
	Answers     json.RawMessage `json:"answers"`
	Expected    []string        `json:"expected"`
}

type RuleSet struct {
	NoGradeAutoTriage bool `json:"noGradeAutoTriage"`
	// MessageAfterCheck: an auto-triaged patient's status message (e.g. Ready for admission, if the
	// hospital sends one) waits until a clinician clicks Checked; off, it goes at once (S42 answers 2:
	// a setting on each HQ, off by default). A set stored before S42 has none: off.
	MessageAfterCheck bool       `json:"messageAfterCheck"`
	Rules             []Rule     `json:"rules"`
	TestCases         []TestCase `json:"testCases"`
}

const unreadable = "The rules could not be read."

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)

// Check reads a rule set and checks its shape. It returns the set to store (names and titles
// trimmed, each action with only its own fields), or a problem in plain words. The procedure ids
// it names are the caller's to check against the trust (ProcedureIDs).
func Check(raw json.RawMessage) (RuleSet, string) {
	var set struct {
		NoGradeAutoTriage *bool      `json:"noGradeAutoTriage"`
		MessageAfterCheck bool       `json:"messageAfterCheck"`
		Rules             []Rule     `json:"rules"`
		TestCases         []TestCase `json:"testCases"`
	}
	if len(raw) > MaxBytes {
		return RuleSet{}, "The rules and their test cases are too long."
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&set); err != nil || dec.More() || set.NoGradeAutoTriage == nil || set.Rules == nil {
		return RuleSet{}, unreadable
	}
	out := RuleSet{NoGradeAutoTriage: *set.NoGradeAutoTriage, MessageAfterCheck: set.MessageAfterCheck, Rules: []Rule{}, TestCases: []TestCase{}}
	if len(set.Rules) > MaxRules {
		return RuleSet{}, fmt.Sprintf("An HQ can have at most %d rules.", MaxRules)
	}
	ids, names := map[string]bool{}, map[string]bool{}
	for _, r := range set.Rules {
		if !idPattern.MatchString(r.ID) || ids[r.ID] {
			return RuleSet{}, unreadable
		}
		ids[r.ID] = true
		r.Name = strings.TrimSpace(r.Name)
		switch {
		case r.Name == "":
			return RuleSet{}, "Give each rule a name."
		case strings.ContainsAny(r.Name, "\r\n") || utf8.RuneCountInString(r.Name) > MaxName:
			return RuleSet{}, fmt.Sprintf("Keep each rule's name on one line, %d characters at most.", MaxName)
		case names[strings.ToLower(r.Name)]:
			return RuleSet{}, fmt.Sprintf("Two rules are called “%s”: give each its own name.", r.Name)
		}
		names[strings.ToLower(r.Name)] = true
		checked, problem := checkRule(r)
		if problem != "" {
			return RuleSet{}, fmt.Sprintf("Rule “%s”: %s", r.Name, problem)
		}
		out.Rules = append(out.Rules, checked)
	}
	if len(set.TestCases) > MaxTestCases {
		return RuleSet{}, fmt.Sprintf("An HQ can have at most %d test cases.", MaxTestCases)
	}
	caseIDs, caseNames := map[string]bool{}, map[string]bool{}
	for _, tc := range set.TestCases {
		if !idPattern.MatchString(tc.ID) || caseIDs[tc.ID] {
			return RuleSet{}, unreadable
		}
		caseIDs[tc.ID] = true
		checked, problem := checkTestCase(tc)
		if problem != "" {
			return RuleSet{}, problem
		}
		// As rules: one name each, so saving a case again under its name replaces it (test report 1, E-2).
		if caseNames[strings.ToLower(checked.Name)] {
			return RuleSet{}, fmt.Sprintf("Two test cases are called “%s”: give each its own name.", checked.Name)
		}
		caseNames[strings.ToLower(checked.Name)] = true
		out.TestCases = append(out.TestCases, checked)
	}
	return out, ""
}

func checkRule(r Rule) (Rule, string) {
	w := r.When
	switch {
	case w.ASA == nil || !slices.Contains([]string{">=", "<=", "="}, w.ASA.Op):
		return r, "choose how the suggested ASA is compared (at least, at most or exactly)."
	case w.ASA.Grade < 1 || w.ASA.Grade > 6:
		return r, "the ASA grade must be I to VI."
	case w.Band != nil && (strings.TrimSpace(w.Band.Score) == "" || len(w.Band.Bands) == 0 || slices.Contains(w.Band.Bands, "")):
		return r, "choose the score and at least one of its bands."
	case w.Code != nil && (strings.TrimSpace(w.Code.Code) == "" || (w.Code.Set != "SNOMED" && w.Code.Set != "ICD10")):
		return r, "choose the code."
	case w.Procedures != nil && len(w.Procedures) == 0:
		return r, "choose at least one procedure."
	case len(r.Then) == 0:
		return r, "add at least one action."
	}
	then := make([]Action, 0, len(r.Then))
	triaged := false
	for _, a := range r.Then {
		var clean Action
		switch a.Kind {
		case AutoTriage:
			if triaged {
				return r, "auto-triage only once."
			}
			triaged = true
			if !slices.Contains(Targets, a.Target) {
				return r, "auto-triage can go to Ready for admission, Ready for POA or POA complete."
			}
			clean = Action{Kind: a.Kind, Target: a.Target}
		case Task:
			title, problem := episode.TaskInput(a.Title, a.Team)
			if problem != "" {
				return r, problem
			}
			clean = Action{Kind: a.Kind, Title: title, Team: a.Team}
		case Flag:
			name := strings.TrimSpace(a.Name)
			if name == "" || strings.ContainsAny(name, "\r\n") || utf8.RuneCountInString(name) > MaxName {
				return r, fmt.Sprintf("name the flag, on one line (%d characters at most).", MaxName)
			}
			clean = Action{Kind: a.Kind, Name: name}
		case RequireAnaesthetistASA:
			clean = Action{Kind: a.Kind}
		case SuggestStatus:
			if !slices.Contains(Suggestable, a.Status) {
				return r, "a rule can suggest Ready for POA, POA complete, On hold, Not ready for admission or Ready for admission."
			}
			clean = Action{Kind: a.Kind, Status: a.Status}
		default:
			return r, "an action isn't one Preopsis knows."
		}
		then = append(then, clean)
	}
	r.Then = then
	return r, ""
}

func checkTestCase(tc TestCase) (TestCase, string) {
	tc.Name = strings.TrimSpace(tc.Name)
	switch {
	case tc.Name == "":
		return tc, "Give each test case a name."
	case strings.ContainsAny(tc.Name, "\r\n") || utf8.RuneCountInString(tc.Name) > MaxName:
		return tc, fmt.Sprintf("Keep each test case's name on one line, %d characters at most.", MaxName)
	case utf8.RuneCountInString(tc.Patient.Name) > MaxName || utf8.RuneCountInString(tc.Patient.Sex) > 20:
		return tc, fmt.Sprintf("Test case “%s”: the sample patient's details are too long.", tc.Name)
	case tc.Patient.Age != nil && (*tc.Patient.Age < 0 || *tc.Patient.Age > 120):
		return tc, fmt.Sprintf("Test case “%s”: the sample patient's age must be 0 to 120.", tc.Name)
	case len(bytes.TrimSpace(tc.Answers)) == 0 || bytes.TrimSpace(tc.Answers)[0] != '{':
		return tc, unreadable
	case tc.Expected == nil || len(tc.Expected) > 200:
		return tc, unreadable
	}
	return tc, ""
}

// ProcedureIDs are the procedures the set names (its rules' and its test cases'), each once.
func (s RuleSet) ProcedureIDs() []uuid.UUID {
	var ids []uuid.UUID
	add := func(id uuid.UUID) {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	for _, r := range s.Rules {
		for _, id := range r.When.Procedures {
			add(id)
		}
	}
	for _, tc := range s.TestCases {
		if tc.ProcedureID != nil {
			add(*tc.ProcedureID)
		}
	}
	return ids
}

// JSON is the set as stored.
func (s RuleSet) JSON() json.RawMessage {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err) // only plain types: can't fail
	}
	return b
}
