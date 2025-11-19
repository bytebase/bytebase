package plsql

import (
	"context"
	"io"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type restoreCase struct {
	Input            string
	BackupDatabase   string
	BackupTable      string
	OriginalDatabase string
	OriginalTable    string
	Result           string
}

func TestRestore(t *testing.T) {
	tests := []restoreCase{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_restore.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, t.Input, &store.PriorBackupDetail_Item{
			SourceTable: &store.PriorBackupDetail_Item_Table{
				Database: "instances/i1/databases/" + t.OriginalDatabase,
				Table:    t.OriginalTable,
			},
			TargetTable: &store.PriorBackupDetail_Item_Table{
				Database: "instances/i1/databases/" + t.BackupDatabase,
				Table:    t.BackupTable,
			},
			StartPosition: &store.Position{
				Line:   0,
				Column: 0,
			},
			EndPosition: &store.Position{
				Line:   math.MaxInt32,
				Column: 0,
			},
		})
		a.NoError(err)

		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result, t.Input)
		}
	}
	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

func fixedMockDatabaseMetadataGetter(_ context.Context, _ string, database string) (string, *model.DatabaseMetadata, error) {
	return database, model.NewDatabaseMetadata(&store.DatabaseSchemaMetadata{
		Name: database,
		Schemas: []*store.SchemaMetadata{
			{
				Name: "",
				Tables: []*store.TableMetadata{
					{
						Name: "T_GENERATED",
						Columns: []*store.ColumnMetadata{
							{
								Name: "A",
							},
							{
								Name: "B",
							},
						},
						Indexes: []*store.IndexMetadata{
							{
								Name:        "T_GENERATED_PK",
								Expressions: []string{"B"},
								Primary:     true,
								Unique:      true,
							},
							{
								Name:        "T_GENERATED_UK",
								Expressions: []string{"A"},
								Unique:      true,
							},
						},
					},
					{
						Name: "T1",
						Columns: []*store.ColumnMetadata{
							{
								Name: "A",
							},
							{
								Name: "B",
							},
							{
								Name: "C",
							},
						},
					},
					{
						Name: "T2",
						Columns: []*store.ColumnMetadata{
							{
								Name: "A",
							},
							{
								Name: "B",
							},
							{
								Name: "C",
							},
						},
					},
					{
						Name: "TEST",
						Columns: []*store.ColumnMetadata{
							{
								Name: "A",
							},
							{
								Name: "B",
							},
							{
								Name: "C",
							},
						},
						Indexes: []*store.IndexMetadata{
							{
								Name:        "TEST_PK",
								Expressions: []string{"A"},
								Primary:     true,
								Unique:      true,
							},
						},
					},
					{
						Name: "TEST2",
						Columns: []*store.ColumnMetadata{
							{
								Name: "A",
							},
							{
								Name: "B",
							},
							{
								Name: "C",
							},
						},
					},
				},
			},
		},
	}, nil, nil, store.Engine_ORACLE, true /* isObjectCaseSensitive */), nil
}
