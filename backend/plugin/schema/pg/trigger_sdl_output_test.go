package pg

import (
	"strings"
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
								Name: "audit_trigger",
								Body: "CREATE TRIGGER audit_trigger AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
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

	assert.Contains(t, result, "CREATE TRIGGER audit_trigger")
	assert.Contains(t, result, "AFTER INSERT ON public.users")
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
								Name: "audit_trigger",
								Body: "CREATE TRIGGER audit_trigger AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
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
	assert.Contains(t, usersFileContent, "CREATE TRIGGER audit_trigger")
	assert.Contains(t, usersFileContent, "AFTER INSERT ON public.users")
}

func TestTriggerSDLWithCompleteDefinition(t *testing.T) {
	// Test that triggers with complete CREATE TRIGGER statements in Body are output correctly
	// This simulates triggers dumped from the database where Body contains the full statement
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "ci_builds",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*storepb.TriggerMetadata{
							{
								Name: "ci_builds_loose_fk_trigger",
								Body: "CREATE TRIGGER ci_builds_loose_fk_trigger AFTER DELETE ON public.ci_builds REFERENCING OLD TABLE AS old_table FOR EACH STATEMENT EXECUTE FUNCTION public.loose_foreign_key_on_builds_projects()",
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

	// Should contain only one CREATE TRIGGER (not duplicated)
	createTriggerCount := strings.Count(result, "CREATE TRIGGER ci_builds_loose_fk_trigger")
	assert.Equal(t, 1, createTriggerCount, "CREATE TRIGGER should appear exactly once, not duplicated")

	// Should contain the complete trigger definition
	assert.Contains(t, result, "AFTER DELETE ON public.ci_builds")
	assert.Contains(t, result, "REFERENCING OLD TABLE AS old_table")
	assert.Contains(t, result, "FOR EACH STATEMENT")
	assert.Contains(t, result, "EXECUTE FUNCTION public.loose_foreign_key_on_builds_projects()")
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
								Body:    "CREATE TRIGGER audit_trigger AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
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

	assert.Contains(t, result, "CREATE TRIGGER audit_trigger")
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
								Name: "audit_insert",
								Body: "CREATE TRIGGER audit_insert AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
							},
							{
								Name: "audit_update",
								Body: "CREATE TRIGGER audit_update AFTER UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
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
