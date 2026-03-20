package cosmosdb

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// SyncInstance syncs the instance meta.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	endpoint := d.connCfg.DataSource.Host
	if common.IsDev() && isLocalhostEndpoint(endpoint) {
		return d.syncInstanceViaREST()
	}

	var databases []*storepb.DatabaseSchemaMetadata
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

func (d *Driver) syncInstanceViaREST() (*db.InstanceMetadata, error) {
	client, err := newEmulatorRESTClient(d.connCfg.DataSource.Host)
	if err != nil {
		return nil, err
	}
	names, err := client.listDatabases()
	if err != nil {
		return nil, err
	}
	var databases []*storepb.DatabaseSchemaMetadata
	for _, name := range names {
		databases = append(databases, &storepb.DatabaseSchemaMetadata{
			Name: name,
		})
	}
	return &db.InstanceMetadata{
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs the database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	endpoint := d.connCfg.DataSource.Host
	if common.IsDev() && isLocalhostEndpoint(endpoint) {
		return d.syncDBSchemaViaREST()
	}

	var containers []*storepb.TableMetadata
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

func (d *Driver) syncDBSchemaViaREST() (*storepb.DatabaseSchemaMetadata, error) {
	client, err := newEmulatorRESTClient(d.connCfg.DataSource.Host)
	if err != nil {
		return nil, err
	}
	names, err := client.listContainers(d.databaseName)
	if err != nil {
		return nil, err
	}
	var containers []*storepb.TableMetadata
	for _, name := range names {
		containers = append(containers, &storepb.TableMetadata{
			Name: name,
		})
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
