package cosmosdb

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance meta.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata
	// List databases names.
	// https://github.com/Azure/azure-sdk-for-go/pull/19769
	queryPager := driver.client.NewQueryDatabasesPager("select * from dbs d", nil)
	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get next page in database list")
		}
		for _, database := range queryResponse.Databases {
			databases = append(databases, &storepb.DatabaseSchemaMetadata{
				Name: database.ID,
			})
		}
	}
	return &db.InstanceMetadata{
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs the database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	var containers []*storepb.TableMetadata
	// List containers names.
	// https://github.com/Azure/azure-sdk-for-go/pull/19769
	database, err := driver.client.NewDatabase(driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", driver.databaseName)
	}
	queryPager := database.NewQueryContainersPager("select * from colls c", nil)
	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get next page in container list")
		}
		for _, container := range queryResponse.Containers {
			containers = append(containers, &storepb.TableMetadata{
				Name: container.ID,
			})
		}
	}
	return &storepb.DatabaseSchemaMetadata{
		Name: driver.databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Tables: containers,
			},
		},
	}, nil
}
