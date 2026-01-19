package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestViewSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name:        "Create new view",
			previousSDL: ``,
			currentSDL: `				
				CREATE VIEW active_users AS 
				SELECT 1 as id, 'test' as name;
			`,
			expectedMigration: `CREATE VIEW active_users AS 
				SELECT 1 as id, 'test' as name;

`,
		},
		{
			name: "Drop view",
			previousSDL: `				
				CREATE VIEW active_users AS 
				SELECT 1 as id, 'test' as name;
			`,
			currentSDL: ``,
			expectedMigration: `DROP VIEW IF EXISTS "public"."active_users";
`,
		},
		{
			name: "Modify view (drop and recreate)",
			previousSDL: `				
				CREATE VIEW active_users AS 
				SELECT 1 as id, 'test' as name;
			`,
			currentSDL: `				
				CREATE VIEW active_users AS 
				SELECT 2 as id, 'updated' as name, 'extra' as email;
			`,
			expectedMigration: `DROP VIEW IF EXISTS "public"."active_users";

CREATE VIEW active_users AS 
				SELECT 2 as id, 'updated' as name, 'extra' as email;

`,
		},
		{
			name:        "Schema-qualified view",
			previousSDL: ``,
			currentSDL: `				
				CREATE VIEW test_schema.expensive_products AS
				SELECT 1 as id, 'product' as name, 150.00 as price;
			`,
			expectedMigration: `CREATE VIEW test_schema.expensive_products AS
				SELECT 1 as id, 'product' as name, 150.00 as price;

`,
		},
		{
			name: "Multiple view changes",
			previousSDL: `
				CREATE VIEW user_summary AS 
				SELECT 1 as id, 'user' as name;
				
				CREATE VIEW order_summary AS 
				SELECT 1 as id, 100.00 as amount;
			`,
			currentSDL: `
				CREATE VIEW user_summary AS 
				SELECT 2 as id, 'updated_user' as name, 'active' as status;
				
				CREATE VIEW order_analytics AS 
				SELECT 1 as user_id, 500.00 as total_amount;
			`,
			expectedMigration: `DROP VIEW IF EXISTS "public"."order_summary";
DROP VIEW IF EXISTS "public"."user_summary";

CREATE VIEW order_analytics AS 
				SELECT 1 as user_id, 500.00 as total_amount;

CREATE VIEW user_summary AS 
				SELECT 2 as id, 'updated_user' as name, 'active' as status;

`,
		},
		{
			name:        "View with dependencies (AST-only mode)",
			previousSDL: ``,
			currentSDL: `
				CREATE VIEW dependent_view AS 
				SELECT id, name, 'derived' as type FROM base_view;
				
				CREATE VIEW base_view AS 
				SELECT 1 as id, 'base' as name;
			`,
			expectedMigration: `CREATE VIEW base_view AS 
				SELECT 1 as id, 'base' as name;

CREATE VIEW dependent_view AS 
				SELECT id, name, 'derived' as type FROM base_view;

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Get SDL diff using AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Step 2: Verify that diff contains view changes with AST nodes
			if len(diff.ViewChanges) > 0 {
				for _, viewDiff := range diff.ViewChanges {
					switch viewDiff.Action {
					case schema.MetadataDiffActionCreate:
						assert.NotNil(t, viewDiff.NewASTNode,
							"Create action should have NewASTNode")
						assert.Nil(t, viewDiff.OldASTNode,
							"Create action should not have OldASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, viewDiff.NewView,
							"AST-only mode should not have metadata")
					case schema.MetadataDiffActionDrop:
						assert.NotNil(t, viewDiff.OldASTNode,
							"Drop action should have OldASTNode")
						assert.Nil(t, viewDiff.NewASTNode,
							"Drop action should not have NewASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, viewDiff.OldView,
							"AST-only mode should not have metadata")
					default:
						// Other actions like MetadataDiffActionAlter
						t.Logf("Encountered view action: %v", viewDiff.Action)
					}
				}
			}

			// Step 3: Generate migration SQL using AST nodes
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			// Step 4: Verify the generated migration matches expectations
			assert.Equal(t, tt.expectedMigration, migrationSQL,
				"Generated migration SQL should match expected output")

			// Step 5: Verify the migration is valid PostgreSQL SQL
			// We can do a basic check to ensure it contains expected keywords
			if tt.expectedMigration != "" {
				if contains(tt.expectedMigration, "CREATE VIEW") {
					assert.Contains(t, migrationSQL, "CREATE VIEW",
						"Migration should contain CREATE VIEW statement")
				}
				if contains(tt.expectedMigration, "DROP VIEW") {
					assert.Contains(t, migrationSQL, "DROP VIEW",
						"Migration should contain DROP VIEW statement")
				}
			}
		})
	}
}

func TestViewMigrationASTOnlyModeValidation(t *testing.T) {
	// Test that ensures AST-only mode works correctly without any metadata extraction
	previousSDL := ``
	currentSDL := `		
		CREATE VIEW product_view AS SELECT 1 as id, 'product' as name;
	`

	// Get diff without metadata extraction
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly one view change
	assert.Len(t, diff.ViewChanges, 1)
	viewDiff := diff.ViewChanges[0]

	// Verify AST-only mode properties
	assert.Equal(t, schema.MetadataDiffActionCreate, viewDiff.Action)
	assert.Equal(t, "public", viewDiff.SchemaName)
	assert.Equal(t, "product_view", viewDiff.ViewName)

	// Critical assertion: No metadata should be present (AST-only mode)
	assert.Nil(t, viewDiff.NewView, "AST-only mode should not extract metadata")
	assert.Nil(t, viewDiff.OldView, "AST-only mode should not extract metadata")

	// But AST nodes should be present
	assert.NotNil(t, viewDiff.NewASTNode, "AST node should be present for CREATE action")
	assert.Nil(t, viewDiff.OldASTNode, "No old AST node for CREATE action")

	// Generate migration should work with AST nodes only
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.Contains(t, migrationSQL, "CREATE VIEW product_view AS SELECT 1 as id, 'product' as name")
}

func TestViewDependencyHandlingInAST(t *testing.T) {
	// Test that verifies dependency extraction works in AST-only mode
	// This test creates two views where one depends on the other
	previousSDL := ``
	currentSDL := `
		-- Create a base view first
		CREATE VIEW base_data AS 
		SELECT 1 as id, 'item1' as name, 100 as value;
		
		-- Create a dependent view that references the base view
		CREATE VIEW summary_report AS 
		SELECT b.name, b.value, b.value * 2 as doubled_value 
		FROM base_data b 
		WHERE b.value > 50;
	`

	// Get diff without metadata extraction (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly two view changes (CREATE actions)
	assert.Len(t, diff.ViewChanges, 2)

	// Verify both views are CREATE actions with AST nodes
	createCount := 0
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			createCount++
			// Verify AST-only mode properties
			assert.NotNil(t, viewDiff.NewASTNode, "AST node should be present for CREATE action")
			assert.Nil(t, viewDiff.OldASTNode, "No old AST node for CREATE action")
			assert.Nil(t, viewDiff.NewView, "AST-only mode should not extract metadata")
		}
	}
	assert.Equal(t, 2, createCount, "Should have two CREATE actions")

	// Generate migration SQL - this should work with dependency resolution
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	// Verify the migration contains both views
	assert.Contains(t, migrationSQL, "CREATE VIEW base_data AS")
	assert.Contains(t, migrationSQL, "CREATE VIEW summary_report AS")

	// The important test: base_data should be created before summary_report
	// due to dependency resolution
	baseIndex := strings.Index(migrationSQL, "CREATE VIEW base_data AS")
	summaryIndex := strings.Index(migrationSQL, "CREATE VIEW summary_report AS")
	assert.True(t, baseIndex < summaryIndex && baseIndex >= 0 && summaryIndex >= 0,
		"base_data should be created before summary_report due to dependency ordering. Migration SQL:\n%s", migrationSQL)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestDropTableAndDependentView_CorrectOrder(t *testing.T) {
	// This test verifies that when dropping both a table and a view that depends on it,
	// the DROP statements are generated in the correct order:
	// 1. DROP VIEW first (dependent object)
	// 2. DROP TABLE second (base object)
	//
	// This is critical because PostgreSQL will fail if we try to drop the table first
	// while the view still depends on it.

	previousSDL := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);

		CREATE VIEW active_users AS
		SELECT id, name, email
		FROM users
		WHERE email IS NOT NULL;
	`

	// Both table and view are removed in current SDL
	currentSDL := ``

	// Get the diff
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have both table and view changes
	assert.Len(t, diff.TableChanges, 1, "Should have one table change")
	assert.Len(t, diff.ViewChanges, 1, "Should have one view change")

	// Generate migration SQL
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Verify both DROP statements are present
	assert.Contains(t, migrationSQL, "DROP VIEW", "Should contain DROP VIEW statement")
	assert.Contains(t, migrationSQL, "DROP TABLE", "Should contain DROP TABLE statement")

	// The critical test: VIEW must be dropped BEFORE TABLE
	// Find the positions of each DROP statement
	viewDropIndex := strings.Index(migrationSQL, "DROP VIEW")
	tableDropIndex := strings.Index(migrationSQL, "DROP TABLE")

	assert.True(t, viewDropIndex >= 0, "DROP VIEW statement should be present")
	assert.True(t, tableDropIndex >= 0, "DROP TABLE statement should be present")
	assert.True(t, viewDropIndex < tableDropIndex,
		"DROP VIEW must come before DROP TABLE because view depends on table.\n"+
			"Current order is incorrect: VIEW at position %d, TABLE at position %d\n"+
			"Migration SQL:\n%s",
		viewDropIndex, tableDropIndex, migrationSQL)
}

func TestDropMultipleTablesAndViews_CorrectOrder(t *testing.T) {
	// Test with multiple tables and views to ensure proper dependency ordering

	previousSDL := `
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			category_id INTEGER REFERENCES categories(id)
		);

		CREATE VIEW product_summary AS
		SELECT p.id, p.name, c.name as category_name
		FROM products p
		JOIN categories c ON p.category_id = c.id;
	`

	// Remove all objects
	currentSDL := ``

	// Get the diff
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Generate migration SQL
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Verify the view is dropped before both tables
	viewDropIndex := strings.Index(migrationSQL, "DROP VIEW")
	productsTableDropIndex := strings.Index(migrationSQL, `DROP TABLE IF EXISTS "public"."products"`)
	categoriesTableDropIndex := strings.Index(migrationSQL, `DROP TABLE IF EXISTS "public"."categories"`)

	assert.True(t, viewDropIndex >= 0, "DROP VIEW should be present")
	assert.True(t, productsTableDropIndex >= 0, "DROP TABLE products should be present")
	assert.True(t, categoriesTableDropIndex >= 0, "DROP TABLE categories should be present")

	// View must be dropped first (this is the critical fix)
	assert.True(t, viewDropIndex < productsTableDropIndex,
		"DROP VIEW must come before DROP TABLE products. Migration SQL:\n%s", migrationSQL)
	assert.True(t, viewDropIndex < categoriesTableDropIndex,
		"DROP VIEW must come before DROP TABLE categories. Migration SQL:\n%s", migrationSQL)

	// Note: In AST-only mode, we cannot reliably extract foreign key dependencies,
	// so table DROP order may not respect FK constraints. The FKs are dropped before tables anyway,
	// so the migration will succeed even if the table order is not optimal.
	t.Log("Note: Products should be dropped before categories (FK dependency), but AST-only mode may not enforce this")
}

func TestCreateTableAndDependentView_CorrectOrder(t *testing.T) {
	// This test verifies that when creating both a table and a view that depends on it,
	// the CREATE statements are generated in the correct order:
	// 1. CREATE TABLE first (base object)
	// 2. CREATE VIEW second (dependent object)
	//
	// This is critical because PostgreSQL will fail if we try to create the view first
	// before the table exists.

	previousSDL := ``

	// Both table and view are added in current SDL
	currentSDL := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);

		CREATE VIEW active_users AS
		SELECT id, name, email
		FROM users
		WHERE email IS NOT NULL;
	`

	// Get the diff
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have both table and view changes
	assert.Len(t, diff.TableChanges, 1, "Should have one table change")
	assert.Len(t, diff.ViewChanges, 1, "Should have one view change")

	// Generate migration SQL
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Verify both CREATE statements are present
	assert.Contains(t, migrationSQL, "CREATE TABLE", "Should contain CREATE TABLE statement")
	assert.Contains(t, migrationSQL, "CREATE VIEW", "Should contain CREATE VIEW statement")

	// The critical test: TABLE must be created BEFORE VIEW
	// Find the positions of each CREATE statement
	tableCreateIndex := strings.Index(migrationSQL, "CREATE TABLE")
	viewCreateIndex := strings.Index(migrationSQL, "CREATE VIEW")

	assert.True(t, tableCreateIndex >= 0, "CREATE TABLE statement should be present")
	assert.True(t, viewCreateIndex >= 0, "CREATE VIEW statement should be present")
	assert.True(t, tableCreateIndex < viewCreateIndex,
		"CREATE TABLE must come before CREATE VIEW because view depends on table.\n"+
			"Current order is incorrect: TABLE at position %d, VIEW at position %d\n"+
			"Migration SQL:\n%s",
		tableCreateIndex, viewCreateIndex, migrationSQL)
}

func TestCreateMultipleTablesAndViews_CorrectOrder(t *testing.T) {
	// Test with multiple tables and views to ensure proper dependency ordering

	previousSDL := ``

	// Create tables and a view that depends on them
	currentSDL := `
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			category_id INTEGER REFERENCES categories(id)
		);

		CREATE VIEW product_summary AS
		SELECT p.id, p.name, c.name as category_name
		FROM products p
		JOIN categories c ON p.category_id = c.id;
	`

	// Get the diff
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Generate migration SQL
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Verify both tables are created before the view
	categoriesTableCreateIndex := strings.Index(migrationSQL, `CREATE TABLE categories`)
	productsTableCreateIndex := strings.Index(migrationSQL, `CREATE TABLE products`)
	viewCreateIndex := strings.Index(migrationSQL, "CREATE VIEW")

	assert.True(t, categoriesTableCreateIndex >= 0, "CREATE TABLE categories should be present")
	assert.True(t, productsTableCreateIndex >= 0, "CREATE TABLE products should be present")
	assert.True(t, viewCreateIndex >= 0, "CREATE VIEW should be present")

	// Both tables must be created before the view (this is the critical fix)
	assert.True(t, categoriesTableCreateIndex < viewCreateIndex,
		"CREATE TABLE categories must come before CREATE VIEW. Migration SQL:\n%s", migrationSQL)
	assert.True(t, productsTableCreateIndex < viewCreateIndex,
		"CREATE TABLE products must come before CREATE VIEW. Migration SQL:\n%s", migrationSQL)

	// Note: The FK constraint means categories should be created before products,
	// and the topological sort should handle this correctly
	assert.True(t, categoriesTableCreateIndex < productsTableCreateIndex,
		"CREATE TABLE categories should come before CREATE TABLE products (FK dependency). Migration SQL:\n%s", migrationSQL)
}
