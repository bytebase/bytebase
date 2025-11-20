package pg

import (
	"strings"
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

	t.Run("INITIALIZATION: First-time SDL with Extension", func(t *testing.T) {
		// This tests the initialization scenario: empty database, user writes first SDL
		// This is a very common bb rollout scenario

		// Previous SDL (empty - new database)
		previousSDL := ``

		// Current SDL (first-time setup with extension and other objects)
		currentSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA "public";

CREATE TYPE "public"."status_type" AS ENUM ('active', 'inactive');

CREATE TABLE "public"."users" (
	"id" uuid NOT NULL,
	"status" status_type NOT NULL
);`

		// Get SDL diff (simulates bb rollout initialization)
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Verify we have extension, enum, and table changes
		require.Equal(t, 1, len(diff.ExtensionChanges), "Should have 1 extension change")
		require.Equal(t, 1, len(diff.EnumTypeChanges), "Should have 1 enum type change")
		require.Equal(t, 1, len(diff.TableChanges), "Should have 1 table change")

		extDiff := diff.ExtensionChanges[0]
		require.Equal(t, schema.MetadataDiffActionCreate, extDiff.Action)
		require.Equal(t, "uuid-ossp", extDiff.ExtensionName)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify correct ordering: extension → enum → table
		extensionPos := strings.Index(migration, "CREATE EXTENSION")
		enumPos := strings.Index(migration, "CREATE TYPE")
		tablePos := strings.Index(migration, "CREATE TABLE")

		require.NotEqual(t, -1, extensionPos, "CREATE EXTENSION should be present")
		require.NotEqual(t, -1, enumPos, "CREATE TYPE should be present")
		require.NotEqual(t, -1, tablePos, "CREATE TABLE should be present")
		require.Less(t, extensionPos, enumPos, "CREATE EXTENSION should come before CREATE TYPE")
		require.Less(t, extensionPos, tablePos, "CREATE EXTENSION should come before CREATE TABLE")
	})

	t.Run("ADD Extension via SDL chunks", func(t *testing.T) {
		// This tests adding an extension to existing database via SDL diff

		// Previous SDL (existing database with table, no extension)
		previousSDL := `CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);`

		// Current SDL (with new extension)
		currentSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA "public";

CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);`

		// Get SDL diff (simulates bb rollout behavior)
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Debug: log all diffs
		t.Logf("ExtensionChanges: %d", len(diff.ExtensionChanges))
		for i, extDiff := range diff.ExtensionChanges {
			t.Logf("  Extension[%d]: Action=%v, Name=%s, HasNewAST=%v",
				i, extDiff.Action, extDiff.ExtensionName,
				extDiff.NewASTNode != nil)
		}

		// Verify we have an ExtensionDiff with CREATE action
		require.Equal(t, 1, len(diff.ExtensionChanges), "Should have 1 extension change")
		extDiff := diff.ExtensionChanges[0]
		require.Equal(t, schema.MetadataDiffActionCreate, extDiff.Action)
		require.Equal(t, "uuid-ossp", extDiff.ExtensionName)
		require.NotNil(t, extDiff.NewASTNode, "NewASTNode should be set for SDL mode")

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify migration creates the extension
		require.Contains(t, migration, "CREATE EXTENSION")
		require.Contains(t, migration, "uuid-ossp")
	})

	t.Run("REMOVE Extension via SDL chunks", func(t *testing.T) {
		// This tests removing an extension via SDL diff

		// Previous SDL (with extension)
		previousSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE "public"."users" (
	"id" uuid NOT NULL,
	"name" text
);`

		// Current SDL (extension removed)
		currentSDL := `CREATE TABLE "public"."users" (
	"id" uuid NOT NULL,
	"name" text
);`

		// Get SDL diff
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Verify we have an ExtensionDiff with DROP action
		require.Equal(t, 1, len(diff.ExtensionChanges), "Should have 1 extension change")
		extDiff := diff.ExtensionChanges[0]
		require.Equal(t, schema.MetadataDiffActionDrop, extDiff.Action)
		require.Equal(t, "uuid-ossp", extDiff.ExtensionName)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify migration drops the extension
		require.Contains(t, migration, "DROP EXTENSION")
		require.Contains(t, migration, "uuid-ossp")
	})

	t.Run("MODIFY Extension via SDL chunks (DROP + CREATE)", func(t *testing.T) {
		// This tests modifying an extension via SDL diff
		// PostgreSQL doesn't support modifying extensions easily, so we use DROP + CREATE pattern

		// Previous SDL (with original version)
		previousSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" VERSION '1.0';`

		// Current SDL (with modified version)
		currentSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" VERSION '1.1';`

		// Get SDL diff
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Verify we have two ExtensionDiff entries: DROP then CREATE
		require.Equal(t, 2, len(diff.ExtensionChanges), "Should have 2 extension changes (DROP + CREATE)")
		dropDiff := diff.ExtensionChanges[0]
		createDiff := diff.ExtensionChanges[1]

		require.Equal(t, schema.MetadataDiffActionDrop, dropDiff.Action)
		require.Equal(t, "uuid-ossp", dropDiff.ExtensionName)

		require.Equal(t, schema.MetadataDiffActionCreate, createDiff.Action)
		require.Equal(t, "uuid-ossp", createDiff.ExtensionName)
		require.NotNil(t, createDiff.NewASTNode, "NewASTNode should be set for SDL mode")

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify migration has both DROP and CREATE
		require.Contains(t, migration, "DROP EXTENSION")
		require.Contains(t, migration, "CREATE EXTENSION")
		require.Contains(t, migration, "uuid-ossp")
	})
}

// TestExtensionCommentChanges tests that extension comment changes are properly detected and generated
func TestExtensionCommentChanges(t *testing.T) {
	// Previous SDL (extension without comment)
	previousSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

	// Current SDL (extension with comment)
	currentSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

COMMENT ON EXTENSION "uuid-ossp" IS 'UUID generation functions';`

	// Get SDL diff
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Verify we have a comment change
	hasCommentDiff := false
	for _, commentDiff := range diff.CommentChanges {
		if commentDiff.ObjectType == schema.CommentObjectTypeExtension &&
			commentDiff.ObjectName == "uuid-ossp" {
			hasCommentDiff = true
			break
		}
	}
	require.True(t, hasCommentDiff, "Should have comment diff for extension")

	// Generate migration
	migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	require.NoError(t, err)
	t.Logf("Generated migration:\n%s", migration)

	// Verify migration includes comment
	require.Contains(t, migration, "COMMENT ON EXTENSION")
	require.Contains(t, migration, "uuid-ossp")
	require.Contains(t, migration, "UUID generation functions")
}

// TestExtensionWithCitextType tests the real-world scenario where citext extension
// is needed before creating tables with citext columns (bb rollout multi-file SDL)
func TestExtensionWithCitextType(t *testing.T) {
	// Simulates multi-file SDL scenario where:
	// - extensions.sql contains: CREATE EXTENSION "citext"
	// - tables.sql contains: CREATE TABLE with citext column
	// Previous SDL (empty database)
	previousSDL := ``

	// Current SDL (multi-file combined: extension + table using citext type)
	// This simulates the content after merging extensions.sql and tables.sql
	currentSDL := `-- Extension: citext
-- Provides case-insensitive text data type
CREATE EXTENSION IF NOT EXISTS "citext" WITH SCHEMA public;

COMMENT ON EXTENSION "citext" IS 'Case-insensitive character string type';

-- Script to create all dependency example objects in correct order
-- For testdb11 schema

-- ============================================================================
-- STEP 1: Create Base Tables (Layer 0)
-- ============================================================================

-- Layer 0 entities have no foreign key dependencies and can be created first

CREATE TABLE "public"."email_addresses" (
	"email_id" serial,
	"email" citext NOT NULL,
	"user_id" integer,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP
);`

	// Get SDL diff
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Verify we have both extension and table changes
	require.Equal(t, 1, len(diff.ExtensionChanges), "Should have 1 extension change")
	require.GreaterOrEqual(t, len(diff.TableChanges), 1, "Should have at least 1 table change")

	extDiff := diff.ExtensionChanges[0]
	require.Equal(t, schema.MetadataDiffActionCreate, extDiff.Action)
	require.Equal(t, "citext", extDiff.ExtensionName)

	// Generate migration
	migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	require.NoError(t, err)
	t.Logf("Generated migration:\n%s", migration)

	// CRITICAL: Verify correct ordering - extension MUST come before table
	extensionPos := strings.Index(migration, "CREATE EXTENSION")
	tablePos := strings.Index(migration, "CREATE TABLE")

	require.NotEqual(t, -1, extensionPos, "CREATE EXTENSION should be present")
	require.NotEqual(t, -1, tablePos, "CREATE TABLE should be present")
	require.Less(t, extensionPos, tablePos, "CREATE EXTENSION must come BEFORE CREATE TABLE (otherwise citext type won't exist)")

	// Verify the extension is citext
	require.Contains(t, migration, `"citext"`)

	// Verify the table uses citext type
	require.Contains(t, migration, "email_addresses")
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

// TestExtensionRoundtripNoDiff tests that dumping SDL from database doesn't create false diffs
// This reproduces the issue where:
// 1. User creates extension with format A (e.g., manual SQL)
// 2. Bytebase dumps SDL from database
// 3. No changes were made, but extension is detected as changed and DROP+CREATE is generated
func TestExtensionRoundtripNoDiff(t *testing.T) {
	t.Run("User creates extension then dump from database - should not generate diff", func(t *testing.T) {
		// STEP 1: User's initial SDL (manual creation)
		userSDL := `CREATE EXTENSION IF NOT EXISTS "citext" WITH SCHEMA public;`

		// Parse user SDL to get metadata (simulate database state)
		chunks, err := ChunkSDLText(userSDL)
		require.NoError(t, err)
		require.Equal(t, 1, len(chunks.Extensions))

		// STEP 2: Simulate database dump - create metadata and dump back to SDL
		// This simulates what happens when Bytebase dumps the current database state
		metadata := &storepb.DatabaseSchemaMetadata{
			Extensions: []*storepb.ExtensionMetadata{
				{
					Name:   "citext",
					Schema: "public",
					// Version might or might not be specified by user
					Version: "",
					// Description might be empty
					Description: "",
				},
			},
		}

		// Create current database schema from metadata (this is the actual database state)
		currentSchema := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)

		// Dump SDL from metadata (this is what getSDLFormat does)
		dumpedSDL, err := getSDLFormat(metadata)
		require.NoError(t, err)

		// STEP 3: Compare user SDL (previous) vs dumped SDL (current)
		// This is what happens in bb rollout when user doesn't change anything
		// Pass currentSchema so usability check can work
		diff, err := GetSDLDiff(dumpedSDL, userSDL, currentSchema, nil)
		require.NoError(t, err)

		// CRITICAL: There should be NO extension changes!
		// The extension is the same, just different text representation
		if len(diff.ExtensionChanges) > 0 {
			t.Error("FAILED: Extension changes detected when there should be none!")
			t.Error("This means Bytebase will try to DROP+CREATE the extension even though nothing changed")

			// Generate migration to see what would happen
			migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
			require.NoError(t, err)

			// Check if DROP is in the migration
			if strings.Contains(migration, "DROP EXTENSION") {
				t.Error("CRITICAL: Migration contains DROP EXTENSION - this will fail with '2BP01' error!")
			}
		}

		require.Equal(t, 0, len(diff.ExtensionChanges), "No changes should be detected for unchanged extension")
	})

	t.Run("Extension with version - roundtrip test", func(t *testing.T) {
		// User creates with explicit version
		userSDL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public VERSION '1.1';`

		// Database metadata after creation
		metadata := &storepb.DatabaseSchemaMetadata{
			Extensions: []*storepb.ExtensionMetadata{
				{
					Name:    "uuid-ossp",
					Schema:  "public",
					Version: "1.1",
				},
			},
		}

		currentSchema := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)

		dumpedSDL, err := getSDLFormat(metadata)
		require.NoError(t, err)

		diff, err := GetSDLDiff(dumpedSDL, userSDL, currentSchema, nil)
		require.NoError(t, err)

		require.Equal(t, 0, len(diff.ExtensionChanges), "No changes for extension with version")
	})

	t.Run("Extension with comment - roundtrip test", func(t *testing.T) {
		// User creates with comment
		userSDL := `CREATE EXTENSION IF NOT EXISTS "citext" WITH SCHEMA public;

COMMENT ON EXTENSION "citext" IS 'Case-insensitive text type';`

		metadata := &storepb.DatabaseSchemaMetadata{
			Extensions: []*storepb.ExtensionMetadata{
				{
					Name:        "citext",
					Schema:      "public",
					Description: "Case-insensitive text type",
				},
			},
		}

		currentSchema := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)

		dumpedSDL, err := getSDLFormat(metadata)
		require.NoError(t, err)

		diff, err := GetSDLDiff(dumpedSDL, userSDL, currentSchema, nil)
		require.NoError(t, err)

		require.Equal(t, 0, len(diff.ExtensionChanges), "No changes for extension with comment")
	})
}
