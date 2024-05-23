package bigquery

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/api/iterator"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata

	it := d.client.Datasets(ctx)
	for {
		dataset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		databases = append(databases, &storepb.DatabaseSchemaMetadata{Name: dataset.DatasetID})
	}

	return &db.InstanceMetadata{
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{}

	ts := d.client.Dataset(d.databaseName).Tables(ctx)
	for {
		t, err := ts.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		schemaMetadata.Tables = append(schemaMetadata.Tables, &storepb.TableMetadata{Name: t.TableID})
	}
	return &storepb.DatabaseSchemaMetadata{
		Name:    d.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
