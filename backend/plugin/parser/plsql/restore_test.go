package plsql

import (
	"context"
	"io"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

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
		yamltest.Record(t, filepath, tests)
	}
}

func TestRestoreOmniBoundaryCases(t *testing.T) {
	t.Run("multi-line update", func(t *testing.T) {
		input := `UPDATE test
SET c1 = 1
WHERE c1 = 1;
UPDATE test
SET c1 = 2
WHERE c1 = 2;`

		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, input, restoreBackupItem(0, math.MaxInt32))
		require.NoError(t, err)
		require.Equal(t, `/*
Original SQL:
UPDATE test
SET c1 = 1
WHERE c1 = 1
UPDATE test
SET c1 = 2
WHERE c1 = 2
*/
MERGE INTO "DB"."TEST" t
USING "bbarchive"."prefix_1_test" b
  ON( t."A" = b."A")
WHEN MATCHED THEN
  UPDATE SET t."C1" = b."C1"
WHEN NOT MATCHED THEN
 INSERT ("A", "B", "C") VALUES (b."A", b."B", b."C");`, result)
	})

	t.Run("position range selects update", func(t *testing.T) {
		input := `DELETE FROM test WHERE c1 = 1;
UPDATE test SET c1 = 2 WHERE c1 = 2;`

		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, input, restoreBackupItem(2, 2))
		require.NoError(t, err)
		require.Contains(t, result, "UPDATE test SET c1 = 2 WHERE c1 = 2")
		require.NotContains(t, result, "DELETE FROM test WHERE c1 = 1")
		require.Contains(t, result, `UPDATE SET t."C1" = b."C1"`)
	})

	t.Run("same-line multi statement", func(t *testing.T) {
		input := `UPDATE test SET c1 = 1 WHERE c1 = 1; UPDATE test SET c1 = 2 WHERE c1 = 2;`

		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, input, restoreBackupItem(0, math.MaxInt32))
		require.NoError(t, err)
		require.Contains(t, result, "UPDATE test SET c1 = 1 WHERE c1 = 1\nUPDATE test SET c1 = 2 WHERE c1 = 2")
		require.Contains(t, result, `UPDATE SET t."C1" = b."C1"`)
	})

	t.Run("range skips non backupable statements", func(t *testing.T) {
		input := `INSERT INTO test (a, b, c) VALUES (1, 1, 1);
UPDATE test SET c1 = 2 WHERE c1 = 2;
CREATE TABLE unrelated (id NUMBER);`

		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, input, restoreBackupItem(0, math.MaxInt32))
		require.NoError(t, err)
		require.Contains(t, result, "UPDATE test SET c1 = 2 WHERE c1 = 2")
		require.NotContains(t, result, "INSERT INTO test")
		require.NotContains(t, result, "CREATE TABLE unrelated")
		require.Contains(t, result, `UPDATE SET t."C1" = b."C1"`)
	})

	t.Run("multi-line delete", func(t *testing.T) {
		input := `DELETE FROM test
WHERE c1 = 1;`

		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, input, restoreBackupItem(0, math.MaxInt32))
		require.NoError(t, err)
		require.Equal(t, `/*
Original SQL:
DELETE FROM test
WHERE c1 = 1
*/
INSERT INTO "DB"."TEST" ("A", "B", "C") SELECT "A", "B", "C" FROM "bbarchive"."prefix_1_test";`, result)
	})

	t.Run("no disjoint unique key", func(t *testing.T) {
		input := `UPDATE test SET a = 1 WHERE c1 = 1;`

		_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, input, restoreBackupItem(0, math.MaxInt32))
		require.Error(t, err)
		require.Contains(t, err.Error(), "no disjoint unique key found")
	})
}

func restoreBackupItem(startLine, endLine int32) *store.PriorBackupDetail_Item {
	return &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/DB",
			Table:    "TEST",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{
			Line:   startLine,
			Column: 0,
		},
		EndPosition: &store.Position{
			Line:   endLine,
			Column: math.MaxInt32,
		},
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
