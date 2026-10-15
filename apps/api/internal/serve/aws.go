package serve

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/rlamba89/poised/apps/api/internal/config"
)

// awsSecrets is config.AWS on real AWS: SSM Parameter Store and Secrets Manager, in the
// function's own region (Lambda sets AWS_REGION). The clients are made on first use, so a
// local server never loads AWS credentials.
type awsSecrets struct {
	once sync.Once
	err  error
	ssm  *ssm.Client
	sm   *secretsmanager.Client
}

func (a *awsSecrets) init(ctx context.Context) error {
	a.once.Do(func() {
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			a.err = err
			return
		}
		a.ssm, a.sm = ssm.NewFromConfig(cfg), secretsmanager.NewFromConfig(cfg)
	})
	return a.err
}

func (a *awsSecrets) Parameter(ctx context.Context, name string) (string, error) {
	if err := a.init(ctx); err != nil {
		return "", err
	}
	out, err := a.ssm.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name), WithDecryption: aws.Bool(true)})
	if _, ok := errors.AsType[*ssmtypes.ParameterNotFound](err); ok {
		return "", fmt.Errorf("%w: %w", config.ErrNotFound, err)
	}
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Parameter.Value), nil
}

func (a *awsSecrets) Secret(ctx context.Context, id string) (string, error) {
	if err := a.init(ctx); err != nil {
		return "", err
	}
	out, err := a.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(id)})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.SecretString), nil
}
