package mysql

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
		result, err := base.Completion(context.Background(), storepb.Engine_MYSQL, text, 1, caretOffset, "db", getMetadataForTest)
		a.NoError(err)
		var filteredResult []base.Candidate
		for _, r := range result {
			switch r.Type {
			case base.CandidateTypeKeyword:
				continue
			default:
				filteredResult = append(filteredResult, r)
			}
		}
		if record {
			tests[i].Want = filteredResult
		} else {
			a.Equal(t.Want, filteredResult)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

func getMetadataForTest(_ context.Context, databaseName string) (*model.DatabaseMetadata, error) {
	if databaseName != "db" {
		return nil, nil
	}

	return model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
							},
						},
					},
					{
						Name: "t2",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
							},
							{
								Name: "c2",
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
			},
		},
	}), nil
}

func catchCaret(s string) (string, int) {
	for i, c := range s {
		if c == '|' {
			return s[:i] + s[i+1:], i
		}
	}
	return s, -1
}
