package lifebox

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)

// Questionnaire is the converted result, ready to store.
type Questionnaire struct {
	Name        string
	Description string
	Sets        []Set
}

// Set is one Question Set: a chapter with its SurveyJS JSON.
type Set struct {
	Name        string
	Description string
	Icon        string
	Content     map[string]any
}

// Library looks up our code library and note categories, so outputs carry their names.
type Library interface {
	CodeDisplay(codeSet, code string) (string, bool)
	CategoryName(id string) (string, bool)
}

type option struct {
	question string // our question name
	value    string // our option value
	many     bool   // Select Many: tested with contains
	set      int    // index of the Question Set it is in
}

type converter struct {
	lib      Library
	cats     map[string]string // export category id → name
	options  map[string]option // Lifebox option id → ours
	taken    map[string]bool
	warnings []string
	set      int
	setName  string
}

// Convert maps a Lifebox export to our model. Warnings list what could not be carried over.
func Convert(e Export, lib Library) (Questionnaire, []string) {
	c := &converter{lib: lib, cats: map[string]string{}, options: map[string]option{}, taken: map[string]bool{}}
	for _, cat := range e.Categories {
		c.cats[cat.ID] = cat.Name
	}
	q := Questionnaire{Name: e.Questionnaire.Name, Description: e.Questionnaire.Description}

	// First give every question and option our ID, so conditions can point forwards and across pages.
	names := map[string]string{}
	for i, s := range e.Questionnaire.QuestionSets {
		for _, p := range s.Pages {
			c.index(i, p.Elements, names)
		}
	}
	for i, s := range e.Questionnaire.QuestionSets {
		c.set, c.setName = i, s.Name
		var pages []any
		for pi, p := range s.Pages {
			pages = append(pages, c.page(p, pi, names))
		}
		q.Sets = append(q.Sets, Set{
			Name:        s.Name,
			Description: s.Description,
			Icon:        icon(s.Icon),
			Content:     map[string]any{"pages": pages},
		})
	}
	return q, c.warnings
}

func (c *converter) warn(format string, args ...any) {
	c.warnings = append(c.warnings, fmt.Sprintf("%s: ", c.setName)+fmt.Sprintf(format, args...))
}

func (c *converter) index(set int, nodes []Node, names map[string]string) {
	for _, n := range nodes {
		switch n.Typename {
		case "QuestionGroupSection":
			names[n.ID] = c.newID("g", 6)
			c.index(set, n.Elements, names)
		case "ClinicalPageSummary":
		default:
			names[n.ID] = c.newID("q", 6)
		}
		used := map[string]bool{}
		switch n.Typename {
		case "QuestionBoolean":
			c.options[n.OptionTrueID] = option{question: names[n.ID], value: newID("o", 4, used), set: set}
			c.options[n.OptionFalseID] = option{question: names[n.ID], value: newID("o", 4, used), set: set}
		case "QuestionRadio", "QuestionCheckbox":
			for _, o := range n.QuestionOptions {
				c.options[o.ID] = option{question: names[n.ID], value: newID("o", 4, used), many: n.Typename == "QuestionCheckbox", set: set}
			}
		}
	}
}

func (c *converter) page(p Page, index int, names map[string]string) map[string]any {
	title := p.Name.String()
	if title == "" {
		title = fmt.Sprintf("Page %d", index+1)
	}
	page := map[string]any{"name": c.newID("p", 6), "title": title}
	if expr := c.expression(deref(p.RenderIf), "page "+title); expr != "" {
		page["visibleIf"] = expr
	}
	var elements []any
	for _, n := range p.Elements {
		if n.Typename == "ClinicalPageSummary" {
			page["clinicalSummary"] = true
			continue
		}
		if el := c.element(n, names); el != nil {
			elements = append(elements, el)
		}
	}
	page["elements"] = orEmpty(elements)
	return page
}

func (c *converter) element(n Node, names map[string]string) map[string]any {
	el := map[string]any{"name": names[n.ID]}
	title := n.Name.String()
	setText := func() {
		if title != "" {
			el["title"] = title
		}
		if d := n.Description.String(); d != "" {
			el["description"] = d
		}
		if n.IsRequired {
			el["isRequired"] = true
		}
	}
	switch n.Typename {
	case "QuestionText":
		el["type"] = "text"
		if n.TextType == "MULTI_LINE" {
			el["type"] = "comment"
		}
		setText()
	case "QuestionDate":
		el["type"], el["inputType"], el["min"] = "text", "date", "1900-01-01"
		if n.DateType == "DAY_MONTH_YEAR_MULTIPLE" {
			// Several dates: a list with one date per row, as the editor's "Allow several dates" writes it.
			delete(el, "inputType")
			delete(el, "min")
			el["type"], el["dateList"], el["rowCount"], el["minRowCount"] = "matrixdynamic", true, 1, 1
			el["addRowText"], el["removeRowText"] = "Add another date", "Remove"
			el["columns"] = []any{map[string]any{"name": "date", "title": " ", "cellType": "text", "inputType": "date", "min": "1900-01-01"}}
		}
		setText()
	case "QuestionBoolean":
		yes, no := c.options[n.OptionTrueID], c.options[n.OptionFalseID]
		el["type"], el["yesNo"], el["colCount"] = "radiogroup", true, 0
		el["choices"] = []any{
			map[string]any{"value": yes.value, "text": orDefault(n.LabelTrue.String(), "Yes")},
			map[string]any{"value": no.value, "text": orDefault(n.LabelFalse.String(), "No")},
		}
		setText()
	case "QuestionRadio", "QuestionCheckbox":
		el["type"] = "radiogroup"
		if n.Typename == "QuestionCheckbox" {
			el["type"] = "checkbox"
		}
		var choices []any
		for _, o := range n.QuestionOptions {
			ch := map[string]any{"value": c.options[o.ID].value, "text": o.Label.String()}
			if o.Typename == "QuestionOptionNone" {
				ch["isExclusive"] = true
			}
			choices = append(choices, ch)
		}
		el["choices"] = orEmpty(choices)
		setText()
	case "QuestionMedication":
		el["type"], el["medicationType"] = "medication", strings.ToLower(orDefault(n.MedicationType, "PRESCRIBED"))
		setText()
	case "QuestionHospitalAdmission":
		el["type"] = "admissions"
		setText()
	case "QuestionBMI":
		el["type"] = "bmi"
		setText()
	case "QuestionDemographic":
		el["type"] = "profile"
		setText()
	case "PanelStatement":
		el["type"], el["html"] = "html", textToHTML(n.Text.String())
	case "QuestionGroupSection":
		el["type"] = "panel"
		setText()
		if deref(n.GroupSectionType) == "PATIENT_PAGE" {
			el["patientPage"] = true
			if n.ShowGroupHeading != nil && !*n.ShowGroupHeading {
				el["showAsHeading"] = false
			}
		}
		var children []any
		for _, child := range n.Elements {
			if child.Typename == "ClinicalPageSummary" {
				continue
			}
			if ch := c.element(child, names); ch != nil {
				children = append(children, ch)
			}
		}
		el["elements"] = orEmpty(children)
	default:
		c.warn("skipped an element of unknown type %q", n.Typename)
		return nil
	}
	if n.IsClinical {
		el["clinicianOnly"] = true
	}
	label := title
	if label == "" {
		label = truncate(n.Text.String(), 50)
	}
	// Lifebox has renderIf and visibleIf on questions; both must hold.
	conds := []string{}
	for _, raw := range []string{deref(n.RenderIf), deref(n.VisibleIf)} {
		if expr := c.expression(raw, fmt.Sprintf("%q", label)); expr != "" {
			conds = append(conds, expr)
		}
	}
	switch len(conds) {
	case 1:
		el["visibleIf"] = conds[0]
	case 2:
		el["visibleIf"] = "(" + conds[0] + ") and (" + conds[1] + ")"
	}
	c.rules(n, el, label)
	return el
}

// rules turns Lifebox disclosures into clinical outputs on the option, or on the question.
func (c *converter) rules(n Node, el map[string]any, label string) {
	for _, r := range n.Rules {
		out := c.output(r, label)
		if out == nil {
			continue
		}
		expr := strings.TrimSpace(r.Expression)
		if expr == "true" || expr == "" {
			appendOutput(el, out)
			continue
		}
		m := hasSelectedOption.FindStringSubmatch(expr)
		if m == nil {
			c.warn("%q: disclosure with an expression we can't map (%s) was skipped", label, expr)
			continue
		}
		opt, ok := c.options[m[1]]
		if !ok || opt.question != el["name"] {
			c.warn("%q: disclosure on an option of another question was skipped", label)
			continue
		}
		for _, ch := range asList(el["choices"]) {
			if ch["value"] == opt.value {
				appendOutput(ch, out)
			}
		}
	}
}

var hasSelectedOption = regexp.MustCompile(`^hasSelectedOption\(\s*\{([^}]+)\}\s*\)$`)

func (c *converter) output(r Rule, label string) map[string]any {
	var codes []any
	for _, code := range r.DisclosureCodes {
		set := strings.ToUpper(code.CodeSet)
		display, ok := c.lib.CodeDisplay(set, code.Code)
		if !ok {
			display = code.Synonym
			c.warn("%q: code %s %s is not in our library; kept with its Lifebox name", label, set, code.Code)
		}
		codes = append(codes, map[string]any{"set": set, "code": code.Code, "display": display})
	}
	out := map[string]any{"id": newID("out", 6, nil), "codes": orEmpty(codes)}
	if text := strings.TrimSpace(deref(r.EpisodeNoteTemplate)); text != "" {
		cat := "Unassigned"
		if id := deref(r.EpisodeNoteCategoryID); id != "" {
			if name, ok := c.lib.CategoryName(id); ok {
				cat = name
			} else if name, ok := c.cats[id]; ok {
				cat = name
			}
		}
		out["note"] = map[string]any{"text": strings.ReplaceAll(text, "%s", "{answer}"), "category": cat}
	}
	if len(codes) == 0 && out["note"] == nil {
		return nil
	}
	return out
}

func appendOutput(target map[string]any, out map[string]any) {
	list, _ := target["clinicalOutputs"].([]any)
	target["clinicalOutputs"] = append(list, out)
}

// ---------------------------------------------------------------- expressions

var (
	containsAnyRe = regexp.MustCompile(`containsAny\(\s*selectedOptionIds\s*,\s*\[([^\]]*)\]\s*\)`)
	containsRe    = regexp.MustCompile(`contains\(\s*selectedOptionIds\s*,\s*\{([^}]+)\}\s*\)`)
	idRe          = regexp.MustCompile(`\{([^}]+)\}`)
	simpleRe      = regexp.MustCompile(`^\{[a-z]_[a-z]+\} (=|contains) '[a-z_]+'$`)
)

// expression translates a Lifebox condition into SurveyJS. It returns "" for none, and
// drops (with a warning) a condition that refers to something we can't find.
func (c *converter) expression(raw, what string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "true" {
		return ""
	}
	failed := false
	conds := func(ids []string) string {
		var parts []string
		for _, id := range ids {
			opt, ok := c.options[id]
			if !ok {
				c.warn("%s: its condition refers to an option that isn't in the export; condition dropped", what)
				failed = true
				return ""
			}
			if opt.set != c.set {
				c.warn("%s: its condition refers to a question in another Question Set; condition dropped", what)
				failed = true
				return ""
			}
			op := "="
			if opt.many {
				op = "contains"
			}
			parts = append(parts, fmt.Sprintf("{%s} %s '%s'", opt.question, op, opt.value))
		}
		if len(parts) == 0 {
			return "(false)"
		}
		return "(" + strings.Join(parts, " or ") + ")"
	}
	out := containsAnyRe.ReplaceAllStringFunc(raw, func(m string) string {
		var ids []string
		for _, id := range idRe.FindAllStringSubmatch(containsAnyRe.FindStringSubmatch(m)[1], -1) {
			ids = append(ids, id[1])
		}
		return conds(ids)
	})
	out = containsRe.ReplaceAllStringFunc(out, func(m string) string {
		return conds([]string{containsRe.FindStringSubmatch(m)[1]})
	})
	if failed {
		return ""
	}
	if strings.Contains(out, "selectedOptionIds") || strings.Contains(out, "hasSelectedOption") {
		c.warn("%s: condition %q uses something we can't translate; condition dropped", what, raw)
		return ""
	}
	out = strings.NewReplacer("&&", " and ", "||", " or ").Replace(out)
	return simplify(strings.Join(strings.Fields(out), " "))
}

// simplify writes one condition the way our Builder reads it: "{q} = 'o'", "{q} <> 'o'",
// "{q} contains 'o'" or "{q} notcontains 'o'".
func simplify(expr string) string {
	neg := false
	if strings.HasPrefix(expr, "!") {
		neg, expr = true, strings.TrimSpace(expr[1:])
	}
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") && simpleRe.MatchString(expr[1:len(expr)-1]) {
		inner := expr[1 : len(expr)-1]
		if !neg {
			return inner
		}
		if strings.Contains(inner, " contains ") {
			return strings.Replace(inner, " contains ", " notcontains ", 1)
		}
		return strings.Replace(inner, " = ", " <> ", 1)
	}
	if neg {
		return "!" + expr
	}
	return expr
}

// ---------------------------------------------------------------- helpers

// icon maps a Lifebox icon name to one of our Tabler icon names.
func icon(name string) string {
	n := strings.ToLower(name)
	for _, m := range []struct{ has, icon string }{
		{"user", "user"}, {"medic", "pill"}, {"pill", "pill"}, {"heart", "heart"}, {"lung", "lungs"},
		{"brain", "brain"}, {"bone", "bone"}, {"stomach", "salad"}, {"digest", "salad"}, {"virus", "virus"},
		{"infect", "virus"}, {"home", "home"}, {"hospital", "building-hospital"}, {"stethoscope", "stethoscope"},
	} {
		if strings.Contains(n, m.has) {
			return m.icon
		}
	}
	return "file-text"
}

// htmlEscaper escapes only what HTML needs, as the editor does (html.EscapeString also encodes quotes).
var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// textToHTML matches the editor's textToHtml: one paragraph per line, escaped.
func textToHTML(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			b.WriteString("<p>&nbsp;</p>")
			continue
		}
		b.WriteString("<p>" + htmlEscaper.Replace(line) + "</p>")
	}
	return b.String()
}

func (c *converter) newID(prefix string, n int) string { return newID(prefix, n, c.taken) }

// newID makes an ID like q_kfbwpx (letters only, as the editor's IDs).
func newID(prefix string, n int, taken map[string]bool) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	for {
		b := make([]byte, n)
		_, _ = rand.Read(b)
		for i := range b {
			b[i] = letters[int(b[i])%len(letters)]
		}
		id := prefix + "_" + string(b)
		if taken == nil || !taken[id] {
			if taken != nil {
				taken[id] = true
			}
			return id
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func orEmpty(list []any) []any {
	if list == nil {
		return []any{}
	}
	return list
}

func asList(v any) []map[string]any {
	list, _ := v.([]any)
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
