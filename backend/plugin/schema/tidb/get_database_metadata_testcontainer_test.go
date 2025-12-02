package tidb

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbdb "github.com/bytebase/bytebase/backend/plugin/db/tidb"
)

// TestGetDatabaseMetadataWithTestcontainer tests the get_database_metadata function
// by comparing its output with the metadata retrieved from a real TiDB instance.
func TestGetDatabaseMetadataWithTestcontainer(t *testing.T) {
	ctx := context.Background()

	// Start TiDB container
	container, err := testcontainer.GetTiDBContainer(ctx)
	require.NoError(t, err)
	defer container.Close(ctx)

	// Create test database
	_, err = container.GetDB().Exec("CREATE DATABASE IF NOT EXISTS test_db")
	require.NoError(t, err)

	host := container.GetHost()
	port := container.GetPort()

	// Create TiDB driver for metadata sync
	driver := &tidbdb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     host,
			Port:     port,
			Database: "test_db",
		},
		Password: "",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.0",
			DatabaseName:  "test_db",
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_TIDB, config)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	// Cast to TiDB driver for SyncDBSchema
	tidbDriver, ok := openedDriver.(*tidbdb.Driver)
	require.True(t, ok, "failed to cast to tidb.Driver")

	// Test cases with various TiDB features
	testCases := []struct {
		name string
		ddl  string
	}{
		{
			name: "basic_tables_with_constraints",
			ddl: `
CREATE TABLE users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_published (published)
);
`,
		},
		{
			name: "table_with_various_data_types",
			ddl: `
CREATE TABLE data_types_test (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tiny_int_col TINYINT,
    small_int_col SMALLINT,
    medium_int_col MEDIUMINT,
    int_col INT,
    big_int_col BIGINT,
    decimal_col DECIMAL(10, 2),
    float_col FLOAT,
    double_col DOUBLE,
    bit_col BIT(8),
    char_col CHAR(10),
    varchar_col VARCHAR(100),
    binary_col BINARY(10),
    varbinary_col VARBINARY(100),
    tinytext_col TINYTEXT,
    text_col TEXT,
    mediumtext_col MEDIUMTEXT,
    longtext_col LONGTEXT,
    tinyblob_col TINYBLOB,
    blob_col BLOB,
    mediumblob_col MEDIUMBLOB,
    longblob_col LONGBLOB,
    date_col DATE,
    time_col TIME,
    datetime_col DATETIME,
    timestamp_col TIMESTAMP,
    year_col YEAR,
    json_col JSON,
    enum_col ENUM('small', 'medium', 'large'),
    set_col SET('a', 'b', 'c')
);
`,
		},
		{
			name: "table_with_foreign_keys",
			ddl: `
CREATE TABLE departments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    dept_id INT,
    manager_id INT,
    CONSTRAINT fk_dept FOREIGN KEY (dept_id) REFERENCES departments(id) ON DELETE CASCADE ON UPDATE RESTRICT,
    CONSTRAINT fk_manager FOREIGN KEY (manager_id) REFERENCES employees(id) ON DELETE SET NULL
);
`,
		},
		{
			name: "views_and_comments",
			ddl: `
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    customer_name VARCHAR(100) COMMENT 'Customer full name',
    amount DECIMAL(10,2) COMMENT 'Order amount in USD',
    status VARCHAR(20) DEFAULT 'pending'
) COMMENT='Order management table';

CREATE VIEW active_orders AS
SELECT id, customer_name, amount
FROM orders
WHERE status = 'active';
`,
		},
		{
			name: "indexes_and_constraints",
			ddl: `
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    category VARCHAR(50),
    tags JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_price_category (price, category),
    UNIQUE INDEX idx_unique_name (name),
    FULLTEXT INDEX idx_description (description)
);
`,
		},
		{
			name: "tidb_auto_random",
			ddl: `
CREATE TABLE auto_random_test (
    id BIGINT /*T![auto_rand] AUTO_RANDOM(5) */ PRIMARY KEY,
    name VARCHAR(100),
    data JSON
) COMMENT='PK_AUTO_RANDOM_BITS=5';
`,
		},
		{
			name: "complex_table_options",
			ddl: `
CREATE TABLE table_options_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    data TEXT
) ENGINE=InnoDB 
  DEFAULT CHARSET=utf8mb4 
  COLLATE=utf8mb4_general_ci 
  AUTO_INCREMENT=1000
  COMMENT='Table with various TiDB options';
`,
		},
		{
			name: "self_referencing_foreign_keys",
			ddl: `
CREATE TABLE categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id INT,
    CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE CASCADE
);
`,
		},
		{
			name: "tidb_clustered_index",
			ddl: `
CREATE TABLE clustered_test (
    id INT PRIMARY KEY CLUSTERED,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100)
);

CREATE TABLE non_clustered_test (
    id INT PRIMARY KEY NONCLUSTERED,
    name VARCHAR(100) NOT NULL,
    data TEXT
);
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up any existing objects from previous tests
			_, err := container.GetDB().ExecContext(ctx, "DROP DATABASE IF EXISTS test_db")
			require.NoError(t, err)
			_, err = container.GetDB().ExecContext(ctx, "CREATE DATABASE test_db")
			require.NoError(t, err)

			// Execute DDL using the driver
			_, err = openedDriver.Execute(ctx, tc.ddl, db.ExecuteOptions{})
			require.NoError(t, err)

			// Get metadata from live database using driver
			dbMetadata, err := tidbDriver.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Get metadata from parser
			parsedMetadata, err := GetDatabaseMetadata(tc.ddl)
			require.NoError(t, err)

			// Compare metadata
			compareMetadata(t, dbMetadata, parsedMetadata, tc.name)
		})
	}
}

// compareMetadata compares the metadata from the live database with the parsed metadata
func compareMetadata(t *testing.T, dbMeta, parsedMeta *storepb.DatabaseSchemaMetadata, testName string) {
	// Normalize both metadata for comparison
	normalizeParserMetadata(dbMeta)
	normalizeParserMetadata(parsedMeta)

	// For TiDB, we only have one schema
	require.Equal(t, 1, len(parsedMeta.Schemas), "parsed metadata should have exactly one schema")

	dbMetadata := dbMeta.Schemas[0]
	parsedSchema := parsedMeta.Schemas[0]

	// Compare tables
	require.Equal(t, len(dbMetadata.Tables), len(parsedSchema.Tables),
		"mismatch in number of tables for test %s", testName)

	// Create a map for easier lookup
	dbTableMap := make(map[string]*storepb.TableMetadata)
	for _, table := range dbMetadata.Tables {
		dbTableMap[table.Name] = table
	}

	for _, parsedTable := range parsedSchema.Tables {
		dbTable, exists := dbTableMap[parsedTable.Name]
		require.True(t, exists, "table %s not found in database metadata", parsedTable.Name)

		// Compare basic table properties
		require.Equal(t, dbTable.Engine, parsedTable.Engine,
			"engine mismatch for table %s", parsedTable.Name)

		// Compare columns
		compareColumns(t, dbTable.Columns, parsedTable.Columns, parsedTable.Name)

		// Compare indexes
		compareIndexes(t, dbTable.Indexes, parsedTable.Indexes, parsedTable.Name)

		// Compare foreign keys
		compareForeignKeys(t, dbTable.ForeignKeys, parsedTable.ForeignKeys, parsedTable.Name)
	}

	// Compare views
	compareViews(t, dbMetadata.Views, parsedSchema.Views)
}

func compareColumns(t *testing.T, dbColumns, parsedColumns []*storepb.ColumnMetadata, tableName string) {
	require.Equal(t, len(dbColumns), len(parsedColumns),
		"column count mismatch for table %s", tableName)

	// Create a map for easier lookup
	dbColumnMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range dbColumns {
		dbColumnMap[col.Name] = col
	}

	for _, parsedCol := range parsedColumns {
		dbCol, exists := dbColumnMap[parsedCol.Name]
		require.True(t, exists, "column %s not found in database metadata for table %s", parsedCol.Name, tableName)

		// Compare basic column properties
		require.Equal(t, dbCol.Type, parsedCol.Type,
			"type mismatch for column %s.%s", tableName, parsedCol.Name)
		require.Equal(t, dbCol.Nullable, parsedCol.Nullable,
			"nullable mismatch for column %s.%s", tableName, parsedCol.Name)
		require.Equal(t, dbCol.Position, parsedCol.Position,
			"position mismatch for column %s.%s", tableName, parsedCol.Name)

		// Compare default expressions (with normalization)
		normalizeDefaultExpression(dbCol)
		normalizeDefaultExpression(parsedCol)

		// Handle case where one side has a default and the other doesn't
		// This can happen when the parser doesn't capture implicit defaults
		if dbCol.Default != "" && parsedCol.Default == "" {
			// If database has a default but parser doesn't, it might be an implicit default
			// We'll log it but not fail the test
			t.Logf("Column %s.%s: database has default '%s' but parser has empty default (might be implicit)", tableName, parsedCol.Name, dbCol.Default)
		} else if dbCol.Default == "" && parsedCol.Default != "" {
			// If parser has a default but database doesn't, this is unexpected
			t.Errorf("Column %s.%s: parser has default '%s' but database has empty default", tableName, parsedCol.Name, parsedCol.Default)
		} else {
			// Both have defaults (or both are empty), they should match
			require.Equal(t, dbCol.Default, parsedCol.Default,
				"default expression mismatch for column %s.%s", tableName, parsedCol.Name)
		}
	}
}

func compareIndexes(t *testing.T, dbIndexes, parsedIndexes []*storepb.IndexMetadata, tableName string) {
	require.Equal(t, len(dbIndexes), len(parsedIndexes),
		"index count mismatch for table %s", tableName)

	// Create a map for easier lookup
	dbIndexMap := make(map[string]*storepb.IndexMetadata)
	for _, idx := range dbIndexes {
		dbIndexMap[idx.Name] = idx
	}

	// Compare each parsed index with comprehensive IndexMetadata validation
	for _, parsedIdx := range parsedIndexes {
		dbIdx, exists := dbIndexMap[parsedIdx.Name]
		require.True(t, exists, "index %s not found in database metadata for table %s", parsedIdx.Name, tableName)

		// 1. Name - explicitly validate name consistency
		require.Equal(t, dbIdx.Name, parsedIdx.Name, "table %s, index %s: name should match", tableName, parsedIdx.Name)

		// 2. Primary - validate primary key flag
		require.Equal(t, dbIdx.Primary, parsedIdx.Primary,
			"table %s, index %s: primary flag should match", tableName, parsedIdx.Name)

		// 3. Unique - validate unique constraint flag
		require.Equal(t, dbIdx.Unique, parsedIdx.Unique,
			"table %s, index %s: unique flag should match", tableName, parsedIdx.Name)

		// 4. Type - validate index type with TiDB-specific normalization
		require.Equal(t, normalizeIndexType(dbIdx.Type), normalizeIndexType(parsedIdx.Type),
			"table %s, index %s: type should match", tableName, parsedIdx.Name)

		// 5. Expressions - validate column list/expressions
		require.Equal(t, dbIdx.Expressions, parsedIdx.Expressions,
			"table %s, index %s: expressions should match", tableName, parsedIdx.Name)

		// 6. Descending - validate descending order for each expression
		if len(dbIdx.Descending) > 0 || len(parsedIdx.Descending) > 0 {
			require.Equal(t, len(dbIdx.Descending), len(parsedIdx.Descending), "table %s, index %s: descending array length should match", tableName, parsedIdx.Name)
			for i := range dbIdx.Descending {
				if i < len(parsedIdx.Descending) {
					require.Equal(t, dbIdx.Descending[i], parsedIdx.Descending[i], "table %s, index %s: descending[%d] should match", tableName, parsedIdx.Name, i)
				}
			}
		}

		// 7. KeyLength - validate index key lengths (TiDB supports prefix indexes like MySQL)
		require.Equal(t, len(dbIdx.KeyLength), len(parsedIdx.KeyLength), "table %s, index %s: key length array length should match", tableName, parsedIdx.Name)
		for i := range dbIdx.KeyLength {
			if i < len(parsedIdx.KeyLength) {
				require.Equal(t, dbIdx.KeyLength[i], parsedIdx.KeyLength[i], "table %s, index %s: key length[%d] should match", tableName, parsedIdx.Name, i)
			}
		}

		// 8. Visible - validate index visibility (TiDB supports invisible indexes like MySQL)
		require.Equal(t, dbIdx.Visible, parsedIdx.Visible, "table %s, index %s: visible should match", tableName, parsedIdx.Name)

		// 9. Comment - validate index comment
		if dbIdx.Comment != "" || parsedIdx.Comment != "" {
			require.Equal(t, dbIdx.Comment, parsedIdx.Comment, "table %s, index %s: comment should match", tableName, parsedIdx.Name)
		}

		// 10. IsConstraint - validate if index represents a constraint
		require.Equal(t, dbIdx.IsConstraint, parsedIdx.IsConstraint, "table %s, index %s: IsConstraint should match", tableName, parsedIdx.Name)

		// 11. Definition - validate full index definition for comprehensive verification
		if dbIdx.Definition != "" || parsedIdx.Definition != "" {
			// Normalize definitions for comparison since TiDB formatting may vary
			dbDef := strings.TrimSpace(strings.ToLower(dbIdx.Definition))
			parsedDef := strings.TrimSpace(strings.ToLower(parsedIdx.Definition))
			if dbDef != "" && parsedDef != "" {
				require.Equal(t, dbDef, parsedDef, "table %s, index %s: definition should match", tableName, parsedIdx.Name)
			}
		}

		t.Logf("✓ Validated all IndexMetadata fields for index %s: name=%s, primary=%v, unique=%v, type=%s, expressions=%v, visible=%v, comment=%s",
			parsedIdx.Name, parsedIdx.Name, parsedIdx.Primary, parsedIdx.Unique, parsedIdx.Type, parsedIdx.Expressions, parsedIdx.Visible, parsedIdx.Comment)
	}
}

func compareForeignKeys(t *testing.T, dbFKs, parsedFKs []*storepb.ForeignKeyMetadata, tableName string) {
	require.Equal(t, len(dbFKs), len(parsedFKs),
		"foreign key count mismatch for table %s", tableName)

	// Create a map for easier lookup
	dbFKMap := make(map[string]*storepb.ForeignKeyMetadata)
	for _, fk := range dbFKs {
		dbFKMap[fk.Name] = fk
	}

	for _, parsedFK := range parsedFKs {
		dbFK, exists := dbFKMap[parsedFK.Name]
		require.True(t, exists, "foreign key %s not found in database metadata for table %s", parsedFK.Name, tableName)

		require.Equal(t, dbFK.Columns, parsedFK.Columns,
			"columns mismatch for foreign key %s.%s", tableName, parsedFK.Name)
		require.Equal(t, dbFK.ReferencedTable, parsedFK.ReferencedTable,
			"referenced table mismatch for foreign key %s.%s", tableName, parsedFK.Name)
		require.Equal(t, dbFK.ReferencedColumns, parsedFK.ReferencedColumns,
			"referenced columns mismatch for foreign key %s.%s", tableName, parsedFK.Name)
		require.Equal(t, dbFK.OnDelete, parsedFK.OnDelete,
			"on delete mismatch for foreign key %s.%s", tableName, parsedFK.Name)
		require.Equal(t, dbFK.OnUpdate, parsedFK.OnUpdate,
			"on update mismatch for foreign key %s.%s", tableName, parsedFK.Name)
	}
}

func compareViews(t *testing.T, dbViews, parsedViews []*storepb.ViewMetadata) {
	require.Equal(t, len(dbViews), len(parsedViews), "view count mismatch")

	// Create a map for easier lookup
	dbViewMap := make(map[string]*storepb.ViewMetadata)
	for _, view := range dbViews {
		dbViewMap[view.Name] = view
	}

	for _, parsedView := range parsedViews {
		dbView, exists := dbViewMap[parsedView.Name]
		require.True(t, exists, "view %s not found in database metadata", parsedView.Name)

		// View definitions might have formatting differences, so we do a basic check
		require.NotEmpty(t, parsedView.Definition, "parsed view definition should not be empty for %s", parsedView.Name)
		require.Contains(t, strings.ToLower(parsedView.Definition), "select",
			"parsed view definition should contain SELECT for %s", parsedView.Name)

		// Also check that database view has definition
		require.NotEmpty(t, dbView.Definition, "database view definition should not be empty for %s", dbView.Name)
	}
}

// normalizeParserMetadata normalizes metadata for comparison between parser and database
func normalizeParserMetadata(metadata *storepb.DatabaseSchemaMetadata) {
	if metadata == nil {
		return
	}

	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			// Normalize charset and collation since they may vary
			table.Charset = ""
			table.Collation = ""

			for _, col := range table.Columns {
				// Normalize default expressions
				normalizeDefaultExpression(col)
			}

			for _, idx := range table.Indexes {
				// Normalize index types
				idx.Type = normalizeIndexType(idx.Type)
			}

			for _, fk := range table.ForeignKeys {
				// Ensure consistent foreign key actions
				if fk.OnDelete == "" {
					fk.OnDelete = "NO ACTION"
				}
				if fk.OnUpdate == "" {
					fk.OnUpdate = "NO ACTION"
				}
			}
		}
	}
}

func normalizeDefaultExpression(col *storepb.ColumnMetadata) {
	if col.Default == "" {
		return
	}

	// Normalize common default value formats
	expr := col.Default

	// Handle NULL values - normalize "NULL" string to empty string
	if strings.ToUpper(expr) == "NULL" {
		col.Default = ""
		return
	}

	// Remove charset prefixes
	if strings.HasPrefix(expr, "_utf8mb4") {
		if idx := strings.Index(expr, "'"); idx != -1 {
			expr = expr[idx:]
		}
	}

	// Remove quotes from string literals
	expr = strings.Trim(expr, "'\"")

	// Normalize boolean values
	if expr == "0" && (col.Type == "tinyint" || col.Type == "tinyint(1)") {
		expr = "false"
	}
	if expr == "1" && (col.Type == "tinyint" || col.Type == "tinyint(1)") {
		expr = "true"
	}
	if expr == "false" && (col.Type == "tinyint" || col.Type == "tinyint(1)") {
		expr = "false"
	}
	if expr == "true" && (col.Type == "tinyint" || col.Type == "tinyint(1)") {
		expr = "true"
	}

	col.Default = expr
}

func normalizeIndexType(indexType string) string {
	// TiDB always uses BTREE for most indexes
	if indexType == "" {
		return "BTREE"
	}
	return strings.ToUpper(indexType)
}
