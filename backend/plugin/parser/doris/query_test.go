package doris

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		statement   string
		valid       bool
		description string
	}{
		{
			statement:   "SELECT * FROM users",
			valid:       true,
			description: "Basic SELECT query",
		},
		{
			statement:   "SELECT * FROM users WHERE id = 1",
			valid:       true,
			description: "SELECT with WHERE clause",
		},
		{
			statement:   "SHOW DATA",
			valid:       true,
			description: "SHOW DATA statement",
		},
		{
			statement:   "SHOW DATA FROM db1",
			valid:       true,
			description: "SHOW DATA with FROM clause",
		},
		{
			statement:   "SHOW DATABASES",
			valid:       true,
			description: "SHOW DATABASES statement",
		},
		{
			statement:   "SHOW TABLES",
			valid:       true,
			description: "SHOW TABLES statement",
		},
		{
			statement:   "SHOW TABLETS FROM table1",
			valid:       true,
			description: "SHOW TABLETS statement",
		},
		{
			statement:   "SHOW VARIABLES",
			valid:       true,
			description: "SHOW VARIABLES statement",
		},
		{
			statement:   "SHOW CREATE TABLE users",
			valid:       true,
			description: "SHOW CREATE TABLE statement",
		},
		{
			statement:   "SHOW CREATE DATABASE db1",
			valid:       true,
			description: "SHOW CREATE DATABASE statement",
		},
		{
			statement:   "INSERT INTO users (id, name) VALUES (1, 'test')",
			valid:       false,
			description: "INSERT statement should be invalid",
		},
		{
			statement:   "UPDATE users SET name = 'test' WHERE id = 1",
			valid:       false,
			description: "UPDATE statement should be invalid",
		},
		{
			statement:   "DELETE FROM users WHERE id = 1",
			valid:       false,
			description: "DELETE statement should be invalid",
		},
		{
			statement:   "CREATE TABLE test (id INT)",
			valid:       false,
			description: "CREATE TABLE should be invalid",
		},
		{
			statement:   "DROP TABLE users",
			valid:       false,
			description: "DROP TABLE should be invalid",
		},
		{
			statement:   "EXPLAIN INSERT INTO users (id, name) VALUES (1, 'test')",
			valid:       true,
			description: "EXPLAIN INSERT should be valid (read-only)",
		},
		{
			statement:   "EXPLAIN UPDATE users SET name = 'test' WHERE id = 1",
			valid:       true,
			description: "EXPLAIN UPDATE should be valid (read-only)",
		},
		{
			statement:   "EXPLAIN DELETE FROM users WHERE id = 1",
			valid:       true,
			description: "EXPLAIN DELETE should be valid (read-only)",
		},
		{
			statement:   "WITH c AS (SELECT 1) SELECT * FROM c",
			valid:       true,
			description: "CTE-prefixed SELECT is read-only",
		},
		{
			statement:   "SELECT EXTRACT(YEAR FROM MONTHS_ADD(NOW(), -1))",
			valid:       true,
			description: "EXTRACT(unit FROM expr) is read-only",
		},
		{
			statement: `SELECT EXTRACT(
  YEAR
  FROM
    MONTHS_ADD(NOW(), -1)
)`,
			valid:       true,
			description: "Multi-line EXTRACT is read-only",
		},
		{
			statement:   "WITH c AS (SELECT EXTRACT(MONTH FROM created_at) m FROM t) SELECT * FROM c",
			valid:       true,
			description: "EXTRACT inside a CTE body is read-only",
		},
		{
			statement:   "SELECT TRIM(s), LEFT(s, 2), RIGHT(s, 2), IF(a, 1, 2) FROM t",
			valid:       true,
			description: "Reserved keywords used as function names are read-only",
		},
		{
			statement:   "SELECT DATABASE()",
			valid:       true,
			description: "DATABASE() is read-only",
		},
		{
			statement:   "SELECT CURRENT_TIMESTAMP(3)",
			valid:       true,
			description: "Datetime function with precision is read-only",
		},
		{
			statement:   "SELECT @@version_comment, @@session.query_timeout, @uservar",
			valid:       true,
			description: "Session and user variables are read-only",
		},
		{
			statement:   "SELECT [1, 2, 3], {'a': 1}, x'1f', b'101'",
			valid:       true,
			description: "Array, map, hex and bit literals are read-only",
		},
		{
			statement:   "SELECT array_map(x -> x + 1, arr) FROM t",
			valid:       true,
			description: "Lambda in a higher-order array function is read-only",
		},
		{
			statement:   "SELECT /*+ SET_VAR(query_timeout = 100) */ a FROM t",
			valid:       true,
			description: "Statement with an optimizer hint is read-only",
		},
		{
			statement:   "SELECT a, SUM(b) FROM t GROUP BY CUBE(a)",
			valid:       true,
			description: "GROUP BY CUBE is read-only",
		},
		{
			statement:   "SELECT CONVERT(s USING utf8), CONVERT(s, SIGNED), BINARY s FROM t",
			valid:       true,
			description: "CONVERT and BINARY forms are read-only",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			a := require.New(t)
			valid, _, err := validateQuery(tc.statement)
			a.NoError(err)
			a.Equal(tc.valid, valid, "statement: %s", tc.statement)
		})
	}

	// Cases that must fail validation — either because they're not read-only
	// or because they don't parse. Both shapes flow through the same code
	// path and must be rejected (either via valid=false or err!=nil).
	rejectCases := []struct {
		statement   string
		description string
	}{
		{
			// CTE-prefixed DML must NOT be accepted as read-only — the
			// keyword-based Classify would have tagged it as SELECT because
			// `WITH` is the leading keyword. AST-based validation catches it
			// (or, currently, the omni parser rejects it as a syntax error;
			// either rejection is acceptable).
			statement:   "WITH c AS (SELECT 1) UPDATE t SET x = 1",
			description: "CTE-prefixed UPDATE must be rejected",
		},
		{
			statement:   "WITH c AS (SELECT 1) DELETE FROM t",
			description: "CTE-prefixed DELETE must be rejected",
		},
		{
			statement:   "WITH c AS (SELECT 1) INSERT INTO t SELECT * FROM c",
			description: "CTE-prefixed INSERT must be rejected",
		},
		{
			statement:   "SELECT a > (select max(a) from t1) FROM",
			description: "Truncated SELECT must be rejected as syntax error",
		},
		{
			// omni's stub parser accepts bare SHOW as *ast.ShowStmt with
			// Type="". AST-content validation rejects it.
			statement:   "SHOW",
			description: "Bare SHOW must be rejected",
		},
		{
			statement:   "DESCRIBE",
			description: "Bare DESCRIBE must be rejected",
		},
		{
			statement:   "EXPLAIN",
			description: "Bare EXPLAIN must be rejected",
		},
		{
			// EXPLAIN over DDL is not a real read-only operation.
			statement:   "EXPLAIN DROP TABLE t",
			description: "EXPLAIN over DDL must be rejected",
		},
	}
	for _, tc := range rejectCases {
		t.Run(tc.description, func(t *testing.T) {
			a := require.New(t)
			valid, _, err := validateQuery(tc.statement)
			a.False(valid && err == nil, "expected rejection, statement: %s", tc.statement)
		})
	}
}
