package chapter

import (
	"encoding/json"
	"strings"
	"testing"
)

const mixed = `{
  "testCases": [{"id": "tc_1"}],
  "pages": [{
    "name": "p_1",
    "elements": [
      {"type": "radiogroup", "name": "q_smoke", "choices": [{"value": "o_y", "clinicalOutputs": [{"id": "out_1", "note": {"text": "Smoker"}}]}]},
      {"type": "comment", "name": "q_secret", "clinicianOnly": true},
      {"type": "panel", "name": "g_1", "elements": [
        {"type": "text", "name": "q_inner"},
        {"type": "text", "name": "q_inner_clin", "clinicianOnly": true}
      ]},
      {"type": "paneldynamic", "name": "q_meds", "templateElements": [{"type": "text", "name": "q_dose", "clinicianOnly": true}]}
    ]
  }, {
    "name": "p_2", "clinicianOnly": true, "elements": [{"type": "text", "name": "q_on_clin_page"}]
  }]
}`

func TestForPatient(t *testing.T) {
	out, names, err := ForPatient(json.RawMessage(mixed))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, gone := range []string{"q_secret", "q_inner_clin", "q_dose", "clinicalOutputs", "Smoker", "testCases", "q_on_clin_page"} {
		if strings.Contains(s, gone) {
			t.Errorf("patient JSON still has %q: %s", gone, s)
		}
	}
	for _, kept := range []string{"q_smoke", "g_1", "q_inner", "q_meds"} {
		if !names[kept] {
			t.Errorf("names is missing %q: %v", kept, names)
		}
	}
	if names["q_secret"] || names["p_1"] {
		t.Errorf("names has clinician-only content or a page: %v", names)
	}
}

func TestFilterAnswers(t *testing.T) {
	names := map[string]bool{"q_smoke": true, "q_other": true}
	data := map[string]json.RawMessage{
		"q_smoke":         json.RawMessage(`"o_y"`),
		"q_other-Comment": json.RawMessage(`"Pipe"`),
		"q_secret":        json.RawMessage(`"injected"`),
	}
	kept, dropped := FilterAnswers(data, names)
	if !dropped || len(kept) != 2 || kept["q_secret"] != nil {
		t.Errorf("kept %v (dropped %v): want q_smoke and q_other-Comment only", kept, dropped)
	}
}
