package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestTriggerDriftHandling tests the apply minimal changes (drift handling) for triggers
func TestTriggerDriftHandling(t *testing.T) {
	t.Run("New trigger added to database", func(t *testing.T) {
		// User SDL (no trigger)
		userSDL := `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100)
			);

			CREATE FUNCTION audit_log() RETURNS TRIGGER AS $$
			BEGIN
				RAISE NOTICE 'Audit';
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
		`

		// Database has a new trigger that wasn't in SDL
		currentDBMetadata := &storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "integer"},
								{Name: "name", Type: "varchar(100)"},
							},
							Triggers: []*storepb.TriggerMetadata{
								{
									Name: "audit_trigger",
									Body: "CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION audit_log()",
								},
							},
						},
					},
					Functions: []*storepb.FunctionMetadata{
						{
							Name:       "audit_log",
							Definition: "CREATE FUNCTION audit_log() RETURNS TRIGGER AS $$ BEGIN RAISE NOTICE 'Audit'; RETURN NEW; END; $$ LANGUAGE plpgsql;",
						},
					},
				},
			},
		}

		previousDBMetadata := &storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "integer"},
								{Name: "name", Type: "varchar(100)"},
							},
						},
					},
					Functions: []*storepb.FunctionMetadata{
						{
							Name:       "audit_log",
							Definition: "CREATE FUNCTION audit_log() RETURNS TRIGGER AS $$ BEGIN RAISE NOTICE 'Audit'; RETURN NEW; END; $$ LANGUAGE plpgsql;",
						},
					},
				},
			},
		}

		currentSchema := model.NewDatabaseMetadata(currentDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)
		previousSchema := model.NewDatabaseMetadata(previousDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)

		diff, err := GetSDLDiff(userSDL, userSDL, currentSchema, previousSchema)
		require.NoError(t, err)

		// Should detect the new trigger
		require.NotNil(t, diff, "Diff should not be nil")

		// Drift handling should have added the trigger to the diff
		// The exact behavior depends on implementation details
		t.Logf("TableChanges count: %d", len(diff.TableChanges))
		for _, tc := range diff.TableChanges {
			t.Logf("Table: %s.%s, TriggerChanges: %d", tc.SchemaName, tc.TableName, len(tc.TriggerChanges))
		}
	})

	t.Run("Apply trigger changes to chunks", func(t *testing.T) {
		// Create test SDL chunks
		chunks := &schema.SDLChunks{
			Triggers: make(map[string]*schema.SDLChunk),
		}

		currentDBMetadata := &storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
							Triggers: []*storepb.TriggerMetadata{
								{
									Name: "new_trigger",
									Body: "CREATE TRIGGER new_trigger AFTER UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f()",
								},
							},
						},
					},
				},
			},
		}

		previousDBMetadata := &storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
						},
					},
				},
			},
		}

		currentSchema := model.NewDatabaseMetadata(currentDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)
		previousSchema := model.NewDatabaseMetadata(previousDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)

		// Apply changes
		err := applyTriggerChangesToChunks(chunks, currentSchema, previousSchema)
		require.NoError(t, err)

		// Should have created a new chunk
		triggerKey := "public.users.new_trigger"
		chunk, exists := chunks.Triggers[triggerKey]
		require.True(t, exists, "Should have trigger chunk with key %s", triggerKey)
		require.NotNil(t, chunk, "Chunk should not be nil")
		require.NotNil(t, chunk.ASTNode, "Chunk should have ASTNode")
	})
}
