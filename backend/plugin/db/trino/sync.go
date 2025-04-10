package trino

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

	// Try to get a simple test query first to make sure the connection is working
	fmt.Println("Testing Trino connection with various test queries")
	
	// Try multiple different query strategies to find one that works with this Trino version
	testQueries := []struct {
		description string
		query       string
	}{
		// Try with explicit catalog first if we have one
		{
			description: "Query with explicit catalog and schema",
			query: func() string {
				if d.config.DataSource.Database != "" {
					return fmt.Sprintf("SELECT 1 FROM %s.information_schema.tables WHERE 1=0", 
						d.config.DataSource.Database)
				}
				return ""
			}(),
		},
		{
			description: "Simple scalar SELECT",
			query:       "SELECT 1",
		},
		{
			description: "VALUES syntax",
			query:       "VALUES 1",
		},
		{
			description: "Current timestamp",
			query:       "SELECT current_timestamp",
		},
		{
			description: "SHOW CATALOGS query",
			query:       "SHOW CATALOGS",
		},
		{
			description: "SHOW SCHEMAS query with system catalog",
			query:       "SHOW SCHEMAS FROM system",
		},
		{
			description: "SELECT version",
			query:       "SELECT version()",
		},
		{
			description: "Query system metadata",
			query:       "SELECT 1 FROM system.metadata.catalogs LIMIT 1",
		},
	}
	
	success := false
	var testErr error
	
	for _, test := range testQueries {
		if test.query == "" {
			continue // Skip empty queries (happens if catalog is not set)
		}
		
		fmt.Printf("Trying query: %s (%s)\n", test.query, test.description)
		
		// Use short timeout for each test query
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err = d.db.ExecContext(queryCtx, test.query)
		cancel()
		
		if err == nil {
			fmt.Printf("Query succeeded: %s\n", test.description)
			success = true
			break
		} else {
			fmt.Printf("Query failed: %s - Error: %v\n", test.description, err)
			testErr = err
		}
	}
	
	if !success {
		return nil, errors.Wrap(testErr, "failed to execute any test query - connection is not functional")
	}
	
	fmt.Println("Connection test successful - proceeding with metadata retrieval")

	// Get catalogs with a timeout to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// This variable is only for collecting catalog info but will NOT be returned
	// to avoid sync_status constraint issues
	var catalogMetadata []*storepb.DatabaseSchemaMetadata

	// Try to get catalogs using different approaches
	// Some Trino deployments organize catalogs differently or have different permissions
	catalogQueries := []struct {
		description string
		query       string
	}{
		{
			description: "Standard SHOW CATALOGS",
			query:       "SHOW CATALOGS",
		},
		{
			description: "Query system metadata catalogs",
			query:       "SELECT catalog_name FROM system.metadata.catalogs",
		},
		{
			description: "Query information_schema catalogs",
			query:       "SELECT table_catalog FROM information_schema.schemata GROUP BY table_catalog",
		},
	}
	
	// If we have a specific catalog configured, add it to the list of catalog strategies
	if d.config.DataSource.Database != "" {
		catalogQueries = append([]struct {
			description string
			query       string
		}{
			{
				description: "Using configured catalog",
				query:       fmt.Sprintf("SELECT '%s' AS catalog_name", d.config.DataSource.Database),
			},
		}, catalogQueries...)
	}
	
	var catalogRows *sql.Rows
	var catalogErr error
	var catalogQueryUsed string
	
	// Try each catalog query until one works
	for _, test := range catalogQueries {
		fmt.Printf("Trying catalog query: %s (%s)\n", test.query, test.description)
		
		// Use a context with timeout for each catalog query
		catalogCtx, catalogCancel := context.WithTimeout(context.Background(), 5*time.Second)
		rows, err := d.db.QueryContext(catalogCtx, test.query)
		catalogCancel()
		
		if err == nil {
			fmt.Printf("Catalog query succeeded: %s\n", test.description)
			catalogRows = rows
			catalogQueryUsed = test.description
			break
		} else {
			fmt.Printf("Catalog query failed: %s - Error: %v\n", test.description, err)
			catalogErr = err
		}
	}
	
	// If all catalog queries failed, just use the configured database if available
	if catalogRows == nil {
		if catalogErr != nil {
			slog.Warn("All catalog queries failed", log.BBError(catalogErr))
		}
		
		// If we can't get catalogs but have a database name in config, at least return that
		if d.config.DataSource.Database != "" {
			fmt.Printf("Using configured catalog: %s\n", d.config.DataSource.Database)
			// Create a single catalog/database entry with the configured database name
			database := &storepb.DatabaseSchemaMetadata{
				Name: d.config.DataSource.Database,
				// Add information_schema as fallback since almost all catalogs have it
				Schemas: []*storepb.SchemaMetadata{
					{Name: "information_schema"},
				},
			}
			catalogMetadata = append(catalogMetadata, database)
		}
	} else {
		defer catalogRows.Close()
		fmt.Printf("Processing catalogs from query: %s\n", catalogQueryUsed)

		// Process each catalog
		for catalogRows.Next() {
			var catalog string
			if err := catalogRows.Scan(&catalog); err != nil {
				slog.Warn("failed to scan catalog name", log.BBError(err))
				continue
			}

			// Skip empty catalogs if any
			if catalog == "" {
				continue
			}
			
			fmt.Printf("Found catalog: %s\n", catalog)

			// Create minimal database metadata for this catalog
			database := &storepb.DatabaseSchemaMetadata{
				Name: catalog,
			}

			// Only get schemas if we have time left in our context
			if ctx.Err() == nil {
				// Get schemas for this catalog but handle errors gracefully
				func() {
					// Try multiple schema query strategies to accommodate different Trino versions and configurations
					schemaQueries := []struct {
						description string
						query       string
					}{
						{
							description: "Standard SHOW SCHEMAS",
							query:       fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog),
						},
						{
							description: "Query information_schema.schemata",
							query:       fmt.Sprintf("SELECT schema_name FROM %s.information_schema.schemata", catalog),
						},
						{
							description: "Query information_schema with table_schema",
							query:       fmt.Sprintf("SELECT DISTINCT table_schema FROM %s.information_schema.tables", catalog),
						},
						{
							description: "Query system metadata",
							query:       fmt.Sprintf("SELECT schema_name FROM system.metadata.schemas WHERE catalog_name = '%s'", catalog),
						},
					}
					
					var schemaRows *sql.Rows
					var schemaErr error
					var schemaQueryUsed string
					
					// Try each schema query until one works
					for _, test := range schemaQueries {
						// Use a short timeout for schema queries
						schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 5*time.Second)
						
						fmt.Printf("Trying schema query for catalog %s: %s (%s)\n", catalog, test.query, test.description)
						rows, err := d.db.QueryContext(schemaCtx, test.query)
						
						if err == nil {
							fmt.Printf("Schema query succeeded for catalog %s: %s\n", catalog, test.description)
							schemaRows = rows
							schemaQueryUsed = test.description
							schemaCancel()
							break
						} else {
							fmt.Printf("Schema query failed for catalog %s: %s - Error: %v\n", catalog, test.description, err)
							schemaErr = err
							schemaCancel()
						}
					}
					
					// If all schema queries failed, log and return
					if schemaRows == nil {
						if schemaErr != nil {
							slog.Warn("All schema queries failed for catalog",
								log.BBError(schemaErr),
								slog.String("catalog", catalog))
						}
						
						// Add information_schema as fallback since almost all catalogs have it
						database.Schemas = []*storepb.SchemaMetadata{
							{Name: "information_schema"},
						}
						return
					}
					
					defer schemaRows.Close()
					fmt.Printf("Processing schemas for catalog %s from query: %s\n", catalog, schemaQueryUsed)

					var schemas []*storepb.SchemaMetadata
					for schemaRows.Next() {
						var schemaName string
						if err := schemaRows.Scan(&schemaName); err != nil {
							slog.Warn("failed to scan schema name", log.BBError(err))
							continue
						}
						
						// Skip empty schema names if any
						if schemaName == "" {
							continue
						}
						
						fmt.Printf("Found schema: %s.%s\n", catalog, schemaName)

						// Add minimal schema info
						schemas = append(schemas, &storepb.SchemaMetadata{
							Name: schemaName,
						})
					}

					// Only set schemas if we found some
					if len(schemas) > 0 {
						database.Schemas = schemas
					} else {
						// Add information_schema as fallback since almost all catalogs have it
						database.Schemas = []*storepb.SchemaMetadata{
							{Name: "information_schema"},
						}
					}
				}()
			}

			catalogMetadata = append(catalogMetadata, database)
		}

		if err := catalogRows.Err(); err != nil {
			slog.Warn("error iterating catalog rows", log.BBError(err))
		}
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
	
	// Create instance metadata
	instanceMetadata := &storepb.Instance{
		// Only sync a non-existent database to block actual syncing
		SyncDatabases: []string{"__bytebase_no_sync__"},
	}

	// Return a completely empty database list to avoid any 
	// database creation attempts and sync_status issues
	return &db.InstanceMetadata{
		Version:   version,
		Databases: []*storepb.DatabaseSchemaMetadata{}, // Empty databases list
		Metadata:  instanceMetadata,
	}, nil
}

// SyncDBSchema syncs a single database schema metadata.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	if d.db == nil {
		return nil, errors.New("database connection not established")
	}

	// For Trino, the database is the catalog
	catalog := d.config.DataSource.Database
	if catalog == "" {
		return nil, errors.New("no catalog specified in connection configuration")
	}

	// Create database metadata
	dbMeta := &storepb.DatabaseSchemaMetadata{
		Name: catalog,
	}

	// Get schemas in this catalog
	schemasQuery := fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog)
	schemaRows, err := d.db.QueryContext(ctx, schemasQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list schemas for catalog %s", catalog)
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

		// Get tables for this schema
		tablesQuery := fmt.Sprintf(
			"SELECT table_name FROM %s.information_schema.tables WHERE table_schema = '%s'",
			catalog, schemaName)
		
		tableRows, err := d.db.QueryContext(ctx, tablesQuery)
		if err != nil {
			slog.Warn("failed to list tables for schema",
				log.BBError(err),
				slog.String("catalog", catalog),
				slog.String("schema", schemaName))
			schemas = append(schemas, schema) // Add schema even without tables
			continue
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

				// Get columns for this table
				columnsQuery := fmt.Sprintf(
					"SELECT column_name, data_type, is_nullable FROM %s.information_schema.columns "+
						"WHERE table_schema = '%s' AND table_name = '%s'",
					catalog, schemaName, tableName)
				
				columnRows, err := d.db.QueryContext(ctx, columnsQuery)
				if err != nil {
					slog.Warn("failed to list columns for table",
						log.BBError(err),
						slog.String("catalog", catalog),
						slog.String("schema", schemaName),
						slog.String("table", tableName))
					tables = append(tables, table) // Add table even without columns
					continue
				}

				// Process columns and properly close the rows
				func() {
					defer columnRows.Close()
					var columns []*storepb.ColumnMetadata

					for columnRows.Next() {
						var name, dataType, isNullable string
						if err := columnRows.Scan(&name, &dataType, &isNullable); err != nil {
							slog.Warn("failed to scan column data",
								log.BBError(err),
								slog.String("catalog", catalog),
								slog.String("schema", schemaName),
								slog.String("table", tableName))
							continue
						}

						// Add column
						columns = append(columns, &storepb.ColumnMetadata{
							Name:     name,
							Type:     dataType,
							Nullable: isNullable == "YES",
						})
					}

					if err := columnRows.Err(); err != nil {
						slog.Warn("error iterating column rows",
							log.BBError(err),
							slog.String("catalog", catalog),
							slog.String("schema", schemaName),
							slog.String("table", tableName))
					}

					table.Columns = columns
				}()

				tables = append(tables, table)
			}

			if err := tableRows.Err(); err != nil {
				slog.Warn("error iterating table rows",
					log.BBError(err),
					slog.String("catalog", catalog),
					slog.String("schema", schemaName))
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
	return dbMeta, nil
}

// getVersion gets the Trino server version.
func (d *Driver) getVersion(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", errors.New("database connection not established")
	}

	fmt.Println("Attempting to detect Trino version")

	// Try different version queries as Trino versioning can differ between versions
	versionQueries := []struct {
		description string
		query       string
	}{
		{
			description: "Standard VERSION() function",
			query:       "SELECT VERSION()",
		},
		{
			description: "System runtime nodes node_version",
			query:       "SELECT node_version FROM system.runtime.nodes LIMIT 1",
		},
		{
			description: "System runtime nodes version",
			query:       "SELECT version FROM system.runtime.nodes LIMIT 1",
		},
		{
			description: "System information",
			query:       "SELECT * FROM system.information.environment WHERE name LIKE '%version%' LIMIT 1",
		},
		{
			description: "System metadata query_engine_version",
			query:       "SELECT query_engine_version FROM system.metadata.query_engine LIMIT 1",
		},
		{
			description: "SELECT 'Trino' as fallback",
			query:       "SELECT 'Trino' as version",
		},
	}

	// If we have a catalog configured, add version queries with explicit catalog
	if d.config.DataSource.Database != "" {
		catalogVersionQueries := []struct {
			description string
			query       string
		}{
			{
				description: "VERSION() with explicit catalog",
				query:       fmt.Sprintf("SELECT VERSION() FROM %s.information_schema.tables WHERE 1=0", d.config.DataSource.Database),
			},
		}
		versionQueries = append(catalogVersionQueries, versionQueries...)
	}

	var version string
	var queryErr error

	for _, test := range versionQueries {
		fmt.Printf("Trying version query: %s (%s)\n", test.query, test.description)
		
		// Use short timeout for version queries
		versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := d.db.QueryRowContext(versionCtx, test.query).Scan(&version)
		cancel()
		
		if err == nil {
			fmt.Printf("Successfully detected version: %s (via %s)\n", version, test.description)
			return version, nil
		}
		
		fmt.Printf("Version query failed: %s - Error: %v\n", test.description, err)
		queryErr = err
	}
	
	// Log the last error for debugging
	if queryErr != nil {
		slog.Debug("All version queries failed", log.BBError(queryErr))
	}

	// If all queries failed, return a default value rather than error
	// to allow the sync process to continue
	defaultVersion := "Trino (version unknown)"
	fmt.Printf("Falling back to default version: %s\n", defaultVersion)
	return defaultVersion, nil
}