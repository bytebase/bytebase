package trino

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type contextKey string

const (
	explicitDBNameKey contextKey = "explicitDbName"
	databaseKey       contextKey = "database"
	resourceIDKey     contextKey = "resourceID"
	nameKey           contextKey = "name"
	requestNameKey    contextKey = "requestName"
	databaseNameKey   contextKey = "databaseName"
	checkKey          contextKey = "check"
	pathKey           contextKey = "path"
	statementKey      contextKey = "statement"
	resourceIDKey2    contextKey = "resourceId"
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

	if len(catalogMetadata) == 0 {
		catalogMetadata = append(catalogMetadata, &storepb.DatabaseSchemaMetadata{
			Name:    "system",
			Schemas: []*storepb.SchemaMetadata{},
		})
	}

	var syncDatabases []string
	for _, db := range catalogMetadata {
		syncDatabases = append(syncDatabases, db.Name)
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: catalogMetadata,
		Metadata: &storepb.Instance{
			SyncDatabases: syncDatabases,
		},
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	databaseName, err := d.getDatabaseNameForSync(ctx)
	if err != nil {
		return nil, err
	}

	catalog := databaseName
	dbMeta := &storepb.DatabaseSchemaMetadata{
		Name: catalog,
	}

	schemaNames, err := d.getSchemaList(ctx, catalog)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schema list for catalog %s", catalog)
	}

	var schemas []*storepb.SchemaMetadata
	for _, schemaName := range schemaNames {
		tables, err := d.fetchTablesForSchema(ctx, schemaName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to fetch tables for schema %s", schemaName)
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

func (d *Driver) queryStringValues(ctx context.Context, query string) ([]string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(queryCtx, query)
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
	query := fmt.Sprintf("SELECT DISTINCT table_schem FROM system.jdbc.schemas WHERE table_catalog = '%s' ORDER BY table_schem", catalog)
	return d.queryStringValues(ctx, query)
}

func (d *Driver) queryColumns(ctx context.Context, query string) ([]*storepb.ColumnMetadata, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(queryCtx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*storepb.ColumnMetadata
	for rows.Next() {
		var name, dataType, isNullable string
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			return nil, errors.Wrap(err, "failed to scan column row")
		}
		columns = append(columns, &storepb.ColumnMetadata{
			Name:     name,
			Type:     dataType,
			Nullable: isNullable == "YES",
		})
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error during column row iteration")
	}

	return columns, nil
}

func (d *Driver) fetchColumnsForTable(ctx context.Context, schema, table string) ([]*storepb.ColumnMetadata, error) {
	query := fmt.Sprintf(
		"SELECT column_name, type_name as data_type, is_nullable "+
			"FROM system.jdbc.columns "+
			"WHERE table_schem = '%s' AND table_name = '%s' "+
			"ORDER BY ordinal_position",
		schema, table)

	return d.queryColumns(ctx, query)
}

func (d *Driver) fetchTablesForSchema(ctx context.Context, schema string) ([]*storepb.TableMetadata, error) {
	query := fmt.Sprintf("SELECT DISTINCT table_name FROM system.jdbc.tables WHERE table_schem = '%s' AND table_type = 'TABLE' ORDER BY table_name", schema)
	tableNames, err := d.queryStringValues(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query tables for schema")
	}

	var tables []*storepb.TableMetadata
	for _, tableName := range tableNames {
		if tableName != "" {
			table := &storepb.TableMetadata{Name: tableName}
			columns, err := d.fetchColumnsForTable(ctx, schema, tableName)
			if err == nil && len(columns) > 0 {
				table.Columns = columns
			}
			tables = append(tables, table)
		}
	}

	return tables, nil
}

func (d *Driver) getDatabaseNameForSync(ctx context.Context) (string, error) {
	resourceID, ok := ctx.Value(resourceIDKey).(string)
	if ok && resourceID != "" {
		if dbName, found := extractDatabaseNameFromResourcePath(resourceID); found {
			return dbName, nil
		}
	}

	if ctx.Value(databaseKey) != nil {
		dbName, ok := ctx.Value(databaseKey).(string)
		if ok && dbName != "" {
			if dbResult, found := extractDatabaseNameFromResourcePath(dbName); found {
				return dbResult, nil
			}

			parts := strings.Split(dbName, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1], nil
			}
		}
	}

	if ctx.Value(databaseNameKey) != nil {
		if dbName, ok := ctx.Value(databaseNameKey).(string); ok && dbName != "" {
			return dbName, nil
		}
	}

	if ctx.Value(checkKey) != nil {
		if explicitDBName, ok := ctx.Value(explicitDBNameKey).(string); ok && explicitDBName != "" {
			return explicitDBName, nil
		}

		for _, key := range []contextKey{requestNameKey, nameKey} {
			if reqName, ok := ctx.Value(key).(string); ok && reqName != "" {
				if dbName, found := extractDatabaseNameFromResourcePath(reqName); found {
					return dbName, nil
				}
			}
		}

		if d.config.DataSource.Database != "" {
			return d.config.DataSource.Database, nil
		}

		catalogs, err := d.queryStringValues(ctx, "SHOW CATALOGS")
		if err != nil {
			slog.Warn("failed to query catalogs", log.BBError(err))
		}

		if len(catalogs) > 0 {
			return catalogs[0], nil
		}

		return "system", nil
	}

	if d.config.DataSource.Database != "" {
		return d.config.DataSource.Database, nil
	}

	foundCatalogs, err := d.queryStringValues(ctx, "SHOW CATALOGS")
	if err != nil || len(foundCatalogs) == 0 {
		return "system", err
	}

	return foundCatalogs[0], nil
}

func extractDatabaseNameFromResourcePath(path string) (string, bool) {
	if path == "" {
		return "", false
	}

	parts := strings.Split(path, "/")
	if len(parts) >= 4 && parts[len(parts)-2] == "databases" {
		return parts[len(parts)-1], true
	}

	return "", false
}
