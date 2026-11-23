package pg

import (
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestDiffSDLMigration_RegistryAndErrorWrapping(t *testing.T) {
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		t.Run(engine.String(), func(t *testing.T) {
			sql, err := schema.DiffSDLMigration(engine,
				"CREATE TABLE public.users (id integer PRIMARY KEY);",
				"CREATE TABLE public.users (id integer PRIMARY KEY, email text);",
				"",
			)
			require.NoError(t, err)
			require.Contains(t, sql, "ALTER TABLE")
			require.Contains(t, sql, "email")
		})
	}

	t.Run("raw dump fallback", func(t *testing.T) {
		rawDump := "SET statement_timeout = 0;\nCREATE TABLE public.users (id integer PRIMARY KEY);"
		target := "CREATE TABLE public.users (id integer PRIMARY KEY, name text);"
		sql, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, rawDump, target, "")
		require.NoError(t, err)
		require.Contains(t, sql, "name")
	})

	t.Run("wraps load errors", func(t *testing.T) {
		_, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, "CREATE TABLE (", "", "")
		require.ErrorContains(t, err, "failed to load source schema")

		_, err = schema.DiffSDLMigration(storepb.Engine_POSTGRES, "", "CREATE TABLE (", "")
		require.ErrorContains(t, err, "failed to load target schema")
	})
}

func TestPGSDLDropAdvices(t *testing.T) {
	current := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{
				{
					Name: "users",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "integer"},
						{Name: "bio", Type: "text", Nullable: true},
					},
				},
				{
					Name:    "legacy",
					Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}},
				},
			},
			Functions: []*metadatapb.FunctionMetadata{
				{Name: "calc", Definition: "CREATE FUNCTION public.calc() RETURNS integer LANGUAGE sql AS $$ SELECT 1 $$;"},
			},
		}},
	}
	meta := model.NewDatabaseMetadata(current, nil, nil, storepb.Engine_POSTGRES, false)

	t.Run("no-op produces no advices", func(t *testing.T) {
		sdl, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
		require.NoError(t, err)
		advices, err := schema.SDLDropAdvices(storepb.Engine_POSTGRES, sdl, meta, "")
		require.NoError(t, err)
		require.Empty(t, advices)
	})

	t.Run("drop table, drop column, and replace function emit warnings", func(t *testing.T) {
		target := strings.TrimSpace(`
CREATE TABLE public.users (id integer NOT NULL);
CREATE FUNCTION public.calc() RETURNS integer LANGUAGE sql AS $$ SELECT 2 $$;
`)
		for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
			advices, err := schema.SDLDropAdvices(engine, target, meta, "")
			require.NoError(t, err)
			require.NotEmpty(t, advices)

			var dropContents, replaceContents []string
			for _, a := range advices {
				require.Equal(t, storepb.Advice_WARNING, a.Status)
				switch a.Code {
				case code.SDLDropOperation.Int32():
					dropContents = append(dropContents, a.Content)
				case code.SDLReplaceOperation.Int32():
					replaceContents = append(replaceContents, a.Content)
				default:
					t.Fatalf("unexpected advice code %d: %+v", a.Code, a)
				}
			}
			joinedDrops := strings.Join(dropContents, "\n")
			require.Contains(t, joinedDrops, "Dropping table 'public.legacy'")
			require.Contains(t, joinedDrops, "Dropping column from table")
			require.Contains(t, strings.Join(replaceContents, "\n"), "public.calc")
		}
	})
}

// TestFilter_BbdataarchiveSchemaFiltered verifies that objects in the
// bbdataarchive schema are excluded from migration output.
func TestFilter_BbdataarchiveSchemaFiltered(t *testing.T) {
	sourceMetadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*metadatapb.TableMetadata{
					{Name: "archived_users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}

	sourceMeta := model.NewDatabaseMetadata(sourceMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, sourceMeta)
	require.NoError(t, err)

	sql, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, sourceSDL, "", "")
	require.NoError(t, err)

	require.NotContains(t, sql, "bbdataarchive")
	require.NotContains(t, sql, "archived_users")
	require.Contains(t, sql, "users")
}

// TestFilter_SkipBackupSchemaInSDLGeneration verifies that MetadataToSDL
// excludes the backup schema entirely from generated SDL text.
func TestFilter_SkipBackupSchemaInSDLGeneration(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "t1", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*metadatapb.TableMetadata{
					{Name: "backup_t1", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
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

// TestFilter_SkipDumpObjectsExcluded verifies that objects marked with
// SkipDump (e.g., extension-created objects) are excluded from SDL generation.
func TestFilter_SkipDumpObjectsExcluded(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "user_table", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
					{Name: "ext_table", SkipDump: true, Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
				Functions: []*metadatapb.FunctionMetadata{
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

// TestFilter_ExtensionPreservedWhenArchiveFiltered verifies that extension
// operations are preserved when bbdataarchive objects are filtered.
func TestFilter_ExtensionPreservedWhenArchiveFiltered(t *testing.T) {
	sql, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, "", `CREATE EXTENSION IF NOT EXISTS "citext";`, "")
	require.NoError(t, err)
	require.Contains(t, sql, "citext")
}

// TestFilter_NoChangesForIdenticalSchemas verifies that two identical
// schemas with backup objects produce no migration.
func TestFilter_NoChangesForIdenticalSchemas(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*metadatapb.TableMetadata{
					{Name: "old_users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}

	meta := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)
	sql, err := schema.SDLMigration(storepb.Engine_POSTGRES, sourceSDL, meta, "")
	require.NoError(t, err)
	require.Empty(t, sql, "identical public schemas with backup objects should produce no migration")
}
