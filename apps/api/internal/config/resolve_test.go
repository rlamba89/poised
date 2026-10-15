package config_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/config"
)

// fakeAWS stands in for SSM Parameter Store and Secrets Manager, and records every fetch.
type fakeAWS struct {
	params, secrets map[string]string
	calls           []string
	notFound        bool // a missing parameter is reported as config.ErrNotFound, as the real client does
}

func (f *fakeAWS) Parameter(_ context.Context, name string) (string, error) {
	f.calls = append(f.calls, "ssm:"+name)
	v, ok := f.params[name]
	if !ok && f.notFound {
		return "", fmt.Errorf("ssm %s: %w", name, config.ErrNotFound)
	}
	if !ok {
		return "", errors.New("ParameterNotFound")
	}
	return v, nil
}

func (f *fakeAWS) Secret(_ context.Context, id string) (string, error) {
	f.calls = append(f.calls, "secret:"+id)
	v, ok := f.secrets[id]
	if !ok {
		return "", errors.New("ResourceNotFoundException")
	}
	return v, nil
}

// lambdaEnv is a QA Lambda's environment, as infra/ApiStack sets it: the secrets and the web
// address are named, not given.
func lambdaEnv() env {
	e := country()
	delete(e, "DATABASE_URL")
	delete(e, "TOKEN_SECRET")
	delete(e, "APP_URL")
	e["AWS_LAMBDA_FUNCTION_NAME"] = "poised-ca-qa-staff-api"
	e["CONFIG_STRICT"] = "true"
	e["MESSAGING"] = "capture"
	e["DATABASE_URL_PARAM"] = "/sj/qa/database-url"
	e["APP_URL_PARAM"] = "/sj/qa/app-url"
	e["TOKEN_SECRET_ARN"] = "arn:aws:secretsmanager:ca-central-1:111111111111:secret:poised-ca-qa-token-AbCdEf"
	// Written by infra's AuthStack (S12), which comes after the web address exists.
	e["STAFF_POOL_ID_PARAM"] = "/sj/qa/cognito-user-pool-id"
	e["STAFF_POOL_CLIENT_ID_PARAM"] = "/sj/qa/cognito-client-id"
	e["STAFF_POOL_DOMAIN_PARAM"] = "/sj/qa/cognito-domain"
	return e
}

func awsWithEverything() *fakeAWS {
	return &fakeAWS{
		params: map[string]string{
			"/sj/qa/database-url":         "postgres://neon.example.test/sj",
			"/sj/qa/app-url":              "https://main.d123.amplifyapp.com",
			"/sj/qa/cognito-user-pool-id": "ca-central-1_AbCdEf123",
			"/sj/qa/cognito-client-id":    "1example23456789client",
			"/sj/qa/cognito-domain":       "https://poised-ca-qa.auth.ca-central-1.amazoncognito.com",
		},
		secrets: map[string]string{
			"arn:aws:secretsmanager:ca-central-1:111111111111:secret:poised-ca-qa-token-AbCdEf": "generated-by-cdk",
		},
	}
}

func TestResolveFetchesNamedValuesFromAWS(t *testing.T) {
	aws := awsWithEverything()
	getenv, err := config.Resolve(context.Background(), lambdaEnv().get, aws)
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if c.DatabaseURL != "postgres://neon.example.test/sj" || c.TokenSecret != "generated-by-cdk" ||
		c.Country.AppURL != "https://main.d123.amplifyapp.com" {
		t.Errorf("got database %q, secret %q, app URL %q", c.DatabaseURL, c.TokenSecret, c.Country.AppURL)
	}
	if c.Cognito.UserPoolID != "ca-central-1_AbCdEf123" || c.Cognito.ClientID != "1example23456789client" ||
		c.Cognito.RedirectURI != "https://main.d123.amplifyapp.com/auth/callback" {
		t.Errorf("cognito = %+v", c.Cognito)
	}
	if len(aws.calls) != 6 {
		t.Errorf("fetched %v; want each value fetched once, at start-up", aws.calls)
	}
	// Anything not named in AWS still comes from the environment.
	if c.Country.TimeZone != "America/Toronto" {
		t.Errorf("time zone = %q", c.Country.TimeZone)
	}
}

func TestResolveLocallyNeverCallsAWS(t *testing.T) {
	aws := &fakeAWS{}
	getenv, err := config.Resolve(context.Background(), required().get, aws)
	if err != nil {
		t.Fatal(err)
	}
	if getenv("DATABASE_URL") != "postgres://x" || getenv("TOKEN_SECRET") != "s3cret" {
		t.Error("plain environment values changed")
	}
	if len(aws.calls) != 0 {
		t.Errorf("called AWS %v with nothing named there", aws.calls)
	}
	if _, err := config.Resolve(context.Background(), required().get, nil); err != nil {
		t.Errorf("no AWS client is needed locally: %v", err)
	}
}

func TestResolveStopsOnAMissingParameter(t *testing.T) {
	aws := awsWithEverything()
	delete(aws.params, "/sj/qa/database-url") // e.g. make aws-setup not run yet in a new account
	_, err := config.Resolve(context.Background(), lambdaEnv().get, aws)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL_PARAM") || !strings.Contains(err.Error(), "/sj/qa/database-url") {
		t.Fatalf("got %v; want an error naming the variable and the parameter", err)
	}
	if strings.Contains(err.Error(), "neon.example.test") {
		t.Error("the error leaks a secret")
	}
}

func TestResolveStopsOnAMissingSecret(t *testing.T) {
	aws := awsWithEverything()
	aws.secrets = nil
	_, err := config.Resolve(context.Background(), lambdaEnv().get, aws)
	if err == nil || !strings.Contains(err.Error(), "TOKEN_SECRET_ARN") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveRefusesAnEmptyFetchedValue(t *testing.T) {
	aws := awsWithEverything()
	aws.params["/sj/qa/app-url"] = "  "
	_, err := config.Resolve(context.Background(), lambdaEnv().get, aws)
	if err == nil || !strings.Contains(err.Error(), "APP_URL_PARAM") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveRefusesAValueGivenBothWays(t *testing.T) {
	e := lambdaEnv()
	e["DATABASE_URL"] = "postgres://somewhere-else"
	_, err := config.Resolve(context.Background(), e.get, awsWithEverything())
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("got %v; a value set both in the environment and in AWS is ambiguous", err)
	}
}

func TestResolveNeedsAnAWSClientWhenAValueIsNamed(t *testing.T) {
	_, err := config.Resolve(context.Background(), lambdaEnv().get, nil)
	if err == nil {
		t.Fatal("named values with no AWS client: want an error")
	}
}

func TestLoadOnLambdaMustBeStrict(t *testing.T) {
	e := required()
	e["AWS_LAMBDA_FUNCTION_NAME"] = "poised-uk-qa-staff-api"
	_, err := config.Load(e.get)
	if err == nil || !strings.Contains(err.Error(), "CONFIG_STRICT") {
		t.Fatalf("got %v; on Lambda there must be no silent fall-back to the local UK defaults", err)
	}
}

func TestLoadForwardedHops(t *testing.T) {
	c, err := config.Load(required().get)
	if err != nil || c.ForwardedHops != 0 {
		t.Fatalf("default: %d, %v; want 0 (RemoteAddr is the client)", c.ForwardedHops, err)
	}
	e := required()
	e["FORWARDED_HOPS"] = "2"
	if c, err := config.Load(e.get); err != nil || c.ForwardedHops != 2 {
		t.Fatalf("got %d, %v", c.ForwardedHops, err)
	}
	for _, bad := range []string{"-1", "x", "9"} {
		e["FORWARDED_HOPS"] = bad
		if _, err := config.Load(e.get); err == nil || !strings.Contains(err.Error(), "FORWARDED_HOPS") {
			t.Errorf("%q: got %v", bad, err)
		}
	}
}

// QA's real sending to its test recipients (S17): the sender address and the recipients are kept
// in SSM (the recipients as a SecureString), so no contact is in the repo or the stack.
func TestResolveFetchesTheSenderAndTestRecipients(t *testing.T) {
	aws := awsWithEverything()
	aws.params["/sj/qa/email-from"] = "sender@ca.example.test"
	aws.params["/sj/qa/messaging-test-recipients"] = "+15550100001,someone@ca.example.test"
	e := lambdaEnv()
	e["EMAIL_FROM_PARAM"] = "/sj/qa/email-from"
	e["MESSAGING_TEST_RECIPIENTS_PARAM"] = "/sj/qa/messaging-test-recipients"
	e["SES_CONFIGURATION_SET"] = "poised-ca-qa"
	e["SMS_CONFIGURATION_SET"] = "poised-ca-qa"
	getenv, err := config.Resolve(context.Background(), e.get, aws)
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sending.EmailFrom != "sender@ca.example.test" || len(c.TestRecipients) != 2 || c.Messaging != "capture" {
		t.Errorf("sending %+v, recipients %d", c.Sending, len(c.TestRecipients))
	}
}

// Deleting QA's test-recipients parameter (make test-recipients-off) switches its real sending
// off: the functions start with no test recipients, rather than failing.
func TestResolveWithoutTheTestRecipientsParameter(t *testing.T) {
	aws := awsWithEverything()
	e := lambdaEnv()
	e["EMAIL_FROM_PARAM"] = "/sj/qa/email-from"
	aws.params["/sj/qa/email-from"] = "sender@ca.example.test"
	e["MESSAGING_TEST_RECIPIENTS_PARAM"] = "/sj/qa/messaging-test-recipients"
	aws.notFound = true
	getenv, err := config.Resolve(context.Background(), e.get, aws)
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(getenv)
	if err != nil || len(c.TestRecipients) != 0 || c.Sending.Enabled() {
		t.Errorf("recipients %d, sending %+v, err %v", len(c.TestRecipients), c.Sending, err)
	}
	// Any other parameter missing still stops the function.
	delete(aws.params, "/sj/qa/app-url")
	if _, err := config.Resolve(context.Background(), e.get, aws); err == nil {
		t.Error("a missing app URL didn't stop it")
	}
}
