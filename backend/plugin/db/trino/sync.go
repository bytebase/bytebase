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

// Define context key types to avoid lint issues
type contextKey string

const (
	explicitDbNameKey contextKey = "explicitDbName"
	databaseKey       contextKey = "database"
	resourceIDKey     contextKey = "resourceID"
	nameKey           contextKey = "name"
	requestNameKey    contextKey = "requestName"
	databaseNameKey   contextKey = "databaseName"
	checkKey          contextKey = "check"
	pathKey           contextKey = "path"
	statementKey      contextKey = "statement"
	resourceIDKey2    contextKey = "resourceId" // lowercase 'd' to match existing code
)

// SyncInstance syncs the Trino instance metadata.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	if d.db == nil {
		return nil, errors.New("database connection not established")
	}

	// Get Trino version (or fallback to a default)
	version, err := d.getVersion(ctx)
	if err != nil {
		// Log the error but continue with a default version
		slog.Warn("failed to get Trino version", log.BBError(err))
		version = "Trino (version unknown)"
	}

	// Verify connection with a simple test query
	testQuery := "SELECT 1"
	if d.config.DataSource.Database != "" {
		testQuery = fmt.Sprintf("SELECT 1 FROM %s.information_schema.tables WHERE 1=0",
			d.config.DataSource.Database)
	}

	queryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = d.db.ExecContext(queryCtx, testQuery)
	cancel()

	if err != nil {
		// Try fallback query if first attempt fails
		fallbackCtx, fallbackCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = d.db.ExecContext(fallbackCtx, "SHOW CATALOGS")
		fallbackCancel()

		if err != nil {
			return nil, errors.Wrap(err, "failed to execute test query - connection is not functional")
		}
	}

	// This variable is only for collecting catalog info but will NOT be returned
	// to avoid sync_status constraint issues
	var catalogMetadata []*storepb.DatabaseSchemaMetadata

	// Get catalog list
	var catalogList []string

	// If we have a specific catalog configured, use it
	if d.config.DataSource.Database != "" {
		catalogList = append(catalogList, d.config.DataSource.Database)
	} else {
		// Try to query catalogs from the server
		catalogCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		rows, err := d.db.QueryContext(catalogCtx, "SHOW CATALOGS")
		defer cancel()

		if err == nil {
			// Process catalogs
			for rows.Next() {
				var catalog string
				if err := rows.Scan(&catalog); err == nil && catalog != "" {
					catalogList = append(catalogList, catalog)
				}
			}
			// Check for errors from rows iteration
			if err := rows.Err(); err != nil {
				slog.Warn("error iterating catalog rows", log.BBError(err))
			}
			rows.Close()
		}

		// If we couldn't get any catalogs, try alternative query
		if len(catalogList) == 0 {
			catalogCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			rows, err := d.db.QueryContext(catalogCtx, "SELECT catalog_name FROM system.metadata.catalogs")
			defer cancel()

			if err == nil {
				for rows.Next() {
					var catalog string
					if err := rows.Scan(&catalog); err == nil && catalog != "" {
						catalogList = append(catalogList, catalog)
					}
				}
				// Check for errors from rows iteration
				if err := rows.Err(); err != nil {
					slog.Warn("error iterating catalog rows", log.BBError(err))
				}
				rows.Close()
			}
		}

		// If still no catalogs, add system as fallback
		if len(catalogList) == 0 {
			catalogList = append(catalogList, "system")
		}
	}

	// Process each catalog - now working with a simple list rather than rows
	for _, catalog := range catalogList {
		// Create a database metadata entry for this catalog
		database := &storepb.DatabaseSchemaMetadata{
			Name: catalog,
		}

		// Get schemas for this catalog
		var schemaList []string

		// Try standard query first
		schemaCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		rows, err := d.db.QueryContext(schemaCtx, fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog))

		if err == nil {
			for rows.Next() {
				var schema string
				if err := rows.Scan(&schema); err == nil && schema != "" {
					schemaList = append(schemaList, schema)
				}
			}
			// Check for errors from rows iteration
			if err := rows.Err(); err != nil {
				slog.Warn("error iterating schema rows", log.BBError(err))
			}
			rows.Close()
		}
		cancel()

		// If that didn't work, try backup approach
		if len(schemaList) == 0 {
			backupCtx, backupCancel := context.WithTimeout(context.Background(), 5*time.Second)
			rows, err := d.db.QueryContext(backupCtx, fmt.Sprintf(
				"SELECT schema_name FROM %s.information_schema.schemata", catalog))

			if err == nil {
				for rows.Next() {
					var schema string
					if err := rows.Scan(&schema); err == nil && schema != "" {
						schemaList = append(schemaList, schema)
					}
				}
				// Check for errors from rows iteration
				if err := rows.Err(); err != nil {
					slog.Warn("error iterating schema rows", log.BBError(err))
				}
				rows.Close()
			}
			backupCancel()
		}

		// Create schema metadata entries
		var schemaMetadataList []*storepb.SchemaMetadata

		// If no schemas were found, add information_schema as a fallback
		if len(schemaList) == 0 {
			schemaMetadataList = append(schemaMetadataList, &storepb.SchemaMetadata{
				Name: "information_schema",
			})
		} else {
			// Add all the schemas we found
			for _, schema := range schemaList {
				schemaMetadataList = append(schemaMetadataList, &storepb.SchemaMetadata{
					Name: schema,
				})
			}
		}

		// Add the schemas to the database metadata
		database.Schemas = schemaMetadataList

		// Add this database to our catalog metadata collection
		catalogMetadata = append(catalogMetadata, database)
	}

	// Always include at least "system" database for browsing
	// even if no catalogs were found
	if len(catalogMetadata) == 0 {
		// Use a well-known system database that won't be synced
		catalogMetadata = append(catalogMetadata, &storepb.DatabaseSchemaMetadata{
			Name: "system",
			Schemas: []*storepb.SchemaMetadata{
				{Name: "information_schema"},
			},
		})
	}

	// Create instance metadata with sync database information
	// Instead of hardcoding the catalogs to sync, use the ones we found dynamically
	var syncDatabases []string
	for _, db := range catalogMetadata {
		syncDatabases = append(syncDatabases, db.Name)
	}

	instanceMetadata := &storepb.Instance{
		// Explicitly set databases to sync with all the catalogs we found
		// The schemas of these databases will be synced by Bytebase
		SyncDatabases: syncDatabases,
	}

	// Return catalog metadata for display in the UI
	return &db.InstanceMetadata{
		Version:   version,
		Databases: catalogMetadata, // Return the catalogs we found
		Metadata:  instanceMetadata,
	}, nil
}

// extractDatabaseNameFromResourcePath extracts the database name from a resource path
// Handles paths like "instances/instance-id/databases/database-name"
func extractDatabaseNameFromResourcePath(path string) (string, bool) {
	if path == "" {
		return "", false
	}

	parts := strings.Split(path, "/")
	if len(parts) >= 4 && parts[len(parts)-2] == "databases" {
		dbName := parts[len(parts)-1]
		return dbName, true
	}

	return "", false
}

// extractExpectedDatabaseName extracts the database name that Bytebase expects to find
// in its schema metadata storage, which might be different from the Trino catalog name
func extractExpectedDatabaseName(ctx context.Context, defaultName string) string {
	// Check all the common patterns for where Bytebase might have included the expected database name

	// First check if there's an explicit DB name in the context
	if explicitDBName, ok := ctx.Value(explicitDbNameKey).(string); ok && explicitDBName != "" {
		return explicitDBName
	}

	// Second, check resources with full path format: "instances/NAME/databases/DBNAME"

	// Try database context value
	if dbPath, ok := ctx.Value(databaseKey).(string); ok && dbPath != "" {
		if strings.Contains(dbPath, "/databases/") {
			parts := strings.Split(dbPath, "/")
			for i, part := range parts {
				if part == "databases" && i+1 < len(parts) {
					dbName := parts[i+1]
					return dbName
				}
			}
		}
	}

	// Try resourceID context value
	if resourceID, ok := ctx.Value(resourceIDKey).(string); ok && resourceID != "" {
		if strings.Contains(resourceID, "/databases/") {
			parts := strings.Split(resourceID, "/")
			for i, part := range parts {
				if part == "databases" && i+1 < len(parts) {
					dbName := parts[i+1]
					return dbName
				}
			}
		}
	}

	// Try name context value
	if name, ok := ctx.Value(nameKey).(string); ok && name != "" {
		if strings.Contains(name, "/databases/") {
			parts := strings.Split(name, "/")
			for i, part := range parts {
				if part == "databases" && i+1 < len(parts) {
					dbName := parts[i+1]
					return dbName
				}
			}
		}
	}

	// If we can't find it in any of the paths, check if requestName has the database
	if reqName, ok := ctx.Value(requestNameKey).(string); ok && reqName != "" {
		if dbName, found := extractDatabaseNameFromResourcePath(reqName); found {
			return dbName
		}
	}

	// Finally, check databaseName context directly
	if dbName, ok := ctx.Value(databaseNameKey).(string); ok && dbName != "" {
		return dbName
	}

	// If all else fails, use the default name (usually the catalog name)
	return defaultName
}

// SyncDBSchema syncs a single database schema metadata.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	if d.db == nil {
		return nil, errors.New("database connection not established")
	}

	// Detect if this is a SQL check context
	isSQLCheck := false

	// Case 1: Check if this is coming from SQL check with a path pattern in name
	if name, ok := ctx.Value(nameKey).(string); ok && name != "" {
		if dbName, found := extractDatabaseNameFromResourcePath(name); found {
			ctx = context.WithValue(ctx, checkKey, true)
			ctx = context.WithValue(ctx, requestNameKey, name)
			ctx = context.WithValue(ctx, explicitDbNameKey, dbName)
			isSQLCheck = true
		}
	}

	// Case 2: Standard check for missing database info
	if !isSQLCheck &&
		ctx.Value(databaseKey) == nil && ctx.Value(databaseNameKey) == nil &&
		ctx.Value(resourceIDKey2) == nil && ctx.Value(resourceIDKey) == nil {
		ctx = context.WithValue(ctx, checkKey, true)
		isSQLCheck = true
	}

	// Case 3: Check if method name or path suggests SQL check
	if !isSQLCheck {
		if reqPath, ok := ctx.Value(pathKey).(string); ok && reqPath != "" {
			if strings.Contains(reqPath, "sql/check") || strings.Contains(reqPath, "/v1/SQLService/Check") {
				ctx = context.WithValue(ctx, checkKey, true)
				isSQLCheck = true
			}
		}

		if !isSQLCheck && ctx.Value(statementKey) != nil {
			if statement, ok := ctx.Value(statementKey).(string); ok && strings.Contains(strings.ToUpper(statement), "INFORMATION_SCHEMA") {
				ctx = context.WithValue(ctx, checkKey, true)
				isSQLCheck = true
			}
		}
	}

	// Get the database name from the URL
	databaseName, err := d.getDatabaseNameForSync(ctx)
	if err != nil {
		return nil, err
	}

	// For SQL checks, create synthetic metadata structure
	if isSQLCheck {
		// Get expected database name that Bytebase will look for in its storage
		expectedDBName := extractExpectedDatabaseName(ctx, databaseName)

		// Honor the user-selected catalog from the database path
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

		// Try to get all schemas for the synthetic metadata
		var allSchemas []*storepb.SchemaMetadata

		// Now try SHOW SCHEMAS
		schemasQuery := fmt.Sprintf("SHOW SCHEMAS FROM %s", expectedDBName)
		schemaRows, err := d.db.QueryContext(ctx, schemasQuery)

		if err == nil {
			for schemaRows.Next() {
				var schemaName string
				if err := schemaRows.Scan(&schemaName); err == nil && schemaName != "" {
					// For each schema, also try to get its tables
					tablesQuery := fmt.Sprintf("SHOW TABLES FROM %s.%s", expectedDBName, schemaName)
					tableRows, tableErr := d.db.QueryContext(ctx, tablesQuery)

					var tables []*storepb.TableMetadata
					if tableErr == nil {
						// Process tables for this schema
						for tableRows.Next() {
							var tableName string
							if scanErr := tableRows.Scan(&tableName); scanErr == nil && tableName != "" {
								// Create table metadata
								table := &storepb.TableMetadata{
									Name: tableName,
								}

								// Fetch columns using system.jdbc.columns for better performance
								columnsQuery := fmt.Sprintf(
									"SELECT column_name, type_name as data_type, 'YES' as is_nullable "+
										"FROM system.jdbc.columns "+
										"WHERE table_schem = '%s' AND table_name = '%s'",
									schemaName, tableName)

								columnCtx, columnCancel := context.WithTimeout(context.Background(), 30*time.Second)
								columnRows, columnErr := d.db.QueryContext(columnCtx, columnsQuery)
								columnCancel()

								if columnErr == nil {
									var columns []*storepb.ColumnMetadata

									for columnRows.Next() {
										var name, dataType, isNullable string
										if err := columnRows.Scan(&name, &dataType, &isNullable); err == nil {
											columns = append(columns, &storepb.ColumnMetadata{
												Name:     name,
												Type:     dataType,
												Nullable: isNullable == "YES",
											})
										}
									}
									// Check for errors from rows iteration
									if err := columnRows.Err(); err != nil {
										slog.Warn("error iterating column rows", log.BBError(err))
									}
									columnRows.Close()

									if len(columns) > 0 {
										table.Columns = columns
									}
								}

								tables = append(tables, table)
							}
						}
						// Check for errors from rows iteration
						if err := tableRows.Err(); err != nil {
							slog.Warn("error iterating table rows", log.BBError(err))
						}
						tableRows.Close()
					}

					allSchemas = append(allSchemas, &storepb.SchemaMetadata{
						Name:   schemaName,
						Tables: tables,
					})
				}
			}
			// Check for errors from rows iteration
			if err := schemaRows.Err(); err != nil {
				slog.Warn("error iterating schema rows", log.BBError(err))
			}
			schemaRows.Close()
		}

		// Add standard information_schema if needed
		var hasInfoSchema bool
		for _, schema := range allSchemas {
			if schema.Name == "information_schema" {
				hasInfoSchema = true
				break
			}
		}

		if !hasInfoSchema {
			// Add information_schema with standard tables
			allSchemas = append(allSchemas, &storepb.SchemaMetadata{
				Name: "information_schema",
				Tables: []*storepb.TableMetadata{
					{
						Name: "tables",
						Columns: []*storepb.ColumnMetadata{
							{Name: "table_catalog", Type: "varchar", Nullable: false},
							{Name: "table_schema", Type: "varchar", Nullable: false},
							{Name: "table_name", Type: "varchar", Nullable: false},
							{Name: "table_type", Type: "varchar", Nullable: false},
						},
					},
					{
						Name: "columns",
						Columns: []*storepb.ColumnMetadata{
							{Name: "table_catalog", Type: "varchar", Nullable: false},
							{Name: "table_schema", Type: "varchar", Nullable: false},
							{Name: "table_name", Type: "varchar", Nullable: false},
							{Name: "column_name", Type: "varchar", Nullable: false},
							{Name: "data_type", Type: "varchar", Nullable: false},
							{Name: "is_nullable", Type: "varchar", Nullable: false},
						},
					},
				},
			})
		}

		// Create database metadata with all discovered schemas
		return &storepb.DatabaseSchemaMetadata{
			Name:    expectedDBName,
			Schemas: allSchemas,
		}, nil
	}

	// Normal flow for actual schema sync (not SQL check)
	catalog := databaseName
	dbMeta := &storepb.DatabaseSchemaMetadata{
		Name: catalog,
	}

	// Get schemas in this catalog
	schemasQuery := fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog)
	schemaRows, err := d.db.QueryContext(ctx, schemasQuery)
	if err != nil {
		// Try an alternative query if the first one fails
		altQuery := fmt.Sprintf("SELECT schema_name FROM %s.information_schema.schemata", catalog)
		schemaRows, err = d.db.QueryContext(ctx, altQuery)

		if err != nil {
			return nil, errors.Wrapf(err, "failed to list schemas for catalog %s", catalog)
		}
	}
	defer schemaRows.Close()

	// Process each schema
	var schemas []*storepb.SchemaMetadata
	for schemaRows.Next() {
		var schemaName string
		if err := schemaRows.Scan(&schemaName); err != nil {
			return nil, errors.Wrap(err, "failed to scan schema name")
		}

		// Create schema metadata
		schema := &storepb.SchemaMetadata{
			Name: schemaName,
		}

		// Get tables for this schema using SHOW TABLES command
		tablesQuery := fmt.Sprintf("SHOW TABLES FROM %s.%s", catalog, schemaName)
		tableCtx, tableCancel := context.WithTimeout(ctx, 5*time.Second)
		tableRows, err := d.db.QueryContext(tableCtx, tablesQuery)
		tableCancel()

		if err != nil {
			// Try alternative query through information_schema
			alt1Query := fmt.Sprintf("SELECT table_name FROM %s.information_schema.tables WHERE table_schema = '%s'",
				catalog, schemaName)

			tableCtx2, tableCancel2 := context.WithTimeout(ctx, 5*time.Second)
			tableRows, err = d.db.QueryContext(tableCtx2, alt1Query)
			tableCancel2()

			if err != nil {
				slog.Warn("failed to list tables for schema",
					log.BBError(err),
					slog.String("catalog", catalog),
					slog.String("schema", schemaName))
				schemas = append(schemas, schema) // Add schema even without tables
				continue
			}
		}

		// Process tables and properly close the rows
		func() {
			defer tableRows.Close()
			var tables []*storepb.TableMetadata

			for tableRows.Next() {
				var tableName string
				if err := tableRows.Scan(&tableName); err != nil {
					slog.Warn("failed to scan table name",
						log.BBError(err),
						slog.String("catalog", catalog),
						slog.String("schema", schemaName))
					continue
				}

				// Create table metadata
				table := &storepb.TableMetadata{
					Name: tableName,
				}

				// First try to get columns using system.jdbc.columns for better performance
				// This is the key improvement we made to fix column discovery
				jdbcColumnsQuery := fmt.Sprintf(
					"SELECT column_name, type_name as data_type, 'YES' as is_nullable "+
						"FROM system.jdbc.columns "+
						"WHERE table_schem = '%s' AND table_name = '%s'",
					schemaName, tableName)

				jdbcColumnRows, jdbcErr := d.db.QueryContext(ctx, jdbcColumnsQuery)

				if jdbcErr == nil {
					var columns []*storepb.ColumnMetadata

					for jdbcColumnRows.Next() {
						var name, dataType, isNullable string
						if scanErr := jdbcColumnRows.Scan(&name, &dataType, &isNullable); scanErr == nil {
							columns = append(columns, &storepb.ColumnMetadata{
								Name:     name,
								Type:     dataType,
								Nullable: isNullable == "YES",
							})
						}
					}
					// Check for errors from rows iteration
					if err := jdbcColumnRows.Err(); err != nil {
						slog.Warn("error iterating JDBC column rows", log.BBError(err))
					}
					jdbcColumnRows.Close()

					if len(columns) > 0 {
						table.Columns = columns
						tables = append(tables, table)
						continue
					}
				}

				// Fall back to information_schema.columns approach if system.jdbc.columns fails
				columnsQuery := fmt.Sprintf(
					"SELECT column_name, data_type, is_nullable FROM %s.information_schema.columns "+
						"WHERE table_catalog = '%s' AND table_schema = '%s' AND table_name = '%s'",
					catalog, catalog, schemaName, tableName)

				columnRows, err := d.db.QueryContext(ctx, columnsQuery)
				if err != nil {
					// If still failing, try a broader query without catalog filter
					altQuery := fmt.Sprintf(
						"SELECT column_name, data_type, is_nullable FROM %s.information_schema.columns "+
							"WHERE table_schema = '%s' AND table_name = '%s'",
						catalog, schemaName, tableName)

					columnRows, err = d.db.QueryContext(ctx, altQuery)
					if err != nil {
						// Add table even without columns
						tables = append(tables, table)
						continue
					}
				}

				// Process columns
				func() {
					defer columnRows.Close()
					var columns []*storepb.ColumnMetadata

					for columnRows.Next() {
						var name, dataType, isNullable string
						if err := columnRows.Scan(&name, &dataType, &isNullable); err != nil {
							continue
						}

						columns = append(columns, &storepb.ColumnMetadata{
							Name:     name,
							Type:     dataType,
							Nullable: isNullable == "YES",
						})
					}
					// Check for errors from rows iteration
					if err := columnRows.Err(); err != nil {
						slog.Warn("error iterating column rows", log.BBError(err))
					}

					if len(columns) > 0 {
						table.Columns = columns
					}
				}()

				tables = append(tables, table)
			}
			// Check for errors from rows iteration
			if err := tableRows.Err(); err != nil {
				slog.Warn("error iterating table rows", log.BBError(err))
			}

			schema.Tables = tables
		}()

		schemas = append(schemas, schema)
	}

	// Check for errors from iterating schema rows
	if err := schemaRows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating schema rows")
	}

	dbMeta.Schemas = schemas

	// If this is a SQL check context, ensure we have standard structure with information_schema
	if ctx.Value(checkKey) != nil {
		// Ensure we have information_schema with standard tables
		var hasInfoSchema bool
		for _, schema := range dbMeta.Schemas {
			if schema.Name == "information_schema" {
				hasInfoSchema = true
				break
			}
		}

		// If we don't have information_schema, try to discover it
		if !hasInfoSchema {
			// Try to query for information_schema tables
			infoSchemaQuery := fmt.Sprintf("SHOW TABLES FROM %s.information_schema", catalog)
			infoTablesCtx, infoTablesCancel := context.WithTimeout(ctx, 5*time.Second)
			infoTablesRows, infoErr := d.db.QueryContext(infoTablesCtx, infoSchemaQuery)
			infoTablesCancel()

			var infoTables []*storepb.TableMetadata
			if infoErr == nil {
				for infoTablesRows.Next() {
					var tableName string
					if scanErr := infoTablesRows.Scan(&tableName); scanErr == nil && tableName != "" {
						infoTables = append(infoTables, &storepb.TableMetadata{
							Name: tableName,
						})
					}
				}
				// Check for errors from rows iteration
				if err := infoTablesRows.Err(); err != nil {
					slog.Warn("error iterating information_schema tables rows", log.BBError(err))
				}
				infoTablesRows.Close()
			}

			// Add information_schema even if empty - tables will be discovered as needed
			dbMeta.Schemas = append(dbMeta.Schemas, &storepb.SchemaMetadata{
				Name:   "information_schema",
				Tables: infoTables,
			})
		}
	}

	return dbMeta, nil
}

// getDatabaseNameForSync gets the database name for schema syncing.
// This handles the case where no catalog is specified in the connection configuration.
func (d *Driver) getDatabaseNameForSync(ctx context.Context) (string, error) {
	// First check if a resourceID is in the context
	resourceID, ok := ctx.Value(resourceIDKey).(string)
	if ok && resourceID != "" {
		if dbName, found := extractDatabaseNameFromResourcePath(resourceID); found {
			return dbName, nil
		}
	}

	// Check if database name is in context directly as "database"
	if ctx.Value(databaseKey) != nil {
		dbName, ok := ctx.Value(databaseKey).(string)
		if ok && dbName != "" {
			// First check if it's a path like "instances/instance-id/databases/database-name"
			if dbResult, found := extractDatabaseNameFromResourcePath(dbName); found {
				return dbResult, nil
			}

			// Otherwise, just use the last part of the path
			parts := strings.Split(dbName, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				return lastPart, nil
			}
		}
	}

	// Check if databaseName is in context
	if ctx.Value(databaseNameKey) != nil {
		dbName, ok := ctx.Value(databaseNameKey).(string)
		if ok && dbName != "" {
			return dbName, nil
		}
	}

	// For SQL checks, we need to handle special cases
	if ctx.Value(checkKey) != nil {
		// First priority: check if we have an explicitly provided database name
		if explicitDbName, ok := ctx.Value(explicitDbNameKey).(string); ok && explicitDbName != "" {
			return explicitDbName, nil
		}

		// Second priority: Look for request name pattern
		if reqName, ok := ctx.Value(requestNameKey).(string); ok && reqName != "" {
			if dbName, found := extractDatabaseNameFromResourcePath(reqName); found {
				return dbName, nil
			}
		}

		// Third priority: Look for name context value
		if reqName, ok := ctx.Value(nameKey).(string); ok && reqName != "" {
			if dbName, found := extractDatabaseNameFromResourcePath(reqName); found {
				return dbName, nil
			}
		}

		// Try to use the user-selected catalog from the config if available
		if d.config.DataSource.Database != "" {
			return d.config.DataSource.Database, nil
		}

		// Otherwise try catalog discovery
		catalogRows, catalogErr := d.db.QueryContext(ctx, "SHOW CATALOGS")
		if catalogErr == nil {
			defer catalogRows.Close()
			var catalogs []string
			for catalogRows.Next() {
				var catalog string
				if scanErr := catalogRows.Scan(&catalog); scanErr == nil && catalog != "" {
					catalogs = append(catalogs, catalog)
				}
			}
			// Check for errors from rows iteration
			if err := catalogRows.Err(); err != nil {
				slog.Warn("error iterating catalog rows", log.BBError(err))
			}

			if len(catalogs) > 0 {
				return catalogs[0], nil
			}
		}

		// Final fallback - use system as it's guaranteed to exist in all Trino installations
		return "system", nil
	}

	// Attempt to extract directly from the driver configuration
	if d.config.DataSource.Database != "" {
		return d.config.DataSource.Database, nil
	}

	// Try to find a suitable catalog as fallback
	rows, err := d.db.QueryContext(ctx, "SHOW CATALOGS")
	if err != nil {
		return "system", nil // Default to system catalog which is more likely to exist in Trino
	}
	defer rows.Close()

	// Keep track of found catalogs
	var foundCatalogs []string
	for rows.Next() {
		var catalogName string
		if err := rows.Scan(&catalogName); err != nil {
			continue
		}

		// Skip empty catalog names
		if catalogName == "" {
			continue
		}

		foundCatalogs = append(foundCatalogs, catalogName)

		// Some preferred catalogs that are commonly available in Trino
		if catalogName == "system" || catalogName == "tpch" || catalogName == "hive" || catalogName == "mysql" {
			return catalogName, nil
		}
	}
	// Check for errors from rows iteration
	if err := rows.Err(); err != nil {
		slog.Warn("error iterating catalog rows", log.BBError(err))
	}

	// If we found any catalogs, use the first one
	if len(foundCatalogs) > 0 {
		return foundCatalogs[0], nil
	}

	// If all else fails, default to system catalog which is standard in Trino
	return "system", nil
}

// getVersion gets the Trino server version.
func (d *Driver) getVersion(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", errors.New("database connection not established")
	}

	// Prioritize catalog-specific version query if we have a catalog
	if d.config.DataSource.Database != "" {
		versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		var version string
		err := d.db.QueryRowContext(versionCtx, fmt.Sprintf(
			"SELECT VERSION() FROM %s.information_schema.tables WHERE 1=0",
			d.config.DataSource.Database)).Scan(&version)
		cancel()
		if err == nil {
			return version, nil
		}
	}

	// Try standard version queries
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

	// Return a default value if all queries failed
	return "Trino (version unknown)", nil
}
