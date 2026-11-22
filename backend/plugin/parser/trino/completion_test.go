package trino

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func hasCandidate(cands []base.Candidate, typ base.CandidateType, text string) bool {
	for _, c := range cands {
		if c.Type == typ && c.Text == text {
			return true
		}
	}
	return false
}

func TestCompletion_Seam(t *testing.T) {
	catalogs := map[string]*metadatapb.DatabaseSchemaMetadata{
		"Company": {
			Name: "Company",
			Schemas: []*metadatapb.SchemaMetadata{{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{Name: "Employees", Columns: []*metadatapb.ColumnMetadata{{Name: "Id", Type: "int"}, {Name: "Name", Type: "varchar"}}},
				},
				Views: []*metadatapb.ViewMetadata{
					{Name: "ActiveEmployees", Columns: []*metadatapb.ColumnMetadata{{Name: "Id", Type: "int"}}},
				},
			}},
		},
		"School": {
			Name: "School",
			Schemas: []*metadatapb.SchemaMetadata{{
				Name:   "sch",
				Tables: []*metadatapb.TableMetadata{{Name: "Student", Columns: []*metadatapb.ColumnMetadata{{Name: "Id", Type: "int"}}}},
			}},
		},
		"Warehouse": {
			Name: "Warehouse",
			Schemas: []*metadatapb.SchemaMetadata{{
				Name:   "wh",
				Tables: []*metadatapb.TableMetadata{{Name: "Item", Columns: []*metadatapb.ColumnMetadata{{Name: "Id", Type: "int"}}}},
			}},
		},
	}

	loads := map[string]int{}
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "Company",
		DefaultSchema:   "dbo",
		Metadata: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
			loads[db]++
			if m, ok := catalogs[db]; ok {
				return "", model.NewDatabaseMetadata(m, nil, nil, storepb.Engine_TRINO, false), nil
			}
			return "", nil, errors.New("not found")
		},
		ListDatabaseNames: func(context.Context, string) ([]string, error) {
			return []string{"Company", "School", "Warehouse"}, nil
		},
	}

	// Registry dispatch, catalog->DATABASE / schema / table / view mapping, and lazy catalog loading.
	gotFrom, err := base.Completion(context.Background(), storepb.Engine_TRINO, cCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeDatabase, "company"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeDatabase, "school"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeSchema, "dbo"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeTable, "employees"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeView, "activeemployees"), "got %v", gotFrom)
	require.NotZero(t, loads["Company"])
	require.Zero(t, loads["School"])
	require.Zero(t, loads["Warehouse"])

	// Naming another catalog loads only it; column candidate type maps cleanly.
	clear(loads)
	gotSchool, err := base.Completion(context.Background(), storepb.Engine_TRINO, cCtx, "SELECT * FROM School.", 1, len("SELECT * FROM School."))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotSchool, base.CandidateTypeSchema, "sch"), "got %v", gotSchool)
	require.NotZero(t, loads["School"])
	require.Zero(t, loads["Warehouse"])

	gotCols, err := base.Completion(context.Background(), storepb.Engine_TRINO, cCtx, "SELECT  FROM Employees", 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotCols, base.CandidateTypeColumn, "id"), "got %v", gotCols)

	// Nil metadata degrades cleanly without object candidates.
	gotNil, err := base.Completion(context.Background(), storepb.Engine_TRINO, base.CompletionContext{Scene: base.SceneTypeAll}, "SELECT * FROM ", 1, len("SELECT * FROM "))
	require.NoError(t, err)
	for _, c := range gotNil {
		require.NotContains(t, []base.CandidateType{
			base.CandidateTypeDatabase, base.CandidateTypeSchema, base.CandidateTypeTable,
			base.CandidateTypeView, base.CandidateTypeColumn,
		}, c.Type)
	}
}
