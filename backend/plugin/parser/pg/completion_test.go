package pg

import (
	"context"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type candidatesTest struct {
	Input string
	Want  []base.Candidate
}

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
		text, caretOffset := catchCaret(t.Input)
		result, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, base.CompletionContext{
			Scene:             base.SceneTypeAll,
			DefaultDatabase:   "db",
			Metadata:          getMetadataForTest,
			ListDatabaseNames: listDatbaseNamesForTest,
		}, text, 1, caretOffset)
		a.NoError(err)
		var filteredResult []base.Candidate
		for _, r := range result {
			switch r.Type {
			case base.CandidateTypeKeyword, base.CandidateTypeFunction:
				continue
			default:
				filteredResult = append(filteredResult, r)
			}
		}
		slices.SortFunc(filteredResult, func(x, y base.Candidate) int {
			if x.Type != y.Type {
				if x.Type < y.Type {
					return -1
				}
				return 1
			}
			if x.Text != y.Text {
				if x.Text < y.Text {
					return -1
				}
				return 1
			}
			if x.Definition < y.Definition {
				return -1
			} else if x.Definition > y.Definition {
				return 1
			}
			return 0
		})

		if record {
			tests[i].Want = filteredResult
		} else {
			a.Equal(t.Want, filteredResult, t.Input)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

func listDatbaseNamesForTest(_ context.Context, _ string) ([]string, error) {
	return []string{"db"}, nil
}

func getMetadataForTest(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
	if databaseName != "db" {
		return "", nil, nil
	}

	return "db", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
								Type: "int",
							},
						},
					},
					{
						Name: "t2",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
								Type: "int",
							},
							{
								Name: "c2",
								Type: "int",
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name: "v1",
						Definition: `CREATE VIEW v1 AS
						SELECT *
						FROM t1
						`,
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name: "seq1",
					},
					{
						Name: "user_id_seq",
					},
				},
			},
			{
				Name: "test",
				Tables: []*storepb.TableMetadata{
					{
						Name: "auto",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "name",
								Type: "varchar",
							},
						},
					},
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "user_id",
								Type: "int",
							},
							{
								Name: "username",
								Type: "varchar",
							},
						},
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name: "order_id_seq",
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true /* isObjectCaseSensitive */), nil
}

func catchCaret(s string) (string, int) {
	for i, c := range s {
		if c == '|' {
			return s[:i] + s[i+1:], i
		}
	}
	return s, -1
}
