package util

import (
	"context"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
