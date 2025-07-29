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
	if awsCredential := connCfg.DataSource.GetAwsCredential(); awsCredential != nil {
		return config.LoadDefaultConfig(ctx,
			config.WithRegion(connCfg.DataSource.GetRegion()),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				awsCredential.AccessKeyId,
				awsCredential.SecretAccessKey,
				awsCredential.SessionToken,
			)),
		)
	}

	return config.LoadDefaultConfig(ctx)
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
