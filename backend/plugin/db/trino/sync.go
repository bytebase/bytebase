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
	if d.db == nil {
		return nil, errors.New("database connection not established")
	}

	version, err := d.getVersion(ctx)
	if err != nil {
		slog.Warn("failed to get Trino version", log.BBError(err))
		version = "Trino (version unknown)"
	}

	if err := d.verifyConnection(ctx); err != nil {
		return nil, err
	}

	catalogList, err := d.getCatalogList(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get catalog list")
	}

	var catalogMetadata []*storepb.DatabaseSchemaMetadata
	for _, catalog := range catalogList {
		schemaList, err := d.getSchemaList(ctx, catalog)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get schema list for catalog %s", catalog)
		}

		catalogMetadata = append(catalogMetadata, &storepb.DatabaseSchemaMetadata{
			Name:    catalog,
			Schemas: createSchemaMetadata(schemaList),
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
	if d.db == nil {
		return nil, errors.New("database connection not established")
	}

	ctx, isSQLCheck := detectSQLCheck(ctx)

	databaseName, err := d.getDatabaseNameForSync(ctx)
	if err != nil {
		return nil, err
	}

	if isSQLCheck {
		return d.processSQLCheck(ctx, databaseName)
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
		tables, err := d.fetchTablesForSchema(ctx, catalog, schemaName)
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

func (d *Driver) verifyConnection(ctx context.Context) error {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_, err := d.db.ExecContext(queryCtx, "SELECT 1")
	cancel()

	if err == nil {
		return nil
	}

	fallbackCtx, fallbackCancel := context.WithTimeout(ctx, 5*time.Second)
	_, err = d.db.ExecContext(fallbackCtx, "SHOW CATALOGS")
	fallbackCancel()

	if err != nil {
		return errors.Wrap(err, "connection not functional")
	}

	return nil
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", errors.New("database connection not established")
	}

	queries := []string{
		"SELECT VERSION()",
		"SELECT node_version FROM system.runtime.nodes LIMIT 1",
		"SELECT version FROM system.runtime.nodes LIMIT 1",
		"SELECT query_engine_version FROM system.metadata.query_engine LIMIT 1",
	}

	for _, query := range queries {
		versionCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		var version string
		err := d.db.QueryRowContext(versionCtx, query).Scan(&version)
		cancel()
		if err == nil {
			return version, nil
		}
	}

	return "Trino (version unknown)", nil
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
	if d.config.DataSource.Database != "" {
		return []string{d.config.DataSource.Database}, nil
	}

	catalogList, err := d.queryStringValues(ctx, "SHOW CATALOGS")
	if err == nil && len(catalogList) > 0 {
		return catalogList, nil
	}

	catalogList, err = d.queryStringValues(ctx, "SELECT catalog_name FROM system.metadata.catalogs")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query system.metadata.catalogs")
	}

	if len(catalogList) > 0 {
		return catalogList, nil
	}

	return []string{"system"}, nil
}

func (d *Driver) getSchemaList(ctx context.Context, catalog string) ([]string, error) {
	query := fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog)
	schemaList, err := d.queryStringValues(ctx, query)
	if err == nil && len(schemaList) > 0 {
		return schemaList, nil
	}

	query = fmt.Sprintf("SELECT table_schem FROM system.jdbc.schemas WHERE table_catalog = '%s'", catalog)
	schemaList, err = d.queryStringValues(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query schema list")
	}

	return schemaList, nil
}

func createSchemaMetadata(schemaList []string) []*storepb.SchemaMetadata {
	var schemas []*storepb.SchemaMetadata
	for _, schema := range schemaList {
		schemas = append(schemas, &storepb.SchemaMetadata{Name: schema})
	}
	return schemas
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

func (d *Driver) fetchColumnsForTable(ctx context.Context, catalog, schema, table string) ([]*storepb.ColumnMetadata, error) {
	jdbcQuery := fmt.Sprintf(
		"SELECT column_name, type_name as data_type, 'YES' as is_nullable "+
			"FROM system.jdbc.columns "+
			"WHERE table_schem = '%s' AND table_name = '%s'",
		schema, table)

	columns, err := d.queryColumns(ctx, jdbcQuery)
	if err == nil && len(columns) > 0 {
		return columns, nil
	}

	describeQueries := []string{
		fmt.Sprintf("DESCRIBE %s.%s.%s", catalog, schema, table),
		fmt.Sprintf("DESCRIBE %s.%s", schema, table),
	}

	var lastErr error
	for _, query := range describeQueries {
		columns, err = d.queryColumns(ctx, query)
		if err == nil && len(columns) > 0 {
			return columns, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, errors.Wrap(lastErr, "failed to fetch columns for table")
	}

	return []*storepb.ColumnMetadata{}, nil
}

func (d *Driver) fetchTablesForSchema(ctx context.Context, catalog, schema string) ([]*storepb.TableMetadata, error) {
	var tables []*storepb.TableMetadata

	queries := []string{
		fmt.Sprintf("SHOW TABLES FROM %s.%s", catalog, schema),
		fmt.Sprintf("SELECT table_name FROM system.jdbc.tables WHERE table_schem = '%s'", schema),
	}

	var lastErr error
	for _, query := range queries {
		tableNames, err := d.queryStringValues(ctx, query)
		if err != nil {
			lastErr = err
			continue
		}

		for _, tableName := range tableNames {
			if tableName != "" {
				table := &storepb.TableMetadata{Name: tableName}
				columns, err := d.fetchColumnsForTable(ctx, catalog, schema, tableName)
				if err != nil {
					slog.Debug("failed to fetch columns for table",
						slog.String("catalog", catalog),
						slog.String("schema", schema),
						slog.String("table", tableName),
						log.BBError(err))
				} else if len(columns) > 0 {
					table.Columns = columns
				}
				tables = append(tables, table)
			}
		}

		if len(tables) > 0 {
			return tables, nil
		}
	}

	if lastErr != nil && len(tables) == 0 {
		return nil, errors.Wrap(lastErr, "failed to fetch tables for schema")
	}

	return tables, nil
}

func detectSQLCheck(ctx context.Context) (context.Context, bool) {
	if name, ok := ctx.Value(nameKey).(string); ok && name != "" {
		if dbName, found := extractDatabaseNameFromResourcePath(name); found {
			ctx = context.WithValue(ctx, checkKey, true)
			ctx = context.WithValue(ctx, requestNameKey, name)
			ctx = context.WithValue(ctx, explicitDBNameKey, dbName)
			return ctx, true
		}
	}

	if ctx.Value(databaseKey) == nil && ctx.Value(databaseNameKey) == nil &&
		ctx.Value(resourceIDKey2) == nil && ctx.Value(resourceIDKey) == nil {
		ctx = context.WithValue(ctx, checkKey, true)
		return ctx, true
	}

	if reqPath, ok := ctx.Value(pathKey).(string); ok &&
		(strings.Contains(reqPath, "sql/check") || strings.Contains(reqPath, "/v1/SQLService/Check")) {
		ctx = context.WithValue(ctx, checkKey, true)
		return ctx, true
	}

	if statement, ok := ctx.Value(statementKey).(string); ok &&
		strings.Contains(strings.ToUpper(statement), "INFORMATION_SCHEMA") {
		ctx = context.WithValue(ctx, checkKey, true)
		return ctx, true
	}

	return ctx, false
}

func (d *Driver) processSQLCheck(ctx context.Context, databaseName string) (*storepb.DatabaseSchemaMetadata, error) {
	expectedDBName := extractExpectedDatabaseName(ctx, databaseName)

	if dbPathStr := fmt.Sprintf("%v", ctx.Value(databaseKey)); dbPathStr != "<nil>" && dbPathStr != "" {
		if dbPathParts := strings.Split(dbPathStr, "/"); len(dbPathParts) > 0 {
			for i, part := range dbPathParts {
				if part == "databases" && i+1 < len(dbPathParts) {
					expectedDBName = dbPathParts[i+1]
					break
				}
			}
		}
	}

	var schemas []*storepb.SchemaMetadata
	schemaNames, err := d.getSchemaList(ctx, expectedDBName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schema list for SQL check on catalog %s", expectedDBName)
	}

	for _, schemaName := range schemaNames {
		tables, err := d.fetchTablesForSchema(ctx, expectedDBName, schemaName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to fetch tables for schema %s during SQL check", schemaName)
		}

		schemas = append(schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tables,
		})
	}

	return &storepb.DatabaseSchemaMetadata{
		Name:    expectedDBName,
		Schemas: schemas,
	}, nil
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

func extractExpectedDatabaseName(ctx context.Context, defaultName string) string {
	if explicitDBName, ok := ctx.Value(explicitDBNameKey).(string); ok && explicitDBName != "" {
		return explicitDBName
	}

	for _, key := range []contextKey{databaseKey, resourceIDKey, nameKey} {
		if value, ok := ctx.Value(key).(string); ok && value != "" {
			if strings.Contains(value, "/databases/") {
				parts := strings.Split(value, "/")
				for i, part := range parts {
					if part == "databases" && i+1 < len(parts) {
						return parts[i+1]
					}
				}
			}
		}
	}

	if reqName, ok := ctx.Value(requestNameKey).(string); ok && reqName != "" {
		if dbName, found := extractDatabaseNameFromResourcePath(reqName); found {
			return dbName
		}
	}

	if dbName, ok := ctx.Value(databaseNameKey).(string); ok && dbName != "" {
		return dbName
	}

	return defaultName
}
