package pg

import (
	"context"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type rollbackCase struct {
	Input  string
	Result []base.BackupStatement
}

func TestBackup(t *testing.T) {
	tests := []rollbackCase{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_backup.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
			DatabaseName:            "",
		}, t.Input, "", "backupSchema", "rollback")
		a.NoError(err)
		slices.SortFunc(result, func(i, j base.BackupStatement) int {
			if i.TargetTableName == j.TargetTableName {
				if i.Statement < j.Statement {
					return -1
				}
				if i.Statement > j.Statement {
					return 1
				}
				return 0
			}
			if i.TargetTableName < j.TargetTableName {
				return -1
			}
			if i.TargetTableName > j.TargetTableName {
				return 1
			}
			return 0
		})

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

func TestBackupUnsyncedDatabase(t *testing.T) {
	_, err := TransformDMLToSelect(context.Background(), base.TransformContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		},
		DatabaseName: "db",
	}, "UPDATE t SET a = 1 WHERE b = 1;", "db", "backupSchema", "rollback")
	require.ErrorContains(t, err, `database metadata for database "db" not found (database not synced)`)
}

func TestBackupRejectsDataModifyingCTE(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantErr   string
	}{
		{
			name:      "delete CTE feeding update",
			statement: "WITH d AS (DELETE FROM test2 WHERE a > 1 RETURNING a) UPDATE test SET c = 1 WHERE a IN (SELECT a FROM d);",
			wantErr:   `data-modifying WITH query "d";`,
		},
		{
			name:      "insert CTE feeding delete",
			statement: "WITH i AS (INSERT INTO test2 (a) VALUES (1) RETURNING a) DELETE FROM test USING i WHERE test.a = i.a;",
			wantErr:   `data-modifying WITH query "i";`,
		},
		{
			name:      "update CTE beside read-only CTE",
			statement: "WITH s AS (SELECT a FROM test2), u AS (UPDATE test2 SET a = a + 1 RETURNING a) UPDATE test SET c = 1 FROM s WHERE test.a = s.a;",
			wantErr:   `data-modifying WITH query "u";`,
		},
		{
			name:      "merge CTE",
			statement: "WITH m AS (MERGE INTO test2 USING test ON test2.a = test.a WHEN MATCHED THEN DELETE RETURNING test2.a) DELETE FROM test WHERE a IN (SELECT a FROM m);",
			wantErr:   `data-modifying WITH query "m";`,
		},
		{
			name:      "rejected after an earlier statement transforms",
			statement: "UPDATE test2 SET a = 1;\nWITH d AS (DELETE FROM test2 RETURNING a) DELETE FROM test WHERE a IN (SELECT a FROM d);",
			wantErr:   `data-modifying WITH query "d";`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
				GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
			}, tc.statement, "", "backupSchema", "rollback")
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, result)
		})
	}
}

func TestBackupAllowsReadOnlyCTE(t *testing.T) {
	for _, statement := range []string{
		"WITH v(a) AS (VALUES (1)) UPDATE test SET c = 1 FROM v WHERE test.a = v.a;",
		"WITH s AS (TABLE test2) DELETE FROM test USING s WHERE test.a = s.a;",
		"WITH s AS ((SELECT a FROM test2) UNION SELECT 1) UPDATE test SET c = 1 FROM s WHERE test.a = s.a;",
	} {
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
		}, statement, "", "backupSchema", "rollback")
		require.NoError(t, err, statement)
		require.Len(t, result, 1, statement)
	}
}
