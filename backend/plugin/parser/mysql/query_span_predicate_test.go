package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// predicateFixtureTables is the schema the span and the engines both see.
var predicateFixtureTables = []struct {
	name    string
	columns []string
	ddl     string
}{
	{name: "t", columns: []string{"a", "b", "c"}, ddl: "CREATE TABLE t (a INT, b INT, c INT)"},
	{name: "t1", columns: []string{"a", "b", "c"}, ddl: "CREATE TABLE t1 (a INT, b INT, c INT)"},
	{name: "u", columns: []string{"b", "c"}, ddl: "CREATE TABLE u (b INT, c INT)"},
	{name: "j", columns: []string{"id", "doc"}, ddl: "CREATE TABLE j (id INT, doc JSON)"},
}

// predicateCase is a statement with the predicate columns its span must
// record. TestQuerySpanPredicateColumnsMatchEngines also runs it on MySQL and
// MariaDB, where the columns the engine reads must be exactly the result
// lineage, the predicate columns, and other.
type predicateCase struct {
	name string
	// statement may name the database as {db}.
	statement string
	want      []string
	// results, if set, is each result column's lineage.
	results [][]string
	// other lists columns the engine reads to group or order rows, which are
	// neither result lineage nor predicates.
	other []string
	// unchecked lists predicate columns the engine compares without checking
	// the privilege to read them: those a USING or NATURAL join merges.
	unchecked []string
	// rejected means both engines refuse the statement.
	rejected bool
	// skip names an engine whose answer cannot be compared, and why.
	skip map[string]string
}

var (
	noMariaDBLateral      = map[string]string{"mariadb": "MariaDB has no LATERAL"}
	mariaDBReadsUnusedCTE = map[string]string{"mariadb": "MariaDB checks privileges in a CTE it never runs"}
)

var predicateCases = []predicateCase{
	// WHERE.
	{name: "where column", statement: "SELECT a FROM t WHERE b = 1", want: []string{"t.b"}, results: [][]string{{"t.a"}}},
	{name: "where function", statement: "SELECT a FROM t WHERE ascii(substr(c, 1, 1)) > 70", want: []string{"t.c"}},
	{name: "where row comparison", statement: "SELECT a FROM t WHERE (b, c) > (1, 2)", want: []string{"t.b", "t.c"}},
	{name: "where between in like", statement: "SELECT a FROM t WHERE b BETWEEN 1 AND c AND a IN (1, 2) AND c LIKE '1%'", want: []string{"t.a", "t.b", "t.c"}},
	{name: "where case cast", statement: "SELECT 1 FROM t WHERE CASE WHEN b > 1 THEN CAST(c AS CHAR) ELSE '' END = '1'", want: []string{"t.b", "t.c"}},
	{name: "where null-safe and regexp", statement: "SELECT 1 FROM t WHERE b <=> NULL OR c REGEXP '^1'", want: []string{"t.b", "t.c"}},
	{name: "where user variable", statement: "SELECT a FROM t WHERE b = @guess", want: []string{"t.b"}},
	{name: "where json path", statement: "SELECT id FROM j WHERE doc->>'$.k' = 'v'", want: []string{"j.doc"}, skip: map[string]string{"mariadb": "MariaDB has no ->> operator"}},
	{name: "where member of", statement: "SELECT id FROM j WHERE 1 MEMBER OF (doc->'$.arr')", want: []string{"j.doc"}, skip: map[string]string{"mariadb": "MariaDB has no MEMBER OF"}},
	{name: "where database-qualified column", statement: "SELECT t.a FROM t WHERE {db}.t.b = 1", want: []string{"t.b"}},
	{name: "where column in other case", statement: "SELECT A FROM t WHERE B = 1", want: []string{"t.b"}, results: [][]string{{"t.a"}}},
	{name: "where quoted identifiers", statement: "SELECT `a` FROM `t` WHERE `b` = 1", want: []string{"t.b"}},
	{name: "count with where", statement: "SELECT count(*) FROM t WHERE c = 0", want: []string{"t.c"}, results: [][]string{{}}},
	{name: "star with where", statement: "SELECT * FROM t WHERE b = 1", want: []string{"t.b"}, results: [][]string{{"t.a"}, {"t.b"}, {"t.c"}}},
	{name: "select list only", statement: "SELECT b FROM t", want: []string{}},
	{name: "condition in select list", statement: "SELECT CASE WHEN b = 1 THEN 1 END AS x FROM t", want: []string{}, results: [][]string{{"t.b"}}},
	{name: "limit values", statement: "SELECT a FROM t LIMIT 1 OFFSET 2", want: []string{}},

	// Joins.
	{name: "join on", statement: "SELECT t.a FROM t JOIN t1 ON t.b = t1.b", want: []string{"t.b", "t1.b"}},
	{name: "left join on", statement: "SELECT t.a FROM t LEFT JOIN t1 ON t1.a = t.a AND t1.c = 0", want: []string{"t.a", "t1.a", "t1.c"}},
	{name: "join using", statement: "SELECT t.a FROM t JOIN t1 USING (b)", want: []string{"t.b", "t1.b"}, unchecked: []string{"t.b", "t1.b"}},
	{name: "natural join", statement: "SELECT 1 FROM t NATURAL JOIN u", want: []string{"t.b", "t.c", "u.b", "u.c"}, unchecked: []string{"t.b", "t.c", "u.b", "u.c"}},
	{name: "right join using column", statement: "SELECT b FROM t RIGHT JOIN t1 USING (b) WHERE b = 99", want: []string{"t.b", "t1.b"}, results: [][]string{{"t.b", "t1.b"}}, unchecked: []string{"t.b"}},
	{name: "straight join", statement: "SELECT t.a FROM t STRAIGHT_JOIN t1 ON t.a = t1.a", want: []string{"t.a", "t1.a"}},
	{name: "nested join", statement: "SELECT 1 FROM t JOIN (t1 JOIN u ON t1.b = u.b) ON t.a = t1.a", want: []string{"t.a", "t1.a", "t1.b", "u.b"}},
	{name: "comma join", statement: "SELECT 1 FROM t, t1 WHERE t.a = t1.a", want: []string{"t.a", "t1.a"}},
	{name: "alias named like another table", statement: "SELECT x.a FROM t AS x JOIN t1 AS t ON x.a = t.a WHERE t.b = 1", want: []string{"t.a", "t1.a", "t1.b"}, results: [][]string{{"t.a"}}},
	{name: "correlated subquery in on", statement: "SELECT 1 FROM t JOIN t1 ON t1.a = (SELECT max(c) FROM u WHERE u.b = t.b)", want: []string{"t.b", "t1.a", "u.b", "u.c"}},
	{name: "cross join", statement: "SELECT t.a FROM t CROSS JOIN t1", want: []string{}, results: [][]string{{"t.a"}}},
	{name: "ambiguous column", statement: "SELECT b FROM t JOIN t1 ON t.a = t1.a", rejected: true},

	// Nested queries.
	{name: "in subquery", statement: "SELECT a FROM t WHERE a IN (SELECT a FROM t1 WHERE b > 0)", want: []string{"t.a", "t1.a", "t1.b"}},
	{name: "correlated exists", statement: "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM t1 WHERE t1.a = t.a AND t1.c = 0)", want: []string{"t.a", "t1.a", "t1.c"}},
	{name: "exists star is not read", statement: "SELECT a FROM t WHERE EXISTS (SELECT * FROM t1 WHERE t1.a = t.a)", want: []string{"t.a", "t1.a"}},
	{name: "exists output decides nothing", statement: "SELECT a FROM t WHERE EXISTS (SELECT b FROM t1 WHERE t1.a = t.a)", want: []string{"t.a", "t1.a"}, other: []string{"t1.b"}},
	{name: "scalar subquery", statement: "SELECT a FROM t WHERE b = (SELECT max(b) FROM t1)", want: []string{"t.b", "t1.b"}},
	{name: "any subquery", statement: "SELECT a FROM t WHERE b > ANY (SELECT b FROM t1 WHERE t1.c = 0)", want: []string{"t.b", "t1.b", "t1.c"}},
	{name: "subquery name resolves inside first", statement: "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM t1 WHERE b = 10)", want: []string{"t1.b"}},
	{name: "subquery name resolves outside", statement: "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM u WHERE u.b = a)", want: []string{"t.a", "u.b"}},
	{name: "two levels out", statement: "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM t1 WHERE EXISTS (SELECT 1 FROM u WHERE u.b = t.b AND u.c = t1.c))", want: []string{"t.b", "t1.c", "u.b", "u.c"}},
	{name: "innermost of two levels first", statement: "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM t1 WHERE EXISTS (SELECT 1 FROM u WHERE b = 1))", want: []string{"u.b"}},
	{name: "condition of select list subquery", statement: "SELECT (SELECT count(*) FROM t1 WHERE t1.c = 0) AS k FROM t", want: []string{"t1.c"}, results: [][]string{{}}},
	{name: "select list subquery resolves inside first", statement: "SELECT (SELECT b FROM t1 LIMIT 1) AS x FROM t", want: []string{}, results: [][]string{{"t1.b"}}},
	{name: "derived table", statement: "SELECT d.a FROM (SELECT a, c FROM t WHERE b = 1) d WHERE d.c = 0", want: []string{"t.b", "t.c"}, results: [][]string{{"t.a"}}},
	{name: "derived table column list", statement: "SELECT x FROM (SELECT a, c FROM t) d (x, y) WHERE y = 0", want: []string{"t.c"}, results: [][]string{{"t.a"}}},
	{name: "lateral", statement: "SELECT l.k FROM t, LATERAL (SELECT count(*) AS k FROM t1 WHERE t1.a = t.a) l", want: []string{"t.a", "t1.a"}, results: [][]string{{}}, skip: noMariaDBLateral},
	{name: "lateral in a join sees the left operand", statement: "SELECT x.b FROM t JOIN LATERAL (SELECT t.b) AS x ON TRUE", want: []string{}, results: [][]string{{"t.b"}}, skip: noMariaDBLateral},
	{name: "lateral in a join condition", statement: "SELECT x.k FROM t JOIN LATERAL (SELECT count(*) AS k FROM t1 WHERE t1.a = t.a) AS x ON TRUE", want: []string{"t.a", "t1.a"}, results: [][]string{{}}, skip: noMariaDBLateral},
	{name: "lateral after two joined tables", statement: "SELECT x.k FROM t JOIN t1 ON t.a = t1.a JOIN LATERAL (SELECT u.c AS k FROM u WHERE u.b = t1.b) AS x ON TRUE", want: []string{"t.a", "t1.a", "t1.b", "u.b"}, results: [][]string{{"u.c"}}, skip: noMariaDBLateral},
	{name: "derived table cannot see its siblings", statement: "SELECT 1 FROM t, (SELECT a FROM t1 WHERE t1.b = t.b) d", rejected: true},
	{name: "every values row", statement: "SELECT * FROM (VALUES ROW(1), ROW((SELECT b FROM t LIMIT 1))) AS v (x)", want: []string{}, results: [][]string{{"t.b"}}, skip: map[string]string{"mariadb": "MariaDB has no VALUES ROW"}},
	{name: "window in nested condition", statement: "SELECT a FROM t WHERE a IN (SELECT a FROM (SELECT a, ROW_NUMBER() OVER (ORDER BY b) AS r FROM t1) x WHERE r = 1)", want: []string{"t.a", "t1.a", "t1.b"}},

	// CTEs.
	{name: "cte", statement: "WITH c AS (SELECT a, b FROM t WHERE c = 0) SELECT a FROM c WHERE b = 1", want: []string{"t.b", "t.c"}, results: [][]string{{"t.a"}}},
	{name: "unread cte", statement: "WITH c AS (SELECT a FROM t WHERE b = 1) SELECT a FROM t1", want: []string{}, results: [][]string{{"t1.a"}}, skip: mariaDBReadsUnusedCTE},
	{name: "cte read in subquery", statement: "WITH c AS (SELECT a FROM t WHERE b = 1) SELECT 1 FROM t1 WHERE t1.a IN (SELECT a FROM c)", want: []string{"t.a", "t.b", "t1.a"}},
	{name: "cte read by cte", statement: "WITH c1 AS (SELECT a FROM t WHERE b = 1), c2 AS (SELECT a FROM c1) SELECT a FROM c2", want: []string{"t.b"}, results: [][]string{{"t.a"}}},
	{name: "cte unread by the other", statement: "WITH c1 AS (SELECT a FROM t WHERE b = 1), c2 AS (SELECT a FROM t1 WHERE c = 0) SELECT a FROM c2", want: []string{"t1.c"}, skip: mariaDBReadsUnusedCTE},
	{name: "recursive cte", statement: "WITH RECURSIVE r (n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM r WHERE n < (SELECT max(a) FROM t)) SELECT n FROM r", want: []string{"t.a"}, results: [][]string{{}}},

	// Set operations.
	{name: "union arm condition", statement: "SELECT a FROM t WHERE b = 1 UNION SELECT a FROM t1", want: []string{"t.b"}, results: [][]string{{"t.a", "t1.a"}}},
	{name: "union filters nothing", statement: "SELECT a FROM t UNION SELECT b FROM t1", want: []string{}, results: [][]string{{"t.a", "t1.b"}}},
	{name: "intersect", statement: "SELECT a FROM t INTERSECT SELECT b FROM t1", want: []string{"t.a", "t1.b"}},
	{name: "except", statement: "SELECT a FROM t EXCEPT SELECT b FROM t1", want: []string{"t.a", "t1.b"}},
	{name: "set operation limit", statement: "SELECT a FROM t UNION ALL SELECT a FROM t1 LIMIT 1", want: []string{}},
	{name: "parenthesized query", statement: "(SELECT a FROM t WHERE b = 1) ORDER BY a LIMIT 1", want: []string{"t.b"}},

	// HAVING.
	{name: "having alias of an aggregate", statement: "SELECT a, count(*) AS n FROM t GROUP BY a HAVING n > 1", want: []string{}, results: [][]string{{"t.a"}, {}}},
	{name: "having alias over a from column", statement: "SELECT c AS b FROM t HAVING b > 150", want: []string{"t.c"}, results: [][]string{{"t.c"}}},
	{name: "having group by column over an alias", statement: "SELECT c AS b FROM t GROUP BY b, c HAVING b > 150", want: []string{"t.b"}, skip: map[string]string{"mariadb": "MariaDB refuses the ambiguous name"}},
	{name: "having aggregate argument reads from", statement: "SELECT c AS b, count(*) AS k FROM t GROUP BY c HAVING max(b) > 150", want: []string{"t.b"}},
	{name: "having any_value reads the alias", statement: "SELECT c AS b FROM t GROUP BY c HAVING any_value(b) > 150", want: []string{"t.c"}, skip: map[string]string{"mariadb": "MariaDB has no ANY_VALUE"}},
	{name: "having group_concat order", statement: "SELECT a FROM t GROUP BY a HAVING group_concat(b ORDER BY c) <> ''", want: []string{"t.b", "t.c"}},
	{name: "having subquery sees from before alias", statement: "SELECT c AS b FROM u HAVING (SELECT b) > 150", want: []string{"u.b"}, results: [][]string{{"u.c"}}},
	{name: "having alias only inside aggregate", statement: "SELECT c AS x FROM u GROUP BY c HAVING max(x) > 1", want: []string{"u.c"}},
	{name: "having select item across join", statement: "SELECT t1.b FROM t JOIN t1 ON t.a = t1.a HAVING b > 5", want: []string{"t.a", "t1.a", "t1.b"}},
	{name: "having plain column", statement: "SELECT b FROM t HAVING b > 1", want: []string{"t.b"}},
	{name: "having qualified column", statement: "SELECT c AS b FROM t GROUP BY c HAVING t.c > 1", want: []string{"t.c"}},
	{name: "having subquery condition", statement: "SELECT a FROM t GROUP BY a HAVING a > (SELECT count(*) FROM t1 WHERE t1.c = 0)", want: []string{"t.a", "t1.c"}},
	{name: "having with group by position", statement: "SELECT c AS b FROM t GROUP BY 1 HAVING b > 150", want: []string{"t.c"}},
	{name: "having duplicate alias", statement: "SELECT b AS x, c AS x FROM u HAVING x > 2", rejected: true},

	// Select aliases elsewhere.
	{name: "select subquery sees an earlier alias", statement: "SELECT c AS x, (SELECT x + 1) AS s FROM u", want: []string{}, results: [][]string{{"u.c"}, {"u.c"}}},
	{name: "select subquery sees from before alias", statement: "SELECT c AS b, (SELECT b) AS s FROM u", want: []string{}, results: [][]string{{"u.c"}, {"u.b"}}},
	{name: "select subquery forward alias", statement: "SELECT (SELECT x + 1) AS s, c AS x FROM u", rejected: true},
	{name: "where subquery alias", statement: "SELECT c AS x FROM u WHERE (SELECT x) > 150", rejected: true},
	{name: "where alias", statement: "SELECT c AS x FROM u WHERE x > 1", rejected: true},
	{name: "group by subquery alias", statement: "SELECT c AS x, count(*) AS k FROM u GROUP BY (SELECT x)", want: []string{}, results: [][]string{{"u.c"}, {}}},
	{name: "group by key", statement: "SELECT count(*) FROM t GROUP BY b", want: []string{}, other: []string{"t.b"}},
	{name: "order by key", statement: "SELECT a FROM t ORDER BY c LIMIT 1", want: []string{}, other: []string{"t.c"}},
	{name: "order by subquery condition", statement: "SELECT a FROM t ORDER BY (SELECT max(b) FROM t1 WHERE t1.c = 0)", want: []string{"t1.c"}, other: []string{"t1.b"}},
	{name: "order by subquery sees from before alias", statement: "SELECT c AS b FROM u ORDER BY (SELECT b) LIMIT 1", want: []string{}, other: []string{"u.b"}},
	{name: "window keys", statement: "SELECT a, ROW_NUMBER() OVER (PARTITION BY b ORDER BY c) AS r FROM t", want: []string{}, results: [][]string{{"t.a"}, {"t.b", "t.c"}}},
	{name: "window clause subquery", statement: "SELECT a, ROW_NUMBER() OVER w AS r FROM t WINDOW w AS (ORDER BY (SELECT max(b) FROM t1 WHERE t1.c = 0))", want: []string{"t1.c"}, other: []string{"t1.b"}},

	// JSON_TABLE.
	{name: "json_table rows", statement: "SELECT jt.v FROM j, JSON_TABLE(j.doc, '$[*]' COLUMNS (v INT PATH '$')) jt", want: []string{"j.doc"}, results: [][]string{{"j.doc"}}},
	{name: "json_table condition", statement: "SELECT jt.v FROM j, JSON_TABLE(j.doc, '$[*]' COLUMNS (v INT PATH '$')) jt WHERE jt.v > 3", want: []string{"j.doc"}},

	// More shapes.
	{name: "not exists and not in", statement: "SELECT a FROM t WHERE NOT EXISTS (SELECT 1 FROM t1 WHERE t1.c = t.c) AND a NOT IN (SELECT b FROM u)", want: []string{"t.a", "t.c", "t1.c", "u.b"}},
	{name: "row in subquery", statement: "SELECT 1 FROM t WHERE (a, b) IN (SELECT a, b FROM t1)", want: []string{"t.a", "t.b", "t1.a", "t1.b"}},
	{name: "all subquery", statement: "SELECT a FROM t WHERE b = ALL (SELECT c FROM u)", want: []string{"t.b", "u.c"}},
	{name: "union in subquery", statement: "SELECT a FROM t WHERE a IN (SELECT a FROM t1 UNION SELECT b FROM u WHERE c = 0)", want: []string{"t.a", "t1.a", "u.b", "u.c"}},
	{name: "intersect in derived table", statement: "SELECT x FROM (SELECT a AS x FROM t INTERSECT SELECT b FROM t1) d", want: []string{"t.a", "t1.b"}, results: [][]string{{"t.a", "t1.b"}}},
	{name: "boolean operators", statement: "SELECT 1 FROM t WHERE NOT b OR (c IS TRUE XOR a)", want: []string{"t.a", "t.b", "t.c"}},
	{name: "convert", statement: "SELECT 1 FROM t WHERE CONVERT(b, CHAR) = '1'", want: []string{"t.b"}},
	{name: "distinct", statement: "SELECT DISTINCT a FROM t WHERE b = 1", want: []string{"t.b"}},
	{name: "select modifiers", statement: "SELECT STRAIGHT_JOIN SQL_CALC_FOUND_ROWS t.a FROM t, t1 WHERE t.b = t1.b", want: []string{"t.b", "t1.b"}},
	{name: "cte shadows a table", statement: "WITH t AS (SELECT a FROM t1 WHERE c = 0) SELECT a FROM t WHERE a > 1", want: []string{"t1.a", "t1.c"}, results: [][]string{{"t1.a"}}},
	{name: "cte column list", statement: "WITH c (x, y) AS (SELECT a, b FROM t) SELECT x FROM c WHERE y = 1", want: []string{"t.b"}, results: [][]string{{"t.a"}}},
	{name: "cte read twice", statement: "WITH c AS (SELECT a FROM t WHERE b = 1) SELECT c1.a FROM c AS c1 JOIN c AS c2 ON c1.a = c2.a", want: []string{"t.a", "t.b"}},
	{name: "json_table joined on", statement: "SELECT jt.v FROM j JOIN JSON_TABLE(j.doc, '$[*]' COLUMNS (v INT PATH '$')) jt ON jt.v = j.id", want: []string{"j.doc", "j.id"}},
	{name: "having alias in other case", statement: "SELECT c AS X FROM u HAVING x > 1", want: []string{"u.c"}},
	{name: "having quoted alias", statement: "SELECT c AS `my col` FROM u HAVING `my col` > 1", want: []string{"u.c"}},
	{name: "having alias in case", statement: "SELECT c AS b FROM u HAVING CASE WHEN b > 1 THEN 1 ELSE 0 END = 1", want: []string{"u.c"}},
	{name: "having count distinct", statement: "SELECT a FROM t GROUP BY a HAVING count(DISTINCT b) > 1", want: []string{"t.b"}},
	{name: "having with rollup", statement: "SELECT a, count(*) AS n FROM t GROUP BY a WITH ROLLUP HAVING a IS NULL", want: []string{"t.a"}},
	{name: "having alias of a joined column", statement: "SELECT t1.c AS b FROM t JOIN t1 ON t.a = t1.a HAVING b > 1", want: []string{"t.a", "t1.a", "t1.c"}},
	{name: "order by alias", statement: "SELECT c AS b FROM u ORDER BY b LIMIT 1", want: []string{}, results: [][]string{{"u.c"}}},
	{name: "order by expression reads from first", statement: "SELECT c AS b FROM u ORDER BY b + 0 LIMIT 1", want: []string{}, other: []string{"u.b"}},
	{name: "group by reads from first", statement: "SELECT max(b) AS c FROM u GROUP BY c", want: []string{}, results: [][]string{{"u.b"}}, other: []string{"u.c"}},
	{name: "exists over a qualified star", statement: "SELECT a FROM t WHERE EXISTS (SELECT t1.* FROM t1 WHERE t1.a = t.a)", want: []string{"t.a", "t1.a"}},
	{name: "derived table in a subquery sees the outer query", statement: "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM (SELECT b FROM t1 WHERE t1.a = t.a) d)", want: []string{"t.a", "t1.a"}, skip: map[string]string{"mariadb": "MariaDB refuses an outer reference in a derived table"}},
	{name: "subquery two levels down sees an alias", statement: "SELECT c AS x, (SELECT (SELECT x + 1)) AS s FROM u", want: []string{}, results: [][]string{{"u.c"}, {"u.c"}}},

	// A view's own conditions are not the caller's; its columns trace to the
	// table. The engines check a view's privileges on the view, so they skip it.
	{name: "view column", statement: "SELECT a FROM v WHERE vb = 1", want: []string{"t.b"}, results: [][]string{{"t.a"}}, skip: map[string]string{"mysql": "privileges are checked on the view", "mariadb": "privileges are checked on the view"}},
}

func TestQuerySpanPredicateColumns(t *testing.T) {
	for _, tc := range predicateCases {
		t.Run(tc.name, func(t *testing.T) {
			span, err := predicateCaseSpan("db", strings.ReplaceAll(tc.statement, "{db}", "db"))
			if tc.rejected {
				return
			}
			require.NoError(t, err)
			require.Nil(t, span.NotFoundError)
			require.Equal(t, sortedOrEmpty(tc.want), columnNames(span.PredicateColumns))
			if tc.results != nil {
				var results [][]string
				for _, result := range span.Results {
					results = append(results, columnNames(result.SourceColumns))
				}
				want := make([][]string, 0, len(tc.results))
				for _, columns := range tc.results {
					want = append(want, sortedOrEmpty(columns))
				}
				require.Equal(t, want, results)
			}
		})
	}
}

func TestQuerySpanPredicateColumnsFallBackByName(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		want      []string
	}{
		{"unknown table in a subquery", "SELECT a FROM t WHERE a IN (SELECT x FROM missing)", []string{"t.a"}},
		{"unknown table in an exists", "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM missing WHERE missing.b = t.b)", []string{"t.b"}},
		{"unknown column", "SELECT a FROM t WHERE b = 1 AND missing = 2", []string{"t.b"}},
		{"unknown table in an order by subquery", "SELECT a FROM t ORDER BY (SELECT 1 FROM missing WHERE c = 0)", []string{"t.c"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			span, err := predicateCaseSpan("db", tc.statement)
			require.NoError(t, err)
			require.Nil(t, span.NotFoundError)
			require.Equal(t, tc.want, columnNames(span.PredicateColumns))
		})
	}
}

// TestQuerySpanPredicateColumnsMatchEngines runs each case on a user denied
// one fixture column at a time: the engine refuses exactly the statements
// that read the column, whichever name or clause the statement reads it by.
func TestQuerySpanPredicateColumnsMatchEngines(t *testing.T) {
	for _, engine := range []struct {
		name        string
		newDatabase func(testing.TB) (*testcontainer.Container, string)
	}{
		{name: "mysql", newDatabase: testcontainer.NewMySQLDatabase},
		{name: "mariadb", newDatabase: testcontainer.NewMariaDBDatabase},
	} {
		t.Run(engine.name, func(t *testing.T) {
			container, database := engine.newDatabase(t)
			oracle := newColumnReadOracle(t, container, database)
			for _, tc := range predicateCases {
				t.Run(tc.name, func(t *testing.T) {
					if reason, ok := tc.skip[engine.name]; ok {
						t.Skip(reason)
					}
					statement := strings.ReplaceAll(tc.statement, "{db}", database)
					read, err := oracle.columnsRead(statement)
					if tc.rejected {
						require.ErrorIs(t, err, errRejected)
						return
					}
					require.NoError(t, err)

					span, err := predicateCaseSpan(database, statement)
					require.NoError(t, err)
					seen := make(base.SourceColumnSet)
					for _, result := range span.Results {
						mergeSourceColumnSetInto(seen, result.SourceColumns)
					}
					mergeSourceColumnSetInto(seen, span.PredicateColumns)
					got := append(columnNames(seen), tc.other...)
					got = slices.DeleteFunc(got, func(column string) bool { return slices.Contains(tc.unchecked, column) })
					slices.Sort(got)
					require.Equal(t, read, slices.Compact(got))
				})
			}
		})
	}
}

var errRejected = errors.New("the engine refuses the statement")

type columnReadOracle struct {
	root *sql.DB
	// deniedOne maps each fixture column to a session that may read every
	// fixture column but that one.
	deniedOne map[string]*sql.DB
}

func newColumnReadOracle(t *testing.T, container *testcontainer.Container, database string) *columnReadOracle {
	t.Helper()
	open := func(user, password string) *sql.DB {
		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, container.GetHost(), container.GetPort(), database))
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		t.Cleanup(func() { require.NoError(t, db.Close()) })
		return db
	}
	oracle := &columnReadOracle{
		root:      open(container.GetUsername(), container.GetPassword()),
		deniedOne: make(map[string]*sql.DB),
	}
	for _, table := range predicateFixtureTables {
		_, err := oracle.root.Exec(table.ddl)
		require.NoError(t, err)
	}
	_, err := oracle.root.Exec("CREATE VIEW v AS SELECT a, b AS vb FROM t WHERE c > 0")
	require.NoError(t, err)

	i := 0
	for _, denied := range predicateFixtureTables {
		for _, deniedColumn := range denied.columns {
			i++
			// Users are server-wide, so they carry the database's name.
			user := fmt.Sprintf("%s_%d", database, i)
			_, err := oracle.root.Exec(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY 'pw'", user))
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := container.GetDB().Exec(fmt.Sprintf("DROP USER '%s'@'%%'", user))
				require.NoError(t, err)
			})
			for _, table := range predicateFixtureTables {
				grant := fmt.Sprintf("GRANT SELECT ON `%s`.`%s` TO '%s'@'%%'", database, table.name, user)
				if table.name == denied.name {
					var allowed []string
					for _, column := range table.columns {
						if column != deniedColumn {
							allowed = append(allowed, "`"+column+"`")
						}
					}
					grant = fmt.Sprintf("GRANT SELECT (%s) ON `%s`.`%s` TO '%s'@'%%'", strings.Join(allowed, ", "), database, table.name, user)
				}
				_, err := oracle.root.Exec(grant)
				require.NoError(t, err)
			}
			oracle.deniedOne[denied.name+"."+deniedColumn] = open(user, "pw")
		}
	}
	return oracle
}

// columnsRead returns the fixture columns the engine reads to run statement,
// or errRejected if it refuses the statement outright.
func (o *columnReadOracle) columnsRead(statement string) ([]string, error) {
	if err := drain(o.root, statement); err != nil {
		return nil, errors.Wrap(errRejected, err.Error())
	}
	read := []string{}
	for column, db := range o.deniedOne {
		err := drain(db, statement)
		if err == nil {
			continue
		}
		var mysqlErr *gomysql.MySQLError
		// 1142 is a table-level refusal, which * over the table raises.
		if errors.As(err, &mysqlErr) && (mysqlErr.Number == 1142 || mysqlErr.Number == 1143) {
			read = append(read, column)
			continue
		}
		return nil, errors.Wrapf(err, "denied %s", column)
	}
	slices.Sort(read)
	return read, nil
}

func drain(db *sql.DB, statement string) error {
	rows, err := db.Query(statement)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
	}
	return rows.Err()
}

func predicateCaseSpan(database, statement string) (*base.QuerySpan, error) {
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{predicateFixtureMetadata(database)})
	return GetQuerySpan(context.Background(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_MYSQL,
	}, base.Statement{Text: statement}, database, "", false)
}

func predicateFixtureMetadata(database string) *metadatapb.DatabaseSchemaMetadata {
	schema := &metadatapb.SchemaMetadata{
		Views: []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT a, b AS vb FROM t WHERE c > 0"}},
	}
	for _, table := range predicateFixtureTables {
		metadata := &metadatapb.TableMetadata{Name: table.name}
		for _, column := range table.columns {
			metadata.Columns = append(metadata.Columns, &metadatapb.ColumnMetadata{Name: column})
		}
		schema.Tables = append(schema.Tables, metadata)
	}
	return &metadatapb.DatabaseSchemaMetadata{Name: database, Schemas: []*metadatapb.SchemaMetadata{schema}}
}

// columnNames returns the table.column names in columns, sorted, skipping
// table-level entries.
func columnNames(columns base.SourceColumnSet) []string {
	names := []string{}
	for column := range columns {
		if column.Column != "" {
			names = append(names, column.Table+"."+column.Column)
		}
	}
	slices.Sort(names)
	return names
}

func sortedOrEmpty(names []string) []string {
	sorted := append([]string{}, names...)
	slices.Sort(sorted)
	return sorted
}
