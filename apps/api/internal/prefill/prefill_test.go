package prefill

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// A Question Set of one page holding these elements (each a JSON object).
func set(id, audience string, elements ...string) Chapter {
	return Chapter{ID: id, Audience: audience, Content: json.RawMessage(`{"pages":[{"name":"p1","elements":[` + strings.Join(elements, ",") + `]}]}`)}
}

const (
	smoker  = `{"type":"radiogroup","name":"smoker","title":"Do you smoke?","choices":[{"value":"never","text":"No, never"},{"value":"ex","text":"I used to"},{"value":"current","text":"Yes"}]}`
	height  = `{"type":"text","name":"height","title":"How tall are you?","inputType":"number","unit":"cm"}`
	weight  = `{"type":"text","name":"weight","title":"What do you weigh?","inputType":"number","unit":"kg","alwaysFresh":true}`
	snoring = `{"type":"radiogroup","name":"snoring","title":"Do you snore loudly?","choices":[{"value":"yes","text":"Yes"},{"value":"no","text":"No"}]}`
)

var (
	march = time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	june  = time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
)

// source is an earlier form on these sets, with the patient's answers per set (as JSON objects).
func source(id string, at time.Time, sets []Chapter, patient map[string]string) Source {
	s := Source{FormID: id, SubmittedAt: at, Chapters: sets, Patient: map[string]json.RawMessage{}, Clinician: map[string]json.RawMessage{}}
	for k, v := range patient {
		s.Patient[k] = json.RawMessage(v)
	}
	return s
}

// fill pre-fills every set of target.
func fill(t *testing.T, target []Chapter, sources ...Source) map[string]Filled {
	t.Helper()
	want := map[string]bool{}
	for _, c := range target {
		want[c.ID] = true
	}
	got, err := Fill(target, want, sources)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// data is a filled set's answers as one JSON object, keys sorted ("{}" when nothing was filled).
func data(f map[string]Filled, set string) string {
	b, _ := json.Marshal(f[set].Data)
	if f[set].Data == nil {
		return "{}"
	}
	return string(b)
}

func TestFillByKey(t *testing.T) {
	old := source("form-1", march, []Chapter{set("c1", "patient", smoker, height)}, map[string]string{"c1": `{"smoker":"ex","height":172}`})
	got := fill(t, []Chapter{set("n1", "patient", height, smoker)}, old)
	if d := data(got, "n1"); d != `{"height":172,"smoker":"ex"}` {
		t.Errorf("data %s", d)
	}
	e := got["n1"].Prefill["smoker"]
	if e.FromForm != "form-1" || !e.AnsweredAt.Equal(march) || string(e.Value) != `"ex"` || e.MatchedBy != "key" {
		t.Errorf("prefill %+v", e)
	}
	if len(got["n1"].Prefill) != 2 {
		t.Errorf("prefill %v", got["n1"].Prefill)
	}
}

func TestFillPrecedence(t *testing.T) {
	sets := []Chapter{set("c1", "patient", smoker, snoring)}
	target := []Chapter{set("n1", "patient", smoker, snoring)}

	t.Run("the clinician's validated copy over the patient's", func(t *testing.T) {
		s := source("f1", march, sets, map[string]string{"c1": `{"smoker":"current","snoring":"no"}`})
		s.Clinician["c1"] = json.RawMessage(`{"smoker":"ex","snoring":"no"}`)
		if d := data(fill(t, target, s), "n1"); d != `{"smoker":"ex","snoring":"no"}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("an answer the clinician cleared isn't filled from the patient's", func(t *testing.T) {
		s := source("f1", march, sets, map[string]string{"c1": `{"smoker":"current","snoring":"no"}`})
		s.Clinician["c1"] = json.RawMessage(`{"snoring":"no"}`)
		if d := data(fill(t, target, s), "n1"); d != `{"snoring":"no"}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("the newest form first, whatever order they come in", func(t *testing.T) {
		older := source("f-march", march, sets, map[string]string{"c1": `{"smoker":"current"}`})
		newer := source("f-june", june, sets, map[string]string{"c1": `{"smoker":"ex","snoring":"yes"}`})
		got := fill(t, target, older, newer)
		if d := data(got, "n1"); d != `{"smoker":"ex","snoring":"yes"}` {
			t.Errorf("data %s", d)
		}
		if got["n1"].Prefill["smoker"].FromForm != "f-june" {
			t.Errorf("from %+v", got["n1"].Prefill["smoker"])
		}
	})
	t.Run("the newest form that asked the question decides, even unanswered", func(t *testing.T) {
		older := source("f-march", march, sets, map[string]string{"c1": `{"smoker":"current","snoring":"yes"}`})
		newer := source("f-june", june, sets, map[string]string{"c1": `{"snoring":"no"}`}) // smoker skipped (or hidden)
		if d := data(fill(t, target, older, newer), "n1"); d != `{"snoring":"no"}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("an older form when the newer one didn't ask it", func(t *testing.T) {
		older := source("f-march", march, sets, map[string]string{"c1": `{"smoker":"current"}`})
		newer := source("f-june", june, []Chapter{set("c9", "patient", snoring)}, map[string]string{"c9": `{"snoring":"no"}`})
		got := fill(t, target, older, newer)
		if d := data(got, "n1"); d != `{"smoker":"current","snoring":"no"}` {
			t.Errorf("data %s", d)
		}
		if got["n1"].Prefill["smoker"].FromForm != "f-march" || got["n1"].Prefill["snoring"].FromForm != "f-june" {
			t.Errorf("prefill %+v", got["n1"].Prefill)
		}
	})
}

func TestNeverFilled(t *testing.T) {
	old := source("f1", march, []Chapter{set("c1", "patient", smoker, height, weight, snoring,
		`{"type":"text","name":"notes","title":"Anything else?"}`,
		`{"type":"signaturepad","name":"sig","title":"Sign here"}`,
		`{"type":"file","name":"letter","title":"Your letter"}`,
		`{"type":"bmi","name":"bmi","title":"Your BMI"}`)},
		map[string]string{"c1": `{"smoker":"ex","height":172,"weight":80,"snoring":"no","notes":"x","sig":"data:x","letter":[{"name":"a.pdf"}],"bmi":{"w":80}}`})

	target := []Chapter{
		set("n1", "patient",
			`{"type":"radiogroup","name":"smoker","title":"Do you smoke?","clinicianOnly":true,"choices":["never","ex","current"]}`,
			weight,
			`{"type":"profile","name":"profile"}`,
			`{"type":"expression","name":"notes","title":"Anything else?","expression":"1"}`,
			`{"type":"signaturepad","name":"sig","title":"Sign here"}`,
			`{"type":"file","name":"letter","title":"Your letter"}`,
			`{"type":"bmi","name":"bmi","title":"Your BMI"}`,
			`{"type":"panel","name":"g","clinicianOnly":true,"elements":[`+height+`]}`),
		set("n2", "clinician", snoring),
	}
	got := fill(t, target, old)
	if len(got) != 0 {
		t.Errorf("filled %+v", got)
	}

	t.Run("a clinician-only page", func(t *testing.T) {
		page := Chapter{ID: "n3", Audience: "patient", Content: json.RawMessage(`{"pages":[{"name":"p","clinicianOnly":true,"elements":[` + snoring + `]}]}`)}
		if got := fill(t, []Chapter{page}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("only the sets asked for", func(t *testing.T) {
		got, err := Fill([]Chapter{set("n1", "patient", smoker), set("n2", "patient", snoring)}, map[string]bool{"n2": true}, []Source{old})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := got["n1"]; ok || data(got, "n2") != `{"snoring":"no"}` {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("a clinician-only question in the earlier form isn't a source", func(t *testing.T) {
		clin := source("f2", june, []Chapter{set("c1", "patient", `{"type":"radiogroup","name":"snoring","title":"Do you snore loudly?","clinicianOnly":true,"choices":[{"value":"yes","text":"Yes"},{"value":"no","text":"No"}]}`), set("c2", "clinician", height)},
			map[string]string{})
		clin.Clinician["c1"] = json.RawMessage(`{"snoring":"yes"}`)
		clin.Clinician["c2"] = json.RawMessage(`{"height":150}`)
		if got := fill(t, []Chapter{set("n1", "patient", snoring, height)}, clin); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
}

func TestChoiceValues(t *testing.T) {
	t.Run("a removed option value isn't filled", func(t *testing.T) {
		old := source("f1", march, []Chapter{set("c1", "patient", smoker)}, map[string]string{"c1": `{"smoker":"ex"}`})
		now := `{"type":"radiogroup","name":"smoker","title":"Do you smoke?","choices":[{"value":"never","text":"No, never"},{"value":"current","text":"Yes"}]}`
		if got := fill(t, []Chapter{set("n1", "patient", now)}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	symptoms := func(choices string) string {
		return `{"type":"checkbox","name":"symptoms","title":"Which of these do you have?","showOtherItem":true,"choices":` + choices + `}`
	}
	t.Run("several values, with Other's text", func(t *testing.T) {
		old := source("f1", march, []Chapter{set("c1", "patient", symptoms(`[{"value":"a","text":"Cough"},{"value":"b","text":"Fever"}]`))},
			map[string]string{"c1": `{"symptoms":["a","other"],"symptoms-Comment":"Headache"}`})
		got := fill(t, []Chapter{set("n1", "patient", symptoms(`[{"value":"b","text":"Fever"},{"value":"a","text":"Cough"}]`))}, old)
		if d := data(got, "n1"); d != `{"symptoms":["a","other"],"symptoms-Comment":"Headache"}` {
			t.Errorf("data %s", d)
		}
		if e := got["n1"].Prefill["symptoms"]; string(e.Comment) != `"Headache"` {
			t.Errorf("prefill %+v", e)
		}
	})
	t.Run("one value that doesn't fit stops the whole answer", func(t *testing.T) {
		old := source("f1", march, []Chapter{set("c1", "patient", symptoms(`[{"value":"a","text":"Cough"},{"value":"b","text":"Fever"}]`))},
			map[string]string{"c1": `{"symptoms":["a","b"]}`})
		if got := fill(t, []Chapter{set("n1", "patient", symptoms(`[{"value":"a","text":"Cough"}]`))}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("Other's text isn't copied where the new question has no Other", func(t *testing.T) {
		old := source("f1", march, []Chapter{set("c1", "patient", smoker)}, map[string]string{"c1": `{"smoker":"ex","smoker-Comment":"stale"}`})
		if d := data(fill(t, []Chapter{set("n1", "patient", smoker)}, old), "n1"); d != `{"smoker":"ex"}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("a comment box's text travels with its question", func(t *testing.T) {
		withComment := `{"type":"radiogroup","name":"snoring","title":"Do you snore loudly?","showCommentArea":true,"choices":[{"value":"yes","text":"Yes"},{"value":"no","text":"No"}]}`
		old := source("f1", march, []Chapter{set("c1", "patient", withComment)}, map[string]string{"c1": `{"snoring":"yes","snoring-Comment":"Only with a cold"}`})
		if d := data(fill(t, []Chapter{set("n1", "patient", withComment)}, old), "n1"); d != `{"snoring":"yes","snoring-Comment":"Only with a cold"}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("a Yes/No switch", func(t *testing.T) {
		sw := `{"type":"boolean","name":"q_smokes","title":"Do you smoke?"}`
		old := source("f1", march, []Chapter{set("c1", "patient", sw)}, map[string]string{"c1": `{"q_smokes":false}`})
		if d := data(fill(t, []Chapter{set("n1", "patient", sw)}, old), "n1"); d != `{"q_smokes":false}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("a grid's rows and columns", func(t *testing.T) {
		grid := `{"type":"matrix","name":"g","title":"How often?","rows":[{"value":"r1","text":"Run"}],"columns":[{"value":"c1","text":"Daily"},{"value":"c2","text":"Never"}]}`
		old := source("f1", march, []Chapter{set("c1", "patient", grid)}, map[string]string{"c1": `{"g":{"r1":"c2"}}`})
		if d := data(fill(t, []Chapter{set("n1", "patient", grid)}, old), "n1"); d != `{"g":{"r1":"c2"}}` {
			t.Errorf("data %s", d)
		}
		bad := source("f1", march, []Chapter{set("c1", "patient", grid)}, map[string]string{"c1": `{"g":{"r1":"c9"}}`})
		if got := fill(t, []Chapter{set("n1", "patient", grid)}, bad); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
}

func TestFillByWording(t *testing.T) {
	diabetes := `{"type":"radiogroup","name":"q_diabetes","title":"Do you have diabetes?","choices":[{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"}]}`
	old := source("f1", march, []Chapter{set("c1", "patient", diabetes, height)}, map[string]string{"c1": `{"q_diabetes":"o_no","height":172}`})

	t.Run("the same text maps the answer to the new question's own value", func(t *testing.T) {
		mine := `{"type":"radiogroup","name":"fu_diabetes","title":"  do you have DIABETES ","choices":[{"value":"n","text":"No"},{"value":"y","text":"Yes"}]}`
		got := fill(t, []Chapter{set("n1", "patient", mine)}, old)
		if d := data(got, "n1"); d != `{"fu_diabetes":"n"}` {
			t.Errorf("data %s", d)
		}
		if e := got["n1"].Prefill["fu_diabetes"]; e.MatchedBy != "wording" || string(e.Value) != `"n"` {
			t.Errorf("prefill %+v", e)
		}
	})
	t.Run("a number option value stays a number", func(t *testing.T) {
		mine := `{"type":"radiogroup","name":"fu_diabetes","title":"Do you have diabetes?","choices":[{"value":1,"text":"Yes"},{"value":2,"text":"No"}]}`
		if d := data(fill(t, []Chapter{set("n1", "patient", mine)}, old), "n1"); d != `{"fu_diabetes":2}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("a choice whose option text is missing", func(t *testing.T) {
		mine := `{"type":"radiogroup","name":"fu_diabetes","title":"Do you have diabetes?","choices":[{"value":"y","text":"Yes"},{"value":"n","text":"Not that I know"}]}`
		if got := fill(t, []Chapter{set("n1", "patient", mine)}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("a different kind of answer", func(t *testing.T) {
		mine := `{"type":"checkbox","name":"fu_diabetes","title":"Do you have diabetes?","choices":[{"value":"y","text":"Yes"},{"value":"n","text":"No"}]}`
		tall := `{"type":"text","name":"fu_height","title":"How tall are you?","inputType":"number","unit":"in"}`
		if got := fill(t, []Chapter{set("n1", "patient", mine, tall)}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("a number of the same unit", func(t *testing.T) {
		tall := `{"type":"text","name":"fu_height","title":"How tall are you?","inputType":"number","unit":"cm"}`
		if d := data(fill(t, []Chapter{set("n1", "patient", tall)}, old), "n1"); d != `{"fu_height":172}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("a text used twice in the new questionnaire never matches by wording", func(t *testing.T) {
		a := `{"type":"text","name":"fu_height","title":"How tall are you?","inputType":"number","unit":"cm"}`
		b := `{"type":"text","name":"fu_height2","title":"How tall are you?","inputType":"number","unit":"cm"}`
		if got := fill(t, []Chapter{set("n1", "patient", a), set("n2", "clinician", b)}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("a text used twice in the earlier questionnaire never matches by wording", func(t *testing.T) {
		twice := source("f1", march, []Chapter{set("c1", "patient", height, `{"type":"text","name":"h2","title":"How tall are you?","inputType":"number","unit":"cm"}`)},
			map[string]string{"c1": `{"height":172,"h2":180}`})
		a := `{"type":"text","name":"fu_height","title":"How tall are you?","inputType":"number","unit":"cm"}`
		if got := fill(t, []Chapter{set("n1", "patient", a)}, twice); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("the editor's placeholder text never matches", func(t *testing.T) {
		q := `{"type":"text","name":"%s","title":"Text question"}`
		src := source("f1", march, []Chapter{set("c1", "patient", strings.Replace(q, "%s", "a", 1))}, map[string]string{"c1": `{"a":"hello"}`})
		if got := fill(t, []Chapter{set("n1", "patient", strings.Replace(q, "%s", "b", 1))}, src); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
}

func TestRepeatingPanels(t *testing.T) {
	allergies := `{"type":"paneldynamic","name":"allergies","title":"Your allergies","templateElements":[` +
		`{"type":"text","name":"allergies_what","title":"What are you allergic to?"},` +
		`{"type":"text","name":"allergies_reaction","title":"What happens?"},` +
		`{"type":"radiogroup","name":"allergies_severe","title":"Is it severe?","choices":[{"value":"y","text":"Yes"},{"value":"n","text":"No"}]}]}`
	old := source("f1", march, []Chapter{set("c1", "patient", allergies)},
		map[string]string{"c1": `{"allergies":[{"allergies_what":"Penicillin","allergies_reaction":"Rash","allergies_severe":"n"},{"allergies_what":"Nuts"}]}`})

	t.Run("rows copied whole, by key", func(t *testing.T) {
		got := fill(t, []Chapter{set("n1", "patient", allergies)}, old)
		want := `{"allergies":[{"allergies_reaction":"Rash","allergies_severe":"n","allergies_what":"Penicillin"},{"allergies_what":"Nuts"}]}`
		if d := data(got, "n1"); d != want {
			t.Errorf("data %s", d)
		}
	})
	t.Run("keeping only the columns the new panel has", func(t *testing.T) {
		fewer := `{"type":"paneldynamic","name":"allergies","title":"Your allergies","templateElements":[{"type":"text","name":"allergies_what","title":"What are you allergic to?"}]}`
		if d := data(fill(t, []Chapter{set("n1", "patient", fewer)}, old), "n1"); d != `{"allergies":[{"allergies_what":"Penicillin"},{"allergies_what":"Nuts"}]}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("by wording, each column by its text and each option by its text", func(t *testing.T) {
		mine := `{"type":"paneldynamic","name":"fu_allergies","title":"Your allergies","templateElements":[` +
			`{"type":"text","name":"fa_what","title":"What are you allergic to?"},` +
			`{"type":"text","name":"fa_reaction","title":"What happens?"},` +
			`{"type":"radiogroup","name":"fa_severe","title":"Is it severe?","choices":[{"value":"no","text":"No"},{"value":"yes","text":"Yes"}]}]}`
		got := fill(t, []Chapter{set("n1", "patient", mine)}, old)
		want := `{"fu_allergies":[{"fa_reaction":"Rash","fa_severe":"no","fa_what":"Penicillin"},{"fa_what":"Nuts"}]}`
		if d := data(got, "n1"); d != want || got["n1"].Prefill["fu_allergies"].MatchedBy != "wording" {
			t.Errorf("data %s %+v", d, got["n1"].Prefill)
		}
	})
	t.Run("by wording, column texts that differ never match", func(t *testing.T) {
		mine := `{"type":"paneldynamic","name":"fu_allergies","title":"Your allergies","templateElements":[` +
			`{"type":"text","name":"fa_what","title":"What are you allergic to?"},` +
			`{"type":"text","name":"fa_reaction","title":"What happens when you have it?"},` +
			`{"type":"radiogroup","name":"fa_severe","title":"Is it severe?","choices":[{"value":"no","text":"No"},{"value":"yes","text":"Yes"}]}]}`
		if got := fill(t, []Chapter{set("n1", "patient", mine)}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("a row's option that doesn't fit stops the whole list", func(t *testing.T) {
		changed := strings.Replace(allergies, `{"value":"n","text":"No"}`, `{"value":"x","text":"No"}`, 1)
		if got := fill(t, []Chapter{set("n1", "patient", changed)}, old); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})
	t.Run("a clinician-only question in a row is neither filled nor copied from", func(t *testing.T) {
		withCheck := `{"type":"paneldynamic","name":"meds","title":"Your medicines","templateElements":[` +
			`{"type":"text","name":"med","title":"Medicine"},{"type":"text","name":"dose_check","title":"Dose checked","clinicianOnly":true}]}`
		src := source("f1", march, []Chapter{set("c1", "patient", withCheck)}, map[string]string{})
		src.Clinician["c1"] = json.RawMessage(`{"meds":[{"med":"Aspirin","dose_check":"ok by nurse"}]}`)
		if d := data(fill(t, []Chapter{set("n1", "patient", withCheck)}, src), "n1"); d != `{"meds":[{"med":"Aspirin"}]}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("a question inside a repeating group isn't filled on its own", func(t *testing.T) {
		src := source("f1", march, []Chapter{set("c1", "patient", `{"type":"text","name":"allergies_what","title":"What are you allergic to?"}`)},
			map[string]string{"c1": `{"allergies_what":"Penicillin"}`})
		if got := fill(t, []Chapter{set("n1", "patient", allergies)}, src); len(got) != 0 {
			t.Errorf("filled %+v", got)
		}
	})

	list := func(name, col2 string) string {
		return `{"type":"matrixdynamic","name":"` + name + `","title":"Your operations","columns":[` +
			`{"name":"op","title":"Operation","cellType":"text"},` +
			`{"name":"` + col2 + `","title":"Where","cellType":"dropdown","choices":[{"value":"h1","text":"Here"},{"value":"h2","text":"Elsewhere"}]}]}`
	}
	ops := source("f1", march, []Chapter{set("c1", "patient", list("ops", "where"))}, map[string]string{"c1": `{"ops":[{"op":"Knee","where":"h2"}]}`})
	t.Run("a repeating list by key", func(t *testing.T) {
		if d := data(fill(t, []Chapter{set("n1", "patient", list("ops", "where"))}, ops), "n1"); d != `{"ops":[{"op":"Knee","where":"h2"}]}` {
			t.Errorf("data %s", d)
		}
	})
	t.Run("a repeating list by wording, columns by text", func(t *testing.T) {
		if d := data(fill(t, []Chapter{set("n1", "patient", list("fu_ops", "place"))}, ops), "n1"); d != `{"fu_ops":[{"op":"Knee","place":"h2"}]}` {
			t.Errorf("data %s", d)
		}
	})
}

func TestNothingToFill(t *testing.T) {
	if got := fill(t, []Chapter{set("n1", "patient", smoker)}); len(got) != 0 {
		t.Errorf("filled %+v", got)
	}
	empty := source("f1", march, []Chapter{set("c1", "patient", smoker)}, map[string]string{"c1": `{"smoker":""}`})
	if got := fill(t, []Chapter{set("n1", "patient", smoker)}, empty); len(got) != 0 {
		t.Errorf("filled %+v", got)
	}
	if _, err := Fill([]Chapter{{ID: "n1", Audience: "patient", Content: json.RawMessage(`[]`)}}, map[string]bool{"n1": true}, nil); err == nil {
		t.Error("bad content: no error")
	}
	// An earlier form's odd answers are skipped, never an error.
	odd := source("f1", march, []Chapter{set("c1", "patient", smoker)}, map[string]string{"c1": `[1,2]`})
	if got := fill(t, []Chapter{set("n1", "patient", smoker)}, odd); !reflect.DeepEqual(got, map[string]Filled{}) {
		t.Errorf("filled %+v", got)
	}
}
