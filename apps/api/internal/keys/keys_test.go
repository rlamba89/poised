package keys

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const chapter = `{
  "pages": [{
    "name": "p_1",
    "elements": [
      {"type": "radiogroup", "name": "q_smokes", "title": "Do you smoke?", "alwaysFresh": true,
       "choices": [{"value": "o_yes", "text": "Yes"}, {"value": "o_no", "text": {"default": "No", "fr": "Non"}}, "Sometimes"]},
      {"type": "checkbox", "name": "q_symptoms", "title": {"default": "Any of these?"}, "choices": [{"value": "o_a"}],
       "showOtherItem": true, "otherText": "Something else"},
      {"type": "text", "name": "q_weight", "title": "Weight", "inputType": "number", "unit": "kg"},
      {"type": "text", "name": "q_dob", "title": "Date of birth", "inputType": "date"},
      {"type": "comment", "name": "q_note", "title": "Nurse note", "clinicianOnly": true},
      {"type": "html", "name": "q_info", "html": "<p>Read this</p>"},
      {"type": "expression", "name": "q_calc", "expression": "1"},
      {"type": "profile", "name": "q_profile"},
      {"type": "panel", "name": "g_heart", "clinicianOnly": true, "elements": [
        {"type": "boolean", "name": "q_heart", "title": "Heart problems?"}
      ]},
      {"type": "paneldynamic", "name": "q_meds", "title": "Your medicines", "templateElements": [
        {"type": "text", "name": "q_med_name", "title": "Medicine"},
        {"type": "dropdown", "name": "q_med_when", "title": "When", "choices": [{"value": "o_am", "text": "Morning"}]},
        {"type": "html", "name": "q_med_help", "html": "x"}
      ]},
      {"type": "matrixdynamic", "name": "q_ops", "title": "Operations", "columns": [{"name": "what", "title": "Operation"}, {"name": "year"}]},
      {"type": "bmi", "name": "q_bmi"}
    ]
  }, {
    "name": "p_2", "clinicianOnly": true, "elements": [{"type": "text", "name": "q_on_clin_page", "title": "Pulse"}]
  }]
}`

func byName(t *testing.T, qs []Question) map[string]Question {
	t.Helper()
	m := map[string]Question{}
	for _, q := range qs {
		if _, dup := m[q.Name]; dup {
			t.Fatalf("Walk listed %s twice", q.Name)
		}
		m[q.Name] = q
	}
	return m
}

func TestWalk(t *testing.T) {
	qs, err := Walk(json.RawMessage(chapter))
	if err != nil {
		t.Fatal(err)
	}
	m := byName(t, qs)

	// Every element but a group (panel) is listed, at any depth, dynamic-panel children included.
	want := []string{"q_smokes", "q_symptoms", "q_weight", "q_dob", "q_note", "q_info", "q_calc", "q_profile", "q_heart",
		"q_meds", "q_med_name", "q_med_when", "q_med_help", "q_ops", "q_bmi", "q_on_clin_page"}
	if len(qs) != len(want) {
		t.Errorf("Walk listed %d elements, want %d: %v", len(qs), len(want), m)
	}
	for _, n := range want {
		if _, ok := m[n]; !ok {
			t.Errorf("Walk missed %s", n)
		}
	}
	if _, ok := m["g_heart"]; ok {
		t.Error("Walk listed a group")
	}

	smokes := m["q_smokes"]
	if smokes.Title != "Do you smoke?" || smokes.Kind != "single" || !smokes.AlwaysFresh || !smokes.Answerable {
		t.Errorf("q_smokes = %+v", smokes)
	}
	wantOpts := []Option{{"o_yes", "Yes"}, {"o_no", "No"}, {"Sometimes", "Sometimes"}} // the default language; a plain string is both
	if !reflect.DeepEqual(smokes.Options, wantOpts) {
		t.Errorf("q_smokes options = %v, want %v", smokes.Options, wantOpts)
	}
	symptoms := m["q_symptoms"]
	if symptoms.Title != "Any of these?" || symptoms.Kind != "multiple" ||
		!reflect.DeepEqual(symptoms.Options, []Option{{"o_a", "o_a"}, {"other", "Something else"}}) {
		t.Errorf("q_symptoms = %+v (an option without text shows its value; Other is an option)", symptoms)
	}
	if w := m["q_weight"]; w.Kind != "number" || w.Unit != "kg" || w.Options != nil {
		t.Errorf("q_weight = %+v", w)
	}
	if m["q_dob"].Kind != "date" || m["q_note"].Kind != "text" || m["q_ops"].Kind != "list" || m["q_meds"].Kind != "list" || m["q_bmi"].Kind != "bmi" {
		t.Errorf("kinds: dob %s, note %s, ops %s, meds %s, bmi %s", m["q_dob"].Kind, m["q_note"].Kind, m["q_ops"].Kind, m["q_meds"].Kind, m["q_bmi"].Kind)
	}

	// Clinician-only: on the question, or on a group or page around it.
	for n, clin := range map[string]bool{"q_note": true, "q_heart": true, "q_on_clin_page": true, "q_smokes": false, "q_med_name": false} {
		if m[n].ClinicianOnly != clin {
			t.Errorf("%s clinician-only = %v, want %v", n, m[n].ClinicianOnly, clin)
		}
	}
	// Statements, calculations and the profile aren't answered by anyone.
	for _, n := range []string{"q_info", "q_calc", "q_profile", "q_med_help"} {
		if m[n].Answerable {
			t.Errorf("%s is answerable", n)
		}
	}
	// A dynamic panel's children belong to its rows; its columns are their texts.
	if m["q_med_name"].InList != "q_meds" || m["q_smokes"].InList != "" || m["q_heart"].InList != "" {
		t.Errorf("InList: med_name %q, smokes %q, heart %q", m["q_med_name"].InList, m["q_smokes"].InList, m["q_heart"].InList)
	}
	if got := m["q_meds"].Columns; !reflect.DeepEqual(got, []Column{{Name: "q_med_name", Text: "Medicine", Kind: "text"}, {Name: "q_med_when", Text: "When", Kind: "single", Options: []Option{{"o_am", "Morning"}}}}) {
		t.Errorf("q_meds columns = %v", got)
	}
	if got := m["q_ops"].Columns; !reflect.DeepEqual(got, []Column{{Name: "what", Text: "Operation"}, {Name: "year", Text: "year"}}) {
		t.Errorf("q_ops columns = %v", got)
	}
	if !reflect.DeepEqual(m["q_med_when"].Options, []Option{{"o_am", "Morning"}}) {
		t.Errorf("q_med_when options = %v", m["q_med_when"].Options)
	}
}

func TestWalkBadContent(t *testing.T) {
	if _, err := Walk(json.RawMessage(`[1]`)); err == nil {
		t.Error("Walk accepted a list")
	}
	qs, err := Walk(json.RawMessage(`{}`))
	if err != nil || len(qs) != 0 {
		t.Errorf("an empty chapter: %v, %v", qs, err)
	}
}

func TestWalkSetsMarksClinicianSets(t *testing.T) {
	qs, err := WalkSets([]Set{
		{Name: "About you", Audience: "patient", Content: json.RawMessage(`{"pages":[{"name":"p","elements":[{"type":"text","name":"q_a"}]}]}`)},
		{Name: "Nurse checks", Audience: "clinician", Content: json.RawMessage(`{"pages":[{"name":"p","elements":[{"type":"text","name":"q_b"}]}]}`)},
	})
	if err != nil || len(qs) != 2 {
		t.Fatalf("WalkSets = %v, %v", qs, err)
	}
	if qs[0].Set != "About you" || qs[0].ClinicianOnly || qs[1].Set != "Nurse checks" || !qs[1].ClinicianOnly {
		t.Errorf("WalkSets = %+v", qs)
	}
}

func set(name, elements string) Set {
	return Set{Name: name, Audience: "patient", Content: json.RawMessage(`{"pages":[{"name":"p","elements":[` + elements + `]}]}`)}
}

func walkSets(t *testing.T, sets ...Set) []Placed {
	t.Helper()
	qs, err := WalkSets(sets)
	if err != nil {
		t.Fatal(err)
	}
	return qs
}

func TestDuplicates(t *testing.T) {
	ok := walkSets(t,
		set("About you", `{"type":"text","name":"q_a","title":"A"},{"type":"panel","name":"g_1","elements":[]}`),
		set("Your health", `{"type":"text","name":"q_b","title":"B"},{"type":"panel","name":"g_1","elements":[]}`),
	)
	if p := Duplicates(ok); len(p) != 0 {
		t.Errorf("no duplicate questions (groups don't count): %v", p)
	}

	two := walkSets(t,
		set("About you", `{"type":"text","name":"q_a","title":"Do you smoke?"}`),
		set("Your health", `{"type":"paneldynamic","name":"q_list","templateElements":[{"type":"text","name":"q_a","title":"Smoking"}]}`),
	)
	p := Duplicates(two)
	if len(p) != 1 || p[0].Set != "Your health" ||
		!strings.Contains(p[0].Message, "q_a") || !strings.Contains(p[0].Message, "“About you”") || !strings.Contains(p[0].Message, "“Your health”") ||
		!strings.Contains(p[0].Message, "Do you smoke?") || !strings.Contains(p[0].Message, "Smoking") {
		t.Errorf("a key in two sets (at any depth): %v", p)
	}

	same := walkSets(t, set("About you", `{"type":"text","name":"q_a"},{"type":"text","name":"q_a"},{"type":"text","name":"q_a"}`))
	if p := Duplicates(same); len(p) != 2 || !strings.Contains(p[0].Message, "twice in “About you”") {
		t.Errorf("a key three times in one set gives two problems: %v", p)
	}
}

func published(q string, versionNo int, content string) Published {
	qs, err := Walk(json.RawMessage(`{"pages":[{"name":"p","elements":[` + content + `]}]}`))
	if err != nil || len(qs) == 0 {
		panic(fmt.Sprint(content, err))
	}
	return Published{Questionnaire: q, VersionNo: versionNo, Question: qs[0]}
}

func TestLockProblems(t *testing.T) {
	pub := map[string][]Published{
		"q_smokes": {published("Demo pre-op HQ", 1,
			`{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?","choices":[{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"}]}`)},
		"q_name": {published("Demo pre-op HQ", 1, `{"type":"text","name":"q_name","title":"Your name"}`)},
	}
	problems := func(t *testing.T, elements string) []Problem {
		t.Helper()
		return LockProblems(walkSets(t, set("About you", elements)), pub)
	}

	t.Run("the same options, reordered, and a reworded question: no problem", func(t *testing.T) {
		p := problems(t, `{"type":"radiogroup","name":"q_smokes","title":"Do you currently smoke?","choices":[{"value":"o_no","text":"No"},{"value":"o_yes","text":" Yes "}]},
			{"type":"text","name":"q_name","title":"Full name"},
			{"type":"radiogroup","name":"q_new","title":"New","choices":[{"value":"o_x","text":"X"}]}`)
		if len(p) != 0 {
			t.Errorf("%v", p)
		}
	})
	t.Run("a removed option", func(t *testing.T) {
		p := problems(t, `{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?","choices":[{"value":"o_yes","text":"Yes"}]}`)
		if len(p) != 1 || p[0].Set != "About you" || p[0].Message !=
			"“Do you smoke?”: the option “No” was removed. A published question's answers can't change: add a new question instead (published in “Demo pre-op HQ” v1)." {
			t.Errorf("%v", p)
		}
	})
	t.Run("an added option", func(t *testing.T) {
		p := problems(t, `{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?","choices":[{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"},{"value":"o_ex","text":"I used to"}]}`)
		if len(p) != 1 || !strings.Contains(p[0].Message, "the option “I used to” was added") {
			t.Errorf("%v", p)
		}
	})
	t.Run("a reworded option", func(t *testing.T) {
		p := problems(t, `{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?","choices":[{"value":"o_yes","text":"Yes, daily"},{"value":"o_no","text":"No"}]}`)
		if len(p) != 1 || !strings.Contains(p[0].Message, "the option “Yes” was reworded to “Yes, daily”") {
			t.Errorf("%v", p)
		}
	})
	t.Run("a translation isn't a reword", func(t *testing.T) {
		p := problems(t, `{"type":"radiogroup","name":"q_smokes","choices":[{"value":"o_yes","text":{"default":"Yes","fr":"Oui"}},{"value":"o_no","text":"No"}]}`)
		if len(p) != 0 {
			t.Errorf("%v", p)
		}
	})
	t.Run("changed into a question without options: its kind of answer changed, one problem", func(t *testing.T) {
		p := problems(t, `{"type":"text","name":"q_smokes","title":"Do you smoke?"}`)
		if len(p) != 1 || !strings.Contains(p[0].Message, "its kind of answer changed from single choice to text") {
			t.Errorf("%v", p)
		}
	})
	t.Run("a published text question given options", func(t *testing.T) {
		p := problems(t, `{"type":"radiogroup","name":"q_name","title":"Your name","choices":[{"value":"o_a","text":"A"}]}`)
		if len(p) != 1 || !strings.Contains(p[0].Message, "from text to single choice") {
			t.Errorf("%v", p)
		}
	})
	t.Run("inside a repeating group too", func(t *testing.T) {
		p := problems(t, `{"type":"paneldynamic","name":"q_list","templateElements":[{"type":"radiogroup","name":"q_smokes","choices":[{"value":"o_yes","text":"Yes"}]}]}`)
		if len(p) != 1 {
			t.Errorf("%v", p)
		}
	})
	t.Run("one change against two sources is one problem naming both", func(t *testing.T) {
		both := map[string][]Published{"q_smokes": {pub["q_smokes"][0], published("Day case HQ", 3,
			`{"type":"radiogroup","name":"q_smokes","choices":[{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"}]}`)}}
		p := LockProblems(walkSets(t, set("About you", `{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?","choices":[{"value":"o_yes","text":"Yes"}]}`)), both)
		if len(p) != 1 || !strings.Contains(p[0].Message, "(published in “Demo pre-op HQ” v1 and “Day case HQ” v3)") {
			t.Errorf("%v", p)
		}
	})
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"Do you smoke?":        "do you smoke",
		"  DO  you\tsmoke ?! ": "do you smoke",
		"Do you smoke—daily?":  "do you smokedaily",
		"Café, 2 a day":        "café 2 a day",
		"":                     "",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func q(t *testing.T, element string) Question {
	t.Helper()
	qs, err := Walk(json.RawMessage(`{"pages":[{"name":"p","elements":[` + element + `]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return qs[0]
}

func TestSameWording(t *testing.T) {
	smokes := q(t, `{"type":"radiogroup","name":"q_a","title":"Do you smoke?","choices":[{"value":"o_y","text":"Yes"},{"value":"o_n","text":"No"}]}`)
	cases := []struct {
		name string
		el   string
		same bool
	}{
		{"same text, case and punctuation aside, options in another order with other values",
			`{"type":"dropdown","name":"q_b","title":"do you SMOKE","choices":[{"value":"o_2","text":"no"},{"value":"o_1","text":"Yes."}]}`, true},
		{"another text", `{"type":"radiogroup","name":"q_b","title":"Do you currently smoke?","choices":[{"value":"o_y","text":"Yes"},{"value":"o_n","text":"No"}]}`, false},
		{"another kind of answer", `{"type":"checkbox","name":"q_b","title":"Do you smoke?","choices":[{"value":"o_y","text":"Yes"},{"value":"o_n","text":"No"}]}`, false},
		{"an option text differs", `{"type":"radiogroup","name":"q_b","title":"Do you smoke?","choices":[{"value":"o_y","text":"Yes"},{"value":"o_n","text":"Never"}]}`, false},
		{"an extra option", `{"type":"radiogroup","name":"q_b","title":"Do you smoke?","choices":[{"value":"o_y","text":"Yes"},{"value":"o_n","text":"No"},{"value":"o_x","text":"Sometimes"}]}`, false},
	}
	for _, c := range cases {
		if got := SameWording(smokes, q(t, c.el)); got != c.same {
			t.Errorf("%s: SameWording = %v, want %v", c.name, got, c.same)
		}
	}

	kg := q(t, `{"type":"text","name":"q_w","title":"Your weight","inputType":"number","unit":"kg"}`)
	if SameWording(kg, q(t, `{"type":"text","name":"q_x","title":"Your weight","inputType":"number","unit":"lb"}`)) {
		t.Error("numbers in other units matched")
	}
	if !SameWording(kg, q(t, `{"type":"text","name":"q_x","title":"Your weight","inputType":"number","unit":"kg"}`)) {
		t.Error("the same number question didn't match")
	}
	if SameWording(q(t, `{"type":"text","name":"q_a"}`), q(t, `{"type":"text","name":"q_b"}`)) {
		t.Error("two questions with no text matched")
	}
	// The editor's placeholder texts for a new question aren't wording anyone chose.
	for _, el := range []string{
		`{"type":"paneldynamic","name":"q_a","title":"Group","templateElements":[{"type":"text","name":"q_b","title":"Text question"}]}`,
		`{"type":"text","name":"q_a","title":"Text question"}`,
		`{"type":"radiogroup","name":"q_a","title":"Select One question","choices":[{"value":"o_1","text":"Option 1"}]}`,
		`{"type":"radiogroup","name":"q_a","title":"Yes / No question","choices":[{"value":"o_1","text":"Yes"}]}`,
	} {
		if a := q(t, el); SameWording(a, a) {
			t.Errorf("a placeholder text matched: %s", el)
		}
	}

	list := q(t, `{"type":"paneldynamic","name":"q_l","title":"Your medicines","templateElements":[{"type":"text","name":"a","title":"Medicine"},{"type":"text","name":"b","title":"Dose"}]}`)
	if !SameWording(list, q(t, `{"type":"paneldynamic","name":"q_m","title":"Your medicines","templateElements":[{"type":"text","name":"x","title":"dose"},{"type":"text","name":"y","title":"Medicine"}]}`)) {
		t.Error("the same repeating list didn't match")
	}
	if SameWording(list, q(t, `{"type":"paneldynamic","name":"q_m","title":"Your medicines","templateElements":[{"type":"text","name":"x","title":"Medicine"},{"type":"text","name":"y","title":"How often"}]}`)) {
		t.Error("a repeating list with other columns matched")
	}
}

func TestRepeatedTexts(t *testing.T) {
	qs := walkSets(t,
		set("A", `{"type":"comment","name":"q_1","title":"Please give details"},{"type":"text","name":"q_2","title":"Your name"}`),
		set("B", `{"type":"comment","name":"q_3","title":"Please give details."},{"type":"html","name":"q_4","html":"x"},{"type":"html","name":"q_5","html":"y"}`),
	)
	got := RepeatedTexts(qs)
	if !got["please give details"] || got["your name"] || len(got) != 1 {
		t.Errorf("RepeatedTexts = %v", got)
	}
}

func TestMatches(t *testing.T) {
	mine := walkSets(t,
		set("About you", `{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?","choices":[{"value":"o_y","text":"Yes"},{"value":"o_n","text":"No"}]},
			{"type":"text","name":"q_height","title":"Your height","inputType":"number","unit":"cm"},
			{"type":"comment","name":"q_details","title":"Please give details"},
			{"type":"comment","name":"q_details2","title":"Please give details"},
			{"type":"text","name":"q_new","title":"Something new"},
			{"type":"text","name":"q_fresh","title":"Today's weight","alwaysFresh":true},
			{"type":"text","name":"q_nurse","title":"Pulse","clinicianOnly":true},
			{"type":"html","name":"q_info","html":"x"},
			{"type":"paneldynamic","name":"q_list","title":"Medicines","templateElements":[{"type":"text","name":"q_med","title":"Medicine"}]}`),
	)
	others := []Version{
		{QuestionnaireID: "hq-1", Questionnaire: "Day case HQ", Questions: walkSets(t,
			set("S", `{"type":"radiogroup","name":"q_smokes","title":"Smoking?","choices":[{"value":"o_y","text":"Yes"}]},
				{"type":"text","name":"q_h","title":"your height","inputType":"number","unit":"cm"}`))},
		// An older version of the same questionnaire: a match in any version counts once.
		{QuestionnaireID: "hq-1", Questionnaire: "Day case HQ", Questions: walkSets(t, set("S", `{"type":"text","name":"q_fresh","title":"x"}`))},
		{QuestionnaireID: "hq-2", Questionnaire: "Knee HQ", Questions: walkSets(t,
			set("S", `{"type":"text","name":"q_h2","title":"Your height","inputType":"number","unit":"cm"},
				{"type":"comment","name":"q_d","title":"Please give details"},
				{"type":"text","name":"q_med","title":"Medicine"},
				{"type":"text","name":"q_pulse","title":"Pulse"}`))},
		// The same wording, but used twice in that questionnaire: never a wording match.
		{QuestionnaireID: "hq-3", Questionnaire: "Hip HQ", Questions: walkSets(t,
			set("S", `{"type":"text","name":"q_x","title":"Your height","inputType":"number","unit":"cm"},{"type":"text","name":"q_y","title":"Your height","inputType":"number","unit":"cm"}`))},
		// A clinician-only question in another HQ isn't a source.
		{QuestionnaireID: "hq-4", Questionnaire: "Nurse HQ", Questions: walkSets(t,
			Set{Name: "N", Audience: "clinician", Content: json.RawMessage(`{"pages":[{"name":"p","elements":[{"type":"text","name":"q_new","title":"Something new"}]}]}`)})},
	}
	got := Matches(mine, others)
	want := map[string][]Match{
		"q_smokes":   {{"hq-1", "Day case HQ", "key"}},
		"q_height":   {{"hq-1", "Day case HQ", "wording"}, {"hq-2", "Knee HQ", "wording"}},
		"q_details":  {}, // used twice in this questionnaire
		"q_details2": {},
		"q_new":      {},
		"q_fresh":    {{"hq-1", "Day case HQ", "key"}}, // the editor shows Always ask fresh instead
		"q_list":     {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Matches =\n%v\nwant\n%v", got, want)
	}
}

func TestWalkAnswerShapes(t *testing.T) {
	qs, err := Walk(json.RawMessage(`{"pages":[{"name":"p","elements":[
		{"type":"matrix","name":"q_grid","title":"How often do you","rows":["Run",{"value":"r_swim","text":"Swim"}],"columns":["Never",{"value":"c_a","text":{"default":"Always"}}]},
		{"type":"rating","name":"q_pain","title":"Pain","rateMin":0,"rateMax":10},
		{"type":"rating","name":"q_pain5","title":"Pain"},
		{"type":"rating","name":"q_count","rateMin":0,"rateCount":3,"rateStep":2},
		{"type":"rating","name":"q_faces","rateValues":[{"value":1,"text":"Sad"},"Happy"]},
		{"type":"boolean","name":"q_yn","labelTrue":"Agree"},
		{"type":"matrixdynamic","name":"q_ops","columns":["what",{"name":"when","title":"Year","cellType":"text","inputType":"number"},{"name":"kind","cellType":"dropdown","choices":["General"]}]},
		{"type":"matrixdropdown","name":"q_md","rows":["A"],"columns":[{"name":"c1","title":"C1","cellType":"checkbox"}]}
	]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	m := byName(t, qs)
	grid := m["q_grid"]
	if grid.Kind != "grid" || !reflect.DeepEqual(grid.Rows, []Option{{"Run", "Run"}, {"r_swim", "Swim"}}) ||
		!reflect.DeepEqual(grid.Options, []Option{{"Never", "Never"}, {"c_a", "Always"}}) {
		t.Errorf("grid = %+v (plain-string rows and columns are both a value and a text)", grid)
	}
	for name, want := range map[string]string{"q_pain": "0 to 10 by 1", "q_pain5": "1 to 5 by 1", "q_count": "0 to 4 by 2", "q_faces": ""} {
		if got := m[name].Scale; got != want {
			t.Errorf("%s scale = %q, want %q", name, got, want)
		}
	}
	if !reflect.DeepEqual(m["q_faces"].Options, []Option{{"1", "Sad"}, {"Happy", "Happy"}}) {
		t.Errorf("rating values = %v", m["q_faces"].Options)
	}
	if !reflect.DeepEqual(m["q_yn"].Options, []Option{{"true", "Agree"}, {"false", "No"}}) {
		t.Errorf("a Yes/No switch's labels = %v", m["q_yn"].Options)
	}
	if got := m["q_ops"].Columns; !reflect.DeepEqual(got, []Column{
		{Name: "what", Text: "what", Kind: ""}, {Name: "when", Text: "Year", Kind: "text:number"},
		{Name: "kind", Text: "kind", Kind: "dropdown", Options: []Option{{"General", "General"}}},
	}) {
		t.Errorf("q_ops columns = %+v", got)
	}
	if md := m["q_md"]; !reflect.DeepEqual(md.Rows, []Option{{"A", "A"}}) || len(md.Columns) != 1 || md.Columns[0].Kind != "checkbox" {
		t.Errorf("q_md = %+v", md)
	}
}

// Valid JSON in a shape the walk doesn't expect is skipped, never an error: one odd published
// chapter must not stop every publish in the trust (verification 1, item 4).
func TestWalkOddShapes(t *testing.T) {
	for _, content := range []string{
		`{"pages":[{"name":"p","elements":[{"type":"matrix","name":"q","columns":[1, null, "A"],"rows":"x"}]}]}`,
		`{"pages":[{"name":"p","elements":"x"}]}`,
		`{"pages":[1, "x", null, {"name":"p","elements":[1, {"type":"panel","elements":{"a":1}}, {"type":"text","name":5,"title":7}]}]}`,
		`{"pages":{"a":1}}`,
		`{"pages":[{"name":"p","elements":[{"type":"radiogroup","name":"q","choices":{"a":1},"rateMin":"x"}]}]}`,
	} {
		if _, err := Walk(json.RawMessage(content)); err != nil {
			t.Errorf("Walk(%s) = %v", content, err)
		}
	}
}

func TestLockFreezesTheWholeAnswer(t *testing.T) {
	pub := func(el string) map[string][]Published {
		p := published("Demo pre-op HQ", 1, el)
		return map[string][]Published{p.Question.Name: {p}}
	}
	cases := []struct {
		name, was, now, want string
	}{
		{"single choice to multiple choice",
			`{"type":"radiogroup","name":"q_k","title":"Smoke?","choices":["Yes","No"]}`,
			`{"type":"checkbox","name":"q_k","title":"Smoke?","choices":["Yes","No"]}`,
			"“Smoke?”: its kind of answer changed from single choice to multiple choice."},
		{"number to date",
			`{"type":"text","name":"q_k","title":"Weight","inputType":"number","unit":"kg"}`,
			`{"type":"text","name":"q_k","title":"Weight","inputType":"date"}`,
			"its kind of answer changed from number to date"},
		{"the unit",
			`{"type":"text","name":"q_k","title":"Weight","inputType":"number","unit":"kg"}`,
			`{"type":"text","name":"q_k","title":"Weight","inputType":"number","unit":"lb"}`,
			"its unit changed from “kg” to “lb”"},
		{"the unit dropped",
			`{"type":"text","name":"q_k","inputType":"number","unit":"kg"}`,
			`{"type":"text","name":"q_k","inputType":"number"}`,
			"its unit changed from “kg” to none"},
		{"a grid's row",
			`{"type":"matrix","name":"q_k","rows":["Run","Swim"],"columns":["Never","Always"]}`,
			`{"type":"matrix","name":"q_k","rows":["Run"],"columns":["Never","Always"]}`,
			"the row “Swim” was removed"},
		{"a grid's column",
			`{"type":"matrix","name":"q_k","rows":["Run"],"columns":["Never","Always"]}`,
			`{"type":"matrix","name":"q_k","rows":["Run"],"columns":["Never",{"value":"Always","text":"Often"}]}`,
			"the column “Always” was reworded to “Often”"},
		{"a rating's scale",
			`{"type":"rating","name":"q_k","rateMin":0,"rateMax":10}`,
			`{"type":"rating","name":"q_k","rateMin":1,"rateMax":5}`,
			"its scale changed from 0 to 10 by 1 to 1 to 5 by 1"},
		{"a rating's values",
			`{"type":"rating","name":"q_k","rateValues":["Low","High"]}`,
			`{"type":"rating","name":"q_k","rateValues":["Low","Medium","High"]}`,
			"the value “Medium” was added"},
		{"a Yes/No switch's label",
			`{"type":"boolean","name":"q_k"}`,
			`{"type":"boolean","name":"q_k","labelTrue":"Yes, often"}`,
			"the label “Yes” was reworded to “Yes, often”"},
		{"a repeating list's column removed",
			`{"type":"paneldynamic","name":"q_k","templateElements":[{"type":"text","name":"q_a","title":"Medicine"},{"type":"text","name":"q_b","title":"Dose"}]}`,
			`{"type":"paneldynamic","name":"q_k","templateElements":[{"type":"text","name":"q_a","title":"Medicine"}]}`,
			"the column “Dose” was removed"},
		{"a repeating list's column changes kind",
			`{"type":"matrixdynamic","name":"q_k","columns":[{"name":"year","title":"Year","cellType":"text"}]}`,
			`{"type":"matrixdynamic","name":"q_k","columns":[{"name":"year","title":"Year","cellType":"text","inputType":"number"}]}`,
			"the column “Year” changed its kind of answer"},
		{"a repeating list's column option",
			`{"type":"matrixdynamic","name":"q_k","columns":[{"name":"kind","title":"Kind","cellType":"dropdown","choices":["General","Local"]}]}`,
			`{"type":"matrixdynamic","name":"q_k","columns":[{"name":"kind","title":"Kind","cellType":"dropdown","choices":["General"]}]}`,
			"the column “Kind”: the option “Local” was removed"},
		{"a signature to a text",
			`{"type":"signaturepad","name":"q_k","title":"Sign"}`,
			`{"type":"text","name":"q_k","title":"Sign"}`,
			"its kind of answer changed from signature to text"},
	}
	for _, c := range cases {
		p := LockProblems(walkSets(t, set("About you", c.now)), pub(c.was))
		if len(p) != 1 || !strings.Contains(p[0].Message, c.want) || !strings.Contains(p[0].Message, "A published question's answers can't change") {
			t.Errorf("%s: %v, want one problem with %q", c.name, p, c.want)
		}
	}

	// Unchanged answers, reordered rows and columns, and a statement that was published: no problem.
	for _, c := range [][2]string{
		{`{"type":"matrix","name":"q_k","rows":["Run","Swim"],"columns":["Never","Always"]}`, `{"type":"matrix","name":"q_k","title":"New text","rows":["Swim","Run"],"columns":["Always","Never"]}`},
		{`{"type":"rating","name":"q_k"}`, `{"type":"rating","name":"q_k","rateMin":1,"rateMax":5,"rateStep":1}`},
		{`{"type":"html","name":"q_k"}`, `{"type":"text","name":"q_k"}`},
	} {
		if p := LockProblems(walkSets(t, set("About you", c[1])), pub(c[0])); len(p) != 0 {
			t.Errorf("%s → %s: %v", c[0], c[1], p)
		}
	}
}

func TestOneOffAnswersNeverPrefill(t *testing.T) {
	sign := `{"type":"signaturepad","name":"q_s","title":"Please sign below"},{"type":"file","name":"q_f","title":"Upload your letter"},{"type":"bmi","name":"q_bmi","title":"BMI Calculator"}`
	mine := walkSets(t, set("A", sign))
	got := Matches(mine, []Version{{QuestionnaireID: "hq-1", Questionnaire: "Other", Questions: walkSets(t, set("B", sign))}})
	if len(got) != 0 {
		t.Errorf("signatures, uploads and the BMI calculator take part in pre-fill: %v", got)
	}
}

func TestSameWordingGridsAndRatings(t *testing.T) {
	pain := q(t, `{"type":"rating","name":"q_a","title":"How bad is your pain?","rateMin":0,"rateMax":10}`)
	if SameWording(pain, q(t, `{"type":"rating","name":"q_b","title":"How bad is your pain?","rateMin":1,"rateMax":5}`)) {
		t.Error("ratings on other scales matched")
	}
	if !SameWording(pain, q(t, `{"type":"rating","name":"q_b","title":"How bad is your pain?","rateMin":0,"rateMax":10}`)) {
		t.Error("the same rating didn't match")
	}
	grid := q(t, `{"type":"matrix","name":"q_a","title":"How often do you","rows":["Run"],"columns":["Never"]}`)
	if SameWording(grid, q(t, `{"type":"matrix","name":"q_b","title":"How often do you","rows":["Swim"],"columns":["Never"]}`)) {
		t.Error("grids with other rows matched")
	}
	if SameWording(grid, q(t, `{"type":"matrix","name":"q_b","title":"How often do you","rows":["Run"],"columns":["Always"]}`)) {
		t.Error("grids with other columns matched")
	}
	if !SameWording(grid, q(t, `{"type":"matrix","name":"q_b","title":"How often do you","rows":["run"],"columns":["Never"]}`)) {
		t.Error("the same grid didn't match")
	}
}

// When the trust already published a key in shapes that disagree (data from before the lock), no
// draft can match them all: one problem says so, instead of "Yes was removed" and "Yes was added"
// (test report 3, failure 1). The lock stays strict (S23 answers, "after test report 3").
func TestLockWhenPublishedShapesDisagree(t *testing.T) {
	pub := map[string][]Published{"q_diabetes": {
		published("Demo pre-op HQ", 1, `{"type":"radiogroup","name":"q_diabetes","title":"Diabetes?","choices":[{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"}]}`),
		published("Old HQ A", 1, `{"type":"boolean","name":"q_diabetes"}`),
		published("Old HQ B", 1, `{"type":"radiogroup","name":"q_diabetes","choices":[{"value":"o_diay","text":"Yes"},{"value":"o_dian","text":"No"}]}`),
		published("Old HQ C", 2, `{"type":"radiogroup","name":"q_diabetes","choices":[{"value":"o_diay","text":"Yes"},{"value":"o_dian","text":"No"}]}`),
	}}
	draft := walkSets(t, set("Your health", `{"type":"radiogroup","name":"q_diabetes","title":"Do you have diabetes?","choices":[{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"}]}`))
	p := LockProblems(draft, pub)
	want := "“Do you have diabetes?”: the trust has already published this question with different answers " +
		"(in “Demo pre-op HQ” v1; “Old HQ A” v1; “Old HQ B” v1 and “Old HQ C” v2), so it can't be published again as it is. " +
		"Add a new question instead, or ask for the old questionnaires to be tidied up."
	if len(p) != 1 || p[0].Set != "Your health" || p[0].Message != want {
		t.Errorf("LockProblems =\n%v\nwant one problem:\n%s", p, want)
	}
	// Published copies that agree are one source, and a matching draft passes.
	agree := map[string][]Published{"q_diabetes": {pub["q_diabetes"][2], pub["q_diabetes"][3]}}
	same := walkSets(t, set("S", `{"type":"radiogroup","name":"q_diabetes","choices":[{"value":"o_dian","text":"No"},{"value":"o_diay","text":"Yes"}]}`))
	if p := LockProblems(same, agree); len(p) != 0 {
		t.Errorf("a draft matching agreeing copies: %v", p)
	}
}

// A date's format and a text question's input format are part of the answer (S23 answers, "after
// verification 2").
func TestWalkFormats(t *testing.T) {
	qs, err := Walk(json.RawMessage(`{"pages":[{"name":"p","elements":[
		{"type":"text","name":"q_dmy","inputType":"date"},
		{"type":"text","name":"q_my","inputType":"month"},
		{"type":"text","name":"q_y","inputType":"number","dateFormat":"y"},
		{"type":"text","name":"q_any"},
		{"type":"comment","name":"q_long"},
		{"type":"text","name":"q_email","inputType":"email","validators":[{"type":"email"}]},
		{"type":"text","name":"q_phone","inputType":"tel","validators":[{"type":"text","maxLength":20},{"type":"regex","regex":"^[0-9]+$"}]},
		{"type":"text","name":"q_url","inputType":"url"},
		{"type":"text","name":"q_num","inputType":"number"}
	]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	m := byName(t, qs)
	for name, want := range map[string]string{
		"q_dmy": "day, month and year", "q_my": "month and year", "q_y": "year only",
		"q_any": "any text", "q_long": "any text", "q_email": "email address", "q_phone": "the pattern “^[0-9]+$”",
		"q_url": "url", "q_num": "",
	} {
		if got := m[name].Format; got != want {
			t.Errorf("%s format = %q, want %q", name, got, want)
		}
	}
	if m["q_y"].Kind != "date" {
		t.Errorf("a year is kind %s, want date", m["q_y"].Kind)
	}
}

func TestLockFreezesFormats(t *testing.T) {
	cases := []struct{ was, now, want string }{
		{`{"type":"text","name":"q_k","title":"When?","inputType":"date"}`, `{"type":"text","name":"q_k","title":"When?","inputType":"number","dateFormat":"y"}`,
			"“When?”: its date format changed from day, month and year to year only"},
		{`{"type":"text","name":"q_k","inputType":"number","dateFormat":"y"}`, `{"type":"text","name":"q_k","inputType":"month"}`,
			"its date format changed from year only to month and year"},
		{`{"type":"text","name":"q_k","title":"Contact"}`, `{"type":"text","name":"q_k","title":"Contact","inputType":"email","validators":[{"type":"email"}]}`,
			"its answer format changed from any text to email address"},
		{`{"type":"text","name":"q_k","inputType":"tel","validators":[{"type":"regex","regex":"^[0-9]+$"}]}`, `{"type":"text","name":"q_k","inputType":"email","validators":[{"type":"email"}]}`,
			"its answer format changed from the pattern “^[0-9]+$” to email address"},
		{`{"type":"matrixdynamic","name":"q_k","dateList":true,"columns":[{"name":"date","cellType":"text","inputType":"date"}]}`,
			`{"type":"matrixdynamic","name":"q_k","dateList":true,"columns":[{"name":"date","cellType":"text","inputType":"number","dateFormat":"y"}]}`,
			"the column “date” changed its kind of answer"},
	}
	for _, c := range cases {
		p := LockProblems(walkSets(t, set("S", c.now)), map[string][]Published{"q_k": {published("HQ", 1, c.was)}})
		if len(p) != 1 || !strings.Contains(p[0].Message, c.want) {
			t.Errorf("%s → %s: %v, want %q", c.was, c.now, p, c.want)
		}
	}
	// One line or several lines of text is the same answer.
	if p := LockProblems(walkSets(t, set("S", `{"type":"comment","name":"q_k"}`)), map[string][]Published{"q_k": {published("HQ", 1, `{"type":"text","name":"q_k"}`)}}); len(p) != 0 {
		t.Errorf("text → comment: %v", p)
	}
}

func TestSameWordingFormats(t *testing.T) {
	dmy := q(t, `{"type":"text","name":"q_a","title":"Date of your operation","inputType":"date"}`)
	if SameWording(dmy, q(t, `{"type":"text","name":"q_b","title":"Date of your operation","inputType":"number","dateFormat":"y"}`)) {
		t.Error("a full date matched a year")
	}
	email := q(t, `{"type":"text","name":"q_a","title":"Contact","inputType":"email","validators":[{"type":"email"}]}`)
	if SameWording(email, q(t, `{"type":"text","name":"q_b","title":"Contact"}`)) {
		t.Error("an email address matched any text")
	}
}

// A repeating group's children are checked as questions of their own; the group reports only a
// child added or removed, so one change is one problem (verification 2, item 2).
func TestLockRepeatingGroupOnce(t *testing.T) {
	was := `{"type":"paneldynamic","name":"q_list","templateElements":[{"type":"text","name":"q_med","title":"Medicine"},{"type":"radiogroup","name":"q_when","title":"When","choices":["Morning","Evening"]}]}`
	pub := map[string][]Published{}
	for _, q := range walkSets(t, set("S", was)) {
		pub[q.Name] = []Published{{Questionnaire: "HQ", VersionNo: 1, Question: q.Question}}
	}
	reworded := `{"type":"paneldynamic","name":"q_list","templateElements":[{"type":"text","name":"q_med","title":"Medicine name"},{"type":"radiogroup","name":"q_when","title":"When","choices":["Morning",{"value":"Evening","text":"Night"}]}]}`
	p := LockProblems(walkSets(t, set("S", reworded)), pub)
	if len(p) != 1 || !strings.Contains(p[0].Message, "“When”: the option “Evening” was reworded to “Night”") {
		t.Errorf("a child's option reworded (and a child's text, which may change): %v, want one problem", p)
	}
	removed := `{"type":"paneldynamic","name":"q_list","templateElements":[{"type":"text","name":"q_med","title":"Medicine"}]}`
	if p := LockProblems(walkSets(t, set("S", removed)), pub); len(p) != 1 || !strings.Contains(p[0].Message, "the column “When” was removed") {
		t.Errorf("a child removed: %v", p)
	}
}

func TestDuplicatesSkipUnnamed(t *testing.T) {
	if p := Duplicates(walkSets(t, set("S", `{"type":"text"},{"type":"text","name":""},{"type":"html"}`))); len(p) != 0 {
		t.Errorf("unnamed elements: %v", p)
	}
}

// A medication question's type (prescribed, non-prescribed, recreational) is part of its answer
// (S23 verification 3, carried into S24): a published medication key can't switch type, and two
// medication blocks of different types never match by wording. A block without a type is
// prescribed (the editor's default).
func TestMedicationTypeIsPartOfTheAnswer(t *testing.T) {
	qs, err := Walk(json.RawMessage(`{"pages":[{"name":"p","elements":[
		{"type":"medication","name":"q_rx","medicationType":"prescribed"},
		{"type":"medication","name":"q_default"},
		{"type":"medication","name":"q_otc","medicationType":"non_prescribed"},
		{"type":"medication","name":"q_rec","medicationType":"recreational"}
	]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	m := byName(t, qs)
	for name, want := range map[string]string{
		"q_rx": "prescribed medication", "q_default": "prescribed medication",
		"q_otc": "non-prescribed medication", "q_rec": "recreational drugs",
	} {
		if got := m[name].Format; got != want {
			t.Errorf("%s format = %q, want %q", name, got, want)
		}
	}

	was := `{"type":"medication","name":"q_k","title":"Your medicines","medicationType":"prescribed"}`
	now := `{"type":"medication","name":"q_k","title":"Your medicines","medicationType":"recreational"}`
	p := LockProblems(walkSets(t, set("S", now)), map[string][]Published{"q_k": {published("HQ", 1, was)}})
	if len(p) != 1 || !strings.Contains(p[0].Message, "“Your medicines”: its medication type changed from prescribed medication to recreational drugs") {
		t.Errorf("prescribed → recreational: %v", p)
	}
	if p := LockProblems(walkSets(t, set("S", `{"type":"medication","name":"q_k","title":"Your medicines"}`)), map[string][]Published{"q_k": {published("HQ", 1, was)}}); len(p) != 0 {
		t.Errorf("prescribed → no type (prescribed): %v", p)
	}

	same := `{"type":"medication","name":"q_b","title":"Your medicines","medicationType":"prescribed"}`
	if !SameWording(q(t, was), q(t, same)) {
		t.Error("two prescribed blocks with the same text don't match")
	}
	if SameWording(q(t, was), q(t, `{"type":"medication","name":"q_b","title":"Your medicines","medicationType":"recreational"}`)) {
		t.Error("a prescribed block matched a recreational one by wording")
	}
}

// "Same as…" (S24 answers 3 and 4): a question may take another's key when it holds the same kind
// of answer. Choices are compatible whatever their options (adopting copies them); other kinds need
// the same unit, format, scale, rows and labels already, since nothing else is copied and the
// publish lock would refuse any difference. Repeating lists are never linked this way.
func TestCompatible(t *testing.T) {
	cases := []struct {
		name, mine, theirs string
		ok                 bool
		reason             string
	}{
		{"single with single (options differ)", `{"type":"radiogroup","name":"a","choices":["Yes","No"]}`,
			`{"type":"radiogroup","name":"b","choices":[{"value":"never","text":"Never"},{"value":"current","text":"I smoke now"}]}`, true, ""},
		{"a dropdown with yes/no buttons", `{"type":"dropdown","name":"a","choices":["A"]}`,
			`{"type":"radiogroup","name":"b","yesNo":true,"choices":[{"value":"yes","text":"Yes"},{"value":"no","text":"No"}]}`, true, ""},
		{"multiple with multiple", `{"type":"checkbox","name":"a","choices":["A"]}`, `{"type":"tagbox","name":"b","choices":["B"]}`, true, ""},
		{"single with multiple", `{"type":"radiogroup","name":"a","choices":["A"]}`, `{"type":"checkbox","name":"b","choices":["A"]}`,
			false, "Its answer is multiple choice; this question's is single choice."},
		{"text for a choice question", `{"type":"radiogroup","name":"a","choices":["A"]}`, `{"type":"text","name":"b"}`,
			false, "Its answer is text; this question's is single choice."},
		{"number with the same unit", `{"type":"text","name":"a","inputType":"number","unit":"kg"}`, `{"type":"text","name":"b","inputType":"number","unit":"kg"}`, true, ""},
		{"number with another unit", `{"type":"text","name":"a","inputType":"number","unit":"lb"}`, `{"type":"text","name":"b","inputType":"number","unit":"kg"}`,
			false, "Its unit is “kg”; this question's is “lb”."},
		{"number without a unit", `{"type":"text","name":"a","inputType":"number"}`, `{"type":"text","name":"b","inputType":"number","unit":"kg"}`,
			false, "Its unit is “kg”; this question's is none."},
		{"date with date", `{"type":"text","name":"a","inputType":"date"}`, `{"type":"text","name":"b","inputType":"date"}`, true, ""},
		{"a full date with a year", `{"type":"text","name":"a","inputType":"date"}`, `{"type":"text","name":"b","inputType":"number","dateFormat":"y"}`,
			false, "Its date format is year only; this question's is day, month and year."},
		{"text with a long answer", `{"type":"text","name":"a"}`, `{"type":"comment","name":"b"}`, true, ""},
		{"text with an email address", `{"type":"text","name":"a"}`, `{"type":"text","name":"b","inputType":"email","validators":[{"type":"email"}]}`,
			false, "Its answer format is email address; this question's is any text."},
		{"a Yes/No switch with another's labels", `{"type":"boolean","name":"a"}`, `{"type":"boolean","name":"b","labelTrue":"Yes, I do"}`,
			false, "Its answers differ from this question's: the label “Yes, I do” was reworded to “Yes”."},
		{"prescribed with recreational", `{"type":"medication","name":"a"}`, `{"type":"medication","name":"b","medicationType":"recreational"}`,
			false, "Its medication type is recreational drugs; this question's is prescribed medication."},
		{"prescribed with prescribed", `{"type":"medication","name":"a","medicationType":"prescribed"}`, `{"type":"medication","name":"b"}`, true, ""},
		{"a repeating group", `{"type":"text","name":"a"}`, `{"type":"paneldynamic","name":"b","templateElements":[{"type":"text","name":"c"}]}`,
			false, "Repeating lists can't be linked with “Same as…”: insert them from the library instead."},
		{"a list of dates", `{"type":"matrixdynamic","name":"a","dateList":true}`, `{"type":"matrixdynamic","name":"b","dateList":true}`,
			false, "Repeating lists can't be linked with “Same as…”: insert them from the library instead."},
		{"a signature", `{"type":"signaturepad","name":"a"}`, `{"type":"signaturepad","name":"b"}`,
			false, "Signatures, file uploads and the BMI block are never pre-filled."},
		{"a statement", `{"type":"text","name":"a"}`, `{"type":"html","name":"b"}`, false, "It holds no answer."},
	}
	for _, c := range cases {
		ok, reason := Compatible(q(t, c.mine), q(t, c.theirs))
		if ok != c.ok || reason != c.reason {
			t.Errorf("%s: (%v, %q), want (%v, %q)", c.name, ok, reason, c.ok, c.reason)
		}
	}
}

// Candidates are the keys "Same as…" offers a question (S24): the trust's published questions and
// the library's, one per key, matching the search, each with whether it may be adopted and why not.
func TestCandidates(t *testing.T) {
	mine := q(t, `{"type":"radiogroup","name":"q_mine","title":"Do you currently smoke?","choices":[{"value":"o_a","text":"Yes"},{"value":"o_b","text":"No"}]}`)
	here := walkSets(t, set("About you", `{"type":"radiogroup","name":"q_mine","title":"Do you currently smoke?","choices":["Yes","No"]},
		{"type":"text","name":"q_taken","title":"Smoking: how many a day?"}`))
	smoker := `{"type":"radiogroup","name":"smoker","title":"Do you smoke?","choices":[{"value":"never","text":"Never"},{"value":"ex","text":"Ex-smoker"},{"value":"current","text":"Current"}]}`
	sources := []Source{
		{Name: "Demo pre-op HQ", Questions: walkSets(t, set("S", smoker+`,
			{"type":"text","name":"q_smoke_note","title":"Smoking history"},
			{"type":"text","name":"q_taken","title":"Smoking: how many a day?"},
			{"type":"text","name":"q_mine","title":"Do you currently smoke?"},
			{"type":"text","name":"q_nurse","title":"Smoking advice given","clinicianOnly":true},
			{"type":"paneldynamic","name":"q_smoke_list","title":"Smoking products","templateElements":[{"type":"text","name":"q_product","title":"Smoking product"}]},
			{"type":"text","name":"q_alcohol","title":"Alcohol units"}`))},
		{Name: "Library", Library: true, Questions: walkSets(t, set("Library", smoker))},
	}
	got := Candidates(mine, here, sources, "smok")
	keysOf := make([]string, len(got))
	for i, c := range got {
		keysOf[i] = c.Key
	}
	// Compatible first, then by title; never the question's own key, a clinician-only question, a
	// repeating group's row question, or one the search doesn't find.
	if want := []string{"smoker", "q_smoke_note", "q_taken", "q_smoke_list"}; !reflect.DeepEqual(keysOf, want) {
		t.Fatalf("candidates %v, want %v", keysOf, want)
	}
	s := got[0]
	if !s.Compatible || s.Reason != "" || s.InThisHQ || s.Title != "Do you smoke?" || s.Kind != "single choice" ||
		!reflect.DeepEqual(s.Sources, []string{"Demo pre-op HQ", "Library"}) || !s.Library ||
		!reflect.DeepEqual(s.Options, []string{"Never", "Ex-smoker", "Current"}) || !strings.Contains(string(s.Element), `"ex"`) {
		t.Errorf("smoker = %+v", s)
	}
	if c := got[2]; c.Compatible || !c.InThisHQ || c.Reason != "This HQ already has a question with this ID, in “About you”." {
		t.Errorf("q_taken = %+v, want refused: already in this HQ", c)
	}
	if c := got[1]; c.Compatible || c.Reason != "Its answer is text; this question's is single choice." {
		t.Errorf("q_smoke_note = %+v", c)
	}
	if c := got[3]; c.Compatible || !strings.Contains(c.Reason, "Repeating lists") {
		t.Errorf("q_smoke_list = %+v", c)
	}
	// The search also finds a key; an empty search finds everything.
	if c := Candidates(mine, here, sources, "ALCOHOL"); len(c) != 1 || c[0].Key != "q_alcohol" {
		t.Errorf("search by text: %+v", c)
	}
	if c := Candidates(mine, here, sources, "smoke_n"); len(c) != 1 || c[0].Key != "q_smoke_note" {
		t.Errorf("search by key: %+v", c)
	}
	if c := Candidates(mine, here, sources, ""); len(c) != 5 {
		t.Errorf("an empty search: %d candidates, want 5", len(c))
	}
	// A key published with answers that disagree can't be adopted.
	odd := append(sources, Source{Name: "Old HQ", Questions: walkSets(t, set("S", `{"type":"radiogroup","name":"smoker","title":"Do you smoke?","choices":["Yes","No"]}`))})
	if c := Candidates(mine, here, odd, "do you smoke"); len(c) != 1 || c[0].Compatible ||
		c[0].Reason != "The trust has already published this question with different answers, so it can't be linked." {
		t.Errorf("disagreeing shapes: %+v", c)
	}
}

// The search keeps a key if any of its questions' texts (or the key) matches, so a published copy
// worded differently can't be skipped and leave the library's copy looking adoptable (S24
// verification 1, finding 3). A key with a fresh question says so (finding 2).
func TestCandidatesSearchSeesEveryCopy(t *testing.T) {
	mine := q(t, `{"type":"radiogroup","name":"q_mine","title":"Smoke?","choices":["Yes","No"]}`)
	sources := []Source{
		{Name: "Old HQ", Questions: walkSets(t, set("S", `{"type":"radiogroup","name":"smoker","title":"Smoking status","choices":["Yes","No"]}`))},
		{Name: "Library", Library: true, Questions: walkSets(t, set("Library", `{"type":"radiogroup","name":"smoker","title":"Do you smoke?","alwaysFresh":true,"choices":[{"value":"never","text":"Never"}]}`))},
	}
	c := Candidates(mine, nil, sources, "do you smoke")
	if len(c) != 1 || c[0].Compatible || !strings.Contains(c[0].Reason, "different answers") || !reflect.DeepEqual(c[0].Sources, []string{"Old HQ", "Library"}) {
		t.Errorf("candidates = %+v, want smoker refused: published with different answers, from both", c)
	}
	if !c[0].AlwaysFresh {
		t.Error("a key with an Always ask fresh question isn't marked fresh")
	}
}

// The editor's known text formats are named in words, not by their pattern (S27; Rahul, 24 Oct):
// the publish refusal says "phone number", "NHS number" or "UK postcode". A custom pattern is still
// shown as itself.
func TestKnownTextFormatsAreNamed(t *testing.T) {
	text := func(regex string) string {
		return `{"type":"text","name":"q_k","title":"Contact","validators":[{"type":"regex","regex":` + strconvQuote(regex) + `}]}`
	}
	for regex, want := range map[string]string{
		`^\+?[0-9 ()-]{7,20}$`:                        "phone number",
		`^\d{3} ?\d{3} ?\d{4}$`:                       "NHS number",
		`^[A-Za-z]{1,2}\d[A-Za-z\d]? ?\d[A-Za-z]{2}$`: "UK postcode",
		`^[0-9]+$`: "the pattern “^[0-9]+$”",
	} {
		if got := byName(t, mustWalk(t, `{"pages":[{"name":"p","elements":[`+text(regex)+`]}]}`))["q_k"].Format; got != want {
			t.Errorf("%s: format %q, want %q", regex, got, want)
		}
	}
	p := LockProblems(walkSets(t, set("S", text(`^\+?[0-9 ()-]{7,20}$`))), map[string][]Published{"q_k": {published("HQ", 1, `{"type":"text","name":"q_k","title":"Contact"}`)}})
	if len(p) != 1 || !strings.Contains(p[0].Message, "its answer format changed from any text to phone number") {
		t.Errorf("refusal %v, want it to say phone number", p)
	}
}

// The patterns named above are the editor's (packages/clinical/src/fields.ts TEXT_FORMATS), so a
// change to one there can't leave Go naming a pattern the editor no longer writes.
func TestKnownTextFormatsMatchTheEditor(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "packages", "clinical", "src", "fields.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for regex, name := range textFormatNames {
		if !strings.Contains(string(src), "regex: "+strconvQuote(regex)) {
			t.Errorf("fields.ts has no format with the regex %s (%s)", regex, name)
		}
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func mustWalk(t *testing.T, content string) []Question {
	t.Helper()
	qs, err := Walk(json.RawMessage(content))
	if err != nil {
		t.Fatal(err)
	}
	return qs
}

// "Same as…" warns before replacing a question's existing link (S29; Rahul, 25 Oct): LinkedTo is
// the text of the question its key already links to, from the library if it's there, else the
// trust's published questionnaires; "" for a key nothing published or in the library has.
func TestLinkedTo(t *testing.T) {
	published := Source{Name: "Demo pre-op HQ", Questions: walkSets(t, set("S", `{"type":"radiogroup","name":"smoker","title":"Smoking status","choices":["Yes","No"]},
		{"type":"text","name":"q_pub","title":"Published only"},
		{"type":"text","name":"q_nurse","title":"Nurse only","clinicianOnly":true},
		{"type":"paneldynamic","name":"q_list","title":"Products","templateElements":[{"type":"text","name":"q_row","title":"Product"}]}`))}
	library := Source{Name: "Library", Library: true, Questions: walkSets(t, set("Library", `{"type":"radiogroup","name":"smoker","title":"Do you smoke?","choices":["Yes","No"]}`))}
	sources := []Source{published, library}
	for name, want := range map[string]string{
		"smoker":  "Do you smoke?",  // the library's wording wins, as for the candidate
		"q_pub":   "Published only", // the trust's published question
		"q_list":  "Products",
		"q_new":   "", // a new question: no link yet
		"q_nurse": "", // clinician-only: never pre-filled, so no link
		"q_row":   "", // a repeating group's row travels with its group
	} {
		if got := LinkedTo(name, sources); got != want {
			t.Errorf("LinkedTo(%q) = %q, want %q", name, got, want)
		}
	}
}
