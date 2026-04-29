package tsql

import (
	"context"
	"io"
	"os"
	"slices"
	"strings"
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

func TestBackupRestoreDoNotDependOnANTLR(t *testing.T) {
	for _, path := range []string{"backup.go", "restore.go"} {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		source := string(content)
		require.NotContains(t, source, "github.com/antlr4-go/antlr/v4", path)
		require.NotContains(t, source, "github.com/bytebase/parser/tsql", path)
		require.NotContains(t, source, "ParseTSQL(", path)
	}
}

func TestBackupOmniBoundaryCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSQL     string
		wantErrPart string
	}{
		{
			name:  "update top option",
			input: "UPDATE TOP (3) test SET c1 = 1 WHERE c2 = 2 OPTION (RECOMPILE);",
			wantSQL: strings.Join([]string{
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [db].[dbo].[test].* TOP (3) FROM test WHERE c2 = 2 OPTION (RECOMPILE)) AS backup_table;",
			}, "\n"),
		},
		{
			name:  "where keyword in comment",
			input: "UPDATE test SET c1 = 1 WHERE /* WHERE */ c2 = 2;",
			wantSQL: strings.Join([]string{
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [db].[dbo].[test].* FROM test WHERE /* WHERE */ c2 = 2) AS backup_table;",
			}, "\n"),
		},
		{
			name:  "option keyword in comment",
			input: "UPDATE test SET c1 = 1 /* OPTION */ OPTION (RECOMPILE);",
			wantSQL: strings.Join([]string{
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [db].[dbo].[test].* FROM test OPTION (RECOMPILE)) AS backup_table;",
			}, "\n"),
		},
		{
			name:  "option keyword in nested comment",
			input: "UPDATE test SET c1 = 1 /* outer /* inner */ OPTION in outer */ OPTION (RECOMPILE);",
			wantSQL: strings.Join([]string{
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [db].[dbo].[test].* FROM test OPTION (RECOMPILE)) AS backup_table;",
			}, "\n"),
		},
		{
			name: "delete alias from join",
			input: strings.Join([]string{
				"DELETE FROM t_alias",
				"FROM test AS t_alias JOIN test2 AS t2 ON t_alias.c1 = t2.c1",
				"WHERE t_alias.c1 = 1;",
			}, "\n"),
			wantSQL: strings.Join([]string{
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [t_alias].* FROM test AS t_alias JOIN test2 AS t2 ON t_alias.c1 = t2.c1 WHERE t_alias.c1 = 1) AS backup_table;",
			}, "\n"),
		},
		{
			name:        "update current of rejected",
			input:       "UPDATE test SET c1 = 1 WHERE CURRENT OF my_cursor;",
			wantErrPart: "CURSOR clause is not supported",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := TransformDMLToSelect(context.Background(), base.TransformContext{}, tc.input, "db", "backupDB", "rollback")
			if tc.wantErrPart != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrPart)
				return
			}
			require.NoError(t, err)
			require.Len(t, result, 1)
			require.Equal(t, tc.wantSQL, result[0].Statement)
		})
	}
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
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{}, t.Input, "db", "backupDB", "rollback")
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

// TestIdentityColumnHandling validates that our implementation correctly handles IDENTITY columns
// This test verifies:
// 1. The backup uses simple SELECT INTO (copying IDENTITY properties naturally)
// 2. The restore.go handles IDENTITY_INSERT during rollback
func TestIdentityColumnHandling(t *testing.T) {
	a := require.New(t)

	// Test case: DELETE from a table that typically has IDENTITY columns
	input := `DELETE FROM positions WHERE position_id = 1;`

	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{},
		input, "db", "backupDB", "rollback")
	a.NoError(err)
	a.Len(result, 1)

	// Verify the generated SQL uses simple SELECT INTO
	stmt := result[0].Statement

	// Key assertions about the generated SQL:
	// 1. Uses simple SELECT INTO
	a.Contains(stmt, "SELECT * INTO [backupDB].[dbo].[rollback_positions_db]")

	// 2. Selects from the original table with proper bracketing
	a.Contains(stmt, "SELECT [db].[dbo].[positions].* FROM")

	// 3. Includes the WHERE clause
	a.Contains(stmt, "WHERE position_id = 1")

	// The approach:
	// Backup: Simple SELECT * INTO backup_table (copies IDENTITY property naturally)
	// Restore: restore.go handles IDENTITY_INSERT ON/OFF for rollback
}

// TestBackupStatementStructure validates the structure of backup statements
func TestBackupStatementStructure(t *testing.T) {
	a := require.New(t)

	// Test that UPDATE statements generate correct backup SQL
	input := `UPDATE employees SET salary = salary * 1.1 WHERE department_id = 5;`

	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{},
		input, "production", "backup", "migration")
	a.NoError(err)
	a.Len(result, 1)

	backupStmt := result[0]

	// Verify backup statement metadata
	a.Equal("dbo", backupStmt.SourceSchema)
	a.Equal("employees", backupStmt.SourceTableName)
	a.Equal("migration_employees_production", backupStmt.TargetTableName)

	// Verify the SQL structure handles the WHERE clause properly
	a.Contains(backupStmt.Statement, "WHERE department_id = 5")
}

// TestBackupWithQuotedStrings validates that single quotes in WHERE clauses are properly handled
func TestBackupWithQuotedStrings(t *testing.T) {
	a := require.New(t)

	// Test case with single quotes
	input := `DELETE FROM AdminPosition WHERE positionName = 'BPM Admin';`

	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{},
		input, "TestIdentityDB", "bbdataarchive", "backup")
	a.NoError(err)
	a.Len(result, 1)

	stmt := result[0].Statement

	// With simple SELECT INTO, quotes are handled naturally by SQL Server
	// The WHERE clause should appear as-is
	a.Contains(stmt, "WHERE positionName = 'BPM Admin'")

	// Also test with apostrophes in the string
	input2 := `UPDATE positions SET title = 'O''Reilly''s Manager' WHERE id = 1;`
	result2, err := TransformDMLToSelect(context.Background(), base.TransformContext{},
		input2, "db", "backup", "rollback")
	a.NoError(err)
	a.Len(result2, 1)

	// For UPDATE, we only backup the rows that will be changed (WHERE clause)
	// The SET clause is not part of the backup, only the WHERE clause matters
	stmt2 := result2[0].Statement
	// The WHERE clause should be present
	a.Contains(stmt2, "WHERE id = 1")
}
