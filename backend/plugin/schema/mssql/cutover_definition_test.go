package mssql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/catalog"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// factsMetadata returns a snapshot carrying every fact the pre-cutover dump
// omitted. stripUnrenderedFacts must make the definition ignore all of them.
func factsMetadata() (*metadatapb.DatabaseSchemaMetadata, *metadatapb.DatabaseSchemaMetadata) {
	withFacts := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:     "dbo",
				SkipDump: true,
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
	// Stripping must not mutate the caller's metadata.
	s := withFacts.Schemas[0]
	require.True(t, s.SkipDump)
	require.Len(t, s.Sequences, 1)
	require.Len(t, s.Synonyms, 1)
	require.True(t, s.Tables[0].SkipDump)
	require.Equal(t, "table comment", s.Tables[0].Comment)
	require.Len(t, s.Tables[0].Triggers, 1)
	require.Equal(t, "DF_t_v", s.Tables[0].Columns[1].DefaultConstraintName)
	require.True(t, s.Views[0].SkipDump)
	require.Equal(t, "view comment", s.Views[0].Comment)
	require.Len(t, s.Views[0].Triggers, 1)
	require.True(t, s.Functions[0].SkipDump)
	require.True(t, s.Procedures[0].SkipDump)
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

// TestTableDefinitionPreservesUnrenderedFacts pins GetTableDefinition to the
// same strip contract as GetDatabaseDefinition: SkipDump, comments, triggers,
// and DefaultConstraintName are stripped before delegating to omni's catalog.
func TestTableDefinitionPreservesUnrenderedFacts(t *testing.T) {
	withFacts, bare := factsMetadata()

	got, err := schema.GetTableDefinition(storepb.Engine_MSSQL, "dbo", withFacts.Schemas[0].Tables[0], nil)
	require.NoError(t, err)
	want, err := GetTableDefinition("dbo", bare.Schemas[0].Tables[0], nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Contains(t, got, "CREATE TABLE [dbo].[t]")
	require.NotContains(t, got, "sp_addextendedproperty")
	require.NotContains(t, got, "DF_t_v")
	require.NotContains(t, got, "CREATE TRIGGER")
	// Stripping must not mutate the caller's table metadata.
	tbl := withFacts.Schemas[0].Tables[0]
	require.True(t, tbl.SkipDump)
	require.Equal(t, "table comment", tbl.Comment)
	require.Len(t, tbl.Triggers, 1)
	require.Equal(t, "DF_t_v", tbl.Columns[1].DefaultConstraintName)
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

// TestDefinitionDoesNotRewriteModuleTextInsideTableLiteral pins the batch
// recognition: a GO line inside a table default's string literal is not a
// module boundary, so a module text embedded there stays untouched.
func TestDefinitionDoesNotRewriteModuleTextInsideTableLiteral(t *testing.T) {
	view := &metadatapb.ViewMetadata{Name: "v", Definition: "ALTER VIEW [dbo].[v] AS SELECT 1;"}
	catalogView, err := catalog.GetViewDefinition(view)
	require.NoError(t, err)

	meta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "dbo",
		Tables: []*metadatapb.TableMetadata{{
			Name: "t",
			Columns: []*metadatapb.ColumnMetadata{
				{Name: "id", Type: "int", Nullable: false},
				{Name: "script", Type: "nvarchar(max)", Nullable: true, Default: "('" + catalogView + "')"},
			},
		}},
		Views: []*metadatapb.ViewMetadata{view},
	}}}

	got, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	require.NoError(t, err)
	require.Contains(t, got, catalogView, "the table literal keeps the catalog text")
	require.Contains(t, got, legacyViewText(view), "the actual view keeps the legacy text")
}

// TestDefinitionHandlesTrailingBatchWhitespace covers a snapshot whose
// definition is only a schema batch: the scan must stop at EOF, not index the
// byte after the trailing blank line.
func TestDefinitionHandlesTrailingBatchWhitespace(t *testing.T) {
	got, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{{Name: "app"}},
	})
	require.NoError(t, err)
	require.Contains(t, got, "CREATE SCHEMA [app]")
}
