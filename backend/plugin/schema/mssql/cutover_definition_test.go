package mssql

import (
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/catalog"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// factsMetadata returns a snapshot carrying every fact the pre-cutover dump
// omitted. stripUnrenderedFacts must make the definition ignore all of them.
func factsMetadata() (*metadatapb.DatabaseSchemaMetadata, *metadatapb.DatabaseSchemaMetadata) {
	withFacts := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:     "dbo",
				Synonyms: []*metadatapb.SynonymMetadata{{Name: "syn", TargetSchema: "dbo", TargetName: "t"}},
				Sequences: []*metadatapb.SequenceMetadata{{
					Name: "seq", DataType: "bigint", Start: "1", Comment: "seq comment",
				}},
				Tables: []*metadatapb.TableMetadata{{
					Name:     "t",
					Comment:  "table comment",
					SkipDump: true,
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "int", Nullable: false, Comment: "id comment"},
						{Name: "v", Type: "int", Default: "((1))", DefaultConstraintName: "DF_t_v"},
					},
					Indexes: []*metadatapb.IndexMetadata{
						{Name: "PK_t", Type: "CLUSTERED", Primary: true, Unique: true, IsConstraint: true, Expressions: []string{"id"}},
					},
					Triggers: []*metadatapb.TriggerMetadata{{Name: "trg", Body: "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS SELECT 1"}},
				}},
				Views: []*metadatapb.ViewMetadata{{
					Name:       "v",
					Definition: "CREATE VIEW [dbo].[v] AS SELECT [id] FROM [dbo].[t]",
					Comment:    "view comment",
					SkipDump:   true,
					Triggers:   []*metadatapb.TriggerMetadata{{Name: "vtrg", Body: "CREATE TRIGGER [dbo].[vtrg] ON [dbo].[v] INSTEAD OF INSERT AS SELECT 1"}},
				}},
				Functions: []*metadatapb.FunctionMetadata{{
					Name:       "f",
					Definition: "CREATE FUNCTION [dbo].[f]() RETURNS int AS BEGIN RETURN 1 END",
					Comment:    "function comment",
					SkipDump:   true,
				}},
				Procedures: []*metadatapb.ProcedureMetadata{{
					Name:       "p",
					Definition: "CREATE PROCEDURE [dbo].[p] AS SELECT 1",
					Comment:    "procedure comment",
					SkipDump:   true,
				}},
			},
		},
	}
	bare := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{{
					Name: "t",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "int", Nullable: false},
						{Name: "v", Type: "int", Default: "((1))"},
					},
					Indexes: []*metadatapb.IndexMetadata{
						{Name: "PK_t", Type: "CLUSTERED", Primary: true, Unique: true, IsConstraint: true, Expressions: []string{"id"}},
					},
				}},
				Views: []*metadatapb.ViewMetadata{{
					Name:       "v",
					Definition: "CREATE VIEW [dbo].[v] AS SELECT [id] FROM [dbo].[t]",
				}},
				Functions: []*metadatapb.FunctionMetadata{{
					Name:       "f",
					Definition: "CREATE FUNCTION [dbo].[f]() RETURNS int AS BEGIN RETURN 1 END",
				}},
				Procedures: []*metadatapb.ProcedureMetadata{{
					Name:       "p",
					Definition: "CREATE PROCEDURE [dbo].[p] AS SELECT 1",
				}},
			},
		},
	}
	return withFacts, bare
}

// TestDefinitionPreservesUnrenderedFacts pins the dump to the pre-cutover
// renderer: sequences, synonyms, triggers, comments, a default constraint name
// and every SkipDump flag are dropped before the catalog sees the snapshot. A
// change that starts rendering one of them fails here, and that change needs its
// own release note rather than riding along with this cutover.
func TestDefinitionPreservesUnrenderedFacts(t *testing.T) {
	withFacts, bare := factsMetadata()

	got, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, withFacts)
	require.NoError(t, err)
	want, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, bare)
	require.NoError(t, err)
	require.Equal(t, want, got, "stripped facts must not reach the dump")

	for _, omitted := range []string{
		"CREATE SEQUENCE", "CREATE SYNONYM", "CREATE TRIGGER",
		"sp_addextendedproperty", "DF_t_v",
	} {
		require.NotContains(t, got, omitted)
	}
	// The old renderer ignored SkipDump, so a flagged object still rendered.
	for _, rendered := range []string{
		"CREATE TABLE [dbo].[t]", "CREATE VIEW [dbo].[v]",
		"CREATE FUNCTION [dbo].[f]", "CREATE PROCEDURE [dbo].[p]",
	} {
		require.Contains(t, got, rendered)
	}
}

// TestDefinitionModuleTextUnchanged pins the two module-text normalizations the
// catalog applies and this package undoes: an ALTER header and a stored trailing
// semicolon.
func TestDefinitionModuleTextUnchanged(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:  "dbo",
		Views: []*metadatapb.ViewMetadata{{Name: "v", Definition: "ALTER VIEW [dbo].[v] AS SELECT 1;"}},
		Procedures: []*metadatapb.ProcedureMetadata{{
			Name: "p", Definition: "ALTER PROCEDURE [dbo].[p] AS SELECT 1;",
		}},
	}}}
	got, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	require.NoError(t, err)
	require.Contains(t, got, "ALTER VIEW [dbo].[v] AS SELECT 1;;", "stored text and its extra semicolon stay")
	require.Contains(t, got, "ALTER PROCEDURE [dbo].[p] AS SELECT 1;", "an ALTER header stays ALTER")
}

// TestDefinitionModuleOrderMoved names the one ordering difference the cutover
// keeps. The catalog creates a function after any function its body names and
// before a table default or view body that references it; the old renderer wrote
// all functions last. The text of each module is unchanged, only its position.
func TestDefinitionModuleOrderMoved(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "dbo",
		Tables: []*metadatapb.TableMetadata{{
			Name: "t",
			Columns: []*metadatapb.ColumnMetadata{
				{Name: "id", Type: "int", Nullable: false},
				{Name: "v", Type: "int", Nullable: false, Default: "([dbo].[f]())"},
			},
			Indexes: []*metadatapb.IndexMetadata{{Name: "PK_t", Type: "CLUSTERED", Primary: true, Unique: true, IsConstraint: true, Expressions: []string{"id"}}},
		}},
		Views: []*metadatapb.ViewMetadata{{Name: "v1", Definition: "CREATE VIEW [dbo].[v1] AS SELECT [dbo].[g]() AS x"}},
		Functions: []*metadatapb.FunctionMetadata{
			{Name: "f", Definition: "CREATE FUNCTION [dbo].[f]() RETURNS int AS BEGIN RETURN 1 END"},
			{Name: "g", Definition: "CREATE FUNCTION [dbo].[g]() RETURNS int AS BEGIN RETURN 2 END"},
			{Name: "h", Definition: "CREATE FUNCTION [dbo].[h]() RETURNS int AS BEGIN RETURN 3 END"},
		},
	}}}
	got, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	require.NoError(t, err)

	at := func(s string) int {
		i := strings.Index(got, s)
		require.GreaterOrEqual(t, i, 0, "missing %q", s)
		return i
	}
	require.Less(t, at("CREATE FUNCTION [dbo].[f]"), at("CREATE TABLE [dbo].[t]"), "a table-default function is hoisted")
	require.Less(t, at("CREATE FUNCTION [dbo].[g]"), at("CREATE VIEW [dbo].[v1]"), "a view function is hoisted")
	require.Less(t, at("CREATE VIEW [dbo].[v1]"), at("CREATE FUNCTION [dbo].[h]"), "an unreferenced function stays last")
}

// TestGetObjectDefinitionKeepsLegacyText pins the single-object renderers to the
// pre-cutover string. The catalog's own renderers trim the extra semicolon and
// rewrite ALTER to CREATE, so the shim keeps the old formula instead.
func TestGetObjectDefinitionKeepsLegacyText(t *testing.T) {
	view, err := GetViewDefinition("dbo", &metadatapb.ViewMetadata{Name: "v", Definition: "CREATE VIEW [dbo].[v] AS SELECT 1;"})
	require.NoError(t, err)
	require.Equal(t, "CREATE VIEW [dbo].[v] AS SELECT 1;;\n\nGO\n\n", view)

	function, err := GetFunctionDefinition("dbo", &metadatapb.FunctionMetadata{Name: "f", Definition: "ALTER FUNCTION [dbo].[f]() RETURNS int AS BEGIN RETURN 1 END"})
	require.NoError(t, err)
	require.Equal(t, "ALTER FUNCTION [dbo].[f]() RETURNS int AS BEGIN RETURN 1 END\n\nGO\n\n", function)

	procedure, err := GetProcedureDefinition("dbo", &metadatapb.ProcedureMetadata{Name: "p", Definition: "CREATE PROCEDURE [dbo].[p] AS SELECT 1"})
	require.NoError(t, err)
	require.Equal(t, "CREATE PROCEDURE [dbo].[p] AS SELECT 1\n\nGO\n\n", procedure)
}

// TestExtendedPropertyConstraintCommentKeepsBytebaseExtraction pins the metadata
// extractor's handling of an extended property on a constraint. For a named
// @level2type = 'CONSTRAINT' argument Bytebase attaches the value to the table,
// while omni's LoadSQL finds no constraint to attach it to and drops it. The
// shim keeps Bytebase's extractor, so the value still reaches the migration as a
// table comment; this is the known difference the cutover does not change.
func TestExtendedPropertyConstraintCommentKeepsBytebaseExtraction(t *testing.T) {
	sql := `CREATE TABLE [dbo].[t] ([id] int NOT NULL, CONSTRAINT [CK_t] CHECK ([id] > 0));
EXEC sp_addextendedproperty @name = N'MS_Description', @value = N'ck comment',
  @level0type = N'SCHEMA', @level0name = N'dbo',
  @level1type = N'TABLE', @level1name = N't',
  @level2type = N'CONSTRAINT', @level2name = N'CK_t';`

	metadata, err := GetDatabaseMetadata(sql)
	require.NoError(t, err)
	require.Equal(t, "ck comment", metadata.Schemas[0].Tables[0].Comment)

	omniCatalog, err := catalog.LoadSQL(sql)
	require.NoError(t, err)
	require.Empty(t, omniCatalog.Metadata().Schemas[0].Tables[0].Comment)
}

// TestDefinitionRewritesEachModuleAtItsOwnSection pins replacement to the
// catalog-emitted section. The function is hoisted because the view names it,
// and its body returns the catalog's own rendering of that view, so a dump-wide
// first-match replace would rewrite the function body and leave the view
// normalized.
func TestDefinitionRewritesEachModuleAtItsOwnSection(t *testing.T) {
	view := &metadatapb.ViewMetadata{
		Name:       "v",
		Definition: "CREATE VIEW [dbo].[v] AS SELECT [dbo].[f]() AS x;",
	}
	catalogView, err := catalog.GetViewDefinition(view)
	require.NoError(t, err)

	function := &metadatapb.FunctionMetadata{
		Name: "f",
		Definition: "CREATE FUNCTION [dbo].[f]() RETURNS nvarchar(max) AS BEGIN RETURN '" +
			catalogView + "' END",
	}
	meta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "dbo",
		Views:     []*metadatapb.ViewMetadata{view},
		Functions: []*metadatapb.FunctionMetadata{function},
	}}}

	got, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	require.NoError(t, err)
	require.Contains(t, got, function.Definition, "the hoisted function body is untouched")
	require.Contains(t, got, legacyViewText(view), "the actual view keeps the legacy text")
}
