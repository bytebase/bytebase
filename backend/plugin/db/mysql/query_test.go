package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetStatementWithResultLimitOfTiDB(t *testing.T) {
	testCases := []struct {
		stmt  string
		count int
		want  string
	}{
		{
			stmt:  "SELECT * FROM t;",
			count: 10,
			want:  "SELECT * FROM `t` LIMIT 10",
		},
		{
			stmt:  "WITH a AS (SELECT * FROM t) SELECT * FROM a;",
			count: 10,
			want:  "WITH `a` AS (SELECT * FROM `t`) SELECT * FROM `a` LIMIT 10",
		},
		{
			stmt:  "SELECT * FROM t1 UNION SELECT * FROM t2;",
			count: 10,
			want:  "SELECT * FROM `t1` UNION SELECT * FROM `t2` LIMIT 10",
		},
		{
			stmt:  "SELECT * FROM t1 INTERSECT SELECT * FROM t2;",
			count: 10,
			want:  "SELECT * FROM `t1` INTERSECT SELECT * FROM `t2` LIMIT 10",
		},
		{
			stmt: "SELECT * FROM t LIMIT 5;",
			// If the statement already has limit clause, we will return the original statement.
			count: 10,
			want:  "SELECT * FROM t LIMIT 5;",
		},
	}

	for _, tc := range testCases {
		got, err := getStatementWithResultLimitForTiDB(tc.stmt, tc.count)
		require.NoError(t, err, tc.stmt)
		require.Equal(t, tc.want, got, tc.stmt)
	}
}

func TestGetStatementWithResultLimitOfMySQL(t *testing.T) {
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
			stmt:  "SELECT * FROM t LIMIT 5;",
			count: 10,
			want:  "SELECT * FROM t LIMIT 5;",
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
		// EXCEPT need mysql >= 8.0.31
		// {
		// 	stmt:  "SELECT firstName FROM employees EXCEPT SELECT contactFirstName FROM customers;",
		// 	count: 20,
		// 	want:  "SELECT firstName FROM employees EXCEPT SELECT contactFirstName FROM customers LIMIT 10;",
		// },
		{
			stmt:  "SELECT col1, col2 FROM table1 PROCEDURE ANALYSE(10, 2000);",
			count: 10,
			want:  "SELECT col1, col2 FROM table1 LIMIT 10 PROCEDURE ANALYSE(10, 2000);",
		},
		{
			stmt:  "SELECT col1, col2 FROM table1 ORDER BY col1;",
			count: 10,
			want:  "SELECT col1, col2 FROM table1 ORDER BY col1 LIMIT 10;",
		},
	}

	for _, tc := range testCases {
		got, err := getStatementWithResultLimitForMySQL(tc.stmt, tc.count)
		require.NoError(t, err, tc.stmt)
		require.Equal(t, tc.want, got, tc.stmt)
	}
}
