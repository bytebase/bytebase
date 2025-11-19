package mysql

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

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
			"test-data/query-span/query_type.yaml",
			"test-data/query-span/standard.yaml",
			"test-data/query-span/case_insensitive.yaml",
			"test-data/query-span/starrocks.yaml",
		}
	)

	a := require.New(t)
	for _, testDataPath := range testDataPaths {
		engine := storepb.Engine_MYSQL
		if strings.Contains(testDataPath, "starrocks") {
			engine = storepb.Engine_STARROCKS
		}

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
				Engine:                  engine,
			}, tc.Statement, tc.DefaultDatabase, "", tc.IgnoreCaseSensitve)
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
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true /* isObjectCaseSensitive */)
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

func TestExtractTableRefs(t *testing.T) {
	tests := []struct {
		statement string
		expected  []base.SchemaResource
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1;",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
			},
		},
		{
			statement: "SELECT * FROM db1.t1 JOIN db2.t2 ON t1.c1 = t2.c1;",
			expected: []base.SchemaResource{
				{
					Database: "db1",
					Table:    "t1",
				},
				{
					Database: "db2",
					Table:    "t2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
				{
					Database: "db",
					Table:    "t2",
				},
			},
		},
	}

	for _, test := range tests {
		parseResult, err := ParseMySQL(test.statement)
		require.NoError(t, err, "failed to parse statement: %s", test.statement)
		require.Len(t, parseResult, 1, "expected one parse result for statement: %s", test.statement)
		require.NotNil(t, parseResult[0].Tree, "parse tree is nil for statement: %s", test.statement)

		tree, ok := parseResult[0].Tree.(antlr.ParserRuleContext)
		require.True(t, ok, "expected parse tree to be of type antlr.RuleContext for statement: %s", test.statement)

		resources := extractTableRefs("db", tree)
		require.Equal(t, test.expected, resources, test.statement)
	}
}
