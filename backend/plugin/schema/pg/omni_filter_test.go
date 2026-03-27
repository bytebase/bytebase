package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestOmniFilter_BbdataarchiveSchemaFiltered verifies that objects in the
// bbdataarchive schema are excluded from migration output.
func TestOmniFilter_BbdataarchiveSchemaFiltered(t *testing.T) {
	// Source has tables in both public and bbdataarchive schemas.
	sourceMetadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*storepb.TableMetadata{
					{Name: "archived_users", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}
	// Target is empty — should only generate DROP for public, not bbdataarchive.
	targetSDL := ""

	sourceMeta := model.NewDatabaseMetadata(sourceMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, sourceMeta)
	require.NoError(t, err)

	sql, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, sourceSDL, targetSDL)
	require.NoError(t, err)

	// bbdataarchive objects must not appear in migration.
	require.NotContains(t, sql, "bbdataarchive")
	require.NotContains(t, sql, "archived_users")
	// public objects should be dropped.
	require.Contains(t, sql, "users")
}

// TestOmniFilter_SkipBackupSchemaInSDLGeneration verifies that MetadataToSDL
// excludes the backup schema entirely from generated SDL text.
func TestOmniFilter_SkipBackupSchemaInSDLGeneration(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*storepb.TableMetadata{
					{Name: "backup_t1", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}
	meta := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)

	sdl, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)

	require.NotContains(t, sdl, "bbdataarchive")
	require.NotContains(t, sdl, "backup_t1")
	require.Contains(t, sdl, "t1")
}

// TestOmniFilter_SkipDumpObjectsExcluded verifies that objects marked with
// SkipDump (e.g., extension-created objects) are excluded from SDL generation.
func TestOmniFilter_SkipDumpObjectsExcluded(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "user_table", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
					{Name: "ext_table", SkipDump: true, Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
				Functions: []*storepb.FunctionMetadata{
					{Name: "my_func", Definition: "CREATE FUNCTION my_func() RETURNS void LANGUAGE sql AS $$ SELECT 1 $$;"},
					{Name: "ext_func", SkipDump: true, Definition: "CREATE FUNCTION ext_func() RETURNS void LANGUAGE sql AS $$ SELECT 1 $$;"},
				},
			},
		},
	}
	meta := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)

	sdl, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)

	require.Contains(t, sdl, "user_table")
	require.Contains(t, sdl, "my_func")
	require.NotContains(t, sdl, "ext_table")
	require.NotContains(t, sdl, "ext_func")
}

// TestOmniFilter_ExtensionPreservedWhenArchiveFiltered verifies that extension
// operations are preserved even when bbdataarchive objects are filtered.
func TestOmniFilter_ExtensionPreservedWhenArchiveFiltered(t *testing.T) {
	sql := omniSDLMigration(t, "", `CREATE EXTENSION IF NOT EXISTS "citext";`)
	// Extension creation should not be filtered.
	require.Contains(t, sql, "citext")
}

// TestOmniFilter_NoChangesForIdenticalSchemas verifies that two identical
// schemas with backup objects produce no migration (backup objects are
// excluded from both sides consistently).
func TestOmniFilter_NoChangesForIdenticalSchemas(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*storepb.TableMetadata{
					{Name: "old_users", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}

	meta := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)
	// Use MetadataToSDL to get the exact SDL that would be generated, then diff against itself.
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)
	sql, err := schema.SDLMigration(storepb.Engine_POSTGRES, sourceSDL, meta)
	require.NoError(t, err)
	require.Empty(t, sql, "identical public schemas with backup objects should produce no migration")
}
