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
	}

	for _, tc := range testCases {
		got, err := getStatementWithResultLimitForMySQL(tc.stmt, tc.count)
		require.NoError(t, err, tc.stmt)
		require.Equal(t, tc.want, got, tc.stmt)
	}
}
