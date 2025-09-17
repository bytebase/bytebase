package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestFunctionSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name:        "Create new function",
			previousSDL: ``,
			currentSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedMigration: `CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;

`,
		},
		{
			name: "Drop function",
			previousSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: ``,
			expectedMigration: `DROP FUNCTION IF EXISTS "public"."calculate_area"(radius numeric);
`,
		},
		{
			name: "Modify function (CREATE OR REPLACE)",
			previousSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159265 * radius * radius;  -- More precise PI
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedMigration: `CREATE OR REPLACE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159265 * radius * radius;  -- More precise PI
				END;
				$$ LANGUAGE plpgsql;

`,
		},
		{
			name:        "Create new procedure (stored as function)",
			previousSDL: ``,
			currentSDL: `
				CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					-- Some procedure logic here
					NULL;
				END;
				$$;
			`,
			expectedMigration: `CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					-- Some procedure logic here
					NULL;
				END;
				$$;

`,
		},
		{
			name: "Overloaded functions with different signatures",
			previousSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER) 
				RETURNS INTEGER AS $$ 
				BEGIN 
					RETURN a + b; 
				END; 
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER) 
				RETURNS INTEGER AS $$ 
				BEGIN 
					RETURN a + b; 
				END; 
				$$ LANGUAGE plpgsql;
				
				CREATE FUNCTION add_numbers(a FLOAT, b FLOAT) 
				RETURNS FLOAT AS $$ 
				BEGIN 
					RETURN a + b; 
				END; 
				$$ LANGUAGE plpgsql;
			`,
			expectedMigration: `CREATE FUNCTION add_numbers(a FLOAT, b FLOAT) 
				RETURNS FLOAT AS $$ 
				BEGIN 
					RETURN a + b; 
				END; 
				$$ LANGUAGE plpgsql;

`,
		},
		{
			name: "Drop one overloaded function",
			previousSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER) 
				RETURNS INTEGER AS $$ 
				BEGIN 
					RETURN a + b; 
				END; 
				$$ LANGUAGE plpgsql;
				
				CREATE FUNCTION add_numbers(a FLOAT, b FLOAT) 
				RETURNS FLOAT AS $$ 
				BEGIN 
					RETURN a + b; 
				END; 
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER) 
				RETURNS INTEGER AS $$ 
				BEGIN 
					RETURN a + b; 
				END; 
				$$ LANGUAGE plpgsql;
			`,
			expectedMigration: `DROP FUNCTION IF EXISTS "public"."add_numbers"(a float, b float);
`,
		},
		{
			name: "Modify one overloaded function",
			previousSDL: `
				CREATE FUNCTION calculate(x INTEGER) 
				RETURNS INTEGER AS $$ 
				BEGIN 
					RETURN x * 2; 
				END; 
				$$ LANGUAGE plpgsql;
				
				CREATE FUNCTION calculate(x FLOAT) 
				RETURNS FLOAT AS $$ 
				BEGIN 
					RETURN x * 2.0; 
				END; 
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE FUNCTION calculate(x INTEGER) 
				RETURNS INTEGER AS $$ 
				BEGIN 
					RETURN x * 3; -- Changed logic
				END; 
				$$ LANGUAGE plpgsql;
				
				CREATE FUNCTION calculate(x FLOAT) 
				RETURNS FLOAT AS $$ 
				BEGIN 
					RETURN x * 2.0; 
				END; 
				$$ LANGUAGE plpgsql;
			`,
			expectedMigration: `CREATE OR REPLACE FUNCTION calculate(x INTEGER) 
				RETURNS INTEGER AS $$ 
				BEGIN 
					RETURN x * 3; -- Changed logic
				END; 
				$$ LANGUAGE plpgsql;

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Get SDL diff using AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Step 2: Verify that diff contains function changes with AST nodes
			if len(diff.FunctionChanges) > 0 {
				for _, funcDiff := range diff.FunctionChanges {
					switch funcDiff.Action {
					case schema.MetadataDiffActionCreate:
						assert.NotNil(t, funcDiff.NewASTNode,
							"Create action should have NewASTNode")
						assert.Nil(t, funcDiff.OldASTNode,
							"Create action should not have OldASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, funcDiff.NewFunction,
							"AST-only mode should not have metadata")
					case schema.MetadataDiffActionDrop:
						assert.NotNil(t, funcDiff.OldASTNode,
							"Drop action should have OldASTNode")
						assert.Nil(t, funcDiff.NewASTNode,
							"Drop action should not have NewASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, funcDiff.OldFunction,
							"AST-only mode should not have metadata")
					case schema.MetadataDiffActionAlter:
						assert.NotNil(t, funcDiff.OldASTNode,
							"Alter action should have OldASTNode")
						assert.NotNil(t, funcDiff.NewASTNode,
							"Alter action should have NewASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, funcDiff.OldFunction,
							"AST-only mode should not have metadata")
						assert.Nil(t, funcDiff.NewFunction,
							"AST-only mode should not have metadata")
					default:
						// Other actions
						t.Logf("Encountered function action: %v", funcDiff.Action)
					}
				}
			}

			// Step 3: Generate migration SQL using AST nodes
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			// Step 4: Verify the generated migration matches expectations
			assert.Equal(t, tt.expectedMigration, migrationSQL,
				"Generated migration SQL should match expected output")

			// Step 5: Verify the migration contains expected keywords for functions
			if tt.expectedMigration != "" {
				if containsString(tt.expectedMigration, "CREATE FUNCTION") {
					assert.Contains(t, migrationSQL, "CREATE FUNCTION",
						"Migration should contain CREATE FUNCTION statement")
				}
				if containsString(tt.expectedMigration, "CREATE PROCEDURE") {
					assert.Contains(t, migrationSQL, "CREATE PROCEDURE",
						"Migration should contain CREATE PROCEDURE statement")
				}
				if containsString(tt.expectedMigration, "DROP FUNCTION") {
					assert.Contains(t, migrationSQL, "DROP FUNCTION",
						"Migration should contain DROP FUNCTION statement")
				}
			}
		})
	}
}

func TestFunctionMigrationASTOnlyModeValidation(t *testing.T) {
	// Test that ensures AST-only mode works correctly without any metadata extraction
	previousSDL := ``
	currentSDL := `
		CREATE FUNCTION get_total(amount DECIMAL) RETURNS DECIMAL AS $$
		BEGIN
			RETURN amount * 1.1;
		END;
		$$ LANGUAGE plpgsql;
	`

	// Get diff without metadata extraction
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly one function change
	assert.Len(t, diff.FunctionChanges, 1)
	funcDiff := diff.FunctionChanges[0]

	// Verify AST-only mode properties
	assert.Equal(t, schema.MetadataDiffActionCreate, funcDiff.Action)
	assert.Equal(t, "public", funcDiff.SchemaName)
	assert.Equal(t, "get_total(amount numeric)", funcDiff.FunctionName)

	// Critical assertion: No metadata should be present (AST-only mode)
	assert.Nil(t, funcDiff.NewFunction, "AST-only mode should not extract metadata")
	assert.Nil(t, funcDiff.OldFunction, "AST-only mode should not extract metadata")

	// But AST nodes should be present
	assert.NotNil(t, funcDiff.NewASTNode, "AST node should be present for CREATE action")
	assert.Nil(t, funcDiff.OldASTNode, "No old AST node for CREATE action")

	// Generate migration should work with AST nodes only
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.Contains(t, migrationSQL, "CREATE FUNCTION get_total(amount DECIMAL) RETURNS DECIMAL")
}

func TestOverloadedFunctionSignatureHandling(t *testing.T) {
	// Test that verifies signature-based identification works correctly in full flow
	previousSDL := ``
	currentSDL := `
		-- Create two functions with same name but different signatures
		CREATE FUNCTION format_value(val INTEGER) 
		RETURNS TEXT AS $$ 
		BEGIN 
			RETURN val::TEXT; 
		END; 
		$$ LANGUAGE plpgsql;
		
		CREATE FUNCTION format_value(val DECIMAL, precision_digits INTEGER) 
		RETURNS TEXT AS $$ 
		BEGIN 
			RETURN ROUND(val, precision_digits)::TEXT; 
		END; 
		$$ LANGUAGE plpgsql;
	`

	// Get diff without metadata extraction (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly two function changes (CREATE actions)
	assert.Len(t, diff.FunctionChanges, 2)

	// Verify both functions are CREATE actions with different signatures
	createCount := 0
	signatures := make(map[string]bool)
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate {
			createCount++
			// Verify AST-only mode properties
			assert.NotNil(t, funcDiff.NewASTNode, "AST node should be present for CREATE action")
			assert.Nil(t, funcDiff.OldASTNode, "No old AST node for CREATE action")
			assert.Nil(t, funcDiff.NewFunction, "AST-only mode should not extract metadata")

			// Collect signatures to verify they're different
			signatures[funcDiff.FunctionName] = true
		}
	}
	assert.Equal(t, 2, createCount, "Should have two CREATE actions")

	// Verify the signatures are different
	assert.Len(t, signatures, 2, "Should have two different function signatures")

	// Expected signatures
	expectedSignatures := map[string]bool{
		"format_value(val integer)":                           true,
		"format_value(val numeric, precision_digits integer)": true,
	}
	assert.Equal(t, expectedSignatures, signatures, "Should have correct function signatures")

	// Generate migration SQL - this should work with both functions
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	// Verify the migration contains both functions
	assert.Contains(t, migrationSQL, "format_value(val INTEGER)")
	assert.Contains(t, migrationSQL, "format_value(val DECIMAL, precision_digits INTEGER)")

	// Verify both functions are created
	functionCount := strings.Count(migrationSQL, "CREATE FUNCTION format_value")
	assert.Equal(t, 2, functionCount, "Should create both overloaded functions")
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
