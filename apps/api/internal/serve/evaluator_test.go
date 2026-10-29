package serve_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/config"
	"github.com/rlamba89/poised/apps/api/internal/evaluate"
	"github.com/rlamba89/poised/apps/api/internal/serve"
)

// S45: the evaluate service is the Lambda EVALUATE_FUNCTION names on AWS, and the address
// EVALUATE_URL names locally. The Lambda client is made only when it's needed.
func TestEvaluator(t *testing.T) {
	made := 0
	client := &struct{ evaluate.LambdaClient }{}
	newClient := func(context.Context) (evaluate.LambdaClient, error) {
		made++
		return client, nil
	}

	e, err := serve.Evaluator(context.Background(), config.Config{EvaluateURL: "http://127.0.0.1:8090"}, newClient)
	if h, ok := e.(evaluate.HTTP); err != nil || !ok || h.URL != "http://127.0.0.1:8090" || made != 0 {
		t.Errorf("EVALUATE_URL: %#v, %v (Lambda clients made: %d)", e, err, made)
	}

	e, err = serve.Evaluator(context.Background(), config.Config{EvaluateFunction: "poised-xx-qa-evaluate"}, newClient)
	if l, ok := e.(evaluate.Lambda); err != nil || !ok || l.Function != "poised-xx-qa-evaluate" || l.Client != client || made != 1 {
		t.Errorf("EVALUATE_FUNCTION: %#v, %v", e, err)
	}

	failing := func(context.Context) (evaluate.LambdaClient, error) { return nil, errors.New("no credentials") }
	if _, err := serve.Evaluator(context.Background(), config.Config{EvaluateFunction: "f"}, failing); err == nil {
		t.Error("a client that can't be made: no error")
	}
}
