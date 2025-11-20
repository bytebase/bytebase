package doris

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetQuerySpan_SystemTable(t *testing.T) {
	type testCase struct {
		Description     string `yaml:"description,omitempty"`
		Statement       string `yaml:"statement,omitempty"`
		DefaultDatabase string `yaml:"defaultDatabase,omitempty"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  string              `yaml:"metadata,omitempty"`
		QuerySpan *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	const (
		record = false
	)

	var (
		testDataPaths = []string{"test-data/query_span_system_table.yaml"}
	)

	a := require.New(t)
	for _, testDataPath := range testDataPaths {
		yamlFile, err := os.Open(testDataPath)
		a.NoError(err)

		var testCases []testCase
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(err)
		a.NoError(yamlFile.Close())
		a.NoError(yaml.Unmarshal(byteValue, &testCases))

		for i, tc := range testCases {
			metadata := &storepb.DatabaseSchemaMetadata{}
			a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), metadata))
			databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
			result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: databaseMetadataGetter,
				ListDatabaseNamesFunc:   databaseNameLister,
			}, tc.Statement, tc.DefaultDatabase, "", false)
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
				// Doris is case-insensitive for most objects
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, false /* isObjectCaseSensitive */)
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
