package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDefinitionProcedure(t *testing.T) {
	tests := []struct {
		name        string
		definition  string
		isProcedure bool
	}{
		{
			name: "Simple FUNCTION",
			definition: `CREATE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "Simple PROCEDURE",
			definition: `CREATE PROCEDURE test_procedure()
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: true,
		},
		{
			name: "CREATE OR REPLACE FUNCTION",
			definition: `CREATE OR REPLACE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "CREATE OR REPLACE PROCEDURE",
			definition: `CREATE OR REPLACE PROCEDURE test_procedure()
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: true,
		},
		{
			name: "FUNCTION with word 'procedure' in comment - should NOT be confused",
			definition: `CREATE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	-- This is not a procedure, it's a function
	-- The word PROCEDURE appears in this comment
	RAISE NOTICE 'This function is not a procedure';
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "FUNCTION with word 'procedure' in string literal - should NOT be confused",
			definition: `CREATE FUNCTION test_function()
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
	RETURN 'This function is not a PROCEDURE';
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "PROCEDURE with complex body",
			definition: `CREATE OR REPLACE PROCEDURE update_user_data(
	user_id INTEGER,
	new_name VARCHAR
)
LANGUAGE plpgsql
AS $$
DECLARE
	old_name VARCHAR;
BEGIN
	-- Get old name
	SELECT name INTO old_name FROM users WHERE id = user_id;

	-- Update the user name
	UPDATE users SET name = new_name WHERE id = user_id;

	-- Log the change
	INSERT INTO audit_log (message) VALUES (
		'Changed user name from ' || old_name || ' to ' || new_name
	);
END;
$$;`,
			isProcedure: true,
		},
		{
			name: "FUNCTION with RETURNS TABLE",
			definition: `CREATE OR REPLACE FUNCTION get_users()
RETURNS TABLE(id INTEGER, name VARCHAR)
LANGUAGE plpgsql
AS $$
BEGIN
	RETURN QUERY SELECT users.id, users.name FROM users;
END;
$$;`,
			isProcedure: false,
		},
		{
			name:        "Empty definition",
			definition:  "",
			isProcedure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDefinitionProcedure(tt.definition)
			assert.Equal(t, tt.isProcedure, result,
				"isDefinitionProcedure returned %v, expected %v for:\n%s",
				result, tt.isProcedure, tt.definition)
		})
	}
}

// TestIsDefinitionProcedureRobustness tests edge cases that would fail with string-based detection
func TestIsDefinitionProcedureRobustness(t *testing.T) {
	tests := []struct {
		name        string
		definition  string
		isProcedure bool
		reason      string
	}{
		{
			name: "FUNCTION with 'CREATE PROCEDURE' in string literal",
			definition: `CREATE FUNCTION test_function()
RETURNS text
LANGUAGE sql
AS $outer$
	SELECT 'Example: CREATE PROCEDURE foo() LANGUAGE plpgsql AS $body$ BEGIN NULL; END; $body$;'
$outer$;`,
			isProcedure: false,
			reason:      "String-based detection might incorrectly identify this as a PROCEDURE due to 'CREATE PROCEDURE' in the string literal",
		},
		{
			name: "FUNCTION with 'PROCEDURE' in multi-line comment",
			definition: `CREATE FUNCTION complex_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	/*
	 * This function performs complex operations
	 * Note: This is NOT a PROCEDURE
	 * PROCEDURE keyword appears in this comment
	 * But it should still be detected as a FUNCTION
	 */
	RAISE NOTICE 'Processing...';
END;
$$;`,
			isProcedure: false,
			reason:      "Multi-line comments with PROCEDURE keyword should not confuse the detector",
		},
		{
			name: "PROCEDURE with unusual formatting",
			definition: `CREATE    OR    REPLACE    PROCEDURE
test_procedure
(
)
LANGUAGE plpgsql
AS
$$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: true,
			reason:      "Should handle unusual whitespace and formatting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDefinitionProcedure(tt.definition)
			assert.Equal(t, tt.isProcedure, result,
				"Test failed: %s\nGot: %v, Expected: %v",
				tt.reason, result, tt.isProcedure)
		})
	}
}
