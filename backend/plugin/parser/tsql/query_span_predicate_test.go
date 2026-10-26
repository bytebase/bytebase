package tsql

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func predicateColumnNames(columns base.SourceColumnSet) []string {
	names := []string{}
	for column := range columns {
		names = append(names, column.Table+"."+column.Column)
	}
	slices.Sort(names)
	return names
}

func TestOmniQuerySpanPredicateColumns(t *testing.T) {
	testCases := []struct {
		statement string
		want      []string
	}{
		{"SELECT a FROM t WHERE b = 1", []string{"t.b"}},
		{"SELECT t.a FROM t JOIN t1 ON t.b = t1.b", []string{"t.b", "t1.b"}},
		{"SELECT t.a FROM t LEFT JOIN t1 ON t1.a = t.a AND t1.c = 'x'", []string{"t.a", "t1.a", "t1.c"}},
		{"SELECT t.a FROM t FULL JOIN t1 ON t1.c = 'x'", []string{"t1.c"}},
		{"SELECT t.a FROM t JOIN t1 ON t.a = t1.a JOIN t2 ON t2.b = t1.b", []string{"t.a", "t1.a", "t1.b", "t2.b"}},
		{"SELECT t.a FROM t JOIN (t1 JOIN t2 ON t1.b = t2.b) ON t.a = t1.a", []string{"t.a", "t1.a", "t1.b", "t2.b"}},
		{"SELECT t.a FROM t JOIN t1 JOIN t2 ON t1.b = t2.b ON t.a = t1.a", []string{"t.a", "t1.a", "t1.b", "t2.b"}},
		{"SELECT t.a FROM t LEFT HASH JOIN t1 ON t.c = t1.c", []string{"t.c", "t1.c"}},
		{"SELECT x.a FROM t AS x JOIN t1 AS y ON x.c = y.c", []string{"t.c", "t1.c"}},
		{"SELECT t.a FROM t JOIN t1 ON t1.a = (SELECT MAX(t2.a) FROM t2 WHERE t2.b = 1)", []string{"t1.a", "t2.a", "t2.b"}},
		{"SELECT t.a FROM t JOIN t1 ON EXISTS (SELECT 1 FROM t2 WHERE t2.b = t1.b)", []string{"t1.b", "t2.b"}},
		{"SELECT d.a FROM (SELECT t.a FROM t JOIN t1 ON t.c = t1.c) AS d", []string{"t.c", "t1.c"}},
		{"WITH c AS (SELECT t.a FROM t JOIN t1 ON t.c = t1.c) SELECT a FROM c", []string{"t.c", "t1.c"}},
		{"SELECT a FROM t WHERE EXISTS (SELECT 1 FROM t1 JOIN t2 ON t1.b = t2.b WHERE t1.a = t.a)", []string{"t.a", "t1.a", "t1.b", "t2.b"}},
		{"SELECT a FROM t WHERE EXISTS (SELECT 1 FROM t1 JOIN t2 ON t2.a = t.c)", []string{"t.c", "t2.a"}},
		{"SELECT (SELECT MAX(t1.a) FROM t1 JOIN t2 ON t1.b = t2.b) AS m FROM t", []string{"t1.b", "t2.b"}},
		{"SELECT t.a FROM t CROSS JOIN t1", []string{}},
		{"SELECT x.a FROM t CROSS APPLY (SELECT t1.a FROM t1 WHERE t1.b = t.b) AS x", []string{"t.b", "t1.b"}},
		{"SELECT s.value FROM t CROSS APPLY STRING_SPLIT(t.c, ',') AS s", []string{"t.c"}},
		{"SELECT j.[key] FROM t OUTER APPLY OPENJSON(t.c) AS j", []string{"t.c"}},
		{"SELECT value FROM STRING_SPLIT((SELECT MAX(t1.c) FROM t1), ',')", []string{"t1.c"}},
		{"SELECT TOP ((SELECT MAX(t1.a) FROM t1)) a FROM t", []string{"t1.a"}},
		{"SELECT TOP ((SELECT COUNT(*) FROM t1 WHERE t1.c = 'x')) a FROM t", []string{"t1.c"}},
		{"SELECT a FROM t ORDER BY a OFFSET (SELECT MAX(t1.a) FROM t1) ROWS FETCH NEXT (SELECT MAX(t1.b) FROM t1) ROWS ONLY", []string{"t1.a", "t1.b"}},
		{"SELECT a FROM t UNION SELECT a FROM t1 ORDER BY a OFFSET (SELECT MAX(t2.a) FROM t2) ROWS", []string{"t2.a"}},
		{"SELECT a FROM t ORDER BY CASE WHEN EXISTS (SELECT 1 FROM t1 WHERE t1.c = 'x') THEN 0 ELSE 1 END", []string{"t1.c"}},
		{"SELECT a FROM t INTERSECT SELECT a FROM t1", []string{"t.a", "t1.a"}},
		{"SELECT a FROM t EXCEPT SELECT b FROM t1", []string{"t.a", "t1.b"}},
		{"SELECT d.a FROM (SELECT a FROM t INTERSECT SELECT a FROM t1) AS d", []string{"t.a", "t1.a"}},
		{"WITH c AS (SELECT a FROM t EXCEPT SELECT b FROM t1) SELECT a FROM c", []string{"t.a", "t1.b"}},
		{"SELECT a FROM t UNION SELECT a FROM t1", []string{}},
		{"SELECT a FROM t UNION ALL SELECT a FROM t1", []string{}},
		{"SELECT a FROM t ORDER BY b", []string{}},
		{"SELECT a AS x FROM t ORDER BY x", []string{}},
		{"SELECT b, COUNT(*) AS n FROM t GROUP BY b", []string{}},
	}
	for _, tc := range testCases {
		t.Run(tc.statement, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			span, err := q.getOmniQuerySpan(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Nil(t, span.NotFoundError)
			require.Equal(t, tc.want, predicateColumnNames(span.PredicateColumns))
		})
	}
}

func TestOmniQuerySpanJoinConditionKeepsLineage(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	span, err := q.getOmniQuerySpan(context.Background(), "SELECT t.a, t1.b FROM t JOIN t1 ON t.c = t1.c")
	require.NoError(t, err)
	require.Len(t, span.Results, 2)
	require.Equal(t, []string{"t.a"}, predicateColumnNames(span.Results[0].SourceColumns))
	require.Equal(t, []string{"t1.b"}, predicateColumnNames(span.Results[1].SourceColumns))
}

func TestOmniQuerySpanRefusesUnsupportedTableSources(t *testing.T) {
	for _, statement := range []string{
		"SELECT * FROM t JOIN db2.dbo.t2 AS u ON t.a = u.x PIVOT (MAX(y) FOR b IN ([1])) AS p",
		"SELECT j.v FROM t CROSS APPLY OPENJSON(t.c) WITH (v NVARCHAR(100) '$.v') AS j",
	} {
		t.Run(statement, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			_, err := q.getOmniQuerySpan(context.Background(), statement)
			require.Error(t, err)
		})
	}
}

func TestOmniQuerySpanJoinConditionNotFoundPropagates(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	span, err := q.getOmniQuerySpan(context.Background(), "SELECT t.a FROM t JOIN t1 ON t1.a IN (SELECT x FROM no_such_table)")
	require.NoError(t, err)
	require.NotNil(t, span.NotFoundError)
}
