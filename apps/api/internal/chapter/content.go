// Package chapter holds the rules for saving a chapter's SurveyJS JSON.
// It reads the JSON only; it never evaluates SurveyJS expressions.
package chapter

import (
	"encoding/json"
	"errors"
)

// MaxContentBytes caps one chapter's JSON. NFR-08's largest chapter
// (60 pages, 600 questions) is well under this.
const MaxContentBytes = 5 << 20

// ErrBadContent is returned when the content is not a SurveyJS survey object.
var ErrBadContent = errors.New("chapter content is not a valid form definition")

// ValidateContent checks the basic shape: a JSON object whose optional
// "pages" is an array of objects.
func ValidateContent(raw json.RawMessage) error {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
		return ErrBadContent
	}
	if pages, ok := doc["pages"]; ok {
		var list []map[string]json.RawMessage
		if err := json.Unmarshal(pages, &list); err != nil {
			return ErrBadContent
		}
		for _, p := range list {
			if p == nil {
				return ErrBadContent
			}
		}
	}
	return nil
}
