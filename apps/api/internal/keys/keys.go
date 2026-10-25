// Package keys reads the question keys in a questionnaire's SurveyJS JSON (plan C4, session S23).
//
// A question's key is its `name`, the stable ID the editor generates (QST-04). Pre-fill (S25) copies
// an answer by key, or failing that by exactly the same wording (S23 answers), so a key must mean
// the same thing everywhere it's used:
//   - a key is used once in a questionnaire (Duplicates);
//   - once published, a keyed question's whole answer is frozen: kind, unit, format, medication
//     type, options (values and wording), rows, scale, columns (LockProblems);
//   - Matches finds the trust's other questionnaires a question would pre-fill from (PRE-05).
//   - Compatible and Candidates are the editor's "Same as…" (S24): which keys a question may take.
//
// It walks the JSON only; it never evaluates SurveyJS expressions.
package keys

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// ErrBadContent is returned when a chapter's JSON isn't an object.
var ErrBadContent = errors.New("chapter content is not a valid form definition")

// Option is one of a choice question's options: its stored value and its text in the default
// language. An option without text shows its value, so its text is the value.
type Option struct {
	Value string `json:"value"`
	Text  string `json:"text"`
}

// Question is one element of a chapter that isn't a group: a question, a statement or a calculation.
type Question struct {
	Name string
	Type string // the SurveyJS type
	// Kind is the kind of answer: single, multiple, boolean, text, number, date, dates, list, or
	// the type itself (grid, rating, bmi, medication, signaturepad, file…).
	Kind string
	Unit string // a number's unit (`unit`), e.g. "kg"
	// Format is a date's format (day, month and year; month and year; year only), a text
	// question's input format (any text, email address, a pattern…) or a medication block's type
	// (prescribed medication, non-prescribed medication, recreational drugs); "" for other kinds.
	Format string
	Title  string // the question text in the default language; "" when it has none
	// Options are what an answer can hold, each a stored value and its text: a choice question's
	// options ("Other" and "None" included), a Yes/No switch's two labels, a grid's columns, a
	// rating's values. nil when it has none.
	Options []Option
	Rows    []Option // a grid's rows
	Scale   string   // a rating's scale when it has no values, e.g. "1 to 5 by 1"
	// Columns are a repeating list's: a dynamic panel's questions, or a dynamic matrix's columns.
	Columns []Column
	// Answerable is false for statements, calculations, the profile and Clinical Summaries, which nobody answers.
	Answerable bool
	// ClinicianOnly is set on the question, a group or page around it, or its Question Set.
	ClinicianOnly bool
	AlwaysFresh   bool   // "Always ask fresh": never pre-filled
	InList        string // the repeating group (dynamic panel) whose rows it's in, or ""
	// Raw is the element's own JSON, as in the chapter: "Same as…" copies its options from it (S24).
	Raw json.RawMessage
}

// Column is one column of a repeating list: its key, its text, its kind of answer and its options.
type Column struct {
	Name    string
	Text    string
	Kind    string
	Options []Option
}

// Label is how a question is named in a message: its text, or its key when it has none.
func (q Question) Label() string {
	if q.Title != "" {
		return q.Title
	}
	return q.Name
}

// notAnswered are the element types that hold no answer (a Clinical Summary's comments, S32, are a
// clinician's answer under a key the JSON never names).
var notAnswered = map[string]bool{"html": true, "expression": true, "image": true, "profile": true, "clinicalsummary": true}

// obj is a JSON object read leniently: a field of an unexpected shape reads as empty, so a chapter
// SurveyJS accepts never stops the walk (one odd published chapter would otherwise stop every
// publish in the trust).
type obj map[string]json.RawMessage

func (o obj) str(k string) string {
	var s string
	_ = json.Unmarshal(o[k], &s)
	return s
}

func (o obj) flag(k string) bool {
	var b bool
	_ = json.Unmarshal(o[k], &b)
	return b
}

// num reads a number, or def when it's missing or not a number.
func (o obj) num(k string, def float64) float64 {
	var f float64
	if json.Unmarshal(o[k], &f) != nil {
		return def
	}
	return f
}

// list reads an array; anything else is an empty one.
func (o obj) list(k string) []json.RawMessage {
	var l []json.RawMessage
	_ = json.Unmarshal(o[k], &l)
	return l
}

// objects reads an array's objects, skipping anything else in it.
func (o obj) objects(k string) []obj {
	var out []obj
	for _, raw := range o.list(k) {
		var e obj
		if json.Unmarshal(raw, &e) == nil && e != nil {
			out = append(out, e)
		}
	}
	return out
}

// Walk lists every element of a chapter except groups (panels), in order, at any depth,
// including a dynamic panel's children. Only content that isn't a JSON object is an error.
func Walk(content json.RawMessage) ([]Question, error) {
	var doc obj
	if err := json.Unmarshal(content, &doc); err != nil || doc == nil {
		return nil, ErrBadContent
	}
	var out []Question
	for _, p := range doc.objects("pages") {
		walk(p.list("elements"), p.flag("clinicianOnly"), "", &out)
	}
	return out, nil
}

func walk(els []json.RawMessage, clinician bool, list string, out *[]Question) {
	for _, raw := range els {
		var el obj
		if json.Unmarshal(raw, &el) != nil || el == nil {
			continue
		}
		clin := clinician || el.flag("clinicianOnly")
		typ := el.str("type")
		if typ == "panel" {
			walk(el.list("elements"), clin, list, out)
			continue
		}
		q := Question{
			Name: el.str("name"), Type: typ, Kind: kindOf(el), Unit: el.str("unit"), Format: formatOf(el), Title: text(el["title"]),
			Answerable: !notAnswered[typ], ClinicianOnly: clin, AlwaysFresh: el.flag("alwaysFresh"), InList: list, Raw: raw,
		}
		switch typ {
		case "matrix":
			q.Rows, q.Options = items(el.list("rows")), items(el.list("columns"))
		case "rating":
			if values := el.list("rateValues"); len(values) > 0 {
				q.Options = items(values)
			} else {
				q.Scale = scaleOf(el)
			}
		case "boolean":
			q.Options = []Option{
				{Value: or(el.str("valueTrue"), "true"), Text: or(text(el["labelTrue"]), "Yes")},
				{Value: or(el.str("valueFalse"), "false"), Text: or(text(el["labelFalse"]), "No")},
			}
		case "paneldynamic":
			for _, c := range el.objects("templateElements") {
				if t := c.str("type"); !notAnswered[t] && t != "panel" {
					q.Columns = append(q.Columns, Column{Name: c.str("name"), Text: labelOf(c.str("name"), c["title"]), Kind: kindOf(c), Options: choices(c)})
				}
			}
		case "matrixdynamic", "matrixdropdown":
			q.Rows = items(el.list("rows"))
			for _, raw := range el.list("columns") {
				q.Columns = append(q.Columns, columnOf(raw))
			}
		default:
			q.Options = choices(el)
		}
		*out = append(*out, q)
		if typ == "paneldynamic" {
			walk(el.list("templateElements"), clin, q.Name, out)
		}
	}
}

func kindOf(el obj) string {
	switch t := el.str("type"); t {
	case "radiogroup", "dropdown":
		return "single"
	case "checkbox", "tagbox":
		return "multiple"
	case "comment":
		return "text"
	case "text":
		switch it := el.str("inputType"); {
		case len(el["dateFormat"]) > 0 || it == "date" || it == "month":
			return "date"
		case it == "number" || el.str("maskType") == "numeric":
			return "number"
		}
		return "text"
	case "matrixdynamic":
		if el.flag("dateList") {
			return "dates"
		}
		return "list"
	case "paneldynamic":
		return "list"
	case "matrix":
		return "grid"
	default:
		return t
	}
}

// formatOf is a date's format, as the editor sets it (fields.ts dateField), a text question's
// input format, read from its validators as the editor writes them (fields.ts setTextFormat), or a
// medication block's type (S23 verification 3: pre-fill must never copy prescribed medicines into a
// list of recreational drugs). A block without a type is prescribed, the editor's default.
func formatOf(el obj) string {
	switch kindOf(el) {
	case "medication":
		switch t := el.str("medicationType"); t {
		case "", "prescribed":
			return "prescribed medication"
		case "non_prescribed":
			return "non-prescribed medication"
		case "recreational":
			return "recreational drugs"
		default:
			return t
		}
	case "date":
		switch {
		case el.str("dateFormat") == "y":
			return "year only"
		case el.str("inputType") == "month":
			return "month and year"
		}
		return "day, month and year"
	case "text":
		for _, v := range el.objects("validators") {
			switch v.str("type") {
			case "email":
				return "email address"
			case "regex":
				if name, ok := textFormatNames[v.str("regex")]; ok {
					return name
				}
				return "the pattern “" + v.str("regex") + "”"
			}
		}
		switch it := el.str("inputType"); it {
		case "email":
			return "email address"
		case "", "text", "tel":
			return "any text"
		default:
			return it
		}
	}
	return ""
}

// textFormatNames are the editor's known text formats by their pattern (packages/clinical/src/
// fields.ts TEXT_FORMATS; a test keeps the two the same), so the publish refusal names a format in
// words: "its answer format changed from any text to phone number" (S27; Rahul, 24 Oct). Any other
// pattern is a custom one, named by itself. "NHS number" and "UK postcode" are hardcoded, as in
// the editor: a recorded exception to A-20 (decisions.md §2, 24 Oct), to move to the per-country
// config when a second country comes.
var textFormatNames = map[string]string{
	`^\+?[0-9 ()-]{7,20}$`:                        "phone number",
	`^\d{3} ?\d{3} ?\d{4}$`:                       "NHS number",
	`^[A-Za-z]{1,2}\d[A-Za-z\d]? ?\d[A-Za-z]{2}$`: "UK postcode",
}

// columnOf reads a dynamic matrix's column: a plain string is its name and text.
func columnOf(raw json.RawMessage) Column {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return Column{Name: s, Text: s}
	}
	var c obj
	_ = json.Unmarshal(raw, &c)
	kind := c.str("cellType")
	if it := c.str("inputType"); it != "" {
		kind += ":" + it
	}
	if df := c.str("dateFormat"); df != "" {
		kind += ":" + df
	}
	return Column{Name: c.str("name"), Text: labelOf(c.str("name"), c["title"]), Kind: kind, Options: choices(c)}
}

// scaleOf is a rating's scale without values: SurveyJS's defaults are 1 to 5 by 1, and a count
// sets the top.
func scaleOf(el obj) string {
	low, step := el.num("rateMin", 1), el.num("rateStep", 1)
	high := el.num("rateMax", 5)
	if n := el.num("rateCount", 0); n > 0 {
		high = low + step*(n-1)
	}
	f := func(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }
	return f(low) + " to " + f(high) + " by " + f(step)
}

// items reads a SurveyJS item list (choices, rows, columns, rating values): a plain string or
// number is both value and text; an object has a value and a text (its value when it has none).
// Anything else is skipped.
func items(list []json.RawMessage) []Option {
	var out []Option
	for _, raw := range list {
		var v any
		if json.Unmarshal(raw, &v) != nil {
			continue
		}
		switch x := v.(type) {
		case string, float64, bool:
			s := fmt.Sprint(x)
			out = append(out, Option{Value: s, Text: s})
		case map[string]any:
			if x["value"] == nil {
				continue
			}
			var c struct {
				Text json.RawMessage `json:"text"`
			}
			_ = json.Unmarshal(raw, &c)
			value := fmt.Sprint(x["value"])
			out = append(out, Option{Value: value, Text: labelOf(value, c.Text)})
		}
	}
	return out
}

// choices are a choice question's options, SurveyJS's own "Other (please specify)" and "None"
// (stored as "other" and "none") included. nil when it has none.
func choices(el obj) []Option {
	out := items(el.list("choices"))
	if el.flag("showOtherItem") || el.flag("hasOther") {
		out = append(out, Option{Value: "other", Text: or(text(el["otherText"]), "Other (please specify)")})
	}
	if el.flag("showNoneItem") || el.flag("hasNone") {
		out = append(out, Option{Value: "none", Text: or(text(el["noneText"]), "None")})
	}
	return out
}

// text reads a SurveyJS string: plain, or per language ({"default": "…", "fr": "…"}). It returns
// the default language's, trimmed.
func text(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) == nil {
		return strings.TrimSpace(m["default"])
	}
	return ""
}

func labelOf(name string, title json.RawMessage) string { return or(text(title), name) }

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Set is one Question Set (chapter) of a questionnaire version.
type Set struct {
	Name     string
	Audience string // patient, clinician or clinician_document
	Content  json.RawMessage
}

// Placed is a question and the Question Set it's in.
type Placed struct {
	Question
	Set string
}

// WalkSets walks every Question Set in order. A clinician set's questions are all clinician-only.
func WalkSets(sets []Set) ([]Placed, error) {
	var out []Placed
	for _, s := range sets {
		qs, err := Walk(s.Content)
		if err != nil {
			return nil, err
		}
		for _, q := range qs {
			q.ClinicianOnly = q.ClinicianOnly || s.Audience != "patient"
			out = append(out, Placed{Question: q, Set: s.Name})
		}
	}
	return out, nil
}

// Problem blocks publishing; Set is the Question Set to fix. The editor shows it as it shows its
// own publish problems.
type Problem struct {
	Set     string `json:"set"`
	Message string `json:"message"`
}

// Duplicates is a problem for each extra use of a key in a questionnaire, naming both Question
// Sets (S23 answers, 3). Groups don't count: nothing is stored under a group's name.
func Duplicates(qs []Placed) []Problem {
	first := map[string]Placed{}
	var out []Problem
	for _, q := range qs {
		if q.Name == "" {
			continue // nothing can be stored under no name (only an import or the API makes one)
		}
		f, seen := first[q.Name]
		if !seen {
			first[q.Name] = q
			continue
		}
		where := fmt.Sprintf("“%s” in “%s” and “%s” in “%s”", f.Label(), f.Set, q.Label(), q.Set)
		if f.Set == q.Set {
			where = fmt.Sprintf("twice in “%s”: “%s” and “%s”", q.Set, f.Label(), q.Label())
		}
		out = append(out, Problem{Set: q.Set, Message: fmt.Sprintf("The question ID %s is used %s. Each question needs its own ID.", q.Name, where)})
	}
	return out
}

// Published is a key's question as published: in the newest published or retired version of a
// questionnaire of the trust that holds the key.
type Published struct {
	Questionnaire string
	VersionNo     int
	Question      Question
}

// LockProblems is a problem for each way a draft question's answer differs from the same key's
// published question (S23 answers, 2, and "after verification 1": frozen completely). Its kind of
// answer, unit, options, a grid's rows and columns, a rating's scale and a repeating list's columns
// can't change. The order of options, rows and columns may, and so may translations and the
// question's own text. A key published on a statement or calculation, which holds no answer, locks
// nothing.
func LockProblems(draft []Placed, published map[string][]Published) []Problem {
	var out []Problem
	for _, d := range draft {
		shapes := publishedShapes(published[d.Name])
		if len(shapes) > 1 {
			// Published before the lock in shapes that disagree: no draft can match them all, and
			// listing each difference would say "Yes was removed" and "Yes was added" at once.
			where := make([]string, len(shapes))
			for i, sh := range shapes {
				where[i] = strings.Join(sh.sources, " and ")
			}
			out = append(out, Problem{Set: d.Set, Message: fmt.Sprintf("“%s”: the trust has already published this question with different answers (in %s), "+
				"so it can't be published again as it is. Add a new question instead, or ask for the old questionnaires to be tidied up.",
				d.Label(), strings.Join(where, "; "))})
			continue
		}
		for _, sh := range shapes {
			for _, c := range answerChanges(sh.question, d.Question) {
				out = append(out, Problem{Set: d.Set, Message: fmt.Sprintf("“%s”: %s. A published question's answers can't change: add a new question instead (published in %s).",
					d.Label(), c, strings.Join(sh.sources, " and "))})
			}
		}
	}
	return out
}

// shape is one way a key was published: a question, and where it was published like that.
type shape struct {
	question Question
	sources  []string
}

// publishedShapes groups a key's published questions by their answer (answerChanges finds no
// difference), in order of first finding. Statements and calculations hold no answer and lock nothing.
func publishedShapes(list []Published) []shape {
	var out []shape
	for _, p := range list {
		if !p.Question.Answerable {
			continue
		}
		where := fmt.Sprintf("“%s” v%d", p.Questionnaire, p.VersionNo)
		i := slices.IndexFunc(out, func(sh shape) bool { return len(answerChanges(sh.question, p.Question)) == 0 })
		if i < 0 {
			out = append(out, shape{question: p.Question, sources: []string{where}})
		} else {
			out[i].sources = append(out[i].sources, where)
		}
	}
	return out
}

// kindNames are the kinds of answer as an author knows them.
var kindNames = map[string]string{
	"single": "single choice", "multiple": "multiple choice", "boolean": "Yes/No switch", "dates": "list of dates",
	"list": "repeating list", "grid": "grid", "bmi": "BMI", "signaturepad": "signature", "file": "file upload",
	"html": "statement", "expression": "calculation", "matrixdropdown": "table",
}

func kindName(k string) string { return or(kindNames[k], k) }

// answerChanges lists how the answer `now` can hold differs from `was`'s. A changed kind of answer
// is the only change listed, since everything else follows from it.
func answerChanges(was, now Question) []string {
	if was.Kind != now.Kind {
		return []string{fmt.Sprintf("its kind of answer changed from %s to %s", kindName(was.Kind), kindName(now.Kind))}
	}
	var out []string
	if was.Format != now.Format {
		out = append(out, fmt.Sprintf("its %s changed from %s to %s", formatNoun(was.Kind), was.Format, now.Format))
	}
	if was.Unit != now.Unit {
		out = append(out, fmt.Sprintf("its unit changed from %s to %s", quoted(was.Unit), quoted(now.Unit)))
	}
	if was.Scale != now.Scale {
		out = append(out, fmt.Sprintf("its scale changed from %s to %s", or(was.Scale, "its own values"), or(now.Scale, "its own values")))
	}
	out = append(out, itemChanges(optionNoun(was), was.Options, now.Options)...)
	out = append(out, itemChanges("row", was.Rows, now.Rows)...)
	for _, w := range was.Columns {
		i := slices.IndexFunc(now.Columns, func(c Column) bool { return c.Name == w.Name })
		if i < 0 {
			out = append(out, fmt.Sprintf("the column “%s” was removed", w.Text))
			continue
		}
		if was.Type == "paneldynamic" {
			continue // its questions are checked as questions of their own (their text may change)
		}
		n := now.Columns[i]
		if n.Text != w.Text {
			out = append(out, fmt.Sprintf("the column “%s” was reworded to “%s”", w.Text, n.Text))
		}
		if n.Kind != w.Kind {
			out = append(out, fmt.Sprintf("the column “%s” changed its kind of answer", w.Text))
		}
		for _, c := range itemChanges("option", w.Options, n.Options) {
			out = append(out, fmt.Sprintf("the column “%s”: %s", w.Text, c))
		}
	}
	for _, n := range now.Columns {
		if !slices.ContainsFunc(was.Columns, func(c Column) bool { return c.Name == n.Name }) {
			out = append(out, fmt.Sprintf("the column “%s” was added", n.Text))
		}
	}
	return out
}

// formatNoun is what a Format is, for a kind of answer.
func formatNoun(kind string) string {
	switch kind {
	case "date":
		return "date format"
	case "medication":
		return "medication type"
	}
	return "answer format"
}

func optionNoun(q Question) string {
	switch q.Kind {
	case "grid":
		return "column"
	case "rating":
		return "value"
	case "boolean":
		return "label"
	}
	return "option"
}

func quoted(s string) string {
	if s == "" {
		return "none"
	}
	return "“" + s + "”"
}

// itemChanges lists the items (options, rows…) removed, reworded or added, matched by value.
func itemChanges(noun string, was, now []Option) []string {
	var out []string
	find := func(list []Option, value string) (Option, bool) {
		i := slices.IndexFunc(list, func(o Option) bool { return o.Value == value })
		if i < 0 {
			return Option{}, false
		}
		return list[i], true
	}
	for _, w := range was {
		n, ok := find(now, w.Value)
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("the %s “%s” was removed", noun, w.Text))
		case n.Text != w.Text:
			out = append(out, fmt.Sprintf("the %s “%s” was reworded to “%s”", noun, w.Text, n.Text))
		}
	}
	for _, n := range now {
		if _, ok := find(was, n.Value); !ok {
			out = append(out, fmt.Sprintf("the %s “%s” was added", noun, n.Text))
		}
	}
	return out
}

// Normalize is a text for comparing wording: lower case, without punctuation, and with single
// spaces between words.
func Normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// placeholders are the texts the editor gives a new question (packages/clinical, newElement).
// Nobody chose that wording, so it never matches by wording.
var placeholders = map[string]bool{
	"text question": true, "yes no question": true, "select one question": true, "select many question": true,
	"date question": true, "number question": true, "rating question": true, "grid question": true,
	"group": true, "section": true, "calculation": true, "upload a file": true, "please sign below": true,
}

// SameWording reports whether two questions are the same by wording (S23 answers): the same text
// (Normalize), the same kind of answer (and unit, and a rating's scale), and the same texts for what
// the answer can hold, in any order: options, a grid's rows, a repeating list's columns (with
// their kinds of answer). A question with no text, or with the editor's placeholder text, never
// matches.
func SameWording(a, b Question) bool {
	if Normalize(a.Title) == "" || placeholders[Normalize(a.Title)] || Normalize(a.Title) != Normalize(b.Title) ||
		a.Kind != b.Kind || (a.Type == "paneldynamic") != (b.Type == "paneldynamic") || a.Unit != b.Unit || a.Format != b.Format || a.Scale != b.Scale {
		return false
	}
	texts := func(list []Option) []string {
		out := make([]string, len(list))
		for i, o := range list {
			out[i] = o.Text
		}
		return out
	}
	columns := func(q Question) []string {
		out := make([]string, len(q.Columns))
		for i, c := range q.Columns {
			out[i] = Normalize(c.Text) + "|" + c.Kind
		}
		return out
	}
	return sameTexts(texts(a.Options), texts(b.Options)) && sameTexts(texts(a.Rows), texts(b.Rows)) && sameTexts(columns(a), columns(b))
}

func sameTexts(a, b []string) bool {
	norm := func(list []string) []string {
		out := make([]string, len(list))
		for i, s := range list {
			out[i] = Normalize(s)
		}
		slices.Sort(out)
		return out
	}
	return slices.Equal(norm(a), norm(b))
}

// RepeatedTexts are the question texts (Normalize) used by more than one answerable question in a
// questionnaire, e.g. "please give details". They never match by wording (S23 answers).
func RepeatedTexts(qs []Placed) map[string]bool {
	seen := map[string]int{}
	for _, q := range qs {
		if t := Normalize(q.Title); q.Answerable && t != "" {
			seen[t]++
		}
	}
	out := map[string]bool{}
	for t, n := range seen {
		if n > 1 {
			out[t] = true
		}
	}
	return out
}

// oneOff are answers never copied into another questionnaire (verification 1, item 2): a signature
// attests to one document, an upload is one-off evidence, and the BMI calculator's weight changes
// (and, a fixed block, it has no "Always ask fresh" switch).
var oneOff = map[string]bool{"signaturepad": true, "file": true, "bmi": true}

// OneOff reports whether a SurveyJS type's answer is never copied into another questionnaire.
func OneOff(typ string) bool { return oneOff[typ] }

// TakesPart reports whether a question takes part in pre-fill: a patient's answer at the top level
// (a repeating group's children travel in its rows), and not a one-off answer. "Always ask fresh"
// is the pre-fill's own rule (S25): such a question still matches, as the editor's label shows.
func TakesPart(q Placed) bool {
	return q.Answerable && !q.ClinicianOnly && q.InList == "" && !oneOff[q.Type]
}

// SameAnswer reports whether two questions' answers have the same shape, as the publish lock
// compares them (answerChanges finds no difference): an answer to one fits the other as it is.
func SameAnswer(a, b Question) bool { return len(answerChanges(a, b)) == 0 }

// Version is one published or retired version of another of the trust's questionnaires.
type Version struct {
	QuestionnaireID string
	Questionnaire   string
	Questions       []Placed
}

// Match is another questionnaire a question would pre-fill from (PRE-05): by "key" (the same
// key) or by "wording" (SameWording).
type Match struct {
	QuestionnaireID string `json:"questionnaireId"`
	Name            string `json:"name"`
	By              string `json:"by"`
}

// Matches lists, for each patient question of mine, the other questionnaires it matches, in the
// order of others: by key in any of their versions, else by wording, never by a text used twice
// in either questionnaire. A question with no match has an empty list ("New question").
func Matches(mine []Placed, others []Version) map[string][]Match {
	repeatedHere := RepeatedTexts(mine)
	type found struct {
		match Match
		order int
	}
	byQuestion := map[string]map[string]*found{} // my key → questionnaire → match
	out := map[string][]Match{}
	for _, m := range mine {
		if TakesPart(m) {
			out[m.Name] = []Match{}
			byQuestion[m.Name] = map[string]*found{}
		}
	}
	for order, v := range others {
		repeatedThere := RepeatedTexts(v.Questions)
		keys := map[string]bool{}
		for _, o := range v.Questions {
			if TakesPart(o) {
				keys[o.Name] = true
			}
		}
		for _, m := range mine {
			if !TakesPart(m) {
				continue
			}
			by := ""
			if keys[m.Name] {
				by = "key"
			} else if t := Normalize(m.Title); !repeatedHere[t] && !repeatedThere[t] {
				for _, o := range v.Questions {
					if TakesPart(o) && SameWording(m.Question, o.Question) {
						by = "wording"
						break
					}
				}
			}
			if by == "" {
				continue
			}
			f := byQuestion[m.Name][v.QuestionnaireID]
			if f == nil {
				byQuestion[m.Name][v.QuestionnaireID] = &found{Match{v.QuestionnaireID, v.Questionnaire, by}, order}
			} else if by == "key" {
				f.match.By = "key"
			}
		}
	}
	for name, qs := range byQuestion {
		list := make([]*found, 0, len(qs))
		for _, f := range qs {
			list = append(list, f)
		}
		slices.SortFunc(list, func(a, b *found) int { return a.order - b.order })
		for _, f := range list {
			out[name] = append(out[name], f.match)
		}
	}
	return out
}

// Compatible reports whether question `mine` may take the key of `theirs` with "Same as…" (S24
// answers 3 and 4), and if not, why, in a sentence for the author. It needs the same kind of
// answer: single choice with single choice (option buttons, a dropdown, Yes/No buttons), multiple
// with multiple, a number with a number of the same unit, a date with a date, text with text.
// Choices are compatible whatever their options, because adopting copies the key and the options;
// for any other kind nothing else is copied, so everything the publish lock freezes must already
// be the same (unit, date or text format, medication type, scale, rows, labels). Repeating lists
// get their keys from the library or a new version, never from "Same as…".
func Compatible(mine, theirs Question) (bool, string) {
	list := func(q Question) bool { return q.Kind == "list" || q.Kind == "dates" || q.Type == "paneldynamic" }
	switch {
	case !mine.Answerable || !theirs.Answerable:
		return false, "It holds no answer."
	case list(mine) || list(theirs):
		return false, "Repeating lists can't be linked with “Same as…”: insert them from the library instead."
	case oneOff[mine.Type] || oneOff[theirs.Type]:
		return false, "Signatures, file uploads and the BMI block are never pre-filled."
	case mine.Kind != theirs.Kind:
		return false, fmt.Sprintf("Its answer is %s; this question's is %s.", kindName(theirs.Kind), kindName(mine.Kind))
	case mine.Kind == "single" || mine.Kind == "multiple":
		return true, ""
	case mine.Unit != theirs.Unit:
		return false, fmt.Sprintf("Its unit is %s; this question's is %s.", quoted(theirs.Unit), quoted(mine.Unit))
	case mine.Format != theirs.Format:
		return false, fmt.Sprintf("Its %s is %s; this question's is %s.", formatNoun(mine.Kind), theirs.Format, mine.Format)
	}
	if c := answerChanges(theirs, mine); len(c) > 0 {
		return false, "Its answers differ from this question's: " + c[0] + "."
	}
	return true, ""
}

// Source is where "Same as…" finds keys: one of the trust's published questionnaires (each key's
// question from its newest version holding it) or the library.
type Source struct {
	Name      string
	Library   bool
	Questions []Placed
}

// Candidate is a key "Same as…" offers (S24): the question as the trust asks it, where it's asked,
// and whether this question may take its key (Compatible, and not already used in this HQ). Element
// is the JSON adopting copies the key and options from: the library's, else the first source's.
type Candidate struct {
	Key        string   `json:"key"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind"`
	Options    []string `json:"options"`
	Sources    []string `json:"sources"`
	Library    bool     `json:"library"`
	Compatible bool     `json:"compatible"`
	Reason     string   `json:"reason"`
	InThisHQ   bool     `json:"inThisHQ"`
	// AlwaysFresh: one of its questions is "Always ask fresh", so the question that adopts the key
	// becomes so too (Rahul, 23 Oct).
	AlwaysFresh bool            `json:"alwaysFresh"`
	Element     json.RawMessage `json:"element"`
}

// LinkedTo is the text of the question a key already links this one to (S29; Rahul, 25 Oct), for
// "Same as…" to warn before replacing it: the library's question if the key is there, else the first
// source's (the trust's newest published version holding it), as a candidate's title is chosen.
// "" when no published or library question takes part in pre-fill with that key (a new question).
func LinkedTo(name string, sources []Source) string {
	title := ""
	for _, s := range sources {
		for _, p := range s.Questions {
			if p.Name != name || !TakesPart(p) {
				continue
			}
			if s.Library {
				return p.Label()
			}
			if title == "" {
				title = p.Label()
			}
		}
	}
	return title
}

// maxCandidates is the most "Same as…" lists at once; the author narrows the search for others.
const maxCandidates = 50

// Candidates lists the keys question `mine` could take, from the sources, whose text (Normalize) or
// key contains the search ("" lists all): one per key, adoptable ones first, then by text. Only a
// patient's top-level answers take part (as pre-fill, TakesPart), and never mine's own key. `here`
// is the questionnaire being edited, as saved: a key it already uses can't be adopted again.
func Candidates(mine Question, here []Placed, sources []Source, search string) []Candidate {
	setOf := map[string]string{}
	for _, h := range here {
		if _, ok := setOf[h.Name]; !ok {
			setOf[h.Name] = h.Set
		}
	}
	search = strings.TrimSpace(search)
	type found struct {
		c         Candidate
		questions []Question
		matches   bool // the key, or one of its questions' texts, matches the search
	}
	byKey := map[string]*found{}
	var order []string
	for _, s := range sources {
		for _, p := range s.Questions {
			if !TakesPart(p) || p.Name == "" || p.Name == mine.Name {
				continue
			}
			f := byKey[p.Name]
			if f == nil {
				f = &found{c: Candidate{Key: p.Name, Title: p.Label(), Element: p.Raw}}
				f.matches = search == "" || strings.Contains(strings.ToLower(p.Name), strings.ToLower(search))
				byKey[p.Name] = f
				order = append(order, p.Name)
			}
			// Every copy counts for the search, so none is skipped (verification 1, finding 3).
			f.matches = f.matches || strings.Contains(Normalize(p.Title), Normalize(search))
			f.c.AlwaysFresh = f.c.AlwaysFresh || p.AlwaysFresh
			if !slices.Contains(f.c.Sources, s.Name) {
				f.c.Sources = append(f.c.Sources, s.Name)
			}
			if s.Library && !f.c.Library {
				f.c.Library, f.c.Title, f.c.Element = true, p.Label(), p.Raw
			}
			f.questions = append(f.questions, p.Question)
		}
	}
	out := make([]Candidate, 0, len(order))
	for _, k := range order {
		f := byKey[k]
		if !f.matches {
			continue
		}
		c := f.c
		shown := f.questions[0]
		c.Kind = kindName(shown.Kind)
		c.Options = []string{}
		for _, o := range shown.Options {
			c.Options = append(c.Options, o.Text)
		}
		published := make([]Published, len(f.questions))
		for i, q := range f.questions {
			published[i] = Published{Question: q}
		}
		set, inHere := setOf[k]
		switch {
		case len(publishedShapes(published)) > 1:
			c.Reason = "The trust has already published this question with different answers, so it can't be linked."
		case inHere:
			c.InThisHQ = true
			c.Reason = fmt.Sprintf("This HQ already has a question with this ID, in “%s”.", set)
		default:
			c.Compatible, c.Reason = Compatible(mine, shown)
		}
		out = append(out, c)
	}
	slices.SortStableFunc(out, func(a, b Candidate) int {
		if a.Compatible != b.Compatible {
			if a.Compatible {
				return -1
			}
			return 1
		}
		return strings.Compare(Normalize(a.Title), Normalize(b.Title))
	})
	if len(out) > maxCandidates {
		out = out[:maxCandidates]
	}
	return out
}

// Element walks one element on its own (a library question, or the question "Same as…" is for),
// as Walk would on a page: itself first, unless it's a group, then whatever it holds.
func Element(element json.RawMessage) ([]Question, error) {
	var el obj
	if err := json.Unmarshal(element, &el); err != nil || el == nil {
		return nil, ErrBadContent
	}
	var out []Question
	walk([]json.RawMessage{element}, false, "", &out)
	return out, nil
}
