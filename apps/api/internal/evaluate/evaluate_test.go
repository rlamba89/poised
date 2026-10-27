package evaluate_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

// The shared fixtures: packages/clinical's evaluate.test.ts and apps/evaluate's tests read the same
// files, so Go's types and the TypeScript handler stay in step (as content's video_links.json does).
type sentFixture struct {
	Input  json.RawMessage `json:"input"`
	Output json.RawMessage `json:"output"`
}

type publishFixture struct {
	Cases []struct {
		Name   string          `json:"name"`
		Input  json.RawMessage `json:"input"`
		Output json.RawMessage `json:"output"`
	} `json:"cases"`
}

func read(t *testing.T, name string, v any) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}

// sameJSON compares two JSON documents by value.
func sameJSON(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(x, y)
}

func TestInputMatchesTheFixture(t *testing.T) {
	var fx sentFixture
	read(t, "evaluate.json", &fx)
	// Every field the TypeScript side reads survives Go's types: what Go sends is the fixture's input.
	var in evaluate.Input
	if err := json.Unmarshal(fx.Input, &in); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if !sameJSON(t, out, fx.Input) {
		t.Errorf("Go's Input doesn't round-trip the fixture:\n got %s\nwant %s", out, fx.Input)
	}
	if len(in.Sets) != 4 || in.Sets[3].Audience != "clinician" || in.Patient.Age != 54 || in.ProcedureID != nil {
		t.Errorf("input = %+v", in)
	}
}

// fixtureServer answers like the evaluate service: the fixture's output for /evaluate.
func fixtureServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestHTTPEvaluate(t *testing.T) {
	var fx sentFixture
	read(t, "evaluate.json", &fx)
	var in evaluate.Input
	if err := json.Unmarshal(fx.Input, &in); err != nil {
		t.Fatal(err)
	}
	var got []byte
	url := fixtureServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/evaluate" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request = %s %s (%s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
		}
		got, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fx.Output)
	})
	res, err := evaluate.HTTP{URL: url}.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !sameJSON(t, got, fx.Input) {
		t.Errorf("sent %s", got)
	}
	// The result as the service gave it, kept whole for form_outputs.
	if !sameJSON(t, res.Raw, fx.Output) {
		t.Errorf("Raw = %s", res.Raw)
	}
	about, sleep, nurse := in.Sets[0].ID, in.Sets[1].ID, in.Sets[3].ID
	if res.Evaluator != "test" {
		t.Errorf("Evaluator = %q", res.Evaluator)
	}
	if !reflect.DeepEqual(res.Shown, []uuid.UUID{about, sleep, nurse}) {
		t.Errorf("Shown = %v", res.Shown)
	}
	if res.SuggestedASA == nil || res.SuggestedASA.Grade != "III" || res.SuggestedASA.Emergency {
		t.Errorf("SuggestedASA = %+v", res.SuggestedASA)
	}
	if len(res.Flags) != 2 || res.Flags[0] != (evaluate.Flag{Level: "red", SetID: about, QuestionID: "q_chest", Text: "Do you get chest pain?: Yes"}) {
		t.Errorf("Flags = %+v", res.Flags)
	}
	if len(res.Rules) != 2 || res.Rules[0].ID != "high_risk" || res.Rules[0].Name != "High risk" || res.Rules[0].Held || len(res.Rules[0].Actions) != 3 ||
		!strings.HasPrefix(res.Rules[0].Line, "Rule fires: High risk (") {
		t.Errorf("Rules = %+v", res.Rules)
	}
	var first struct{ Kind, Title, Team string }
	if err := json.Unmarshal(res.Rules[0].Actions[0], &first); err != nil || first.Kind != "task" || first.Title != "Book anaesthetic review" || first.Team != "anaesthetist" {
		t.Errorf("first action = %+v (%v)", first, err)
	}
	want := []evaluate.PatientSet{{ID: about, Complete: true}, {ID: sleep, Complete: false}}
	if !reflect.DeepEqual(res.PatientSets, want) {
		t.Errorf("PatientSets = %+v", res.PatientSets)
	}
	// S42: each set's scores, which auto-triage reads (an Incomplete one stops it).
	if len(res.Sets) != 3 || res.Sets[0].ID != about || len(res.Sets[0].Scores) != 1 ||
		res.Sets[0].Scores[0] != (evaluate.Score{QuestionID: "q_score", Incomplete: false}) {
		t.Errorf("Sets = %+v", res.Sets)
	}
}

// S44: Complete review asks which sets are shown for the answers that count now.
func TestHTTPShown(t *testing.T) {
	var fx sentFixture
	read(t, "shown.json", &fx)
	var in evaluate.ShownInput
	if err := json.Unmarshal(fx.Input, &in); err != nil {
		t.Fatal(err)
	}
	if out, _ := json.Marshal(in); !sameJSON(t, out, fx.Input) {
		t.Errorf("Go's ShownInput doesn't round-trip the fixture:\n got %s\nwant %s", out, fx.Input)
	}
	var got []byte
	url := fixtureServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/shown" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request = %s %s (%s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
		}
		got, _ = io.ReadAll(r.Body)
		w.Write(fx.Output)
	})
	shown, err := evaluate.HTTP{URL: url}.Shown(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !sameJSON(t, got, fx.Input) {
		t.Errorf("sent %s", got)
	}
	if want := []uuid.UUID{in.Sets[0].ID, in.Sets[2].ID, in.Sets[3].ID}; !reflect.DeepEqual(shown, want) {
		t.Errorf("shown = %v, want %v", shown, want)
	}
}

func TestHTTPPublishProblems(t *testing.T) {
	var fx publishFixture
	read(t, "publish_problems.json", &fx)
	for _, c := range fx.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var in evaluate.PublishInput
			if err := json.Unmarshal(c.Input, &in); err != nil {
				t.Fatal(err)
			}
			if out, _ := json.Marshal(in); !sameJSON(t, out, c.Input) {
				t.Errorf("Go's PublishInput doesn't round-trip the fixture: %s", out)
			}
			url := fixtureServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/publish-problems" {
					t.Errorf("path = %s", r.URL.Path)
				}
				w.Write(c.Output)
			})
			got, err := evaluate.HTTP{URL: url}.PublishProblems(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			var want struct{ Problems []evaluate.Problem }
			if err := json.Unmarshal(c.Output, &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want.Problems) {
				t.Errorf("problems = %+v, want %+v", got, want.Problems)
			}
		})
	}
}

// Each way the call can fail is an error, and none holds an answer (the error is logged and stored).
func TestHTTPFailures(t *testing.T) {
	in := evaluate.Input{FormID: uuid.New(), Answers: map[uuid.UUID]json.RawMessage{uuid.New(): json.RawMessage(`{"q_secret":"o_y"}`)}}
	cases := map[string]http.HandlerFunc{
		"a server error": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"q_secret"}`, http.StatusInternalServerError)
		},
		"a reply that isn't JSON": func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("q_secret")) },
		"a reply with no evaluator": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"shown":[],"q_secret":1}`))
		},
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := evaluate.HTTP{URL: fixtureServer(t, h)}.Evaluate(context.Background(), in)
			if err == nil {
				t.Fatal("no error")
			}
			if strings.Contains(err.Error(), "q_secret") {
				t.Errorf("the error holds the reply: %v", err)
			}
		})
	}
	t.Run("nothing listening", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		if _, err := (evaluate.HTTP{URL: url}).Evaluate(context.Background(), in); err == nil {
			t.Fatal("no error")
		}
	})
	t.Run("a timeout", func(t *testing.T) {
		url := fixtureServer(t, func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body) // so the server notices the client hang up
			select {
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
			}
		})
		start := time.Now()
		_, err := evaluate.HTTP{URL: url, Timeout: 50 * time.Millisecond}.Evaluate(context.Background(), in)
		if err == nil || time.Since(start) > 2*time.Second {
			t.Fatalf("err = %v after %v", err, time.Since(start))
		}
	})
	t.Run("shown: a reply with no list, and a server error", func(t *testing.T) {
		for _, h := range []http.HandlerFunc{
			func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"q_secret":1}`)) },
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "q_secret", http.StatusInternalServerError)
			},
		} {
			_, err := evaluate.HTTP{URL: fixtureServer(t, h)}.Shown(context.Background(), evaluate.ShownInput{FormID: uuid.New()})
			if err == nil || strings.Contains(err.Error(), "q_secret") {
				t.Errorf("err = %v", err)
			}
		}
	})
	t.Run("publish checks failing", func(t *testing.T) {
		url := fixtureServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) })
		if _, err := (evaluate.HTTP{URL: url}).PublishProblems(context.Background(), evaluate.PublishInput{}); err == nil {
			t.Fatal("no error")
		}
	})
}

func TestRecorder(t *testing.T) {
	r := &evaluate.Recorder{}
	set := uuid.New()
	in := evaluate.Input{FormID: uuid.New(), Sets: []evaluate.Set{{ID: set, Name: "About you", Audience: "patient", Content: json.RawMessage(`{}`)}}}
	res, err := r.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	// By default it answers like the service for an HQ that gives nothing: every set shown.
	if res.Evaluator != evaluate.FakeVersion || !reflect.DeepEqual(res.Shown, []uuid.UUID{set}) || len(res.Raw) == 0 {
		t.Errorf("result = %+v", res)
	}
	if got := r.Inputs(); len(got) != 1 || got[0].FormID != in.FormID {
		t.Errorf("Inputs = %+v", got)
	}
	r.Fail(errors.New("down"))
	if _, err := r.Evaluate(context.Background(), in); err == nil {
		t.Error("Fail: no error")
	}
	if _, err := r.PublishProblems(context.Background(), evaluate.PublishInput{}); err == nil {
		t.Error("Fail: publish checks didn't fail")
	}
	r.Fail(nil)
	r.SetProblems(evaluate.Problem{Set: "Rules", Message: "Test case “X” fails"})
	if p, err := r.PublishProblems(context.Background(), evaluate.PublishInput{}); err != nil || len(p) != 1 {
		t.Errorf("problems = %v, %v", p, err)
	}
	if len(r.Published()) != 2 {
		t.Errorf("Published = %d", len(r.Published()))
	}

	// S44: Shown gives every set unless told some are hidden; Fail fails it too.
	other := uuid.New()
	review := evaluate.ShownInput{FormID: in.FormID, Sets: append(in.Sets, evaluate.Set{ID: other, Name: "Nurse", Audience: "clinician", Content: json.RawMessage(`{}`)})}
	if got, err := r.Shown(context.Background(), review); err != nil || !reflect.DeepEqual(got, []uuid.UUID{set, other}) {
		t.Errorf("Shown = %v, %v", got, err)
	}
	r.Hide(set)
	if got, _ := r.Shown(context.Background(), review); !reflect.DeepEqual(got, []uuid.UUID{other}) {
		t.Errorf("Shown after Hide = %v", got)
	}
	if got := r.ShownInputs(); len(got) != 2 || got[0].FormID != in.FormID {
		t.Errorf("ShownInputs = %+v", got)
	}
	r.Fail(errors.New("down"))
	if _, err := r.Shown(context.Background(), review); err == nil {
		t.Error("Fail: Shown didn't fail")
	}
}
