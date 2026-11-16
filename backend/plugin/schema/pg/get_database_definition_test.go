package pg

import (
	"os"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

type getDatabaseDefinitionCase struct {
	Description string `yaml:"description"`
	Metadata    string `yaml:"metadata"`
	Expected    string `yaml:"expected"`
}

// TestGetDatabaseDefinition pins the DDL generated for each metadata input,
// and reloads that DDL into an omni catalog to prove it is valid PostgreSQL.
//
// The input is metadata rather than a schema text on purpose: deriving it by
// parsing would cap this test's reach at whatever the loader happens to extract,
// so any generator branch nothing feeds would go untested.
func TestGetDatabaseDefinition(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/get_database_definition.yaml"
	)

	var tests []getDatabaseDefinitionCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			var metadata metadatapb.DatabaseSchemaMetadata
			require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), &metadata))

			definition, err := GetDatabaseDefinition(schema.GetDefinitionContext{PrintHeader: true}, &metadata)
			require.NoError(t, err)
			require.NotEmpty(t, definition)

			_, err = catalog.New().Exec(definition, &catalog.ExecOptions{ContinueOnError: true})
			require.NoError(t, err, "generated definition should load into the omni catalog")

			// A YAML block scalar cannot carry a leading newline, so the blank
			// line this header opens with is trimmed rather than pinned.
			definition = strings.TrimPrefix(definition, "\n")

			if record {
				tests[i].Expected = definition
				return
			}
			require.Equal(t, tc.Expected, definition)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

func TestParseQualifiedTypeIdent(t *testing.T) {
	testCases := []struct {
		input  string
		schema string
		name   string
		ok     bool
	}{
		{"integer", "", "", false},
		{"character varying(50)", "", "", false},
		{"numeric(9,6)", "", "", false},
		{"text[]", "", "", false},
		{"timestamp(6) with time zone", "", "", false},
		{"public.addr", "public", "addr", true},
		{"public.addr[]", "public", "addr", true},
		{"public.addr(5)", "public", "addr", true},
		{`"My Schema"."Weird Type"`, "My Schema", "Weird Type", true},
		{`"has""quote".t`, `has"quote`, "t", true},
		{`other.geo`, "other", "geo", true},
	}
	for _, tc := range testCases {
		schemaName, typeName, ok := parseQualifiedTypeIdent(tc.input)
		assert.Equal(t, tc.ok, ok, "input %q", tc.input)
		assert.Equal(t, tc.schema, schemaName, "input %q", tc.input)
		assert.Equal(t, tc.name, typeName, "input %q", tc.input)
	}
}

func TestSortCompositeTypesTopologically(t *testing.T) {
	// Chain across schemas: public.a3 -> other.b2 -> public.c1, with names
	// chosen so alphabetical order is the reverse of dependency order.
	c1 := &metadatapb.CompositeTypeMetadata{
		Name:       "c1",
		Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "x", Type: "integer"}},
	}
	b2 := &metadatapb.CompositeTypeMetadata{
		Name:       "b2",
		Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "c", Type: "public.c1"}},
	}
	a3 := &metadatapb.CompositeTypeMetadata{
		Name:       "a3",
		Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "b", Type: "other.b2[]"}},
	}
	ordered := sortCompositeTypesTopologically([]qualifiedCompositeType{
		{Schema: "public", Composite: a3},
		{Schema: "other", Composite: b2},
		{Schema: "public", Composite: c1},
	})
	require.Len(t, ordered, 3)
	assert.Equal(t, "c1", ordered[0].Composite.Name)
	assert.Equal(t, "b2", ordered[1].Composite.Name)
	assert.Equal(t, "a3", ordered[2].Composite.Name)

	// Independent types come out in deterministic (schema, name) order.
	ordered = sortCompositeTypesTopologically([]qualifiedCompositeType{
		{Schema: "public", Composite: &metadatapb.CompositeTypeMetadata{Name: "zz"}},
		{Schema: "public", Composite: &metadatapb.CompositeTypeMetadata{Name: "aa"}},
	})
	require.Len(t, ordered, 2)
	assert.Equal(t, "aa", ordered[0].Composite.Name)
	assert.Equal(t, "zz", ordered[1].Composite.Name)
}

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
