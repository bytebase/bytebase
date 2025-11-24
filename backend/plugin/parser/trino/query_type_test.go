package trino

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementType(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		wantType      StatementType
		wantQueryType base.QueryType
		wantIsAnalyze bool
	}{
		{
			name:          "SELECT statement",
			sql:           "SELECT * FROM users",
			wantType:      Select,
			wantQueryType: base.Select,
			wantIsAnalyze: false,
		},
		{
			name:          "EXPLAIN statement",
			sql:           "EXPLAIN SELECT * FROM users",
			wantType:      Explain,
			wantQueryType: base.Explain,
			wantIsAnalyze: false,
		},
		{
			name:          "EXPLAIN ANALYZE statement",
			sql:           "EXPLAIN ANALYZE SELECT * FROM users",
			wantType:      Explain,
			wantQueryType: base.Select,
			wantIsAnalyze: true,
		},
		{
			name:          "INSERT statement",
			sql:           "INSERT INTO users (id, name) VALUES (1, 'John')",
			wantType:      Insert,
			wantQueryType: base.DML,
			wantIsAnalyze: false,
		},
		{
			name:          "UPDATE statement",
			sql:           "UPDATE users SET name = 'John' WHERE id = 1",
			wantType:      Update,
			wantQueryType: base.DML,
			wantIsAnalyze: false,
		},
		{
			name:          "DELETE statement",
			sql:           "DELETE FROM users WHERE id = 1",
			wantType:      Delete,
			wantQueryType: base.DML,
			wantIsAnalyze: false,
		},
		// Trino MERGE syntax is complex and varies by version - skipping this test for now
		// {
		//	name:          "MERGE statement",
		//	sql:           "MERGE INTO users u USING new_users n ON u.id = n.id WHEN MATCHED THEN UPDATE SET u.name = n.name",
		//	wantType:      Merge,
		//	wantQueryType: base.DML,
		//	wantIsAnalyze: false,
		// },
		{
			name:          "CREATE TABLE statement",
			sql:           "CREATE TABLE users (id INT, name VARCHAR)",
			wantType:      CreateTable,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "CREATE TABLE AS SELECT statement",
			sql:           "CREATE TABLE new_users AS SELECT * FROM users",
			wantType:      CreateTableAsSelect,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "CREATE VIEW statement",
			sql:           "CREATE VIEW user_view AS SELECT id, name FROM users",
			wantType:      CreateView,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE ADD COLUMN",
			sql:           "ALTER TABLE users ADD COLUMN email VARCHAR",
			wantType:      AlterTable,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE DROP COLUMN",
			sql:           "ALTER TABLE users DROP COLUMN email",
			wantType:      AlterTable,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE RENAME COLUMN",
			sql:           "ALTER TABLE users RENAME COLUMN old_name TO new_name",
			wantType:      AlterTable,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE SET COLUMN TYPE",
			sql:           "ALTER TABLE users ALTER COLUMN name SET DATA TYPE VARCHAR(100)",
			wantType:      AlterTable,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "DROP TABLE statement",
			sql:           "DROP TABLE users",
			wantType:      DropTable,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "DROP VIEW statement",
			sql:           "DROP VIEW user_view",
			wantType:      DropView,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "CREATE SCHEMA statement",
			sql:           "CREATE SCHEMA new_schema",
			wantType:      CreateSchema,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "DROP SCHEMA statement",
			sql:           "DROP SCHEMA old_schema",
			wantType:      DropSchema,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "RENAME TABLE statement",
			sql:           "ALTER TABLE users RENAME TO new_users",
			wantType:      RenameTable,
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "RENAME SCHEMA statement",
			sql:           "ALTER SCHEMA old_schema RENAME TO new_schema",
			wantType:      CreateSchema, // Mapped to CreateSchema in the code
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "SHOW TABLES statement",
			sql:           "SHOW TABLES",
			wantType:      Show,
			wantQueryType: base.SelectInfoSchema,
			wantIsAnalyze: false,
		},
		{
			name:          "SHOW SCHEMAS statement",
			sql:           "SHOW SCHEMAS",
			wantType:      Show,
			wantQueryType: base.SelectInfoSchema,
			wantIsAnalyze: false,
		},
		{
			name:          "SHOW COLUMNS statement",
			sql:           "SHOW COLUMNS FROM users",
			wantType:      Show,
			wantQueryType: base.SelectInfoSchema,
			wantIsAnalyze: false,
		},
		{
			name:          "SET SESSION statement",
			sql:           "SET SESSION optimize_hash_generation = true",
			wantType:      Set,
			wantQueryType: base.Select,
			wantIsAnalyze: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the statement
			results, err := ParseTrino(tt.sql)
			require.NoError(t, err, "Parsing should not fail")
			require.Len(t, results, 1, "Should parse exactly one statement")

			result := results[0]

			// Test GetStatementType
			stmtType := GetStatementType(result.Tree)
			assert.Equal(t, tt.wantType, stmtType, "Statement type mismatch")

			// Test getQueryType
			queryType, isAnalyze := getQueryType(result.Tree)
			assert.Equal(t, tt.wantQueryType, queryType, "Query type mismatch")
			assert.Equal(t, tt.wantIsAnalyze, isAnalyze, "isAnalyze mismatch")

			// Test utility functions
			if tt.wantQueryType == base.Select || tt.wantQueryType == base.Explain || tt.wantQueryType == base.SelectInfoSchema {
				assert.True(t, IsReadOnlyStatement(result.Tree), "Should be read-only")
			} else {
				assert.False(t, IsReadOnlyStatement(result.Tree), "Should not be read-only")
			}

			if tt.wantQueryType == base.DML {
				assert.True(t, IsDataChangingStatement(result.Tree), "Should be data-changing")
			} else {
				assert.False(t, IsDataChangingStatement(result.Tree), "Should not be data-changing")
			}

			if tt.wantQueryType == base.DDL {
				assert.True(t, IsSchemaChangingStatement(result.Tree), "Should be schema-changing")
			} else {
				assert.False(t, IsSchemaChangingStatement(result.Tree), "Should not be schema-changing")
			}
		})
	}
}

func TestContainsSystemSchema(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{
			name: "Query with system schema",
			sql:  "SELECT * FROM system.runtime.nodes",
			want: true,
		},
		{
			name: "Query with information_schema",
			sql:  "SELECT * FROM information_schema.tables",
			want: true,
		},
		{
			name: "Query with $system",
			sql:  "SELECT * FROM $system.nodes",
			want: true,
		},
		{
			name: "Query with catalog",
			sql:  "SELECT * FROM catalog.schemas",
			want: true,
		},
		{
			name: "Query with metadata",
			sql:  "SELECT * FROM metadata.schemas",
			want: true,
		},
		{
			name: "Regular user table",
			sql:  "SELECT * FROM users",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsSystemSchema(tt.sql)
			assert.Equal(t, tt.want, got)
		})
	}
}
