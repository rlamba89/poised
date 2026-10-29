package evaluate_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/evaluate"
)

// fakeLambda stands in for the AWS SDK's Lambda client (S45): it records each invoke, and answers
// with reply's output, or blocks until the call's context ends when block is set.
type fakeLambda struct {
	mu       sync.Mutex
	calls    []*lambda.InvokeInput
	payloads []map[string]json.RawMessage
	deadline []time.Duration // how long each call's context had left when it arrived
	reply    func(op string, input json.RawMessage) (*lambda.InvokeOutput, error)
	block    bool
}

func (f *fakeLambda) Invoke(ctx context.Context, in *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	f.mu.Lock()
	f.calls = append(f.calls, in)
	var p map[string]json.RawMessage
	_ = json.Unmarshal(in.Payload, &p)
	f.payloads = append(f.payloads, p)
	if d, ok := ctx.Deadline(); ok {
		f.deadline = append(f.deadline, time.Until(d))
	}
	f.mu.Unlock()
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	var op string
	_ = json.Unmarshal(p["op"], &op)
	return f.reply(op, p["input"])
}

// answers replies like the evaluate Lambda (apps/evaluate/src/lambda.ts): {status, body}.
func answers(body json.RawMessage) func(string, json.RawMessage) (*lambda.InvokeOutput, error) {
	return func(string, json.RawMessage) (*lambda.InvokeOutput, error) {
		payload, err := json.Marshal(map[string]any{"status": 200, "body": body})
		return &lambda.InvokeOutput{StatusCode: 200, Payload: payload}, err
	}
}

func TestLambdaEvaluate(t *testing.T) {
	var fx sentFixture
	read(t, "evaluate.json", &fx)
	var in evaluate.Input
	if err := json.Unmarshal(fx.Input, &in); err != nil {
		t.Fatal(err)
	}
	fake := &fakeLambda{reply: answers(fx.Output)}
	res, err := evaluate.Lambda{Client: fake, Function: "poised-xx-qa-evaluate"}.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	// One synchronous invoke of the named function, its payload {op, input}.
	if len(fake.calls) != 1 {
		t.Fatalf("%d invokes", len(fake.calls))
	}
	call := fake.calls[0]
	if aws.ToString(call.FunctionName) != "poised-xx-qa-evaluate" || call.InvocationType != types.InvocationTypeRequestResponse {
		t.Errorf("invoke = %s, %s", aws.ToString(call.FunctionName), call.InvocationType)
	}
	if string(fake.payloads[0]["op"]) != `"evaluate"` || !sameJSON(t, fake.payloads[0]["input"], fx.Input) {
		t.Errorf("payload = %s", call.Payload)
	}
	// The reply's body is the result, as the HTTP service's is: kept whole for form_outputs.
	if !sameJSON(t, res.Raw, fx.Output) || res.Evaluator != "test" || res.SuggestedASA == nil || res.SuggestedASA.Grade != "III" || len(res.Rules) != 2 {
		t.Errorf("result = %+v", res)
	}
	// The patient's waiting: the same time limit as the HTTP call's.
	if d := fake.deadline[0]; d > evaluate.EvaluateTimeout || d < evaluate.EvaluateTimeout-time.Second {
		t.Errorf("deadline = %v, want about %v", d, evaluate.EvaluateTimeout)
	}
}

func TestLambdaShown(t *testing.T) {
	var fx sentFixture
	read(t, "shown.json", &fx)
	var in evaluate.ShownInput
	if err := json.Unmarshal(fx.Input, &in); err != nil {
		t.Fatal(err)
	}
	fake := &fakeLambda{reply: answers(fx.Output)}
	shown, err := evaluate.Lambda{Client: fake, Function: "f"}.Shown(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if string(fake.payloads[0]["op"]) != `"shown"` || !sameJSON(t, fake.payloads[0]["input"], fx.Input) {
		t.Errorf("payload = %s", fake.calls[0].Payload)
	}
	if want := []uuid.UUID{in.Sets[0].ID, in.Sets[2].ID, in.Sets[3].ID}; !reflect.DeepEqual(shown, want) {
		t.Errorf("shown = %v, want %v", shown, want)
	}
}

func TestLambdaPublishProblems(t *testing.T) {
	var fx publishFixture
	read(t, "publish_problems.json", &fx)
	for _, c := range fx.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var in evaluate.PublishInput
			if err := json.Unmarshal(c.Input, &in); err != nil {
				t.Fatal(err)
			}
			fake := &fakeLambda{reply: answers(c.Output)}
			got, err := evaluate.Lambda{Client: fake, Function: "f"}.PublishProblems(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			if string(fake.payloads[0]["op"]) != `"publish-problems"` || !sameJSON(t, fake.payloads[0]["input"], c.Input) {
				t.Errorf("payload = %s", fake.calls[0].Payload)
			}
			var want struct{ Problems []evaluate.Problem }
			if err := json.Unmarshal(c.Output, &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want.Problems) {
				t.Errorf("problems = %+v, want %+v", got, want.Problems)
			}
			// Every test case runs: the publish time limit, as over HTTP.
			if d := fake.deadline[0]; d > evaluate.PublishTimeout || d < evaluate.PublishTimeout-time.Second {
				t.Errorf("deadline = %v, want about %v", d, evaluate.PublishTimeout)
			}
		})
	}
}

// Each way the invoke can fail is an error (so the send fails safe: EVALUATE_FAILED), and none
// holds an answer or the function's reply.
func TestLambdaFailures(t *testing.T) {
	in := evaluate.Input{FormID: uuid.New(), Answers: map[uuid.UUID]json.RawMessage{uuid.New(): json.RawMessage(`{"q_secret":"o_y"}`)}}
	out := func(o lambda.InvokeOutput) func(string, json.RawMessage) (*lambda.InvokeOutput, error) {
		return func(string, json.RawMessage) (*lambda.InvokeOutput, error) { return &o, nil }
	}
	cases := map[string]func(string, json.RawMessage) (*lambda.InvokeOutput, error){
		"the invoke fails (e.g. no permission)": func(string, json.RawMessage) (*lambda.InvokeOutput, error) {
			return nil, errors.New("AccessDeniedException: not authorized")
		},
		"a function error (it threw, timed out or ran out of memory)": out(lambda.InvokeOutput{
			StatusCode: 200, FunctionError: aws.String("Unhandled"), Payload: []byte(`{"errorMessage":"q_secret"}`),
		}),
		"a reply with status 500": out(lambda.InvokeOutput{StatusCode: 200, Payload: []byte(`{"status":500,"body":{"error":"q_secret"}}`)}),
		"a reply with status 400": out(lambda.InvokeOutput{StatusCode: 200, Payload: []byte(`{"status":400,"body":{"error":"q_secret"}}`)}),
		"a reply that isn't JSON":  out(lambda.InvokeOutput{StatusCode: 200, Payload: []byte(`q_secret`)}),
		"a reply with no evaluator": out(lambda.InvokeOutput{
			StatusCode: 200, Payload: []byte(`{"status":200,"body":{"shown":[],"q_secret":1}}`),
		}),
		"an invoke status that isn't 200": out(lambda.InvokeOutput{StatusCode: 202, Payload: []byte(`{"status":200,"body":{"evaluator":"x"}}`)}),
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := evaluate.Lambda{Client: &fakeLambda{reply: reply}, Function: "f"}.Evaluate(context.Background(), in)
			if err == nil {
				t.Fatal("no error")
			}
			if strings.Contains(err.Error(), "q_secret") {
				t.Errorf("the error holds the reply: %v", err)
			}
		})
	}
	t.Run("a timeout", func(t *testing.T) {
		start := time.Now()
		_, err := evaluate.Lambda{Client: &fakeLambda{block: true}, Function: "f", Timeout: 50 * time.Millisecond}.Evaluate(context.Background(), in)
		if err == nil || time.Since(start) > 2*time.Second {
			t.Fatalf("err = %v after %v", err, time.Since(start))
		}
	})
	t.Run("shown and publish checks: a function error", func(t *testing.T) {
		bad := out(lambda.InvokeOutput{StatusCode: 200, FunctionError: aws.String("Unhandled"), Payload: []byte(`{"errorMessage":"q_secret"}`)})
		l := evaluate.Lambda{Client: &fakeLambda{reply: bad}, Function: "f"}
		if _, err := l.Shown(context.Background(), evaluate.ShownInput{FormID: uuid.New()}); err == nil || strings.Contains(err.Error(), "q_secret") {
			t.Errorf("Shown: %v", err)
		}
		if _, err := l.PublishProblems(context.Background(), evaluate.PublishInput{}); err == nil || strings.Contains(err.Error(), "q_secret") {
			t.Errorf("PublishProblems: %v", err)
		}
	})
}

// infra's alarm counts the APIs' EVALUATE_FAILED lines (S45): the two sides are in different
// languages, so compare them here, as internal/delivery does for DELIVERY_FAILED.
func TestInfraAgrees(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "infra", "api_stack.py"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `EVALUATE_FAILED_MARKER = "`+evaluate.FailedMarker+`"`) {
		t.Errorf("infra/api_stack.py's EVALUATE_FAILED_MARKER isn't %q", evaluate.FailedMarker)
	}
	// The ops the Go side invokes are the ones the Lambda takes (apps/evaluate/src/lambda.ts).
	ts, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluate", "src", "handler.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ts), `OP_NAMES = ["evaluate", "publish-problems", "shown"]`) {
		t.Error("apps/evaluate/src/handler.ts's OP_NAMES aren't evaluate, publish-problems, shown")
	}
}
