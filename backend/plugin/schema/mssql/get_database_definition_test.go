package mssql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestDefinitionRegistry(t *testing.T) {
	table := &metadatapb.TableMetadata{
		Name: "t",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id", Type: "int", Nullable: false},
		},
	}
	view := &metadatapb.ViewMetadata{
		Name:       "v",
		Definition: "CREATE VIEW [dbo].[v] AS SELECT 1;",
	}
	function := &metadatapb.FunctionMetadata{
		Name:       "f",
		Definition: "ALTER FUNCTION [dbo].[f]() RETURNS int AS BEGIN RETURN 1 END",
	}
	procedure := &metadatapb.ProcedureMetadata{
		Name:       "p",
		Definition: "CREATE PROCEDURE [dbo].[p] AS SELECT 1",
	}
	dbMeta := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:       "dbo",
			Tables:     []*metadatapb.TableMetadata{table},
			Views:      []*metadatapb.ViewMetadata{view},
			Functions:  []*metadatapb.FunctionMetadata{function},
			Procedures: []*metadatapb.ProcedureMetadata{procedure},
		}},
	}

	got, err := schema.GetDatabaseDefinition(storepb.Engine_MSSQL, schema.GetDefinitionContext{}, dbMeta)
	require.NoError(t, err)
	require.Contains(t, got, "CREATE TABLE [dbo].[t]")
	require.Contains(t, got, "CREATE VIEW [dbo].[v] AS SELECT 1;;")
	require.Contains(t, got, "ALTER FUNCTION [dbo].[f]")
	require.Contains(t, got, "CREATE PROCEDURE [dbo].[p]")

	tableDDL, err := schema.GetTableDefinition(storepb.Engine_MSSQL, "dbo", table, nil)
	require.NoError(t, err)
	require.Contains(t, tableDDL, "CREATE TABLE [dbo].[t]")

	viewDDL, err := schema.GetViewDefinition(storepb.Engine_MSSQL, "dbo", view)
	require.NoError(t, err)
	require.Equal(t, "CREATE VIEW [dbo].[v] AS SELECT 1;;\n\nGO\n\n", viewDDL)

	fnDDL, err := schema.GetFunctionDefinition(storepb.Engine_MSSQL, "dbo", function)
	require.NoError(t, err)
	require.Equal(t, "ALTER FUNCTION [dbo].[f]() RETURNS int AS BEGIN RETURN 1 END\n\nGO\n\n", fnDDL)

	procDDL, err := schema.GetProcedureDefinition(storepb.Engine_MSSQL, "dbo", procedure)
	require.NoError(t, err)
	require.Equal(t, "CREATE PROCEDURE [dbo].[p] AS SELECT 1\n\nGO\n\n", procDDL)

	require.Nil(t, stripUnrenderedFacts(nil))
	require.Nil(t, stripTableUnrenderedFacts(nil))
}
