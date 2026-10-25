package chapter

import (
	"bytes"
	"encoding/json"
)

// VideoType is the Video element's type (S34): a YouTube or Vimeo player with an optional
// "I have watched this video" tick, which can be required. The tick's answer is `true`.
const VideoType = "video"

// Tick is a required "I have watched this video" tick: its question's name and title.
type Tick struct {
	Name  string
	Title string
}

// conditions are the properties that make an element, page or set depend on answers.
var conditions = []string{"visibleIf", "enableIf", "requiredIf", "chapterVisibleIf"}

// RequiredTicks lists the required video ticks that every patient sees in a chapter: not
// clinician-only, and with no condition on the video, any group or page around it, or the whole
// Question Set. Go never evaluates conditions, so a tick behind one is left to the browser's check
// (plan 2.1); these are the ones the server can hold a submit back for.
func RequiredTicks(raw json.RawMessage) ([]Tick, error) {
	patient, _, err := ForPatient(raw)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(patient, &doc); err != nil || doc == nil {
		return nil, ErrBadContent
	}
	if conditional(doc) {
		return nil, nil
	}
	var out []Tick
	var walk func(els []any)
	walk = func(els []any) {
		for _, item := range els {
			el, ok := item.(map[string]any)
			if !ok || conditional(el) {
				continue
			}
			if el["type"] == VideoType && el["watchTick"] == true && el["isRequired"] == true {
				name, _ := el["name"].(string)
				out = append(out, Tick{Name: name, Title: textOf(el["title"])})
			}
			if el["type"] == "panel" { // a repeating group's rows are its own answer: not walked
				children, _ := el["elements"].([]any)
				walk(children)
			}
		}
	}
	pages, _ := doc["pages"].([]any)
	for _, p := range pages {
		if page, ok := p.(map[string]any); ok && !conditional(page) {
			els, _ := page["elements"].([]any)
			walk(els)
		}
	}
	return out, nil
}

// Unticked are the ticks whose answer isn't `true`.
func Unticked(ticks []Tick, answers map[string]json.RawMessage) []Tick {
	var out []Tick
	for _, t := range ticks {
		if !bytes.Equal(bytes.TrimSpace(answers[t.Name]), []byte("true")) {
			out = append(out, t)
		}
	}
	return out
}

func conditional(x map[string]any) bool {
	for _, k := range conditions {
		if s, _ := x[k].(string); s != "" {
			return true
		}
	}
	return false
}

// textOf is a SurveyJS string in the default language: plain, or {"default": …} once translated.
func textOf(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		s, _ := t["default"].(string)
		return s
	}
	return ""
}
