package plsql

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

const (
	instanceIDA = "INSTANCE_ID_A"
	instanceIDB = "INSTANCE_ID_B"
)

func TestGetQuerySpan(t *testing.T) {
	type testCase struct {
		Description     string `yaml:"description,omitempty"`
		Statement       string `yaml:"statement,omitempty"`
		DefaultDatabase string `yaml:"defaultDatabase,omitempty"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata              string              `yaml:"metadata,omitempty"`
		CrossDatabaseMetadata string              `yaml:"crossDatabaseMetadata,omitempty"`
		QuerySpan             *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	const (
		record       = false
		testDataPath = "test-data/query_span.yaml"
	)

	a := require.New(t)
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
		list := []*storepb.DatabaseSchemaMetadata{metadata}
		crossDatabase := &storepb.DatabaseSchemaMetadata{}
		if tc.CrossDatabaseMetadata != "" {
			a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.CrossDatabaseMetadata), crossDatabase))
			list = append(list, crossDatabase)
		}
		databaseMetadataGetter, databaseNamesLister, linkedDatabaseMetadataGetter := buildMockDatabaseMetadataGetter(list)
		result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
			InstanceID:                    instanceIDA,
			GetDatabaseMetadataFunc:       databaseMetadataGetter,
			ListDatabaseNamesFunc:         databaseNamesLister,
			GetLinkedDatabaseMetadataFunc: linkedDatabaseMetadataGetter,
		}, tc.Statement, tc.DefaultDatabase, "", false)
		a.NoError(err)
		a.NotNil(result)
		resultYaml := result.ToYaml()
		if record {
			testCases[i].QuerySpan = resultYaml
		} else {
			a.Equal(tc.QuerySpan, resultYaml, "statement: %s", tc.Statement)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(testDataPath, byteValue, 0644)
		a.NoError(err)
	}
}

func buildMockDatabaseMetadataGetter(defaultMetadata []*storepb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc, base.GetLinkedDatabaseMetadataFunc) {
	return func(_ context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
			databaseMetadata := defaultMetadata
			if instanceID == instanceIDB {
				databaseMetadata = getLinkedDatabaseMetadata()
			}
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return databaseName, databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, instanceID string) ([]string, error) {
			databaseMetadata := defaultMetadata
			if instanceID == instanceIDB {
				return listLinkedDatabaseNames()
			}
			names := make([]string, 0, len(databaseMetadata))
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}, func(_ context.Context, _, linkedDatabaseName, _ string) (string, string, *model.DatabaseMetadata, error) {
			databaseMetadata := defaultMetadata
			var linkedDBInfo *storepb.LinkedDatabaseMetadata
			for _, metadata := range databaseMetadata {
				for _, linkedDatabase := range metadata.GetLinkedDatabases() {
					if linkedDatabase.Name == linkedDatabaseName {
						linkedDBInfo = linkedDatabase
						break
					}
				}
				if linkedDBInfo != nil {
					break
				}
			}
			if linkedDBInfo == nil {
				return "", "", nil, errors.Errorf("linked database %q not found", linkedDatabaseName)
			}

			for _, metadata := range getLinkedDatabaseMetadata() {
				if metadata.Name == linkedDBInfo.Username {
					return instanceIDB, metadata.Name, model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */), nil
				}
			}

			return "", "", nil, errors.Errorf("database %q not found", linkedDBInfo.Username)
		}
}

func listLinkedDatabaseNames() ([]string, error) {
	return []string{"SCHEMA1", "SCHEMA2"}, nil
}

func getLinkedDatabaseMetadata() []*storepb.DatabaseSchemaMetadata {
	return []*storepb.DatabaseSchemaMetadata{
		{
			Name: "SCHEMA1",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "",
					Tables: []*storepb.TableMetadata{
						{
							Name: "LT1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "LC1",
									Type: "int",
								},
							},
						},
						{
							Name: "LT2",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "LC1",
									Type: "int",
								},
								{
									Name: "LC2",
									Type: "int",
								},
							},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name: "LV1",
							Definition: `SELECT LC1, LC2
											FROM LT2
							`,
						},
					},
				},
			},
		},
	}
}

func TestGetAccessTables(t *testing.T) {
	tests := []struct {
		statement string
		expected  []base.SchemaResource
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Table:    "T1",
				},
			},
		},
		{
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1;",
			expected: []base.SchemaResource{
				{
					Database: "SCHEMA1",
					Table:    "T1",
				},
				{
					Database: "SCHEMA2",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Table:    "T1",
				},
				{
					Database: "DB",
					Table:    "T2",
				},
			},
		},
	}

	for _, test := range tests {
		results, err := ParsePLSQL(test.statement)
		require.NoError(t, err)
		require.NotEmpty(t, results)
		resources := getAccessTables("DB", results[0].Tree)
		require.Equal(t, test.expected, resources, test.statement)
	}
}
