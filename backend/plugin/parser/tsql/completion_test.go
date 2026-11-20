package tsql

import (
	"context"
	"os"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestCompletionTableRefRule(t *testing.T) {
	statement, caretLine, caretPosition := getCaretPosition("TRUNCATE TABLE |")
	getter, lister := buildMockDatabaseMetadataGetterLister()
	results, err := Completion(context.Background(), base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "Company",
		Metadata:          getter,
		ListDatabaseNames: lister,
	}, statement, caretLine, caretPosition)
	require.NoError(t, err)

	require.Contains(t, results, base.Candidate{
		Text: "Employees",
		Type: base.CandidateTypeTable,
	})
}

func TestCompletionUnsyncedDatabase(t *testing.T) {
	for _, input := range []string{"SELECT * FROM |", "SELECT |"} {
		statement, caretLine, caretPosition := getCaretPosition(input)
		results, err := Completion(context.Background(), base.CompletionContext{
			Scene:           base.SceneTypeAll,
			DefaultDatabase: "Company",
			Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
				return "", nil, nil
			},
			ListDatabaseNames: func(context.Context, string) ([]string, error) {
				return []string{"Company"}, nil
			},
		}, statement, caretLine, caretPosition)
		require.NoError(t, err, input)
		for _, candidate := range results {
			require.NotContains(t, []base.CandidateType{base.CandidateTypeSchema, base.CandidateTypeTable, base.CandidateTypeColumn}, candidate.Type, input)
		}
	}
}

func TestCompletionDoesNotDependOnANTLR(t *testing.T) {
	content, err := os.ReadFile("completion.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/tsql")
}

func getCaretPosition(statement string) (string, int, int) {
	lines := strings.Split(statement, "\n")
	for i, line := range lines {
		if offset := strings.Index(line, "|"); offset != -1 {
			newLine := strings.Replace(line, "|", "", 1)
			lines[i] = newLine
			return strings.Join(lines, "\n"), i + 1, offset
		}
	}
	panic("caret position not found")
}

var databaseMetadatas = []*metadatapb.DatabaseSchemaMetadata{
	{
		Name: "Company",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "Employees",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name: "Id",
								Type: "int",
							},
							{
								Name: "Name",
								Type: "varchar",
							},
						},
					},
					{
						Name: "Address",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name: "EmployeeId",
								Type: "int",
							},
							{
								Name: "Street",
								Type: "varchar",
							},
						},
					},
				},
				Sequences: []*metadatapb.SequenceMetadata{
					{
						Name: "EmployeeIdSeq",
					},
					{
						Name: "OrderSeq",
					},
				},
				Procedures: []*metadatapb.ProcedureMetadata{
					{
						Name: "SyncEmployees",
					},
				},
			},
			{
				Name: "MySchema",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "SalaryLevel",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name: "Id",
								Type: "int",
							},
							{
								Name: "SalaryUpBound",
								Type: "int",
							},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{
						Name: "SalaryView",
					},
				},
				Sequences: []*metadatapb.SequenceMetadata{
					{
						Name: "SalarySeq",
					},
				},
				Procedures: []*metadatapb.ProcedureMetadata{
					{
						Name: "SyncSalary",
					},
				},
			},
		},
	},
	{
		Name: "School",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "Student",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name: "Id",
								Type: "int",
							},
							{
								Name: "ParentName",
								Type: "varchar",
							},
						},
					},
				},
				Sequences: []*metadatapb.SequenceMetadata{
					{
						Name: "StudentSeq",
					},
				},
				Procedures: []*metadatapb.ProcedureMetadata{
					{
						Name: "SyncStudents",
					},
				},
			},
		},
	},
}

func buildMockDatabaseMetadataGetterLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadatas {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MSSQL, false /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return "", databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(context.Context, string) ([]string, error) {
			var names []string
			for _, metadata := range databaseMetadatas {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}
