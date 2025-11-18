//nolint:revive
package util

import (
	"context"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func GetAWSConnectionConfig(ctx context.Context, connCfg db.ConnectionConfig) (aws.Config, error) {
	region := connCfg.DataSource.GetRegion()

	// Only use static credentials if access key is provided
	// If awsCredential exists but AccessKeyId is empty, fall back to default credential chain
	// (EC2 instance role, env vars, etc.) for cross-account role assumption
	if awsCredential := connCfg.DataSource.GetAwsCredential(); awsCredential != nil && awsCredential.AccessKeyId != "" {
		return config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				awsCredential.AccessKeyId,
				awsCredential.SecretAccessKey,
				awsCredential.SessionToken,
			)),
		)
	}

	// Use default credential chain when no static credentials provided
	return config.LoadDefaultConfig(ctx, config.WithRegion(region))
}

func GetGCPConnectionConfig(ctx context.Context, connCfg db.ConnectionConfig) (*cloudsqlconn.Dialer, error) {
	if gcpCredential := connCfg.DataSource.GetGcpCredential(); gcpCredential != nil {
		return cloudsqlconn.NewDialer(ctx,
			cloudsqlconn.WithCredentialsJSON([]byte(gcpCredential.Content)),
		)
	}
	return cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN())
}

func GetAzureConnectionConfig(connCfg db.ConnectionConfig) (azcore.TokenCredential, error) {
	if azureCredential := connCfg.DataSource.GetAzureCredential(); azureCredential != nil {
		c, err := azidentity.NewClientSecretCredential(azureCredential.TenantId, azureCredential.ClientId, azureCredential.ClientSecret, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create client secret credential")
		}
		return c, nil
	}

	c, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to found default Azure credential")
	}
	return c, nil
}
