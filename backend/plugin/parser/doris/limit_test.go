package doris

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetStatementWithResultLimit(t *testing.T) {
	testCases := []struct {
		stmt  string
		count int
		want  string
	}{
		{
			stmt:  "SELECT * FROM t;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 10;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 10;",
			count: 5,
			want:  "SELECT * FROM t LIMIT 5;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 5;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 5;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 123;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 10;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 10 OFFSET 123;",
			count: 5,
			want:  "SELECT * FROM t LIMIT 5 OFFSET 123;",
		},
		{
			stmt:  "SELECT * FROM t2 JOIN t1 ON t2.c2 = t1.c2 where t2.c2 > 10;",
			count: 10,
			want:  "SELECT * FROM t2 JOIN t1 ON t2.c2 = t1.c2 where t2.c2 > 10 LIMIT 10;",
		},
		{
			stmt:  "SELECT * FROM t1 ORDER BY c2;",
			count: 10,
			want:  "SELECT * FROM t1 ORDER BY c2 LIMIT 10;",
		},
		{
			stmt:  "SELECT COUNT(col1) AS col2 FROM t GROUP BY col2 HAVING col2 = 2;",
			count: 10,
			want:  "SELECT COUNT(col1) AS col2 FROM t GROUP BY col2 HAVING col2 = 2 LIMIT 10;",
		},
		{
			stmt:  "SELECT firstName, lastName FROM employees UNION SELECT contactFirstName, contactLastName FROM customers;",
			count: 10,
			// SetOpStmt.Limit is unreliable in the Doris grammar (see limit.go),
			// so a set operation always takes the CTE-wrap fallback.
			want: "SELECT * FROM (SELECT firstName, lastName FROM employees UNION SELECT contactFirstName, contactLastName FROM customers) result LIMIT 10;",
		},
		{
			stmt:  "SELECT customerNumber, checkNumber, amount FROM payments WHERE amount = (SELECT MAX(amount) FROM payments);",
			count: 10,
			want:  "SELECT customerNumber, checkNumber, amount FROM payments WHERE amount = (SELECT MAX(amount) FROM payments) LIMIT 10;",
		},
		{
			stmt:  "SELECT firstName, lastName FROM employees UNION SELECT contactFirstName, contactLastName FROM customers LIMIT 10;",
			count: 10,
			want:  "SELECT * FROM (SELECT firstName, lastName FROM employees UNION SELECT contactFirstName, contactLastName FROM customers LIMIT 10) result LIMIT 10;",
		},
		{
			// The outer statement is a SelectStmt (its WITH clause), so it still
			// gets the precise in-place edit despite the UNION ALL inside the CTE.
			stmt:  "WITH RECURSIVE cte_count (n) AS ( SELECT 1 UNION ALL SELECT n + 1 FROM cte_count WHERE n < 3 ) SELECT n FROM cte_count;",
			count: 10,
			want:  "WITH RECURSIVE cte_count (n) AS ( SELECT 1 UNION ALL SELECT n + 1 FROM cte_count WHERE n < 3 ) SELECT n FROM cte_count LIMIT 10;",
		},
		{
			stmt:  "SELECT firstName FROM employees EXCEPT SELECT contactFirstName FROM customers;",
			count: 10,
			want:  "SELECT * FROM (SELECT firstName FROM employees EXCEPT SELECT contactFirstName FROM customers) result LIMIT 10;",
		},
		{
			stmt:  "SELECT col1, col2 FROM table1 ORDER BY col1;",
			count: 10,
			want:  "SELECT col1, col2 FROM table1 ORDER BY col1 LIMIT 10;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 0,20;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 0,10;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 123,20;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 123,10;",
		},
		{
			// A literal 0 count is kept, not replaced with the requested cap.
			stmt:  "SELECT * FROM t LIMIT 5,0;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 5,0;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 0,\n1000000;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 0,\n10;",
		},
		{
			stmt:  "SELECT * FROM t LIMIT 0 /* offset comment */, 1000000;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 0 /* offset comment */, 10;",
		},
		// SHOW statements should not be wrapped
		{
			stmt:  "SHOW DATA",
			count: 10,
			want:  "SHOW DATA",
		},
		{
			stmt:  "SHOW TABLES",
			count: 10,
			want:  "SHOW TABLES",
		},
		{
			stmt:  "show data",
			count: 10,
			want:  "show data",
		},
		{
			stmt:  "  SHOW DATA  ",
			count: 10,
			want:  "  SHOW DATA  ",
		},
		{
			stmt:  "SELECT * FROM t -- note",
			count: 10,
			want:  "SELECT * FROM t LIMIT 10 -- note",
		},
		{
			stmt:  "SELECT * FROM t /* block comment */;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 10 /* block comment */;",
		},
		{
			stmt:  "SELECT * FROM t # hash comment",
			count: 10,
			want:  "SELECT * FROM t LIMIT 10 # hash comment",
		},
		{
			stmt:  "SELECT * FROM t;\n",
			count: 10,
			want:  "SELECT * FROM t LIMIT 10;\n",
		},
		// A set operation always takes the CTE-wrap fallback (see limit.go), so an
		// existing LIMIT on it — larger, smaller, or comma-form — is never capped
		// in place; the fallback's own LIMIT is still the one that binds.
		{
			stmt:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 100;",
			count: 10,
			want:  "SELECT * FROM (SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 100) result LIMIT 10;",
		},
		{
			stmt:  "SELECT a FROM t1 INTERSECT SELECT b FROM t2;",
			count: 10,
			want:  "SELECT * FROM (SELECT a FROM t1 INTERSECT SELECT b FROM t2) result LIMIT 10;",
		},
		// Doris has no dedicated node for a parenthesized top-level query: the
		// grammar strips the parens transparently and produces a plain
		// SelectStmt whose Loc always lands after the closing paren(s), so a
		// bare append at Loc.End still lands outside them and is valid SQL.
		{
			stmt:  "(SELECT * FROM t)",
			count: 10,
			want:  "(SELECT * FROM t) LIMIT 10",
		},
		{
			stmt:  "((SELECT * FROM t))",
			count: 10,
			want:  "((SELECT * FROM t)) LIMIT 10",
		},
		{
			stmt:  "(SELECT * FROM t) LIMIT 1000000",
			count: 10,
			want:  "(SELECT * FROM t) LIMIT 10",
		},
		{
			stmt:  "(SELECT * FROM t) ORDER BY 1",
			count: 10,
			want:  "(SELECT * FROM t) ORDER BY 1 LIMIT 10",
		},
	}

	for _, tc := range testCases {
		got := statementWithResultLimit(tc.stmt, tc.count, "")
		require.Equal(t, tc.want, got, tc.stmt)
	}
}
