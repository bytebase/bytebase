package tsql

import (
	"context"
	"errors"
	"os"
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

func TestCompletionDoesNotDependOnANTLR(t *testing.T) {
	content, err := os.ReadFile("tsql.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/tsql")
}

func TestCompletion_Seam(t *testing.T) {
	dbs := map[string]*metadatapb.DatabaseSchemaMetadata{
		"Company": {
			Name: "Company",
			Schemas: []*metadatapb.SchemaMetadata{
				{
					Name: "dbo",
					Tables: []*metadatapb.TableMetadata{
						{
							Name: "Employees",
							Columns: []*metadatapb.ColumnMetadata{
								{Name: "Id", Type: "int"},
								{Name: "Name", Type: "varchar"},
							},
						},
						{
							Name:    "Order Items",
							Columns: []*metadatapb.ColumnMetadata{{Name: "Item ID", Type: "int"}},
						},
					},
					Sequences:  []*metadatapb.SequenceMetadata{{Name: "EmployeeIdSeq"}},
					Procedures: []*metadatapb.ProcedureMetadata{{Name: "SyncEmployees"}},
				},
				{
					Name: "MySchema",
					Tables: []*metadatapb.TableMetadata{
						{Name: "SalaryLevel", Columns: []*metadatapb.ColumnMetadata{{Name: "SalaryUpBound", Type: "int"}}},
					},
					Views: []*metadatapb.ViewMetadata{{Name: "SalaryView"}},
				},
			},
		},
		"School": {
			Name: "School",
			Schemas: []*metadatapb.SchemaMetadata{{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{Name: "Student", Columns: []*metadatapb.ColumnMetadata{{Name: "ParentName", Type: "varchar"}}},
				},
			}},
		},
	}

	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "Company",
		Metadata: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
			if m, ok := dbs[db]; ok {
				return "", model.NewDatabaseMetadata(m, nil, nil, storepb.Engine_MSSQL, false), nil
			}
			return "", nil, errors.New("not found")
		},
		ListDatabaseNames: func(context.Context, string) ([]string, error) {
			return []string{"Company", "School"}, nil
		},
	}

	// Registry dispatch, quoting, and schema/table/sequence candidate mapping.
	gotFrom, err := base.Completion(context.Background(), storepb.Engine_MSSQL, cCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeTable, "Employees"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeTable, "[Order Items]"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeSchema, "MySchema"), "got %v", gotFrom)
	require.True(t, hasCandidate(gotFrom, base.CandidateTypeSequence, "EmployeeIdSeq"), "got %v", gotFrom)

	// View, routine, database, column, function (), and $action candidate mappings.
	gotView, err := base.Completion(context.Background(), storepb.Engine_MSSQL, cCtx, "SELECT * FROM MySchema.", 1, len("SELECT * FROM MySchema."))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotView, base.CandidateTypeView, "SalaryView"), "got %v", gotView)

	gotProc, err := base.Completion(context.Background(), storepb.Engine_MSSQL, cCtx, "EXEC ", 1, len("EXEC "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotProc, base.CandidateTypeRoutine, "SyncEmployees"), "got %v", gotProc)

	gotDB, err := base.Completion(context.Background(), storepb.Engine_MSSQL, cCtx, "USE ", 1, len("USE "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotDB, base.CandidateTypeDatabase, "School"), "got %v", gotDB)

	gotColFn, err := base.Completion(context.Background(), storepb.Engine_MSSQL, cCtx, "SELECT COU FROM Employees", 1, len("SELECT COU"))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotColFn, base.CandidateTypeFunction, "COUNT()"), "got %v", gotColFn)

	// DefaultSchema wiring resolves non-dbo tables unqualified.
	schemaCtx := cCtx
	schemaCtx.DefaultSchema = "MySchema"
	gotSchema, err := base.Completion(context.Background(), storepb.Engine_MSSQL, schemaCtx, "SELECT  FROM SalaryLevel", 1, len("SELECT "))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotSchema, base.CandidateTypeColumn, "SalaryUpBound"), "got %v", gotSchema)

	// Pseudo-column $action surfaces unquoted as a column candidate.
	mergeSQL := "MERGE INTO Employees AS d USING Employees AS s ON d.Id = s.Id WHEN MATCHED THEN UPDATE SET d.Name = s.Name OUTPUT "
	gotMerge, err := base.Completion(context.Background(), storepb.Engine_MSSQL, cCtx, mergeSQL, 1, len(mergeSQL))
	require.NoError(t, err)
	require.True(t, hasCandidate(gotMerge, base.CandidateTypeColumn, "$action"), "got %v", gotMerge)

	// Nil or failing metadata callback returns no object candidates and no error.
	for _, badCtx := range []base.CompletionContext{
		{Scene: base.SceneTypeAll, DefaultDatabase: "Company"},
		{
			Scene:           base.SceneTypeAll,
			DefaultDatabase: "Company",
			Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
				return "", nil, errors.New("metadata unavailable")
			},
			ListDatabaseNames: func(context.Context, string) ([]string, error) {
				return []string{"Company"}, nil
			},
		},
	} {
		got, err := base.Completion(context.Background(), storepb.Engine_MSSQL, badCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
		require.NoError(t, err)
		for _, c := range got {
			require.NotContains(t, []base.CandidateType{base.CandidateTypeSchema, base.CandidateTypeTable, base.CandidateTypeColumn}, c.Type)
		}
	}
}
