package evaluate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/google/uuid"
)

// LambdaClient is the one call Lambda makes on the AWS SDK's client (*lambda.Client), so a test can
// give it a fake.
type LambdaClient interface {
	Invoke(ctx context.Context, in *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

// Lambda invokes the evaluate Lambda (S45, D-4) named by Function (EVALUATE_FUNCTION), synchronously,
// with the payload {"op": "evaluate" | "publish-problems" | "shown", "input": …}. It answers
// {"status": 200, "body": …}, the body being what the HTTP service's reply would be
// (apps/evaluate/src/lambda.ts). A failed invoke, a function error (it threw, timed out or ran out
// of memory), or a status other than 200 is an error, so the send fails safe as with HTTP.
type Lambda struct {
	Client   LambdaClient
	Function string
	Timeout  time.Duration // 0: EvaluateTimeout or PublishTimeout
}

func (l Lambda) Evaluate(ctx context.Context, in Input) (Result, error) {
	return evaluateVia(ctx, l.invoke, l.Timeout, in)
}

func (l Lambda) PublishProblems(ctx context.Context, in PublishInput) ([]Problem, error) {
	return publishProblemsVia(ctx, l.invoke, l.Timeout, in)
}

// Shown is the sets shown, in order, for the answers that count now (S44).
func (l Lambda) Shown(ctx context.Context, in ShownInput) ([]uuid.UUID, error) {
	return shownVia(ctx, l.invoke, l.Timeout, in)
}

// invoke runs one operation and returns its 200 reply's body. Its errors never hold the payload or
// the function's error message (either could quote an answer): only the operation and the kind.
func (l Lambda) invoke(ctx context.Context, op string, timeout time.Duration, in any) ([]byte, error) {
	payload, err := json.Marshal(struct {
		Op    string `json:"op"`
		Input any    `json:"input"`
	}{op, in})
	if err != nil {
		return nil, fmt.Errorf("evaluate: lambda %s: %w", op, err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := l.Client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(l.Function),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluate: lambda %s: %w", op, err)
	}
	if out.FunctionError != nil {
		return nil, fmt.Errorf("evaluate: lambda %s: function error (%s)", op, aws.ToString(out.FunctionError))
	}
	if out.StatusCode != 200 {
		return nil, fmt.Errorf("evaluate: lambda %s: invoke status %d", op, out.StatusCode)
	}
	if len(out.Payload) > maxReply {
		return nil, fmt.Errorf("evaluate: lambda %s: the reply is too big", op)
	}
	var reply struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}
	if json.Unmarshal(out.Payload, &reply) != nil {
		return nil, fmt.Errorf("evaluate: lambda %s: the reply isn't JSON", op)
	}
	if reply.Status != 200 {
		return nil, fmt.Errorf("evaluate: lambda %s: status %d", op, reply.Status)
	}
	return reply.Body, nil
}
