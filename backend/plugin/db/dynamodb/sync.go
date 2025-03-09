package dynamodb

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{}

	tableNames, err := d.listAllTables(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list all tables")
	}

	// DynamoDB do not support batch describe table, the operation that we call describe table one by one may be
	// very slow because the multi round trip to AWS.
	// We may need to optimize this in the future.
	for _, tableName := range tableNames {
		tableMetadata, err := d.syncTable(ctx, tableName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to describe table: %s", tableName)
		}
		if tableMetadata != nil {
			schemaMetadata.Tables = append(schemaMetadata.Tables, tableMetadata)
		}
	}

	return &storepb.DatabaseSchemaMetadata{
		Name: d.config.ConnectionContext.DatabaseName,
		Schemas: []*storepb.SchemaMetadata{
			schemaMetadata,
		},
	}, nil
}

func (d *Driver) syncTable(ctx context.Context, tableName string) (*storepb.TableMetadata, error) {
	tableMetadata := &storepb.TableMetadata{
		Name: tableName,
	}
	out, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: &tableName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to describe table: %s", tableName)
	}
	if out.Table == nil {
		return nil, errors.Errorf("table not found: %s", tableName)
	}
	if out.Table.TableStatus != types.TableStatusActive {
		return nil, nil
	}
	if out.Table.KeySchema != nil {
		var hashKeyAttributes, rangeKeyAttributes []string
		for _, keySchema := range out.Table.KeySchema {
			if keySchema.KeyType == types.KeyTypeHash {
				hashKeyAttributes = append(hashKeyAttributes, *keySchema.AttributeName)
			} else if keySchema.KeyType == types.KeyTypeRange {
				rangeKeyAttributes = append(rangeKeyAttributes, *keySchema.AttributeName)
			}
		}
		tableMetadata.Indexes = append(tableMetadata.Indexes, &storepb.IndexMetadata{
			Name:        strings.Join(hashKeyAttributes, ","),
			Expressions: append([]string{}, hashKeyAttributes...),
			Type:        "HASH",
		})
		tableMetadata.Indexes = append(tableMetadata.Indexes, &storepb.IndexMetadata{
			Name:        strings.Join(rangeKeyAttributes, ","),
			Expressions: append([]string{}, rangeKeyAttributes...),
			Type:        "RANGE",
		})
		columnsMap := make(map[string]bool)
		for _, key := range hashKeyAttributes {
			columnsMap[key] = true
		}
		for _, key := range rangeKeyAttributes {
			columnsMap[key] = true
		}
		sortedColumns := make([]string, 0, len(columnsMap))
		for key := range columnsMap {
			sortedColumns = append(sortedColumns, key)
		}
		sort.Strings(sortedColumns)
		tableMetadata.Columns = make([]*storepb.ColumnMetadata, 0, len(sortedColumns))
		for _, key := range sortedColumns {
			tableMetadata.Columns = append(tableMetadata.Columns, &storepb.ColumnMetadata{
				Name: key,
			})
		}
	}
	if out.Table.TableName != nil {
		tableMetadata.Name = *out.Table.TableName
	}
	if out.Table.ItemCount != nil {
		tableMetadata.RowCount = *out.Table.ItemCount
	}

	return tableMetadata, nil
}

func (d *Driver) listAllTables(ctx context.Context) ([]string, error) {
	var result []string
	var limit int32 = 100
	var exclusiveStartTableName *string
	for {
		out, err := d.client.ListTables(ctx, &dynamodb.ListTablesInput{
			Limit:                   &limit,
			ExclusiveStartTableName: exclusiveStartTableName,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list tables")
		}
		result = append(result, out.TableNames...)
		if out.LastEvaluatedTableName == nil {
			break
		}
		exclusiveStartTableName = out.LastEvaluatedTableName
	}
	return result, nil
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
