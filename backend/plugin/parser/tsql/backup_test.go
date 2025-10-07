package tsql

import (
	"context"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

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
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

// TestIdentityColumnHandling validates that our implementation correctly handles IDENTITY columns
// This test verifies:
// 1. The generated dynamic SQL properly detects IDENTITY columns at runtime
// 2. IDENTITY columns are cast to their base types (removing IDENTITY property) during backup
// 3. The backup table can receive explicit values without IDENTITY_INSERT errors
func TestIdentityColumnHandling(t *testing.T) {
	a := require.New(t)

	// Test case: DELETE from a table that typically has IDENTITY columns
	input := `DELETE FROM positions WHERE position_id = 1;`

	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{},
		input, "db", "backupDB", "rollback")
	a.NoError(err)
	a.Len(result, 1)

	// Verify the generated SQL uses dynamic column detection
	stmt := result[0].Statement

	// Key assertions about the generated SQL:
	// 1. Uses COLUMNPROPERTY to detect IDENTITY columns
	a.Contains(stmt, "COLUMNPROPERTY(OBJECT_ID('[db].[dbo].[positions]'), c.name, 'IsIdentity')")

	// 2. Casts IDENTITY columns to their base type
	a.Contains(stmt, "CAST(' + QUOTENAME(c.name) + ' AS ' +")
	a.Contains(stmt, "TYPE_NAME(c.user_type_id)")

	// 3. Uses STRING_AGG to build column list dynamically
	a.Contains(stmt, "STRING_AGG(")

	// 4. Creates backup table with SELECT INTO
	a.Contains(stmt, "SELECT ' + @cols + ' INTO [backupDB].[dbo].[rollback_positions_db]")

	// 5. Uses dynamic SQL execution
	a.Contains(stmt, "EXEC sp_executesql @sql")

	// The key difference from the original implementation:
	// Original: SELECT * INTO backup_table (copies IDENTITY property, causing insert failures)
	// New: SELECT <columns with IDENTITY cast to base types> INTO backup_table (no IDENTITY property)
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

// TestBackupWithQuotedStrings validates that single quotes in WHERE clauses are properly escaped
func TestBackupWithQuotedStrings(t *testing.T) {
	a := require.New(t)

	// Test case with single quotes that need escaping
	input := `DELETE FROM AdminPosition WHERE positionName = 'BPM Admin';`

	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{},
		input, "TestIdentityDB", "bbdataarchive", "backup")
	a.NoError(err)
	a.Len(result, 1)

	stmt := result[0].Statement

	// The dynamic SQL should have escaped quotes (doubled single quotes)
	// The original: WHERE positionName = 'BPM Admin'
	// Should become: WHERE positionName = ''BPM Admin'' in the dynamic SQL string
	a.Contains(stmt, "WHERE positionName = ''BPM Admin''")

	// Also test with apostrophes in the string
	input2 := `UPDATE positions SET title = 'O''Reilly''s Manager' WHERE id = 1;`
	result2, err := TransformDMLToSelect(context.Background(), base.TransformContext{},
		input2, "db", "backup", "rollback")
	a.NoError(err)
	a.Len(result2, 1)

	// For UPDATE, we only backup the rows that will be changed (WHERE clause)
	// The SET clause is not part of the backup, only the WHERE clause matters
	stmt2 := result2[0].Statement
	// The WHERE clause should be properly escaped
	a.Contains(stmt2, "WHERE id = 1")
}
