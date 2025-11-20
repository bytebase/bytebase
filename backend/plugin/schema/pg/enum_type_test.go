package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestEnumTypeInSDLOutput tests that enum types are included in SDL output
func TestEnumTypeInSDLOutput(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{
						Name:    "status_type",
						Values:  []string{"active", "inactive", "pending"},
						Comment: "User status enum",
					},
					{
						Name:   "priority_type",
						Values: []string{"low", "medium", "high", "urgent"},
					},
				},
			},
		},
	}

	// Test single-file SDL format
	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err)

	// Verify enum types are present
	require.Contains(t, sdl, "CREATE TYPE")
	require.Contains(t, sdl, "status_type")
	require.Contains(t, sdl, "priority_type")
	require.Contains(t, sdl, "'active'")
	require.Contains(t, sdl, "'inactive'")
	require.Contains(t, sdl, "'pending'")
	require.Contains(t, sdl, "'low'")
	require.Contains(t, sdl, "'medium'")
	require.Contains(t, sdl, "'high'")
	require.Contains(t, sdl, "'urgent'")

	// Verify comment
	require.Contains(t, sdl, "COMMENT ON TYPE")
	require.Contains(t, sdl, "User status enum")
}

// TestEnumTypeInMultiFileSDL tests that enum types are included in multi-file SDL
func TestEnumTypeInMultiFileSDL(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{
						Name:    "color_type",
						Values:  []string{"red", "green", "blue"},
						Comment: "RGB color enum",
					},
				},
			},
		},
	}

	// Test multi-file SDL format
	result, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the enum type file
	var enumFile *schema.File
	for i := range result.Files {
		if result.Files[i].Name == "schemas/public/types.sql" {
			enumFile = &result.Files[i]
			break
		}
	}
	require.NotNil(t, enumFile, "Enum type file not found")

	// Verify content includes enum type
	require.Contains(t, enumFile.Content, "CREATE TYPE")
	require.Contains(t, enumFile.Content, "color_type")
	require.Contains(t, enumFile.Content, "'red'")
	require.Contains(t, enumFile.Content, "'green'")
	require.Contains(t, enumFile.Content, "'blue'")
	require.Contains(t, enumFile.Content, "COMMENT ON TYPE")
	require.Contains(t, enumFile.Content, "RGB color enum")
}

// TestEnumTypeDependencyOrdering tests that enum types have correct dependency ordering in migrations
func TestEnumTypeDependencyOrdering(t *testing.T) {
	t.Run("CREATE: Enum before Table", func(t *testing.T) {
		// Test that when creating an enum type used by a table,
		// CREATE TYPE comes before CREATE TABLE

		oldMetadata := &storepb.DatabaseSchemaMetadata{
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
				},
			},
		}

		newMetadata := &storepb.DatabaseSchemaMetadata{
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
									Type: "integer",
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

		// Find positions of CREATE TYPE and CREATE TABLE
		typePos := -1
		tablePos := -1
		for i := 0; i < len(migration); i++ {
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

		require.NotEqual(t, -1, typePos, "CREATE TYPE not found in migration")
		require.NotEqual(t, -1, tablePos, "CREATE TABLE not found in migration")
		require.Less(t, typePos, tablePos, "CREATE TYPE should come before CREATE TABLE")
	})

	t.Run("DROP: Table before Enum", func(t *testing.T) {
		// Test that when dropping an enum type used by a table,
		// DROP TABLE comes before DROP TYPE

		oldMetadata := &storepb.DatabaseSchemaMetadata{
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
									Type: "integer",
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

		// Find positions of DROP TABLE and DROP TYPE
		tablePos := -1
		typePos := -1
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
		}

		require.NotEqual(t, -1, tablePos, "DROP TABLE not found in migration")
		require.NotEqual(t, -1, typePos, "DROP TYPE not found in migration")
		require.Less(t, tablePos, typePos, "DROP TABLE should come before DROP TYPE")
	})

	t.Run("ADD Enum via SDL chunks", func(t *testing.T) {
		// This tests adding an enum type via SDL diff

		// Previous SDL (no enum)
		previousSDL := `CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);`

		// Current SDL (with new enum type)
		currentSDL := `CREATE TYPE "public"."status_type" AS ENUM (
	'active',
	'inactive',
	'pending'
);

CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);`

		// Get SDL diff (simulates bb rollout behavior)
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Debug: log all diffs
		t.Logf("EnumTypeChanges: %d", len(diff.EnumTypeChanges))
		for i, enumDiff := range diff.EnumTypeChanges {
			t.Logf("  Enum[%d]: Action=%v, Schema=%s, Name=%s, HasNewAST=%v",
				i, enumDiff.Action, enumDiff.SchemaName, enumDiff.EnumTypeName,
				enumDiff.NewASTNode != nil)
		}

		// Verify we have an EnumTypeDiff with CREATE action
		require.Equal(t, 1, len(diff.EnumTypeChanges), "Should have 1 enum type change")
		enumDiff := diff.EnumTypeChanges[0]
		require.Equal(t, schema.MetadataDiffActionCreate, enumDiff.Action)
		require.Equal(t, "public", enumDiff.SchemaName)
		require.Equal(t, "status_type", enumDiff.EnumTypeName)
		require.NotNil(t, enumDiff.NewASTNode, "NewASTNode should be set for SDL mode")

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify migration creates the enum type
		require.Contains(t, migration, "CREATE TYPE")
		require.Contains(t, migration, "status_type")
		require.Contains(t, migration, "'active'")
		require.Contains(t, migration, "'inactive'")
		require.Contains(t, migration, "'pending'")
	})

	t.Run("REMOVE Enum via SDL chunks", func(t *testing.T) {
		// This tests removing an enum type via SDL diff

		// Previous SDL (with enum)
		previousSDL := `CREATE TYPE "public"."status_type" AS ENUM (
	'active',
	'inactive'
);

CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);`

		// Current SDL (enum removed)
		currentSDL := `CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);`

		// Get SDL diff
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Verify we have an EnumTypeDiff with DROP action
		require.Equal(t, 1, len(diff.EnumTypeChanges), "Should have 1 enum type change")
		enumDiff := diff.EnumTypeChanges[0]
		require.Equal(t, schema.MetadataDiffActionDrop, enumDiff.Action)
		require.Equal(t, "public", enumDiff.SchemaName)
		require.Equal(t, "status_type", enumDiff.EnumTypeName)

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify migration drops the enum type
		require.Contains(t, migration, "DROP TYPE")
		require.Contains(t, migration, "status_type")
	})

	t.Run("MODIFY Enum via SDL chunks (DROP + CREATE)", func(t *testing.T) {
		// This tests modifying an enum type via SDL diff
		// PostgreSQL doesn't support renaming enum values, so we use DROP + CREATE pattern

		// Previous SDL (with original values)
		previousSDL := `CREATE TYPE "public"."status_type" AS ENUM (
	'active',
	'inactive'
);`

		// Current SDL (with modified values)
		currentSDL := `CREATE TYPE "public"."status_type" AS ENUM (
	'active',
	'inactive',
	'pending'
);`

		// Get SDL diff
		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, diff)

		// Verify we have two EnumTypeDiff entries: DROP then CREATE
		require.Equal(t, 2, len(diff.EnumTypeChanges), "Should have 2 enum type changes (DROP + CREATE)")
		dropDiff := diff.EnumTypeChanges[0]
		createDiff := diff.EnumTypeChanges[1]

		require.Equal(t, schema.MetadataDiffActionDrop, dropDiff.Action)
		require.Equal(t, "public", dropDiff.SchemaName)
		require.Equal(t, "status_type", dropDiff.EnumTypeName)

		require.Equal(t, schema.MetadataDiffActionCreate, createDiff.Action)
		require.Equal(t, "public", createDiff.SchemaName)
		require.Equal(t, "status_type", createDiff.EnumTypeName)
		require.NotNil(t, createDiff.NewASTNode, "NewASTNode should be set for SDL mode")

		// Generate migration
		migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
		require.NoError(t, err)
		t.Logf("Generated migration:\n%s", migration)

		// Verify migration has both DROP and CREATE
		require.Contains(t, migration, "DROP TYPE")
		require.Contains(t, migration, "CREATE TYPE")
		require.Contains(t, migration, "'pending'")
	})
}

// TestEnumTypeCommentChanges tests that enum type comment changes are properly detected and generated
func TestEnumTypeCommentChanges(t *testing.T) {
	// Previous SDL (enum without comment)
	previousSDL := `CREATE TYPE "public"."status_type" AS ENUM (
	'active',
	'inactive'
);`

	// Current SDL (enum with comment)
	currentSDL := `CREATE TYPE "public"."status_type" AS ENUM (
	'active',
	'inactive'
);

COMMENT ON TYPE "public"."status_type" IS 'User status enum';`

	// Get SDL diff
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Verify we have a comment change
	hasCommentDiff := false
	for _, commentDiff := range diff.CommentChanges {
		if commentDiff.ObjectType == schema.CommentObjectTypeType &&
			commentDiff.ObjectName == "status_type" {
			hasCommentDiff = true
			break
		}
	}
	require.True(t, hasCommentDiff, "Should have comment diff for enum type")

	// Generate migration
	migration, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	require.NoError(t, err)
	t.Logf("Generated migration:\n%s", migration)

	// Verify migration includes comment
	require.Contains(t, migration, "COMMENT ON TYPE")
	require.Contains(t, migration, "status_type")
	require.Contains(t, migration, "User status enum")
}
