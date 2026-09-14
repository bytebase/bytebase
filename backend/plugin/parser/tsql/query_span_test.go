package tsql

import (
	"context"
	"io"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
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
		// Metadata is the protojson encoded metadatapb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  []string            `yaml:"metadata,omitempty"`
		QuerySpan *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	var (
		record        = false
		testDataPaths = []string{
			"test-data/query-span/standard.yaml",
			"test-data/query-span/join.yaml",
			"test-data/query-span/case-sensitivity.yaml",
			"test-data/query-span/query_type.yaml",
			"test-data/query-span/predicate.yaml",
			"test-data/query-span/regression.yaml",
		}
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
			var ms []*metadatapb.DatabaseSchemaMetadata
			for _, metadata := range tc.Metadata {
				storepbMetadata := &metadatapb.DatabaseSchemaMetadata{}
				a.NoErrorf(common.ProtojsonUnmarshaler.Unmarshal([]byte(metadata), storepbMetadata), "cases %d", i+1)
				ms = append(ms, storepbMetadata)
			}
			databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter(ms)
			result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: databaseMetadataGetter,
				ListDatabaseNamesFunc:   databaseNameLister,
				TempTables:              make(map[string]*base.PhysicalTable),
			}, base.Statement{Text: tc.Statement}, tc.DefaultDatabase, "dbo", tc.IgnoreCaseSensitve)
			a.NoErrorf(err, "statement: %s", tc.Statement)
			resultYaml := result.ToYaml()
			if record {
				testCases[i].QuerySpan = resultYaml
			} else {
				a.Equalf(tc.QuerySpan, resultYaml, "statement: %s", tc.Statement)
			}
		}

		if record {
			yamltest.Record(t, testDataPath, testCases)
		}
	}
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MSSQL, false /* isObjectCaseSensitive */)
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

func TestGetQuerySpanCyclicViewReference(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Views: []*metadatapb.ViewMetadata{
					{Name: "v1", Definition: "CREATE VIEW [dbo].[v1] AS SELECT * FROM v2"},
					{Name: "v2", Definition: "CREATE VIEW [dbo].[v2] AS SELECT * FROM v1"},
				},
			},
		},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})

	_, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		TempTables:              make(map[string]*base.PhysicalTable),
	}, base.Statement{Text: "SELECT * FROM v1"}, "db", "dbo", true)
	require.Error(t, err)
	require.ErrorContains(t, err, "cyclic view reference")
}
