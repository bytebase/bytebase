package starrocks

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
			want:  "SELECT firstName, lastName FROM employees UNION SELECT contactFirstName, contactLastName FROM customers LIMIT 10;",
		}, {
			stmt:  "SELECT customerNumber, checkNumber, amount FROM payments WHERE amount = (SELECT MAX(amount) FROM payments);",
			count: 10,
			want:  "SELECT customerNumber, checkNumber, amount FROM payments WHERE amount = (SELECT MAX(amount) FROM payments) LIMIT 10;",
		}, {
			stmt:  "SELECT firstName, lastName FROM employees UNION SELECT contactFirstName, contactLastName FROM customers LIMIT 10;",
			count: 10,
			want:  "SELECT firstName, lastName FROM employees UNION SELECT contactFirstName, contactLastName FROM customers LIMIT 10;",
		}, {
			stmt:  "WITH RECURSIVE cte_count (n) AS ( SELECT 1 UNION ALL SELECT n + 1 FROM cte_count WHERE n < 3 ) SELECT n FROM cte_count;",
			count: 10,
			want:  "WITH RECURSIVE cte_count (n) AS ( SELECT 1 UNION ALL SELECT n + 1 FROM cte_count WHERE n < 3 ) SELECT n FROM cte_count LIMIT 10;",
		},
		{
			stmt:  "SELECT firstName FROM employees EXCEPT SELECT contactFirstName FROM customers;",
			count: 10,
			want:  "SELECT firstName FROM employees EXCEPT SELECT contactFirstName FROM customers LIMIT 10;",
		},
		{
			stmt:  "SELECT col1, col2 FROM table1 PROCEDURE ANALYSE(10, 2000);",
			count: 10,
			want:  "SELECT * FROM (SELECT col1, col2 FROM table1 PROCEDURE ANALYSE(10, 2000)) result LIMIT 10;",
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
			stmt:  "SHOW DATA FROM db1",
			count: 10,
			want:  "SHOW DATA FROM db1",
		},
		{
			stmt:  "SHOW DATABASES",
			count: 10,
			want:  "SHOW DATABASES",
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
			stmt:  `SELECT * FROM orders INTO OUTFILE "s3://bucket/export/" FORMAT AS PARQUET PROPERTIES("s3.endpoint" = "s3.amazonaws.com");`,
			count: 10,
			want:  `SELECT * FROM orders LIMIT 10 INTO OUTFILE "s3://bucket/export/" FORMAT AS PARQUET PROPERTIES("s3.endpoint" = "s3.amazonaws.com");`,
		},
		{
			stmt:  "SELECT * FROM person LATERAL VIEW EXPLODE(ARRAY(30, 60)) tableName AS c_age;",
			count: 10,
			want:  "SELECT * FROM (SELECT * FROM person LATERAL VIEW EXPLODE(ARRAY(30, 60)) tableName AS c_age) result LIMIT 10;",
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
		{
			stmt:  "SELECT * FROM t;  \n",
			count: 10,
			want:  "SELECT * FROM t LIMIT 10;  \n",
		},
		// UNION with existing outer LIMIT — rewrite if larger
		{
			stmt:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 100;",
			count: 10,
			want:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 10;",
		},
		// UNION with existing outer LIMIT — keep if smaller
		{
			stmt:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 5;",
			count: 10,
			want:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 5;",
		},
		// INTERSECT with no LIMIT
		{
			stmt:  "SELECT a FROM t1 INTERSECT SELECT b FROM t2;",
			count: 10,
			want:  "SELECT a FROM t1 INTERSECT SELECT b FROM t2 LIMIT 10;",
		},
		// Parenthesized query with branch-local LIMIT — outer LIMIT added
		{
			stmt:  "SELECT a FROM t1 UNION (SELECT b FROM t2 LIMIT 1);",
			count: 10,
			want:  "SELECT a FROM t1 UNION (SELECT b FROM t2 LIMIT 1) LIMIT 10;",
		},
		// INTO OUTFILE with existing LIMIT
		{
			stmt:  `SELECT * FROM orders LIMIT 100 INTO OUTFILE "s3://bucket/export/";`,
			count: 10,
			want:  `SELECT * FROM orders LIMIT 10 INTO OUTFILE "s3://bucket/export/";`,
		},
		// Comma-style LIMIT on UNION
		{
			stmt:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 0,1000000;",
			count: 10,
			want:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 0,10;",
		},
		{
			stmt:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 0 /* offset comment */, 1000000;",
			count: 10,
			want:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 0 /* offset comment */, 10;",
		},
		// Comma-style LIMIT on UNION — keep if smaller
		{
			stmt:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 0,5;",
			count: 10,
			want:  "SELECT a FROM t1 UNION SELECT b FROM t2 LIMIT 0,5;",
		},
		// Comment followed by unparsed content should still trigger CTE fallback
		{
			stmt:  "SELECT * FROM person /* c */ LATERAL VIEW EXPLODE(ARRAY(30, 60)) tableName AS c_age;",
			count: 10,
			want:  "SELECT * FROM (SELECT * FROM person /* c */ LATERAL VIEW EXPLODE(ARRAY(30, 60)) tableName AS c_age) result LIMIT 10;",
		},
	}

	for _, tc := range testCases {
		got := getStatementWithResultLimit(tc.stmt, tc.count)
		require.Equal(t, tc.want, got, tc.stmt)
	}
}
