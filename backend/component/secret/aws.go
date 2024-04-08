package secret

import (
	"context"
	"encoding/json"
	"os"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// getAWSClient returns the AWS secret manager client.
//
// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
func getAWSClient(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (*secretsmanager.Client, error) {
	environmentConfig := externalSecret.GetAwsEnvironmentConfig()
	if environmentConfig == nil {
		return nil, errors.Errorf("empty aws environment config")
	}

	region := os.Getenv(environmentConfig.Region)
	if region == "" {
		return nil, errors.Errorf("AWS region not found in environment %s", environmentConfig.Region)
	}
	accessKeyID := os.Getenv(environmentConfig.AccessKeyId)
	if accessKeyID == "" {
		return nil, errors.Errorf("AWS access key id not found in environment %s", environmentConfig.AccessKeyId)
	}
	secretAccessKey := os.Getenv(environmentConfig.SecretAccessKey)
	if secretAccessKey == "" {
		return nil, errors.Errorf("AWS secret access key not found in environment %s", environmentConfig.SecretAccessKey)
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKeyID,
				secretAccessKey,
				os.Getenv(environmentConfig.SessionToken),
			),
		),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to init aws config: %v", err.Error())
	}

	return secretsmanager.NewFromConfig(cfg), nil
}

func getSecretFromAWS(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (string, error) {
	client, err := getAWSClient(ctx, externalSecret)
	if err != nil {
		return "", err
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(externalSecret.SecretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	secret, err := client.GetSecretValue(ctx, input)
	if err != nil {
		// For a list of exceptions thrown, see
		// https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
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
		return "", errors.Errorf("cannot convert %s value to string", externalSecret.PasswordKeyName)
	}
	return val, nil
}
