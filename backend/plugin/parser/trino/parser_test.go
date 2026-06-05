package trino

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTrinoParser(t *testing.T) {
	// Simple query parsing test.
	sql := "SELECT a.id, b.name FROM table1 a JOIN table2 b ON a.id = b.id"

	results, err := parseTrinoSQL(sql)
	require.NoError(t, err, "Failed to parse valid SQL")
	require.Len(t, results, 1, "Should parse exactly one statement")
	require.NotNil(t, results[0].Node(), "Parse tree should not be nil")

	// Verify the statement is read-only.
	assert.True(t, IsReadOnlyStatement(results[0].Node()), "SELECT statement should be read-only")

	// Test data-changing statement detection.
	sqlDML := "INSERT INTO users (id, name) VALUES (1, 'John')"
	resultsDML, err := parseTrinoSQL(sqlDML)
	require.NoError(t, err)
	require.Len(t, resultsDML, 1)
	assert.True(t, IsDataChangingStatement(resultsDML[0].Node()), "INSERT statement should be data-changing")

	// Test schema-changing statement detection.
	sqlDDL := "CREATE TABLE test (id INT, name VARCHAR)"
	resultsDDL, err := parseTrinoSQL(sqlDDL)
	require.NoError(t, err)
	require.Len(t, resultsDDL, 1)
	assert.True(t, IsSchemaChangingStatement(resultsDDL[0].Node()), "CREATE TABLE should be schema-changing")
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
			expected: base.Select, // EXPLAIN ANALYZE executes the query, so it is classified as Select.
		},
		{
			sql:      "SHOW TABLES;",
			expected: base.SelectInfoSchema,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.sql, func(t *testing.T) {
			// queryTypeFromText applies the same system-schema promotion the
			// query-span path uses, so a SELECT over system.* is SelectInfoSchema.
			queryType := queryTypeFromText(tc.sql)
			assert.Equal(t, tc.expected, queryType)
		})
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
