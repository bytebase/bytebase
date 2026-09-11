package cosmosdb

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// SyncInstance syncs the instance meta.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	endpoint := d.connCfg.DataSource.Host
	if common.IsDev() && isLocalhostEndpoint(endpoint) {
		return d.syncInstanceViaREST()
	}

	var databases []*metadatapb.DatabaseSchemaMetadata
	queryPager := d.client.NewQueryDatabasesPager("select * from dbs d", nil)
	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get next page in database list")
		}
		for _, database := range queryResponse.Databases {
			databases = append(databases, &metadatapb.DatabaseSchemaMetadata{
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
	var databases []*metadatapb.DatabaseSchemaMetadata
	for _, name := range names {
		databases = append(databases, &metadatapb.DatabaseSchemaMetadata{
			Name: name,
		})
	}
	return &db.InstanceMetadata{
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs the database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	endpoint := d.connCfg.DataSource.Host
	if common.IsDev() && isLocalhostEndpoint(endpoint) {
		return d.syncDBSchemaViaREST()
	}

	var containers []*metadatapb.TableMetadata
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
			containers = append(containers, &metadatapb.TableMetadata{
				Name: container.ID,
			})
		}
	}
	return &metadatapb.DatabaseSchemaMetadata{
		Name: d.databaseName,
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Tables: containers,
			},
		},
	}, nil
}

func (d *Driver) syncDBSchemaViaREST() (*metadatapb.DatabaseSchemaMetadata, error) {
	client, err := newEmulatorRESTClient(d.connCfg.DataSource.Host)
	if err != nil {
		return nil, err
	}
	names, err := client.listContainers(d.databaseName)
	if err != nil {
		return nil, err
	}
	var containers []*metadatapb.TableMetadata
	for _, name := range names {
		containers = append(containers, &metadatapb.TableMetadata{
			Name: name,
		})
	}
	return &metadatapb.DatabaseSchemaMetadata{
		Name: d.databaseName,
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Tables: containers,
			},
		},
	}, nil
}
