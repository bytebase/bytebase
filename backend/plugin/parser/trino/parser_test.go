package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTrinoParser(t *testing.T) {
	// Simple query parsing test
	sql := "SELECT a.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id"

	results, err := ParseTrino(sql)
	require.NoError(t, err, "Failed to parse valid SQL")
	require.Len(t, results, 1, "Should parse exactly one statement")
	require.NotNil(t, results[0].Tree, "Parse tree should not be nil")

	// Verify statement type
	stmtType := GetStatementType(results[0].Tree)
	assert.Equal(t, Select, stmtType, "Statement should be recognized as SELECT")

	// Verify the statement is read-only
	isReadOnly := IsReadOnlyStatement(results[0].Tree)
	assert.True(t, isReadOnly, "SELECT statement should be read-only")

	// Test data-changing statement detection
	sqlDML := "INSERT INTO users (id, name) VALUES (1, 'John')"
	resultsDML, err := ParseTrino(sqlDML)
	require.NoError(t, err)
	require.Len(t, resultsDML, 1)
	isDML := IsDataChangingStatement(resultsDML[0].Tree)
	assert.True(t, isDML, "INSERT statement should be data-changing")

	// Test schema-changing statement detection
	sqlDDL := "CREATE TABLE test (id INT, name VARCHAR)"
	resultsDDL, err := ParseTrino(sqlDDL)
	require.NoError(t, err)
	require.Len(t, resultsDDL, 1)
	isDDL := IsSchemaChangingStatement(resultsDDL[0].Tree)
	assert.True(t, isDDL, "CREATE TABLE should be schema-changing")
}

func TestTrinoQueryType(t *testing.T) {
	testCases := []struct {
		sql      string
		expected base.QueryType
	}{
		{
			sql:      "SELECT * FROM system.runtime.nodes;",
			expected: base.SelectInfoSchema,
		},
		{
			sql:      "SELECT * FROM users;",
			expected: base.Select,
		},
		{
			sql:      "INSERT INTO users (id, name) VALUES (1, 'John');",
			expected: base.DML,
		},
		{
			sql:      "UPDATE users SET name = 'Jane' WHERE id = 1;",
			expected: base.DML,
		},
		{
			sql:      "DELETE FROM users WHERE id = 1;",
			expected: base.DML,
		},
		{
			sql:      "CREATE TABLE users (id INT, name VARCHAR);",
			expected: base.DDL,
		},
		{
			sql:      "DROP TABLE users;",
			expected: base.DDL,
		},
		{
			sql:      "EXPLAIN SELECT * FROM users;",
			expected: base.Explain,
		},
		{
			sql:      "EXPLAIN ANALYZE SELECT * FROM users;",
			expected: base.Select, // Special case for EXPLAIN ANALYZE
		},
		{
			sql:      "SHOW TABLES;",
			expected: base.SelectInfoSchema,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.sql, func(t *testing.T) {
			results, err := ParseTrino(tc.sql)
			require.NoError(t, err)
			require.Len(t, results, 1)

			queryType, _ := getQueryType(results[0].Tree)
			assert.Equal(t, tc.expected, queryType)
		})
	}
}

func TestGetQuerySpan(t *testing.T) {
	testCases := []struct {
		name     string
		sql      string
		database string
		schema   string
		wantType base.QueryType
	}{
		{
			name:     "Simple SELECT",
			sql:      "SELECT id, name FROM users;",
			database: "mydb",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "SELECT with JOIN",
			sql:      "SELECT u.id, u.name, o.product FROM users u JOIN orders o ON u.id = o.user_id;",
			database: "mydb",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "INSERT",
			sql:      "INSERT INTO users (id, name) VALUES (1, 'John');",
			database: "mydb",
			schema:   "public",
			wantType: base.DML,
		},
		{
			name:     "UPDATE",
			sql:      "UPDATE users SET name = 'Jane' WHERE id = 1;",
			database: "mydb",
			schema:   "public",
			wantType: base.DML,
		},
		{
			name:     "DELETE",
			sql:      "DELETE FROM users WHERE id = 1;",
			database: "mydb",
			schema:   "public",
			wantType: base.DML,
		},
		{
			name:     "CREATE TABLE",
			sql:      "CREATE TABLE users (id INT, name VARCHAR);",
			database: "mydb",
			schema:   "public",
			wantType: base.DDL,
		},
		{
			name:     "EXPLAIN",
			sql:      "EXPLAIN SELECT * FROM users;",
			database: "mydb",
			schema:   "public",
			wantType: base.Explain,
		},
		{
			name:     "EXPLAIN ANALYZE",
			sql:      "EXPLAIN ANALYZE SELECT * FROM users;",
			database: "mydb",
			schema:   "public",
			wantType: base.Select,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock GetQuerySpanContext without metadata function
			gCtx := base.GetQuerySpanContext{
				InstanceID: "test-instance",
				Engine:     storepb.Engine_TRINO,
				// No metadata function provided - mock implementation will handle this
			}

			// Special handling for non-SELECT queries that will skip metadata lookup
			if tc.wantType != base.Select && tc.wantType != base.Explain && tc.wantType != base.SelectInfoSchema {
				span, err := GetQuerySpan(context.Background(), gCtx, tc.sql, tc.database, tc.schema, true)
				if assert.NoError(t, err) {
					assert.Equal(t, tc.wantType, span.Type)
					assert.Empty(t, span.Results, "Results should be empty for non-SELECT queries")
				}
				return
			}

			// For SELECT queries, we expect a basic query span with the correct type
			span, err := GetQuerySpan(context.Background(), gCtx, tc.sql, tc.database, tc.schema, true)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.wantType, span.Type)
			}
		})
	}
}

func TestMain(m *testing.M) {
	// Run the tests and print results to stdout
	m.Run()
}
