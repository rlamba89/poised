package chapter

import (
	"encoding/json"
	"testing"
)

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name string
		json string
		ok   bool
	}{
		{"empty survey", `{}`, true},
		{"pages of objects", `{"pages":[{"name":"page1","elements":[]}]}`, true},
		{"not an object", `[]`, false},
		{"null", `null`, false},
		{"string", `"x"`, false},
		{"pages not an array", `{"pages":{}}`, false},
		{"page not an object", `{"pages":[1]}`, false},
		{"null page", `{"pages":[null]}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContent(json.RawMessage(tt.json))
			if (err == nil) != tt.ok {
				t.Errorf("ValidateContent(%s) = %v, want ok=%v", tt.json, err, tt.ok)
			}
		})
	}
}
