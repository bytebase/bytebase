package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestTriggerSDLSingleFileOutput(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*storepb.TriggerMetadata{
							{
								Name:   "audit_trigger",
								Timing: "AFTER",
								Event:  "INSERT",
								Body:   "FOR EACH ROW EXECUTE FUNCTION audit_log()",
							},
						},
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)

	assert.Contains(t, result, "CREATE TRIGGER")
	assert.Contains(t, result, "audit_trigger")
	assert.Contains(t, result, "AFTER INSERT ON")
	assert.Contains(t, result, "FOR EACH ROW EXECUTE FUNCTION audit_log()")
}

func TestTriggerSDLMultiFileOutput(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*storepb.TriggerMetadata{
							{
								Name:   "audit_trigger",
								Timing: "AFTER",
								Event:  "INSERT",
								Body:   "FOR EACH ROW EXECUTE FUNCTION audit_log()",
							},
						},
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetMultiFileDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the users table file
	var usersFileContent string
	for _, file := range result.Files {
		if file.Name == "schemas/public/tables/users.sql" {
			usersFileContent = file.Content
			break
		}
	}

	require.NotEmpty(t, usersFileContent, "Should have users table file")
	assert.Contains(t, usersFileContent, "CREATE TRIGGER")
	assert.Contains(t, usersFileContent, "audit_trigger")
	assert.Contains(t, usersFileContent, "AFTER INSERT ON")
}

func TestTriggerSDLWithComment(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*storepb.TriggerMetadata{
							{
								Name:    "audit_trigger",
								Timing:  "AFTER",
								Event:   "INSERT",
								Body:    "FOR EACH ROW EXECUTE FUNCTION audit_log()",
								Comment: "Audit log trigger",
							},
						},
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)

	assert.Contains(t, result, "CREATE TRIGGER")
	assert.Contains(t, result, "COMMENT ON TRIGGER")
	assert.Contains(t, result, "audit_trigger")
	assert.Contains(t, result, "Audit log trigger")
}

func TestMultipleTriggersSDL(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*storepb.TriggerMetadata{
							{
								Name:   "audit_insert",
								Timing: "AFTER",
								Event:  "INSERT",
								Body:   "FOR EACH ROW EXECUTE FUNCTION audit_log()",
							},
							{
								Name:   "audit_update",
								Timing: "AFTER",
								Event:  "UPDATE",
								Body:   "FOR EACH ROW EXECUTE FUNCTION audit_log()",
							},
						},
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)

	assert.Contains(t, result, "audit_insert")
	assert.Contains(t, result, "audit_update")
}
