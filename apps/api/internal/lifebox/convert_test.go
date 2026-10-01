package lifebox

import (
	"encoding/json"
	"strings"
	"testing"
)

type fakeLib struct{}

func (fakeLib) CodeDisplay(set, code string) (string, bool) {
	if set == "SNOMED" && code == "77176002" {
		return "Smoker", true
	}
	return "", false
}

func (fakeLib) CategoryName(id string) (string, bool) {
	if id == "cat-1" {
		return "Lifestyle", true
	}
	return "", false
}

// A Lifebox export shaped like the GraphQL response (fields as the fragments name them).
const fixture = `{
  "questionnaire": {
    "name": "HJE Full HQ", "description": "Pre-op",
    "questionSets": [
      {
        "id": "set-1", "name": "About you", "icon": "User01",
        "pages": [
          {
            "id": "page-1", "name": {"text": "Smoking"}, "renderIf": null,
            "elements": [
              {"__typename": "QuestionBoolean", "id": "q-smoke", "name": {"text": "Do you smoke?"}, "isRequired": true,
               "optionTrueId": "opt-yes", "optionFalseId": "opt-no", "labelTrue": {"text": "Yes"}, "labelFalse": {"text": "No"},
               "rules": [{"expression": "hasSelectedOption({opt-yes})", "episodeNoteTemplate": "Current smoker", "episodeNoteCategoryId": "cat-1",
                          "disclosureCodes": [{"code": "77176002", "codeSet": "SNOMED", "synonym": "Smoker (finding)"}, {"code": "Z72.0", "codeSet": "icd10", "synonym": "Tobacco use"}]}]},
              {"__typename": "QuestionText", "id": "q-many", "name": {"text": "How many per day?"}, "textType": "MULTI_LINE", "isRequired": false,
               "visibleIf": "containsAny(selectedOptionIds, [{opt-yes}])",
               "rules": [{"expression": "true", "episodeNoteTemplate": "Smokes %s a day", "episodeNoteCategoryId": "unknown", "disclosureCodes": []}]},
              {"__typename": "QuestionCheckbox", "id": "q-allergy", "name": {"text": "Allergies?"}, "isRequired": true,
               "questionOptions": [
                 {"__typename": "QuestionOptionString", "id": "opt-latex", "label": {"text": "Latex"}},
                 {"__typename": "QuestionOptionNone", "id": "opt-none", "label": {"text": "No known allergies"}}]},
              {"__typename": "PanelStatement", "id": "st-1", "text": {"text": "Review the patient's allergies.\nThis entry will appear on the POA Summary."}, "isClinical": true,
               "visibleIf": "!containsAny(selectedOptionIds, [{opt-none}])"},
              {"__typename": "ClinicalPageSummary", "id": "sum-1", "isClinical": true}
            ]
          },
          {
            "id": "page-2", "name": {"text": "Latex"}, "renderIf": "containsAny(selectedOptionIds, [{opt-latex}]) && !containsAny(selectedOptionIds, [{opt-yes}])",
            "elements": [
              {"__typename": "QuestionMedication", "id": "q-med", "name": {"text": "Medicines"}, "medicationType": "NON_PRESCRIBED"},
              {"__typename": "QuestionDate", "id": "q-date", "name": {"text": "When?"}, "dateType": "DAY_MONTH_YEAR_MULTIPLE"}
            ]
          }
        ]
      },
      {
        "id": "set-2", "name": "Heart", "icon": "Heart",
        "pages": [{"id": "page-3", "name": {"text": "Heart"}, "elements": [
          {"__typename": "QuestionText", "id": "q-heart", "name": {"text": "Tell us more"}, "visibleIf": "containsAny(selectedOptionIds, [{opt-yes}])"}
        ]}]
      }
    ]
  },
  "categories": [{"id": "unknown", "name": "Cardiac"}]
}`

func convertFixture(t *testing.T) (Questionnaire, []string) {
	t.Helper()
	var e Export
	if err := json.Unmarshal([]byte(fixture), &e); err != nil {
		t.Fatal(err)
	}
	return Convert(e, fakeLib{})
}

func TestConvert(t *testing.T) {
	q, warnings := convertFixture(t)
	if q.Name != "HJE Full HQ" || len(q.Sets) != 2 {
		t.Fatalf("got %q with %d sets", q.Name, len(q.Sets))
	}
	if q.Sets[0].Icon != "user" || q.Sets[1].Icon != "heart" {
		t.Errorf("icons: %q, %q", q.Sets[0].Icon, q.Sets[1].Icon)
	}
	pages := q.Sets[0].Content["pages"].([]any)
	p1 := pages[0].(map[string]any)
	if p1["title"] != "Smoking" || p1["clinicalSummary"] != true {
		t.Errorf("page 1: %v", p1)
	}
	els := p1["elements"].([]any)
	if len(els) != 4 {
		t.Fatalf("page 1 has %d elements, want 4 (the summary becomes a page flag)", len(els))
	}
	smoke, many, allergy, statement := els[0].(map[string]any), els[1].(map[string]any), els[2].(map[string]any), els[3].(map[string]any)

	// Yes / No with stable option IDs and outputs on Yes.
	if smoke["type"] != "radiogroup" || smoke["yesNo"] != true || smoke["isRequired"] != true {
		t.Errorf("smoke: %v", smoke)
	}
	yes := smoke["choices"].([]any)[0].(map[string]any)
	outs := yes["clinicalOutputs"].([]any)
	if len(outs) != 1 {
		t.Fatalf("yes outputs: %v", yes)
	}
	out := outs[0].(map[string]any)
	codes := out["codes"].([]any)
	if len(codes) != 2 || codes[0].(map[string]any)["display"] != "Smoker" || codes[1].(map[string]any)["set"] != "ICD10" {
		t.Errorf("codes: %v", codes)
	}
	if note := out["note"].(map[string]any); note["text"] != "Current smoker" || note["category"] != "Lifestyle" {
		t.Errorf("note: %v", note)
	}

	// Conditions in the Builder's simple form; long text; %s becomes {answer}; category from the export.
	wantIf := "{" + smoke["name"].(string) + "} = '" + yes["value"].(string) + "'"
	if many["visibleIf"] != wantIf || many["type"] != "comment" || many["isRequired"] != nil {
		t.Errorf("many: %v, want visibleIf %s", many, wantIf)
	}
	qOut := many["clinicalOutputs"].([]any)[0].(map[string]any)["note"].(map[string]any)
	if qOut["text"] != "Smokes {answer} a day" || qOut["category"] != "Cardiac" {
		t.Errorf("question output: %v", qOut)
	}

	// None of these is exclusive; "is not" on Select Many is notcontains.
	none := allergy["choices"].([]any)[1].(map[string]any)
	if allergy["type"] != "checkbox" || none["isExclusive"] != true {
		t.Errorf("allergy: %v", allergy)
	}
	if want := "{" + allergy["name"].(string) + "} notcontains '" + none["value"].(string) + "'"; statement["visibleIf"] != want {
		t.Errorf("statement visibleIf %v, want %s", statement["visibleIf"], want)
	}
	if statement["type"] != "html" || statement["clinicianOnly"] != true || statement["html"] != "<p>Review the patient's allergies.</p><p>This entry will appear on the POA Summary.</p>" {
		t.Errorf("statement: %v", statement)
	}

	// Page logic with and / not; medication type; several dates become a list of dates.
	p2 := pages[1].(map[string]any)
	if vi, _ := p2["visibleIf"].(string); !strings.Contains(vi, " and !({") || !strings.Contains(vi, "contains '") {
		t.Errorf("page 2 visibleIf: %v", p2["visibleIf"])
	}
	med := p2["elements"].([]any)[0].(map[string]any)
	if med["type"] != "medication" || med["medicationType"] != "non_prescribed" {
		t.Errorf("medication: %v", med)
	}
	if dates := p2["elements"].([]any)[1].(map[string]any); dates["type"] != "matrixdynamic" || dates["dateList"] != true {
		t.Errorf("several dates: %v", dates)
	}

	// A condition on another Question Set is dropped with a warning.
	heart := q.Sets[1].Content["pages"].([]any)[0].(map[string]any)["elements"].([]any)[0].(map[string]any)
	if heart["visibleIf"] != nil {
		t.Errorf("cross-set condition kept: %v", heart["visibleIf"])
	}
	all := strings.Join(warnings, "\n")
	for _, w := range []string{"ICD10 Z72.0 is not in our library", "another Question Set"} {
		if !strings.Contains(all, w) {
			t.Errorf("missing warning %q in:\n%s", w, all)
		}
	}
}

func TestSimplify(t *testing.T) {
	for in, want := range map[string]string{
		"({q_a} = 'o_b')":                      "{q_a} = 'o_b'",
		"!({q_a} = 'o_b')":                     "{q_a} <> 'o_b'",
		"!({q_a} contains 'o_b')":              "{q_a} notcontains 'o_b'",
		"({q_a} = 'o_b' or {q_a} = 'o_c')":     "({q_a} = 'o_b' or {q_a} = 'o_c')",
		"({q_a} = 'o_b') and ({q_c} = 'o_d')": "({q_a} = 'o_b') and ({q_c} = 'o_d')",
	} {
		if got := simplify(in); got != want {
			t.Errorf("simplify(%q) = %q, want %q", in, got, want)
		}
	}
}
