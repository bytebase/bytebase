package pg

import (
	"os"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func pgTable(name string, rows int64, columns ...*metadatapb.ColumnMetadata) *metadatapb.TableMetadata {
	return &metadatapb.TableMetadata{Name: name, Columns: columns, RowCount: rows}
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

func newCockroachDatabaseMetadata(tables []*metadatapb.TableMetadata, views []*metadatapb.ViewMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:   "public",
				Tables: tables,
				Views:  views,
			},
		},
	}, nil, nil, storepb.Engine_COCKROACHDB, true)
}

func newCockroachDatabaseMetadataWithFunctions(functions []*metadatapb.FunctionMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:      "public",
				Functions: functions,
			},
		},
	}, nil, nil, storepb.Engine_COCKROACHDB, true)
}

// TestDiffMigrationRouting proves both engines' metadata diffs reach omni's
// catalog. CockroachDB shares the PostgreSQL catalog; its legacy generator is
// gone.
func TestDiffMigrationRouting(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "integer"}
	note := &metadatapb.ColumnMetadata{Name: "note", Type: "text", Nullable: true}
	source := newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable("t", 0, id)}, nil)
	target := newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable("t", 0, id, note)}, nil)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE public.t ADD COLUMN note text", sql)

	sql, err = schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE public.t ADD COLUMN note text", sql)
}

// TestPostgresSchemaEntryPoints pins the public schema surface the API uses to
// omni's pg catalog. This cutover changes only CockroachDB's registration; every
// Postgres entry point below keeps the implementation it had.
func TestPostgresSchemaEntryPoints(t *testing.T) {
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Extensions: []*metadatapb.ExtensionMetadata{{Name: "pg_trgm", Schema: "public", Version: "1.6"}},
		Schemas:    []*metadatapb.SchemaMetadata{schemaMeta},
	}
	ctx := schema.GetDefinitionContext{PrintHeader: true}

	definition, err := schema.GetDatabaseDefinition(storepb.Engine_POSTGRES, ctx, metadata)
	require.NoError(t, err)
	require.Contains(t, definition, `CREATE TABLE`)
	require.Contains(t, definition, `"public"."t"`)
	require.Contains(t, definition, `CREATE VIEW`)
	require.Contains(t, definition, `"public"."v"`)
	require.Contains(t, definition, `CREATE MATERIALIZED VIEW`)
	require.Contains(t, definition, `"public"."mv"`)
	require.Contains(t, definition, `CREATE FUNCTION public.f()`)
	require.Contains(t, definition, `CREATE SEQUENCE`)
	require.Contains(t, definition, `"public"."s"`)
	require.Contains(t, definition, `CREATE TYPE`)
	require.Contains(t, definition, `"public"."e"`)
	require.Contains(t, definition, `CREATE EXTENSION IF NOT EXISTS "pg_trgm"`)

	schemaDefinition, err := schema.GetSchemaDefinition(storepb.Engine_POSTGRES, schemaMeta)
	require.NoError(t, err)
	require.Contains(t, schemaDefinition, `COMMENT ON SCHEMA`)
	require.Contains(t, schemaDefinition, `CREATE TABLE`)
	tableDefinition, err := schema.GetTableDefinition(storepb.Engine_POSTGRES, "public", table, []*metadatapb.SequenceMetadata{sequence})
	require.NoError(t, err)
	require.Contains(t, tableDefinition, `CREATE TABLE`)
	require.Contains(t, tableDefinition, `"public"."t"`)
	viewDefinition, err := schema.GetViewDefinition(storepb.Engine_POSTGRES, "public", view)
	require.NoError(t, err)
	require.Contains(t, viewDefinition, `CREATE VIEW`)
	require.Contains(t, viewDefinition, `"public"."v"`)
	materializedViewDefinition, err := schema.GetMaterializedViewDefinition(storepb.Engine_POSTGRES, "public", materializedView)
	require.NoError(t, err)
	require.Contains(t, materializedViewDefinition, `CREATE MATERIALIZED VIEW`)
	require.Contains(t, materializedViewDefinition, `"public"."mv"`)
	functionDefinition, err := schema.GetFunctionDefinition(storepb.Engine_POSTGRES, "public", function)
	require.NoError(t, err)
	require.Contains(t, functionDefinition, `CREATE FUNCTION`)
	require.Contains(t, functionDefinition, `public.f()`)
	sequenceDefinition, err := schema.GetSequenceDefinition(storepb.Engine_POSTGRES, "public", sequence)
	require.NoError(t, err)
	require.Contains(t, sequenceDefinition, `CREATE SEQUENCE`)
	require.Contains(t, sequenceDefinition, `"public"."s"`)

	multiFile, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_POSTGRES, ctx, metadata)
	require.NoError(t, err)
	require.NotEmpty(t, multiFile.Files)

	db := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	sdl, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, db)
	require.NoError(t, err)
	require.NotEmpty(t, sdl)
	sdlDiff, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, sdl, sdl, "")
	require.NoError(t, err)
	require.Empty(t, sdlDiff)
	migration, err := schema.DiffMigration(storepb.Engine_POSTGRES, db, db)
	require.NoError(t, err)
	require.Empty(t, migration)
	advices, err := schema.SDLDropAdvices(storepb.Engine_POSTGRES, "", db, "")
	require.NoError(t, err)
	require.NotEmpty(t, advices)
}

type cutoverParityCase struct {
	Description     string `yaml:"description"`
	Difference      string `yaml:"difference"`
	OldMetadata     string `yaml:"oldMetadata"`
	NewMetadata     string `yaml:"newMetadata"`
	LegacyExpected  string `yaml:"legacyExpected"`
	CatalogExpected string `yaml:"catalogExpected"`
	CatalogError    string `yaml:"catalogError,omitempty"`
}

// TestCockroachMigrationCutover replays every CockroachDB metadata-diff case the
// deleted Bytebase generator covered. The legacy text was captured before the
// generator was removed, so it is the differential rather than a live second
// path. Each case names the difference it records; a future catalog that makes a
// case match its legacy text still has that difference documented here.
func TestCockroachMigrationCutover(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/cutover_parity.yaml"
	)
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	var cases []cutoverParityCase
	require.NoError(t, yaml.Unmarshal(content, &cases))
	require.NotEmpty(t, cases)

	for i, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			require.NotEmpty(t, tc.Difference, "each case names the difference it records")
			oldDB := parseCockroachMetadata(t, tc.OldMetadata)
			newDB := parseCockroachMetadata(t, tc.NewMetadata)

			migration, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, oldDB, newDB)

			if record {
				cases[i].CatalogExpected = migration
				cases[i].CatalogError = ""
				if err != nil {
					cases[i].CatalogExpected = ""
					cases[i].CatalogError = err.Error()
				}
				return
			}
			if tc.CatalogError != "" {
				require.ErrorContains(t, err, tc.CatalogError)
				require.NotEmpty(t, tc.LegacyExpected, "the legacy generator produced SQL this case no longer does")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.CatalogExpected, migration)
			if tc.LegacyExpected != tc.CatalogExpected {
				require.NotEqual(t, tc.LegacyExpected, migration, tc.Difference)
			}
		})
	}

	if record {
		yamltest.Record(t, filepath, cases)
	}
}

func parseCockroachMetadata(t *testing.T, encoded string) *model.DatabaseMetadata {
	t.Helper()
	metadata := &metadatapb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(encoded), metadata))
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_COCKROACHDB, true)
}

// TestCockroachMetadataMigrationNoOps pins the cases the old generator covered
// that produce no migration at all. They have no legacy differential, so they
// stay out of the cutover fixture.
func TestCockroachMetadataMigrationNoOps(t *testing.T) {
	unparseableView := &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT @@@omni_unsupported@@@"}
	composites := []*metadatapb.CompositeTypeMetadata{{
		Name:    "addr",
		Comment: "c",
		Attributes: []*metadatapb.CompositeTypeAttribute{
			{Name: "street", Type: "text", Collation: `"C"`, Comment: "s"},
		},
	}}

	tests := []struct {
		name   string
		source *model.DatabaseMetadata
		target *model.DatabaseMetadata
	}{
		{
			name:   "identical raw view does not load",
			source: newCockroachDatabaseMetadata(nil, []*metadatapb.ViewMetadata{unparseableView}),
			target: newCockroachDatabaseMetadata(nil, []*metadatapb.ViewMetadata{unparseableView}),
		},
		{
			name:   "identical composite types",
			source: newCockroachDatabaseMetadataWithCompositeTypes(composites),
			target: newCockroachDatabaseMetadataWithCompositeTypes(composites),
		},
		{
			name: "skip-dump composite types",
			source: newCockroachDatabaseMetadataWithCompositeTypes([]*metadatapb.CompositeTypeMetadata{{
				Name:     "ext_owned",
				SkipDump: true,
				Attributes: []*metadatapb.CompositeTypeAttribute{
					{Name: "x", Type: "integer"},
				},
			}}),
			target: newCockroachDatabaseMetadataWithCompositeTypes(nil),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, tc.source, tc.target)
			require.NoError(t, err)
			require.Empty(t, sql)
		})
	}
}

func newCockroachDatabaseMetadataWithCompositeTypes(compositeTypes []*metadatapb.CompositeTypeMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:           "public",
				CompositeTypes: compositeTypes,
			},
		},
	}, nil, nil, storepb.Engine_COCKROACHDB, true)
}

// TestCockroachMetadataMigrationIgnoresLegacyFunctionComparer records that the
// schema.FunctionComparer registry no longer governs CockroachDB: the catalog
// compares the stored definitions itself, so a comparer that declares two
// different bodies equal cannot suppress the replace.
func TestCockroachMetadataMigrationIgnoresLegacyFunctionComparer(t *testing.T) {
	schema.RegisterFunctionComparer(storepb.Engine_COCKROACHDB, &alwaysEqualFunctionComparer{})
	t.Cleanup(func() {
		schema.RegisterFunctionComparer(storepb.Engine_COCKROACHDB, &schema.DefaultFunctionComparer{})
	})
	source := newCockroachDatabaseMetadataWithFunctions([]*metadatapb.FunctionMetadata{
		newPGFunctionMetadata("SELECT 1"),
	})
	target := newCockroachDatabaseMetadataWithFunctions([]*metadatapb.FunctionMetadata{
		newPGFunctionMetadata("SELECT 2"),
	})

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)

	require.NoError(t, err)
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION")
	require.Contains(t, sql, "SELECT 2")
}

func newPGFunctionMetadata(body string) *metadatapb.FunctionMetadata {
	return &metadatapb.FunctionMetadata{
		Name:      "calculate_area",
		Signature: "calculate_area()",
		Definition: "CREATE FUNCTION public.calculate_area() RETURNS integer LANGUAGE sql AS $$ " +
			body + " $$",
	}
}

type alwaysEqualFunctionComparer struct{}

func (*alwaysEqualFunctionComparer) Equal(*metadatapb.FunctionMetadata, *metadatapb.FunctionMetadata) bool {
	return true
}

func (*alwaysEqualFunctionComparer) CompareDetailed(*metadatapb.FunctionMetadata, *metadatapb.FunctionMetadata) (*schema.FunctionComparisonResult, error) {
	return nil, nil
}

func TestOmniDiffMetadataMigrationDiffsAroundAnUnchangedUnloadableView(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "integer"}
	note := &metadatapb.ColumnMetadata{Name: "note", Type: "text", Nullable: true}
	view := &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT @@@omni_unsupported@@@"}

	sql, err := diffMetadataMigration(
		newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable("t", 0, id)}, []*metadatapb.ViewMetadata{view}),
		newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable("t", 0, id, note)}, []*metadatapb.ViewMetadata{view}),
	)
	require.NoError(t, err)
	require.Contains(t, sql, `ADD COLUMN note text`)
	require.NotContains(t, sql, "VIEW")
}

func TestOmniDiffMetadataMigrationRefusesAChangedUnloadableObject(t *testing.T) {
	_, err := diffMetadataMigration(
		newPGDatabaseMetadata(nil, []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT @@@old_definition@@@"}}),
		newPGDatabaseMetadata(nil, []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT @@@new_definition@@@"}}),
	)
	require.ErrorContains(t, err, "rel:public.v")
}

func TestOmniDiffMetadataMigrationIgnoresStatisticsOfAnUnloadableTable(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "integer"}
	unparsed := func(rows int64) *metadatapb.TableMetadata {
		return pgTable("unparsed", rows, &metadatapb.ColumnMetadata{Name: "a", Type: "no_such_type(((", Nullable: true})
	}
	note := &metadatapb.ColumnMetadata{Name: "note", Type: "text", Nullable: true}

	sql, err := diffMetadataMigration(
		newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable("t", 0, id), unparsed(10)}, nil),
		newPGDatabaseMetadata([]*metadatapb.TableMetadata{pgTable("t", 0, id, note), unparsed(20)}, nil),
	)
	require.NoError(t, err)
	require.Contains(t, sql, `ADD COLUMN note text`)
}

// TestIdentityGenerationBecomesIdentityColumn covers CockroachDB sync, which
// records identityGeneration without isIdentity. The catalog only installs an
// identity column from the flag, so the shim sets it before loading.
func TestIdentityGenerationBecomesIdentityColumn(t *testing.T) {
	id := &metadatapb.ColumnMetadata{
		Name:               "id",
		Type:               "bigint",
		IdentityGeneration: metadatapb.ColumnMetadata_ALWAYS,
	}
	source := newCockroachDatabaseMetadata(nil, nil)
	target := newCockroachDatabaseMetadata([]*metadatapb.TableMetadata{pgTable("t", 0, id)}, nil)

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)
	require.NoError(t, err)
	require.Contains(t, sql, "GENERATED ALWAYS AS IDENTITY")
}

// TestRecreatedViewReappliesUnchangedComment covers a dependency-driven view
// recreation: the comment did not change, so the catalog's comment diff is
// empty, but the recreated view would otherwise lose it.
func TestRecreatedViewReappliesUnchangedComment(t *testing.T) {
	view := func(definition string) *metadatapb.ViewMetadata {
		return &metadatapb.ViewMetadata{
			Name:       "v",
			Definition: definition,
			Comment:    "ids",
			DependencyColumns: []*metadatapb.DependencyColumn{{
				Schema: "public", Table: "t", Column: "id",
			}},
		}
	}
	source := newCockroachDatabaseMetadata(
		[]*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
		[]*metadatapb.ViewMetadata{view("SELECT id FROM public.t")},
	)
	target := newCockroachDatabaseMetadata(
		[]*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "bigint"})},
		[]*metadatapb.ViewMetadata{view("SELECT id::bigint AS id FROM public.t")},
	)

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)
	require.NoError(t, err)
	require.Contains(t, sql, "DROP VIEW")
	require.Contains(t, sql, `COMMENT ON VIEW "public"."v" IS 'ids'`)
	require.Equal(t, 1, strings.Count(sql, "COMMENT ON VIEW"))
}

// TestUnloadableUnchangedViewRecreatedWhenDependencyChanges covers a view whose
// definition omni cannot parse. Both sides hold the same stand-in, so the
// catalog has no dependency edge; DependencyColumns must still force the drop
// and recreation around the column change.
func TestUnloadableUnchangedViewRecreatedWhenDependencyChanges(t *testing.T) {
	view := func() *metadatapb.ViewMetadata {
		return &metadatapb.ViewMetadata{
			Name:       "v",
			Definition: "SELECT FROM",
			DependencyColumns: []*metadatapb.DependencyColumn{{
				Schema: "public", Table: "t", Column: "id",
			}},
		}
	}
	source := newCockroachDatabaseMetadata(
		[]*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
		[]*metadatapb.ViewMetadata{view()},
	)
	target := newCockroachDatabaseMetadata(
		[]*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "bigint"})},
		[]*metadatapb.ViewMetadata{view()},
	)

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)
	require.NoError(t, err)
	drop := strings.Index(sql, `DROP VIEW IF EXISTS "public"."v"`)
	alter := strings.Index(sql, "ALTER TABLE public.t ALTER COLUMN id TYPE bigint")
	create := strings.Index(sql, `CREATE VIEW "public"."v" AS`)
	require.GreaterOrEqual(t, drop, 0, sql)
	require.GreaterOrEqual(t, alter, 0, sql)
	require.GreaterOrEqual(t, create, 0, sql)
	require.Less(t, drop, alter, "the view is dropped before the column change")
	require.Less(t, alter, create, "the view is recreated after the column change")
}

// TestUnloadableUnchangedMaterializedViewRecreatedWithIndexes covers the same
// fallback for a materialized view: the recreation keeps WITH NO DATA and its
// indexes, which a bare CREATE ... AS would drop.
func TestUnloadableUnchangedMaterializedViewRecreatedWithIndexes(t *testing.T) {
	view := func() *metadatapb.MaterializedViewMetadata {
		return &metadatapb.MaterializedViewMetadata{
			Name:       "mv",
			Definition: "SELECT FROM",
			Indexes: []*metadatapb.IndexMetadata{{
				Name: "idx_mv", Expressions: []string{"id"}, Type: "btree",
			}},
			DependencyColumns: []*metadatapb.DependencyColumn{{
				Schema: "public", Table: "t", Column: "id",
			}},
		}
	}
	source := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:              "public",
		Tables:            []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
		MaterializedViews: []*metadatapb.MaterializedViewMetadata{view()},
	}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	target := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:              "public",
		Tables:            []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "bigint"})},
		MaterializedViews: []*metadatapb.MaterializedViewMetadata{view()},
	}}}, nil, nil, storepb.Engine_COCKROACHDB, true)

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)
	require.NoError(t, err)
	require.Contains(t, sql, `DROP MATERIALIZED VIEW IF EXISTS "public"."mv"`)
	require.Contains(t, sql, `CREATE MATERIALIZED VIEW "public"."mv"`)
	require.Contains(t, sql, "WITH NO DATA")
	require.Contains(t, sql, `CREATE INDEX "idx_mv" ON "public"."mv"`)
}

// TestUnloadableDependentViewsRestoreInDependencyOrder pins the order among the
// restored views: drops run dependents first, creates dependencies first.
func TestUnloadableDependentViewsRestoreInDependencyOrder(t *testing.T) {
	view := func(name string, deps ...*metadatapb.DependencyColumn) *metadatapb.ViewMetadata {
		return &metadatapb.ViewMetadata{Name: name, Definition: "SELECT FROM", DependencyColumns: deps}
	}
	tableDep := func() *metadatapb.DependencyColumn {
		return &metadatapb.DependencyColumn{Schema: "public", Table: "t", Column: "id"}
	}
	snapshot := func(columnType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{
				view("a", tableDep(), &metadatapb.DependencyColumn{Schema: "public", Table: "b"}),
				view("b", tableDep()),
			},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer"), snapshot("bigint"))
	require.NoError(t, err)
	dropA := strings.Index(sql, `DROP VIEW IF EXISTS "public"."a"`)
	dropB := strings.Index(sql, `DROP VIEW IF EXISTS "public"."b"`)
	createA := strings.Index(sql, `CREATE VIEW "public"."a"`)
	createB := strings.Index(sql, `CREATE VIEW "public"."b"`)
	require.GreaterOrEqual(t, dropA, 0, sql)
	require.GreaterOrEqual(t, dropB, 0, sql)
	require.GreaterOrEqual(t, createA, 0, sql)
	require.GreaterOrEqual(t, createB, 0, sql)
	require.Less(t, dropA, dropB, "the dependent view is dropped first")
	require.Less(t, createB, createA, "the dependency is created first")
}

// TestBackupSchemaDependentViewIsRejected keeps the backup-schema contract: a
// bbdataarchive view that must be dropped for a restored object cannot be
// emitted, so the migration is refused rather than producing a lower drop that
// the remaining backup view blocks.
func TestBackupSchemaDependentViewIsRejected(t *testing.T) {
	snapshot := func(columnType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "bbdataarchive",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v",
				Definition: "SELECT FROM",
				DependencyColumns: []*metadatapb.DependencyColumn{{
					Schema: "bbdataarchive", Table: "t", Column: "id",
				}},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	_, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer"), snapshot("bigint"))
	require.ErrorContains(t, err, "backup-schema view")
}

// TestSplitCompositeRetypeAddsAttributeBeforeItsComment covers a split release
// whose attribute also gains a comment: the ADD must land after the new relation
// exists but before the COMMENT that targets the attribute.
func TestSplitCompositeRetypeAddsAttributeBeforeItsComment(t *testing.T) {
	snapshot := func(table, attributeType, comment string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable(table, 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name: "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{{
					Name: "r", Type: attributeType, Comment: comment,
				}},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB,
		snapshot("old_t", "public.old_t", ""), snapshot("new_t", "public.new_t", "row comment"))
	require.NoError(t, err)
	add := strings.Index(sql, `ADD ATTRIBUTE "r" public.new_t`)
	comment := strings.Index(sql, "COMMENT ON COLUMN")
	require.GreaterOrEqual(t, add, 0, sql)
	require.GreaterOrEqual(t, comment, 0, sql)
	require.Less(t, add, comment, sql)
}

// TestSplitCompositeRetypeReappliesUnchangedComment covers the split path when
// the attribute keeps its comment: DROP ATTRIBUTE deletes it, the catalog sees
// no comment change, so the add must be followed by the comment.
func TestSplitCompositeRetypeReappliesUnchangedComment(t *testing.T) {
	snapshot := func(table, attributeType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable(table, 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name: "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{{
					Name: "r", Type: attributeType, Comment: "row comment",
				}},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB,
		snapshot("old_t", "public.old_t"), snapshot("new_t", "public.new_t"))
	require.NoError(t, err)
	add := strings.Index(sql, `ADD ATTRIBUTE "r" public.new_t`)
	comment := strings.Index(sql, `COMMENT ON COLUMN "public"."c"."r" IS 'row comment'`)
	require.GreaterOrEqual(t, add, 0, sql)
	require.GreaterOrEqual(t, comment, 0, sql)
	require.Less(t, add, comment, sql)
	require.Equal(t, 1, strings.Count(sql, "COMMENT ON COLUMN"), sql)
}

// TestSplitCompositeRetypeKeepsTargetAttributeOrder pins the order of attributes
// split from one dropped row type onto the same created type: composite fields are
// positional, so both ADDs must follow the target's attribute order.
func TestSplitCompositeRetypeKeepsTargetAttributeOrder(t *testing.T) {
	snapshot := func(table string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable(table, 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name: "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{
					{Name: "a", Type: "public." + table},
					{Name: "b", Type: "public." + table},
				},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("old_t"), snapshot("new_t"))
	require.NoError(t, err)
	dropTable := strings.Index(sql, `DROP TABLE "public"."old_t"`)
	dropA := strings.Index(sql, `DROP ATTRIBUTE "a"`)
	dropB := strings.Index(sql, `DROP ATTRIBUTE "b"`)
	addA := strings.Index(sql, `ADD ATTRIBUTE "a" public.new_t`)
	addB := strings.Index(sql, `ADD ATTRIBUTE "b" public.new_t`)
	for name, index := range map[string]int{"drop table": dropTable, "drop a": dropA, "drop b": dropB, "add a": addA, "add b": addB} {
		require.GreaterOrEqual(t, index, 0, "%s missing from:\n%s", name, sql)
	}
	require.Less(t, dropA, dropTable, "attribute a is released before the old row type is dropped")
	require.Less(t, dropB, dropTable, "attribute b is released before the old row type is dropped")
	require.Less(t, addA, addB, "split attributes keep the target's attribute order")
}

// TestSplitCompositeRetypeReaddsUnchangedSuffix covers a split attribute that
// precedes an unchanged one. ADD ATTRIBUTE appends, so the unchanged attribute
// must be released and re-added too, or the composite ends up in the wrong order.
func TestSplitCompositeRetypeReaddsUnchangedSuffix(t *testing.T) {
	snapshot := func(table string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable(table, 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name: "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{
					{Name: "a", Type: "public." + table},
					{Name: "z", Type: "text", Comment: "keep"},
				},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("old_t"), snapshot("new_t"))
	require.NoError(t, err)
	dropTable := strings.Index(sql, `DROP TABLE "public"."old_t"`)
	dropA := strings.Index(sql, `DROP ATTRIBUTE "a"`)
	dropZ := strings.Index(sql, `DROP ATTRIBUTE "z"`)
	addA := strings.Index(sql, `ADD ATTRIBUTE "a" public.new_t`)
	addZ := strings.Index(sql, `ADD ATTRIBUTE "z" text`)
	for name, index := range map[string]int{"drop table": dropTable, "drop a": dropA, "drop z": dropZ, "add a": addA, "add z": addZ} {
		require.GreaterOrEqual(t, index, 0, "%s missing from:\n%s", name, sql)
	}
	require.Less(t, dropA, dropTable, sql)
	require.Less(t, dropZ, dropTable, "the unchanged attribute is released before the row type drop")
	require.Less(t, addA, addZ, "the re-added suffix keeps the target's attribute order")
	require.Contains(t, sql, `COMMENT ON COLUMN "public"."c"."z" IS 'keep'`, "the re-added attribute keeps its comment")
}

// TestRestoredUnloadableViewReleasesRowTypeDependents covers an unchanged
// unloadable view that must be recreated for a column change while an unchanged
// composite attribute holds its row type: the attribute is dropped before the
// view and re-added after it, or the view drop finds a dependent object.
func TestRestoredUnloadableViewReleasesRowTypeDependents(t *testing.T) {
	snapshot := func(columnType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v",
				Definition: "SELECT FROM",
				DependencyColumns: []*metadatapb.DependencyColumn{{
					Schema: "public", Table: "t", Column: "id",
				}},
			}},
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name:       "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "r", Type: "public.v"}},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer"), snapshot("bigint"))
	require.NoError(t, err)
	dropAttribute := strings.Index(sql, `ALTER TYPE "public"."c" DROP ATTRIBUTE "r"`)
	dropView := strings.Index(sql, `DROP VIEW IF EXISTS "public"."v"`)
	createView := strings.Index(sql, `CREATE VIEW "public"."v"`)
	addAttribute := strings.Index(sql, `ALTER TYPE "public"."c" ADD ATTRIBUTE "r" public.v`)
	for name, index := range map[string]int{"drop attribute": dropAttribute, "drop view": dropView, "create view": createView, "add attribute": addAttribute} {
		require.GreaterOrEqual(t, index, 0, "%s missing from:\n%s", name, sql)
	}
	require.Less(t, dropAttribute, dropView, "the row-type dependent is released before the view is dropped")
	require.Less(t, createView, addAttribute, "the attribute is re-added after the view is recreated")
}

// TestRestoredUnloadableViewReleasesRetypedRowTypeAttribute covers an existing
// attribute retyped to a restored view's row type: the catalog cannot order its
// ALTER TYPE against the drop, so it is released and re-added like a split.
func TestRestoredUnloadableViewReleasesRetypedRowTypeAttribute(t *testing.T) {
	snapshot := func(columnType, attributeType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v",
				Definition: "SELECT FROM",
				DependencyColumns: []*metadatapb.DependencyColumn{{
					Schema: "public", Table: "t", Column: "id",
				}},
			}},
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name: "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{
					{Name: "r", Type: attributeType},
					{Name: "z", Type: "text", Comment: "keep"},
				},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer", "text"), snapshot("bigint", "public.v"))
	require.NoError(t, err)
	dropAttribute := strings.Index(sql, `ALTER TYPE "public"."c" DROP ATTRIBUTE "r"`)
	dropView := strings.Index(sql, `DROP VIEW IF EXISTS "public"."v"`)
	createView := strings.Index(sql, `CREATE VIEW "public"."v"`)
	addAttribute := strings.Index(sql, `ALTER TYPE "public"."c" ADD ATTRIBUTE "r" public.v`)
	require.GreaterOrEqual(t, dropAttribute, 0, sql)
	require.GreaterOrEqual(t, dropView, 0, sql)
	require.GreaterOrEqual(t, createView, 0, sql)
	require.GreaterOrEqual(t, addAttribute, 0, sql)
	require.Less(t, dropAttribute, dropView, "the retyped attribute is released before the view is dropped")
	require.Less(t, createView, addAttribute, "the attribute is re-added after the view is recreated")
	require.NotContains(t, sql, `ADD ATTRIBUTE "r" text`, "the catalog's own alter is replaced")
	require.Contains(t, sql, `COMMENT ON COLUMN "public"."c"."z" IS 'keep'`, "the re-added unchanged attribute keeps its comment")
}

// TestTypeNameMatchingKeepsIdentifierBoundaries pins quoted identifier parsing:
// a dot inside quotes belongs to that identifier, so `"a.b"."c"` and `"a"."b.c"`
// name different relations.
func TestTypeNameMatchingKeepsIdentifierBoundaries(t *testing.T) {
	dottedSchema := objectKey("a.b", "c")
	dottedObject := objectKey("a", "b.c")
	require.True(t, referencesType(`"a.b"."c"`, dottedSchema))
	require.True(t, referencesType(`"a"."b.c"`, dottedObject))
	require.False(t, referencesType(`"a"."b.c"`, dottedSchema), `"a"."b.c" is not "a.b"."c"`)
	require.False(t, referencesType(`"a.b"."c"`, dottedObject))
	require.True(t, referencesType("c", dottedSchema), "an unqualified name matches the object part")
	require.False(t, referencesType("c", dottedObject))

	for _, tc := range []struct{ typeName, key string }{
		{`"a"."b.c"`, dottedSchema},
		{`"a.b"."c"`, dottedObject},
	} {
		_, ok := referencedRelation(tc.typeName, map[string]bool{tc.key: true})
		require.False(t, ok, "%s does not name %s", tc.typeName, tc.key)
	}
	for _, tc := range []struct{ typeName, key string }{
		{`"a.b"."c"`, dottedSchema},
		{`"a"."b.c"`, dottedObject},
		{`"a.b"."c"[]`, dottedSchema},
	} {
		key, ok := referencedRelation(tc.typeName, map[string]bool{tc.key: true})
		require.True(t, ok, "%s names %s", tc.typeName, tc.key)
		require.Equal(t, tc.key, key)
	}

	upper := objectKey("public", "Foo")
	lower := objectKey("public", "foo")
	require.True(t, referencesType(`"public"."Foo"`, upper))
	require.False(t, referencesType(`"public"."Foo"`, lower), "a quoted identifier matches its case exactly")
	require.True(t, referencesType("public.foo", lower), "an unquoted identifier folds case")
	require.True(t, referencesType("public.Foo", lower))
	_, ok := referencedRelation(`"public"."Foo"`, map[string]bool{lower: true})
	require.False(t, ok, `"public"."Foo" is not "public"."foo"`)
	key, ok := referencedRelation("public.Foo", map[string]bool{lower: true})
	require.True(t, ok)
	require.Equal(t, lower, key)
}

// TestReleaseMovesTheAttributeNamedLikeItsType covers an attribute whose name
// repeats its composite type's name: the release must find the operation for that
// attribute, not the first one that mentions the type.
func TestReleaseMovesTheAttributeNamedLikeItsType(t *testing.T) {
	snapshot := func(table string) *model.DatabaseMetadata {
		attributeType := "text"
		schemaMeta := &metadatapb.SchemaMetadata{
			Name: "public",
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name: "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{
					{Name: "x", Type: attributeType},
					{Name: "c", Type: attributeType},
				},
			}},
		}
		if table != "" {
			schemaMeta.Tables = []*metadatapb.TableMetadata{pgTable(table, 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})}
			schemaMeta.CompositeTypes[0].Attributes[0].Type = "public." + table
			schemaMeta.CompositeTypes[0].Attributes[1].Type = "public." + table
		}
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{schemaMeta}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("old_t"), snapshot(""))
	require.NoError(t, err)
	dropTable := strings.Index(sql, `DROP TABLE "public"."old_t"`)
	attributeC := strings.Index(sql, `ALTER ATTRIBUTE "c" TYPE text`)
	require.GreaterOrEqual(t, dropTable, 0, sql)
	require.GreaterOrEqual(t, attributeC, 0, sql)
	require.Less(t, attributeC, dropTable, "the attribute named like its type is released before the drop")
}

// TestOpTargetsSelectedKeepsDistinctCommentOwners keeps a comment on `"a"."b.c"`
// with its owner when `"a.b"."c"` is the selected view: their flattened
// descriptions are equal, but the quoted targets are not.
func TestOpTargetsSelectedKeepsDistinctCommentOwners(t *testing.T) {
	selected := map[string]bool{objectKey("a.b", "c"): true}
	other := catalog.MigrationOp{Type: catalog.OpComment, ObjectName: "a.b.c", SQL: `COMMENT ON VIEW "a"."b.c" IS 'new'`}
	require.False(t, opTargetsSelected(other, selected, nil), "the other view's comment is not the selected view's")
	own := catalog.MigrationOp{Type: catalog.OpComment, ObjectName: "a.b.c", SQL: `COMMENT ON VIEW "a.b"."c" IS 'new'`}
	require.True(t, opTargetsSelected(own, selected, nil))
	column := catalog.MigrationOp{Type: catalog.OpComment, ObjectName: "a.b.c", SQL: `COMMENT ON COLUMN "a.b"."c"."id" IS 'id'`}
	require.True(t, opTargetsSelected(column, selected, nil), "an accessory of the selected view belongs to it")
}

// TestRestoredUnloadableViewRefusesRowTypeDependents keeps a table column, or an
// attribute the target adds, out of a restored view's drop: releasing a column
// would discard its data, and a new attribute would need the view before its
// ALTER TYPE, so the migration is refused instead.
func TestRestoredUnloadableViewRefusesRowTypeDependents(t *testing.T) {
	type world struct {
		tables         []*metadatapb.TableMetadata
		compositeTypes []*metadatapb.CompositeTypeMetadata
	}
	snapshot := func(columnType string, w world) *model.DatabaseMetadata {
		tables := append([]*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})}, w.tables...)
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: tables,
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v",
				Definition: "SELECT FROM",
				DependencyColumns: []*metadatapb.DependencyColumn{{
					Schema: "public", Table: "t", Column: "id",
				}},
			}},
			CompositeTypes: w.compositeTypes,
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	rowTypeColumn := func() *metadatapb.TableMetadata {
		return pgTable("holds", 0, &metadatapb.ColumnMetadata{Name: "r", Type: "public.v"})
	}
	tests := []struct {
		name   string
		source world
		target world
	}{
		{
			name:   "unchanged column",
			source: world{tables: []*metadatapb.TableMetadata{rowTypeColumn()}},
			target: world{tables: []*metadatapb.TableMetadata{rowTypeColumn()}},
		},
		{
			name:   "removed column",
			source: world{tables: []*metadatapb.TableMetadata{rowTypeColumn()}},
			target: world{tables: []*metadatapb.TableMetadata{pgTable("holds", 0)}},
		},
		{
			name:   "retyped column",
			source: world{tables: []*metadatapb.TableMetadata{rowTypeColumn()}},
			target: world{tables: []*metadatapb.TableMetadata{pgTable("holds", 0, &metadatapb.ColumnMetadata{Name: "r", Type: "text"})}},
		},
		{
			name:   "added column",
			source: world{tables: []*metadatapb.TableMetadata{pgTable("holds", 0)}},
			target: world{tables: []*metadatapb.TableMetadata{rowTypeColumn()}},
		},
		{
			name:   "added attribute",
			source: world{},
			target: world{compositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name:       "c",
				Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "r", Type: "public.v"}},
			}}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer", tc.source), snapshot("bigint", tc.target))
			require.ErrorContains(t, err, "uses the row type of view")
		})
	}
}

// TestLoadableDependentViewFollowsUnloadableView covers a loadable unchanged
// view that depends on an unloadable one: dropping the lower view fails while
// the upper still references it, so the closure must restore both.
func TestLoadableDependentViewFollowsUnloadableView(t *testing.T) {
	snapshot := func(columnType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{
				{
					Name:       "upper",
					Definition: "SELECT id FROM public.lower",
					DependencyColumns: []*metadatapb.DependencyColumn{{
						Schema: "public", Table: "lower",
					}},
				},
				{
					Name:       "lower",
					Definition: "SELECT FROM",
					DependencyColumns: []*metadatapb.DependencyColumn{{
						Schema: "public", Table: "t", Column: "id",
					}},
				},
			},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer"), snapshot("bigint"))
	require.NoError(t, err)
	dropUpper := strings.Index(sql, `DROP VIEW IF EXISTS "public"."upper"`)
	dropLower := strings.Index(sql, `DROP VIEW IF EXISTS "public"."lower"`)
	createLower := strings.Index(sql, `CREATE VIEW "public"."lower"`)
	createUpper := strings.Index(sql, `CREATE VIEW "public"."upper"`)
	require.GreaterOrEqual(t, dropUpper, 0, sql)
	require.GreaterOrEqual(t, dropLower, 0, sql)
	require.GreaterOrEqual(t, createLower, 0, sql)
	require.GreaterOrEqual(t, createUpper, 0, sql)
	require.Less(t, dropUpper, dropLower, sql)
	require.Less(t, createLower, createUpper, sql)
}

// TestChangedDependentViewIsDroppedBeforeUnloadableView covers a changed upper
// view: it is not recreated by the fallback, but it must still be dropped before
// the lower view it depends on.
func TestChangedDependentViewIsDroppedBeforeUnloadableView(t *testing.T) {
	snapshot := func(columnType, upperDefinition string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{
				{
					Name:       "upper",
					Definition: upperDefinition,
					DependencyColumns: []*metadatapb.DependencyColumn{{
						Schema: "public", Table: "lower",
					}},
				},
				{
					Name:       "lower",
					Definition: "SELECT FROM",
					DependencyColumns: []*metadatapb.DependencyColumn{{
						Schema: "public", Table: "t", Column: "id",
					}},
				},
			},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB,
		snapshot("integer", "SELECT id FROM public.lower"),
		snapshot("bigint", "SELECT id, 1 AS x FROM public.lower"))
	require.NoError(t, err)
	dropUpper := strings.Index(sql, `DROP VIEW IF EXISTS "public"."upper"`)
	dropLower := strings.Index(sql, `DROP VIEW IF EXISTS "public"."lower"`)
	recreateUpper := strings.Index(sql, `CREATE VIEW "public"."upper"`)
	require.GreaterOrEqual(t, dropUpper, 0, sql)
	require.GreaterOrEqual(t, dropLower, 0, sql)
	require.GreaterOrEqual(t, recreateUpper, 0, sql)
	require.Less(t, dropUpper, dropLower, sql)
	require.Less(t, dropLower, recreateUpper, sql)
}

// TestChangedViewKeepsOldDependencyEdge covers a target definition that stops
// depending on the unloadable view: the currently installed view still does, so
// its drop must be ordered before the lower view's drop.
func TestChangedViewKeepsOldDependencyEdge(t *testing.T) {
	snapshot := func(columnType, upperDefinition string, upperDeps []*metadatapb.DependencyColumn) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{
				{Name: "upper", Definition: upperDefinition, DependencyColumns: upperDeps},
				{
					Name:       "lower",
					Definition: "SELECT FROM",
					DependencyColumns: []*metadatapb.DependencyColumn{{
						Schema: "public", Table: "t", Column: "id",
					}},
				},
			},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB,
		snapshot("integer", "SELECT id FROM public.lower", []*metadatapb.DependencyColumn{{Schema: "public", Table: "lower"}}),
		snapshot("bigint", "SELECT 1 AS one", nil))
	require.NoError(t, err)
	dropUpper := strings.Index(sql, `DROP VIEW IF EXISTS "public"."upper"`)
	dropLower := strings.Index(sql, `DROP VIEW IF EXISTS "public"."lower"`)
	require.GreaterOrEqual(t, dropUpper, 0, sql)
	require.GreaterOrEqual(t, dropLower, 0, sql)
	require.Less(t, dropUpper, dropLower, sql)
}

// TestCommentOnlyViewChangeIsRecreated covers a selected dependent whose only
// change is its comment: the catalog emits no replacement, so the fallback must
// synthesize one after the drop.
func TestCommentOnlyViewChangeIsRecreated(t *testing.T) {
	snapshot := func(columnType, comment string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{
				{
					Name: "upper", Definition: "SELECT id FROM public.lower", Comment: comment,
					DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "lower"}},
				},
				{
					Name: "lower", Definition: "SELECT FROM",
					DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "t", Column: "id"}},
				},
			},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB,
		snapshot("integer", "old comment"), snapshot("bigint", "new comment"))
	require.NoError(t, err)
	dropUpper := strings.Index(sql, `DROP VIEW IF EXISTS "public"."upper"`)
	createUpper := strings.Index(sql, `CREATE VIEW "public"."upper"`)
	require.GreaterOrEqual(t, dropUpper, 0, sql)
	require.GreaterOrEqual(t, createUpper, 0, sql)
	require.Less(t, dropUpper, createUpper, sql)
	comment := strings.Index(sql, "COMMENT ON VIEW")
	require.GreaterOrEqual(t, comment, 0, sql)
	require.Less(t, createUpper, comment, sql)
}

// TestIndexOnlyMatViewChangeRecreatesBeforeItsIndex covers a selected dependent
// materialized view whose only change is an index: the synthesized definition
// carries the index, so the catalog index ops must not run before it.
func TestIndexOnlyMatViewChangeRecreatesBeforeItsIndex(t *testing.T) {
	snapshot := func(columnType string, unique bool) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{{
				Name: "lower", Definition: "SELECT FROM",
				DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "t", Column: "id"}},
			}},
			MaterializedViews: []*metadatapb.MaterializedViewMetadata{{
				Name:       "mv",
				Definition: "SELECT id FROM public.lower",
				Indexes:    []*metadatapb.IndexMetadata{{Name: "idx_mv", Expressions: []string{"id"}, Unique: unique}},
				DependencyColumns: []*metadatapb.DependencyColumn{{
					Schema: "public", Table: "lower",
				}},
			}},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer", false), snapshot("bigint", true))
	require.NoError(t, err)
	createView := strings.Index(sql, `CREATE MATERIALIZED VIEW "public"."mv"`)
	createIndex := strings.Index(sql, `CREATE UNIQUE INDEX "idx_mv"`)
	require.GreaterOrEqual(t, createView, 0, sql)
	require.GreaterOrEqual(t, createIndex, 0, sql)
	require.Less(t, createView, createIndex, sql)
	require.NotContains(t, sql, "DROP INDEX")
}

// TestIndexByNameUsesOwner covers snapshots where two relations carry an index
// of the same name: the lookup must use the relation named in the create SQL.
func TestIndexByNameUsesOwner(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "public",
		Tables: []*metadatapb.TableMetadata{
			{Name: "a", Indexes: []*metadatapb.IndexMetadata{{Name: "idx", Definition: "definition a"}}},
			{Name: "b", Indexes: []*metadatapb.IndexMetadata{{Name: "idx", Definition: "definition b"}}},
		},
	}}}
	require.Equal(t, "definition a", indexByName(meta, "public", "public", "a", "idx").GetDefinition())
	require.Equal(t, "definition b", indexByName(meta, "public", "public", "b", "idx").GetDefinition())
}

// TestRemovedDependentViewDropsBeforeUnloadableView covers an old-only upper
// view: it is not recreated, but its drop must precede the lower view it
// depends on.
func TestRemovedDependentViewDropsBeforeUnloadableView(t *testing.T) {
	lower := func() *metadatapb.ViewMetadata {
		return &metadatapb.ViewMetadata{
			Name: "lower", Definition: "SELECT FROM",
			DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "t", Column: "id"}},
		}
	}
	upper := &metadatapb.ViewMetadata{
		Name: "upper", Definition: "SELECT id FROM public.lower",
		DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "lower"}},
	}
	snapshot := func(columnType string, views ...*metadatapb.ViewMetadata) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views:  views,
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB,
		snapshot("integer", upper, lower()), snapshot("bigint", lower()))
	require.NoError(t, err)
	dropUpper := strings.Index(sql, `DROP VIEW IF EXISTS "public"."upper"`)
	dropLower := strings.Index(sql, `DROP VIEW IF EXISTS "public"."lower"`)
	require.GreaterOrEqual(t, dropUpper, 0, sql)
	require.GreaterOrEqual(t, dropLower, 0, sql)
	require.Less(t, dropUpper, dropLower, sql)
}

// TestSplitRetypeToCreatedType covers a composite attribute moving to a type
// this migration creates: the attribute is dropped before the old table and
// added after the new type exists.
func TestSplitRetypeToCreatedType(t *testing.T) {
	oldMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:           "public",
		Tables:         []*metadatapb.TableMetadata{pgTable("old_t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
		CompositeTypes: []*metadatapb.CompositeTypeMetadata{{Name: "c", Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "r", Type: "public.old_t"}}}},
	}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	newMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:           "public",
		Tables:         []*metadatapb.TableMetadata{pgTable("new_t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: "integer"})},
		EnumTypes:      []*metadatapb.EnumTypeMetadata{{Name: "e", Values: []string{"a", "b"}}},
		CompositeTypes: []*metadatapb.CompositeTypeMetadata{{Name: "c", Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "r", Type: "public.e"}}}},
	}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, oldMeta, newMeta)
	require.NoError(t, err)
	dropAttribute := strings.Index(sql, `DROP ATTRIBUTE "r"`)
	createType := strings.Index(sql, `CREATE TYPE "public"."e"`)
	addAttribute := strings.Index(sql, `ADD ATTRIBUTE "r" public.e`)
	require.GreaterOrEqual(t, dropAttribute, 0, sql)
	require.GreaterOrEqual(t, createType, 0, sql)
	require.GreaterOrEqual(t, addAttribute, 0, sql)
	require.Less(t, dropAttribute, createType, sql)
	require.Less(t, createType, addAttribute, sql)
}

// TestReversedDependentViewsUseSeparateDropAndCreateOrders covers dependents
// whose relationship reverses: drops follow the old graph, creates the target.
func TestReversedDependentViewsUseSeparateDropAndCreateOrders(t *testing.T) {
	view := func(name, definition string, deps ...string) *metadatapb.ViewMetadata {
		v := &metadatapb.ViewMetadata{Name: name, Definition: definition}
		for _, dep := range deps {
			v.DependencyColumns = append(v.DependencyColumns, &metadatapb.DependencyColumn{Schema: "public", Table: dep})
		}
		return v
	}
	snapshot := func(columnType string, reversed bool) *model.DatabaseMetadata {
		lower := view("lower", "SELECT FROM", "t")
		lower.Columns = []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}
		views := []*metadatapb.ViewMetadata{lower}
		if reversed {
			views = append(views, view("a", "SELECT id FROM public.lower", "lower"), view("b", "SELECT id FROM public.a", "a"))
		} else {
			views = append(views, view("b", "SELECT id FROM public.lower", "lower"), view("a", "SELECT id FROM public.b", "b"))
		}
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views:  views,
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}

	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer", false), snapshot("bigint", true))
	require.NoError(t, err)
	dropB := strings.Index(sql, `DROP VIEW IF EXISTS "public"."b"`)
	dropA := strings.Index(sql, `DROP VIEW IF EXISTS "public"."a"`)
	dropLower := strings.Index(sql, `DROP VIEW IF EXISTS "public"."lower"`)
	createLower := strings.Index(sql, `CREATE VIEW "public"."lower"`)
	createA := strings.Index(sql, `CREATE VIEW "public"."a"`)
	createB := strings.Index(sql, `CREATE VIEW "public"."b"`)
	require.GreaterOrEqual(t, dropB, 0, sql)
	require.GreaterOrEqual(t, dropA, 0, sql)
	require.GreaterOrEqual(t, dropLower, 0, sql)
	require.GreaterOrEqual(t, createLower, 0, sql)
	require.GreaterOrEqual(t, createA, 0, sql)
	require.GreaterOrEqual(t, createB, 0, sql)
	require.Less(t, dropA, dropB, sql)
	require.Less(t, dropB, dropLower, sql)
	require.Less(t, createLower, createA, sql)
	require.Less(t, createA, createB, sql)
}

// TestAddedDependentViewFollowsRestoredView covers a target-only view above a
// restored view: it has no drop, but its create must follow the lower create.
func TestAddedDependentViewFollowsRestoredView(t *testing.T) {
	snapshot := func(columnType string, addUpper bool) *model.DatabaseMetadata {
		views := []*metadatapb.ViewMetadata{{
			Name: "lower", Definition: "SELECT FROM",
			DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "t", Column: "id"}},
		}}
		if addUpper {
			views = append(views, &metadatapb.ViewMetadata{
				Name: "upper", Definition: "SELECT id FROM public.lower",
				DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "lower"}},
			})
		}
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views:  views,
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer", false), snapshot("bigint", true))
	require.NoError(t, err)
	createLower := strings.Index(sql, `CREATE VIEW "public"."lower"`)
	createUpper := strings.Index(sql, `CREATE VIEW "public"."upper"`)
	require.GreaterOrEqual(t, createLower, 0, sql)
	require.GreaterOrEqual(t, createUpper, 0, sql)
	require.Less(t, createLower, createUpper, sql)
	require.NotContains(t, sql, `DROP VIEW IF EXISTS "public"."upper"`)
}

// TestIndirectUnloadableDependentIsRestored covers an unloadable view whose
// dependency on the retyped column is through a loadable lower view.
func TestIndirectUnloadableDependentIsRestored(t *testing.T) {
	snapshot := func(columnType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{
				{
					Name: "lower", Definition: "SELECT id FROM public.t",
					DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "t", Column: "id"}},
				},
				{
					Name: "upper", Definition: "SELECT FROM",
					DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "lower"}},
				},
			},
		}}}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer"), snapshot("bigint"))
	require.NoError(t, err)
	dropUpper := strings.Index(sql, `DROP VIEW IF EXISTS "public"."upper"`)
	createUpper := strings.Index(sql, `CREATE VIEW "public"."upper"`)
	require.GreaterOrEqual(t, dropUpper, 0, sql)
	require.GreaterOrEqual(t, createUpper, 0, sql)
	require.Less(t, dropUpper, createUpper, sql)
}

// TestAddedSchemaDependentViewFollowsRestoredView covers a view in a target-only
// schema: it is a create-only candidate and must follow the restored view it
// references.
func TestAddedSchemaDependentViewFollowsRestoredView(t *testing.T) {
	snapshot := func(columnType string, addSchema bool) *model.DatabaseMetadata {
		schemas := []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{pgTable("t", 0, &metadatapb.ColumnMetadata{Name: "id", Type: columnType})},
			Views: []*metadatapb.ViewMetadata{{
				Name: "lower", Definition: "SELECT FROM",
				DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "t", Column: "id"}},
			}},
		}}
		if addSchema {
			schemas = append(schemas, &metadatapb.SchemaMetadata{
				Name: "extra",
				Views: []*metadatapb.ViewMetadata{{
					Name: "upper", Definition: "SELECT id FROM public.lower",
					DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "lower"}},
				}},
			})
		}
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: schemas}, nil, nil, storepb.Engine_COCKROACHDB, true)
	}
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, snapshot("integer", false), snapshot("bigint", true))
	require.NoError(t, err)
	createLower := strings.Index(sql, `CREATE VIEW "public"."lower"`)
	createUpper := strings.Index(sql, `CREATE VIEW "extra"."upper"`)
	require.GreaterOrEqual(t, createLower, 0, sql)
	require.GreaterOrEqual(t, createUpper, 0, sql)
	require.Less(t, createLower, createUpper, sql)
}

// TestObjectKeyIsUnambiguous covers identifiers containing dots: the key must
// not collide for schema "a.b"/view "c" and schema "a"/view "b.c".
func TestObjectKeyIsUnambiguous(t *testing.T) {
	require.NotEqual(t, objectKey("a.b", "c"), objectKey("a", "b.c"))
	schemaName, objectName := splitObjectKey(objectKey("a.b", "c"))
	require.Equal(t, "a.b", schemaName)
	require.Equal(t, "c", objectName)
}
