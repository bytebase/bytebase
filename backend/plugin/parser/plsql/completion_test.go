package plsql

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
		result, err := base.Completion(context.Background(), storepb.Engine_ORACLE, base.CompletionContext{
			Scene:             base.SceneTypeAll,
			DefaultDatabase:   "SCHEMA1",
			Metadata:          getMetadataForTest,
			ListDatabaseNames: listDatabaseNamesForTest,
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

func listDatabaseNamesForTest(_ context.Context) ([]string, error) {
	return []string{"SCHEMA1", "SCHEMA2", "SCHEMA3"}, nil
}

func getMetadataForTest(_ context.Context, databaseName string) (string, *model.DatabaseMetadata, error) {
	switch databaseName {
	case "SCHEMA1":
		return "SCHEMA1", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: databaseName,
					Tables: []*storepb.TableMetadata{
						{
							Name: "T1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
							},
						},
						{
							Name: "T2",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
								{
									Name: "C2",
									Type: "int",
								},
							},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name: "V1",
							Definition: `CREATE VIEW v1 AS
											SELECT *
											FROM t1
							`,
						},
					},
				},
			},
		}), nil
	case "SCHEMA2":
		return "SCHEMA2", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: databaseName,
					Tables: []*storepb.TableMetadata{
						{
							Name: "T1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
							},
						},
						{
							Name: "T2",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
								{
									Name: "C2",
									Type: "int",
								},
							},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name: "V1",
							Definition: `CREATE VIEW v1 AS
											SELECT *
											FROM t1
							`,
						},
					},
				},
			},
		}), nil
	case "SCHEMA3":
		return "SCHEMA3", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: databaseName,
					Tables: []*storepb.TableMetadata{
						{
							Name: "T1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
							},
						},
						{
							Name: "T2",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
								{
									Name: "C2",
									Type: "int",
								},
							},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name: "V1",
							Definition: `CREATE VIEW v1 AS
											SELECT *
											FROM t1
							`,
						},
					},
				},
			},
		}), nil
	default:
		return "", nil, nil
	}
}

func catchCaret(s string) (string, int) {
	for i, c := range s {
		if c == '|' {
			return s[:i] + s[i+1:], i
		}
	}
	return s, -1
}
