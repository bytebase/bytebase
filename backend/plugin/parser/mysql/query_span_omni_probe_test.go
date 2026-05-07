package mysql

import (
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestMySQLOmniQuerySpanMigrationProbe verifies that omni can parse and expose
// the AST shapes the MySQL query-span migration will rely on. It does not use
// or change the production query-span path.
func TestMySQLOmniQuerySpanMigrationProbe(t *testing.T) {
	t.Run("FixtureParseCoverage", probeMySQLOmniFixtureParseCoverage)
	t.Run("StructuralInvariants", probeMySQLOmniStructuralInvariants)
}

type mysqlOmniProbeFixtureEntry struct {
	Description string `yaml:"description"`
	Statement   string `yaml:"statement"`
}

var mysqlOmniProbeFixturePaths = []string{
	"test-data/query-span/query_type.yaml",
	"test-data/query-span/standard.yaml",
	"test-data/query-span/case_insensitive.yaml",
	"test-data/query-span/starrocks.yaml",
}

func probeMySQLOmniFixtureParseCoverage(t *testing.T) {
	type failure struct {
		fixture   string
		index     int
		statement string
		err       error
	}

	totals := map[string][2]int{} // fixture -> [total, passed]
	var failures []failure
	for _, path := range mysqlOmniProbeFixturePaths {
		f, err := os.Open(path)
		require.NoError(t, err)
		data, err := io.ReadAll(f)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		var entries []mysqlOmniProbeFixtureEntry
		require.NoError(t, yaml.Unmarshal(data, &entries))

		total := 0
		passed := 0
		for i, entry := range entries {
			if strings.TrimSpace(entry.Statement) == "" {
				continue
			}
			total++
			list, err := ParseMySQLOmni(entry.Statement)
			if err != nil {
				failures = append(failures, failure{
					fixture:   path,
					index:     i,
					statement: entry.Statement,
					err:       err,
				})
				continue
			}
			if list == nil || list.Len() == 0 {
				failures = append(failures, failure{
					fixture:   path,
					index:     i,
					statement: entry.Statement,
					err:       errEmptyOmniStatement,
				})
				continue
			}
			passed++
		}
		totals[path] = [2]int{total, passed}
	}

	t.Log("=== MySQL omni parse coverage per fixture ===")
	var keys []string
	for k := range totals {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	totalAll, passedAll := 0, 0
	for _, k := range keys {
		v := totals[k]
		totalAll += v[0]
		passedAll += v[1]
		t.Logf("  %s: %d/%d", k, v[1], v[0])
	}
	t.Logf("  TOTAL: %d/%d", passedAll, totalAll)

	if len(failures) > 0 {
		t.Logf("=== %d parse failures ===", len(failures))
		for _, f := range failures {
			t.Logf("  [%s case %d] err: %v\n    SQL: %s", f.fixture, f.index, f.err, firstMySQLOmniProbeLine(f.statement))
		}
		t.Errorf("omni failed to parse %d/%d fixture statements", len(failures), totalAll)
	}
}

var errEmptyOmniStatement = errors.New("omni returned no statements")

func firstMySQLOmniProbeLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " ..."
	}
	return s
}

func probeMySQLOmniStructuralInvariants(t *testing.T) {
	t.Run("select_targets_aliases_and_stars", func(t *testing.T) {
		sel := mustMySQLOmniSelect(t, "SELECT a, b AS alias_b, db.t.c, t.* FROM db.t AS t")
		require.Len(t, sel.TargetList, 4)

		first := mustMySQLOmniTargetColumn(t, sel.TargetList[0])
		require.Equal(t, "a", first.Column)
		require.Empty(t, first.Table)
		require.False(t, first.Star)

		aliasTarget, ok := sel.TargetList[1].(*ast.ResTarget)
		require.Truef(t, ok, "second target is %T", sel.TargetList[1])
		require.Equal(t, "alias_b", aliasTarget.Name)
		aliasColumn, ok := aliasTarget.Val.(*ast.ColumnRef)
		require.Truef(t, ok, "alias target value is %T", aliasTarget.Val)
		require.Equal(t, "b", aliasColumn.Column)

		qualified := mustMySQLOmniTargetColumn(t, sel.TargetList[2])
		require.Equal(t, "db", qualified.Schema)
		require.Equal(t, "t", qualified.Table)
		require.Equal(t, "c", qualified.Column)

		tableStar := mustMySQLOmniTargetColumn(t, sel.TargetList[3])
		require.Equal(t, "t", tableStar.Table)
		require.True(t, tableStar.Star)
	})

	t.Run("join_on_and_using", func(t *testing.T) {
		sel := mustMySQLOmniSelect(t, "SELECT * FROM t1 JOIN db.t2 AS t2 ON t1.a = t2.a JOIN t3 USING (id)")
		require.Len(t, sel.From, 1)
		require.Len(t, collectMySQLOmniNodes[*ast.TableRef](sel), 3)
		require.NotEmpty(t, collectMySQLOmniNodes[*ast.OnCondition](sel))
		using := collectMySQLOmniNodes[*ast.UsingCondition](sel)
		require.Len(t, using, 1)
		require.Equal(t, []string{"id"}, using[0].Columns)
	})

	t.Run("derived_table_alias_and_column_aliases", func(t *testing.T) {
		sel := mustMySQLOmniSelect(t, "SELECT x.c1 FROM (SELECT a, b FROM t) AS x(c1, c2)")
		require.Len(t, sel.From, 1)
		subquery, ok := sel.From[0].(*ast.SubqueryExpr)
		require.Truef(t, ok, "from item is %T", sel.From[0])
		require.Equal(t, "x", subquery.Alias)
		require.Equal(t, []string{"c1", "c2"}, subquery.Columns)
		require.NotNil(t, subquery.Select)
	})

	t.Run("recursive_cte_with_set_operation", func(t *testing.T) {
		sel := mustMySQLOmniSelect(t, "WITH RECURSIVE cte(a, b) AS (SELECT a, b FROM t UNION SELECT a, b FROM t2) SELECT * FROM cte")
		require.Len(t, sel.CTEs, 1)
		cte := sel.CTEs[0]
		require.Equal(t, "cte", cte.Name)
		require.True(t, cte.Recursive)
		require.Equal(t, []string{"a", "b"}, cte.Columns)
		require.NotNil(t, cte.Select)
		require.Equal(t, ast.SetOpUnion, cte.Select.SetOp)
		require.NotNil(t, cte.Select.Left)
		require.NotNil(t, cte.Select.Right)
	})

	t.Run("correlated_scalar_subquery", func(t *testing.T) {
		sel := mustMySQLOmniSelect(t, "SELECT city, (SELECT COUNT(*) FROM paintings p WHERE g.id = p.gallery_id) AS total FROM galleries g")
		require.Len(t, sel.TargetList, 2)
		subqueries := collectMySQLOmniNodes[*ast.SubqueryExpr](sel)
		require.Len(t, subqueries, 1)
		require.NotNil(t, subqueries[0].Select)

		var sawOuterRef, sawInnerRef bool
		for _, col := range collectMySQLOmniNodes[*ast.ColumnRef](sel) {
			if col.Table == "g" && col.Column == "id" {
				sawOuterRef = true
			}
			if col.Table == "p" && col.Column == "gallery_id" {
				sawInnerRef = true
			}
		}
		require.True(t, sawOuterRef, "expected correlated outer column g.id")
		require.True(t, sawInnerRef, "expected inner column p.gallery_id")
	})

	t.Run("json_table_table_source", func(t *testing.T) {
		sel := mustMySQLOmniSelect(t, "SELECT jt.id FROM t, JSON_TABLE(t.doc, '$[*]' COLUMNS(id INT PATH '$.id')) AS jt")
		jsonTables := collectMySQLOmniNodes[*ast.JsonTableExpr](sel)
		require.Len(t, jsonTables, 1)
		require.Equal(t, "jt", jsonTables[0].Alias)
		require.Len(t, jsonTables[0].Columns, 1)
		require.Equal(t, "id", jsonTables[0].Columns[0].Name)
	})

	t.Run("query_type_statement_roots", func(t *testing.T) {
		require.IsType(t, &ast.ExplainStmt{}, mustMySQLOmniNode(t, "EXPLAIN ANALYZE SELECT * FROM t"))
		require.IsType(t, &ast.ShowStmt{}, mustMySQLOmniNode(t, "SHOW DATABASES"))
		require.IsType(t, &ast.SetStmt{}, mustMySQLOmniNode(t, "SET CHARSET DEFAULT"))
		require.IsType(t, &ast.CreateTableStmt{}, mustMySQLOmniNode(t, "CREATE TABLE t(a INT)"))
		require.IsType(t, &ast.InsertStmt{}, mustMySQLOmniNode(t, "INSERT INTO t VALUES(1)"))
	})
}

func mustMySQLOmniNode(t *testing.T, sql string) ast.Node {
	t.Helper()
	list, err := ParseMySQLOmni(sql)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 1)
	require.NotNil(t, list.Items[0])
	return list.Items[0]
}

func mustMySQLOmniSelect(t *testing.T, sql string) *ast.SelectStmt {
	t.Helper()
	sel, ok := mustMySQLOmniNode(t, sql).(*ast.SelectStmt)
	require.Truef(t, ok, "statement is not *ast.SelectStmt: %s", sql)
	return sel
}

func mustMySQLOmniTargetColumn(t *testing.T, target ast.ExprNode) *ast.ColumnRef {
	t.Helper()
	switch v := target.(type) {
	case *ast.ColumnRef:
		return v
	case *ast.ResTarget:
		col, ok := v.Val.(*ast.ColumnRef)
		require.Truef(t, ok, "target value is %T", v.Val)
		return col
	default:
		t.Fatalf("target is %T", target)
		return nil
	}
}

func collectMySQLOmniNodes[T ast.Node](root ast.Node) []T {
	var nodes []T
	ast.Inspect(root, func(n ast.Node) bool {
		if typed, ok := n.(T); ok {
			nodes = append(nodes, typed)
		}
		return true
	})
	return nodes
}
