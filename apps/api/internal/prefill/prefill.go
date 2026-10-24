// Package prefill fills a new HQ form's Question Sets from the patient's earlier answers (plan C4,
// session S25; PRE-01, PRE-06): if they answered something before, they only confirm it.
//
// A question of the new form takes an earlier answer when the earlier form asked the same
// question (S23 answers, "pre-fill matches by key, then by exact wording"):
//   - by key: the same `name` (library inserts, "Same as…", new versions);
//   - otherwise by wording: keys.SameWording (the same text, kind of answer and option texts),
//     never a text used twice in either questionnaire.
//
// The rules ("if in doubt, don't fill"):
//   - the earlier forms are taken newest submit first, and the newest one that asked the
//     question decides: if its answer is empty, or doesn't fit, nothing is filled (an older
//     answer may no longer be true);
//   - another trust's forms (S26; always used since S27, without asking: the caller reads them
//     one trust at a time) come after all of the new form's own trust's: they're tried only
//     for questions none of its own trust's forms asked. S23's lock freezes a keyed question's
//     options within one trust, not across trusts, so another trust's `q_smokes` may mean
//     something else (S26 verification 1). So another trust's question matches by key only if
//     the key is a **platform library key** (the same for every trust; Rahul, S26 answers "After
//     verification 1"), else by wording with all its safeguards; and either way every choice is
//     mapped by its option's text (no fill if a text doesn't match);
//   - per Question Set of an earlier form, the clinician's validated copy is used if there is
//     one, else the patient's answers;
//   - never filled: clinician-only questions, "Always ask fresh" ones, the profile, statements and
//     calculations, signatures, uploads, the BMI block (keys.TakesPart);
//   - a choice answer is copied only if each of its values is one of the new question's options:
//     by key, the same value; by wording, the option with the same text, as the new question's own
//     value. Any value that doesn't fit stops the whole answer;
//   - a repeating group's or list's rows are copied whole, keeping only the columns the new one
//     has, each cell fitted as a question of its own; a row that would lose every answer it has
//     stops the whole answer (S26); "Other" and comment text (`<key>-Comment`)
//     travels with its question.
//
// It walks the JSON only; it never evaluates SurveyJS expressions.
package prefill

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/keys"
)

// Chapter is one Question Set of a questionnaire version, as stored.
type Chapter struct {
	ID       string
	Audience string // patient, clinician or clinician_document
	Content  json.RawMessage
}

// Source is one of the patient's earlier submitted HQ forms: its version's Question Sets, and its
// answers per Question Set (chapter ID → SurveyJS data). Clinician holds only validated copies.
// Questionnaire is its questionnaire's name; OtherTrust marks a form of another trust than the new
// form's (S26), with its Hospital's name, both for the clinician's marker (PRE-04).
type Source struct {
	FormID        string
	SubmittedAt   time.Time
	Chapters      []Chapter
	Patient       map[string]json.RawMessage
	Clinician     map[string]json.RawMessage
	Questionnaire string
	OtherTrust    bool
	Hospital      string
}

// Entry says where a pre-filled answer came from: the form, its submit date (the date the patient
// sees), the value copied (so a later change shows, S26), "Other" or comment text copied with it,
// and how the question matched ("key" or "wording"). Questionnaire, and for another trust's form
// OtherTrust and Hospital, are stored with it (S26) so the clinician's marker never has to read
// another trust's records. Another trust's entry keeps no id of that trust's: no FromForm (S26
// verification 1); its own audit log records the source episode.
type Entry struct {
	FromForm      string          `json:"fromForm,omitempty"` // "" for another trust's: none of its ids is kept
	AnsweredAt    time.Time       `json:"answeredAt"`
	Value         json.RawMessage `json:"value"`
	Comment       json.RawMessage `json:"comment,omitempty"`
	MatchedBy     string          `json:"matchedBy"`
	Questionnaire string          `json:"questionnaire,omitempty"`
	OtherTrust    bool            `json:"otherTrust,omitempty"`
	Hospital      string          `json:"hospital,omitempty"`
}

// Filled is one Question Set's pre-filled answers (SurveyJS data) and where each came from.
type Filled struct {
	Data    map[string]json.RawMessage
	Prefill map[string]Entry
}

// Fill pre-fills the Question Sets named in `sets` of the new form's version (target: every set
// of the version, in order, so texts used twice anywhere in it are known). The result holds only
// the sets something was filled in. Only target content that isn't a form definition is an error;
// an earlier form's odd content or answers are skipped. library is the platform library's keys
// (LibraryKeys): the only keys another trust's form may match by.
func Fill(target []Chapter, sets map[string]bool, sources []Source, library map[string]bool) (map[string]Filled, error) {
	mine, err := walk(target)
	if err != nil {
		return nil, err
	}
	sources = slices.Clone(sources)
	slices.SortStableFunc(sources, func(a, b Source) int {
		if a.OtherTrust != b.OtherTrust { // the form's own trust first
			if a.OtherTrust {
				return 1
			}
			return -1
		}
		return b.SubmittedAt.Compare(a.SubmittedAt)
	})
	earlier := make([]earlierForm, 0, len(sources))
	for _, s := range sources {
		if f, ok := readSource(s); ok {
			earlier = append(earlier, f)
		}
	}
	repeated := keys.RepeatedTexts(mine)
	out := map[string]Filled{}
	for _, q := range mine {
		if !sets[q.Set] || !keys.TakesPart(q) || q.AlwaysFresh || q.Name == "" {
			continue
		}
		for _, f := range earlier {
			s, by := f.find(q, repeated, library)
			if by == "" {
				continue
			}
			// The newest form that asked it decides.
			answers := f.answers[s.Set]
			// Another trust's choices always by their text: its option values aren't this trust's.
			value, ok := fit(context{mine, f.questions}, q, s, answers[s.Name], by == "key" && !f.otherTrust)
			if !ok {
				break
			}
			got := out[q.Set]
			if got.Data == nil {
				got = Filled{Data: map[string]json.RawMessage{}, Prefill: map[string]Entry{}}
			}
			e := Entry{FromForm: f.id, AnsweredAt: f.at, Value: mustJSON(value), MatchedBy: by, Questionnaire: f.questionnaire}
			if f.otherTrust {
				e.FromForm, e.OtherTrust, e.Hospital = "", true, f.hospital
			}
			got.Data[q.Name] = e.Value
			if c, ok := answers[s.Name+"-Comment"]; ok && !empty(c) && takesComment(q, value) {
				e.Comment = mustJSON(c)
				got.Data[q.Name+"-Comment"] = e.Comment
			}
			got.Prefill[q.Name] = e
			out[q.Set] = got
			break
		}
	}
	return out, nil
}

// walk walks a version's Question Sets; each question's Set is its chapter's ID.
func walk(chs []Chapter) ([]keys.Placed, error) {
	sets := make([]keys.Set, len(chs))
	for i, c := range chs {
		sets[i] = keys.Set{Name: c.ID, Audience: c.Audience, Content: c.Content}
	}
	return keys.WalkSets(sets)
}

// earlierForm is a Source, walked and read.
type earlierForm struct {
	id string
	at time.Time
	// questionnaire, otherTrust, hospital: as Source's
	questionnaire string
	otherTrust    bool
	hospital      string
	questions     []keys.Placed
	repeated      map[string]bool
	answers       map[string]map[string]any // chapter ID → its answers
}

func readSource(s Source) (earlierForm, bool) {
	qs, err := walk(s.Chapters)
	if err != nil {
		return earlierForm{}, false
	}
	f := earlierForm{id: s.FormID, at: s.SubmittedAt, questionnaire: s.Questionnaire, otherTrust: s.OtherTrust, hospital: s.Hospital,
		questions: qs, repeated: keys.RepeatedTexts(qs), answers: map[string]map[string]any{}}
	for _, c := range s.Chapters {
		raw, ok := s.Clinician[c.ID]
		if !ok {
			raw = s.Patient[c.ID]
		}
		var data map[string]any
		if json.Unmarshal(raw, &data) == nil {
			f.answers[c.ID] = data
		}
	}
	return f, true
}

// find is the earlier form's question that q matches, and how: by key, else by wording; "" for none.
// Another trust's form matches by key only for a platform library key: its own keys aren't held to
// this trust's lock.
func (f earlierForm) find(q keys.Placed, repeatedHere, library map[string]bool) (keys.Placed, string) {
	for _, s := range f.questions {
		if (!f.otherTrust || library[q.Name]) && keys.TakesPart(s) && s.Name == q.Name {
			return s, "key"
		}
	}
	if t := keys.Normalize(q.Title); repeatedHere[t] || f.repeated[t] {
		return keys.Placed{}, ""
	}
	for _, s := range f.questions {
		if keys.TakesPart(s) && keys.SameWording(q.Question, s.Question) {
			return s, "wording"
		}
	}
	return keys.Placed{}, ""
}

// context holds both questionnaires' questions, for a repeating group's children.
type context struct{ mine, theirs []keys.Placed }

// children are a repeating group's patient questions (InList), in its Question Set: a
// clinician-only one is neither filled nor copied from.
func children(all []keys.Placed, panel keys.Placed) []keys.Placed {
	var out []keys.Placed
	for _, q := range all {
		if q.InList == panel.Name && q.Set == panel.Set && q.Answerable && !q.ClinicianOnly {
			out = append(out, q)
		}
	}
	return out
}

// fit is the earlier answer v (to question s) as an answer to question q, or false if it doesn't
// fit. byKey: the questions share a key, so option values are kept; else options are matched by
// their text.
func fit(c context, q, s keys.Placed, v any, byKey bool) (any, bool) {
	if empty(v) || q.Kind != s.Kind || q.Unit != s.Unit || q.Format != s.Format || q.Scale != s.Scale ||
		(q.Type == "paneldynamic") != (s.Type == "paneldynamic") {
		return nil, false
	}
	switch {
	case q.Type == "paneldynamic":
		return fitRows(v, byKey, columnsOf(c.mine, q), columnsOf(c.theirs, s), func(mine, theirs column, cell any) (any, bool) {
			return fit(c, mine.q, theirs.q, cell, byKey)
		})
	case q.Type == "matrixdynamic":
		return fitRows(v, byKey, listColumns(q), listColumns(s), func(mine, theirs column, cell any) (any, bool) {
			if mine.kind != theirs.kind {
				return nil, false
			}
			if len(mine.options) == 0 && len(theirs.options) == 0 {
				return cell, scalar(cell)
			}
			return fitChoice(mine.options, theirs.options, cell, byKey)
		})
	case q.Kind == "grid":
		return fitGrid(q, s, v, byKey)
	case q.Kind == "single", q.Kind == "boolean", q.Kind == "rating" && len(q.Options) > 0:
		return fitOne(optionsOf(q), optionsOf(s), v, byKey)
	case q.Kind == "multiple":
		return fitChoice(optionsOf(q), optionsOf(s), v, byKey)
	case q.Kind == "text", q.Kind == "number", q.Kind == "date", q.Kind == "rating":
		return v, scalar(v)
	}
	// Anything else (a medication list, a table…) only as it is, into the same answer.
	return v, keys.SameAnswer(s.Question, q.Question)
}

// option is an answer's possible value as stored (a string, a number, true/false) and its text.
type option struct {
	value any
	text  string
}

// optionsOf reads a question's options with their stored values: a choice question's (Other and
// None included), a Yes/No switch's, a rating's own values. keys.Question has them as strings.
func optionsOf(q keys.Placed) []option {
	var el map[string]json.RawMessage
	_ = json.Unmarshal(q.Raw, &el)
	switch q.Type {
	case "boolean":
		yes, no := decode(el["valueTrue"]), decode(el["valueFalse"])
		if yes == nil {
			yes = true
		}
		if no == nil {
			no = false
		}
		return []option{{yes, q.Options[0].Text}, {no, q.Options[1].Text}}
	case "rating":
		return items(el["rateValues"], q.Options)
	}
	return items(el["choices"], q.Options)
}

// items pairs a SurveyJS item list's stored values with the texts keys read for them (in the same
// order; the Other and None options keys adds come after, with their own values).
func items(raw json.RawMessage, texts []keys.Option) []option {
	var list []json.RawMessage
	_ = json.Unmarshal(raw, &list)
	var values []any
	for _, it := range list {
		switch x := decode(it).(type) {
		case string, float64, bool:
			values = append(values, x)
		case map[string]any:
			if x["value"] != nil {
				values = append(values, x["value"])
			}
		}
	}
	out := make([]option, len(texts))
	for i, t := range texts {
		out[i] = option{value: t.Value, text: t.Text}
		if i < len(values) && fmt.Sprint(values[i]) == t.Value {
			out[i].value = values[i]
		}
	}
	return out
}

// fitOne fits one choice value: by key, the new question's option with the same value; by
// wording, the one whose text is the earlier option's (exactly one).
func fitOne(mine, theirs []option, v any, byKey bool) (any, bool) {
	if !scalar(v) {
		return nil, false
	}
	key := fmt.Sprint(v)
	find := func(list []option, match func(option) bool) []option {
		var out []option
		for _, o := range list {
			if match(o) {
				out = append(out, o)
			}
		}
		return out
	}
	if byKey {
		if m := find(mine, func(o option) bool { return fmt.Sprint(o.value) == key }); len(m) == 1 {
			return m[0].value, true
		}
		return nil, false
	}
	was := find(theirs, func(o option) bool { return fmt.Sprint(o.value) == key })
	if len(was) != 1 {
		return nil, false
	}
	m := find(mine, func(o option) bool { return keys.Normalize(o.text) == keys.Normalize(was[0].text) })
	if len(m) != 1 {
		return nil, false
	}
	return m[0].value, true
}

// fitChoice fits a choice answer: one value, or a list of them (multiple choice), all or nothing.
func fitChoice(mine, theirs []option, v any, byKey bool) (any, bool) {
	list, isList := v.([]any)
	if !isList {
		return fitOne(mine, theirs, v, byKey)
	}
	out := make([]any, 0, len(list))
	for _, x := range list {
		y, ok := fitOne(mine, theirs, x, byKey)
		if !ok {
			return nil, false
		}
		out = append(out, y)
	}
	return out, len(out) > 0
}

// fitGrid fits a grid's answer ({row: column}): its rows and columns are both options.
func fitGrid(q, s keys.Placed, v any, byKey bool) (any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	var el, sel map[string]json.RawMessage
	_ = json.Unmarshal(q.Raw, &el)
	_ = json.Unmarshal(s.Raw, &sel)
	rows, theirRows := items(el["rows"], q.Rows), items(sel["rows"], s.Rows)
	cols, theirCols := items(el["columns"], q.Options), items(sel["columns"], s.Options)
	out := map[string]any{}
	for row, col := range m {
		r, ok := fitOne(rows, theirRows, row, byKey)
		if !ok {
			return nil, false
		}
		c, ok := fitOne(cols, theirCols, col, byKey)
		if !ok {
			return nil, false
		}
		out[fmt.Sprint(r)] = c
	}
	return out, true
}

// column is one column of a repeating group (a question) or list (a cell type and its options).
type column struct {
	name, text, kind string
	options          []option
	q                keys.Placed // a repeating group's question
}

func columnsOf(all []keys.Placed, panel keys.Placed) []column {
	var out []column
	for _, q := range children(all, panel) {
		out = append(out, column{name: q.Name, text: q.Title, q: q})
	}
	return out
}

// listColumns reads a repeating list's (matrixdynamic) columns with their options' stored values.
func listColumns(q keys.Placed) []column {
	var el map[string]json.RawMessage
	_ = json.Unmarshal(q.Raw, &el)
	var raws []json.RawMessage
	_ = json.Unmarshal(el["columns"], &raws)
	out := make([]column, len(q.Columns))
	for i, c := range q.Columns {
		out[i] = column{name: c.Name, text: c.Text, kind: c.Kind}
		if i < len(raws) {
			var col map[string]json.RawMessage
			_ = json.Unmarshal(raws[i], &col)
			out[i].options = items(col["choices"], c.Options)
		}
	}
	return out
}

// fitRows fits a repeating group's or list's rows: each earlier column goes to the new column with
// the same name (by key) or text (by wording, exactly one); columns the new one lacks are left
// out, and each cell is fitted by cell. A cell that doesn't fit stops the whole answer, and so does
// a row whose every answer is in columns the new one lacks (it would be lost without a word).
func fitRows(v any, byKey bool, mine, theirs []column, cell func(mine, theirs column, v any) (any, bool)) (any, bool) {
	rows, ok := v.([]any)
	if !ok {
		return nil, false
	}
	to := map[string]column{}   // their column's name → mine
	from := map[string]column{} // their column's name → theirs
	for _, t := range theirs {
		var m []column
		for _, c := range mine {
			if (byKey && c.name == t.name) || (!byKey && keys.Normalize(c.text) == keys.Normalize(t.text) && keys.Normalize(t.text) != "") {
				m = append(m, c)
			}
		}
		switch {
		case len(m) == 1:
			to[t.name], from[t.name] = m[0], t
		case len(m) > 1:
			return nil, false
		}
	}
	taken := map[string]bool{}
	for _, c := range to {
		if taken[c.name] {
			return nil, false // two earlier columns into one
		}
		taken[c.name] = true
	}
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		row, ok := r.(map[string]any)
		if !ok {
			return nil, false
		}
		kept := map[string]any{}
		answered := false
		for k, x := range row {
			if empty(x) {
				continue
			}
			base, isComment := strings.CutSuffix(k, "-Comment")
			answered = answered || !isComment // a stale "Other" text alone isn't an answer
			c, mapped := to[base]
			if !mapped {
				continue
			}
			if isComment {
				if c.q.Name != "" && takesComment(c.q, row[base]) {
					kept[c.name+"-Comment"] = x
				}
				continue
			}
			y, ok := cell(c, from[base], x)
			if !ok {
				return nil, false
			}
			kept[c.name] = y
		}
		switch {
		case len(kept) > 0:
			out = append(out, kept)
		case answered:
			return nil, false
		}
	}
	return out, len(out) > 0
}

// takesComment reports whether question q keeps `<name>-Comment` with this answer: its comment box
// (showCommentArea), or its "Other (please specify)" chosen.
func takesComment(q keys.Placed, value any) bool {
	var el map[string]json.RawMessage
	_ = json.Unmarshal(q.Raw, &el)
	if flag(el["showCommentArea"]) || flag(el["hasComment"]) {
		return true
	}
	if !flag(el["showOtherItem"]) && !flag(el["hasOther"]) {
		return false
	}
	if list, ok := value.([]any); ok {
		return slices.ContainsFunc(list, func(x any) bool { return x == "other" })
	}
	return value == "other"
}

func flag(raw json.RawMessage) bool {
	var b bool
	return json.Unmarshal(raw, &b) == nil && b
}

func decode(raw json.RawMessage) any {
	var v any
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return v
}

func scalar(v any) bool {
	switch v.(type) {
	case string, float64, bool:
		return true
	}
	return false
}

// empty is SurveyJS's "no answer": missing, null, "", [] or {}.
func empty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil { // decoded from JSON, so it encodes
		panic(err)
	}
	return b
}

// LibraryKeys are the keys of the platform library's questions (library_questions with no trust),
// each element's and its children's (a "yes/no + details" group keeps both, S24), for Fill.
// An element that can't be read adds nothing.
func LibraryKeys(elements []json.RawMessage) map[string]bool {
	out := map[string]bool{}
	var add func(raw json.RawMessage)
	add = func(raw json.RawMessage) {
		var el struct {
			Name             string            `json:"name"`
			Elements         []json.RawMessage `json:"elements"`
			TemplateElements []json.RawMessage `json:"templateElements"`
		}
		if json.Unmarshal(raw, &el) != nil {
			return
		}
		if el.Name != "" {
			out[el.Name] = true
		}
		for _, c := range append(el.Elements, el.TemplateElements...) {
			add(c)
		}
	}
	for _, e := range elements {
		add(e)
	}
	return out
}
