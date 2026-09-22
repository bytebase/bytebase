package plsql

import (
	"context"
	"io"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	oracleast "github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
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
		// Metadata is the protojson encoded metadatapb.DatabaseSchemaMetadata,
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
		metadata := &metadatapb.DatabaseSchemaMetadata{}
		a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), metadata))
		list := []*metadatapb.DatabaseSchemaMetadata{metadata}
		crossDatabase := &metadatapb.DatabaseSchemaMetadata{}
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
		}, base.Statement{Text: tc.Statement}, tc.DefaultDatabase, "", false)
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
		yamltest.Record(t, testDataPath, testCases)
	}
}

func buildMockDatabaseMetadataGetter(defaultMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc, base.GetLinkedDatabaseMetadataFunc) {
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
		}, func(_ context.Context, _, linkedDatabaseName, schemaName string) (string, string, *model.DatabaseMetadata, error) {
			databaseMetadata := defaultMetadata
			var linkedDBInfo *metadatapb.LinkedDatabaseMetadata
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
				// Production resolves an unknown link to nothing, not to an error.
				return "", "", nil, nil
			}

			// As in production, the written schema wins over the link user, and a link resolves to
			// nothing when no database of that name exists. LOOPBACK points back at the connected
			// instance; every other link reaches instance B.
			databaseName := linkedDBInfo.Username
			if schemaName != "" {
				databaseName = schemaName
			}
			candidates, instanceID := getLinkedDatabaseMetadata(), instanceIDB
			if linkedDBInfo.Name == "LOOPBACK" {
				candidates, instanceID = defaultMetadata, instanceIDA
			}
			for _, metadata := range candidates {
				if metadata.Name == databaseName {
					return instanceID, metadata.Name, model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */), nil
				}
			}

			return "", "", nil, nil
		}
}

func listLinkedDatabaseNames() ([]string, error) {
	return []string{"SCHEMA1", "SCHEMA2"}, nil
}

func getLinkedDatabaseMetadata() []*metadatapb.DatabaseSchemaMetadata {
	return []*metadatapb.DatabaseSchemaMetadata{
		{
			Name: "SCHEMA1",
			Schemas: []*metadatapb.SchemaMetadata{
				{
					Name: "",
					Tables: []*metadatapb.TableMetadata{
						{
							Name: "LT1",
							Columns: []*metadatapb.ColumnMetadata{
								{
									Name: "LC1",
									Type: "int",
								},
							},
						},
						{
							Name: "LT2",
							Columns: []*metadatapb.ColumnMetadata{
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
					Views: []*metadatapb.ViewMetadata{
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

func TestGetQuerySpanRefusesSystemTableWithLink(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name:            "PUBLIC",
		Schemas:         []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{{Name: "T", Columns: []*metadatapb.ColumnMetadata{{Name: "A"}}}}}},
		LinkedDatabases: []*metadatapb.LinkedDatabaseMetadata{{Name: "REMOTE", Username: "SCHEMA1"}},
	}
	databaseMetadataGetter, databaseNamesLister, linkedDatabaseMetadataGetter := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		InstanceID:                    instanceIDA,
		GetDatabaseMetadataFunc:       databaseMetadataGetter,
		ListDatabaseNamesFunc:         databaseNamesLister,
		GetLinkedDatabaseMetadataFunc: linkedDatabaseMetadataGetter,
	}
	// A linked table is a user table: next to a local system table the statement is mixed,
	// and on its own it is never an info-schema read.
	_, err := GetQuerySpan(context.TODO(), gCtx, base.Statement{Text: "SELECT * FROM SYS.DBA_USERS, T@REMOTE;"}, "PUBLIC", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)
	span, err := GetQuerySpan(context.TODO(), gCtx, base.Statement{Text: "SELECT * FROM SYS.DBA_USERS@REMOTE;"}, "PUBLIC", "", false)
	require.NoError(t, err)
	require.Equal(t, base.Select, span.Type)
}

func TestGetAccessTables(t *testing.T) {
	tests := []struct {
		statement string
		expected  []base.SchemaResource
		linked    []base.SchemaResource
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
		{
			statement: "SELECT * FROM schema1.t1@remote;",
			linked: []base.SchemaResource{
				{
					Database:     "SCHEMA1",
					Table:        "T1",
					LinkedServer: "REMOTE",
				},
			},
		},
		{
			statement: "SELECT * FROM t1@remote;",
			linked: []base.SchemaResource{
				{
					Table:        "T1",
					LinkedServer: "REMOTE",
				},
			},
		},
		{
			statement: "SELECT * FROM t1, schema1.t2@remote WHERE EXISTS (SELECT 1 FROM t3@remote);",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Table:    "T1",
				},
			},
			linked: []base.SchemaResource{
				{
					Database:     "SCHEMA1",
					Table:        "T2",
					LinkedServer: "REMOTE",
				},
				{
					Table:        "T3",
					LinkedServer: "REMOTE",
				},
			},
		},
		{
			statement: "SELECT * FROM sys.dba_users@remote;",
			linked: []base.SchemaResource{
				{
					Database:     "SYS",
					Table:        "DBA_USERS",
					LinkedServer: "REMOTE",
				},
			},
		},
		{
			statement: "SELECT sysdate FROM dual;",
		},
		// Through a link an unqualified DUAL resolves in the link user's schema first.
		{
			statement: "SELECT sysdate FROM dual@remote;",
			linked: []base.SchemaResource{
				{
					Table:        "DUAL",
					LinkedServer: "REMOTE",
				},
			},
		},
		{
			statement: "SELECT * FROM t1, sys.dual;",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Table:    "T1",
				},
			},
		},
		{
			statement: "SELECT sysdate FROM sys.dual@remote;",
		},
		// Oracle accepts the public synonym's owner only quoted (ORA-00903 unquoted).
		{
			statement: `SELECT sysdate FROM "PUBLIC".dual;`,
		},
		// A schema-qualified DUAL is a table in that schema, local or remote.
		{
			statement: "SELECT * FROM s.dual@remote;",
			linked: []base.SchemaResource{
				{
					Database:     "S",
					Table:        "DUAL",
					LinkedServer: "REMOTE",
				},
			},
		},
		{
			statement: "SELECT * FROM app.dual;",
			expected: []base.SchemaResource{
				{
					Database: "APP",
					Table:    "DUAL",
				},
			},
		},
		// Oracle rejects a link after a partition clause (ORA-03048), so it names none.
		{
			statement: "SELECT * FROM t1 PARTITION (p1)@remote;",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Table:    "T1",
				},
			},
		},
		// `t@remote@q` names the link REMOTE@Q.
		{
			statement: "SELECT * FROM s.t@remote@q;",
			linked: []base.SchemaResource{
				{
					Database:     "S",
					Table:        "T",
					LinkedServer: "REMOTE@Q",
				},
			},
		},
		{
			statement: "SELECT * FROM t1@remote@q, t1@remote;",
			linked: []base.SchemaResource{
				{
					Table:        "T1",
					LinkedServer: "REMOTE@Q",
				},
				{
					Table:        "T1",
					LinkedServer: "REMOTE",
				},
			},
		},
	}

	for _, test := range tests {
		results, err := ParsePLSQL(test.statement)
		require.NoError(t, err)
		require.NotEmpty(t, results)
		require.NotEmpty(t, results.Items)
		raw, ok := results.Items[0].(*oracleast.RawStmt)
		require.True(t, ok)
		local, linked := collectOmniAccessTables("DB", raw.Stmt)
		require.Equal(t, test.expected, local, test.statement)
		require.Equal(t, test.linked, linked, test.statement)
	}
}
