package trino

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestQueryTypeClassification verifies the statement classification (the
// query_type contract) and the read-only / data-changing / schema-changing
// helpers against the omni-backed AST. It replaces the legacy
// TestGetStatementType, which additionally asserted an ANTLR-specific
// StatementType enum that no longer exists; the user-facing behaviour
// (base.QueryType + the Is*Statement predicates) is preserved here.
func TestQueryTypeClassification(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		wantQueryType base.QueryType
		wantIsAnalyze bool
	}{
		{
			name:          "SELECT statement",
			sql:           "SELECT * FROM users",
			wantQueryType: base.Select,
			wantIsAnalyze: false,
		},
		{
			name:          "EXPLAIN statement",
			sql:           "EXPLAIN SELECT * FROM users",
			wantQueryType: base.Explain,
			wantIsAnalyze: false,
		},
		{
			name:          "EXPLAIN ANALYZE statement",
			sql:           "EXPLAIN ANALYZE SELECT * FROM users",
			wantQueryType: base.Select,
			wantIsAnalyze: true,
		},
		{
			name:          "INSERT statement",
			sql:           "INSERT INTO users (id, name) VALUES (1, 'John')",
			wantQueryType: base.DML,
			wantIsAnalyze: false,
		},
		{
			name:          "UPDATE statement",
			sql:           "UPDATE users SET name = 'John' WHERE id = 1",
			wantQueryType: base.DML,
			wantIsAnalyze: false,
		},
		{
			name:          "DELETE statement",
			sql:           "DELETE FROM users WHERE id = 1",
			wantQueryType: base.DML,
			wantIsAnalyze: false,
		},
		{
			name:          "CREATE TABLE statement",
			sql:           "CREATE TABLE users (id INT, name VARCHAR)",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "CREATE TABLE AS SELECT statement",
			sql:           "CREATE TABLE new_users AS SELECT * FROM users",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "CREATE VIEW statement",
			sql:           "CREATE VIEW user_view AS SELECT id, name FROM users",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE ADD COLUMN",
			sql:           "ALTER TABLE users ADD COLUMN email VARCHAR",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE DROP COLUMN",
			sql:           "ALTER TABLE users DROP COLUMN email",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE RENAME COLUMN",
			sql:           "ALTER TABLE users RENAME COLUMN old_name TO new_name",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "ALTER TABLE SET COLUMN TYPE",
			sql:           "ALTER TABLE users ALTER COLUMN name SET DATA TYPE VARCHAR(100)",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "DROP TABLE statement",
			sql:           "DROP TABLE users",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "DROP VIEW statement",
			sql:           "DROP VIEW user_view",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "CREATE SCHEMA statement",
			sql:           "CREATE SCHEMA new_schema",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "DROP SCHEMA statement",
			sql:           "DROP SCHEMA old_schema",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "RENAME TABLE statement",
			sql:           "ALTER TABLE users RENAME TO new_users",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "RENAME SCHEMA statement",
			sql:           "ALTER SCHEMA old_schema RENAME TO new_schema",
			wantQueryType: base.DDL,
			wantIsAnalyze: false,
		},
		{
			name:          "SHOW TABLES statement",
			sql:           "SHOW TABLES",
			wantQueryType: base.SelectInfoSchema,
			wantIsAnalyze: false,
		},
		{
			name:          "SHOW SCHEMAS statement",
			sql:           "SHOW SCHEMAS",
			wantQueryType: base.SelectInfoSchema,
			wantIsAnalyze: false,
		},
		{
			name:          "SHOW COLUMNS statement",
			sql:           "SHOW COLUMNS FROM users",
			wantQueryType: base.SelectInfoSchema,
			wantIsAnalyze: false,
		},
		{
			name:          "SET SESSION statement",
			sql:           "SET SESSION optimize_hash_generation = true",
			wantQueryType: base.Select,
			wantIsAnalyze: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseTrinoSQL(tt.sql)
			require.NoError(t, err, "Parsing should not fail")
			require.Len(t, parsed, 1, "Should parse exactly one statement")
			node := parsed[0].Node()

			queryType, isAnalyze := getQueryType(node)
			assert.Equal(t, tt.wantQueryType, queryType, "Query type mismatch")
			assert.Equal(t, tt.wantIsAnalyze, isAnalyze, "isAnalyze mismatch")

			if tt.wantQueryType == base.Select || tt.wantQueryType == base.Explain || tt.wantQueryType == base.SelectInfoSchema {
				assert.True(t, IsReadOnlyStatement(node), "Should be read-only")
			} else {
				assert.False(t, IsReadOnlyStatement(node), "Should not be read-only")
			}

			if tt.wantQueryType == base.DML {
				assert.True(t, IsDataChangingStatement(node), "Should be data-changing")
			} else {
				assert.False(t, IsDataChangingStatement(node), "Should not be data-changing")
			}

			if tt.wantQueryType == base.DDL {
				assert.True(t, IsSchemaChangingStatement(node), "Should be schema-changing")
			} else {
				assert.False(t, IsSchemaChangingStatement(node), "Should not be schema-changing")
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
