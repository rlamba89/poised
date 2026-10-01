package chapter

import (
	"encoding/json"
	"strings"
)

// ForPatient returns the chapter JSON as a patient may receive it (VEW-01, plan 10.2): without
// clinician-only questions and groups at any depth, without disclosures (clinicalOutputs) and
// without test cases. It also returns the names of the elements left, which are the only
// answer keys a patient may save. It walks the JSON only; it never evaluates expressions.
func ForPatient(raw json.RawMessage) (json.RawMessage, map[string]bool, error) {
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, ErrBadContent
	}
	names := map[string]bool{}
	doc = strip(doc, names)
	out, err := json.Marshal(doc)
	return out, names, err
}

// elementLists are the keys whose arrays hold pages, questions or groups.
var elementLists = map[string]bool{"pages": true, "elements": true, "templateElements": true}

func strip(v any, names map[string]bool) any {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "clinicalOutputs")
		delete(x, "testCases")
		for k, child := range x {
			list, ok := child.([]any)
			if !elementLists[k] || !ok {
				x[k] = strip(child, names)
				continue
			}
			kept := []any{}
			for _, item := range list {
				el, isObj := item.(map[string]any)
				if isObj && el["clinicianOnly"] == true {
					continue
				}
				if name, _ := el["name"].(string); isObj && k != "pages" && name != "" {
					names[name] = true
				}
				kept = append(kept, strip(item, names))
			}
			x[k] = kept
		}
		return x
	case []any:
		for i, item := range x {
			x[i] = strip(item, names)
		}
		return x
	}
	return v
}

// FilterAnswers keeps only the answers to the named questions, including SurveyJS's
// "<name>-Comment" entries for "Other (please specify)". It reports whether anything was dropped.
func FilterAnswers(data map[string]json.RawMessage, names map[string]bool) (map[string]json.RawMessage, bool) {
	kept := make(map[string]json.RawMessage, len(data))
	for k, v := range data {
		if names[k] || names[strings.TrimSuffix(k, "-Comment")] {
			kept[k] = v
		}
	}
	return kept, len(kept) != len(data)
}
