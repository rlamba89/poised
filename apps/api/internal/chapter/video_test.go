package chapter

import (
	"encoding/json"
	"reflect"
	"testing"
)

const videos = `{
  "pages": [{
    "name": "p_1",
    "elements": [
      {"type": "video", "name": "q_must", "title": "Your anaesthetic", "watchTick": true, "isRequired": true},
      {"type": "video", "name": "q_optional", "title": "Fasting", "watchTick": true},
      {"type": "video", "name": "q_no_tick", "title": "Prehab", "isRequired": true},
      {"type": "video", "name": "q_shown_if", "title": "Smoking", "watchTick": true, "isRequired": true, "visibleIf": "{q_smoke} = 'o_y'"},
      {"type": "video", "name": "q_clin", "title": "For nurses", "watchTick": true, "isRequired": true, "clinicianOnly": true},
      {"type": "panel", "name": "g_1", "elements": [
        {"type": "video", "name": "q_in_group", "title": {"default": "Recovery", "de": "Erholung"}, "watchTick": true, "isRequired": true}
      ]},
      {"type": "panel", "name": "g_2", "visibleIf": "{q_x} = 'o_y'", "elements": [
        {"type": "video", "name": "q_in_hidden_group", "title": "Hidden", "watchTick": true, "isRequired": true}
      ]}
    ]
  }, {
    "name": "p_2", "visibleIf": "{q_x} = 'o_n'",
    "elements": [{"type": "video", "name": "q_on_hidden_page", "title": "Page", "watchTick": true, "isRequired": true}]
  }]
}`

// RequiredTicks lists the required ticks every patient sees: Go never evaluates conditions, so a
// tick behind any condition is left to the browser's check (S34).
func TestRequiredTicks(t *testing.T) {
	got, err := RequiredTicks(json.RawMessage(videos))
	if err != nil {
		t.Fatal(err)
	}
	want := []Tick{{Name: "q_must", Title: "Your anaesthetic"}, {Name: "q_in_group", Title: "Recovery"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RequiredTicks = %+v, want %+v", got, want)
	}
}

func TestRequiredTicksWholeSetCondition(t *testing.T) {
	got, err := RequiredTicks(json.RawMessage(`{"chapterVisibleIf": "{age} > 60", "pages": [{"name": "p", "elements": [{"type": "video", "name": "q_v", "watchTick": true, "isRequired": true}]}]}`))
	if err != nil || len(got) != 0 {
		t.Errorf("RequiredTicks = %v, %v; want none: the whole set has a condition", got, err)
	}
}

func TestUnticked(t *testing.T) {
	ticks := []Tick{{Name: "q_a", Title: "A"}, {Name: "q_b", Title: "B"}, {Name: "q_c", Title: "C"}}
	answers := map[string]json.RawMessage{"q_a": json.RawMessage(`true`), "q_b": json.RawMessage(`false`)}
	if got := Unticked(ticks, answers); !reflect.DeepEqual(got, []Tick{{Name: "q_b", Title: "B"}, {Name: "q_c", Title: "C"}}) {
		t.Errorf("Unticked = %v", got)
	}
	if _, err := RequiredTicks(json.RawMessage(`[]`)); err != ErrBadContent {
		t.Errorf("bad content: %v", err)
	}
}
