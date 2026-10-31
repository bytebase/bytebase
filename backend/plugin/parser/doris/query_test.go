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
		{
			statement:   "(SELECT 1) LIMIT 1",
			valid:       true,
			description: "Parenthesized query with trailing LIMIT is read-only",
		},
		{
			statement:   "(SELECT 1 LIMIT 1) LIMIT 2",
			valid:       true,
			description: "GroupedQuery wrapper (repeated clause groups) is read-only",
		},
		{
			statement:   "SELECT 1 LIMIT 5 ORDER BY 1",
			valid:       true,
			description: "Engine-lenient clause order wraps in GroupedQuery, still read-only",
		},
		{
			statement:   "(WITH c AS (SELECT 1) SELECT * FROM c)",
			valid:       true,
			description: "Parenthesized WITH group (CTE scope boundary) is read-only",
		},
		{
			statement:   "SELECT 1 UNION (SELECT 2) LIMIT 5",
			valid:       true,
			description: "Set operation with parenthesized operand and trailing LIMIT is read-only",
		},
		{
			statement:   "EXPLAIN (SELECT 1)",
			valid:       true,
			description: "EXPLAIN over a parenthesized query is read-only",
		},
		{
			statement:   "EXPLAIN (SELECT 1 LIMIT 1) LIMIT 2",
			valid:       true,
			description: "EXPLAIN over a grouped query is read-only",
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
		{
			// Strict parsing (BYT-10085): a valid prefix followed by junk no
			// longer silently truncates — it is a syntax error. Before the
			// fix this parsed as `SELECT a FROM t` and ran.
			statement:   "SELECT a FROM t xx yy zz",
			description: "Trailing junk after a valid prefix must be rejected",
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

// A session SET is permitted but is not a read: validateQuery accepts it while
// reporting isExplicitReadOnly false, which is how the driver knows to run it with
// ExecContext instead of reading a result set. Doris had no case for *ast.SetStmt, so
// `SET time_zone = 'UTC'` was refused outright with "Support read-only command
// statements only" while MySQL, Postgres and TiDB all allowed it.
func TestValidateQuerySetStatements(t *testing.T) {
	for _, tc := range []struct {
		name             string
		statement        string
		wantValid        bool
		wantExplicitRead bool
	}{
		{"a session variable", "SET time_zone = 'UTC'", true, false},
		{"the @@ spelling", "SET @@time_zone = 'UTC'", true, false},
		{"an explicit session scope", "SET SESSION time_zone = 'UTC'", true, false},
		// Global scope is permitted here as it is for MySQL and Postgres: this is not
		// the permission gate, it only picks the execution path.
		{"a global scope", "SET GLOBAL time_zone = 'UTC'", true, false},
		{"SET NAMES", "SET NAMES utf8mb4", true, false},
		{"UNSET VARIABLE", "UNSET VARIABLE time_zone", true, false},
		{"a read stays explicitly read-only", "SELECT 1", true, true},
		{"a SET anywhere in a script needs executing", "SELECT 1; SET time_zone = 'UTC'", true, false},
		{"a SET does not launder a DDL beside it", "SET time_zone = 'UTC'; DROP TABLE t", false, false},
		{"SET PASSWORD administers the server", "SET PASSWORD FOR 'u' = 'x'", false, false},
		{"SET DEFAULT STORAGE VAULT administers the server", "SET DEFAULT STORAGE VAULT v", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			valid, explicitReadOnly, err := validateQuery(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantValid, valid)
			require.Equal(t, tc.wantExplicitRead, explicitReadOnly)
		})
	}
}

// TestValidateQueryFileExport pins the INTO OUTFILE gate. The clause writes
// the result set to S3 or HDFS, so an export is not a read however it is
// spelled — and the keyword appearing in a string, a quoted identifier or a
// comment is not an export at all. Both halves are read off the AST, which
// is what omni's Doris grammar gained the clause for (BOT-92).
func TestValidateQueryFileExport(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		wantValid bool
	}{
		// Every position the clause can occupy: it attaches to whichever
		// node is the statement's outermost query.
		{"plain", "SELECT 1 INTO OUTFILE '/tmp/x'", false},
		{"s3 with format and properties", `SELECT * FROM t INTO OUTFILE "s3://bucket/export/" FORMAT AS PARQUET PROPERTIES("s3.endpoint" = "example.com")`, false},
		{"after ORDER BY and LIMIT", "SELECT * FROM t ORDER BY a LIMIT 10 INTO OUTFILE '/tmp/x'", false},
		{"on a set operation", "SELECT a FROM t1 UNION SELECT a FROM t2 INTO OUTFILE '/tmp/x'", false},
		{"after parentheses", "(SELECT * FROM t) INTO OUTFILE '/tmp/x'", false},
		{"on a grouped query", "(SELECT 1 ORDER BY 1) ORDER BY 1 INTO OUTFILE '/tmp/x'", false},
		{"with a CTE", "WITH c AS (SELECT 1 AS a) SELECT * FROM c INTO OUTFILE '/tmp/x'", false},
		// The engine fails EXPLAIN of an export with the same broker/path
		// error as the export itself, so the clause is not inert under it.
		{"under EXPLAIN", "EXPLAIN SELECT 1 INTO OUTFILE '/tmp/x'", false},
		{"beside a read in one script", "SELECT 1; SELECT 2 INTO OUTFILE '/tmp/x'", false},

		// The keyword where it is not a clause. A text or token match could
		// not tell these from the real thing; the tree can.
		{"in a string literal", `SELECT a FROM t WHERE a = 'into outfile'`, true},
		{"in a quoted identifier", "SELECT `INTO OUTFILE` FROM t", true},
		{"in a block comment", "SELECT * FROM t /* INTO OUTFILE */", true},
		{"in a line comment", "SELECT * FROM t -- INTO OUTFILE 's3://bucket/'", true},
		{"a plain read", "SELECT * FROM t", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			valid, _, err := validateQuery(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantValid, valid)
		})
	}

	// MySQL's INTO DUMPFILE and INTO @var are not Doris syntax: they never
	// reach the gate, they fail to parse.
	for _, statement := range []string{
		"SELECT 1 INTO DUMPFILE '/tmp/x'",
		"SELECT 1 INTO @v",
		// The clause cannot appear inside a query, only after one.
		"SELECT * FROM (SELECT a FROM t INTO OUTFILE '/tmp/x') x",
		"SELECT 1 UNION (SELECT a FROM t INTO OUTFILE '/tmp/x')",
		// A statement that will not lex was refused as a possible export
		// while the gate read tokens; it is now reported as the syntax
		// error it is.
		"SELECT 'unterminated",
	} {
		t.Run("syntax error: "+statement, func(t *testing.T) {
			_, _, err := validateQuery(statement)
			require.Error(t, err)
		})
	}
}
