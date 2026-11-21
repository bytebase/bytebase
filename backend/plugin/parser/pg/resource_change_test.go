package pg

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResourcesSeam(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:       "db",
		SearchPath: `"$user", public`,
		Schemas: []*metadatapb.SchemaMetadata{
			{Name: "app", Functions: []*metadatapb.FunctionMetadata{{Name: "t1"}}},
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name:     "t1",
					RowCount: 42,
					Indexes:  []*metadatapb.IndexMetadata{{Name: "t1_idx"}},
				}},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	const sql = "SET search_path TO app;\nUPDATE t1 SET a = 1;\nDROP INDEX public.t1_idx;\nDROP TABLE t1;"
	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, sql)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)

	summary, err := base.ExtractChangedResources(storepb.Engine_POSTGRES, "db", "", dbMetadata, asts, sql)
	require.NoError(t, err)
	require.Equal(t, 1, summary.DMLCount)
	require.Equal(t, []string{"SET LOCAL search_path TO \"app\";\nUPDATE t1 SET a = 1;"}, summary.DMLStatements)

	// Selected schema falls back to public and ignores routines of the same name.
	insStmts, err := base.ParseStatements(storepb.Engine_POSTGRES, "INSERT INTO t1 VALUES (1), (2);")
	require.NoError(t, err)
	insSummary, err := base.ExtractChangedResources(storepb.Engine_POSTGRES, "db", "app", dbMetadata, base.ExtractASTs(insStmts), "INSERT INTO t1 VALUES (1), (2);")
	require.NoError(t, err)
	require.Equal(t, 2, insSummary.InsertCount)
	want := model.NewChangedResources(dbMetadata)
	want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: "t1"}, false)
	require.Equal(t, want, insSummary.ChangedResources)

	// Empty AST list returns empty summary; foreign AST fails cleanly.
	emptySummary, err := base.ExtractChangedResources(storepb.Engine_POSTGRES, "db", "", dbMetadata, nil, "")
	require.NoError(t, err)
	require.Empty(t, emptySummary.DMLStatements)

	_, err = base.ExtractChangedResources(storepb.Engine_POSTGRES, "db", "", dbMetadata, []base.AST{wrongAST{}}, "")
	require.ErrorContains(t, err, "expected AST for PostgreSQL")
}
