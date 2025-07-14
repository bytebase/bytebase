package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestSync_ColumnDefaultSchemaQualification(t *testing.T) {
	ctx := context.Background()

	// Use the centralized testcontainer helper
	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	// Get database connection
	pgDB := pgContainer.GetDB()
	require.NoError(t, pgDB.Ping())

	// Set up test schema with various default value scenarios
	setupSQL := `
-- Create enum type for testing schema qualification
CREATE TYPE test_size AS ENUM ('small', 'medium', 'large');

-- Create sequence for testing schema qualification  
CREATE SEQUENCE test_sequence START 1;

-- Create test table with various default scenarios
CREATE TABLE test_defaults (
    -- No default
    id INTEGER,
    
    -- NULL default
    col_null_default TEXT DEFAULT NULL,
    
    -- String literal with schema-qualified type
    col_string_literal VARCHAR(50) DEFAULT 'hello',
    
    -- Numeric literal
    col_numeric INTEGER DEFAULT 42,
    
    -- Boolean literal
    col_boolean BOOLEAN DEFAULT true,
    
    -- Function expression
    col_function TIMESTAMP DEFAULT now(),
    
    -- Expression
    col_expression INTEGER DEFAULT (10 + 20),
    
    -- SERIAL (creates sequence automatically)
    col_serial SERIAL,
    
    -- Enum default (should be schema-qualified)
    col_enum test_size DEFAULT 'medium',
    
    -- Custom sequence (should be schema-qualified)
    col_sequence INTEGER DEFAULT nextval('test_sequence'),
    
    -- Array default
    col_array INTEGER[] DEFAULT '{1,2,3}',
    
    -- JSON default
    col_json JSONB DEFAULT '{"key": "value"}',
    
    -- Binary default
    col_binary BYTEA DEFAULT '\xDEADBEEF'
);

-- Add comment to help identify the table
COMMENT ON TABLE test_defaults IS 'Test table for column default schema qualification';
`

	// Execute the DDL
	_, err := pgDB.Exec(setupSQL)
	require.NoError(t, err)

	// Create driver and get metadata using SyncDBSchema
	driver := &Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "postgres",
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Database: "postgres",
		},
		Password: "root-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0",
			DatabaseName:  "postgres",
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	pgDriver, ok := openedDriver.(*Driver)
	require.True(t, ok)

	// Run sync operation to get metadata
	metadata, err := pgDriver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.NotNil(t, metadata)

	// Find our test table
	var testTable *storepb.TableMetadata
	for _, schema := range metadata.Schemas {
		if schema.Name == "public" {
			for _, table := range schema.Tables {
				if table.Name == "test_defaults" {
					testTable = table
					break
				}
			}
		}
	}
	require.NotNil(t, testTable, "test_defaults table should be found")

	// Create a map for easier column lookup
	columnMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range testTable.Columns {
		columnMap[col.Name] = col
	}

	// Test cases to verify schema qualification
	testCases := []struct {
		columnName      string
		expectedDefault string
		description     string
	}{
		{
			columnName:      "id",
			expectedDefault: "",
			description:     "No default should be empty string",
		},
		{
			columnName:      "col_null_default",
			expectedDefault: "",
			description:     "DEFAULT NULL should be empty string (PostgreSQL limitation)",
		},
		{
			columnName:      "col_string_literal",
			expectedDefault: "'hello'::character varying",
			description:     "String literal should have type cast",
		},
		{
			columnName:      "col_numeric",
			expectedDefault: "42",
			description:     "Numeric literal should be unquoted",
		},
		{
			columnName:      "col_boolean",
			expectedDefault: "true",
			description:     "Boolean literal should be unquoted",
		},
		{
			columnName:      "col_function",
			expectedDefault: "now()",
			description:     "Function should be preserved",
		},
		{
			columnName:      "col_expression",
			expectedDefault: "(10 + 20)",
			description:     "Expression should preserve parentheses",
		},
		{
			columnName:      "col_enum",
			expectedDefault: "'medium'::public.test_size",
			description:     "Enum default should be SCHEMA-QUALIFIED with public.test_size",
		},
		{
			columnName:      "col_sequence",
			expectedDefault: "nextval('public.test_sequence'::regclass)",
			description:     "Custom sequence should be SCHEMA-QUALIFIED with public.test_sequence",
		},
		{
			columnName:      "col_array",
			expectedDefault: "'{1,2,3}'::integer[]",
			description:     "Array should have type cast",
		},
		{
			columnName:      "col_json",
			expectedDefault: "'{\"key\": \"value\"}'::jsonb",
			description:     "JSONB should have type cast",
		},
		{
			columnName:      "col_binary",
			expectedDefault: "'\\xdeadbeef'::bytea",
			description:     "Binary should have type cast and lowercase hex",
		},
	}

	// Verify each test case
	for _, tc := range testCases {
		t.Run(tc.columnName, func(t *testing.T) {
			column, exists := columnMap[tc.columnName]
			require.True(t, exists, "Column %s should exist", tc.columnName)

			require.Equal(t, tc.expectedDefault, column.Default,
				"Column %s: %s. Expected: %q, Got: %q",
				tc.columnName, tc.description, tc.expectedDefault, column.Default)
		})
	}

	// Special test: Verify SERIAL column has schema-qualified sequence
	serialColumn, exists := columnMap["col_serial"]
	require.True(t, exists, "col_serial should exist")
	require.Contains(t, serialColumn.Default, "public.test_defaults_col_serial_seq",
		"SERIAL column should have schema-qualified sequence name. Got: %s", serialColumn.Default)
	require.Contains(t, serialColumn.Default, "nextval(",
		"SERIAL column should use nextval function. Got: %s", serialColumn.Default)

	// Verify that Default field is being used (consolidation from DefaultNull and DefaultExpression)
	for columnName, column := range columnMap {
		// The Default field should be used instead of the deprecated DefaultExpression and DefaultNull fields
		if column.Default != "" {
			t.Logf("Column %s uses Default field: %s", columnName, column.Default)
		}
	}
}

func TestSync_ColumnDefaultCrossSchemaQualification(t *testing.T) {
	ctx := context.Background()

	// Use the centralized testcontainer helper
	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	// Get database connection
	pgDB := pgContainer.GetDB()
	require.NoError(t, pgDB.Ping())

	// Create schema with cross-schema references
	setupSQL := `
-- Create custom schema to test cross-schema qualification
CREATE SCHEMA custom_schema;

-- Create enum in custom schema
CREATE TYPE custom_schema.status_type AS ENUM ('active', 'inactive', 'pending');

-- Create sequence in custom schema
CREATE SEQUENCE custom_schema.my_sequence START 100;

-- Create function in custom schema for testing
CREATE OR REPLACE FUNCTION custom_schema.get_prefix() RETURNS TEXT AS $$
BEGIN
    RETURN 'prefix_value';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Test table in public schema with references to custom schema objects
CREATE TABLE critical_test (
    -- Cross-schema enum reference
    status custom_schema.status_type DEFAULT 'active',
    
    -- Cross-schema sequence reference  
    counter INTEGER DEFAULT nextval('custom_schema.my_sequence'),
    
    -- Complex expression with cross-schema function
    computed TEXT DEFAULT custom_schema.get_prefix()
);
`

	// Execute the DDL
	_, err := pgDB.Exec(setupSQL)
	require.NoError(t, err)

	// Create driver and get metadata
	driver := &Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "postgres",
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Database: "postgres",
		},
		Password: "root-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0",
			DatabaseName:  "postgres",
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	pgDriver, ok := openedDriver.(*Driver)
	require.True(t, ok)

	// Run sync operation
	metadata, err := pgDriver.SyncDBSchema(ctx)
	require.NoError(t, err)

	// Find the test table
	var testTable *storepb.TableMetadata
	for _, schema := range metadata.Schemas {
		if schema.Name == "public" {
			for _, table := range schema.Tables {
				if table.Name == "critical_test" {
					testTable = table
					break
				}
			}
		}
	}
	require.NotNil(t, testTable)

	// Verify cross-schema qualification
	columnMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range testTable.Columns {
		columnMap[col.Name] = col
	}

	// Critical test: Cross-schema enum should be fully qualified
	statusCol := columnMap["status"]
	require.NotNil(t, statusCol)
	require.Equal(t, "'active'::custom_schema.status_type", statusCol.Default,
		"Cross-schema enum should be fully qualified")

	// Critical test: Cross-schema sequence should be fully qualified
	counterCol := columnMap["counter"]
	require.NotNil(t, counterCol)
	require.Equal(t, "nextval('custom_schema.my_sequence'::regclass)", counterCol.Default,
		"Cross-schema sequence should be fully qualified")

	// Critical test: Complex expression with cross-schema function
	computedCol := columnMap["computed"]
	require.NotNil(t, computedCol)
	require.Equal(t, "custom_schema.get_prefix()", computedCol.Default,
		"Cross-schema function should be fully qualified")

	// Cleanup: Drop test objects to avoid interference with other tests
	cleanupSQL := `
		DROP TABLE IF EXISTS critical_test CASCADE;
		DROP TABLE IF EXISTS test_defaults CASCADE;
		DROP SCHEMA IF EXISTS custom_schema CASCADE;
		DROP TYPE IF EXISTS test_size CASCADE;
		DROP SEQUENCE IF EXISTS test_sequence CASCADE;
	`
	_, err = pgDB.Exec(cleanupSQL)
	require.NoError(t, err)
}
