package trino

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Trino version")
	}

	catalogList, err := d.getCatalogList(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get catalog list")
	}

	var catalogMetadata []*storepb.DatabaseSchemaMetadata
	for _, catalog := range catalogList {
		catalogMetadata = append(catalogMetadata, &storepb.DatabaseSchemaMetadata{
			Name: catalog,
		})
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: catalogMetadata,
		Metadata:  &storepb.Instance{},
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	catalog := d.databaseName
	dbMeta := &storepb.DatabaseSchemaMetadata{
		Name: catalog,
	}

	schemaNames, err := d.getSchemaList(ctx, catalog)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schema list for catalog %s", catalog)
	}

	// Fetch all tables in all schemas in one query
	allTables, err := d.fetchAllTablesForCatalog(ctx, catalog, schemaNames)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch tables for catalog %s", catalog)
	}

	// Fetch all columns for all tables in one query
	allColumns, err := d.fetchAllColumnsForCatalog(ctx, catalog)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch columns for catalog %s", catalog)
	}

	// Organize the data into schemas
	var schemas []*storepb.SchemaMetadata
	for _, schemaName := range schemaNames {
		tables := allTables[schemaName]

		// Add column metadata to each table
		for i, table := range tables {
			if columns, ok := allColumns[schemaName+"."+table.Name]; ok {
				tables[i].Columns = columns
			}
		}

		schemas = append(schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tables,
		})
	}

	dbMeta.Schemas = schemas
	return dbMeta, nil
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	var version string

	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

func (d *Driver) queryStringValues(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}
		if value != "" {
			results = append(results, value)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error during row iteration")
	}

	return results, nil
}

func (d *Driver) getCatalogList(ctx context.Context) ([]string, error) {
	query := "SELECT name FROM system.metadata.catalogs ORDER BY name"
	return d.queryStringValues(ctx, query)
}

func (d *Driver) getSchemaList(ctx context.Context, catalog string) ([]string, error) {
	query := "SELECT table_schem FROM system.jdbc.schemas WHERE table_catalog = ? ORDER BY table_schem"
	return d.queryStringValues(ctx, query, catalog)
}

func (d *Driver) fetchAllTablesForCatalog(ctx context.Context, catalog string, schemas []string) (map[string][]*storepb.TableMetadata, error) {
	query := "SELECT table_schem, table_name FROM system.jdbc.tables WHERE table_cat = ? AND table_type = 'TABLE' ORDER BY table_schem, table_name"

	rows, err := d.db.QueryContext(ctx, query, catalog)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query tables for catalog")
	}
	defer rows.Close()

	allTables := make(map[string][]*storepb.TableMetadata)
	for _, schema := range schemas {
		allTables[schema] = []*storepb.TableMetadata{}
	}

	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			return nil, errors.Wrap(err, "failed to scan table row")
		}

		if tableName != "" {
			table := &storepb.TableMetadata{Name: tableName}
			allTables[schemaName] = append(allTables[schemaName], table)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error during table row iteration")
	}

	return allTables, nil
}

func (d *Driver) fetchAllColumnsForCatalog(ctx context.Context, catalog string) (map[string][]*storepb.ColumnMetadata, error) {
	query := "SELECT table_schem, table_name, column_name, type_name, is_nullable FROM system.jdbc.columns WHERE table_cat = ? ORDER BY table_schem, table_name, ordinal_position"

	rows, err := d.db.QueryContext(ctx, query, catalog)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query columns for catalog")
	}
	defer rows.Close()

	allColumns := make(map[string][]*storepb.ColumnMetadata)

	for rows.Next() {
		var schemaName, tableName, columnName, dataType, isNullable string
		if err := rows.Scan(&schemaName, &tableName, &columnName, &dataType, &isNullable); err != nil {
			return nil, errors.Wrap(err, "failed to scan column row")
		}

		column := &storepb.ColumnMetadata{
			Name:     columnName,
			Type:     dataType,
			Nullable: isNullable == "YES",
		}

		tableKey := schemaName + "." + tableName
		allColumns[tableKey] = append(allColumns[tableKey], column)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error during column row iteration")
	}

	return allColumns, nil
}
