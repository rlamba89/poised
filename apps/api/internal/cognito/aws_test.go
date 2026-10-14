package cognito

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// fakeIDP records AdminCreateUser calls and answers each with the next error in errs (nil: ok).
type fakeIDP struct {
	calls []*cip.AdminCreateUserInput
	errs  []error
}

func (f *fakeIDP) AdminCreateUser(_ context.Context, in *cip.AdminCreateUserInput, _ ...func(*cip.Options)) (*cip.AdminCreateUserOutput, error) {
	f.calls = append(f.calls, in)
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	return &cip.AdminCreateUserOutput{}, nil
}

func attrs(in *cip.AdminCreateUserInput) map[string]string {
	m := map[string]string{}
	for _, a := range in.UserAttributes {
		m[*a.Name] = *a.Value
	}
	return m
}

func TestCreateUserInvitesByEmailWithTheEmailVerified(t *testing.T) {
	f := &fakeIDP{}
	a := &AWS{PoolID: "ca-central-1_Pool", idp: f}
	if err := a.CreateUser(context.Background(), "zoe@trust.example"); err != nil {
		t.Fatal(err)
	}
	in := f.calls[0]
	if *in.UserPoolId != "ca-central-1_Pool" || *in.Username != "zoe@trust.example" || in.MessageAction != "" {
		t.Errorf("input = %+v", in)
	}
	if got := attrs(in); got["email"] != "zoe@trust.example" || got["email_verified"] != "true" {
		t.Errorf("attributes = %v", got)
	}
	if len(in.DesiredDeliveryMediums) != 1 || in.DesiredDeliveryMediums[0] != types.DeliveryMediumTypeEmail {
		t.Errorf("delivery = %v", in.DesiredDeliveryMediums)
	}
}

// The pool may already have the person (e.g. QA's database was re-created, or the test-users
// command made them): they can sign in with that account, so the invite still succeeds.
func TestCreateUserForSomeoneThePoolHasIsFine(t *testing.T) {
	f := &fakeIDP{errs: []error{&types.UsernameExistsException{}}}
	if err := (&AWS{PoolID: "p", idp: f}).CreateUser(context.Background(), "zoe@trust.example"); err != nil {
		t.Fatal(err)
	}
}

func TestCreateUserPassesOtherErrorsOn(t *testing.T) {
	f := &fakeIDP{errs: []error{&types.LimitExceededException{}}}
	if err := (&AWS{PoolID: "p", idp: f}).CreateUser(context.Background(), "zoe@trust.example"); err == nil {
		t.Fatal("no error")
	}
}

func TestResendInvite(t *testing.T) {
	t.Run("resends", func(t *testing.T) {
		f := &fakeIDP{}
		if err := (&AWS{PoolID: "p", idp: f}).ResendInvite(context.Background(), "zoe@trust.example"); err != nil {
			t.Fatal(err)
		}
		if f.calls[0].MessageAction != types.MessageActionTypeResend || *f.calls[0].Username != "zoe@trust.example" {
			t.Errorf("input = %+v", f.calls[0])
		}
	})
	t.Run("someone the pool doesn't have is created (e.g. a re-created pool)", func(t *testing.T) {
		f := &fakeIDP{errs: []error{&types.UserNotFoundException{}, nil}}
		if err := (&AWS{PoolID: "p", idp: f}).ResendInvite(context.Background(), "zoe@trust.example"); err != nil {
			t.Fatal(err)
		}
		if len(f.calls) != 2 || f.calls[1].MessageAction != "" {
			t.Errorf("calls = %d; want a resend, then a create", len(f.calls))
		}
	})
	t.Run("someone who already set their password needs no new one", func(t *testing.T) {
		f := &fakeIDP{errs: []error{&types.UnsupportedUserStateException{}}}
		if err := (&AWS{PoolID: "p", idp: f}).ResendInvite(context.Background(), "zoe@trust.example"); err != nil {
			t.Fatal(err)
		}
	})
}

// tokenEndpoint is the managed login's /oauth2/token: it answers code "good" with an ID token.
func tokenEndpoint(t *testing.T, got *url.Values) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth2/token" ||
			r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_ = r.ParseForm()
		*got = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		if r.PostForm.Get("code") != "good" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id_token":"the.id.token","access_token":"a","refresh_token":"r","expires_in":3600,"token_type":"Bearer"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestExchangeSendsTheCodeAndVerifier(t *testing.T) {
	var form url.Values
	srv := tokenEndpoint(t, &form)
	a := &AWS{ClientID: "client1", Domain: srv.URL, RedirectURI: "https://app.example.test/auth/callback"}
	tok, err := a.Exchange(context.Background(), "good", "the-verifier")
	if err != nil || tok != "the.id.token" {
		t.Fatalf("got %q, %v", tok, err)
	}
	want := url.Values{"grant_type": {"authorization_code"}, "client_id": {"client1"}, "code": {"good"},
		"redirect_uri": {"https://app.example.test/auth/callback"}, "code_verifier": {"the-verifier"}}
	for k, v := range want {
		if form.Get(k) != v[0] {
			t.Errorf("%s = %q, want %q", k, form.Get(k), v[0])
		}
	}
}

func TestExchangeRefusedCodeIsErrCode(t *testing.T) {
	var form url.Values
	srv := tokenEndpoint(t, &form)
	a := &AWS{ClientID: "client1", Domain: srv.URL, RedirectURI: "https://app.example.test/auth/callback"}
	if _, err := a.Exchange(context.Background(), "used-already", "v"); !errors.Is(err, ErrCode) {
		t.Fatalf("got %v, want ErrCode", err)
	}
}

func TestExchangeServerTroubleIsNotErrCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oops", http.StatusInternalServerError)
	}))
	defer srv.Close()
	a := &AWS{ClientID: "client1", Domain: srv.URL}
	if _, err := a.Exchange(context.Background(), "good", "v"); err == nil || errors.Is(err, ErrCode) {
		t.Fatalf("got %v", err)
	}
}
