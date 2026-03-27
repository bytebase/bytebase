package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestExtensionInSDLOutput tests that extensions are included in SDL output
func TestExtensionInSDLOutput(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Extensions: []*storepb.ExtensionMetadata{
			{
				Name:        "uuid-ossp",
				Schema:      "public",
				Version:     "1.1",
				Description: "UUID generation extension",
			},
			{
				Name:    "pg_trgm",
				Version: "1.6",
			},
		},
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
			},
		},
	}

	// Test single-file SDL format
	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err)

	// Verify extensions are present
	require.Contains(t, sdl, "CREATE EXTENSION")
	require.Contains(t, sdl, "uuid-ossp")
	require.Contains(t, sdl, "pg_trgm")
	require.Contains(t, sdl, `WITH SCHEMA "public"`)
	require.Contains(t, sdl, `VERSION '1.1'`)
	require.Contains(t, sdl, `VERSION '1.6'`)

	// Verify comment
	require.Contains(t, sdl, "COMMENT ON EXTENSION")
	require.Contains(t, sdl, "UUID generation extension")
}

// TestExtensionInMultiFileSDL tests that extensions are included in multi-file SDL
func TestExtensionInMultiFileSDL(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Extensions: []*storepb.ExtensionMetadata{
			{
				Name:        "postgis",
				Schema:      "public",
				Version:     "3.3.2",
				Description: "PostGIS geometry and geography types",
			},
		},
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
			},
		},
	}

	// Test multi-file SDL format
	result, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the extension file
	var extensionFile *schema.File
	for i := range result.Files {
		if result.Files[i].Name == "extensions.sql" {
			extensionFile = &result.Files[i]
			break
		}
	}
	require.NotNil(t, extensionFile, "Extension file not found")

	// Verify content includes extension
	require.Contains(t, extensionFile.Content, "CREATE EXTENSION")
	require.Contains(t, extensionFile.Content, "postgis")
	require.Contains(t, extensionFile.Content, `WITH SCHEMA "public"`)
	require.Contains(t, extensionFile.Content, `VERSION '3.3.2'`)
	require.Contains(t, extensionFile.Content, "COMMENT ON EXTENSION")
	require.Contains(t, extensionFile.Content, "PostGIS geometry and geography types")
}

// TestExtensionDependencyOrdering tests that extensions have correct dependency ordering in migrations
func TestExtensionDependencyOrdering(t *testing.T) {
	t.Run("CREATE: Extension before Enum Types and Tables", func(t *testing.T) {
		// Test that when creating an extension,
		// CREATE EXTENSION comes before CREATE TYPE and CREATE TABLE

		oldMetadata := &storepb.DatabaseSchemaMetadata{
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
				},
			},
		}

		newMetadata := &storepb.DatabaseSchemaMetadata{
			Extensions: []*storepb.ExtensionMetadata{
				{
					Name: "uuid-ossp",
				},
			},
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					EnumTypes: []*storepb.EnumTypeMetadata{
						{
							Name:   "status_type",
							Values: []string{"active", "inactive"},
						},
					},
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "id",
									Type: "uuid",
								},
								{
									Name: "status",
									Type: "status_type",
								},
							},
						},
					},
				},
			},
		}

		// Convert to model.DatabaseSchema
		oldSchema := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_POSTGRES, false)
		newSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_POSTGRES, false)

		// Get diff
		diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, oldSchema, newSchema)
		require.NoError(t, err)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)
		require.NotEmpty(t, migration)

		// Find positions of CREATE EXTENSION, CREATE TYPE, and CREATE TABLE
		extensionPos := -1
		typePos := -1
		tablePos := -1
		for i := 0; i < len(migration); i++ {
			if extensionPos == -1 && i+len("CREATE EXTENSION") <= len(migration) {
				if migration[i:i+len("CREATE EXTENSION")] == "CREATE EXTENSION" {
					extensionPos = i
				}
			}
			if typePos == -1 && i+len("CREATE TYPE") <= len(migration) {
				if migration[i:i+len("CREATE TYPE")] == "CREATE TYPE" {
					typePos = i
				}
			}
			if tablePos == -1 && i+len("CREATE TABLE") <= len(migration) {
				if migration[i:i+len("CREATE TABLE")] == "CREATE TABLE" {
					tablePos = i
				}
			}
		}

		require.NotEqual(t, -1, extensionPos, "CREATE EXTENSION not found in migration")
		require.NotEqual(t, -1, typePos, "CREATE TYPE not found in migration")
		require.NotEqual(t, -1, tablePos, "CREATE TABLE not found in migration")
		require.Less(t, extensionPos, typePos, "CREATE EXTENSION should come before CREATE TYPE")
		require.Less(t, extensionPos, tablePos, "CREATE EXTENSION should come before CREATE TABLE")
	})

	t.Run("DROP: Tables and Enum Types before Extension", func(t *testing.T) {
		// Test that when dropping an extension,
		// DROP TABLE and DROP TYPE come before DROP EXTENSION

		oldMetadata := &storepb.DatabaseSchemaMetadata{
			Extensions: []*storepb.ExtensionMetadata{
				{
					Name: "uuid-ossp",
				},
			},
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					EnumTypes: []*storepb.EnumTypeMetadata{
						{
							Name:   "status_type",
							Values: []string{"active", "inactive"},
						},
					},
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "id",
									Type: "uuid",
								},
								{
									Name: "status",
									Type: "status_type",
								},
							},
						},
					},
				},
			},
		}

		newMetadata := &storepb.DatabaseSchemaMetadata{
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
				},
			},
		}

		// Convert to model.DatabaseSchema
		oldSchema := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_POSTGRES, false)
		newSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_POSTGRES, false)

		// Get diff
		diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, oldSchema, newSchema)
		require.NoError(t, err)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)
		require.NotEmpty(t, migration)

		// Find positions of DROP TABLE, DROP TYPE, and DROP EXTENSION
		tablePos := -1
		typePos := -1
		extensionPos := -1
		for i := 0; i < len(migration); i++ {
			if tablePos == -1 && i+len("DROP TABLE") <= len(migration) {
				if migration[i:i+len("DROP TABLE")] == "DROP TABLE" {
					tablePos = i
				}
			}
			if typePos == -1 && i+len("DROP TYPE") <= len(migration) {
				if migration[i:i+len("DROP TYPE")] == "DROP TYPE" {
					typePos = i
				}
			}
			if extensionPos == -1 && i+len("DROP EXTENSION") <= len(migration) {
				if migration[i:i+len("DROP EXTENSION")] == "DROP EXTENSION" {
					extensionPos = i
				}
			}
		}

		require.NotEqual(t, -1, tablePos, "DROP TABLE not found in migration")
		require.NotEqual(t, -1, typePos, "DROP TYPE not found in migration")
		require.NotEqual(t, -1, extensionPos, "DROP EXTENSION not found in migration")
		require.Less(t, tablePos, extensionPos, "DROP TABLE should come before DROP EXTENSION")
		require.Less(t, typePos, extensionPos, "DROP TYPE should come before DROP EXTENSION")
	})
}

// TestExtensionNotFilteredByArchiveSchemaFilter tests that extension changes are preserved
// when FilterPostgresArchiveSchema is applied (bb rollout workflow uses this filter)
func TestExtensionNotFilteredByArchiveSchemaFilter(t *testing.T) {
	// Create a diff with extension changes
	diff := &schema.MetadataDiff{
		DatabaseName: "testdb",
		ExtensionChanges: []*schema.ExtensionDiff{
			{
				Action:        schema.MetadataDiffActionCreate,
				ExtensionName: "citext",
				NewExtension: &storepb.ExtensionMetadata{
					Name:   "citext",
					Schema: "public",
				},
			},
		},
		TableChanges: []*schema.TableDiff{
			{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: "public",
				TableName:  "test_table",
			},
			{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: "bbdataarchive", // This should be filtered
				TableName:  "archive_table",
			},
		},
	}

	// Apply the archive schema filter (this is what bb rollout does)
	filtered := schema.FilterPostgresArchiveSchema(diff)

	// Verify extension changes are preserved (database-level objects)
	require.Equal(t, 1, len(filtered.ExtensionChanges), "Extension changes should be preserved")
	require.Equal(t, "citext", filtered.ExtensionChanges[0].ExtensionName)

	// Verify archive schema tables are filtered out
	require.Equal(t, 1, len(filtered.TableChanges), "Only non-archive tables should remain")
	require.Equal(t, "public", filtered.TableChanges[0].SchemaName)
	require.Equal(t, "test_table", filtered.TableChanges[0].TableName)
}

// TestExtensionSchemaSyncBug reproduces the customer-reported bug where Schema Sync:
// 1. Incorrectly generates DROP EXTENSION even though the extension is identical in source and target
// 2. Generates CREATE SCHEMA for a phantom schema named after the extension
//
