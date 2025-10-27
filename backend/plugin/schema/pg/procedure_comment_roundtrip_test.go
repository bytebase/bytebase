package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestProcedureCommentRoundtrip tests the full cycle:
// 1. Database metadata with function + procedure (both with comments)
// 2. Generate SDL from metadata (this is what "bytebase sdl dump" does)
// 3. Use that SDL as "previous"
// 4. Create "current" SDL with procedure removed
// 5. Generate migration
// 6. Verify function comment is NOT touched
func TestProcedureCommentRoundtrip(t *testing.T) {
	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	// Step 1: Create database metadata with both function and procedure (with comments)
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "test_db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Functions: []*storepb.FunctionMetadata{
					{
						Name:      "test_function",
						Signature: "test_function()",
						Definition: `CREATE OR REPLACE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $function$
BEGIN
	RAISE NOTICE 'Test function executed';
END
$function$;`,
						Comment: "A test function that raises a notice",
					},
					{
						Name:      "new_procedure",
						Signature: "new_procedure()",
						Definition: `CREATE PROCEDURE new_procedure()
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
						Comment: "A test procedure",
					},
				},
			},
		},
	}

	// Step 2: Generate SDL from metadata (simulating "bytebase sdl dump")
	previousSDL, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err, "Failed to generate SDL from metadata")

	t.Logf("Generated SDL from metadata:\n%s", previousSDL)

	// Verify the generated SDL contains both objects with their comments
	assert.Contains(t, previousSDL, "CREATE OR REPLACE FUNCTION", "Should contain function")
	assert.Contains(t, previousSDL, "CREATE PROCEDURE", "Should contain procedure")
	assert.Contains(t, previousSDL, "COMMENT ON FUNCTION", "Should contain function comment")
	assert.Contains(t, previousSDL, "COMMENT ON PROCEDURE", "Should contain procedure comment")

	// Step 3: Create current SDL with procedure removed (only function remains)
	metadataCurrent := &storepb.DatabaseSchemaMetadata{
		Name: "test_db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Functions: []*storepb.FunctionMetadata{
					{
						Name:      "test_function",
						Signature: "test_function()",
						Definition: `CREATE OR REPLACE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $function$
BEGIN
	RAISE NOTICE 'Test function executed';
END
$function$;`,
						Comment: "A test function that raises a notice",
					},
				},
			},
		},
	}

	currentSDL, err := GetDatabaseDefinition(ctx, metadataCurrent)
	require.NoError(t, err, "Failed to generate current SDL")

	t.Logf("Current SDL (procedure removed):\n%s", currentSDL)

	// Step 4: Generate diff and migration
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err, "Failed to generate SDL diff")

	// Log the diff for debugging
	t.Logf("Function changes: %d", len(diff.FunctionChanges))
	for i, fc := range diff.FunctionChanges {
		t.Logf("  [%d] Action=%v, Schema=%s, Function=%s", i, fc.Action, fc.SchemaName, fc.FunctionName)
	}

	t.Logf("Comment changes: %d", len(diff.CommentChanges))
	for i, cc := range diff.CommentChanges {
		t.Logf("  [%d] Action=%v, Type=%v, Schema=%s, Object=%s, OldComment=%q, NewComment=%q",
			i, cc.Action, cc.ObjectType, cc.SchemaName, cc.ObjectName, cc.OldComment, cc.NewComment)
	}

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err, "Failed to generate migration")

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Step 5: Verify the migration is correct
	// Should drop the procedure
	assert.Contains(t, migrationSQL, "DROP PROCEDURE IF EXISTS", "Should drop procedure")
	assert.Contains(t, migrationSQL, "new_procedure", "Should mention new_procedure")

	// Should NOT touch the function comment
	// This is the key assertion - we should NOT see any COMMENT statement about test_function
	if strings.Contains(migrationSQL, "test_function") && strings.Contains(migrationSQL, "COMMENT") {
		t.Errorf("Migration incorrectly contains COMMENT statement about test_function:\n%s", migrationSQL)
	}

	// Should NOT contain "COMMENT ON FUNCTION ... IS NULL"
	assert.NotContains(t, migrationSQL, "IS NULL", "Should not try to remove any comments")
}
