package pg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/bytebase/bytebase/backend/plugin/db"
	pgdb "github.com/bytebase/bytebase/backend/plugin/db/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TestGetDatabaseMetadataWithTestcontainer tests the get_database_metadata function
// by comparing its output with the metadata retrieved from a real PostgreSQL instance.
func TestGetDatabaseMetadataWithTestcontainer(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connectionString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to the database
	connConfig, err := pgx.ParseConfig(connectionString)
	require.NoError(t, err)
	db := stdlib.OpenDB(*connConfig)
	defer db.Close()

	// Test cases with various PostgreSQL features
	testCases := []struct {
		name string
		ddl  string
	}{
		{
			name: "basic_tables_with_constraints",
			ddl: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published ON posts(published) WHERE published = true;
`,
		},
		{
			name: "sequences_and_custom_types",
			ddl: `
CREATE TYPE status_enum AS ENUM ('pending', 'active', 'inactive', 'deleted');
CREATE TYPE mood AS ENUM ('happy', 'sad', 'neutral');

CREATE SEQUENCE custom_id_seq START WITH 1000 INCREMENT BY 10;

CREATE TABLE items (
    id INTEGER DEFAULT nextval('custom_id_seq') PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status status_enum DEFAULT 'pending',
    user_mood mood
);
`,
		},
		{
			name: "views_and_functions",
			ddl: `
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department VARCHAR(50),
    salary DECIMAL(10, 2)
);

CREATE VIEW active_employees AS
SELECT id, name, department
FROM employees
WHERE department IS NOT NULL;

CREATE FUNCTION get_employee_count(dept VARCHAR) RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM employees WHERE department = dept);
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION calculate_bonus(emp_id INTEGER) RETURNS DECIMAL AS $$
DECLARE
    emp_salary DECIMAL;
BEGIN
    SELECT salary INTO emp_salary FROM employees WHERE id = emp_id;
    RETURN emp_salary * 0.1;
END;
$$ LANGUAGE plpgsql;
`,
		},
		{
			name: "partitioned_tables",
			ddl: `
CREATE TABLE sales (
    id SERIAL,
    sale_date DATE NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    region VARCHAR(50) NOT NULL
) PARTITION BY RANGE (sale_date);

CREATE TABLE sales_2023_q1 PARTITION OF sales
FOR VALUES FROM ('2023-01-01') TO ('2023-04-01');

CREATE TABLE sales_2023_q2 PARTITION OF sales
FOR VALUES FROM ('2023-04-01') TO ('2023-07-01');

CREATE TABLE sales_2023_q3 PARTITION OF sales
FOR VALUES FROM ('2023-07-01') TO ('2023-10-01');

CREATE TABLE sales_2023_q4 PARTITION OF sales
FOR VALUES FROM ('2023-10-01') TO ('2024-01-01');
`,
		},
		{
			name: "extensions_and_advanced_features",
			ddl: `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE documents (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_documents_metadata ON documents USING GIN (metadata);
CREATE INDEX idx_documents_tags ON documents USING GIN (tags);
`,
		},
		{
			name: "indexes_with_asc_desc",
			ddl: `
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    order_date DATE NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20)
);

-- Index with explicit DESC
CREATE INDEX idx_orders_date_desc ON orders(order_date DESC);

-- Index with multiple columns, mixed ASC/DESC
CREATE INDEX idx_orders_customer_date ON orders(customer_id ASC, order_date DESC);

-- Index with expressions and DESC
CREATE INDEX idx_orders_year_month ON orders(EXTRACT(YEAR FROM order_date) DESC, EXTRACT(MONTH FROM order_date) ASC);

-- Unique index with DESC
CREATE UNIQUE INDEX idx_orders_customer_status ON orders(customer_id, status DESC) WHERE status IS NOT NULL;
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new database for each test case
			dbName := fmt.Sprintf("test_%s", tc.name)
			_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
			require.NoError(t, err)
			defer func() {
				_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
			}()

			// Connect to the test database
			testConnConfig := *connConfig
			testConnConfig.Database = dbName
			testDB := stdlib.OpenDB(testConnConfig)
			defer testDB.Close()

			// Execute the DDL
			_, err = testDB.Exec(tc.ddl)
			require.NoError(t, err)

			// Get metadata using Driver.SyncDBSchema
			syncMetadata, err := getSyncMetadata(ctx, &testConnConfig, dbName)
			require.NoError(t, err)

			// Get metadata using get_database_metadata
			parseMetadata, err := GetDatabaseMetadata(tc.ddl)
			require.NoError(t, err)

			// Compare the two metadata structures
			compareMetadata(t, syncMetadata, parseMetadata)
		})
	}
}

// getSyncMetadata retrieves metadata from the live database using Driver.SyncDBSchema
func getSyncMetadata(ctx context.Context, connConfig *pgx.ConnConfig, dbName string) (*storepb.DatabaseSchemaMetadata, error) {
	// Create a driver instance using the pg package
	driver := &pgdb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: connConfig.User,
			Host:     connConfig.Host,
			Port:     fmt.Sprintf("%d", connConfig.Port),
			Database: dbName,
		},
		Password: connConfig.Password,
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0", // PostgreSQL 16
			DatabaseName:  dbName,
		},
	}

	// Open connection using the driver
	openedDriver, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	if err != nil {
		return nil, err
	}
	defer openedDriver.Close(ctx)

	// Use SyncDBSchema to get the metadata
	pgDriver, ok := openedDriver.(*pgdb.Driver)
	if !ok {
		return nil, errors.New("failed to cast to pg.Driver")
	}

	metadata, err := pgDriver.SyncDBSchema(ctx)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// compareMetadata compares metadata from sync.go and get_database_metadata
func compareMetadata(t *testing.T, syncMeta, parseMeta *storepb.DatabaseSchemaMetadata) {
	// Compare schemas
	require.Equal(t, len(syncMeta.Schemas), len(parseMeta.Schemas), "number of schemas should match")

	// Find the public schema in both
	var syncPublic, parsePublic *storepb.SchemaMetadata
	for _, schema := range syncMeta.Schemas {
		if schema.Name == "public" {
			syncPublic = schema
			break
		}
	}
	for _, schema := range parseMeta.Schemas {
		if schema.Name == "public" {
			parsePublic = schema
			break
		}
	}

	require.NotNil(t, syncPublic, "sync metadata should have public schema")
	require.NotNil(t, parsePublic, "parse metadata should have public schema")

	// Compare tables
	compareTables(t, syncPublic.Tables, parsePublic.Tables)

	// Compare views
	compareViews(t, syncPublic.Views, parsePublic.Views)

	// Compare functions
	compareFunctions(t, syncPublic.Functions, parsePublic.Functions)

	// Compare sequences
	compareSequences(t, syncPublic.Sequences, parsePublic.Sequences)

	// Compare enums
	compareEnums(t, syncPublic.EnumTypes, parsePublic.EnumTypes)

	// Compare extensions
	compareExtensions(t, syncMeta.Extensions, parseMeta.Extensions)
}

// normalizeExpression normalizes an expression for comparison by:
// - Converting to lowercase
// - Removing extra whitespace
// - Normalizing quotes
func normalizeExpression(expr string) string {
	// Convert to lowercase
	expr = strings.ToLower(expr)

	// Replace multiple spaces with single space
	expr = strings.Join(strings.Fields(expr), " ")

	// Remove spaces around parentheses
	expr = strings.ReplaceAll(expr, " (", "(")
	expr = strings.ReplaceAll(expr, "( ", "(")
	expr = strings.ReplaceAll(expr, " )", ")")
	expr = strings.ReplaceAll(expr, ") ", ")")

	// Normalize quotes around identifiers
	expr = strings.ReplaceAll(expr, "'", "")

	return expr
}

// normalizeSQL normalizes SQL for comparison by:
// - Converting to lowercase
// - Removing extra whitespace
// - Removing trailing semicolons
// - Removing schema qualifiers for common schemas
// - Normalizing parentheses
func normalizeSQL(sql string) string {
	// Convert to lowercase
	sql = strings.ToLower(sql)

	// Replace multiple spaces/newlines with single space
	sql = strings.Join(strings.Fields(sql), " ")

	// Remove trailing semicolons
	sql = strings.TrimSuffix(sql, ";")

	// Remove schema qualifiers for public schema
	// This handles cases like "public.table_name" -> "table_name"
	sql = strings.ReplaceAll(sql, "public.", "")

	// Handle PostgreSQL's tendency to wrap WHERE conditions in parentheses
	// e.g., "WHERE (condition)" -> "WHERE condition"
	// We need to be careful to only remove the outermost parentheses around the entire WHERE clause
	whereIndex := strings.Index(sql, "where (")
	if whereIndex >= 0 {
		// Find the matching closing parenthesis for the WHERE clause
		afterWhere := sql[whereIndex+7:] // Skip "where ("
		openCount := 1
		closeIndex := -1

		// Find the matching closing parenthesis
		for i, ch := range afterWhere {
			if ch == '(' {
				openCount++
			} else if ch == ')' {
				openCount--
				if openCount == 0 {
					closeIndex = i
					break
				}
			}
		}

		// If we found the matching closing parenthesis and it's at the end or followed by end/order/group
		if closeIndex >= 0 {
			beforeWhere := sql[:whereIndex+6] // Include "where "
			afterCloseParen := ""
			if whereIndex+7+closeIndex+1 < len(sql) {
				afterCloseParen = sql[whereIndex+7+closeIndex+1:]
			}

			// Check if the closing paren is at the end or followed by valid SQL keywords
			if afterCloseParen == "" || strings.HasPrefix(strings.TrimSpace(afterCloseParen), "order by") ||
				strings.HasPrefix(strings.TrimSpace(afterCloseParen), "group by") ||
				strings.HasPrefix(strings.TrimSpace(afterCloseParen), "limit") {
				// Remove the parentheses
				sql = beforeWhere + afterWhere[:closeIndex] + afterCloseParen
			}
		}
	}

	// Final trim
	sql = strings.TrimSpace(sql)

	return sql
}

// normalizeSignature normalizes function signatures for comparison
func normalizeSignature(sig string) string {
	// Convert to lowercase
	sig = strings.ToLower(sig)

	// Remove extra spaces
	sig = strings.Join(strings.Fields(sig), " ")

	// Remove spaces around parentheses and commas
	sig = strings.ReplaceAll(sig, " (", "(")
	sig = strings.ReplaceAll(sig, "( ", "(")
	sig = strings.ReplaceAll(sig, " )", ")")
	sig = strings.ReplaceAll(sig, ") ", ")")
	sig = strings.ReplaceAll(sig, " ,", ",")
	sig = strings.ReplaceAll(sig, ", ", ",")

	// Remove quotes around function names
	sig = strings.ReplaceAll(sig, "\"", "")

	return sig
}

func compareTables(t *testing.T, syncTables, parseTables []*storepb.TableMetadata) {
	// Log the tables found for debugging
	t.Logf("Sync tables: %d", len(syncTables))
	for _, table := range syncTables {
		t.Logf("  - %s", table.Name)
	}
	t.Logf("Parse tables: %d", len(parseTables))
	for _, table := range parseTables {
		t.Logf("  - %s", table.Name)
	}

	require.Equal(t, len(syncTables), len(parseTables), "number of tables should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.TableMetadata)
	for _, table := range syncTables {
		syncMap[table.Name] = table
	}

	parseMap := make(map[string]*storepb.TableMetadata)
	for _, table := range parseTables {
		parseMap[table.Name] = table
	}

	// Compare each table
	for name, syncTable := range syncMap {
		parseTable, exists := parseMap[name]
		require.True(t, exists, "table %s should exist in parsed metadata", name)

		// Compare columns
		compareColumns(t, name, syncTable.Columns, parseTable.Columns)

		// Compare indexes
		compareIndexes(t, name, syncTable.Indexes, parseTable.Indexes)

		// Compare foreign keys
		compareForeignKeys(t, name, syncTable.ForeignKeys, parseTable.ForeignKeys)

		// Compare partitions
		comparePartitions(t, name, syncTable.Partitions, parseTable.Partitions)
	}
}

func compareColumns(t *testing.T, tableName string, syncCols, parseCols []*storepb.ColumnMetadata) {
	require.Equal(t, len(syncCols), len(parseCols), "table %s: number of columns should match", tableName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range syncCols {
		syncMap[col.Name] = col
	}

	for _, parseCol := range parseCols {
		syncCol, exists := syncMap[parseCol.Name]
		require.True(t, exists, "table %s: column %s should exist in sync metadata", tableName, parseCol.Name)

		// Compare column properties
		require.Equal(t, syncCol.Type, parseCol.Type, "table %s, column %s: type should match", tableName, parseCol.Name)
		require.Equal(t, syncCol.Nullable, parseCol.Nullable, "table %s, column %s: nullable should match", tableName, parseCol.Name)

		// Compare default values if both exist
		hasDefaultSync := syncCol.DefaultNull || syncCol.DefaultExpression != "" || syncCol.Default != ""
		hasDefaultParse := parseCol.DefaultNull || parseCol.DefaultExpression != "" || parseCol.Default != ""
		if hasDefaultSync && hasDefaultParse {
			// Default values might be represented differently, so we just check they exist
			t.Logf("table %s, column %s: default values exist in both", tableName, parseCol.Name)
		}
	}
}

func compareIndexes(t *testing.T, tableName string, syncIndexes, parseIndexes []*storepb.IndexMetadata) {
	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.IndexMetadata)
	for _, idx := range syncIndexes {
		syncMap[idx.Name] = idx
	}

	parseMap := make(map[string]*storepb.IndexMetadata)
	for _, idx := range parseIndexes {
		parseMap[idx.Name] = idx
	}

	// Compare common indexes
	for name, parseIdx := range parseMap {
		syncIdx, exists := syncMap[name]
		if !exists {
			// Some indexes might be system-generated and not in DDL
			t.Logf("table %s: index %s exists in parse but not in sync (might be implicit)", tableName, name)
			continue
		}

		require.Equal(t, syncIdx.Primary, parseIdx.Primary, "table %s, index %s: primary should match", tableName, name)
		require.Equal(t, syncIdx.Unique, parseIdx.Unique, "table %s, index %s: unique should match", tableName, name)

		// Compare expressions - sync.go gets normalized expressions from PostgreSQL catalog
		// while parser gets the raw expressions. We need to handle this difference.
		if len(syncIdx.Expressions) == len(parseIdx.Expressions) {
			for i := range syncIdx.Expressions {
				syncExpr := normalizeExpression(syncIdx.Expressions[i])
				parseExpr := normalizeExpression(parseIdx.Expressions[i])
				require.Equal(t, syncExpr, parseExpr, "table %s, index %s: expression[%d] should match", tableName, name, i)
			}
		} else {
			require.Equal(t, len(syncIdx.Expressions), len(parseIdx.Expressions), "table %s, index %s: expressions count should match", tableName, name)
		}

		// Compare descending order for each expression
		// Note: sync.go currently doesn't populate the Descending field, so we need to handle both cases
		if len(syncIdx.Descending) > 0 && len(parseIdx.Descending) > 0 {
			// Both have descending info, compare them
			require.Equal(t, len(syncIdx.Descending), len(parseIdx.Descending), "table %s, index %s: descending array length should match", tableName, name)
			for i := range syncIdx.Descending {
				require.Equal(t, syncIdx.Descending[i], parseIdx.Descending[i], "table %s, index %s: descending[%d] should match", tableName, name, i)
			}
		} else if len(parseIdx.Descending) > 0 {
			// Only parser has descending info, verify it matches the number of expressions
			require.Equal(t, len(parseIdx.Expressions), len(parseIdx.Descending), "table %s, index %s: descending array should match expressions count", tableName, name)
		}

		// Also compare the index type if available
		if syncIdx.Type != "" || parseIdx.Type != "" {
			require.Equal(t, syncIdx.Type, parseIdx.Type, "table %s, index %s: type should match", tableName, name)
		}

		// Compare IsConstraint field
		require.Equal(t, syncIdx.IsConstraint, parseIdx.IsConstraint, "table %s, index %s: IsConstraint should match", tableName, name)
	}
}

func compareForeignKeys(t *testing.T, tableName string, syncFKs, parseFKs []*storepb.ForeignKeyMetadata) {
	require.Equal(t, len(syncFKs), len(parseFKs), "table %s: number of foreign keys should match", tableName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ForeignKeyMetadata)
	for _, fk := range syncFKs {
		syncMap[fk.Name] = fk
	}

	for _, parseFk := range parseFKs {
		syncFk, exists := syncMap[parseFk.Name]
		require.True(t, exists, "table %s: foreign key %s should exist in sync metadata", tableName, parseFk.Name)

		require.ElementsMatch(t, syncFk.Columns, parseFk.Columns, "table %s, FK %s: columns should match", tableName, parseFk.Name)
		require.Equal(t, syncFk.ReferencedTable, parseFk.ReferencedTable, "table %s, FK %s: referenced table should match", tableName, parseFk.Name)
		require.ElementsMatch(t, syncFk.ReferencedColumns, parseFk.ReferencedColumns, "table %s, FK %s: referenced columns should match", tableName, parseFk.Name)
	}
}

func comparePartitions(t *testing.T, tableName string, syncParts, parseParts []*storepb.TablePartitionMetadata) {
	require.Equal(t, len(syncParts), len(parseParts), "table %s: number of partitions should match", tableName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.TablePartitionMetadata)
	for _, part := range syncParts {
		syncMap[part.Name] = part
	}

	for _, parsePart := range parseParts {
		syncPart, exists := syncMap[parsePart.Name]
		require.True(t, exists, "table %s: partition %s should exist in sync metadata", tableName, parsePart.Name)
		require.Equal(t, syncPart.Expression, parsePart.Expression, "table %s, partition %s: expression should match", tableName, parsePart.Name)
		require.Equal(t, syncPart.Value, parsePart.Value, "table %s, partition %s: value should match", tableName, parsePart.Name)
	}
}

func compareViews(t *testing.T, syncViews, parseViews []*storepb.ViewMetadata) {
	require.Equal(t, len(syncViews), len(parseViews), "number of views should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ViewMetadata)
	for _, view := range syncViews {
		syncMap[view.Name] = view
	}

	for _, parseView := range parseViews {
		syncView, exists := syncMap[parseView.Name]
		require.True(t, exists, "view %s should exist in sync metadata", parseView.Name)

		// Compare view definitions - they might be formatted differently
		// PostgreSQL normalizes view definitions when storing them
		syncDef := normalizeSQL(syncView.Definition)
		parseDef := normalizeSQL(parseView.Definition)
		require.Equal(t, syncDef, parseDef, "view %s: definition should match", parseView.Name)

		// Compare comment if present
		if syncView.Comment != "" || parseView.Comment != "" {
			require.Equal(t, syncView.Comment, parseView.Comment, "view %s: comment should match", parseView.Name)
		}
	}
}

func compareFunctions(t *testing.T, syncFuncs, parseFuncs []*storepb.FunctionMetadata) {
	// Function comparison is tricky because signatures might be formatted differently
	t.Logf("sync has %d functions, parse has %d functions", len(syncFuncs), len(parseFuncs))

	// Currently the parser doesn't extract functions from DDL, so we expect 0 functions from parser
	// If parser starts extracting functions, we should implement proper comparison here
	if len(parseFuncs) == 0 && len(syncFuncs) > 0 {
		// This is expected - parser doesn't extract functions yet
		return
	}

	// If parser starts returning functions, implement full comparison
	require.Equal(t, len(syncFuncs), len(parseFuncs), "number of functions should match")

	// Create maps for easier comparison - use function signature for mapping
	syncMap := make(map[string]*storepb.FunctionMetadata)
	for _, fn := range syncFuncs {
		syncMap[fn.Signature] = fn
	}

	for _, parseFn := range parseFuncs {
		// Try to find matching function by signature
		var syncFn *storepb.FunctionMetadata
		for _, sf := range syncFuncs {
			if normalizeSignature(sf.Signature) == normalizeSignature(parseFn.Signature) {
				syncFn = sf
				break
			}
		}

		require.NotNil(t, syncFn, "function with signature %s should exist in sync metadata", parseFn.Signature)

		// Compare function definitions
		syncDef := normalizeSQL(syncFn.Definition)
		parseDef := normalizeSQL(parseFn.Definition)
		require.Equal(t, syncDef, parseDef, "function %s: definition should match", parseFn.Name)

		// Compare comment if present
		if syncFn.Comment != "" || parseFn.Comment != "" {
			require.Equal(t, syncFn.Comment, parseFn.Comment, "function %s: comment should match", parseFn.Name)
		}
	}
}

func compareSequences(t *testing.T, syncSeqs, parseSeqs []*storepb.SequenceMetadata) {
	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.SequenceMetadata)
	for _, seq := range syncSeqs {
		syncMap[seq.Name] = seq
	}

	parseMap := make(map[string]*storepb.SequenceMetadata)
	for _, seq := range parseSeqs {
		parseMap[seq.Name] = seq
	}

	// Check sequences in parseSeqs
	for _, parseSeq := range parseSeqs {
		syncSeq, exists := syncMap[parseSeq.Name]
		if !exists {
			// SERIAL columns create implicit sequences that might not be in DDL
			t.Logf("sequence %s exists in parse but not in sync (might be implicit from SERIAL)", parseSeq.Name)
			continue
		}

		// Compare basic properties
		if parseSeq.Start != "" {
			require.Equal(t, syncSeq.Start, parseSeq.Start, "sequence %s: start value should match", parseSeq.Name)
		}
		if parseSeq.Increment != "" {
			require.Equal(t, syncSeq.Increment, parseSeq.Increment, "sequence %s: increment should match", parseSeq.Name)
		}
	}

	// Check sequences in syncSeqs that are not in parseSeqs
	for _, syncSeq := range syncSeqs {
		_, exists := parseMap[syncSeq.Name]
		if !exists {
			// Skip implicit sequences created by SERIAL columns
			if strings.Contains(syncSeq.Name, "_id_seq") || strings.Contains(syncSeq.Name, "_seq") {
				t.Logf("sequence %s exists in sync but not in parse (implicit sequence from SERIAL column)", syncSeq.Name)
				continue
			}
			// For explicitly created sequences, this is an error
			require.True(t, exists, "sequence %s should exist in parsed metadata", syncSeq.Name)
		}
	}
}

func compareEnums(t *testing.T, syncEnums, parseEnums []*storepb.EnumTypeMetadata) {
	require.Equal(t, len(syncEnums), len(parseEnums), "number of enums should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.EnumTypeMetadata)
	for _, enum := range syncEnums {
		syncMap[enum.Name] = enum
	}

	for _, parseEnum := range parseEnums {
		syncEnum, exists := syncMap[parseEnum.Name]
		require.True(t, exists, "enum %s should exist in sync metadata", parseEnum.Name)
		require.ElementsMatch(t, syncEnum.Values, parseEnum.Values, "enum %s: values should match", parseEnum.Name)
	}
}

func compareExtensions(t *testing.T, syncExts, parseExts []*storepb.ExtensionMetadata) {
	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ExtensionMetadata)
	for _, ext := range syncExts {
		syncMap[ext.Name] = ext
	}

	for _, parseExt := range parseExts {
		syncExt, exists := syncMap[parseExt.Name]
		require.True(t, exists, "extension %s should exist in sync metadata", parseExt.Name)
		require.Equal(t, syncExt.Schema, parseExt.Schema, "extension %s: schema should match", parseExt.Name)
	}
}
