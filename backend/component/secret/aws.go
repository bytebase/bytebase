package secret

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func getSecretFromAWS(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (string, error) {
	// for AWS auth we will use the default credentials (environment)
	// ref:
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", errors.Wrapf(err, "failed to init aws config: %v", err.Error())
	}

	client := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(externalSecret.SecretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	secret, err := client.GetSecretValue(ctx, input)
	if err != nil {
		// For a list of exceptions thrown, see
		// https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return "", errors.Wrapf(err, "cannot found secret %s", externalSecret.SecretName)
		}
		return "", errors.Wrapf(err, "failed to get aws secret")
	}

	if secret.SecretString == nil {
		return "", errors.Errorf("empty secret string")
	}

	dataMap := make(map[string]any)
	if err := json.Unmarshal([]byte(*secret.SecretString), &dataMap); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal aws secret string")
	}
	val, ok := dataMap[externalSecret.PasswordKeyName].(string)
	if !ok {
		return "", errors.Errorf("cannot get value for %s, please make sure the secret exists", externalSecret.PasswordKeyName)
	}
	return val, nil
}
