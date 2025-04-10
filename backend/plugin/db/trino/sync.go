package trino

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get version")
	}

	var databases []*storepb.DatabaseSchemaMetadata

	catalogRows, err := d.db.QueryContext(ctx, "SHOW CATALOGS")
	if err != nil {
		return nil, err
	}
	defer catalogRows.Close()

	for catalogRows.Next() {
		var catalog string
		if err := catalogRows.Scan(&catalog); err != nil {
			return nil, err
		}

		database := &storepb.DatabaseSchemaMetadata{
			Name: catalog,
		}

		schemaQuery := fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog)
		// Use an IIFE (Immediately Invoked Function Expression) to encapsulate the schema query
		// This lets us use defer for the rows while avoiding the "defer in loop" issue
		schemas, err := func() ([]*storepb.SchemaMetadata, error) {
			schemaRows, err := d.db.QueryContext(ctx, schemaQuery)
			if err != nil {
				return nil, err
			}
			defer schemaRows.Close()
			var schemas []*storepb.SchemaMetadata
			for schemaRows.Next() {
				var schemaName string
				if err := schemaRows.Scan(&schemaName); err != nil {
					return nil, err
				}
				schemas = append(schemas, &storepb.SchemaMetadata{
					Name: schemaName,
				})
			}
			// Check for errors from iterating over rows
			if err := schemaRows.Err(); err != nil {
				return nil, err
			}

			return schemas, nil
		}()

		if err != nil {
			// skip catalog if schemas can't be retrieved
			continue
		}

		database.Schemas = schemas
		databases = append(databases, database)
	}

	// Check for errors from iterating over catalog rows
	if err := catalogRows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
	}, nil
}

// func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
func (*Driver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, nil
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	var version string
	query := "SELECT VERSION()"
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		return "", errors.Wrapf(err, "failed to query")
	}
	return version, nil
}
