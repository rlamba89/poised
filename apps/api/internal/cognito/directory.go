package cognito

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// Directory invites staff into the user pool (STF-01). Cognito emails a new person a
// temporary password; they set their own at first sign-in on the managed login page.
type Directory struct {
	api    *cip.Client
	poolID string
}

// NewDirectory returns a Directory for a pool, using AWS credentials from cfg (locally, the
// AWS_PROFILE in .env; on Lambda, the function's role).
func NewDirectory(cfg aws.Config, poolID string) *Directory {
	return &Directory{api: cip.NewFromConfig(cfg), poolID: poolID}
}

// Invite creates the person in the pool and returns their Cognito subject. An email that is
// already in the pool is not an error: nothing is sent and their subject is returned, so an
// existing account only gains a membership (STF-02).
func (d *Directory) Invite(ctx context.Context, email, name string) (string, error) {
	out, err := d.api.AdminCreateUser(ctx, &cip.AdminCreateUserInput{
		UserPoolId: aws.String(d.poolID),
		Username:   aws.String(email),
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
			{Name: aws.String("email_verified"), Value: aws.String("true")},
			{Name: aws.String("name"), Value: aws.String(name)},
		},
		DesiredDeliveryMediums: []types.DeliveryMediumType{types.DeliveryMediumTypeEmail},
	})
	var exists *types.UsernameExistsException
	if errors.As(err, &exists) {
		got, err := d.api.AdminGetUser(ctx, &cip.AdminGetUserInput{UserPoolId: aws.String(d.poolID), Username: aws.String(email)})
		if err != nil {
			return "", fmt.Errorf("get existing user: %w", err)
		}
		return subOf(got.UserAttributes)
	}
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}
	return subOf(out.User.Attributes)
}

func subOf(attrs []types.AttributeType) (string, error) {
	for _, a := range attrs {
		if aws.ToString(a.Name) == "sub" {
			return aws.ToString(a.Value), nil
		}
	}
	return "", errors.New("cognito user has no sub")
}
