package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
)

var awsCfg = sync.OnceValues(func() (aws.Config, error) { return awsconfig.LoadDefaultConfig(context.Background()) })

// awsPool is the staff user pool. Enroll needs the app client to allow ADMIN_USER_PASSWORD_AUTH,
// which infra's AuthStack turns on for QA only.
type awsPool struct {
	poolID, clientID string
	c                *cip.Client
}

func (p *awsPool) client() (*cip.Client, error) {
	if p.c == nil {
		cfg, err := awsCfg()
		if err != nil {
			return nil, err
		}
		p.c = cip.NewFromConfig(cfg)
	}
	return p.c, nil
}

func (p *awsPool) Find(ctx context.Context, email string) (bool, bool, error) {
	c, err := p.client()
	if err != nil {
		return false, false, err
	}
	out, err := c.AdminGetUser(ctx, &cip.AdminGetUserInput{UserPoolId: &p.poolID, Username: &email})
	if missing := new(types.UserNotFoundException); errors.As(err, &missing) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	for _, m := range out.UserMFASettingList {
		if m == "SOFTWARE_TOKEN_MFA" {
			return true, true, nil
		}
	}
	return true, false, nil
}

func (p *awsPool) Create(ctx context.Context, email string) error {
	c, err := p.client()
	if err != nil {
		return err
	}
	_, err = c.AdminCreateUser(ctx, &cip.AdminCreateUserInput{
		UserPoolId: &p.poolID, Username: &email,
		MessageAction: types.MessageActionTypeSuppress, // a made-up address: no email
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: &email},
			{Name: aws.String("email_verified"), Value: aws.String("true")},
		},
	})
	return err
}

func (p *awsPool) Delete(ctx context.Context, email string) error {
	c, err := p.client()
	if err != nil {
		return err
	}
	_, err = c.AdminDeleteUser(ctx, &cip.AdminDeleteUserInput{UserPoolId: &p.poolID, Username: &email})
	return err
}

func (p *awsPool) SetPassword(ctx context.Context, email, password string) error {
	c, err := p.client()
	if err != nil {
		return err
	}
	_, err = c.AdminSetUserPassword(ctx, &cip.AdminSetUserPasswordInput{
		UserPoolId: &p.poolID, Username: &email, Password: &password, Permanent: true,
	})
	return err
}

// Enroll signs in as the user (the pool then asks for MFA_SETUP, since MFA is required), gets an
// authenticator secret, proves it with a code, and finishes the sign-in.
func (p *awsPool) Enroll(ctx context.Context, email, password string) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	start, err := c.AdminInitiateAuth(ctx, &cip.AdminInitiateAuthInput{
		UserPoolId: &p.poolID, ClientId: &p.clientID, AuthFlow: types.AuthFlowTypeAdminUserPasswordAuth,
		AuthParameters: map[string]string{"USERNAME": email, "PASSWORD": password},
	})
	if err != nil {
		return "", err
	}
	if start.ChallengeName != types.ChallengeNameTypeMfaSetup {
		return "", fmt.Errorf("expected the MFA_SETUP challenge, got %q", start.ChallengeName)
	}
	assoc, err := c.AssociateSoftwareToken(ctx, &cip.AssociateSoftwareTokenInput{Session: start.Session})
	if err != nil {
		return "", err
	}
	secret := aws.ToString(assoc.SecretCode)
	code, err := cognito.TOTP(secret, time.Now())
	if err != nil {
		return "", err
	}
	verified, err := c.VerifySoftwareToken(ctx, &cip.VerifySoftwareTokenInput{
		Session: assoc.Session, UserCode: &code, FriendlyDeviceName: aws.String("qa-users"),
	})
	if err != nil {
		return "", err
	}
	if verified.Status != types.VerifySoftwareTokenResponseTypeSuccess {
		return "", fmt.Errorf("the authenticator code was refused (%s)", verified.Status)
	}
	if _, err := c.AdminRespondToAuthChallenge(ctx, &cip.AdminRespondToAuthChallengeInput{
		UserPoolId: &p.poolID, ClientId: &p.clientID, ChallengeName: types.ChallengeNameTypeMfaSetup,
		Session: verified.Session, ChallengeResponses: map[string]string{"USERNAME": email},
	}); err != nil {
		return "", err
	}
	_, err = c.AdminSetUserMFAPreference(ctx, &cip.AdminSetUserMFAPreferenceInput{
		UserPoolId: &p.poolID, Username: &email,
		SoftwareTokenMfaSettings: &types.SoftwareTokenMfaSettingsType{Enabled: true, PreferredMfa: true},
	})
	return secret, err
}

// ssmStore keeps the test users as one SecureString parameter.
type ssmStore struct {
	name string
	c    *ssm.Client
}

func (s *ssmStore) client() (*ssm.Client, error) {
	if s.c == nil {
		cfg, err := awsCfg()
		if err != nil {
			return nil, err
		}
		s.c = ssm.NewFromConfig(cfg)
	}
	return s.c, nil
}

func (s *ssmStore) Load(ctx context.Context) ([]testUser, error) {
	c, err := s.client()
	if err != nil {
		return nil, err
	}
	out, err := c.GetParameter(ctx, &ssm.GetParameterInput{Name: &s.name, WithDecryption: aws.Bool(true)})
	if missing := new(ssmtypes.ParameterNotFound); errors.As(err, &missing) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decode(aws.ToString(out.Parameter.Value))
}

func (s *ssmStore) Save(ctx context.Context, users []testUser) error {
	c, err := s.client()
	if err != nil {
		return err
	}
	v, err := encode(users)
	if err != nil {
		return err
	}
	_, err = c.PutParameter(ctx, &ssm.PutParameterInput{
		Name: &s.name, Value: &v, Type: ssmtypes.ParameterTypeSecureString, Overwrite: aws.Bool(true),
		Description: aws.String("Poised QA: the test staff users' passwords and authenticator secrets (cmd/qa-users)"),
	})
	return err
}
