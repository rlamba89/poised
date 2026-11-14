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

// S48: the list is packages/clinical's (list.json, written by npm run assessments), in its order.
func TestAllFromTheList(t *testing.T) {
	var ids, names []string
	for _, d := range All {
		ids = append(ids, d.ID)
		names = append(names, d.Name)
		if len(d.Content) == 0 {
			t.Errorf("%s has no content", d.ID)
		}
	}
	if strings.Join(ids, ",") != "stopbang,dasi,prisma7,apfel,must,morse,vte,rockwood" ||
		strings.Join(names, ",") != "STOP-Bang,DASI,PRISMA-7,Apfel PONV,MUST,Morse Fall Scale,VTE risk,Clinical Frailty Scale" {
		t.Errorf("All = %v %v", ids, names)
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
		// S47 verification 2, minor 5: an observation fills a number item only (the BMI).
		{"an observation on a Yes/No item", `{"stopbang_snoring": "o_yes"}`, `{"stopbang_snoring": "observation"}`, "where"},
		{"an empty answer", `{"stopbang_neck": null}`, `{"stopbang_neck": "clinician"}`, "Neck: that answer isn't"},
		{"an empty age", `{"patientAge": null}`, `{"patientAge": "profile"}`, "Age"},
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
	all["patientAge"] = json.RawMessage(`62`)
	all["patientSex"] = json.RawMessage(`"male"`)
	if got := Unanswered(d.Content, all); got != "" {
		t.Errorf("every item answered: %q", got)
	}
	delete(all, "stopbang_neck")
	if got := Unanswered(d.Content, all); got != "Neck" {
		t.Errorf("no neck: %q, want Neck", got)
	}
	// The age and sex the content names are needed too (S47 verification 4).
	all["stopbang_neck"] = json.RawMessage(`"o_no"`)
	delete(all, "patientSex")
	if got := Unanswered(d.Content, all); got != "Sex" {
		t.Errorf("no sex: %q, want Sex", got)
	}
	if got := Unanswered(d.Content, map[string]json.RawMessage{}); got == "" {
		t.Error("nothing answered, but none missing")
	}
	// DASI names neither: its twelve items are enough.
	dasi, _ := Lookup("dasi")
	twelve := map[string]json.RawMessage{}
	for _, el := range questionList(dasi.Content) {
		twelve[el.Name] = json.RawMessage(`"o_yes"`)
	}
	if len(twelve) != 12 {
		t.Fatalf("DASI has %d questions, want 12", len(twelve))
	}
	if got := Unanswered(dasi.Content, twelve); got != "" {
		t.Errorf("DASI answered: %q", got)
	}
	delete(twelve, "dasi_run")
	if got := Unanswered(dasi.Content, twelve); got != "Run a short distance" {
		t.Errorf("DASI without running: %q", got)
	}
	// PRISMA-7 names the age and sex; Apfel the sex only.
	prisma, _ := Lookup("prisma7")
	if got := Unanswered(prisma.Content, map[string]json.RawMessage{"prisma7_limit": json.RawMessage(`"o_no"`), "prisma7_help": json.RawMessage(`"o_no"`), "prisma7_home": json.RawMessage(`"o_no"`), "prisma7_count_on": json.RawMessage(`"o_no"`), "prisma7_aid": json.RawMessage(`"o_no"`)}); got != "Age" {
		t.Errorf("PRISMA-7 without the age: %q", got)
	}
	apfel, _ := Lookup("apfel")
	got := Unanswered(apfel.Content, map[string]json.RawMessage{"apfel_smoking": json.RawMessage(`"o_no"`), "apfel_ponv": json.RawMessage(`"o_no"`), "apfel_opioids": json.RawMessage(`"o_yes"`), "patientAge": json.RawMessage(`40`)})
	if got != "Sex" {
		t.Errorf("Apfel without the sex: %q", got)
	}
}

// S48 verification 1 (Rahul, 7 Nov): a "profile" age or sex must be the patient's record's: the
// age on the day at the hospital, and the sex as recorded.
func TestProfileProblem(t *testing.T) {
	for _, c := range []struct {
		name, data, want string
	}{
		{"both as the record", `{"patientAge": 46, "patientSex": "female"}`, ""},
		{"neither sent", `{"dasi_run": "o_yes"}`, ""},
		{"a sex the record doesn't say", `{"patientSex": "male"}`, "Sex doesn't match the patient's record"},
		{"an age that isn't today's", `{"patientAge": 47}`, "Age doesn't match the patient's record"},
	} {
		var data map[string]json.RawMessage
		_ = json.Unmarshal([]byte(c.data), &data)
		got := ProfileProblem(data, 46, "female")
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	// A record of Other or Not known can't back a male or female sex.
	if got := ProfileProblem(map[string]json.RawMessage{"patientSex": json.RawMessage(`"male"`)}, 46, "other"); got == "" {
		t.Error("male accepted for a patient recorded as Other")
	}
}

// S49: MUST's numbers (the BMI and today's weight may be the observations'; the weight lost the
// HQ's, 0 when the patient said No), VTE's 30 items and the age, Rockwood's one level.
func TestS49Assessments(t *testing.T) {
	raw := func(s string) map[string]json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	src := func(s string) map[string]string {
		var m map[string]string
		_ = json.Unmarshal([]byte(s), &m)
		return m
	}
	must, _ := Lookup("must")
	mustData := raw(`{"must_bmi": 19.5, "must_weight": 64, "must_wtloss_kg": 0, "must_acute": "o_no"}`)
	mustSources := src(`{"must_bmi": "observation", "must_weight": "observation", "must_wtloss_kg": "hq", "must_acute": "clinician"}`)
	if got := Check(must.Content, mustData, mustSources); got != "" {
		t.Errorf("MUST: %q", got)
	}
	if got := Unanswered(must.Content, mustData); got != "" {
		t.Errorf("MUST answered: %q", got)
	}
	for _, c := range []struct{ name, data, sources, want string }{
		{"a weight of 0", `{"must_weight": 0}`, `{"must_weight": "observation"}`, "Weight now: a number from 1 to 400."},
		{"a negative weight lost", `{"must_wtloss_kg": -1}`, `{"must_wtloss_kg": "hq"}`, "Weight lost: a number from 0 to 100."},
		{"the HQ's Yes/No isn't an item", `{"must_wtloss": "o_no"}`, `{"must_wtloss": "hq"}`, "doesn't belong"},
		{"the percentage isn't an answer", `{"must_wtloss_pct": 5}`, `{"must_wtloss_pct": "clinician"}`, "doesn't belong"},
	} {
		if got := Check(must.Content, raw(c.data), src(c.sources)); !strings.Contains(got, c.want) {
			t.Errorf("MUST, %s: %q, want %q", c.name, got, c.want)
		}
	}
	delete(mustData, "must_wtloss_kg")
	if got := Unanswered(must.Content, mustData); got != "Weight lost" {
		t.Errorf("MUST without the weight lost: %q", got)
	}

	vte, _ := Lookup("vte")
	if n := len(questionList(vte.Content)); n != 29 {
		t.Errorf("VTE has %d questions, want 29 (and the age)", n)
	}
	every := map[string]json.RawMessage{}
	for _, el := range questionList(vte.Content) {
		every[el.Name] = json.RawMessage(`"o_no"`)
	}
	every["vte_bmi"] = json.RawMessage(`31.2`)
	if got := Unanswered(vte.Content, every); got != "Age" {
		t.Errorf("VTE without the age: %q", got)
	}
	every["patientAge"] = json.RawMessage(`64`)
	if got := Unanswered(vte.Content, every); got != "" {
		t.Errorf("VTE answered: %q", got)
	}

	cfs, _ := Lookup("rockwood")
	if got := Unanswered(cfs.Content, map[string]json.RawMessage{}); got != "Level" {
		t.Errorf("Rockwood without a level: %q", got)
	}
	if got := Check(cfs.Content, raw(`{"cfs_level": "o_cfs_10"}`), src(`{"cfs_level": "clinician"}`)); got == "" {
		t.Error("a tenth level accepted")
	}
	if got := Check(cfs.Content, raw(`{"cfs_level": "o_cfs_5"}`), src(`{"cfs_level": "clinician"}`)); got != "" {
		t.Errorf("Rockwood level 5: %q", got)
	}
}
