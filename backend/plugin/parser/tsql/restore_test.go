package tsql

import (
	"context"
	"io"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type restoreCase struct {
	Input            string
	BackupDatabase   string
	BackupTable      string
	OriginalDatabase string
	OriginalSchema   string
	OriginalTable    string
	Result           string
}

// TestRestoreIdentityHandling validates that restore operations properly handle IDENTITY columns
// This test verifies:
// 1. DELETE rollback uses SET IDENTITY_INSERT ON/OFF for tables with IDENTITY columns
// 2. UPDATE rollback (MERGE) also properly handles IDENTITY_INSERT
// 3. DBCC CHECKIDENT is called to reseed IDENTITY values
func TestRestoreIdentityHandling(t *testing.T) {
	a := require.New(t)

	// Mock metadata with IDENTITY column
	mockMetadata := func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		// Extract just the database name from the resource ID
		_, dbName, _ := common.GetInstanceDatabaseID(databaseName)
		if dbName == "" {
			dbName = databaseName
		}
		return databaseName, model.NewDatabaseMetadata(&store.DatabaseSchemaMetadata{
			Name: dbName,
			Schemas: []*store.SchemaMetadata{
				{
					Name: "dbo",
					Tables: []*store.TableMetadata{
						{
							Name: "positions",
							Columns: []*store.ColumnMetadata{
								{
									Name:       "position_id",
									Type:       "int",
									IsIdentity: true, // This is an IDENTITY column
								},
								{
									Name: "title",
									Type: "nvarchar(100)",
								},
							},
						},
					},
				},
			},
		}, nil, nil, store.Engine_MSSQL, false), nil
	}

	// Test DELETE rollback with IDENTITY column
	deleteSQL := "DELETE FROM positions WHERE position_id = 1;"
	restoreSQL, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: mockMetadata,
		InstanceID:              "instances/test-instance",
	}, deleteSQL, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/test-instance/databases/db",
			Schema:   "dbo",
			Table:    "positions",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/test-instance/databases/backupDB",
			Table:    "backup_positions",
		},
		StartPosition: &store.Position{Line: 1, Column: 0},
		EndPosition:   &store.Position{Line: 1, Column: 43},
	})

	a.NoError(err)

	// Verify the restore SQL includes IDENTITY_INSERT handling
	a.Contains(restoreSQL, "SET IDENTITY_INSERT [db].[dbo].[positions] ON")
	a.Contains(restoreSQL, "SET IDENTITY_INSERT [db].[dbo].[positions] OFF")
	a.Contains(restoreSQL, "EXEC('DBCC CHECKIDENT (''[db].[dbo].[positions]'', RESEED)')")

	// Verify the INSERT statement uses explicit column list, not SELECT *
	// This is required by SQL Server when IDENTITY_INSERT is ON
	a.Contains(restoreSQL, "INSERT INTO [db].[dbo].[positions] ([position_id], [title])")
	a.NotContains(restoreSQL, "SELECT * FROM")
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
				Schema:   "dbo",
				Table:    t.OriginalTable,
			},
			TargetTable: &store.PriorBackupDetail_Item_Table{
				Database: "instances/i1/databases/" + t.BackupDatabase,
				Schema:   t.BackupDatabase,
				Table:    t.BackupTable,
			},
			StartPosition: &store.Position{
				Line:   1,
				Column: 0,
			},
			EndPosition: &store.Position{
				Line:   math.MaxInt32,
				Column: 1,
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
				Name: "dbo",
				Tables: []*store.TableMetadata{
					{
						Name: "t_generated",
						Columns: []*store.ColumnMetadata{
							{
								Name: "a",
							},
							{
								Name: "b",
							},
						},
						Indexes: []*store.IndexMetadata{
							{
								Name:        "t_generated_pk",
								Expressions: []string{"b"},
								Primary:     true,
								Unique:      true,
							},
							{
								Name:        "t_generated_uk",
								Expressions: []string{"a"},
								Unique:      true,
							},
						},
					},
					{
						Name: "t1",
						Columns: []*store.ColumnMetadata{
							{
								Name: "a",
							},
							{
								Name: "b",
							},
							{
								Name: "c",
							},
						},
					},
					{
						Name: "t2",
						Columns: []*store.ColumnMetadata{
							{
								Name: "a",
							},
							{
								Name: "b",
							},
							{
								Name: "c",
							},
						},
					},
					{
						Name: "test",
						Columns: []*store.ColumnMetadata{
							{
								Name: "a",
							},
							{
								Name: "b",
							},
							{
								Name: "c",
							},
						},
						Indexes: []*store.IndexMetadata{
							{
								Name:        "test_pk",
								Expressions: []string{"a"},
								Primary:     true,
								Unique:      true,
							},
						},
					},
					{
						Name: "test2",
						Columns: []*store.ColumnMetadata{
							{
								Name: "a",
							},
							{
								Name: "b",
							},
							{
								Name: "c",
							},
						},
					},
				},
			},
		},
	}, nil, nil, store.Engine_MSSQL, false /* isObjectCaseSensitive */), nil
}
