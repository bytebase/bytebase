package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	// Import MySQL driver (TiDB is compatible with MySQL protocol)
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbdb "github.com/bytebase/bytebase/backend/plugin/db/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestGenerateMigrationWithTestcontainer tests the generate migration function
// by applying migrations and rollback to verify the schema can be restored.
func TestGenerateMigrationWithTestcontainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TiDB testcontainer test in short mode")
	}

	ctx := context.Background()

	// Start TiDB container
	req := testcontainers.ContainerRequest{
		Image:        "pingcap/tidb:v8.5.0",
		ExposedPorts: []string{"4000/tcp"},
		WaitingFor: wait.ForLog("server is running MySQL protocol").
			WithStartupTimeout(5 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection details
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "4000")
	require.NoError(t, err)

	// Test cases with various schema changes
	testCases := []struct {
		name          string
		initialSchema string
		migrationDDL  string
		description   string
	}{
		{
			name: "reverse_comments_with_special_characters",
			initialSchema: `
CREATE TABLE test_table (
    id INT NOT NULL AUTO_INCREMENT,
    simple_field VARCHAR(100),
    field_with_quotes VARCHAR(100) COMMENT 'Field with "double" and ''single'' quotes',
    field_with_symbols VARCHAR(100) COMMENT 'Field with symbols: @#$%^&*()_+-={}[]|:";''<>?,./ and more!',
    field_with_unicode VARCHAR(100) COMMENT 'Unicode test: 你好世界 🌍 café naïve résumé Москва العالم 한국어',
    field_multiline TEXT COMMENT 'Multi-line comment:
Line 1 with regular text
Line 2 with "quotes" and symbols @#$
Line 3 with unicode: 测试 🚀 café
Final line with mixed content',
    field_with_sql VARCHAR(100) COMMENT 'Comment with SQL-like content: SELECT * FROM table WHERE id = ''123'' AND name LIKE "%test%"',
    field_with_html VARCHAR(100) COMMENT 'HTML content: <div class="test">Hello & "World"</div> <!-- comment -->',
    field_with_json VARCHAR(100) COMMENT 'JSON example: {"name": "test", "value": 123, "nested": {"key": "value with spaces"}}',
    field_with_escape VARCHAR(100) COMMENT 'Escape sequences: \n \t \r " '' \\',
    PRIMARY KEY (id)
) COMMENT = 'Table with "double quotes" and ''single quotes'' in comment';

CREATE TABLE special_comments_table (
    id BIGINT NOT NULL AUTO_RANDOM COMMENT 'ID with "quotes", symbols @#$, unicode 测试🌟, and
multi-line
content',
    data_field JSON COMMENT 'Updated JSON field: {"new": "structure", "with": ["array", "of", "strings"], "escapes": "\n\t\r"}',
    html_field TEXT,
    sql_field VARCHAR(255) COMMENT 'SQL patterns: SELECT * FROM users WHERE name = ''O''Brien'' AND age > 21',
    unicode_field VARCHAR(200) COMMENT '多语言支持: English, 中文, العربية, Русский, 日本語, 한국어, Français, Español',
    symbols_field VARCHAR(100) COMMENT 'All symbols: !@#$%^&*()_+-={}[]|:";''<>?,./ plus tab	and newline
test',
    url_field VARCHAR(300) COMMENT 'URL with params: https://example.com/path?param1=value1&param2="quoted value"&param3=50%+discount',
    regex_field VARCHAR(150) COMMENT 'Regex pattern: ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$ for email validation',
    PRIMARY KEY (id),
    INDEX idx_unicode (unicode_field)
) COMMENT = 'Special characters test table with:
- Double quotes: "test"
- Single quotes: ''test''
- Unicode: 特殊字符测试表 🎯
- Symbols: @#$%^&*()
- HTML: <div>content</div>
- JSON: {"key": "value"}
- URLs: https://example.com
- Multi-line content with various encodings';
`,
			migrationDDL: `
-- Drop table with special comments
DROP TABLE special_comments_table;

-- Drop columns with special character comments
ALTER TABLE test_table DROP COLUMN field_with_escape;
ALTER TABLE test_table DROP COLUMN field_with_json;
ALTER TABLE test_table DROP COLUMN field_with_html;
ALTER TABLE test_table DROP COLUMN field_with_sql;
ALTER TABLE test_table DROP COLUMN field_multiline;
ALTER TABLE test_table DROP COLUMN field_with_unicode;
ALTER TABLE test_table DROP COLUMN field_with_symbols;
ALTER TABLE test_table DROP COLUMN field_with_quotes;

-- Remove table comment
ALTER TABLE test_table COMMENT = '';
`,
			description: "Reverse of comments with special characters - removing all special character comments",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Initialize the database schema and get schema result A
			portInt, err := strconv.Atoi(port.Port())
			require.NoError(t, err)

			// Add a delay to ensure TiDB is fully ready
			time.Sleep(3 * time.Second)

			t.Logf("Connecting to TiDB at %s:%d", host, portInt)
			testDB, err := openTestDatabase(host, portInt, "root", "", "test")
			require.NoError(t, err, "Failed to connect to TiDB database")
			defer testDB.Close()

			// Clean up any existing objects from previous tests
			cleanupDatabase(t, testDB)

			// Execute initial schema
			if err := executeStatements(testDB, tc.initialSchema); err != nil {
				t.Fatalf("Failed to execute initial schema: %v", err)
			}

			schemaA, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "", "test")
			require.NoError(t, err)

			// Step 2: Do some migration and get schema result B
			if err := executeStatements(testDB, tc.migrationDDL); err != nil {
				t.Fatalf("Failed to execute migration DDL: %v", err)
			}

			schemaB, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "", "test")
			require.NoError(t, err)

			// Step 3: Call generate migration to get the rollback DDL
			// Convert to model.DatabaseSchema
			dbSchemaA := model.NewDatabaseSchema(schemaA, nil, nil, storepb.Engine_TIDB, false)
			dbSchemaB := model.NewDatabaseSchema(schemaB, nil, nil, storepb.Engine_TIDB, false)

			// Get diff from B to A (to generate rollback)
			diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_TIDB, dbSchemaB, dbSchemaA)
			require.NoError(t, err)

			// Log the diff for debugging
			t.Logf("Test case: %s", tc.description)
			t.Logf("Table changes: %d", len(diff.TableChanges))
			for _, tc := range diff.TableChanges {
				t.Logf("  Table: %s, Action: %v", tc.TableName, tc.Action)
			}
			t.Logf("View changes: %d", len(diff.ViewChanges))
			for _, vc := range diff.ViewChanges {
				t.Logf("  View: %s, Action: %v", vc.ViewName, vc.Action)
			}
			t.Logf("Function changes: %d", len(diff.FunctionChanges))
			for _, fc := range diff.FunctionChanges {
				t.Logf("  Function: %s, Action: %v", fc.FunctionName, fc.Action)
			}

			// Generate rollback migration
			rollbackDDL, err := schema.GenerateMigration(storepb.Engine_TIDB, diff)
			require.NoError(t, err)

			t.Logf("Rollback DDL:\n%s", rollbackDDL)

			// Step 4: Run rollback DDL and get schema result C
			if err := executeStatements(testDB, rollbackDDL); err != nil {
				t.Fatalf("Failed to execute rollback DDL: %v", err)
			}

			schemaC, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "", "test")
			require.NoError(t, err)

			// Step 5: Compare schema result A and C to ensure they are the same
			normalizeMetadataForComparison(schemaA)
			normalizeMetadataForComparison(schemaC)

			// Use cmp with protocmp for proto message comparison
			if diff := cmp.Diff(schemaA, schemaC, protocmp.Transform()); diff != "" {
				t.Errorf("Schema mismatch after rollback (-want +got):\n%s", diff)
			}
		})
	}
}

// openTestDatabase opens a connection to the test TiDB database
func openTestDatabase(host string, port int, username, password, database string) (*sql.DB, error) {
	var dsn string
	if password == "" {
		dsn = fmt.Sprintf("%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&multiStatements=true",
			username, host, port, database)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&multiStatements=true",
			username, password, host, port, database)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open connection to TiDB")
	}

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Try to ping with retries
	var pingErr error
	for i := 0; i < 10; i++ {
		if pingErr = db.Ping(); pingErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if pingErr != nil {
		return nil, errors.Wrapf(pingErr, "failed to ping TiDB database after retries")
	}

	return db, nil
}

// executeStatements executes multiple SQL statements
func executeStatements(db *sql.DB, statements string) error {
	// MySQL driver supports multi-statement execution natively
	if _, err := db.Exec(statements); err != nil {
		return errors.Wrapf(err, "failed to execute statements")
	}
	return nil
}

// getSyncMetadataForGenerateMigration retrieves metadata from the live database using Driver.SyncDBSchema
func getSyncMetadataForGenerateMigration(ctx context.Context, host string, port int, username, password, database string) (*storepb.DatabaseSchemaMetadata, error) {
	// Create a driver instance using the tidb package
	driver := &tidbdb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: username,
			Host:     host,
			Port:     fmt.Sprintf("%d", port),
			Database: database,
		},
		Password: password,
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "v8.5.0",
			DatabaseName:  database,
		},
	}

	// Open connection using the driver
	openedDriver, err := driver.Open(ctx, storepb.Engine_TIDB, config)
	if err != nil {
		return nil, err
	}
	defer openedDriver.Close(ctx)

	// Use SyncDBSchema to get the metadata
	tidbDriver, ok := openedDriver.(*tidbdb.Driver)
	if !ok {
		return nil, errors.New("failed to cast to tidb.Driver")
	}

	metadata, err := tidbDriver.SyncDBSchema(ctx)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// cleanupDatabase removes all objects from the database
func cleanupDatabase(_ *testing.T, db *sql.DB) {
	// Disable foreign key checks
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	defer func() {
		_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}()

	// Drop all tables
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var tables []string
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err == nil {
				tables = append(tables, table)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, table := range tables {
			_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
		}
	}

	// Drop all views
	rows, err = db.Query("SELECT table_name FROM information_schema.views WHERE table_schema = DATABASE()")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var views []string
		for rows.Next() {
			var view string
			if err := rows.Scan(&view); err == nil {
				views = append(views, view)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, view := range views {
			_, _ = db.Exec(fmt.Sprintf("DROP VIEW IF EXISTS `%s`", view))
		}
	}

	// Drop all procedures (TiDB may support them)
	rows, err = db.Query("SELECT routine_name FROM information_schema.routines WHERE routine_schema = DATABASE() AND routine_type = 'PROCEDURE'")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var procedures []string
		for rows.Next() {
			var proc string
			if err := rows.Scan(&proc); err == nil {
				procedures = append(procedures, proc)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, proc := range procedures {
			_, _ = db.Exec(fmt.Sprintf("DROP PROCEDURE IF EXISTS `%s`", proc))
		}
	}

	// Drop all functions (TiDB may support them)
	rows, err = db.Query("SELECT routine_name FROM information_schema.routines WHERE routine_schema = DATABASE() AND routine_type = 'FUNCTION'")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var functions []string
		for rows.Next() {
			var fn string
			if err := rows.Scan(&fn); err == nil {
				functions = append(functions, fn)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, fn := range functions {
			_, _ = db.Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS `%s`", fn))
		}
	}
}

// normalizeMetadataForComparison normalizes metadata to ignore differences that don't affect schema equality
func normalizeMetadataForComparison(metadata *storepb.DatabaseSchemaMetadata) {
	// Clear database name as it might differ
	metadata.Name = ""

	// Normalize schemas
	for _, schema := range metadata.Schemas {
		// Normalize tables
		for _, table := range schema.Tables {
			table.DataSize = 0
			table.IndexSize = 0
			table.RowCount = 0

			// Clear auto-increment and auto-random values as they might differ
			for _, col := range table.Columns {
				if col.Default == "AUTO_INCREMENT" {
					col.Default = "AUTO_INCREMENT"
				} else if strings.HasPrefix(col.Default, "AUTO_RANDOM") {
					// Keep the AUTO_RANDOM marker but normalize the value
					col.Default = "AUTO_RANDOM"
				}
				// Clear column position as it might change during DDL operations
				col.Position = 0
			}

			// Sort columns by name for consistent comparison
			sortColumnsByName(table.Columns)

			// Sort indexes by name
			sortIndexesByName(table.Indexes)

			// Sort foreign keys by name
			sortForeignKeysByName(table.ForeignKeys)

			// Sort check constraints by name
			sortCheckConstraintsByName(table.CheckConstraints)

			// Sort triggers by name
			sortTriggersByName(table.Triggers)
		}

		// Sort all collections for consistent comparison
		sortTablesByName(schema.Tables)
		sortViewsByName(schema.Views)
		sortFunctionsByName(schema.Functions)
		sortProceduresByName(schema.Procedures)
	}

	// Sort schemas by name
	sortSchemasByName(metadata.Schemas)

	// Sort extensions by name
	sortExtensionsByName(metadata.Extensions)
}

// Sorting helper functions
func sortSchemasByName(schemas []*storepb.SchemaMetadata) {
	slices.SortFunc(schemas, func(a, b *storepb.SchemaMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortTablesByName(tables []*storepb.TableMetadata) {
	slices.SortFunc(tables, func(a, b *storepb.TableMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortColumnsByName(columns []*storepb.ColumnMetadata) {
	slices.SortFunc(columns, func(a, b *storepb.ColumnMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortIndexesByName(indexes []*storepb.IndexMetadata) {
	slices.SortFunc(indexes, func(a, b *storepb.IndexMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortForeignKeysByName(fks []*storepb.ForeignKeyMetadata) {
	slices.SortFunc(fks, func(a, b *storepb.ForeignKeyMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortCheckConstraintsByName(checks []*storepb.CheckConstraintMetadata) {
	slices.SortFunc(checks, func(a, b *storepb.CheckConstraintMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortViewsByName(views []*storepb.ViewMetadata) {
	slices.SortFunc(views, func(a, b *storepb.ViewMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortFunctionsByName(functions []*storepb.FunctionMetadata) {
	slices.SortFunc(functions, func(a, b *storepb.FunctionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortProceduresByName(procedures []*storepb.ProcedureMetadata) {
	slices.SortFunc(procedures, func(a, b *storepb.ProcedureMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortExtensionsByName(extensions []*storepb.ExtensionMetadata) {
	slices.SortFunc(extensions, func(a, b *storepb.ExtensionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortTriggersByName(triggers []*storepb.TriggerMetadata) {
	slices.SortFunc(triggers, func(a, b *storepb.TriggerMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}
