package tokenizer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

// TestSetLineForPGCreateTableStmt tests that the tokenizer handles various CREATE TABLE statements,
// including partition tables without columns/constraints and regular tables with columns
func TestSetLineForPGCreateTableStmt(t *testing.T) {
	testCases := []struct {
		name           string
		statement      string
		hasColumns     bool
		hasConstraints bool
		expectError    bool
	}{
		{
			name:           "Partition table DEFAULT - no columns or constraints",
			statement:      "CREATE TABLE mes.insp_oqc_log_default PARTITION OF mes.insp_oqc_log DEFAULT;",
			hasColumns:     false,
			hasConstraints: false,
			expectError:    false,
		},
		{
			name:           "Partition table range - no columns or constraints",
			statement:      "CREATE TABLE mes.insp_oqc_log_2025q1 PARTITION OF mes.insp_oqc_log FOR VALUES FROM ('2025-01-01') TO ('2025-04-01');",
			hasColumns:     false,
			hasConstraints: false,
			expectError:    false,
		},
		{
			name:           "Partition table list - no columns or constraints",
			statement:      "CREATE TABLE cities_ab PARTITION OF cities FOR VALUES IN ('New York', 'Chicago');",
			hasColumns:     false,
			hasConstraints: false,
			expectError:    false,
		},
		{
			name:           "Partition table hash - no columns or constraints",
			statement:      "CREATE TABLE orders_p0 PARTITION OF orders FOR VALUES WITH (modulus 4, remainder 0);",
			hasColumns:     false,
			hasConstraints: false,
			expectError:    false,
		},
		{
			name:           "CREATE TABLE AS SELECT - no columns or constraints",
			statement:      "CREATE TABLE new_table AS SELECT * FROM old_table;",
			hasColumns:     false,
			hasConstraints: false,
			expectError:    false,
		},
		{
			name:           "Regular table with columns",
			statement:      "CREATE TABLE test_table (id INT PRIMARY KEY, name VARCHAR(100));",
			hasColumns:     true,
			hasConstraints: false,
			expectError:    false,
		},
		{
			name: "Regular table with columns - multiline",
			statement: `CREATE TABLE test_table (
				id INT PRIMARY KEY,
				name VARCHAR(100),
				created_at TIMESTAMP
			);`,
			hasColumns:     true,
			hasConstraints: false,
			expectError:    false,
		},
		{
			name:           "Table with columns and table constraints",
			statement:      "CREATE TABLE test (id INT, name VARCHAR(100), UNIQUE(name));",
			hasColumns:     true,
			hasConstraints: true,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock CreateTableStmt
			createTableStmt := &ast.CreateTableStmt{}

			// Add columns if test case has columns
			if tc.hasColumns {
				createTableStmt.ColumnList = []*ast.ColumnDef{
					{ColumnName: "id", ConstraintList: []*ast.ConstraintDef{{Name: "PRIMARY KEY"}}},
					{ColumnName: "name"},
				}
				// Add created_at for multiline case
				if len(tc.statement) > 100 {
					createTableStmt.ColumnList = append(createTableStmt.ColumnList,
						&ast.ColumnDef{ColumnName: "created_at"})
				}
			}

			// Add constraints if test case has constraints
			if tc.hasConstraints {
				createTableStmt.ConstraintList = []*ast.ConstraintDef{
					{Name: "unique_name"},
				}
			}

			// Create tokenizer
			tokenizer := NewTokenizer(tc.statement)

			// Test SetLineForPGCreateTableStmt
			err := tokenizer.SetLineForPGCreateTableStmt(createTableStmt, 1)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err, "SetLineForPGCreateTableStmt should handle: %s", tc.name)

				// For tables with columns, verify that columns have their lines set
				if tc.hasColumns {
					for _, col := range createTableStmt.ColumnList {
						require.NotEqual(t, 0, col.LastLine(), "Column %s should have LastLine set", col.ColumnName)
					}
				}
			}
		})
	}
}
