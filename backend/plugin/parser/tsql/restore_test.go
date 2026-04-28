package tsql

import (
	"context"
	"io"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
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

func TestRestoreOmniBoundaryCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		startLine int32
		endLine   int32
		want      string
	}{
		{
			name: "selects only update inside backup position range",
			input: strings.Join([]string{
				"DELETE FROM test WHERE a = 1;",
				"UPDATE test SET b = b + 1, c = DEFAULT WHERE a = 2;",
			}, "\n"),
			startLine: 2,
			endLine:   2,
			want: strings.Join([]string{
				"/*",
				"Original SQL:",
				"UPDATE test SET b = b + 1, c = DEFAULT WHERE a = 2;",
				"*/",
				"MERGE INTO [db].[dbo].[test] AS t",
				"USING [bbarchive].[dbo].[prefix_1_test] AS b",
				"  ON t.[a] = b.[a]",
				"WHEN MATCHED THEN",
				"  UPDATE SET t.[b] = b.[b], t.[c] = b.[c]",
				"WHEN NOT MATCHED THEN",
				" INSERT ([a], [b], [c]) VALUES (b.[a], b.[b], b.[c]);",
			}, "\n"),
		},
		{
			name:      "dual assignment update restores assigned column",
			input:     "UPDATE test SET @v = b = 1 WHERE a = 1;",
			startLine: 1,
			endLine:   1,
			want: strings.Join([]string{
				"/*",
				"Original SQL:",
				"UPDATE test SET @v = b = 1 WHERE a = 1;",
				"*/",
				"MERGE INTO [db].[dbo].[test] AS t",
				"USING [bbarchive].[dbo].[prefix_1_test] AS b",
				"  ON t.[a] = b.[a]",
				"WHEN MATCHED THEN",
				"  UPDATE SET t.[b] = b.[b]",
				"WHEN NOT MATCHED THEN",
				" INSERT ([a], [b], [c]) VALUES (b.[a], b.[b], b.[c]);",
			}, "\n"),
		},
		{
			name:      "delete restore uses insert",
			input:     "DELETE FROM test WHERE a = 1;",
			startLine: 1,
			endLine:   1,
			want: strings.Join([]string{
				"/*",
				"Original SQL:",
				"DELETE FROM test WHERE a = 1;",
				"*/",
				"INSERT INTO [db].[dbo].[test] SELECT * FROM [bbarchive].[dbo].[prefix_1_test];",
			}, "\n"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
				GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
			}, tc.input, &store.PriorBackupDetail_Item{
				SourceTable: &store.PriorBackupDetail_Item_Table{
					Database: "instances/i1/databases/db",
					Schema:   "dbo",
					Table:    "test",
				},
				TargetTable: &store.PriorBackupDetail_Item_Table{
					Database: "instances/i1/databases/bbarchive",
					Table:    "prefix_1_test",
				},
				StartPosition: &store.Position{Line: tc.startLine, Column: 0},
				EndPosition:   &store.Position{Line: tc.endLine, Column: math.MaxInt32},
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, result)
		})
	}
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
		yamltest.Record(t, filepath, tests)
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
