package tsql

import (
	"slices"
	"testing"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/stretchr/testify/require"
)

func TestCollectOmniSelectPredicateColumnRefs(t *testing.T) {
	type testCase struct {
		name string
		sql  string
		// want lists the expected column names (ColumnRef.Column) reachable
		// from the SelectStmt's predicate contexts, order-insensitive.
		want []string
	}
	cases := []testCase{
		{"simple_eq", "SELECT * FROM t WHERE a = 1", []string{"a"}},
		{"col_vs_col", "SELECT * FROM t WHERE a > b", []string{"a", "b"}},
		{"and_or", "SELECT * FROM t WHERE a = 1 AND (b = 2 OR c = 3)", []string{"a", "b", "c"}},
		{"not", "SELECT * FROM t WHERE NOT (a > 0)", []string{"a"}},
		{"between", "SELECT * FROM t WHERE a BETWEEN b AND c", []string{"a", "b", "c"}},
		{"like", "SELECT * FROM t WHERE a LIKE b", []string{"a", "b"}},
		{"is_null", "SELECT * FROM t WHERE a IS NULL", []string{"a"}},
		{"in_literal_list", "SELECT * FROM t WHERE a IN (1, 2, 3)", []string{"a"}},
		{"in_subquery", "SELECT * FROM t WHERE a IN (SELECT c FROM t1 WHERE d = 1)", []string{"a", "d"}},
		{"exists", "SELECT * FROM t WHERE EXISTS (SELECT 1 FROM t1 WHERE d = e)", []string{"d", "e"}},
		{"contains_predicate", "SELECT * FROM t WHERE CONTAINS(a, 'foo')", []string{"a"}},
		{"freetext_predicate", "SELECT * FROM t WHERE FREETEXT(a, 'foo')", []string{"a"}},
		{"fulltext_star", "SELECT * FROM t WHERE CONTAINS(*, 'foo')", []string{}},
		{"case_expr", "SELECT * FROM t WHERE CASE WHEN a = 1 THEN b ELSE c END = 1", []string{"a", "b", "c"}},
		{"iif", "SELECT * FROM t WHERE IIF(a = 1, b, c) > 0", []string{"a", "b", "c"}},
		{"coalesce", "SELECT * FROM t WHERE COALESCE(a, b, c) = 0", []string{"a", "b", "c"}},
		{"nullif", "SELECT * FROM t WHERE NULLIF(a, b) IS NOT NULL", []string{"a", "b"}},
		{"cast_expr", "SELECT * FROM t WHERE CAST(a AS INT) > 5", []string{"a"}},
		{"convert_expr", "SELECT * FROM t WHERE CONVERT(INT, a) > 5", []string{"a"}},
		{"try_cast_expr", "SELECT * FROM t WHERE TRY_CAST(a AS INT) > 5", []string{"a"}},
		{"func_args", "SELECT * FROM t WHERE LEN(a) > LEN(b)", []string{"a", "b"}},
		{"paren_expr", "SELECT * FROM t WHERE ((a + b) > (c - d))", []string{"a", "b", "c", "d"}},
		{"collate_expr", "SELECT * FROM t WHERE a COLLATE Latin1_General_CS_AS = 'x'", []string{"a"}},
		// Note: JOIN ON is intentionally NOT collected by this helper (parity
		// with the legacy ANTLR extractor). Only WHERE/HAVING contribute.
		{"join_on", "SELECT * FROM t1 JOIN t2 ON t1.a = t2.b", []string{}},
		{"multi_join_on", "SELECT * FROM t1 JOIN t2 ON t1.a = t2.b JOIN t3 ON t2.c = t3.d", []string{}},
		{"having", "SELECT a, SUM(b) FROM t GROUP BY a HAVING SUM(b) > 10", []string{"b"}},
		{"correlated_subquery", "SELECT * FROM t1 WHERE EXISTS (SELECT 1 FROM t2 WHERE t2.a = t1.b)", []string{"a", "b"}},
		{"subquery_comparison_any", "SELECT * FROM t WHERE a > ANY (SELECT b FROM t1 WHERE c = 1)", []string{"a", "c"}},
		{"subquery_comparison_all", "SELECT * FROM t WHERE a <= ALL (SELECT b FROM t1 WHERE c = 1)", []string{"a", "c"}},
		{"nested_subquery", "SELECT * FROM t WHERE EXISTS (SELECT 1 FROM t1 WHERE EXISTS (SELECT 1 FROM t2 WHERE x = y))", []string{"x", "y"}},
		// The `c` inside MAX(c) lives in the subquery's SELECT list, not its WHERE,
		// so it is intentionally excluded by this helper's contract. Subquery OUTPUT
		// columns become outer predicate columns only after the extractor's
		// table-source resolution.
		{"scalar_subquery_in_where", "SELECT * FROM t WHERE (SELECT MAX(c) FROM t1 WHERE d = 1) > 5", []string{"d"}},
		{"no_predicate", "SELECT * FROM t", []string{}},
		{"set_op_union_both_have_where", "SELECT * FROM t WHERE a = 1 UNION SELECT * FROM t2 WHERE b = 2", []string{"a", "b"}},
		{"cte_body_where", "WITH c AS (SELECT * FROM t WHERE a = 1) SELECT * FROM c WHERE x = 2", []string{"a", "x"}},
		{"where_and_join_on", "SELECT * FROM t1 JOIN t2 ON t1.a = t2.b WHERE t1.c > 0", []string{"c"}},
		{"cross_apply_condition_none", "SELECT * FROM t1 CROSS APPLY fn(t1.a) AS x", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := ParseTSQLOmni(tc.sql)
			require.NoError(t, err, "parse failed for %s", tc.sql)
			require.NotEmpty(t, stmts)
			sel, ok := stmts[0].AST.(*ast.SelectStmt)
			require.Truef(t, ok, "want *SelectStmt, got %T", stmts[0].AST)

			refs := collectOmniSelectPredicateColumnRefs(sel)
			got := make([]string, 0, len(refs))
			for _, r := range refs {
				got = append(got, r.Column)
			}
			slices.Sort(got)
			want := append([]string{}, tc.want...)
			slices.Sort(want)
			require.ElementsMatch(t, want, got, "sql: %s", tc.sql)
		})
	}
}

func TestCollectOmniPredicateColumnRefs_QualifiedRefs(t *testing.T) {
	// Ensure qualified column refs (db.schema.table.col) are returned with
	// all qualifier fields intact so the extractor can resolve them later.
	sql := "SELECT * FROM db1.dbo.t WHERE db1.dbo.t.a = 1"
	stmts, err := ParseTSQLOmni(sql)
	require.NoError(t, err)
	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	require.True(t, ok)
	refs := collectOmniSelectPredicateColumnRefs(sel)
	require.Len(t, refs, 1)
	r := refs[0]
	require.Equal(t, "db1", r.Database)
	require.Equal(t, "dbo", r.Schema)
	require.Equal(t, "t", r.Table)
	require.Equal(t, "a", r.Column)
}

func TestCollectOmniPredicateColumnRefs_Directly(t *testing.T) {
	// Exercise the non-Select entry point: caller hands in a bare ExprNode.
	sql := "SELECT * FROM t WHERE a > b AND c IS NULL"
	stmts, err := ParseTSQLOmni(sql)
	require.NoError(t, err)
	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	require.True(t, ok)

	refs := collectOmniPredicateColumnRefs(sel.WhereClause)
	got := make([]string, 0, len(refs))
	for _, r := range refs {
		got = append(got, r.Column)
	}
	slices.Sort(got)
	require.ElementsMatch(t, []string{"a", "b", "c"}, got)
}
