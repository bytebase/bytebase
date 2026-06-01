package plsql

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
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{}, t.Input, "DB", "backupDB", "rollback")
		a.NoError(err)
		slices.SortFunc(result, func(a, b base.BackupStatement) int {
			if a.TargetTableName == b.TargetTableName {
				if a.Statement < b.Statement {
					return -1
				}
				if a.Statement > b.Statement {
					return 1
				}
				return 0
			}
			if a.TargetTableName < b.TargetTableName {
				return -1
			}
			if a.TargetTableName > b.TargetTableName {
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

func TestBackupOmniBoundaryCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSQL string
	}{
		{
			name:  "update returning is not copied into select suffix",
			input: "UPDATE test SET c1 = 1 WHERE c1 = 2 RETURNING c1 INTO :old_c1;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test WHERE c1 = 2;`,
		},
		{
			name:  "delete returning is not copied into select suffix",
			input: "DELETE FROM test WHERE c1 = 2 RETURNING c1 INTO :old_c1;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test WHERE c1 = 2;`,
		},
		{
			name:  "update log errors is not copied into select suffix",
			input: "UPDATE test SET c1 = 1 WHERE c1 = 2 LOG ERRORS INTO err$_test REJECT LIMIT UNLIMITED;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test WHERE c1 = 2;`,
		},
		{
			name:  "where keyword in block comment",
			input: "UPDATE test SET c1 = 1 WHERE /* WHERE */ c1 = 2;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test WHERE /* WHERE */ c1 = 2;`,
		},
		{
			name:  "where keyword in string literal",
			input: "DELETE FROM test WHERE note = 'WHERE should stay literal';",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test WHERE note = 'WHERE should stay literal';`,
		},
		{
			name:  "update without where backs up whole table",
			input: "UPDATE test SET c1 = 1;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test;`,
		},
		{
			name:  "delete without where backs up whole table",
			input: "DELETE FROM test;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test;`,
		},
		{
			name:  "update partition target",
			input: "UPDATE test PARTITION (p1) SET c1 = 1 WHERE c1 = 2;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test PARTITION (p1) WHERE c1 = 2;`,
		},
		{
			name:  "delete subpartition target",
			input: "DELETE FROM test SUBPARTITION (sp1) WHERE c1 = 2;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "TEST".* FROM test SUBPARTITION (sp1) WHERE c1 = 2;`,
		},
		{
			name:  "update from appends source table list",
			input: "UPDATE test t SET c1 = s.c1 FROM source_table s WHERE t.id = s.id;",
			wantSQL: `CREATE TABLE "backupDB"."rollback_TEST_DB" AS
  SELECT "T".* FROM test t, source_table s WHERE t.id = s.id;`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := TransformDMLToSelect(context.Background(), base.TransformContext{}, tc.input, "DB", "backupDB", "rollback")
			require.NoError(t, err)
			require.Len(t, result, 1)
			require.Equal(t, tc.wantSQL, result[0].Statement)
		})
	}
}
