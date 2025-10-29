package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetStatementTypesANTLR(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		expectedTypes []string
	}{
		{
			name:          "CREATE TABLE",
			sql:           "CREATE TABLE t1 (id INT);",
			expectedTypes: []string{"CREATE_TABLE"},
		},
		{
			name:          "CREATE VIEW",
			sql:           "CREATE VIEW v1 AS SELECT * FROM t1;",
			expectedTypes: []string{"CREATE_VIEW"},
		},
		{
			name:          "CREATE INDEX",
			sql:           "CREATE INDEX idx_name ON t1(name);",
			expectedTypes: []string{"CREATE_INDEX"},
		},
		{
			name:          "CREATE SEQUENCE",
			sql:           "CREATE SEQUENCE seq1;",
			expectedTypes: []string{"CREATE_SEQUENCE"},
		},
		{
			name:          "CREATE SCHEMA",
			sql:           "CREATE SCHEMA schema1;",
			expectedTypes: []string{"CREATE_SCHEMA"},
		},
		{
			name:          "CREATE FUNCTION",
			sql:           "CREATE FUNCTION func1() RETURNS INT AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;",
			expectedTypes: []string{"CREATE_FUNCTION"},
		},
		{
			name:          "DROP TABLE",
			sql:           "DROP TABLE t1;",
			expectedTypes: []string{"DROP_TABLE"},
		},
		{
			name:          "DROP INDEX",
			sql:           "DROP INDEX idx_name;",
			expectedTypes: []string{"DROP_INDEX"},
		},
		{
			name:          "DROP SCHEMA",
			sql:           "DROP SCHEMA schema1;",
			expectedTypes: []string{"DROP_SCHEMA"},
		},
		{
			name:          "ALTER TABLE",
			sql:           "ALTER TABLE t1 ADD COLUMN name VARCHAR(100);",
			expectedTypes: []string{"ALTER_TABLE"},
		},
		{
			name:          "ALTER SEQUENCE",
			sql:           "ALTER SEQUENCE seq1 RESTART WITH 100;",
			expectedTypes: []string{"ALTER_SEQUENCE"},
		},
		{
			name:          "INSERT",
			sql:           "INSERT INTO t1 (id) VALUES (1);",
			expectedTypes: []string{"INSERT"},
		},
		{
			name:          "UPDATE",
			sql:           "UPDATE t1 SET name = 'test' WHERE id = 1;",
			expectedTypes: []string{"UPDATE"},
		},
		{
			name:          "DELETE",
			sql:           "DELETE FROM t1 WHERE id = 1;",
			expectedTypes: []string{"DELETE"},
		},
		{
			name:          "COMMENT",
			sql:           "COMMENT ON TABLE t1 IS 'test table';",
			expectedTypes: []string{"COMMENT"},
		},
		{
			name: "Multiple statements",
			sql: `CREATE TABLE t1 (id INT);
				  CREATE INDEX idx_id ON t1(id);
				  INSERT INTO t1 (id) VALUES (1);`,
			expectedTypes: []string{"CREATE_TABLE", "CREATE_INDEX", "INSERT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseResult, err := ParsePostgreSQL(tt.sql)
			require.NoError(t, err)

			types, err := GetStatementTypesANTLR(parseResult)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.expectedTypes, types)
		})
	}
}

func TestGetStatementTypesWithPositionsANTLR(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []StatementTypeWithPosition
	}{
		{
			name: "Single statement",
			sql:  "CREATE TABLE t1 (id INT);",
			expected: []StatementTypeWithPosition{
				{
					Type: "CREATE_TABLE",
					Line: 1,
					Text: "CREATE TABLE t1 (id INT);",
				},
			},
		},
		{
			name: "Multiple statements",
			sql: `CREATE TABLE t1 (id INT);
DROP TABLE t2;
INSERT INTO t1 VALUES (1);`,
			expected: []StatementTypeWithPosition{
				{
					Type: "CREATE_TABLE",
					Line: 1,
				},
				{
					Type: "DROP_TABLE",
					Line: 2,
				},
				{
					Type: "INSERT",
					Line: 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseResult, err := ParsePostgreSQL(tt.sql)
			require.NoError(t, err)

			results, err := GetStatementTypesWithPositionsANTLR(parseResult)
			require.NoError(t, err)
			require.Len(t, results, len(tt.expected))

			for i, expected := range tt.expected {
				require.Equal(t, expected.Type, results[i].Type, "Statement %d type mismatch", i)
				require.Equal(t, expected.Line, results[i].Line, "Statement %d line mismatch", i)
				if expected.Text != "" {
					// Check that text contains expected content (may not include semicolon)
					require.Contains(t, results[i].Text, "CREATE TABLE t1", "Statement %d text mismatch", i)
				}
			}
		})
	}
}
