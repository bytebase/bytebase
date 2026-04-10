package mysql

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

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
				GetDatabaseMetadataFunc:   databaseMetadataGetter,
				ListDatabaseNamesFunc:     databaseNameLister,
				GetDatabaseDefinitionFunc: mockGetDatabaseDefinition,
				Engine:                    engine,
			}, base.Statement{Text: tc.Statement}, tc.DefaultDatabase, "", tc.IgnoreCaseSensitve)
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

func mockGetDatabaseDefinition(meta *storepb.DatabaseSchemaMetadata) (string, error) {
	if meta == nil {
		return "", nil
	}
	var b strings.Builder
	for _, s := range meta.Schemas {
		for _, t := range s.Tables {
			fmt.Fprintf(&b, "CREATE TABLE `%s`.`%s` (\n", meta.Name, t.Name)
			for i, c := range t.Columns {
				colType := c.Type
				if colType == "" {
					colType = "text"
				}
				fmt.Fprintf(&b, "  `%s` %s", c.Name, colType)
				if i < len(t.Columns)-1 {
					b.WriteString(",")
				}
				b.WriteString("\n")
			}
			b.WriteString(");\n")
		}
		for _, v := range s.Views {
			if v.Definition == "" {
				continue
			}
			def := strings.TrimSuffix(strings.TrimSpace(v.Definition), ";")
			fmt.Fprintf(&b, "CREATE VIEW `%s`.`%s` AS %s;\n", meta.Name, v.Name, def)
		}
	}
	return b.String(), nil
}
