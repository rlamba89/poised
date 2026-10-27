// Package evaluate calls the evaluate service (plan C6, session S41; D-1): the one place the server
// runs SurveyJS. Go never evaluates SurveyJS expressions itself (CLAUDE.md's key rule); the service
// runs packages/clinical's evaluate() and checkPublish() (packages/clinical/src/evaluate.ts):
//   - Evaluate: when an HQ is sent, Go sends the published version, the patient's answers and the
//     patient context, and stores what comes back (form_outputs);
//   - PublishProblems: when a version is published, the browser's publish checks, so the API alone
//     can't publish a version whose test cases fail;
//   - Shown: on Complete review (S44), the sets shown for the answers that count now, so Go can
//     refuse while one of them isn't validated.
//
// Locally the service is `npm run evaluate` (apps/evaluate), which make dev starts, at EVALUATE_URL;
// S45 puts it in a Lambda. Tests use Recorder. The shared fixtures in testdata are read by both this
// package's tests and the TypeScript tests, so the two sides stay in step.
//
// Nothing here logs, and no error holds an answer or the service's reply.
package evaluate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Evaluator is the evaluate service. Deps.Evaluator in internal/httpapi.
type Evaluator interface {
	Evaluate(ctx context.Context, in Input) (Result, error)
	PublishProblems(ctx context.Context, in PublishInput) ([]Problem, error)
	Shown(ctx context.Context, in ShownInput) ([]uuid.UUID, error)
}

// Set is one Question Set of the published version, as stored (clinician-only parts too).
type Set struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	Audience string          `json:"audience"`
	Content  json.RawMessage `json:"content"`
}

// Patient is all the service is told about the patient: their age on the day they sent their
// answers (at the hospital), and their sex. No name or other details.
type Patient struct {
	Age int    `json:"age"`
	Sex string `json:"sex"`
}

// Input is a sent HQ.
type Input struct {
	FormID      uuid.UUID                     `json:"formId"` // the only thing the service's error log names
	Sets        []Set                         `json:"sets"`
	Rules       json.RawMessage               `json:"rules"`   // the version's ASA rules as stored (S40); null before S40
	Answers     map[uuid.UUID]json.RawMessage `json:"answers"` // the patient's answers, by set
	Patient     Patient                       `json:"patient"`
	ProcedureID *uuid.UUID                    `json:"procedureId"`
}

// ShownInput is an HQ under review (S44): every set, the answers that count now by set (the
// clinician's copy where there is one, else the patient's), and the patient as on the day they sent.
type ShownInput struct {
	FormID  uuid.UUID                     `json:"formId"` // the only thing the service's error log names
	Sets    []Set                         `json:"sets"`
	Answers map[uuid.UUID]json.RawMessage `json:"answers"`
	Patient Patient                       `json:"patient"`
}

// Result is what a sent HQ's answers give: the same result the clinician's view shows for them
// before any validation. Raw is the whole reply, stored as it came; the fields are what Go reads.
type Result struct {
	Evaluator    string          `json:"evaluator"` // which build worked it out
	Shown        []uuid.UUID     `json:"shown"`     // the Question Sets shown, in order
	Sets         []SetResult     `json:"sets"`      // each shown set's scores (S42: an Incomplete one stops auto-triage); its disclosures are in Raw
	SuggestedASA *SuggestedASA   `json:"suggestedAsa"`
	Flags        []Flag          `json:"flags"`       // every red and amber flag
	Rules        []FiredRule     `json:"rules"`       // the rules that fire, in order (S42, S43 act on them)
	PatientSets  []PatientSet    `json:"patientSets"` // the sets the patient's own form shows (S44)
	Raw          json.RawMessage `json:"-"`
}

// SetResult is one shown set's part of the result, as Go reads it.
type SetResult struct {
	ID     uuid.UUID `json:"id"`
	Scores []Score   `json:"scores"`
}

// Score is a Calculation's result; Incomplete while one of its inputs has no answer (S38).
type Score struct {
	QuestionID string `json:"questionId"`
	Incomplete bool   `json:"incomplete"`
}

// SuggestedASA is the suggested grade (I…VI), E for an emergency; its reasons are in Raw.
type SuggestedASA struct {
	Grade     string `json:"grade"`
	Emergency bool   `json:"emergency"`
}

// Flag is a disclosure with a review flag, in words.
type Flag struct {
	Level      string    `json:"level"` // "red" or "amber"
	SetID      uuid.UUID `json:"setId"`
	QuestionID string    `json:"questionId"`
	Text       string    `json:"text"`
}

// FiredRule is a rule that fires: its actions (as asarules stores them), held when its auto-triage
// was taken out (no ASA grade, and the HQ's setting says so), and the line test cases write.
type FiredRule struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Actions []json.RawMessage `json:"actions"`
	Held    bool              `json:"held"`
	// HeldReason says why its auto-triage was taken out (S42): noGrade (the HQ's setting), or one of
	// the guards every send applies: flag, incomplete, anaesthetistAsa.
	HeldReason string `json:"heldReason,omitempty"`
	Line       string `json:"line"`
}

// PatientSet is a set the patient's own form shows, and whether it's complete.
type PatientSet struct {
	ID       uuid.UUID `json:"id"`
	Complete bool      `json:"complete"`
}

// PublishSet is one Question Set of a version being published.
type PublishSet struct {
	Name     string          `json:"name"`
	Audience string          `json:"audience"` // the checks need it (S39: only a clinician set may test the suggested ASA)
	Content  json.RawMessage `json:"content"`
}

// PublishInput is a version's Question Sets, in order, and its ASA rules.
type PublishInput struct {
	Sets  []PublishSet    `json:"sets"`
	Rules json.RawMessage `json:"rules"`
}

// Problem blocks publishing, as the browser's Publish window lists it.
type Problem struct {
	Set     string `json:"set"`
	Message string `json:"message"`
}

// FailedMarker starts the log line written when a call fails (internal/httpapi), with the episode
// form's id or the version's and never an answer. S45's alarm counts these lines, as OPS-05's
// counts DELIVERY_FAILED.
const FailedMarker = "EVALUATE_FAILED"

// The calls' time limits: a patient waits on Evaluate, a clinician on Shown; PublishProblems runs
// every test case.
const (
	EvaluateTimeout = 5 * time.Second
	PublishTimeout  = 20 * time.Second
	maxReply        = 20 << 20
)

// HTTP calls the service at URL (EVALUATE_URL), e.g. http://127.0.0.1:8090.
type HTTP struct {
	URL     string
	Timeout time.Duration // 0: EvaluateTimeout or PublishTimeout
	Client  *http.Client  // nil: http.DefaultClient
}

func (h HTTP) Evaluate(ctx context.Context, in Input) (Result, error) {
	var res Result
	raw, err := h.post(ctx, "/evaluate", cmpDuration(h.Timeout, EvaluateTimeout), in)
	if err != nil {
		return res, err
	}
	return decodeResult(raw)
}

func (h HTTP) PublishProblems(ctx context.Context, in PublishInput) ([]Problem, error) {
	raw, err := h.post(ctx, "/publish-problems", cmpDuration(h.Timeout, PublishTimeout), in)
	if err != nil {
		return nil, err
	}
	var out struct {
		Problems *[]Problem `json:"problems"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Problems == nil {
		return nil, errors.New("evaluate: /publish-problems: the reply isn't a list of problems")
	}
	return *out.Problems, nil
}

// Shown is the sets shown, in order, for the answers that count now (S44).
func (h HTTP) Shown(ctx context.Context, in ShownInput) ([]uuid.UUID, error) {
	raw, err := h.post(ctx, "/shown", cmpDuration(h.Timeout, EvaluateTimeout), in)
	if err != nil {
		return nil, err
	}
	var out struct {
		Shown *[]uuid.UUID `json:"shown"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Shown == nil {
		return nil, errors.New("evaluate: /shown: the reply isn't a list of sets")
	}
	return *out.Shown, nil
}

// decodeResult checks the reply is a result (it names its evaluator) and keeps it whole.
func decodeResult(raw []byte) (Result, error) {
	var res Result
	if json.Unmarshal(raw, &res) != nil || res.Evaluator == "" {
		return Result{}, errors.New("evaluate: /evaluate: the reply isn't a result")
	}
	res.Raw = raw
	return res, nil
}

func cmpDuration(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

// post sends in as JSON and returns the 200 reply's body. Its errors never hold the body.
func (h HTTP) post(ctx context.Context, path string, timeout time.Duration, in any) ([]byte, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %s: %w", path, err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(h.URL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("evaluate: %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("evaluate: %s: status %d", path, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxReply+1))
	if err != nil {
		return nil, fmt.Errorf("evaluate: %s: reading the reply: %w", path, err)
	}
	if len(raw) > maxReply {
		return nil, fmt.Errorf("evaluate: %s: the reply is too big", path)
	}
	return raw, nil
}

// FakeVersion is the Recorder's evaluator version.
const FakeVersion = "fake"

// Recorder is the fake for tests: it records each call and answers as told. By default Evaluate
// answers like the service for an HQ that gives nothing (every set shown, no disclosures, no rule
// fires, every patient set complete), PublishProblems finds none and Shown gives every set.
type Recorder struct {
	mu        sync.Mutex
	inputs    []Input
	published []PublishInput
	reviewed  []ShownInput
	result    *Result
	problems  []Problem
	hidden    []uuid.UUID
	err       error
}

// Hide makes Shown leave these sets out, as if their conditions hid them (none: show all again).
func (r *Recorder) Hide(sets ...uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hidden = sets
}

// ShownInputs are the HQs sent to Shown, in order.
func (r *Recorder) ShownInputs() []ShownInput {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.reviewed)
}

func (r *Recorder) Shown(_ context.Context, in ShownInput) ([]uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reviewed = append(r.reviewed, in)
	if r.err != nil {
		return nil, r.err
	}
	shown := []uuid.UUID{}
	for _, s := range in.Sets {
		if !slices.Contains(r.hidden, s.ID) {
			shown = append(shown, s.ID)
		}
	}
	return shown, nil
}

// Fail makes every call fail with err (nil: succeed again).
func (r *Recorder) Fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

// Reply makes Evaluate answer res (its Raw is filled in from the fields when empty).
func (r *Recorder) Reply(res Result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.result = &res
}

// SetProblems makes PublishProblems find these.
func (r *Recorder) SetProblems(p ...Problem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.problems = p
}

// Inputs are the HQs sent to Evaluate, in order.
func (r *Recorder) Inputs() []Input {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.inputs)
}

// Published are the versions sent to PublishProblems, in order.
func (r *Recorder) Published() []PublishInput {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.published)
}

func (r *Recorder) Evaluate(_ context.Context, in Input) (Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputs = append(r.inputs, in)
	if r.err != nil {
		return Result{}, r.err
	}
	var res Result
	if r.result != nil {
		res = *r.result
	} else {
		res = Result{Evaluator: FakeVersion, Shown: []uuid.UUID{}, Flags: []Flag{}, Rules: []FiredRule{}, PatientSets: []PatientSet{}}
		for _, s := range in.Sets {
			res.Shown = append(res.Shown, s.ID)
			if s.Audience == "patient" {
				res.PatientSets = append(res.PatientSets, PatientSet{ID: s.ID, Complete: true})
			}
		}
	}
	if len(res.Raw) == 0 {
		raw, err := json.Marshal(res)
		if err != nil {
			return Result{}, err
		}
		res.Raw = raw
	}
	return res, nil
}

func (r *Recorder) PublishProblems(_ context.Context, in PublishInput) ([]Problem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.published = append(r.published, in)
	if r.err != nil {
		return nil, r.err
	}
	return slices.Clone(r.problems), nil
}
