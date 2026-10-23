package pg

import (
	"context"
	"slices"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/ast"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func predicateColumnNames(columns base.SourceColumnSet) []string {
	names := []string{}
	for column := range columns {
		if column.Column != "" {
			names = append(names, column.Table+"."+column.Column)
		}
	}
	slices.Sort(names)
	return names
}

func TestGetQuerySpanPredicateColumns(t *testing.T) {
	testCases := []struct {
		name      string
		statement string
		want      []string
	}{
		{"where", "SELECT id FROM t WHERE ssn = '1'", []string{"t.ssn"}},
		{"where function", "SELECT id FROM t WHERE ascii(substr(ssn, 1, 1)) > 70", []string{"t.ssn"}},
		{"row comparison", "SELECT id FROM t WHERE (ssn, id) > ('1', 0)", []string{"t.id", "t.ssn"}},
		{"join on", "SELECT o.amt FROM o JOIN t ON t.id = o.id AND t.ssn = '1'", []string{"o.id", "t.id", "t.ssn"}},
		{"left join on", "SELECT o.amt FROM o LEFT JOIN t ON t.id = o.id AND t.ssn = '1'", []string{"o.id", "t.id", "t.ssn"}},
		{"join using", "SELECT a.id FROM t a JOIN t b USING (ssn)", []string{"t.ssn"}},
		{"having", "SELECT id FROM t GROUP BY id HAVING max(ssn) > '1'", []string{"t.ssn"}},
		{"in subquery", "SELECT amt FROM o WHERE id IN (SELECT id FROM t WHERE ssn = '1')", []string{"o.id", "t.id", "t.ssn"}},
		{"subquery output", "SELECT amt FROM o WHERE '1' = (SELECT max(ssn) FROM t)", []string{"t.ssn"}},
		{"correlated exists", "SELECT amt FROM o WHERE EXISTS (SELECT 1 FROM t WHERE t.id = o.id AND t.ssn = '1')", []string{"o.id", "t.id", "t.ssn"}},
		{"subquery in select list", "SELECT (SELECT count(*) FROM t WHERE ssn = '1') AS c", []string{"t.ssn"}},
		{"derived table", "SELECT id FROM (SELECT * FROM t WHERE ssn = '1') s", []string{"t.ssn"}},
		{"derived table output", "SELECT s.id FROM (SELECT id, ssn AS x FROM t) s WHERE s.x = '1'", []string{"t.ssn"}},
		{"lateral subquery", "SELECT o.amt FROM o, LATERAL (SELECT 1 FROM t WHERE t.id = o.id AND t.ssn = '1') l", []string{"o.id", "t.id", "t.ssn"}},
		{"cte", "WITH c AS (SELECT id FROM t WHERE ssn = '1') SELECT id FROM c", []string{"t.ssn"}},
		{"cte output", "WITH c AS (SELECT id, ssn FROM t) SELECT id FROM c WHERE ssn = '1'", []string{"t.ssn"}},
		{"set operation arm", "SELECT id FROM t WHERE ssn = '1' UNION SELECT id FROM o", []string{"t.ssn"}},
		{"except filters by the other branch", "SELECT 'guess' AS x EXCEPT SELECT ssn FROM t", []string{"t.ssn"}},
		{"intersect filters by the other branch", "SELECT 'guess' AS x INTERSECT ALL SELECT ssn FROM t", []string{"t.ssn"}},
		{"union filters nothing", "SELECT 'guess' AS x UNION SELECT ssn FROM t", []string{}},
		{"explain analyze", "EXPLAIN ANALYZE SELECT id FROM t WHERE ssn = '1'", []string{"t.ssn"}},
		{"aggregate filter", "SELECT sum(id) FILTER (WHERE ssn = '1') AS x FROM t", []string{"t.ssn"}},
		{"natural join", "SELECT 1 FROM t NATURAL JOIN (SELECT '1'::text AS ssn) x", []string{"t.ssn"}},
		{"full join using", "SELECT id FROM t FULL JOIN o USING (id) WHERE id > 0", []string{"o.id", "t.id"}},
		{"limit subquery", "SELECT id FROM t LIMIT (SELECT count(*) FROM t WHERE ssn = '1')", []string{"t.ssn"}},
		{"unknown function", "SELECT id FROM t WHERE levenshtein(ssn, 'a') > 0", []string{"t.ssn"}},
		{"unknown function in from", "SELECT t.id FROM t, LATERAL unknown_rows(t.ssn) f WHERE f.a = 1", []string{"t.ssn"}},
		{"select list only", "SELECT ssn FROM t", []string{}},
		{"condition in select list", "SELECT CASE WHEN ssn = '1' THEN 1 END AS x FROM t", []string{}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			span := spanFor(t, tc.statement, healthySchema())
			require.Equal(t, tc.want, predicateColumnNames(span.PredicateColumns))
		})
	}
}

func TestGetQuerySpanPredicateColumnsOfFunctionInFrom(t *testing.T) {
	schema := healthySchema()
	schema.Schemas[0].Functions = []*metadatapb.FunctionMetadata{
		{Name: "probe", Definition: "CREATE FUNCTION public.probe(guess text) RETURNS TABLE(id int) LANGUAGE plpgsql AS $$ BEGIN RETURN QUERY SELECT t.id FROM t WHERE t.ssn = guess; END; $$"},
		{Name: "probe_sql", Definition: "CREATE FUNCTION public.probe_sql(guess text) RETURNS SETOF int LANGUAGE sql AS $$ SELECT id FROM t WHERE ssn = guess $$"},
	}
	for _, statement := range []string{
		"SELECT count(*) AS c FROM probe('guess')",
		"SELECT count(*) AS c FROM probe_sql('guess')",
	} {
		t.Run(statement, func(t *testing.T) {
			span := spanFor(t, statement, schema)
			require.Equal(t, []string{"t.ssn"}, predicateColumnNames(span.PredicateColumns))
		})
	}
}

func TestGetQuerySpanResultLineage(t *testing.T) {
	testCases := []struct {
		name      string
		statement string
		want      []string
	}{
		{"set operation in subquery", "SELECT s.a FROM (SELECT ssn AS a FROM t UNION SELECT email FROM t) s", []string{"t.email", "t.ssn"}},
		{"set operation in cte", "WITH c AS (SELECT ssn FROM t UNION SELECT email FROM t) SELECT ssn FROM c", []string{"t.email", "t.ssn"}},
		{"except returns left branch rows", "SELECT s.a FROM (SELECT ssn AS a FROM t EXCEPT SELECT email FROM t) s", []string{"t.ssn"}},
		{"window before column in subquery", "SELECT s.x FROM (SELECT row_number() OVER (ORDER BY ssn) AS rn, email AS x FROM t) s", []string{"t.email"}},
		{"full join using column", "SELECT id FROM t FULL JOIN o USING (id)", []string{"o.id", "t.id"}},
		{"aggregate order by", "SELECT string_agg(email, ',' ORDER BY ssn) AS x FROM t", []string{"t.email", "t.ssn"}},
		{"unknown function in from", "SELECT f.a FROM t CROSS JOIN LATERAL unknown_rows(t.ssn) f", []string{"t.ssn"}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			span := spanFor(t, tc.statement, healthySchema())
			require.Len(t, span.Results, 1)
			require.Equal(t, tc.want, predicateColumnNames(span.Results[0].SourceColumns))
		})
	}
}

func newFallbackTestExtractor(t *testing.T) *omniQuerySpanExtractor {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{healthySchema()})
	e := newOmniQuerySpanExtractor("db", nil, base.GetQuerySpanContext{GetDatabaseMetadataFunc: getter, ListDatabaseNamesFunc: lister})
	e.ctx = context.Background()
	require.NoError(t, e.initCatalog())
	return e
}

func TestCollectFallbackPredicateColumns(t *testing.T) {
	testCases := []struct {
		name      string
		statement string
		want      []string
	}{
		{"where", "SELECT id FROM t WHERE ssn = '1'", []string{"t.ssn"}},
		{"qualified reference matches its table", "SELECT o.amt FROM o JOIN t ON o.id = t.id WHERE o.amt > 0", []string{"o.amt", "o.id", "t.id"}},
		{"unqualified reference matches every table", "SELECT 1 FROM o JOIN t USING (id) WHERE id > 0", []string{"o.id", "t.id"}},
		{"reused alias matches every table it names", "SELECT 1 FROM t x WHERE EXISTS (SELECT 1 FROM o x WHERE x.id = 1)", []string{"o.id", "t.id"}},
		{"subquery in a condition reads every name in it", "SELECT 1 FROM o WHERE EXISTS (WITH c AS (SELECT id FROM t WHERE ssn = '1') SELECT 1 FROM c)", []string{"o.id", "t.id", "t.ssn"}},
		{"cte column alias", "WITH c(a, b) AS (SELECT id, ssn FROM t) SELECT a FROM c WHERE b = '1'", []string{"t.ssn"}},
		{"recursive cte column alias", "WITH RECURSIVE c(a, n) AS (SELECT ssn, 1 FROM t UNION ALL SELECT a, n + 1 FROM c WHERE n < 3) SELECT a FROM c", []string{}},
		{"derived column alias", "SELECT 1 FROM (SELECT id, ssn FROM t) s(a, b) WHERE s.b = '1'", []string{"t.ssn"}},
		{"output name", "SELECT 1 FROM (SELECT ssn AS x FROM t) s WHERE x = '1'", []string{"t.ssn"}},
		{"whole row", "SELECT 1 FROM t WHERE t::text LIKE '%1%'", []string{"t.email", "t.id", "t.ssn"}},
		{"aliased whole row", "SELECT 1 FROM t x WHERE ROW(x.*) IS NOT NULL", []string{"t.email", "t.id", "t.ssn"}},
		{"join using", "SELECT 1 FROM t a JOIN t b USING (ssn)", []string{"t.ssn"}},
		{"having", "SELECT id FROM t GROUP BY id HAVING max(ssn) > '1'", []string{"t.ssn"}},
		{"aggregate filter", "SELECT count(*) FILTER (WHERE ssn = '1') FROM t", []string{"t.ssn"}},
		{"natural join reads both sides", "SELECT 1 FROM o NATURAL JOIN (SELECT id FROM t) x", []string{"o.amt", "o.id", "t.id"}},
		{"select list only", "SELECT ssn FROM t", []string{}},
		{"except filters by the other branch", "SELECT 'guess' EXCEPT SELECT ssn FROM t", []string{"t.ssn"}},
		{"intersect over star", "SELECT 1, 'a', 'b' INTERSECT SELECT * FROM t", []string{"t.email", "t.id", "t.ssn"}},
		{"union filters nothing", "SELECT 'guess' UNION SELECT ssn FROM t", []string{}},
	}
	accesses := base.SourceColumnSet{
		{Database: "db", Schema: "public", Table: "t"}: true,
		{Database: "db", Schema: "public", Table: "o"}: true,
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := newFallbackTestExtractor(t)
			stmts, err := ParsePg(tc.statement)
			require.NoError(t, err)
			e.collectFallbackPredicateColumns(stmts[0].AST, accesses)
			require.Equal(t, tc.want, predicateColumnNames(e.predicateColumns))
		})
	}
}

func TestGetQuerySpanFallbackKeepsPredicateColumns(t *testing.T) {
	// * over the unknown columns of a function fails omni's analysis.
	statement := "SELECT * FROM o JOIN t ON t.id = o.id AND t.ssn = '1' CROSS JOIN unknown_rows() f"
	stmts, err := ParsePg(statement)
	require.NoError(t, err)
	_, err = newFallbackTestExtractor(t).analyzeSelect(stmts[0].AST.(*ast.SelectStmt))
	require.Error(t, err, "the statement must fail analysis to reach the fallback")

	span := spanFor(t, statement, healthySchema())
	require.Equal(t, []string{"o.id", "t.id", "t.ssn"}, predicateColumnNames(span.PredicateColumns))
}
