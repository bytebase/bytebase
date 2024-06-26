package tsql

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type candidatesTest struct {
	Description string           `yaml:"description"`
	Input       string           `yaml:"input"`
	Want        []base.Candidate `yaml:"want"`
}

// TestCompletion tests the Transact-SQL auto-completion, all the test cases are stored in the file.
//
// - Description: The description of the test case.
//
// - Input: The input statement with the caret position marked by "|".
//
// - Want: The expected completion candidates.
//
// Our Test suite will determine the caret position - line(0-based) and column(1-based) by the position of the "|", actually,
// this caret position is as same as the position in the monaco-editor(LSP?).
func TestCompletion(t *testing.T) {
	tests := []candidatesTest{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_completion.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		statement, caretLine, caretPosition := getCaretPosition(t.Input)
		getter, lister := buildMockDatabaseMetadataGetterLister()
		results, err := Completion(context.Background(), base.CompletionContext{
			Scene:             base.SceneTypeAll,
			DefaultDatabase:   "Company",
			Metadata:          getter,
			ListDatabaseNames: lister,
		}, statement, caretLine, caretPosition)
		a.NoErrorf(err, "Case %02d: %s", i, t.Description)
		var filteredResult []base.Candidate
		for _, r := range results {
			switch r.Type {
			case base.CandidateTypeKeyword, base.CandidateTypeFunction:
				continue
			default:
				filteredResult = append(filteredResult, r)
			}
		}

		if record {
			tests[i].Want = filteredResult
		} else {
			a.Equalf(t.Want, filteredResult, t.Input, "Case %02d: %s", i, t.Description)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
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

var databaseMetadatas = []*storepb.DatabaseSchemaMetadata{
	{
		Name: "Company",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*storepb.TableMetadata{
					{
						Name: "Employees",
						Columns: []*storepb.ColumnMetadata{
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
						Columns: []*storepb.ColumnMetadata{
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
			},
			{
				Name: "MySchema",
				Tables: []*storepb.TableMetadata{
					{
						Name: "SalaryLevel",
						Columns: []*storepb.ColumnMetadata{
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
			},
		},
	},
	{
		Name: "School",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*storepb.TableMetadata{
					{
						Name: "Student",
						Columns: []*storepb.ColumnMetadata{
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
			},
		},
	},
}

func buildMockDatabaseMetadataGetterLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadatas {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return "", databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(context.Context) ([]string, error) {
			var names []string
			for _, metadata := range databaseMetadatas {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}
