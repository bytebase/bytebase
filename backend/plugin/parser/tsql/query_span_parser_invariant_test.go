package tsql

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/bytebase/omni/mssql/ast"
	"gopkg.in/yaml.v3"
)

// TestOmniQuerySpanParserInvariants verifies that the omni MSSQL parser can
// handle every statement used by the query-span fixtures, and that the AST
// continues to populate the fields the query-span extractor depends on. Run with
//
//	go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/tsql -run ^TestOmniQuerySpanParserInvariants$
func TestOmniQuerySpanParserInvariants(t *testing.T) {
	t.Run("FixtureParseCoverage", querySpanFixtureParseCoverage)
	t.Run("StructuralInvariants", querySpanStructuralInvariants)
}

type fixtureEntry struct {
	Description string `yaml:"description"`
	Statement   string `yaml:"statement"`
}

var querySpanFixturePaths = []string{
	"test-data/query-span/standard.yaml",
	"test-data/query-span/join.yaml",
	"test-data/query-span/case-sensitivity.yaml",
	"test-data/query-span/query_type.yaml",
	"test-data/query-span/predicate.yaml",
	"test-data/query-span/regression.yaml",
}

func querySpanFixtureParseCoverage(t *testing.T) {
	type failure struct {
		fixture   string
		index     int
		statement string
		err       error
	}

	totals := map[string][2]int{} // fixture -> [total, passed]
	var failures []failure

	for _, path := range querySpanFixturePaths {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var entries []fixtureEntry
		if err := yaml.Unmarshal(data, &entries); err != nil {
			t.Fatalf("yaml %s: %v", path, err)
		}
		total := 0
		passed := 0
		for i, e := range entries {
			if strings.TrimSpace(e.Statement) == "" {
				continue
			}
			total++
			_, err := ParseTSQLOmni(e.Statement)
			if err != nil {
				failures = append(failures, failure{fixture: path, index: i, statement: e.Statement, err: err})
				continue
			}
			passed++
		}
		totals[path] = [2]int{total, passed}
	}

	t.Log("=== Omni parse coverage per fixture ===")
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
			t.Logf("  [%s case %d] err: %v\n    SQL: %s", f.fixture, f.index, f.err, firstLine(f.statement))
		}
		t.Errorf("omni failed to parse %d/%d fixture statements", len(failures), totalAll)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

// querySpanStructuralInvariants runs hand-crafted SQL statements and asserts
// the omni AST populates fields the extractor relies on. A failure means we
// must either update omni or adapt the extractor.
func querySpanStructuralInvariants(t *testing.T) {
	type invariant struct {
		name  string
		sql   string
		check func(*testing.T, ast.Node)
	}

	invariants := []invariant{
		{
			name: "res_target_as_alias",
			sql:  "SELECT c AS a FROM t",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				rt := firstOf[*ast.ResTarget](t, sel.TargetList)
				if rt.Name != "a" {
					t.Errorf("want ResTarget.Name=a, got %q", rt.Name)
				}
			},
		},
		{
			name: "res_target_equals_alias",
			sql:  "SELECT a = c FROM t",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.TargetList == nil || sel.TargetList.Len() == 0 {
					t.Fatal("empty TargetList")
				}
				// Document actual shape — omni may lose the alias completely for this form.
				switch v := sel.TargetList.Items[0].(type) {
				case *ast.ResTarget:
					t.Logf("omni parses `a = c` as ResTarget{Name=%q, Val=%T} — extractor must recover alias from Val if Name is empty", v.Name, v.Val)
					if v.Name != "a" {
						if be, ok := v.Val.(*ast.BinaryExpr); ok && be.Op == ast.BinOpEq {
							t.Logf("  recovery: Val is BinaryExpr{Op=Eq, Left=%T, Right=%T} — extractor can pull alias from Left *ColumnRef.Column", be.Left, be.Right)
						}
						t.Errorf("ResTarget.Name: want a, got %q (parser does not attribute the assignment form)", v.Name)
					}
				case *ast.SelectAssign:
					t.Errorf("omni parses `a = c` as SelectAssign(var=%q) — that's for `SET @a = c`, not column alias", v.Variable)
				default:
					t.Errorf("first TargetList item is %T", v)
				}
			},
		},
		{
			name: "res_target_bare_column",
			sql:  "SELECT c FROM t",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.TargetList == nil || sel.TargetList.Len() == 0 {
					t.Fatal("empty TargetList")
				}
				switch v := sel.TargetList.Items[0].(type) {
				case *ast.ResTarget:
					if v.Name != "" {
						t.Errorf("ResTarget.Name: want empty for bare column, got %q", v.Name)
					}
					if _, ok := v.Val.(*ast.ColumnRef); !ok {
						t.Errorf("ResTarget.Val: want *ColumnRef, got %T", v.Val)
					}
				case *ast.ColumnRef:
					t.Log("note: bare column appears directly (not wrapped in ResTarget)")
				default:
					t.Errorf("first TargetList item is %T", v)
				}
			},
		},
		{
			name: "star_unqualified",
			sql:  "SELECT * FROM t",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				star := findFirst[*ast.StarExpr](sel)
				if star == nil {
					t.Fatal("no StarExpr")
				}
				if star.Qualifier != "" {
					t.Errorf("want Qualifier empty, got %q", star.Qualifier)
				}
			},
		},
		{
			name: "star_qualified",
			sql:  "SELECT t1.* FROM t1",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				star := findFirst[*ast.StarExpr](sel)
				if star == nil {
					t.Fatal("no StarExpr")
				}
				if star.Qualifier != "t1" {
					t.Errorf("want Qualifier=t1, got %q", star.Qualifier)
				}
			},
		},
		{
			name: "derived_table_no_column_list",
			sql:  "SELECT a FROM (SELECT a FROM t) AS x",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				at := findFirst[*ast.AliasedTableRef](sel.FromClause)
				if at == nil {
					t.Fatal("no AliasedTableRef")
				}
				if at.Alias != "x" {
					t.Errorf("Alias: want x, got %q", at.Alias)
				}
			},
		},
		{
			name: "derived_table_column_list",
			sql:  "SELECT c1, c2 FROM (SELECT a, b FROM t) AS x(c1, c2)",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				at := findFirst[*ast.AliasedTableRef](sel.FromClause)
				if at == nil {
					t.Fatal("no AliasedTableRef in FROM")
				}
				if at.Alias != "x" {
					t.Errorf("Alias: want x, got %q", at.Alias)
				}
				if at.Columns == nil || at.Columns.Len() != 2 {
					n := 0
					if at.Columns != nil {
						n = at.Columns.Len()
					}
					t.Errorf("Columns len: want 2, got %d", n)
				}
			},
		},
		{
			name: "cte_column_list",
			sql:  "WITH cte(c1, c2) AS (SELECT a, b FROM t) SELECT c1 FROM cte",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.WithClause == nil || sel.WithClause.CTEs == nil || sel.WithClause.CTEs.Len() == 0 {
					t.Fatal("missing WITH clause")
				}
				c, ok := sel.WithClause.CTEs.Items[0].(*ast.CommonTableExpr)
				if !ok {
					t.Fatalf("CTEs.Items[0] is %T, want *CommonTableExpr", sel.WithClause.CTEs.Items[0])
				}
				if !strings.EqualFold(c.Name, "cte") {
					t.Errorf("CTE name: want cte, got %q", c.Name)
				}
				if c.Columns == nil || c.Columns.Len() != 2 {
					n := 0
					if c.Columns != nil {
						n = c.Columns.Len()
					}
					t.Errorf("CTE Columns len: want 2, got %d", n)
				}
			},
		},
		{
			name: "declare_table_variable_columns",
			sql:  "DECLARE @t TABLE (id INT, name NVARCHAR(50))",
			check: func(t *testing.T, n ast.Node) {
				d, ok := n.(*ast.DeclareStmt)
				if !ok {
					t.Fatalf("want *DeclareStmt, got %T", n)
				}
				if d.Variables == nil || d.Variables.Len() == 0 {
					t.Fatal("no Variables")
				}
				v, ok := d.Variables.Items[0].(*ast.VariableDecl)
				if !ok {
					t.Fatalf("Variables.Items[0] is %T, want *VariableDecl", d.Variables.Items[0])
				}
				if !v.IsTable {
					t.Error("want IsTable=true")
				}
				if v.TableDef == nil || v.TableDef.Len() != 2 {
					n := 0
					if v.TableDef != nil {
						n = v.TableDef.Len()
					}
					t.Errorf("TableDef len: want 2, got %d", n)
				}
				// Verify ColumnDef has DataType populated.
				if v.TableDef != nil && v.TableDef.Len() > 0 {
					cd, ok := v.TableDef.Items[0].(*ast.ColumnDef)
					if !ok {
						t.Errorf("TableDef.Items[0] is %T, want *ColumnDef", v.TableDef.Items[0])
					} else if cd.DataType == nil {
						t.Error("ColumnDef.DataType is nil")
					} else if !strings.EqualFold(cd.DataType.Name, "int") {
						t.Errorf("DataType.Name: want INT, got %q", cd.DataType.Name)
					}
				}
			},
		},
		{
			name: "union_set_op",
			sql:  "SELECT a FROM t1 UNION SELECT a FROM t2",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.Op != ast.SetOpUnion {
					t.Errorf("want Op=SetOpUnion, got %v", sel.Op)
				}
				if sel.Larg == nil || sel.Rarg == nil {
					t.Error("want non-nil Larg/Rarg")
				}
			},
		},
		{
			name: "union_all_set_op",
			sql:  "SELECT a FROM t1 UNION ALL SELECT a FROM t2",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.Op != ast.SetOpUnion {
					t.Errorf("want Op=SetOpUnion, got %v", sel.Op)
				}
				if !sel.All {
					t.Error("want All=true for UNION ALL")
				}
			},
		},
		{
			name: "intersect_except",
			sql:  "SELECT a FROM t1 INTERSECT SELECT a FROM t2 EXCEPT SELECT a FROM t3",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				// Just confirm Op is Intersect or Except at the top and arms exist.
				if sel.Op != ast.SetOpIntersect && sel.Op != ast.SetOpExcept {
					t.Errorf("want Op=Intersect or Except, got %v", sel.Op)
				}
				if sel.Larg == nil || sel.Rarg == nil {
					t.Error("want non-nil Larg/Rarg")
				}
			},
		},
		{
			name: "correlated_subquery_alias",
			sql:  "SELECT a FROM outerT AS o WHERE EXISTS (SELECT 1 FROM innerT i WHERE i.a = o.a)",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.WhereClause == nil {
					t.Fatal("missing WHERE")
				}
				var foundExists, foundOuterRef bool
				ast.Inspect(sel.WhereClause, func(n ast.Node) bool {
					if _, ok := n.(*ast.ExistsExpr); ok {
						foundExists = true
					}
					if cr, ok := n.(*ast.ColumnRef); ok && strings.EqualFold(cr.Table, "o") {
						foundOuterRef = true
					}
					return true
				})
				if !foundExists {
					t.Error("missing ExistsExpr in WHERE")
				}
				if !foundOuterRef {
					t.Error("missing correlated column ref o.a")
				}
			},
		},
		{
			name: "cross_apply_tvf_no_col_list",
			sql:  "SELECT * FROM t1 CROSS APPLY fn(t1.a) AS x",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				j := findFirst[*ast.JoinClause](sel.FromClause)
				if j == nil {
					t.Fatal("no JoinClause")
				}
				if j.Type != ast.JoinCrossApply {
					t.Errorf("JoinType: want CrossApply, got %v", j.Type)
				}
			},
		},
		{
			name: "cross_apply_tvf",
			sql:  "SELECT * FROM t1 CROSS APPLY fn(t1.a) AS x(v)",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				j := findFirst[*ast.JoinClause](sel.FromClause)
				if j == nil {
					t.Fatal("no JoinClause in FROM")
				}
				if j.Type != ast.JoinCrossApply {
					t.Errorf("JoinType: want CrossApply, got %v", j.Type)
				}
			},
		},
		{
			name: "pivot",
			sql:  "SELECT * FROM src PIVOT (SUM(amt) FOR cat IN ([a],[b])) AS p",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if p := findFirst[*ast.PivotExpr](sel.FromClause); p == nil {
					t.Error("no PivotExpr")
				}
			},
		},
		{
			name: "unpivot",
			sql:  "SELECT * FROM src UNPIVOT (v FOR c IN ([x],[y])) AS u",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if u := findFirst[*ast.UnpivotExpr](sel.FromClause); u == nil {
					t.Error("no UnpivotExpr")
				}
			},
		},
		{
			name: "values_insert",
			sql:  "INSERT INTO t(a,b) VALUES (1,2),(3,4)",
			check: func(t *testing.T, n ast.Node) {
				if vc := findFirst[*ast.ValuesClause](n); vc == nil {
					t.Error("no ValuesClause in INSERT — parser may lack VALUES support entirely")
				}
			},
		},
		{
			name: "values_table_constructor",
			sql:  "SELECT * FROM (VALUES (1,2),(3,4)) AS v(a,b)",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if vc := findFirst[*ast.ValuesClause](sel.FromClause); vc == nil {
					t.Error("no ValuesClause")
				}
			},
		},
		{
			name: "contains_predicate",
			sql:  "SELECT a FROM t WHERE CONTAINS(b, 'foo')",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.WhereClause == nil {
					t.Error("WHERE is nil")
				}
				// Omni likely parses CONTAINS as a FuncCallExpr; document actual node shape.
				var kind string
				switch sel.WhereClause.(type) {
				case *ast.FuncCallExpr:
					kind = "FuncCallExpr"
				case *ast.BinaryExpr:
					kind = "BinaryExpr"
				default:
					kind = fmt.Sprintf("%T", sel.WhereClause)
				}
				t.Logf("CONTAINS predicate shape: %s — extractor must handle this form", kind)
			},
		},
		{
			name: "freetext_predicate",
			sql:  "SELECT a FROM t WHERE FREETEXT(b, 'foo')",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.WhereClause == nil {
					t.Error("WHERE is nil")
				}
				t.Logf("FREETEXT WHERE node: %T", sel.WhereClause)
			},
		},
		{
			name: "bracketed_identifier",
			sql:  "SELECT [col name] FROM [my table]",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				col := findFirst[*ast.ColumnRef](sel)
				if col == nil {
					t.Fatal("no ColumnRef")
				}
				if col.Column != "col name" {
					t.Errorf("Column: want \"col name\" (unquoted), got %q", col.Column)
				}
				tr := findFirst[*ast.TableRef](sel)
				if tr == nil {
					t.Fatal("no TableRef")
				}
				if tr.Object != "my table" {
					t.Errorf("TableRef.Object: want \"my table\", got %q", tr.Object)
				}
			},
		},
		{
			name: "quoted_identifier",
			sql:  "SELECT \"col name\" FROM \"my table\"",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				col := findFirst[*ast.ColumnRef](sel)
				if col == nil {
					t.Fatal("no ColumnRef")
				}
				if col.Column != "col name" {
					t.Errorf("Column: want \"col name\" (unquoted), got %q", col.Column)
				}
			},
		},
		{
			name: "cross_database_reference",
			sql:  "SELECT c FROM db1.dbo.t",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				tr := findFirst[*ast.TableRef](sel.FromClause)
				if tr == nil {
					t.Fatal("no TableRef in FROM")
				}
				if tr.Database != "db1" || tr.Schema != "dbo" || tr.Object != "t" {
					t.Errorf("TableRef: got db=%q schema=%q obj=%q", tr.Database, tr.Schema, tr.Object)
				}
			},
		},
		{
			name: "linked_server_reference",
			sql:  "SELECT c FROM srv1.db1.dbo.t",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				tr := findFirst[*ast.TableRef](sel.FromClause)
				if tr == nil {
					t.Fatal("no TableRef in FROM")
				}
				if tr.Server != "srv1" || tr.Database != "db1" || tr.Schema != "dbo" || tr.Object != "t" {
					t.Errorf("TableRef: got srv=%q db=%q schema=%q obj=%q", tr.Server, tr.Database, tr.Schema, tr.Object)
				}
			},
		},
		{
			name: "table_variable_in_from",
			sql:  "DECLARE @t TABLE(id INT); SELECT id FROM @t",
			check: func(t *testing.T, _ ast.Node) {
				// ParseTSQLOmni gives a slice; this invariant only checks the first stmt (DECLARE).
				// Walk the full parse to confirm the second has @t as TableVarRef.
				stmts, err := ParseTSQLOmni("DECLARE @t TABLE(id INT); SELECT id FROM @t")
				if err != nil {
					t.Fatalf("parse: %v", err)
				}
				if len(stmts) < 2 {
					t.Fatalf("want 2 stmts, got %d", len(stmts))
				}
				sel, ok := stmts[1].AST.(*ast.SelectStmt)
				if !ok {
					t.Fatalf("stmt[1] is %T, want *SelectStmt", stmts[1].AST)
				}
				tv := findFirst[*ast.TableVarRef](sel.FromClause)
				if tv == nil {
					t.Error("no TableVarRef in SELECT FROM @t")
				} else if tv.Name != "@t" {
					t.Errorf("TableVarRef.Name: want @t, got %q", tv.Name)
				}
			},
		},
		{
			name: "temp_table_hash_prefix",
			sql:  "SELECT c FROM #t",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				tr := findFirst[*ast.TableRef](sel.FromClause)
				if tr == nil {
					t.Fatal("no TableRef in FROM")
				}
				if tr.Object != "#t" {
					t.Errorf("TableRef.Object for temp table: got %q — extractor must recognize '#' prefix as local temp", tr.Object)
				}
			},
		},
		{
			name: "xml_nodes_method",
			sql:  "SELECT T.c.value('@id', 'int') FROM @xml.nodes('/root/item') AS T(c)",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if m := findFirst[*ast.TableVarMethodCallRef](sel.FromClause); m == nil {
					t.Error("no TableVarMethodCallRef")
				}
			},
		},
		{
			name: "for_xml_clause",
			sql:  "SELECT a FROM t FOR XML AUTO",
			check: func(t *testing.T, n ast.Node) {
				sel := mustSelect(t, n)
				if sel.ForClause == nil {
					t.Error("ForClause is nil")
				}
			},
		},
		{
			name: "merge_output",
			sql:  "MERGE tgt USING src ON tgt.id = src.id WHEN MATCHED THEN UPDATE SET v = src.v;",
			check: func(t *testing.T, n ast.Node) {
				if _, ok := n.(*ast.MergeStmt); !ok {
					t.Errorf("want *MergeStmt, got %T", n)
				}
			},
		},
	}

	for _, p := range invariants {
		t.Run(p.name, func(t *testing.T) {
			stmts, err := ParseTSQLOmni(p.sql)
			if err != nil {
				t.Fatalf("parse err: %v\nSQL: %s", err, p.sql)
			}
			if len(stmts) == 0 || stmts[0].AST == nil {
				t.Fatal("empty parse result")
			}
			p.check(t, stmts[0].AST)
		})
	}
}

func mustSelect(t *testing.T, n ast.Node) *ast.SelectStmt {
	t.Helper()
	sel, ok := n.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected *SelectStmt, got %T", n)
	}
	return sel
}

// findFirst walks root (any ast.Node) and returns the first node that type-asserts to T, or nil.
func findFirst[T ast.Node](root ast.Node) T {
	var zero T
	if root == nil {
		return zero
	}
	var found T
	ast.Inspect(root, func(n ast.Node) bool {
		if cast, ok := n.(T); ok {
			// Only take the first match; keep walking is harmless but we stop to save work.
			if any(found) == any(zero) {
				found = cast
			}
			return false
		}
		return true
	})
	return found
}

// firstOf returns the first item in a List that type-asserts to T, or nil.
// Used when callers know the expected wrapper type (e.g. *ResTarget in TargetList).
func firstOf[T ast.Node](t *testing.T, list *ast.List) T {
	t.Helper()
	var zero T
	if list == nil || list.Len() == 0 {
		t.Fatal("list empty")
	}
	cast, ok := list.Items[0].(T)
	if !ok {
		t.Fatalf("first item is %T, want %T", list.Items[0], zero)
	}
	return cast
}
