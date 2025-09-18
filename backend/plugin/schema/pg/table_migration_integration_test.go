package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestTableSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name:        "Create new table",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255) UNIQUE,
					created_at TIMESTAMP DEFAULT NOW()
				);
			`,
			expectedMigration: `CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255) UNIQUE,
					created_at TIMESTAMP DEFAULT NOW()
				);

`,
		},
		{
			name: "Drop table",
			previousSDL: `
				CREATE TABLE old_table (
					id INTEGER PRIMARY KEY,
					data TEXT
				);
			`,
			currentSDL: ``,
			expectedMigration: `DROP TABLE IF EXISTS "public"."old_table";
`,
		},
		{
			name:        "Create table with constraints",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) CHECK (price > 0),
					category_id INTEGER,
					CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
				);
			`,
			expectedMigration: `CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) CHECK (price > 0),
					category_id INTEGER,
					CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
				);

`,
		},
		{
			name:        "Create schema-qualified table",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE test_schema.items (
					id BIGSERIAL PRIMARY KEY,
					description TEXT
				);
			`,
			expectedMigration: `CREATE TABLE test_schema.items (
					id BIGSERIAL PRIMARY KEY,
					description TEXT
				);

`,
		},
		{
			name: "Multiple tables with different operations",
			previousSDL: `
				CREATE TABLE table_a (id INTEGER PRIMARY KEY);
				CREATE TABLE table_b (id INTEGER PRIMARY KEY);
			`,
			currentSDL: `
				CREATE TABLE table_a (id INTEGER PRIMARY KEY);
				CREATE TABLE table_c (id INTEGER PRIMARY KEY);
			`,
			expectedMigration: `DROP TABLE IF EXISTS "public"."table_b";

CREATE TABLE table_c (id INTEGER PRIMARY KEY);

`,
		},
		{
			name:        "Create table with various data types",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE data_types_test (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					title VARCHAR(100),
					content TEXT,
					count INTEGER,
					amount DECIMAL(15,2),
					is_active BOOLEAN DEFAULT false,
					created_at TIMESTAMPTZ DEFAULT NOW(),
					metadata JSONB
				);
			`,
			expectedMigration: `CREATE TABLE data_types_test (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					title VARCHAR(100),
					content TEXT,
					count INTEGER,
					amount DECIMAL(15,2),
					is_active BOOLEAN DEFAULT false,
					created_at TIMESTAMPTZ DEFAULT NOW(),
					metadata JSONB
				);

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Get SDL diff using AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Step 2: Verify that diff contains table changes with AST nodes
			if len(diff.TableChanges) > 0 {
				for _, tableDiff := range diff.TableChanges {
					switch tableDiff.Action {
					case schema.MetadataDiffActionCreate:
						assert.NotNil(t, tableDiff.NewASTNode,
							"Create action should have NewASTNode")
						assert.Nil(t, tableDiff.OldASTNode,
							"Create action should not have OldASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, tableDiff.NewTable,
							"AST-only mode should not have metadata")
					case schema.MetadataDiffActionDrop:
						assert.NotNil(t, tableDiff.OldASTNode,
							"Drop action should have OldASTNode")
						assert.Nil(t, tableDiff.NewASTNode,
							"Drop action should not have NewASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, tableDiff.OldTable,
							"AST-only mode should not have metadata")
					default:
						// Other actions like ALTER
						t.Logf("Encountered table action: %v", tableDiff.Action)
					}
				}
			}

			// Step 3: Generate migration SQL using AST nodes
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			// Step 4: Verify the generated migration matches expectations
			assert.Equal(t, tt.expectedMigration, migrationSQL,
				"Generated migration SQL should match expected output")

			// Step 5: Verify the migration contains expected keywords for tables
			if tt.expectedMigration != "" {
				if containsTableString(tt.expectedMigration, "CREATE TABLE") {
					assert.Contains(t, migrationSQL, "CREATE TABLE",
						"Migration should contain CREATE TABLE statement")
				}
				if containsTableString(tt.expectedMigration, "DROP TABLE") {
					assert.Contains(t, migrationSQL, "DROP TABLE",
						"Migration should contain DROP TABLE statement")
				}
			}
		})
	}
}

func TestTableMigrationASTOnlyModeValidation(t *testing.T) {
	// Test that ensures AST-only mode works correctly without any metadata extraction
	previousSDL := ``
	currentSDL := `
		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'active'
		);
	`

	// Get diff without metadata extraction
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly one table change
	assert.Len(t, diff.TableChanges, 1)
	tableDiff := diff.TableChanges[0]

	// Verify AST-only mode properties
	assert.Equal(t, schema.MetadataDiffActionCreate, tableDiff.Action)
	assert.Equal(t, "public", tableDiff.SchemaName)
	assert.Equal(t, "test_table", tableDiff.TableName)

	// Critical assertion: No metadata should be present (AST-only mode)
	assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")
	assert.Nil(t, tableDiff.OldTable, "AST-only mode should not extract metadata")

	// But AST nodes should be present
	assert.NotNil(t, tableDiff.NewASTNode, "AST node should be present for CREATE action")
	assert.Nil(t, tableDiff.OldASTNode, "No old AST node for CREATE action")

	// Generate migration should work with AST nodes only
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.Contains(t, migrationSQL, "CREATE TABLE test_table")
	assert.Contains(t, migrationSQL, "id SERIAL PRIMARY KEY")
	assert.Contains(t, migrationSQL, "name VARCHAR(255) NOT NULL")
	assert.Contains(t, migrationSQL, "status VARCHAR(50) DEFAULT 'active'")
}

func TestMultipleTablesHandling(t *testing.T) {
	// Test that verifies multiple tables are handled correctly
	previousSDL := `
		CREATE TABLE table1 (id INTEGER PRIMARY KEY);
	`
	currentSDL := `
		CREATE TABLE table1 (id INTEGER PRIMARY KEY);
		CREATE TABLE table2 (id SERIAL PRIMARY KEY, name TEXT);
		CREATE TABLE table3 (id UUID PRIMARY KEY, data JSONB);
	`

	// Get diff without metadata extraction (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly two table changes (CREATE actions for table2 and table3)
	assert.Len(t, diff.TableChanges, 2)

	// Verify both tables are CREATE actions
	createCount := 0
	tableNames := make(map[string]bool)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			createCount++
			// Verify AST-only mode properties
			assert.NotNil(t, tableDiff.NewASTNode, "AST node should be present for CREATE action")
			assert.Nil(t, tableDiff.OldASTNode, "No old AST node for CREATE action")
			assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")

			// Collect table names to verify they're different
			tableNames[tableDiff.TableName] = true
		}
	}
	assert.Equal(t, 2, createCount, "Should have two CREATE actions")

	// Verify the table names are correct
	expectedTables := map[string]bool{
		"table2": true,
		"table3": true,
	}
	assert.Equal(t, expectedTables, tableNames, "Should have correct table names")

	// Generate migration SQL - this should work with both tables
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	// Verify the migration contains both tables
	assert.Contains(t, migrationSQL, "CREATE TABLE table2")
	assert.Contains(t, migrationSQL, "CREATE TABLE table3")

	// Verify both tables are created
	tableCount := strings.Count(migrationSQL, "CREATE TABLE")
	assert.Equal(t, 2, tableCount, "Should create both tables")
}

func TestComplexTableWithConstraintsAndIndexes(t *testing.T) {
	// Test with complex table definition including constraints and indexes
	previousSDL := ``
	currentSDL := `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			order_date DATE DEFAULT CURRENT_DATE,
			total_amount DECIMAL(12,2) NOT NULL CHECK (total_amount >= 0),
			status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered')),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			CONSTRAINT unique_customer_date UNIQUE (customer_id, order_date)
		);
	`

	// Get diff in AST-only mode
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have one table creation
	assert.Len(t, diff.TableChanges, 1)
	tableDiff := diff.TableChanges[0]

	// Verify properties
	assert.Equal(t, schema.MetadataDiffActionCreate, tableDiff.Action)
	assert.Equal(t, "public", tableDiff.SchemaName)
	assert.Equal(t, "orders", tableDiff.TableName)
	assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")
	assert.NotNil(t, tableDiff.NewASTNode, "AST node should be present")

	// Generate migration
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	// Verify the migration contains the complex table definition
	assert.Contains(t, migrationSQL, "CREATE TABLE orders")
	assert.Contains(t, migrationSQL, "id SERIAL PRIMARY KEY")
	assert.Contains(t, migrationSQL, "total_amount DECIMAL(12,2) NOT NULL CHECK (total_amount >= 0)")
	assert.Contains(t, migrationSQL, "CONSTRAINT fk_customer FOREIGN KEY")
	assert.Contains(t, migrationSQL, "CONSTRAINT unique_customer_date UNIQUE")
}

// Helper function to check if a string contains a substring (table tests)
func containsTableString(s, substr string) bool {
	return strings.Contains(s, substr)
}
