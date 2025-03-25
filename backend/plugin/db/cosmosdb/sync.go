package cosmosdb

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance meta.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata
	// List databases names.
	// https://github.com/Azure/azure-sdk-for-go/pull/19769
	queryPager := d.client.NewQueryDatabasesPager("select * from dbs d", nil)
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
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	var containers []*storepb.TableMetadata
	// List containers names.
	// https://github.com/Azure/azure-sdk-for-go/pull/19769
	database, err := d.client.NewDatabase(d.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", d.databaseName)
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
		Name: d.databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Tables: containers,
			},
		},
	}, nil
}
