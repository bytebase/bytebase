package trino

import (
	"context"
	"database/sql"
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

func (d *Driver) processRows(ctx context.Context, query string, handler func(rows *sql.Rows) error) error {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(queryCtx, query)
	if err != nil {
		return err
	}

	defer rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	return handler(rows)
}

func scanStringValues(rows *sql.Rows, logErrorPrefix string) []string {
	var results []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err == nil && value != "" {
			results = append(results, value)
		}
	}

	if err := rows.Err(); err != nil {
		slog.Warn(logErrorPrefix, log.BBError(err))
	}

	return results
}

func (d *Driver) getCatalogList(ctx context.Context) []string {
	if d.config.DataSource.Database != "" {
		return []string{d.config.DataSource.Database}
	}

	var catalogList []string

	// Try SHOW CATALOGS
	err := d.processRows(ctx, "SHOW CATALOGS", func(rows *sql.Rows) error {
		catalogList = scanStringValues(rows, "error iterating catalog rows")
		return nil
	})

	if err == nil && len(catalogList) > 0 {
		return catalogList
	}

	// Try system.metadata.catalogs
	if err := d.processRows(ctx, "SELECT catalog_name FROM system.metadata.catalogs", func(rows *sql.Rows) error {
		catalogList = scanStringValues(rows, "error iterating catalog rows")
		return nil
	}); err != nil {
		slog.Warn("failed to query system.metadata.catalogs", log.BBError(err))
	}

	if len(catalogList) > 0 {
		return catalogList
	}

	return []string{"system"}
}

func (d *Driver) getSchemaList(ctx context.Context, catalog string) []string {
	var schemaList []string

	// Try SHOW SCHEMAS
	query := fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog)
	err := d.processRows(ctx, query, func(rows *sql.Rows) error {
		schemaList = scanStringValues(rows, "error iterating schema rows")
		return nil
	})

	if err == nil && len(schemaList) > 0 {
		return schemaList
	}

	// Try JDBC metadata
	query = fmt.Sprintf("SELECT table_schem FROM system.jdbc.schemas WHERE table_catalog = '%s'", catalog)
	if err := d.processRows(ctx, query, func(rows *sql.Rows) error {
		schemaList = scanStringValues(rows, "error iterating schema rows")
		return nil
	}); err != nil {
		slog.Warn("failed to query schema list", log.BBError(err))
	}

	return schemaList
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

func createSchemaMetadata(schemaList []string) []*storepb.SchemaMetadata {
	var schemas []*storepb.SchemaMetadata
	for _, schema := range schemaList {
		schemas = append(schemas, &storepb.SchemaMetadata{Name: schema})
	}
	return schemas
}

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

	var catalogMetadata []*storepb.DatabaseSchemaMetadata
	catalogList := d.getCatalogList(ctx)

	for _, catalog := range catalogList {
		schemaList := d.getSchemaList(ctx, catalog)
		schemaMetadataList := createSchemaMetadata(schemaList)

		database := &storepb.DatabaseSchemaMetadata{
			Name:    catalog,
			Schemas: schemaMetadataList,
		}

		catalogMetadata = append(catalogMetadata, database)
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

	instanceMetadata := &storepb.Instance{
		SyncDatabases: syncDatabases,
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: catalogMetadata,
		Metadata:  instanceMetadata,
	}, nil
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

func (d *Driver) fetchColumnsForTable(ctx context.Context, catalog, schema, table string) []*storepb.ColumnMetadata {
	var columns []*storepb.ColumnMetadata

	// Try JDBC columns API
	jdbcQuery := fmt.Sprintf(
		"SELECT column_name, type_name as data_type, 'YES' as is_nullable "+
			"FROM system.jdbc.columns "+
			"WHERE table_schem = '%s' AND table_name = '%s'",
		schema, table)

	err := d.processRows(ctx, jdbcQuery, func(rows *sql.Rows) error {
		for rows.Next() {
			var name, dataType, isNullable string
			if err := rows.Scan(&name, &dataType, &isNullable); err == nil {
				columns = append(columns, &storepb.ColumnMetadata{
					Name:     name,
					Type:     dataType,
					Nullable: isNullable == "YES",
				})
			}
		}
		if err := rows.Err(); err != nil {
			slog.Warn("error iterating column rows", log.BBError(err))
		}
		return nil
	})

	if err == nil && len(columns) > 0 {
		return columns
	}

	// Try DESCRIBE as fallback
	describeQueries := []string{
		fmt.Sprintf("DESCRIBE %s.%s.%s", catalog, schema, table),
		fmt.Sprintf("DESCRIBE %s.%s", schema, table),
	}

	for _, query := range describeQueries {
		err = d.processRows(ctx, query, func(rows *sql.Rows) error {
			for rows.Next() {
				var name, dataType, isNullable string
				if err := rows.Scan(&name, &dataType, &isNullable); err == nil {
					columns = append(columns, &storepb.ColumnMetadata{
						Name:     name,
						Type:     dataType,
						Nullable: isNullable == "YES",
					})
				}
			}
			if err := rows.Err(); err != nil {
				slog.Warn("error iterating column rows", log.BBError(err))
			}
			return nil
		})

		if err == nil && len(columns) > 0 {
			break
		}
	}

	return columns
}

func (d *Driver) fetchTablesForSchema(ctx context.Context, catalog, schema string) []*storepb.TableMetadata {
	var tables []*storepb.TableMetadata

	// Try SHOW TABLES first
	queries := []string{
		fmt.Sprintf("SHOW TABLES FROM %s.%s", catalog, schema),
		fmt.Sprintf("SELECT table_name FROM system.jdbc.tables WHERE table_catalog = '%s' AND table_schem = '%s'", catalog, schema),
	}

	for _, query := range queries {
		err := d.processRows(ctx, query, func(rows *sql.Rows) error {
			for rows.Next() {
				var tableName string
				if err := rows.Scan(&tableName); err == nil && tableName != "" {
					table := &storepb.TableMetadata{Name: tableName}
					columns := d.fetchColumnsForTable(ctx, catalog, schema, tableName)
					if len(columns) > 0 {
						table.Columns = columns
					}
					tables = append(tables, table)
				}
			}
			if err := rows.Err(); err != nil {
				slog.Warn("error iterating table rows", log.BBError(err))
			}
			return nil
		})

		if err == nil && len(tables) > 0 {
			break
		}
	}

	return tables
}

func detectSQLCheck(ctx context.Context) (context.Context, bool) {
	// Case 1: Path pattern in name
	if name, ok := ctx.Value(nameKey).(string); ok && name != "" {
		if dbName, found := extractDatabaseNameFromResourcePath(name); found {
			ctx = context.WithValue(ctx, checkKey, true)
			ctx = context.WithValue(ctx, requestNameKey, name)
			ctx = context.WithValue(ctx, explicitDBNameKey, dbName)
			return ctx, true
		}
	}

	// Case 2: Missing database info
	if ctx.Value(databaseKey) == nil && ctx.Value(databaseNameKey) == nil &&
		ctx.Value(resourceIDKey2) == nil && ctx.Value(resourceIDKey) == nil {
		ctx = context.WithValue(ctx, checkKey, true)
		return ctx, true
	}

	// Case 3: Path or statement suggests SQL check
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

	// Override from database path if available
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
	schemaNames := d.getSchemaList(ctx, expectedDBName)

	for _, schemaName := range schemaNames {
		tables := d.fetchTablesForSchema(ctx, expectedDBName, schemaName)
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

	schemaNames := d.getSchemaList(ctx, catalog)

	var schemas []*storepb.SchemaMetadata
	for _, schemaName := range schemaNames {
		tables := d.fetchTablesForSchema(ctx, catalog, schemaName)
		schemas = append(schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tables,
		})
	}

	dbMeta.Schemas = schemas
	return dbMeta, nil
}

func (d *Driver) getDatabaseNameForSync(ctx context.Context) (string, error) {
	// Try context values in priority order
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

	// For SQL checks, try additional sources
	if ctx.Value(checkKey) != nil {
		// Explicit database name has priority
		if explicitDBName, ok := ctx.Value(explicitDBNameKey).(string); ok && explicitDBName != "" {
			return explicitDBName, nil
		}

		// Try request or name contexts
		for _, key := range []contextKey{requestNameKey, nameKey} {
			if reqName, ok := ctx.Value(key).(string); ok && reqName != "" {
				if dbName, found := extractDatabaseNameFromResourcePath(reqName); found {
					return dbName, nil
				}
			}
		}

		// Try configured database
		if d.config.DataSource.Database != "" {
			return d.config.DataSource.Database, nil
		}

		// Try catalog discovery
		var catalogs []string
		if err := d.processRows(ctx, "SHOW CATALOGS", func(rows *sql.Rows) error {
			catalogs = scanStringValues(rows, "error iterating catalog rows")
			return nil
		}); err != nil {
			slog.Warn("failed to query catalogs", log.BBError(err))
		}

		if len(catalogs) > 0 {
			return catalogs[0], nil
		}

		return "system", nil
	}

	// Default to configuration
	if d.config.DataSource.Database != "" {
		return d.config.DataSource.Database, nil
	}

	// Final fallback: query catalogs
	var foundCatalogs []string
	err := d.processRows(ctx, "SHOW CATALOGS", func(rows *sql.Rows) error {
		foundCatalogs = scanStringValues(rows, "error iterating catalog rows")
		return nil
	})

	if err != nil || len(foundCatalogs) == 0 {
		return "system", err
	}

	return foundCatalogs[0], nil
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
