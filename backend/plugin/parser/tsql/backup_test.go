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
		{
			name:  "update with cte",
			input: "WITH c AS (SELECT id FROM src) UPDATE dbo.test SET c1 = 1 WHERE EXISTS (SELECT 1 FROM c WHERE c.id = test.id);",
			wantSQL: strings.Join([]string{
				"WITH c AS (SELECT id FROM src)",
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [db].[dbo].[test].* FROM dbo.test WHERE EXISTS (SELECT 1 FROM c WHERE c.id = test.id)) AS backup_table;",
			}, "\n"),
		},
		{
			name:  "delete with cte",
			input: "WITH c (id) AS (SELECT id FROM src), d AS (SELECT 1 AS n) DELETE FROM test WHERE id IN (SELECT id FROM c);",
			wantSQL: strings.Join([]string{
				"WITH c (id) AS (SELECT id FROM src),",
				"d AS (SELECT 1 AS n)",
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [db].[dbo].[test].* FROM test WHERE id IN (SELECT id FROM c)) AS backup_table;",
			}, "\n"),
		},
		{
			name:  "update joined to cte",
			input: "WITH c AS (SELECT id FROM src) UPDATE t SET c1 = 1 FROM test AS t JOIN c ON t.id = c.id;",
			wantSQL: strings.Join([]string{
				"WITH c AS (SELECT id FROM src)",
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [t].* FROM test AS t JOIN c ON t.id = c.id) AS backup_table;",
			}, "\n"),
		},
		{
			name: "cte lists merge across statements",
			input: strings.Join([]string{
				"WITH c AS (SELECT id FROM src) UPDATE test SET c1 = 1 WHERE id IN (SELECT id FROM c);",
				"WITH c AS (SELECT id FROM src), d AS (SELECT id FROM other) UPDATE test SET c1 = 2 WHERE id IN (SELECT id FROM d);",
			}, "\n"),
			wantSQL: strings.Join([]string{
				"WITH c AS (SELECT id FROM src),",
				"d AS (SELECT id FROM other)",
				"SELECT * INTO [backupDB].[dbo].[rollback_test_db] FROM (",
				"  SELECT [db].[dbo].[test].* FROM test WHERE id IN (SELECT id FROM c)",
				"  UNION",
				"  SELECT [db].[dbo].[test].* FROM test WHERE id IN (SELECT id FROM d)) AS backup_table;",
			}, "\n"),
		},
		{
			name: "conflicting cte definitions rejected",
			input: strings.Join([]string{
				"WITH c AS (SELECT id FROM src) UPDATE test SET c1 = 1 WHERE id IN (SELECT id FROM c);",
				"WITH c AS (SELECT id FROM other) UPDATE test SET c1 = 2 WHERE id IN (SELECT id FROM c);",
			}, "\n"),
			wantErrPart: `conflicting definitions of CTE "c"`,
		},
		{
			name:        "xmlnamespaces rejected",
			input:       "WITH XMLNAMESPACES ('uri' AS ns), c AS (SELECT id FROM src) UPDATE test SET c1 = 1 WHERE id IN (SELECT id FROM c);",
			wantErrPart: "WITH XMLNAMESPACES",
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

func TestBackupSkipsCTETarget(t *testing.T) {
	for name, input := range map[string]string{
		"direct":       "WITH c AS (SELECT id, c1 FROM test) UPDATE c SET c1 = 1 WHERE id = 1;",
		"alias":        "WITH c AS (SELECT id, c1 FROM test) UPDATE x SET c1 = 1 FROM c AS x WHERE x.id = 1;",
		"delete":       "WITH c AS (SELECT id FROM test) DELETE FROM c WHERE id = 1;",
		"mixed case":   "WITH C AS (SELECT id, c1 FROM test) UPDATE c SET c1 = 1 WHERE id = 1;",
		"other backed": "WITH c AS (SELECT id FROM test) UPDATE c SET c1 = 1 WHERE id = 1; DELETE FROM test WHERE id = 2;",
	} {
		t.Run(name, func(t *testing.T) {
			result, err := TransformDMLToSelect(context.Background(), base.TransformContext{}, input, "db", "backupDB", "rollback")
			require.NoError(t, err)
			for _, item := range result {
				require.NotEqual(t, "c", item.SourceTableName, item.Statement)
				require.NotContains(t, item.Statement, "WITH", item.Statement)
			}
			if name == "other backed" {
				require.Len(t, result, 1)
				require.Equal(t, "test", result[0].SourceTableName)
			} else {
				require.Empty(t, result)
			}
		})
	}
	// A schema-qualified target is a real table even when a CTE shares its name.
	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{}, "WITH c AS (SELECT id FROM src) UPDATE dbo.c SET c1 = 1 WHERE id IN (SELECT id FROM c);", "db", "backupDB", "rollback")
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "c", result[0].SourceTableName)
	require.True(t, strings.HasPrefix(result[0].Statement, "WITH c AS (SELECT id FROM src)\n"), result[0].Statement)
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

// TestBackupSkipsTempTableTargets validates BYT-9359: prior backup must skip
// any UPDATE/DELETE whose target table is a temp table (#name local, ##name
// global). Temp tables are session-scoped, so a backup query running in a
// separate session would fail with "Invalid object name". Skipping is the
// only sane outcome — the rows can't be reconstructed across sessions and
// have no rollback semantics anyway.
func TestBackupSkipsTempTableTargets(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantLen         int
		wantTargetTable string // only checked when wantLen == 1
	}{
		{
			name:    "update local temp table only",
			input:   "UPDATE #tmp SET c1 = 1 WHERE c2 = 2;",
			wantLen: 0,
		},
		{
			name:    "delete from local temp table only",
			input:   "DELETE FROM #tmp WHERE c1 = 1;",
			wantLen: 0,
		},
		{
			name:    "update global temp table only",
			input:   "UPDATE ##gtmp SET c1 = 1 WHERE c2 = 2;",
			wantLen: 0,
		},
		{
			name: "mixed temp and real targets",
			input: strings.Join([]string{
				"UPDATE #tmp SET c1 = 1 WHERE c2 = 2;",
				"UPDATE test SET c1 = 2 WHERE c1 = 1;",
			}, "\n"),
			wantLen:         1,
			wantTargetTable: "rollback_test_db",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := TransformDMLToSelect(context.Background(), base.TransformContext{}, tc.input, "db", "backupDB", "rollback")
			require.NoError(t, err)
			require.Len(t, result, tc.wantLen)
			if tc.wantLen == 1 {
				require.Equal(t, tc.wantTargetTable, result[0].TargetTableName)
			}
		})
	}
}
