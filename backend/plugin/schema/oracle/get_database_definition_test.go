package oracle

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestOracleDefinitionRegistryAndShims(t *testing.T) {
	table := &metadatapb.TableMetadata{
		Name: "T1",
		Columns: []*metadatapb.ColumnMetadata{
			{
				Name:               "ID",
				Type:               "NUMBER",
				Nullable:           false,
				IsIdentity:         true,
				IdentityGeneration: metadatapb.ColumnMetadata_ALWAYS,
				IdentitySeed:       1,
				IdentityIncrement:  1,
			},
		},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "IDX_T1_ID", Expressions: []string{"ID"}, Visible: false},
		},
	}
	mv := &metadatapb.MaterializedViewMetadata{
		Name:       "MV",
		Definition: "SELECT 1 FROM DUAL",
		Indexes:    []*metadatapb.IndexMetadata{{Name: "IDX_MV", Expressions: []string{"1"}, Visible: false}},
	}
	dbMeta := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:              "APP",
			Tables:            []*metadatapb.TableMetadata{table},
			Views:             []*metadatapb.ViewMetadata{{Name: "V", Definition: "SELECT 1 FROM DUAL"}},
			MaterializedViews: []*metadatapb.MaterializedViewMetadata{mv},
		}},
	}

	t.Run("database definition routes through registry and strips unrendered facts without mutating input", func(t *testing.T) {
		got, err := schema.GetDatabaseDefinition(storepb.Engine_ORACLE, schema.GetDefinitionContext{}, dbMeta)
		require.NoError(t, err)
		require.Contains(t, got, `CREATE TABLE "T1"`)
		require.Contains(t, got, `CREATE VIEW "V"`)
		require.Contains(t, got, `CREATE MATERIALIZED VIEW "MV"`)
		require.NotContains(t, got, "IDENTITY")
		require.NotContains(t, got, "INVISIBLE")
		require.True(t, table.Columns[0].IsIdentity)
		require.False(t, table.Indexes[0].Visible)
		require.False(t, mv.Indexes[0].Visible)
	})

	t.Run("table definition respects SkipDump and strips unrendered facts", func(t *testing.T) {
		skipped, err := schema.GetTableDefinition(storepb.Engine_ORACLE, "", &metadatapb.TableMetadata{
			Name:     "OT",
			SkipDump: true,
			Columns:  []*metadatapb.ColumnMetadata{{Name: "X", Type: "NUMBER"}},
		}, nil)
		require.NoError(t, err)
		require.Empty(t, skipped)

		got, err := schema.GetTableDefinition(storepb.Engine_ORACLE, "", table, nil)
		require.NoError(t, err)
		require.Contains(t, got, `CREATE TABLE "T1"`)
		require.NotContains(t, got, "IDENTITY")
		require.NotContains(t, got, "INVISIBLE")
		require.True(t, table.Columns[0].IsIdentity)
		require.False(t, table.Indexes[0].Visible)
	})

	t.Run("materialized view definition drops index visibility", func(t *testing.T) {
		got, err := schema.GetMaterializedViewDefinition(storepb.Engine_ORACLE, "", mv)
		require.NoError(t, err)
		require.Contains(t, got, `CREATE MATERIALIZED VIEW "MV"`)
		require.NotContains(t, got, "INVISIBLE")
		require.False(t, mv.Indexes[0].Visible)
	})

	t.Run("object definitions registered for view, function, and procedure", func(t *testing.T) {
		viewDDL, err := schema.GetViewDefinition(storepb.Engine_ORACLE, "", &metadatapb.ViewMetadata{
			Name:       "V",
			Definition: "SELECT 1 FROM DUAL",
		})
		require.NoError(t, err)
		require.Contains(t, viewDDL, `CREATE VIEW "V"`)

		fnDDL, err := schema.GetFunctionDefinition(storepb.Engine_ORACLE, "", &metadatapb.FunctionMetadata{
			Name:       "F",
			Definition: "FUNCTION F RETURN NUMBER IS BEGIN RETURN 1; END;",
		})
		require.NoError(t, err)
		require.Contains(t, fnDDL, "CREATE OR REPLACE FUNCTION F")

		procDDL, err := schema.GetProcedureDefinition(storepb.Engine_ORACLE, "", &metadatapb.ProcedureMetadata{
			Name:       "P",
			Definition: "PROCEDURE P IS BEGIN NULL; END;",
		})
		require.NoError(t, err)
		require.Contains(t, procDDL, "CREATE OR REPLACE PROCEDURE P")
	})

	t.Run("nil metadata returns safely from stripUnrenderedFacts", func(t *testing.T) {
		require.Nil(t, stripUnrenderedFacts(nil))
	})
}
