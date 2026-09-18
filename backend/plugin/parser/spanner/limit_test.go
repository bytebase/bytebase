package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetStatementWithResultLimit(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      string
	}{
		{
			name:      "select",
			statement: "SELECT * FROM t",
			want:      "SELECT * FROM t LIMIT 100",
		},
		{
			name:      "with",
			statement: "WITH a AS (SELECT 1 AS x) SELECT x FROM a",
			want:      "WITH a AS (SELECT 1 AS x) SELECT x FROM a LIMIT 100",
		},
		{
			name:      "lowercase with, order by, and trailing semicolon",
			statement: "with a as (select 1 x), b as (select x from a) select x from b order by x desc;",
			want:      "with a as (select 1 x), b as (select x from a) select x from b order by x desc LIMIT 100;",
		},
		{
			name:      "with and set operation",
			statement: "WITH a AS (SELECT 1 AS x) (SELECT x FROM a) UNION ALL (SELECT x FROM a) ORDER BY x",
			want:      "WITH a AS (SELECT 1 AS x) (SELECT x FROM a) UNION ALL (SELECT x FROM a) ORDER BY x LIMIT 100",
		},
		{
			name:      "set operation",
			statement: "SELECT 1 UNION ALL SELECT 2",
			want:      "SELECT 1 UNION ALL SELECT 2 LIMIT 100",
		},
		{
			name:      "inner limit belongs to the parenthesized query",
			statement: "(SELECT x FROM t LIMIT 500)",
			want:      "(SELECT x FROM t LIMIT 500) LIMIT 100",
		},
		{
			name:      "trailing comment",
			statement: "SELECT x FROM t -- note",
			want:      "SELECT x FROM t LIMIT 100 -- note",
		},
		{
			name:      "lower limit is kept",
			statement: "WITH a AS (SELECT x FROM t) SELECT x FROM a LIMIT 10",
			want:      "WITH a AS (SELECT x FROM t) SELECT x FROM a LIMIT 10",
		},
		{
			name:      "higher limit is lowered",
			statement: "WITH a AS (SELECT x FROM t) SELECT x FROM a ORDER BY x LIMIT 5000 OFFSET 10",
			want:      "WITH a AS (SELECT x FROM t) SELECT x FROM a ORDER BY x LIMIT 100 OFFSET 10",
		},
		{
			name:      "non-literal limit is unchanged",
			statement: "WITH a AS (SELECT x FROM t) SELECT x FROM a LIMIT CAST(5000 AS INT64)",
			want:      "WITH a AS (SELECT x FROM t) SELECT x FROM a LIMIT CAST(5000 AS INT64)",
		},
		{
			name:      "for update is unchanged",
			statement: "SELECT x FROM t ORDER BY x FOR UPDATE",
			want:      "SELECT x FROM t ORDER BY x FOR UPDATE",
		},
		{
			name:      "unparsable query is unchanged",
			statement: "WITH a AS (SELECT x FROM t) SELECT x FROM a |> WHERE x > 1",
			want:      "WITH a AS (SELECT x FROM t) SELECT x FROM a |> WHERE x > 1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, statementWithResultLimit(tc.statement, 100, ""))
		})
	}
}
