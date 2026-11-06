package assessment

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefinitions(t *testing.T) {
	d, ok := Lookup("stopbang")
	if !ok || d.Name != "STOP-Bang" {
		t.Fatalf("stopbang = %+v, %v", d, ok)
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("an unknown assessment was found")
	}
	var v map[string]any
	if err := json.Unmarshal(d.Content, &v); err != nil || v["pages"] == nil {
		t.Fatalf("its content isn't a survey: %v", err)
	}
	names := map[string]bool{}
	for n := range questions(d.Content) {
		names[n] = true
	}
	for _, k := range []string{"stopbang_bmi", "stopbang_snoring", "stopbang_tired", "stopbang_observed", "stopbang_pressure", "stopbang_neck"} {
		if !names[k] {
			t.Errorf("no question %s", k)
		}
	}
	if names["stopbang_score"] || names["stopbang"] {
		t.Error("the score and the Group aren't answers")
	}
}

func TestCheck(t *testing.T) {
	d, _ := Lookup("stopbang")
	ok := `{"stopbang_snoring": "o_yes", "stopbang_neck": "o_no", "stopbang_bmi": 36.4}`
	okSources := `{"stopbang_snoring": "hq", "stopbang_neck": "clinician", "stopbang_bmi": "observation"}`
	for _, c := range []struct {
		name, data, sources, want string
	}{
		{"every item, each with its source", ok, okSources, ""},
		{"nothing yet", `{}`, `{}`, ""},
		{"an answer that isn't a question of it", `{"q_other": "o_yes"}`, `{"q_other": "hq"}`, "doesn't belong"},
		{"the score isn't an answer", `{"stopbang_score": 5}`, `{"stopbang_score": "hq"}`, "doesn't belong"},
		{"an option it hasn't", `{"stopbang_neck": "o_maybe"}`, `{"stopbang_neck": "clinician"}`, "Neck"},
		{"an option as a number", `{"stopbang_neck": 1}`, `{"stopbang_neck": "clinician"}`, "Neck"},
		{"a BMI as text", `{"stopbang_bmi": "36"}`, `{"stopbang_bmi": "clinician"}`, "BMI"},
		{"a BMI below its minimum", `{"stopbang_bmi": 2}`, `{"stopbang_bmi": "clinician"}`, "BMI: a number from 5 to 150."},
		{"a BMI above its maximum", `{"stopbang_bmi": 151}`, `{"stopbang_bmi": "clinician"}`, "BMI: a number from 5 to 150."},
		{"an answer without a source", ok, `{"stopbang_snoring": "hq"}`, "where"},
		{"a source without an answer", `{}`, `{"stopbang_neck": "clinician"}`, "where"},
		{"a source that isn't one", `{"stopbang_neck": "o_no"}`, `{"stopbang_neck": "guess"}`, "where"},
		// S47 answers 7: the patient's age and sex are stored at Confirm, source profile, and only they.
		{"the age and sex from the record", `{"patientAge": 62, "patientSex": "male"}`, `{"patientAge": "profile", "patientSex": "profile"}`, ""},
		{"female", `{"patientSex": "female"}`, `{"patientSex": "profile"}`, ""},
		{"profile on another item", `{"stopbang_neck": "o_no"}`, `{"stopbang_neck": "profile"}`, "where"},
		{"the age from somewhere else", `{"patientAge": 62}`, `{"patientAge": "clinician"}`, "where"},
		{"an age that isn't a whole number", `{"patientAge": 62.5}`, `{"patientAge": "profile"}`, "Age"},
		{"an age as text", `{"patientAge": "62"}`, `{"patientAge": "profile"}`, "Age"},
		{"a negative age", `{"patientAge": -1}`, `{"patientAge": "profile"}`, "Age"},
		{"a sex the score can't use", `{"patientSex": "other"}`, `{"patientSex": "profile"}`, "Sex"},
	} {
		var data map[string]json.RawMessage
		var sources map[string]string
		if err := json.Unmarshal([]byte(c.data), &data); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(c.sources), &sources); err != nil {
			t.Fatal(err)
		}
		got := Check(d.Content, data, sources)
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestUnanswered(t *testing.T) {
	d, _ := Lookup("stopbang")
	all := map[string]json.RawMessage{}
	for _, k := range []string{"stopbang_snoring", "stopbang_tired", "stopbang_observed", "stopbang_pressure", "stopbang_neck"} {
		all[k] = json.RawMessage(`"o_no"`)
	}
	all["stopbang_bmi"] = json.RawMessage(`30`)
	if got := Unanswered(d.Content, all); got != "" {
		t.Errorf("every item answered: %q", got)
	}
	delete(all, "stopbang_neck")
	if got := Unanswered(d.Content, all); got != "Neck" {
		t.Errorf("no neck: %q, want Neck", got)
	}
	if got := Unanswered(d.Content, map[string]json.RawMessage{}); got == "" {
		t.Error("nothing answered, but none missing")
	}
}
