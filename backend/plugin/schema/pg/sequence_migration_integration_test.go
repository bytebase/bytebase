package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestSequenceSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name:        "Create new sequence",
			previousSDL: ``,
			currentSDL: `
				CREATE SEQUENCE my_sequence
				START WITH 1
				INCREMENT BY 1
				MINVALUE 1
				MAXVALUE 9223372036854775807
				CACHE 1;
			`,
			expectedMigration: `CREATE SEQUENCE my_sequence
				START WITH 1
				INCREMENT BY 1
				MINVALUE 1
				MAXVALUE 9223372036854775807
				CACHE 1;

`,
		},
		{
			name: "Drop sequence",
			previousSDL: `
				CREATE SEQUENCE user_id_seq
				START WITH 1
				INCREMENT BY 1
				MINVALUE 1
				MAXVALUE 2147483647
				CACHE 1;
			`,
			currentSDL: ``,
			expectedMigration: `DROP SEQUENCE IF EXISTS "public"."user_id_seq";
`,
		},
		{
			name: "Modify sequence (drop and recreate)",
			previousSDL: `
				CREATE SEQUENCE order_seq
				START WITH 1
				INCREMENT BY 1
				CACHE 1;
			`,
			currentSDL: `
				CREATE SEQUENCE order_seq
				START WITH 100
				INCREMENT BY 5
				CACHE 10;
			`,
			expectedMigration: `DROP SEQUENCE IF EXISTS "public"."order_seq";

CREATE SEQUENCE order_seq
				START WITH 100
				INCREMENT BY 5
				CACHE 10;

`,
		},
		{
			name:        "Create sequence with data type",
			previousSDL: ``,
			currentSDL: `
				CREATE SEQUENCE bigint_sequence AS bigint
				START WITH 1
				INCREMENT BY 1;
			`,
			expectedMigration: `CREATE SEQUENCE bigint_sequence AS bigint
				START WITH 1
				INCREMENT BY 1;

`,
		},
		{
			name:        "Create schema-qualified sequence",
			previousSDL: ``,
			currentSDL: `
				CREATE SEQUENCE test_schema.seq_test
				START WITH 10
				INCREMENT BY 2;
			`,
			expectedMigration: `CREATE SEQUENCE test_schema.seq_test
				START WITH 10
				INCREMENT BY 2;

`,
		},
		{
			name: "Multiple sequences with different operations",
			previousSDL: `
				CREATE SEQUENCE seq_a START WITH 1;
				CREATE SEQUENCE seq_b START WITH 1;
			`,
			currentSDL: `
				CREATE SEQUENCE seq_a START WITH 1;
				CREATE SEQUENCE seq_c START WITH 1;
			`,
			expectedMigration: `DROP SEQUENCE IF EXISTS "public"."seq_b";

CREATE SEQUENCE seq_c START WITH 1;

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Get SDL diff using AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Step 2: Verify that diff contains sequence changes with AST nodes
			if len(diff.SequenceChanges) > 0 {
				for _, seqDiff := range diff.SequenceChanges {
					switch seqDiff.Action {
					case schema.MetadataDiffActionCreate:
						assert.NotNil(t, seqDiff.NewASTNode,
							"Create action should have NewASTNode")
						assert.Nil(t, seqDiff.OldASTNode,
							"Create action should not have OldASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, seqDiff.NewSequence,
							"AST-only mode should not have metadata")
					case schema.MetadataDiffActionDrop:
						assert.NotNil(t, seqDiff.OldASTNode,
							"Drop action should have OldASTNode")
						assert.Nil(t, seqDiff.NewASTNode,
							"Drop action should not have NewASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, seqDiff.OldSequence,
							"AST-only mode should not have metadata")
					default:
						// Other actions
						t.Logf("Encountered sequence action: %v", seqDiff.Action)
					}
				}
			}

			// Step 3: Generate migration SQL using AST nodes
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			// Step 4: Verify the generated migration matches expectations
			assert.Equal(t, tt.expectedMigration, migrationSQL,
				"Generated migration SQL should match expected output")

			// Step 5: Verify the migration contains expected keywords for sequences
			if tt.expectedMigration != "" {
				if containsSequenceString(tt.expectedMigration, "CREATE SEQUENCE") {
					assert.Contains(t, migrationSQL, "CREATE SEQUENCE",
						"Migration should contain CREATE SEQUENCE statement")
				}
				if containsSequenceString(tt.expectedMigration, "DROP SEQUENCE") {
					assert.Contains(t, migrationSQL, "DROP SEQUENCE",
						"Migration should contain DROP SEQUENCE statement")
				}
			}
		})
	}
}

func TestSequenceMigrationASTOnlyModeValidation(t *testing.T) {
	// Test that ensures AST-only mode works correctly without any metadata extraction
	previousSDL := ``
	currentSDL := `
		CREATE SEQUENCE test_sequence
		START WITH 100
		INCREMENT BY 2
		CACHE 5;
	`

	// Get diff without metadata extraction
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly one sequence change
	assert.Len(t, diff.SequenceChanges, 1)
	seqDiff := diff.SequenceChanges[0]

	// Verify AST-only mode properties
	assert.Equal(t, schema.MetadataDiffActionCreate, seqDiff.Action)
	assert.Equal(t, "public", seqDiff.SchemaName)
	assert.Equal(t, "test_sequence", seqDiff.SequenceName)

	// Critical assertion: No metadata should be present (AST-only mode)
	assert.Nil(t, seqDiff.NewSequence, "AST-only mode should not extract metadata")
	assert.Nil(t, seqDiff.OldSequence, "AST-only mode should not extract metadata")

	// But AST nodes should be present
	assert.NotNil(t, seqDiff.NewASTNode, "AST node should be present for CREATE action")
	assert.Nil(t, seqDiff.OldASTNode, "No old AST node for CREATE action")

	// Generate migration should work with AST nodes only
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.Contains(t, migrationSQL, "CREATE SEQUENCE test_sequence")
	assert.Contains(t, migrationSQL, "START WITH 100")
	assert.Contains(t, migrationSQL, "INCREMENT BY 2")
	assert.Contains(t, migrationSQL, "CACHE 5")
}

func TestMultipleSequencesHandling(t *testing.T) {
	// Test that verifies multiple sequences are handled correctly
	previousSDL := `
		CREATE SEQUENCE seq1 START WITH 1;
	`
	currentSDL := `
		CREATE SEQUENCE seq1 START WITH 1;
		CREATE SEQUENCE seq2 START WITH 10 INCREMENT BY 2;
		CREATE SEQUENCE seq3 AS bigint START WITH 1000;
	`

	// Get diff without metadata extraction (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly two sequence changes (CREATE actions for seq2 and seq3)
	assert.Len(t, diff.SequenceChanges, 2)

	// Verify both sequences are CREATE actions
	createCount := 0
	sequenceNames := make(map[string]bool)
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate {
			createCount++
			// Verify AST-only mode properties
			assert.NotNil(t, seqDiff.NewASTNode, "AST node should be present for CREATE action")
			assert.Nil(t, seqDiff.OldASTNode, "No old AST node for CREATE action")
			assert.Nil(t, seqDiff.NewSequence, "AST-only mode should not extract metadata")

			// Collect sequence names to verify they're different
			sequenceNames[seqDiff.SequenceName] = true
		}
	}
	assert.Equal(t, 2, createCount, "Should have two CREATE actions")

	// Verify the sequence names are correct
	expectedSequences := map[string]bool{
		"seq2": true,
		"seq3": true,
	}
	assert.Equal(t, expectedSequences, sequenceNames, "Should have correct sequence names")

	// Generate migration SQL - this should work with both sequences
	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	// Verify the migration contains both sequences
	assert.Contains(t, migrationSQL, "CREATE SEQUENCE seq2")
	assert.Contains(t, migrationSQL, "CREATE SEQUENCE seq3")

	// Verify both sequences are created
	sequenceCount := strings.Count(migrationSQL, "CREATE SEQUENCE")
	assert.Equal(t, 2, sequenceCount, "Should create both sequences")
}

// Helper function to check if a string contains a substring (sequence tests)
func containsSequenceString(s, substr string) bool {
	return strings.Contains(s, substr)
}
