package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	// DynamoDB do not have the concept of Database, which is important concept in Bytebase.
	// We use the format {account_id}-{region} as the pseudo database name.
	stsClient := sts.NewFromConfig(d.awsConfig)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get caller identity")
	}
	if identity.Account == nil {
		return nil, errors.New("account id is empty in the caller identity")
	}
	region := d.awsConfig.Region
	if region == "" {
		return nil, errors.New("region is empty in the AWS config")
	}
	databaseName := formatDatabaseName(*identity.Account, region)

	return &db.InstanceMetadata{
		Databases: []*storepb.DatabaseSchemaMetadata{
			{
				Name: databaseName,
			},
		},
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	panic("implement me")
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	panic("implement me")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	panic("implement me")
}

func formatDatabaseName(accountID string, region string) string {
	return accountID + "-" + region
}
