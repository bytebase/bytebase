package cockroachdb

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

func snapshot(tables ...*metadatapb.TableMetadata) *metadatapb.DatabaseSchemaMetadata {
	return &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: tables,
			Views: []*metadatapb.ViewMetadata{{
				Name:              "v",
				Definition:        "SELECT id FROM app.public.t",
				DependencyColumns: []*metadatapb.DependencyColumn{{Schema: "public", Table: "t", Column: "id"}},
			}},
			Functions: []*metadatapb.FunctionMetadata{{
				Name:       "calc",
				Signature:  "calc()",
				Definition: "CREATE FUNCTION public.calc()\n\tRETURNS INT8\n\tLANGUAGE SQL\n\tAS $$\n\tSELECT 1;\n$$",
			}},
		}},
	}
}

func table(columns ...*metadatapb.ColumnMetadata) *metadatapb.TableMetadata {
	return &metadatapb.TableMetadata{
		Name:    "t",
		Columns: columns,
		Indexes: []*metadatapb.IndexMetadata{{
			Name: "t_pkey", Primary: true, Unique: true, Expressions: []string{"id"},
			Definition: "CREATE UNIQUE INDEX t_pkey ON app.public.t USING btree (id ASC)",
		}},
	}
}

var (
	id   = &metadatapb.ColumnMetadata{Name: "id", Type: "bigint"}
	note = &metadatapb.ColumnMetadata{Name: "note", Type: "text", Nullable: true}
)

// TestDefinitions proves CockroachDB's definitions come from omni's
// CockroachDB catalog: the primary key inside CREATE TABLE, and the
// database's own name dropped from the view.
func TestDefinitions(t *testing.T) {
	meta := snapshot(table(id))
	def, err := schema.GetDatabaseDefinition(storepb.Engine_COCKROACHDB, schema.GetDefinitionContext{PrintHeader: true}, meta)
	require.NoError(t, err)
	require.Contains(t, def, "CREATE TABLE \"public\".\"t\" (\n    \"id\" bigint NOT NULL,\n    CONSTRAINT \"t_pkey\" PRIMARY KEY (id ASC)\n);")
	require.Contains(t, def, "SELECT id FROM public.t;")
	require.NotContains(t, def, "app.public")

	tableDef, err := schema.GetTableDefinition(storepb.Engine_COCKROACHDB, "public", meta.Schemas[0].Tables[0], nil)
	require.NoError(t, err)
	require.Contains(t, tableDef, "CREATE TABLE")

	files, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_COCKROACHDB, schema.GetDefinitionContext{}, meta)
	require.NoError(t, err)
	var names []string
	for _, file := range files.Files {
		names = append(names, file.Name)
	}
	require.Equal(t, []string{"schemas/public/tables/t.sql", "schemas/public/views/v.sql", "schemas/public/functions/calc.sql"}, names)
}

func TestDiffMigration(t *testing.T) {
	source := model.NewDatabaseMetadata(snapshot(table(id)), nil, nil, storepb.Engine_COCKROACHDB, true)
	target := model.NewDatabaseMetadata(snapshot(table(id, note)), nil, nil, storepb.Engine_COCKROACHDB, true)
	sql, err := schema.DiffMigration(storepb.Engine_COCKROACHDB, source, target)
	require.NoError(t, err)
	require.Equal(t, `ALTER TABLE "public"."t" ADD COLUMN "note" text`, sql)
}

func TestDiffSDLMigration(t *testing.T) {
	sql, err := schema.DiffSDLMigration(storepb.Engine_COCKROACHDB,
		"CREATE TABLE public.users (id INT8 PRIMARY KEY);",
		"CREATE TABLE public.users (id INT8 PRIMARY KEY, email STRING);",
		"",
	)
	require.NoError(t, err)
	require.Equal(t, `ALTER TABLE "public"."users" ADD COLUMN "email" STRING`, sql)

	_, err = schema.DiffSDLMigration(storepb.Engine_COCKROACHDB, "CREATE TABLE (", "", "")
	require.ErrorContains(t, err, "failed to load source schema")
	_, err = schema.DiffSDLMigration(storepb.Engine_COCKROACHDB, "", "CREATE TABLE (", "")
	require.ErrorContains(t, err, "failed to load target schema")
}

func TestSDLDropAdvices(t *testing.T) {
	current := snapshot(table(id, note), &metadatapb.TableMetadata{Name: "legacy", Columns: []*metadatapb.ColumnMetadata{id}})
	meta := model.NewDatabaseMetadata(current, nil, nil, storepb.Engine_COCKROACHDB, false)

	sdl, err := schema.MetadataToSDL(storepb.Engine_COCKROACHDB, meta)
	require.NoError(t, err)
	advices, err := schema.SDLDropAdvices(storepb.Engine_COCKROACHDB, sdl, meta, "")
	require.NoError(t, err)
	require.Empty(t, advices)

	target := strings.TrimSpace(`
CREATE TABLE public.t (id INT8 PRIMARY KEY);
CREATE VIEW public.v AS SELECT id FROM public.t;
CREATE FUNCTION public.calc() RETURNS INT8 LANGUAGE SQL AS $$ SELECT 2 $$;
`)
	advices, err = schema.SDLDropAdvices(storepb.Engine_COCKROACHDB, target, meta, "")
	require.NoError(t, err)
	var drops, replaces []string
	for _, advice := range advices {
		require.Equal(t, storepb.Advice_WARNING, advice.Status)
		switch advice.Code {
		case code.SDLDropOperation.Int32():
			drops = append(drops, advice.Content)
		case code.SDLReplaceOperation.Int32():
			replaces = append(replaces, advice.Content)
		default:
			t.Fatalf("unexpected advice %+v", advice)
		}
	}
	require.ElementsMatch(t, []string{
		"Dropping table 'public.legacy' will result in data loss.",
		"Dropping column from table 'public.t' will result in data loss.",
	}, drops)
	require.Equal(t, []string{"Function 'public.calc' definition will be replaced."}, replaces)
}
