package cognito

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// idp is the part of the Cognito API the API uses; tests fake it.
type idp interface {
	AdminCreateUser(ctx context.Context, in *cip.AdminCreateUserInput, opts ...func(*cip.Options)) (*cip.AdminCreateUserOutput, error)
}

// AWS is the real staff user pool (S12), on AWS: invitations by AdminCreateUser (Cognito emails
// the temporary password), and the managed login's code exchange.
type AWS struct {
	PoolID, ClientID string
	Domain           string // the managed login's address, https://…amazoncognito.com
	RedirectURI      string // the app's /auth/callback, as registered on the app client
	HTTP             *http.Client

	once sync.Once
	err  error
	idp  idp
}

// client makes the SDK client on first use (in the function's own region, from its role), so
// nothing loads AWS credentials until an invite is sent.
func (a *AWS) client(ctx context.Context) (idp, error) {
	a.once.Do(func() {
		if a.idp != nil {
			return
		}
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			a.err = err
			return
		}
		a.idp = cip.NewFromConfig(cfg)
	})
	return a.idp, a.err
}

func (a *AWS) create(ctx context.Context, email string, action types.MessageActionType) error {
	c, err := a.client(ctx)
	if err != nil {
		return err
	}
	_, err = c.AdminCreateUser(ctx, &cip.AdminCreateUserInput{
		UserPoolId:             aws.String(a.PoolID),
		Username:               aws.String(email),
		MessageAction:          action,
		DesiredDeliveryMediums: []types.DeliveryMediumType{types.DeliveryMediumTypeEmail},
		// Marked verified, so the ID token's email can be trusted and "Forgot password" works.
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
			{Name: aws.String("email_verified"), Value: aws.String("true")},
		},
	})
	return err
}

// CreateUser creates the account and Cognito emails the invitation with a temporary password
// (valid for 7 days, as the pool is set up). If the pool already has the person, they sign in
// with that account, so that's not an error.
func (a *AWS) CreateUser(ctx context.Context, email string) error {
	err := a.create(ctx, email, "")
	if exists := new(types.UsernameExistsException); errors.As(err, &exists) {
		log.Printf("cognito: %s already has an account in the user pool; no new invitation sent", email)
		return nil
	}
	return err
}

// ResendInvite sends a new temporary password. Someone the pool doesn't have (e.g. after the
// pool was re-created) is created; someone who has already set a password needs none.
func (a *AWS) ResendInvite(ctx context.Context, email string) error {
	err := a.create(ctx, email, types.MessageActionTypeResend)
	if missing := new(types.UserNotFoundException); errors.As(err, &missing) {
		return a.CreateUser(ctx, email)
	}
	if state := new(types.UnsupportedUserStateException); errors.As(err, &state) {
		log.Printf("cognito: %s has already set a password; no new invitation sent", email)
		return nil
	}
	return err
}

// Exchange trades the code from the managed login's redirect for the ID token, at the token
// endpoint, with the PKCE verifier (the app client has no secret).
func (a *AWS) Exchange(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{
		"grant_type": {"authorization_code"}, "client_id": {a.ClientID}, "code": {code},
		"redirect_uri": {a.RedirectURI}, "code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Domain+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	hc := a.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	res, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("cognito token endpoint: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var out struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(body, &out)
	switch {
	case res.StatusCode == http.StatusBadRequest && out.Error != "":
		// invalid_grant: the code is wrong, used or expired; or the verifier doesn't match.
		return "", fmt.Errorf("%w (%s)", ErrCode, out.Error)
	case res.StatusCode != http.StatusOK:
		return "", fmt.Errorf("cognito token endpoint: %s", res.Status)
	case out.IDToken == "":
		return "", errors.New("cognito token endpoint: no id_token in the answer")
	}
	return out.IDToken, nil
}
