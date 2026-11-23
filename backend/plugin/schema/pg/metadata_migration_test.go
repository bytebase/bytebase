package pg

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func pgTable(columns ...*metadatapb.ColumnMetadata) *metadatapb.TableMetadata {
	return &metadatapb.TableMetadata{Name: "t", Columns: columns}
}

func newPGDatabaseMetadata(tables []*metadatapb.TableMetadata, views []*metadatapb.ViewMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:   "public",
				Tables: tables,
				Views:  views,
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
}

// TestDiffMigrationRouting proves both engines' metadata diffs reach omni's
// catalog. CockroachDB shares the PostgreSQL catalog; its legacy generator is
// gone.
func TestDiffMigrationRouting(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "integer"}
	note := &metadatapb.ColumnMetadata{Name: "note", Type: "text", Nullable: true}
	source := newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable(id)}, nil)
	target := newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable(id, note)}, nil)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE public.t ADD COLUMN note text", sql)

	sql, err = schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE public.t ADD COLUMN note text", sql)
}

// TestDiffMigrationSeamOptions verifies Bytebase-side adapter options:
// backup schema exclusion (SkipSchemas), search_path forwarding
// (SourceSearchPath/TargetSearchPath), error propagation, and nil inputs.
func TestDiffMigrationSeamOptions(t *testing.T) {
	backupSchema := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "integer"}

	t.Run("excludes backup schema from migration and preflight", func(t *testing.T) {
		oldMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Schemas: []*metadatapb.SchemaMetadata{
				{Name: "public", Tables: []*metadatapb.TableMetadata{pgTable(id)}},
				{
					Name:  backupSchema,
					Views: []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT @@@old@@@"}},
				},
			},
		}, nil, nil, storepb.Engine_POSTGRES, true)
		newMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Schemas: []*metadatapb.SchemaMetadata{
				{Name: "public", Tables: []*metadatapb.TableMetadata{pgTable(id, &metadatapb.ColumnMetadata{Name: "note", Type: "text", Nullable: true})}},
				{
					Name:  backupSchema,
					Views: []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT @@@new@@@"}},
				},
			},
		}, nil, nil, storepb.Engine_POSTGRES, true)

		sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, oldMeta, newMeta)
		require.NoError(t, err)
		require.Equal(t, "ALTER TABLE public.t ADD COLUMN note text", sql)
		require.NotContains(t, sql, backupSchema)
	})

	t.Run("forwards configured search_path including pg_catalog and $user", func(t *testing.T) {
		mk := func(searchPath string, withExtra bool) *model.DatabaseMetadata {
			cols := []*metadatapb.ColumnMetadata{{Name: "v", Type: "custom"}}
			if withExtra {
				cols = append(cols, &metadatapb.ColumnMetadata{Name: "extra", Type: "integer", Nullable: true})
			}
			return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
				SearchPath: searchPath,
				Schemas: []*metadatapb.SchemaMetadata{
					{Name: "s1", EnumTypes: []*metadatapb.EnumTypeMetadata{{Name: "custom", Values: []string{"a"}}}},
					{Name: "s2", EnumTypes: []*metadatapb.EnumTypeMetadata{{Name: "custom", Values: []string{"b"}}}},
					{Name: "public", Tables: []*metadatapb.TableMetadata{pgTable(cols...)}},
				},
			}, nil, nil, storepb.Engine_POSTGRES, true)
		}

		// With "$user", s1, s2, the unqualified type "custom" is ambiguous
		// because $user is unresolved in offline metadata diffing.
		_, err := schema.DiffMigration(storepb.Engine_POSTGRES, mk(`"$user", s1, s2`, false), mk(`"$user", s1, s2`, true))
		require.Error(t, err)

		// Resolving search_path to s1, s2 unambiguously picks s1.custom.
		sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, mk(`s1, s2`, false), mk(`s1, s2`, true))
		require.NoError(t, err)
		require.Equal(t, "ALTER TABLE public.t ADD COLUMN extra integer", sql)
	})

	t.Run("handles nil metadata wrappers", func(t *testing.T) {
		target := newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable(id)}, nil)
		sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, nil, target)
		require.NoError(t, err)
		require.Contains(t, sql, `CREATE TABLE "public"."t"`)
	})

	t.Run("propagates catalog migration errors", func(t *testing.T) {
		_, err := schema.DiffMigration(
			storepb.Engine_POSTGRES,
			newPGDatabaseMetadata(nil, []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT @@@old@@@"}}),
			newPGDatabaseMetadata(nil, []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT @@@new@@@"}}),
		)
		require.ErrorContains(t, err, "rel:public.v")
	})
}

type postgresEntryPoint struct {
	metadata         *metadatapb.DatabaseSchemaMetadata
	schema           *metadatapb.SchemaMetadata
	table            *metadatapb.TableMetadata
	view             *metadatapb.ViewMetadata
	materializedView *metadatapb.MaterializedViewMetadata
	function         *metadatapb.FunctionMetadata
	sequence         *metadatapb.SequenceMetadata
}

func newPostgresEntryPoint() postgresEntryPoint {
	table := &metadatapb.TableMetadata{
		Name:    "t",
		Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer", Nullable: false}},
		Indexes: []*metadatapb.IndexMetadata{{
			Name: "t_pkey", Expressions: []string{"id"}, Primary: true, Unique: true, IsConstraint: true,
		}},
	}
	view := &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT id FROM public.t"}
	materializedView := &metadatapb.MaterializedViewMetadata{Name: "mv", Definition: "SELECT id FROM public.t"}
	function := &metadatapb.FunctionMetadata{Name: "f", Signature: "f()", Definition: "CREATE FUNCTION public.f() RETURNS integer LANGUAGE sql AS $$ SELECT 1 $$"}
	sequence := &metadatapb.SequenceMetadata{Name: "s", DataType: "bigint", Start: "1", Increment: "1"}
	enum := &metadatapb.EnumTypeMetadata{Name: "e", Values: []string{"a", "b"}}
	schemaMeta := &metadatapb.SchemaMetadata{
		Name:              "public",
		Comment:           "app schema",
		Tables:            []*metadatapb.TableMetadata{table},
		Views:             []*metadatapb.ViewMetadata{view},
		MaterializedViews: []*metadatapb.MaterializedViewMetadata{materializedView},
		Functions:         []*metadatapb.FunctionMetadata{function},
		Sequences:         []*metadatapb.SequenceMetadata{sequence},
		EnumTypes:         []*metadatapb.EnumTypeMetadata{enum},
	}
	return postgresEntryPoint{
		metadata: &metadatapb.DatabaseSchemaMetadata{
			Extensions: []*metadatapb.ExtensionMetadata{{Name: "pg_trgm", Schema: "public", Version: "1.6"}},
			Schemas:    []*metadatapb.SchemaMetadata{schemaMeta},
		},
		schema:           schemaMeta,
		table:            table,
		view:             view,
		materializedView: materializedView,
		function:         function,
		sequence:         sequence,
	}
}

func TestPostgresDefinitionEntryPoints(t *testing.T) {
	entry := newPostgresEntryPoint()
	ctx := schema.GetDefinitionContext{PrintHeader: true}

	definition, err := schema.GetDatabaseDefinition(storepb.Engine_POSTGRES, ctx, entry.metadata)
	require.NoError(t, err)
	require.Contains(t, definition, `"public"."t"`)
	require.Contains(t, definition, `CREATE EXTENSION IF NOT EXISTS "pg_trgm"`)

	schemaDefinition, err := schema.GetSchemaDefinition(storepb.Engine_POSTGRES, entry.schema)
	require.NoError(t, err)
	require.Contains(t, schemaDefinition, `COMMENT ON SCHEMA`)
	tableDefinition, err := schema.GetTableDefinition(storepb.Engine_POSTGRES, "public", entry.table, []*metadatapb.SequenceMetadata{entry.sequence})
	require.NoError(t, err)
	require.Contains(t, tableDefinition, `CREATE TABLE`)
	viewDefinition, err := schema.GetViewDefinition(storepb.Engine_POSTGRES, "public", entry.view)
	require.NoError(t, err)
	require.Contains(t, viewDefinition, `CREATE VIEW`)
	materializedViewDefinition, err := schema.GetMaterializedViewDefinition(storepb.Engine_POSTGRES, "public", entry.materializedView)
	require.NoError(t, err)
	require.Contains(t, materializedViewDefinition, `CREATE MATERIALIZED VIEW`)
	functionDefinition, err := schema.GetFunctionDefinition(storepb.Engine_POSTGRES, "public", entry.function)
	require.NoError(t, err)
	require.Contains(t, functionDefinition, `CREATE FUNCTION`)
	sequenceDefinition, err := schema.GetSequenceDefinition(storepb.Engine_POSTGRES, "public", entry.sequence)
	require.NoError(t, err)
	require.Contains(t, sequenceDefinition, `CREATE SEQUENCE`)

	multiFile, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_POSTGRES, ctx, entry.metadata)
	require.NoError(t, err)
	require.NotEmpty(t, multiFile.Files)
}
