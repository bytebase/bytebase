package mongodb

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

	for i, tc := range tests {
		text, caretOffset := catchCaret(tc.Input)
		result, err := base.Completion(context.Background(), storepb.Engine_MONGODB, base.CompletionContext{
			Scene:           base.SceneTypeAll,
			DefaultDatabase: "test",
			Metadata:        getMetadataForTest,
		}, text, 1, caretOffset)
		a.NoError(err)

		// Sort results for consistent comparison
		slices.SortFunc(result, func(x, y base.Candidate) int {
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
			return 0
		})

		if record {
			tests[i].Want = result
		} else {
			a.Equal(tc.Want, result, tc.Input)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

func getMetadataForTest(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
	if databaseName != "test" {
		return "", nil, nil
	}

	return "test", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
					{Name: "orders"},
					{Name: "my-collection"}, // special char - needs bracket notation
				},
			},
		},
	}, nil, nil, storepb.Engine_MONGODB, true /* isObjectCaseSensitive */), nil
}

func catchCaret(s string) (string, int) {
	for i, c := range s {
		if c == '|' {
			return s[:i] + s[i+1:], i
		}
	}
	return s, -1
}
