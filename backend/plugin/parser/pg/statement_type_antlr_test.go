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
		name:          "DROP SEQUENCE",
		sql:           "DROP SEQUENCE seq1;",
		expectedTypes: []string{"DROP_SEQUENCE"},
	},
	{
		name:          "DROP EXTENSION",
		sql:           "DROP EXTENSION postgis;",
		expectedTypes: []string{"DROP_EXTENSION"},
	},
	{
		name:          "DROP DATABASE",
		sql:           "DROP DATABASE testdb;",
		expectedTypes: []string{"DROP_DATABASE"},
	},
	{
		name:          "DROP TYPE",
		sql:           "DROP TYPE custom_type;",
		expectedTypes: []string{"DROP_TYPE"},
	},
	{
		name:          "DROP TRIGGER",
		sql:           "DROP TRIGGER trig1 ON t1;",
		expectedTypes: []string{"DROP_TRIGGER"},
	},
	{
		name:          "DROP VIEW",
		sql:           "DROP VIEW v1;",
		expectedTypes: []string{"DROP_TABLE"},
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
		{
			name:          "RENAME TABLE",
			sql:           "ALTER TABLE t1 RENAME TO t2;",
			expectedTypes: []string{"RENAME_TABLE"},
		},
		{
			name:          "RENAME COLUMN",
			sql:           "ALTER TABLE t1 RENAME COLUMN old_name TO new_name;",
			expectedTypes: []string{"RENAME_COLUMN"},
		},
		{
			name:          "RENAME COLUMN without COLUMN keyword",
			sql:           "ALTER TABLE t1 RENAME old_name TO new_name;",
			expectedTypes: []string{"RENAME_COLUMN"},
		},
		{
			name:          "RENAME CONSTRAINT",
			sql:           "ALTER TABLE t1 RENAME CONSTRAINT old_constraint TO new_constraint;",
			expectedTypes: []string{"RENAME_CONSTRAINT"},
		},
		{
			name:          "RENAME INDEX",
			sql:           "ALTER INDEX idx_old RENAME TO idx_new;",
			expectedTypes: []string{"RENAME_INDEX"},
		},
		{
			name:          "RENAME SCHEMA",
			sql:           "ALTER SCHEMA old_schema RENAME TO new_schema;",
			expectedTypes: []string{"RENAME_SCHEMA"},
		},
		{
			name:          "RENAME SEQUENCE",
			sql:           "ALTER SEQUENCE seq_old RENAME TO seq_new;",
			expectedTypes: []string{"RENAME_SEQUENCE"},
		},
		{
			name:          "RENAME VIEW",
			sql:           "ALTER VIEW v_old RENAME TO v_new;",
			expectedTypes: []string{"RENAME_VIEW"},
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
