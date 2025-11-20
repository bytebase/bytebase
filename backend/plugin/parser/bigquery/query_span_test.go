package bigquery

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	parser "github.com/bytebase/parser/googlesql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetQuerySpan(t *testing.T) {
	type testCase struct {
		Description        string `yaml:"description,omitempty"`
		Statement          string `yaml:"statement,omitempty"`
		DefaultDatabase    string `yaml:"defaultDatabase,omitempty"`
		IgnoreCaseSensitve bool   `yaml:"ignoreCaseSensitive,omitempty"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  string              `yaml:"metadata,omitempty"`
		QuerySpan *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	var (
		record        = false
		testDataPaths = []string{
			"test-data/query-span/standard.yaml",
		}
	)

	a := require.New(t)
	for _, testDataPath := range testDataPaths {
		testDataPath := testDataPath

		yamlFile, err := os.Open(testDataPath)
		a.NoError(err)

		var testCases []testCase
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(err)
		a.NoError(yamlFile.Close())
		a.NoError(yaml.Unmarshal(byteValue, &testCases))

		for i, tc := range testCases {
			metadata := &storepb.DatabaseSchemaMetadata{}
			a.NoErrorf(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), metadata), "cases %d", i+1)
			databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
			result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: databaseMetadataGetter,
				ListDatabaseNamesFunc:   databaseNameLister,
			}, tc.Statement, tc.DefaultDatabase, "dbo", tc.IgnoreCaseSensitve)
			a.NoErrorf(err, "statement: %s", tc.Statement)
			resultYaml := result.ToYaml()
			if record {
				testCases[i].QuerySpan = resultYaml
			} else {
				a.Equalf(tc.QuerySpan, resultYaml, "statement: %s", tc.Statement)
			}
		}

		if record {
			byteValue, err := yaml.Marshal(testCases)
			a.NoError(err)
			err = os.WriteFile(testDataPath, byteValue, 0644)
			a.NoError(err)
		}
	}
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*storepb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_BIGQUERY, true /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return "", databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			var names []string
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}

func TestGetPossibleColumnResources(t *testing.T) {
	testCases := []struct {
		inputExpr string
		want      [][]string
	}{
		// Skip the subquery.
		{
			inputExpr: "(SELECT a FROM t)",
			want:      nil,
		},
		{
			inputExpr: "a",
			want:      [][]string{{"a"}},
		},
		{
			inputExpr: "a.b",
			want:      [][]string{{"a", "b"}},
		},
		{
			inputExpr: "function_return_json(a.b).c.d",
			want:      [][]string{{"a", "b"}},
		},
		{
			inputExpr: "a.b + c.d",
			want:      [][]string{{"c", "d"}, {"a", "b"}},
		},
	}

	for _, tc := range testCases {
		inputStream := antlr.NewInputStream(tc.inputExpr)
		lexer := parser.NewGoogleSQLLexer(inputStream)
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		p := parser.NewGoogleSQLParser(stream)
		// Remove default error listener and add our own error listener.
		lexer.RemoveErrorListeners()
		lexerErrorListener := &base.ParseErrorListener{
			Statement: tc.inputExpr,
		}
		lexer.AddErrorListener(lexerErrorListener)

		p.RemoveErrorListeners()
		parserErrorListener := &base.ParseErrorListener{
			Statement: tc.inputExpr,
		}
		p.AddErrorListener(parserErrorListener)

		p.BuildParseTrees = true

		tree := p.Expression()
		require.Nil(t, lexerErrorListener.Err, "input: %s", tc.inputExpr)
		require.Nil(t, parserErrorListener.Err, "input: %s", tc.inputExpr)
		got := getPossibleColumnResources(tree)
		require.Equal(t, tc.want, got, "input: %s", tc.inputExpr)
	}
}
