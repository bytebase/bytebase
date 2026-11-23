package plsql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oracle/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResourcesSeam(t *testing.T) {
	dbMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "HR",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "HR",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "EMPLOYEES",
					},
					{
						Name: "DEPARTMENTS",
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "IDX_DEPT_NAME"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_ORACLE, false)

	parsed, err := base.ParseStatements(storepb.Engine_ORACLE, "UPDATE employees SET id = 2; INSERT INTO employees VALUES (1); TRUNCATE TABLE employees; DROP INDEX idx_dept_name;")
	require.NoError(t, err)

	var asts []base.AST
	for _, stmt := range parsed {
		if stmt.AST != nil {
			asts = append(asts, stmt.AST)
		}
	}
	// Non-*AST and nil entries are skipped without error.
	asts = append(asts, &fakeAST{}, nil)

	summary, err := base.ExtractChangedResources(storepb.Engine_ORACLE, "HR", "", dbMeta, asts, "")
	require.NoError(t, err)
	require.Equal(t, 1, summary.DMLCount)
	require.Equal(t, 1, summary.InsertCount)
	require.Len(t, summary.DMLStatements, 1)

	want := model.NewChangedResources(dbMeta)
	want.AddTable("HR", "", &storepb.ChangedResourceTable{Name: "EMPLOYEES"}, true)
	want.AddTable("HR", "", &storepb.ChangedResourceTable{Name: "DEPARTMENTS"}, false)
	require.Equal(t, want, summary.ChangedResources)

	// DatabaseOnly resources call AddDatabase.
	dbOnlySummary, err := base.ExtractChangedResourcesFromASTs(nil, []base.AST{&fakeAST{}}, false, func(base.AST) (analysis.StatementChanges, bool, error) {
		return analysis.StatementChanges{
			Resources: []analysis.ChangedResource{
				{Database: "APP", DatabaseOnly: true},
			},
		}, true, nil
	})
	require.NoError(t, err)
	wantDBOnly := model.NewChangedResources(nil)
	wantDBOnly.AddDatabase("APP")
	require.Equal(t, wantDBOnly, dbOnlySummary.ChangedResources)
}
