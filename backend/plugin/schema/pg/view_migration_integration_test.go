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
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
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
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
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
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
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
