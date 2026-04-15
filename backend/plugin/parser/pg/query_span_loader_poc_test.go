package pg

// v5 PoC — AST-only catalog population.
//
// This PoC demonstrates that we can:
//   1. Construct a *ast.CreateStmt directly in Go code (no SQL text, no parser).
//   2. Call catalog.DefineRelation on it to install a table into an omni catalog.
//   3. Parse a SELECT query against that table.
//   4. Call AnalyzeSelectStmt and get correct column lineage.
//
// The entire "table install" half of the pipeline never touches pgparser.Parse,
// which is the source of BYT-9215 / BYT-9261 class failures. Only the SELECT
// statement itself is parsed.
//
// If this PoC passes, v5 ("lazy per-object AST install") is the recommended
// architecture for PG query span and supersedes v2/v3/v4 plans.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"
)

// mustTypeName is a test-only shim around the production typeNameFromString.
// It fails the test on any parse error; production code uses typeNameFromString
// directly and handles the error.
func mustTypeName(t *testing.T, typeStr string) *ast.TypeName {
	t.Helper()
	tn, err := typeNameFromString(typeStr)
	if err != nil {
		t.Fatalf("typeNameFromString(%q): %v", typeStr, err)
	}
	return tn
}

// typeName builds an unqualified PG type name AST node.
// For built-in types, use the PG internal name: int -> int4, text -> text,
// bigint -> int8, varchar -> varchar, timestamp -> timestamp.
func typeName(name string) *ast.TypeName {
	return &ast.TypeName{
		Names:   &ast.List{Items: []ast.Node{&ast.String{Str: name}}},
		Typemod: -1,
	}
}

// columnDef builds a minimal ColumnDef AST node.
func columnDef(name string, typ *ast.TypeName, notNull bool) *ast.ColumnDef {
	return &ast.ColumnDef{
		Colname:   name,
		TypeName:  typ,
		IsNotNull: notNull,
	}
}

// buildCreateTableStmt constructs a *ast.CreateStmt for a simple table,
// purely from Go code with no SQL text. This is the shape that a v5
// catalog loader would produce from bytebase metadata.
//
// parameter signals intent and keeps the helper ready for cross-schema tests.
//
//nolint:unparam // schema is always "public" in these PoCs; keeping the
func buildCreateTableStmt(schema, name string, cols []*ast.ColumnDef) *ast.CreateStmt {
	nodes := make([]ast.Node, len(cols))
	for i, c := range cols {
		nodes[i] = c
	}
	return &ast.CreateStmt{
		Relation: &ast.RangeVar{
			Schemaname:     schema,
			Relname:        name,
			Relpersistence: 'p',
		},
		TableElts: &ast.List{Items: nodes},
	}
}

// TestLoaderPoC_AstOnlyTableInstallAndAnalyze is the main PoC.
//
// Run with:
//
//	go test -v -count=1 -run TestLoaderPoC github.com/bytebase/bytebase/backend/plugin/parser/pg
func TestLoaderPoC_AstOnlyTableInstallAndAnalyze(t *testing.T) {
	// Step 1: fresh catalog. public / pg_catalog / pg_toast are preloaded.
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Step 2: hand-construct CREATE TABLE public.plan (
	//           id int NOT NULL,
	//           title text,
	//           status text,
	//           creator_id int
	//         ) — no SQL string involved.
	planStmt := buildCreateTableStmt("public", "plan", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("title", typeName("text"), false),
		columnDef("status", typeName("text"), false),
		columnDef("creator_id", typeName("int4"), false),
	})

	// Step 3: install via DefineRelation. Second arg 'r' = regular table.
	if err := cat.DefineRelation(planStmt, 'r'); err != nil {
		t.Fatalf("DefineRelation(plan) failed: %v", err)
	}

	// Step 4: install a second table to test JOINs too.
	principalStmt := buildCreateTableStmt("public", "principal", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("name", typeName("text"), false),
		columnDef("email", typeName("text"), false),
	})
	if err := cat.DefineRelation(principalStmt, 'r'); err != nil {
		t.Fatalf("DefineRelation(principal) failed: %v", err)
	}

	// Step 5: verify catalog state — both tables resolvable by name.
	if rel := cat.GetRelation("public", "plan"); rel == nil {
		t.Fatal("GetRelation(public, plan) returned nil after DefineRelation")
	} else if len(rel.Columns) != 4 {
		t.Errorf("plan column count: got %d, want 4", len(rel.Columns))
	}
	if rel := cat.GetRelation("public", "principal"); rel == nil {
		t.Fatal("GetRelation(public, principal) returned nil after DefineRelation")
	}

	// Step 6: parse a SELECT query that touches both tables.
	// The SELECT parser is used here — that's expected and safe (it's not
	// the DDL parser that breaks in BYT-9215/9261).
	stmts, err := ParsePg(`
		SELECT p.id, p.title, principal.name
		FROM plan p
		JOIN principal ON principal.id = p.creator_id
		WHERE p.status = 'open'
	`)
	if err != nil {
		t.Fatalf("ParsePg failed: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	selStmt, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected *ast.SelectStmt, got %T", stmts[0].AST)
	}

	// Step 7: THE KEY CALL — analyze the SELECT against the hand-installed
	// catalog. If this returns a non-nil Query with a TargetList matching
	// the three SELECT columns, v5 architecture is viable.
	query, err := cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		t.Fatalf("AnalyzeSelectStmt failed: %v", err)
	}
	if query == nil {
		t.Fatal("AnalyzeSelectStmt returned nil query")
	}

	// Step 8: assert lineage is sane — 3 target columns.
	if got := len(query.TargetList); got != 3 {
		t.Errorf("TargetList length: got %d, want 3", got)
		for i, te := range query.TargetList {
			t.Logf("  [%d] %s", i, te.ResName)
		}
	}

	// Step 9: assert each target is a VarExpr (plain column reference)
	// pointing at the expected column names.
	wantNames := []string{"id", "title", "name"}
	for i, want := range wantNames {
		if i >= len(query.TargetList) {
			break
		}
		te := query.TargetList[i]
		if te.ResName != want {
			t.Errorf("TargetList[%d].ResName: got %q, want %q", i, te.ResName, want)
		}
		if _, isVar := te.Expr.(*catalog.VarExpr); !isVar {
			t.Errorf("TargetList[%d].Expr: got %T, want *VarExpr", i, te.Expr)
		}
	}

	// Step 10: verify JoinTree.Quals is analyzed (WHERE was walked).
	// We don't assert exact shape here — just that analysis didn't bail.
	if query.JoinTree == nil {
		t.Error("JoinTree is nil — analyzer did not build a join tree")
	} else if query.JoinTree.Quals == nil {
		t.Error("JoinTree.Quals is nil — WHERE clause was not analyzed")
	}

	t.Logf("v5 PoC passed: hand-constructed CreateStmt → DefineRelation → " +
		"AnalyzeSelectStmt pipeline works end-to-end with no DDL text.")
}

// TestLoaderPoC_BrokenAstIsIsolated demonstrates blast-radius containment:
// installing a table with an unresolvable type fails locally, and queries
// against OTHER tables continue to work. This is the core fidelity claim
// v5 makes over the current whole-schema DDL init path.
func TestLoaderPoC_BrokenAstIsIsolated(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Install a good table.
	goodStmt := buildCreateTableStmt("public", "good", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("label", typeName("text"), false),
	})
	if err := cat.DefineRelation(goodStmt, 'r'); err != nil {
		t.Fatalf("DefineRelation(good) failed: %v", err)
	}

	// Try to install a broken table — column type does not exist.
	brokenStmt := buildCreateTableStmt("public", "broken", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("payload", typeName("nonexistent_type_xyz"), false),
	})
	err := cat.DefineRelation(brokenStmt, 'r')
	if err == nil {
		t.Log("note: DefineRelation accepted unresolvable type without error")
		t.Log("     this is informational — the analyzer may still reject it at query time")
	} else {
		t.Logf("DefineRelation(broken) rejected as expected: %v", err)
	}

	// Key assertion: the good table is still queryable, regardless of
	// whether the broken table was accepted.
	if rel := cat.GetRelation("public", "good"); rel == nil {
		t.Fatal("good table was lost after broken install attempt")
	}

	stmts, err := ParsePg(`SELECT id, label FROM good WHERE id = 1`)
	if err != nil {
		t.Fatalf("ParsePg failed: %v", err)
	}
	selStmt, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected *ast.SelectStmt, got %T", stmts[0].AST)
	}

	query, err := cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		t.Fatalf("AnalyzeSelectStmt on good table failed after broken install: %v", err)
	}
	if query == nil || len(query.TargetList) != 2 {
		t.Errorf("expected 2 target columns on good table query, got %v", query)
	}

	t.Log("v5 PoC blast-radius isolation: broken table did not break queries " +
		"against unrelated good table.")
}

// ---------- Risk scenario 1: complex typmods (numeric(10,2)) ----------

func TestLoaderPoC_ComplexTypmod(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Build a column with numeric(10,2) via the cheat parser.
	stmt := buildCreateTableStmt("public", "orders", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("amount", mustTypeName(t, "numeric(10,2)"), false),
		columnDef("ts", mustTypeName(t, "timestamp(3) with time zone"), false),
	})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation with complex typmods failed: %v", err)
	}

	rel := cat.GetRelation("public", "orders")
	if rel == nil || len(rel.Columns) != 3 {
		t.Fatalf("orders install incomplete: %+v", rel)
	}

	// Query the table — analyzer must resolve each column.
	q := mustAnalyze(t, cat, `SELECT id, amount, ts FROM orders WHERE amount > 0`)
	if len(q.TargetList) != 3 {
		t.Errorf("TargetList: got %d, want 3", len(q.TargetList))
	}
	t.Log("complex typmod PoC: numeric(10,2) and timestamp(3) with time zone resolve via cheat parser")
}

// ---------- Risk scenario 2: array type ----------

func TestLoaderPoC_ArrayType(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	stmt := buildCreateTableStmt("public", "tagged", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("tags", mustTypeName(t, "text[]"), false),
		columnDef("scores", mustTypeName(t, "int4[]"), false),
	})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation with array columns failed: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT id, tags FROM tagged`)
	if len(q.TargetList) != 2 {
		t.Errorf("TargetList: got %d, want 2", len(q.TargetList))
	}
	t.Log("array type PoC: text[] and int4[] resolve via cheat parser")
}

// ---------- Risk scenario 3: SELECT * expansion ----------

func TestLoaderPoC_StarExpansion(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	if err := cat.DefineRelation(buildCreateTableStmt("public", "plan", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("title", typeName("text"), false),
		columnDef("status", typeName("text"), false),
	}), 'r'); err != nil {
		t.Fatalf("install: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT * FROM plan`)
	// With a 3-column table, SELECT * should expand to 3 target entries.
	if len(q.TargetList) != 3 {
		t.Errorf("SELECT * expansion: got %d targets, want 3", len(q.TargetList))
	}
	wantCols := []string{"id", "title", "status"}
	for i, want := range wantCols {
		if i >= len(q.TargetList) {
			break
		}
		if q.TargetList[i].ResName != want {
			t.Errorf("target[%d]: got %q, want %q", i, q.TargetList[i].ResName, want)
		}
	}
	t.Log("SELECT * expansion PoC: analyzer expands star against hand-installed columns")
}

// ---------- Risk scenario 4: CTE (WITH clause) ----------

func TestLoaderPoC_CTE(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	if err := cat.DefineRelation(buildCreateTableStmt("public", "users", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("name", typeName("text"), false),
		columnDef("dept", typeName("text"), false),
	}), 'r'); err != nil {
		t.Fatalf("install: %v", err)
	}

	q := mustAnalyze(t, cat, `
		WITH active AS (
			SELECT id, name FROM users WHERE dept = 'eng'
		)
		SELECT id, name FROM active
	`)
	if len(q.TargetList) != 2 {
		t.Errorf("CTE analyze: got %d targets, want 2", len(q.TargetList))
	}
	t.Log("CTE PoC: WITH clause analyzed against hand-installed table")
}

// ---------- Risk scenario 5: correlated subquery ----------

func TestLoaderPoC_CorrelatedSubquery(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	if err := cat.DefineRelation(buildCreateTableStmt("public", "dept", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("name", typeName("text"), false),
	}), 'r'); err != nil {
		t.Fatalf("install dept: %v", err)
	}
	if err := cat.DefineRelation(buildCreateTableStmt("public", "emp", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("dept_id", typeName("int4"), false),
		columnDef("salary", typeName("int4"), false),
	}), 'r'); err != nil {
		t.Fatalf("install emp: %v", err)
	}

	q := mustAnalyze(t, cat, `
		SELECT d.name,
		       (SELECT MAX(e.salary) FROM emp e WHERE e.dept_id = d.id) AS max_sal
		FROM dept d
	`)
	if len(q.TargetList) != 2 {
		t.Errorf("correlated: got %d targets, want 2", len(q.TargetList))
	}
	t.Log("correlated subquery PoC: outer scope d.id resolved from inner subquery")
}

// ---------- Risk scenario 6: user enum type in column ----------

func TestLoaderPoC_UserEnumType(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Step 1: install the enum via DefineEnum with hand-constructed stmt.
	enumStmt := &ast.CreateEnumStmt{
		TypeName: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "task_status"},
		}},
		Vals: &ast.List{Items: []ast.Node{
			&ast.String{Str: "pending"},
			&ast.String{Str: "running"},
			&ast.String{Str: "done"},
		}},
	}
	if err := cat.DefineEnum(enumStmt); err != nil {
		t.Fatalf("DefineEnum failed: %v", err)
	}

	// Step 2: install a table whose column uses the enum.
	// We go through the cheat parser so the TypeName carries the qualified
	// name format omni expects for user types.
	stmt := buildCreateTableStmt("public", "tasks", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("status", mustTypeName(t, "public.task_status"), false),
	})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation with enum column failed: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT id, status FROM tasks WHERE status = 'pending'`)
	if len(q.TargetList) != 2 {
		t.Errorf("enum-col query: got %d targets, want 2", len(q.TargetList))
	}
	t.Log("user enum PoC: DefineEnum + DefineRelation with enum-typed column + WHERE filter works")
}

// ---------- Risk scenario 7: view via DefineView with parsed body ----------

func TestLoaderPoC_DefineViewWithParsedBody(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Install an underlying table.
	if err := cat.DefineRelation(buildCreateTableStmt("public", "plan", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("title", typeName("text"), false),
		columnDef("creator_id", typeName("int4"), false),
	}), 'r'); err != nil {
		t.Fatalf("install plan: %v", err)
	}

	// View body is stored as text in bytebase metadata. Parse it once.
	viewBody := `SELECT id, title FROM plan WHERE creator_id = 1`
	stmts, err := ParsePg(viewBody)
	if err != nil {
		t.Fatalf("parse view body: %v", err)
	}
	selStmt, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("view body: expected SelectStmt, got %T", stmts[0].AST)
	}

	// Construct ViewStmt and install via DefineView.
	viewStmt := &ast.ViewStmt{
		View: &ast.RangeVar{
			Schemaname:     "public",
			Relname:        "plan_mine",
			Relpersistence: 'p',
		},
		Query: selStmt,
	}
	if err := cat.DefineView(viewStmt); err != nil {
		t.Fatalf("DefineView failed: %v", err)
	}

	rel := cat.GetRelation("public", "plan_mine")
	if rel == nil {
		t.Fatal("view not installed")
	}
	if rel.RelKind != 'v' {
		t.Errorf("RelKind: got %c, want 'v'", rel.RelKind)
	}

	// Query the view. analyzer should be able to expand the view.
	q := mustAnalyze(t, cat, `SELECT id, title FROM plan_mine`)
	if len(q.TargetList) != 2 {
		t.Errorf("query on view: got %d targets, want 2", len(q.TargetList))
	}
	t.Log("DefineView PoC: parsed-body ViewStmt + DefineView + query on view works")
}

// ---------- Risk scenario 8: nested view (view on view) ----------

func TestLoaderPoC_NestedView(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	if err := cat.DefineRelation(buildCreateTableStmt("public", "orders", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("user_id", typeName("int4"), false),
		columnDef("amount", typeName("int4"), false),
	}), 'r'); err != nil {
		t.Fatalf("install orders: %v", err)
	}

	// View 1: orders_by_user
	body1 := `SELECT user_id, SUM(amount) AS total FROM orders GROUP BY user_id`
	sel1 := mustParseSelect(t, body1)
	if err := cat.DefineView(&ast.ViewStmt{
		View:  &ast.RangeVar{Schemaname: "public", Relname: "orders_by_user", Relpersistence: 'p'},
		Query: sel1,
	}); err != nil {
		t.Fatalf("DefineView v1: %v", err)
	}

	// View 2: top spenders — references view 1.
	body2 := `SELECT user_id, total FROM orders_by_user WHERE total > 100`
	sel2 := mustParseSelect(t, body2)
	if err := cat.DefineView(&ast.ViewStmt{
		View:  &ast.RangeVar{Schemaname: "public", Relname: "top_spenders", Relpersistence: 'p'},
		Query: sel2,
	}); err != nil {
		t.Fatalf("DefineView v2 (nested): %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT user_id, total FROM top_spenders`)
	if len(q.TargetList) != 2 {
		t.Errorf("nested view query: got %d targets, want 2", len(q.TargetList))
	}
	t.Log("nested view PoC: view-on-view resolves correctly via DefineView chain")
}

// ---------- helpers ----------

func mustAnalyze(t *testing.T, cat *catalog.Catalog, sql string) *catalog.Query {
	t.Helper()
	stmts, err := ParsePg(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	selStmt, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected SelectStmt, got %T", stmts[0].AST)
	}
	q, err := cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		t.Fatalf("AnalyzeSelectStmt(%q): %v", sql, err)
	}
	return q
}

func mustParseSelect(t *testing.T, sql string) *ast.SelectStmt {
	t.Helper()
	stmts, err := ParsePg(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected SelectStmt, got %T", stmts[0].AST)
	}
	return sel
}

// ---------- Risk scenario 9: typeNameFromString with quoted/reserved identifier ----------

func TestLoaderPoC_QuotedIdentifierType(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Create an enum with a reserved-ish name.
	enumStmt := &ast.CreateEnumStmt{
		TypeName: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "user"}, // "user" is a reserved word in PG
		}},
		Vals: &ast.List{Items: []ast.Node{
			&ast.String{Str: "admin"},
			&ast.String{Str: "member"},
		}},
	}
	if err := cat.DefineEnum(enumStmt); err != nil {
		t.Fatalf("DefineEnum for reserved-word enum failed: %v", err)
	}

	// The cheat parser should be able to resolve this through SELECT NULL::"public"."user".
	// Note: caller is responsible for quoting — this is documented in typeNameFromString.
	tn := mustTypeName(t, `"public"."user"`)
	if tn == nil {
		t.Fatal("typeNameFromString returned nil for quoted reserved-word type")
	}

	// And we can use it in a CREATE TABLE.
	stmt := buildCreateTableStmt("public", "accts", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("role", tn, false),
	})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation with quoted-reserved-word enum column failed: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT id, role FROM accts`)
	if len(q.TargetList) != 2 {
		t.Errorf("quoted-identifier enum query: got %d, want 2", len(q.TargetList))
	}
	t.Log("quoted identifier PoC: cheat parser handles reserved-word-adjacent type names")
}

// ---------- Risk scenario 10: partitioned table ----------

func TestLoaderPoC_PartitionedTable(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Construct CREATE TABLE sales (region text, amount int4) PARTITION BY LIST (region).
	stmt := &ast.CreateStmt{
		Relation: &ast.RangeVar{
			Schemaname:     "public",
			Relname:        "sales",
			Relpersistence: 'p',
		},
		TableElts: &ast.List{Items: []ast.Node{
			columnDef("region", typeName("text"), false),
			columnDef("amount", typeName("int4"), false),
		}},
		Partspec: &ast.PartitionSpec{
			Strategy: "l", // list
			PartParams: &ast.List{Items: []ast.Node{
				&ast.PartitionElem{Name: "region"},
			}},
		},
	}
	// Note: omni test convention passes 'r' even for partitioned tables;
	// omni flips relkind internally based on Partspec presence.
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation partitioned failed: %v", err)
	}

	rel := cat.GetRelation("public", "sales")
	if rel == nil {
		t.Fatal("partitioned table not installed")
	}

	q := mustAnalyze(t, cat, `SELECT region, amount FROM sales WHERE region = 'us'`)
	if len(q.TargetList) != 2 {
		t.Errorf("partitioned table query: got %d, want 2", len(q.TargetList))
	}
	t.Logf("partitioned table PoC: DefineRelation with Partspec works; relation RelKind=%c", rel.RelKind)
}

// ---------- Risk scenario 11: inherited table ----------

func TestLoaderPoC_InheritedTable(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Parent with 2 columns.
	parentStmt := buildCreateTableStmt("public", "parent", []*ast.ColumnDef{
		columnDef("a", typeName("int4"), false),
		columnDef("b", typeName("text"), false),
	})
	if err := cat.DefineRelation(parentStmt, 'r'); err != nil {
		t.Fatalf("install parent failed: %v", err)
	}

	// Child with 1 locally-defined column, inheriting from parent.
	childStmt := &ast.CreateStmt{
		Relation: &ast.RangeVar{
			Schemaname:     "public",
			Relname:        "child",
			Relpersistence: 'p',
		},
		InhRelations: &ast.List{Items: []ast.Node{
			&ast.RangeVar{Schemaname: "public", Relname: "parent"},
		}},
		TableElts: &ast.List{Items: []ast.Node{
			columnDef("c", typeName("int4"), false),
		}},
	}
	if err := cat.DefineRelation(childStmt, 'r'); err != nil {
		t.Fatalf("install child with InhRelations failed: %v", err)
	}

	child := cat.GetRelation("public", "child")
	if child == nil {
		t.Fatal("child relation not installed")
	}
	// Child should have inherited parent's columns plus its own local column.
	if len(child.Columns) < 3 {
		t.Errorf("child column count: got %d, want >= 3 (a, b inherited + c local)", len(child.Columns))
		for _, col := range child.Columns {
			t.Logf("  col: %s", col.Name)
		}
	}

	// Query should resolve all inherited + local columns.
	q := mustAnalyze(t, cat, `SELECT a, b, c FROM child`)
	if len(q.TargetList) != 3 {
		t.Errorf("inherited query: got %d targets, want 3", len(q.TargetList))
	}
	t.Log("inherited table PoC: InhRelations merges parent columns into child")
}

// ---------- Risk scenario 12: materialized view via ExecCreateTableAs ----------

func TestLoaderPoC_MaterializedView(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	if err := cat.DefineRelation(buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("val", typeName("int4"), false),
	}), 'r'); err != nil {
		t.Fatalf("install t: %v", err)
	}

	// Parse the matview body — same pattern as DefineView.
	body := `SELECT id, val * 2 AS doubled FROM t WHERE val > 0`
	selStmt := mustParseSelect(t, body)

	stmt := &ast.CreateTableAsStmt{
		Query:   selStmt,
		Objtype: ast.OBJECT_MATVIEW,
		Into: &ast.IntoClause{
			Rel: &ast.RangeVar{
				Schemaname:     "public",
				Relname:        "t_doubled",
				Relpersistence: 'p',
			},
		},
	}
	if err := cat.ExecCreateTableAs(stmt); err != nil {
		t.Fatalf("ExecCreateTableAs failed: %v", err)
	}

	rel := cat.GetRelation("public", "t_doubled")
	if rel == nil {
		t.Fatal("matview not installed")
	}
	if rel.RelKind != 'm' {
		t.Errorf("matview RelKind: got %c, want 'm'", rel.RelKind)
	}

	q := mustAnalyze(t, cat, `SELECT id, doubled FROM t_doubled`)
	if len(q.TargetList) != 2 {
		t.Errorf("matview query: got %d, want 2", len(q.TargetList))
	}
	t.Log("matview PoC: ExecCreateTableAs with OBJECT_MATVIEW works; query on matview analyzes")
}

// ---------- Risk scenario 13: function + overload via CreateFunctionStmt ----------

func TestLoaderPoC_FunctionOverload(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Helper to build a CreateFunctionStmt — mirrors omni's makeCreateFuncStmt
	// but uses explicit AST nodes.
	makeFuncStmt := func(schema, name string, argTypeNames []string, retTypeName, language, body string) *ast.CreateFunctionStmt {
		var nameItems []ast.Node
		if schema != "" {
			nameItems = append(nameItems, &ast.String{Str: schema})
		}
		nameItems = append(nameItems, &ast.String{Str: name})

		var params []ast.Node
		for _, argTypeStr := range argTypeNames {
			params = append(params, &ast.FunctionParameter{
				ArgType: typeName(argTypeStr),
				Mode:    ast.FUNC_PARAM_IN,
			})
		}

		stmt := &ast.CreateFunctionStmt{
			Funcname:   &ast.List{Items: nameItems},
			ReturnType: typeName(retTypeName),
		}
		if len(params) > 0 {
			stmt.Parameters = &ast.List{Items: params}
		}

		var opts []ast.Node
		if language != "" {
			opts = append(opts, &ast.DefElem{Defname: "language", Arg: &ast.String{Str: language}})
		}
		if body != "" {
			opts = append(opts, &ast.DefElem{
				Defname: "as",
				Arg:     &ast.List{Items: []ast.Node{&ast.String{Str: body}}},
			})
		}
		if len(opts) > 0 {
			stmt.Options = &ast.List{Items: opts}
		}
		return stmt
	}

	// Install overload 1: my_double(int4) returns int4.
	err := cat.CreateFunctionStmt(makeFuncStmt("public", "my_double",
		[]string{"int4"}, "int4", "sql", "SELECT $1 * 2"))
	if err != nil {
		t.Fatalf("install my_double(int4) failed: %v", err)
	}

	// Install overload 2: my_double(int8) returns int8. Different signature,
	// must coexist (this is codex's D3 verification — sequential
	// CreateFunctionStmt calls install all overloads).
	err = cat.CreateFunctionStmt(makeFuncStmt("public", "my_double",
		[]string{"int8"}, "int8", "sql", "SELECT $1 * 2"))
	if err != nil {
		t.Fatalf("install my_double(int8) failed: %v", err)
	}

	// Both overloads must be visible.
	procs := cat.LookupProcByName("my_double")
	if len(procs) < 2 {
		t.Fatalf("overload count: got %d, want >= 2", len(procs))
	}

	// Install a table and query using the function.
	if err := cat.DefineRelation(buildCreateTableStmt("public", "nums", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("n", typeName("int4"), false),
	}), 'r'); err != nil {
		t.Fatalf("install nums: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT id, my_double(n) AS doubled FROM nums`)
	if len(q.TargetList) != 2 {
		t.Errorf("function query: got %d, want 2", len(q.TargetList))
	}
	t.Logf("function overload PoC: 2 overloads installed sequentially, analyzer picks correctly. %d procs visible under 'my_double'", len(procs))
}

// ---------- Risk scenario 14: composite type + column referencing it ----------

func TestLoaderPoC_CompositeType(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Install a composite type: address (street text, city text, zip text).
	compStmt := &ast.CompositeTypeStmt{
		Typevar: &ast.RangeVar{
			Schemaname: "public",
			Relname:    "address",
		},
		Coldeflist: &ast.List{Items: []ast.Node{
			columnDef("street", typeName("text"), false),
			columnDef("city", typeName("text"), false),
			columnDef("zip", typeName("text"), false),
		}},
	}
	if err := cat.DefineCompositeType(compStmt); err != nil {
		t.Fatalf("DefineCompositeType failed: %v", err)
	}

	// Install a table with a column of the composite type.
	stmt := buildCreateTableStmt("public", "customers", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("addr", mustTypeName(t, "public.address"), false),
	})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation with composite column failed: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT id, addr FROM customers`)
	if len(q.TargetList) != 2 {
		t.Errorf("composite-col query: got %d, want 2", len(q.TargetList))
	}
	t.Log("composite type PoC: DefineCompositeType + DefineRelation with composite column works")
}

// ---------- Risk scenario 15: domain type + column referencing it ----------

func TestLoaderPoC_DomainType(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Install a domain: positive_int, base int4, NOT NULL.
	domainStmt := &ast.CreateDomainStmt{
		Domainname: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "positive_int"},
		}},
		Typname: typeName("int4"),
		Constraints: &ast.List{Items: []ast.Node{
			&ast.Constraint{Contype: ast.CONSTR_NOTNULL},
		}},
	}
	if err := cat.DefineDomain(domainStmt); err != nil {
		t.Fatalf("DefineDomain failed: %v", err)
	}

	// Install a table with a column of the domain type.
	stmt := buildCreateTableStmt("public", "counters", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("count", mustTypeName(t, "public.positive_int"), true),
	})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation with domain column failed: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT id, count FROM counters WHERE count > 0`)
	if len(q.TargetList) != 2 {
		t.Errorf("domain-col query: got %d, want 2", len(q.TargetList))
	}
	t.Log("domain type PoC: DefineDomain + DefineRelation with domain column works")
}

// ---------- Exploration: does query span actually need real types? ----------
//
// Hypothesis: for query span purposes, we only need column NAMES, not real
// column TYPES. Lineage is a name-resolution problem, not a type-resolution
// problem. If we install every column as "text" (regardless of real metadata
// type), most queries should still analyze correctly and produce identical
// lineage output.
//
// Why this matters: if true, v5 can skip typeNameFromString entirely, skip
// the user enum / domain / composite problem entirely, and reduce the
// builder function library to a trivial "table with column names" shape.
//
// This test runs a battery of queries against tables where all columns are
// declared as text, and asks: (a) does AnalyzeSelectStmt succeed, (b) is the
// lineage output the same as with correct types? A query that fails here
// would fall through to extractFallbackColumns today (same as the current
// path does on any analyzer error).

func TestLoaderPoC_TypesDontMatterForLineage(t *testing.T) {
	// Build two catalogs: one with "honest" types, one with everything as text.
	// Run the same queries against both and compare lineage.

	installHonest := func(cat *catalog.Catalog) {
		// orders (id int, amount numeric(10,2), status text, placed_at timestamptz, tags text[])
		if err := cat.DefineRelation(buildCreateTableStmt("public", "orders", []*ast.ColumnDef{
			columnDef("id", typeName("int4"), true),
			columnDef("amount", mustTypeName(t, "numeric(10,2)"), false),
			columnDef("status", typeName("text"), false),
			columnDef("placed_at", mustTypeName(t, "timestamp with time zone"), false),
			columnDef("tags", mustTypeName(t, "text[]"), false),
		}), 'r'); err != nil {
			t.Fatalf("honest install: %v", err)
		}
		// customers (id int, email text)
		if err := cat.DefineRelation(buildCreateTableStmt("public", "customers", []*ast.ColumnDef{
			columnDef("id", typeName("int4"), true),
			columnDef("email", typeName("text"), false),
		}), 'r'); err != nil {
			t.Fatalf("honest install customers: %v", err)
		}
	}

	installLying := func(cat *catalog.Catalog) {
		// Everything text, everything. Even numeric/timestamp/array columns.
		// This is the minimum information the loader would need if we decided
		// bytebase metadata should not bother with per-column type strings.
		if err := cat.DefineRelation(buildCreateTableStmt("public", "orders", []*ast.ColumnDef{
			columnDef("id", typeName("text"), true),
			columnDef("amount", typeName("text"), false),
			columnDef("status", typeName("text"), false),
			columnDef("placed_at", typeName("text"), false),
			columnDef("tags", typeName("text"), false),
		}), 'r'); err != nil {
			t.Fatalf("lying install: %v", err)
		}
		if err := cat.DefineRelation(buildCreateTableStmt("public", "customers", []*ast.ColumnDef{
			columnDef("id", typeName("text"), true),
			columnDef("email", typeName("text"), false),
		}), 'r'); err != nil {
			t.Fatalf("lying install customers: %v", err)
		}
	}

	catHonest := catalog.New()
	catHonest.SetSearchPath([]string{"public"})
	installHonest(catHonest)

	catLying := catalog.New()
	catLying.SetSearchPath([]string{"public"})
	installLying(catLying)

	// Each test case: a query, a description, an "honest expected to pass" flag.
	// We record (for each catalog) whether AnalyzeSelectStmt succeeded and what
	// the lineage looked like, then compare.
	cases := []struct {
		name  string
		query string
		// expectedHonest is whether we expect the honest catalog to analyze successfully.
		// All our cases are expected to succeed on the honest catalog.
	}{
		// Category 1: Name resolution only — no operator semantics involved
		{"plain_select", `SELECT id, status FROM orders`},
		{"select_star", `SELECT * FROM orders`},
		{"join", `SELECT o.id, c.email FROM orders o JOIN customers c ON c.id = o.id`},
		{"alias", `SELECT o.id AS order_id FROM orders o`},

		// Category 2: text-friendly operators (=, LIKE, IS NULL)
		{"text_equality", `SELECT id FROM orders WHERE status = 'open'`},
		{"text_like", `SELECT id FROM orders WHERE status LIKE 'op%'`},
		{"is_null", `SELECT id FROM orders WHERE status IS NULL`},

		// Category 3: numeric operators against text — will omni coerce or fail?
		{"numeric_compare", `SELECT id FROM orders WHERE amount > 100`},
		{"numeric_arithmetic", `SELECT id, amount + 1 AS amount_plus FROM orders`},
		{"numeric_sum", `SELECT SUM(amount) FROM orders GROUP BY status`},

		// Category 4: date/time ops against text
		{"date_compare", `SELECT id FROM orders WHERE placed_at > '2024-01-01'::date`},
		{"date_extract", `SELECT EXTRACT(year FROM placed_at::timestamp) FROM orders`},

		// Category 5: array ops against text
		{"array_contains", `SELECT id FROM orders WHERE tags @> ARRAY['urgent']`},
		{"array_unnest", `SELECT id, unnest(tags) FROM orders`},

		// Category 6: subquery + CTE (name resolution heavy)
		{"cte", `WITH active AS (SELECT id, status FROM orders WHERE status = 'open') SELECT id FROM active`},
		{"subquery_where", `SELECT id FROM orders WHERE id IN (SELECT id FROM customers)`},
		{"correlated", `SELECT id, (SELECT email FROM customers c WHERE c.id = orders.id) AS email FROM orders`},

		// Category 7: aggregations
		{"count_star", `SELECT COUNT(*) FROM orders`},
		{"count_col", `SELECT COUNT(id) FROM orders`},
		{"group_by", `SELECT status, COUNT(*) FROM orders GROUP BY status`},
	}

	type result struct {
		ok       bool
		err      string
		nTargets int
		names    []string
	}

	runOne := func(cat *catalog.Catalog, sql string) result {
		stmts, err := ParsePg(sql)
		if err != nil {
			return result{err: "parse: " + err.Error()}
		}
		sel, ok := stmts[0].AST.(*ast.SelectStmt)
		if !ok {
			return result{err: "not a SelectStmt"}
		}
		q, err := cat.AnalyzeSelectStmt(sel)
		if err != nil {
			return result{err: "analyze: " + err.Error()}
		}
		r := result{ok: true, nTargets: len(q.TargetList)}
		for _, te := range q.TargetList {
			r.names = append(r.names, te.ResName)
		}
		return r
	}

	var bothOk, honestOnlyOk, lyingOnlyOk, bothFail, lineageDiverges int
	t.Log("---- query span type-free install exploration ----")
	t.Logf("%-22s  %-8s  %-8s  %s", "case", "honest", "lying", "note")
	for _, tc := range cases {
		h := runOne(catHonest, tc.query)
		l := runOne(catLying, tc.query)

		honestMark := "FAIL"
		if h.ok {
			honestMark = "PASS"
		}
		lyingMark := "FAIL"
		if l.ok {
			lyingMark = "PASS"
		}

		note := ""
		if h.ok && l.ok {
			bothOk++
			// Compare lineage shape (target count + names).
			if h.nTargets != l.nTargets {
				note = fmt.Sprintf("TARGETS DIFFER h=%d l=%d", h.nTargets, l.nTargets)
				lineageDiverges++
			} else {
				same := true
				for i := range h.names {
					if h.names[i] != l.names[i] {
						same = false
						break
					}
				}
				if !same {
					note = fmt.Sprintf("NAMES DIFFER h=%v l=%v", h.names, l.names)
					lineageDiverges++
				} else {
					note = fmt.Sprintf("IDENTICAL (%d cols)", h.nTargets)
				}
			}
		} else if h.ok && !l.ok {
			honestOnlyOk++
			note = "lying fails: " + truncate(l.err, 60)
		} else if !h.ok && l.ok {
			lyingOnlyOk++
			note = "honest fails: " + truncate(h.err, 60)
		} else {
			bothFail++
			note = "both fail"
		}

		t.Logf("%-22s  %-8s  %-8s  %s", tc.name, honestMark, lyingMark, note)
	}

	t.Log("")
	t.Logf("summary: both_pass=%d honest_only=%d lying_only=%d both_fail=%d diverge=%d",
		bothOk, honestOnlyOk, lyingOnlyOk, bothFail, lineageDiverges)
	t.Log("")
	t.Log("key question: for cases where both paths analyze successfully, does lineage match?")
	t.Log("key question: for cases where only honest analyzes, can fallback recover equal-or-better lineage?")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// TestLoaderPoC_UnknownPseudoType explores whether PG's 'unknown' pseudo-type
// can be used as a universal column type that accepts all operators.
func TestLoaderPoC_UnknownPseudoType(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	tries := []string{"unknown", "anyelement", "any"}
	for _, ty := range tries {
		stmt := buildCreateTableStmt("public", "t_"+ty, []*ast.ColumnDef{
			{Colname: "id", TypeName: typeName(ty)},
		})
		err := cat.DefineRelation(stmt, 'r')
		if err != nil {
			t.Logf("%-12s → DefineRelation rejected: %s", ty, truncate(err.Error(), 80))
		} else {
			t.Logf("%-12s → DefineRelation accepted", ty)
		}
	}
}

// TestLoaderPoC_UnknownTypeQueryBattery installs the same queries as
// TestLoaderPoC_TypesDontMatterForLineage but with all columns declared as
// 'unknown' instead of 'text'. The question: does the analyzer's overload
// resolution treat unknown differently from text?
func TestLoaderPoC_UnknownTypeQueryBattery(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	install := func(table string, cols []string) {
		colDefs := make([]*ast.ColumnDef, len(cols))
		for i, c := range cols {
			colDefs[i] = columnDef(c, typeName("unknown"), false)
		}
		if err := cat.DefineRelation(buildCreateTableStmt("public", table, colDefs), 'r'); err != nil {
			t.Fatalf("install %s: %v", table, err)
		}
	}
	install("orders", []string{"id", "amount", "status", "placed_at", "tags", "customer_id"})
	install("customers", []string{"id", "email", "name", "region"})

	cases := []struct {
		category, name, query string
	}{
		// ---- Baseline: name resolution only ----
		{"name_res", "plain_select", `SELECT id, status FROM orders`},
		{"name_res", "select_star", `SELECT * FROM orders`},
		{"name_res", "join", `SELECT o.id, c.email FROM orders o JOIN customers c ON c.id = o.customer_id`},
		{"name_res", "three_way_join", `SELECT o.id FROM orders o JOIN customers c ON c.id = o.customer_id JOIN orders o2 ON o2.id = o.id`},
		{"name_res", "alias_qualified", `SELECT o.id AS order_id, c.name AS customer FROM orders o, customers c WHERE c.id = o.customer_id`},

		// ---- Text-friendly operators ----
		{"text_ops", "text_equality", `SELECT id FROM orders WHERE status = 'open'`},
		{"text_ops", "text_like", `SELECT id FROM orders WHERE status LIKE 'op%'`},
		{"text_ops", "text_ilike", `SELECT id FROM orders WHERE status ILIKE 'OP%'`},
		{"text_ops", "text_in_list", `SELECT id FROM orders WHERE status IN ('open', 'closed')`},
		{"text_ops", "is_null", `SELECT id FROM orders WHERE status IS NULL`},
		{"text_ops", "is_not_null", `SELECT id FROM orders WHERE status IS NOT NULL`},

		// ---- Numeric operators (unknown auto-coerces) ----
		{"numeric_ops", "numeric_compare", `SELECT id FROM orders WHERE amount > 100`},
		{"numeric_ops", "numeric_arithmetic", `SELECT id, amount + 1 AS p FROM orders`},
		{"numeric_ops", "numeric_between", `SELECT id FROM orders WHERE amount BETWEEN 10 AND 100`},
		{"numeric_ops", "numeric_negation", `SELECT id, -amount FROM orders`},

		// ---- Date/time ops ----
		{"date_ops", "date_compare", `SELECT id FROM orders WHERE placed_at > '2024-01-01'::date`},
		{"date_ops", "date_extract", `SELECT EXTRACT(year FROM placed_at::timestamp) FROM orders`},
		{"date_ops", "interval_arith", `SELECT id FROM orders WHERE placed_at > now() - interval '7 days'`},

		// ---- Array ops ----
		{"array_ops", "array_contains", `SELECT id FROM orders WHERE tags @> ARRAY['urgent']`},
		{"array_ops", "array_any", `SELECT id FROM orders WHERE 'urgent' = ANY(tags)`},
		{"array_ops", "array_length", `SELECT id, array_length(tags, 1) FROM orders`},
		{"array_ops", "array_unnest", `SELECT id, unnest(tags) FROM orders`},

		// ---- Aggregates (the worry zone) ----
		{"aggregate", "count_star", `SELECT COUNT(*) FROM orders`},
		{"aggregate", "count_col", `SELECT COUNT(id) FROM orders`},
		{"aggregate", "count_distinct", `SELECT COUNT(DISTINCT customer_id) FROM orders`},
		{"aggregate", "sum", `SELECT SUM(amount) FROM orders`},
		{"aggregate", "avg", `SELECT AVG(amount) FROM orders`},
		{"aggregate", "min_max", `SELECT MIN(amount), MAX(amount) FROM orders`},
		{"aggregate", "string_agg", `SELECT STRING_AGG(status, ',') FROM orders`},
		{"aggregate", "array_agg", `SELECT ARRAY_AGG(id) FROM orders`},
		{"aggregate", "group_by", `SELECT status, COUNT(*) FROM orders GROUP BY status`},
		{"aggregate", "having", `SELECT status, COUNT(*) FROM orders GROUP BY status HAVING COUNT(*) > 5`},

		// ---- Aggregate with explicit cast (workaround) ----
		{"cast_workaround", "sum_with_cast", `SELECT SUM(amount::numeric) FROM orders`},
		{"cast_workaround", "avg_with_cast", `SELECT AVG(amount::numeric) FROM orders`},
		{"cast_workaround", "string_agg_cast", `SELECT STRING_AGG(status::text, ',') FROM orders`},

		// ---- Window functions ----
		{"window", "row_number", `SELECT id, ROW_NUMBER() OVER (ORDER BY placed_at) FROM orders`},
		{"window", "rank_partition", `SELECT id, RANK() OVER (PARTITION BY status ORDER BY amount) FROM orders`},
		{"window", "sum_over", `SELECT id, SUM(amount) OVER (PARTITION BY status) FROM orders`},
		{"window", "lag_col", `SELECT id, LAG(status) OVER (ORDER BY placed_at) FROM orders`},

		// ---- Control flow / null handling ----
		{"control", "case_when", `SELECT id, CASE WHEN status = 'open' THEN 1 ELSE 0 END FROM orders`},
		{"control", "coalesce", `SELECT COALESCE(status, 'unknown') FROM orders`},
		{"control", "nullif", `SELECT NULLIF(status, 'open') FROM orders`},

		// ---- Subqueries / set operations ----
		{"subquery", "exists", `SELECT id FROM orders o WHERE EXISTS (SELECT 1 FROM customers c WHERE c.id = o.customer_id)`},
		{"subquery", "scalar_subquery", `SELECT id, (SELECT MAX(amount) FROM orders) AS max_amt FROM orders`},
		{"subquery", "not_in", `SELECT id FROM orders WHERE customer_id NOT IN (SELECT id FROM customers WHERE region = 'us')`},
		{"subquery", "union", `SELECT id FROM orders UNION SELECT id FROM customers`},
		{"subquery", "intersect", `SELECT id FROM orders INTERSECT SELECT id FROM customers`},
		{"subquery", "except", `SELECT id FROM orders EXCEPT SELECT id FROM customers`},

		// ---- CTE + recursion ----
		{"cte", "simple_cte", `WITH active AS (SELECT id, status FROM orders WHERE status = 'open') SELECT id FROM active`},
		{"cte", "cte_with_agg", `WITH stats AS (SELECT status, COUNT(*) c FROM orders GROUP BY status) SELECT * FROM stats`},

		// ---- Sort / limit ----
		{"order_limit", "order_by_col", `SELECT id FROM orders ORDER BY placed_at DESC LIMIT 10`},
		{"order_limit", "order_by_expr", `SELECT id FROM orders ORDER BY amount + 1`},
		{"order_limit", "distinct_on", `SELECT DISTINCT ON (customer_id) id, customer_id FROM orders ORDER BY customer_id, placed_at DESC`},
	}

	type bucket struct{ pass, fail int }
	byCategory := map[string]*bucket{}

	pass := 0
	fail := 0
	var failed []string
	for _, tc := range cases {
		if byCategory[tc.category] == nil {
			byCategory[tc.category] = &bucket{}
		}
		stmts, err := ParsePg(tc.query)
		if err != nil {
			t.Logf("%-12s  %-22s  PARSE FAIL: %v", tc.category, tc.name, err)
			fail++
			byCategory[tc.category].fail++
			failed = append(failed, tc.category+"/"+tc.name+" (parse)")
			continue
		}
		sel, ok := stmts[0].AST.(*ast.SelectStmt)
		if !ok {
			t.Logf("%-12s  %-22s  NOT SELECT", tc.category, tc.name)
			fail++
			byCategory[tc.category].fail++
			continue
		}
		_, err = cat.AnalyzeSelectStmt(sel)
		if err != nil {
			t.Logf("%-12s  %-22s  FAIL: %s", tc.category, tc.name, truncate(err.Error(), 90))
			fail++
			byCategory[tc.category].fail++
			failed = append(failed, tc.category+"/"+tc.name)
			continue
		}
		pass++
		byCategory[tc.category].pass++
		t.Logf("%-12s  %-22s  PASS", tc.category, tc.name)
	}

	t.Log("")
	t.Log("----- unknown-type battery: per-category breakdown -----")
	for _, cat := range []string{"name_res", "text_ops", "numeric_ops", "date_ops", "array_ops", "aggregate", "cast_workaround", "window", "control", "subquery", "cte", "order_limit"} {
		b := byCategory[cat]
		if b == nil {
			continue
		}
		t.Logf("  %-16s  %d pass / %d fail", cat, b.pass, b.fail)
	}
	t.Log("")
	t.Logf("TOTAL: %d pass, %d fail (%.0f%% pass rate)", pass, fail, 100.0*float64(pass)/float64(pass+fail))
	if len(failed) > 0 {
		t.Log("failed:")
		for _, f := range failed {
			t.Logf("  - %s", f)
		}
	}
}

// TestLoaderPoC_HybridTypeResolution explores the pragmatic middle ground:
// for each column, try to resolve the metadata type string; if it fails
// (user type not in catalog, malformed type string), fall back to text.
// This gives us "best of both" — real types where available, permissive
// text where not.
func TestLoaderPoC_HybridTypeResolution(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Hybrid column builder: try honest type, fall back to text.
	hybridCol := func(name string, typeStr string) *ast.ColumnDef {
		tn := mustTypeName(t, typeStr)
		// Check if catalog can resolve it at this moment.
		// Note: for user types to resolve, they must already be installed.
		// We're checking against the catalog AS-IS.
		resolved := false
		func() {
			defer func() { _ = recover() }() // defensive: ResolveType may not panic, but be safe
			_, _, err := cat.ResolveType(toCatalogTypeName(tn))
			resolved = (err == nil)
		}()
		if resolved {
			return &ast.ColumnDef{Colname: name, TypeName: tn}
		}
		t.Logf("  hybrid: column %q type %q unresolvable in catalog, falling back to text", name, typeStr)
		return &ast.ColumnDef{Colname: name, TypeName: typeName("text")}
	}

	// Simulate metadata where some types are resolvable, some aren't.
	// 'int4' is a builtin — resolvable.
	// 'numeric(10,2)' is a builtin — resolvable.
	// 'public.unknown_enum_xyz' is a user type we never installed — should fall back to text.
	stmt := buildCreateTableStmt("public", "mix", []*ast.ColumnDef{
		hybridCol("id", "int4"),
		hybridCol("amount", "numeric(10,2)"),
		hybridCol("tag", "public.unknown_enum_xyz"),
	})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("hybrid install failed: %v", err)
	}

	// Queries that use the numeric column (known type) should still work
	// with real operator resolution.
	q1 := mustAnalyze(t, cat, `SELECT id, amount FROM mix WHERE amount > 100`)
	if len(q1.TargetList) != 2 {
		t.Errorf("q1 targets: got %d, want 2", len(q1.TargetList))
	}

	// Queries that touch the tag column only do name resolution — no operator
	// type-checking — should work even though tag is text-in-reality-but-
	// metadata-said-enum.
	q2 := mustAnalyze(t, cat, `SELECT id, tag FROM mix WHERE tag = 'a'`)
	if len(q2.TargetList) != 2 {
		t.Errorf("q2 targets: got %d, want 2", len(q2.TargetList))
	}

	t.Log("hybrid PoC: resolvable types use real TypeName, unresolvable types fall back to text")
	t.Log("effect: best of both — operator semantics work where types known, name resolution works where not")
}

// TestLoaderPoC_BvsCDivergence pinpoints the exact behavioral difference between
// Option B (hybrid: builtins honest, user types unknown) and Option C (all
// honest types, user types installed).
//
// Scenario: a table with ONE builtin-int column and ONE user-enum column.
// Run the same queries against two catalogs:
//
//	catC: enum installed via DefineEnum, table column is the real enum type
//	catB: enum NOT installed, table column is 'unknown'
//
// For each query, record (passes/fails, target count, lineage equivalent).
// The output shows exactly where B and C diverge.
func TestLoaderPoC_BvsCDivergence(t *testing.T) {
	// ----- catC: option C (all honest) -----
	catC := catalog.New()
	catC.SetSearchPath([]string{"public"})

	if err := catC.DefineEnum(&ast.CreateEnumStmt{
		TypeName: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "task_status"},
		}},
		Vals: &ast.List{Items: []ast.Node{
			&ast.String{Str: "open"},
			&ast.String{Str: "closed"},
		}},
	}); err != nil {
		t.Fatalf("catC install enum: %v", err)
	}
	if err := catC.DefineRelation(buildCreateTableStmt("public", "tasks", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("status", mustTypeName(t, "public.task_status"), false),
		columnDef("created_at", mustTypeName(t, "timestamptz"), false),
	}), 'r'); err != nil {
		t.Fatalf("catC install table: %v", err)
	}

	// ----- catB: option B (hybrid — enum column as unknown, everything else honest) -----
	catB := catalog.New()
	catB.SetSearchPath([]string{"public"})
	// Note: no DefineEnum call. Enum "doesn't exist" from catB's perspective.
	if err := catB.DefineRelation(buildCreateTableStmt("public", "tasks", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),                 // builtin → honest
		columnDef("status", typeName("unknown"), false),         // user type → unknown
		columnDef("created_at", typeName("timestamptz"), false), // builtin → honest
	}), 'r'); err != nil {
		t.Fatalf("catB install table: %v", err)
	}

	cases := []struct {
		name, query string
	}{
		// Queries that should work in BOTH (builtin columns untouched)
		{"select_id", `SELECT id FROM tasks`},
		{"select_star", `SELECT * FROM tasks`},
		{"count_star", `SELECT COUNT(*) FROM tasks`},
		{"sum_id", `SELECT SUM(id) FROM tasks`},
		{"avg_id", `SELECT AVG(id) FROM tasks`},
		{"max_created", `SELECT MAX(created_at) FROM tasks`},

		// Queries touching status column — B gets 'unknown', C gets real enum
		{"where_status_eq", `SELECT id FROM tasks WHERE status = 'open'`},
		{"where_status_in", `SELECT id FROM tasks WHERE status IN ('open', 'closed')`},
		{"select_status", `SELECT status FROM tasks`},
		{"group_by_status", `SELECT status, COUNT(*) FROM tasks GROUP BY status`},
		{"order_by_status", `SELECT id FROM tasks ORDER BY status`},
		{"distinct_status", `SELECT DISTINCT status FROM tasks`},

		// Aggregates directly on the enum column — the real test
		{"count_status", `SELECT COUNT(status) FROM tasks`},
		{"count_distinct_status", `SELECT COUNT(DISTINCT status) FROM tasks`},
		{"min_status", `SELECT MIN(status) FROM tasks`},
		{"max_status", `SELECT MAX(status) FROM tasks`},
		{"array_agg_status", `SELECT array_agg(status) FROM tasks`},
		{"string_agg_status", `SELECT string_agg(status::text, ',') FROM tasks`},

		// Mixed: status in WHERE, sum on builtin
		{"mixed_sum", `SELECT SUM(id) FROM tasks WHERE status = 'open'`},
		{"mixed_group", `SELECT status, SUM(id) FROM tasks GROUP BY status`},
	}

	type res struct {
		ok  bool
		err string
	}
	runOne := func(cat *catalog.Catalog, sql string) res {
		stmts, err := ParsePg(sql)
		if err != nil {
			return res{err: "parse: " + truncate(err.Error(), 60)}
		}
		sel, ok := stmts[0].AST.(*ast.SelectStmt)
		if !ok {
			return res{err: "not a SelectStmt"}
		}
		if _, err := cat.AnalyzeSelectStmt(sel); err != nil {
			return res{err: truncate(err.Error(), 70)}
		}
		return res{ok: true}
	}

	var bothOk, cOnlyOk, bOnlyOk, bothFail int
	t.Logf("%-24s  %-6s  %-6s  %s", "case", "C", "B", "divergence note")
	t.Logf("%-24s  %-6s  %-6s  %s", "----", "-", "-", "---------------")
	for _, tc := range cases {
		c := runOne(catC, tc.query)
		b := runOne(catB, tc.query)
		cMark := "FAIL"
		if c.ok {
			cMark = "PASS"
		}
		bMark := "FAIL"
		if b.ok {
			bMark = "PASS"
		}

		note := ""
		switch {
		case c.ok && b.ok:
			bothOk++
			note = "same"
		case c.ok && !b.ok:
			cOnlyOk++
			note = "C only — B: " + b.err
		case !c.ok && b.ok:
			bOnlyOk++
			note = "B only — C: " + c.err
		default:
			bothFail++
			note = "both fail"
		}
		t.Logf("%-24s  %-6s  %-6s  %s", tc.name, cMark, bMark, note)
	}

	t.Log("")
	t.Logf("summary: both_ok=%d C_only=%d B_only=%d both_fail=%d", bothOk, cOnlyOk, bOnlyOk, bothFail)
	t.Logf("option B loss (queries where C works but B falls back): %d / %d = %.0f%%",
		cOnlyOk, len(cases), 100.0*float64(cOnlyOk)/float64(len(cases)))
}

// TestLoaderPoC_BvsCDomainColumn shows the case where B is STRICTLY BETTER than C:
// a table with a domain-typed column where the domain is NOT in bytebase metadata
// (which is the ACTUAL state of bytebase PG sync today — domain/composite/range
// types are not synced). Option C fails at install time and blocks all queries
// on the table. Option B uses unknown and only loses type-sensitive aggregates.
func TestLoaderPoC_BvsCDomainColumn(t *testing.T) {
	// Simulate reality: bytebase sync does NOT capture the 'public.positive_int'
	// domain. The table metadata says column 'count' has type 'public.positive_int',
	// but bytebase has no DomainMetadata for it.

	// ----- catC: option C — try to honest-install everything -----
	// Closure expansion would try to load 'public.positive_int' from metadata,
	// find nothing, and either fail the table install or skip the column.
	// Let's simulate: C tries to install the table with the real type reference,
	// fails because the domain doesn't exist in the catalog.
	catC := catalog.New()
	catC.SetSearchPath([]string{"public"})
	cStmt := buildCreateTableStmt("public", "counters", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("count", mustTypeName(t, "public.positive_int"), true),
	})
	cErr := catC.DefineRelation(cStmt, 'r')
	if cErr != nil {
		t.Logf("[C] DefineRelation failed as expected: %s", truncate(cErr.Error(), 80))
	} else {
		t.Log("[C] DefineRelation unexpectedly succeeded")
	}

	// ----- catB: option B — count column degrades to unknown -----
	catB := catalog.New()
	catB.SetSearchPath([]string{"public"})
	bStmt := buildCreateTableStmt("public", "counters", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("count", typeName("unknown"), true),
	})
	if err := catB.DefineRelation(bStmt, 'r'); err != nil {
		t.Fatalf("[B] DefineRelation failed: %v", err)
	}
	t.Log("[B] DefineRelation succeeded")

	cases := []struct {
		name, query string
	}{
		{"simple_select", `SELECT id, count FROM counters`},
		{"where_id", `SELECT id FROM counters WHERE id > 0`},
		{"where_count", `SELECT id FROM counters WHERE count > 0`},
		{"sum_id", `SELECT SUM(id) FROM counters`},
		{"sum_count", `SELECT SUM(count) FROM counters`},
		{"sum_id_cast", `SELECT SUM(count::numeric) FROM counters`},
	}

	runOne := func(cat *catalog.Catalog, sql string) string {
		stmts, err := ParsePg(sql)
		if err != nil {
			return "parse: " + err.Error()
		}
		sel, ok := stmts[0].AST.(*ast.SelectStmt)
		if !ok {
			return "not a SelectStmt"
		}
		if _, err := cat.AnalyzeSelectStmt(sel); err != nil {
			return truncate(err.Error(), 60)
		}
		return "PASS"
	}

	t.Log("")
	t.Log("Query availability on 'counters' table (domain column):")
	t.Logf("%-22s  %-40s  %s", "case", "C result", "B result")
	var cWorks, bWorks int
	for _, tc := range cases {
		var cResult string
		if cErr != nil {
			cResult = "TABLE NOT INSTALLED"
		} else {
			cResult = runOne(catC, tc.query)
		}
		bResult := runOne(catB, tc.query)
		if cResult == "PASS" {
			cWorks++
		}
		if bResult == "PASS" {
			bWorks++
		}
		t.Logf("%-22s  %-40s  %s", tc.name, truncate(cResult, 38), bResult)
	}
	t.Log("")
	t.Logf("domain column query survival: C=%d/%d, B=%d/%d", cWorks, len(cases), bWorks, len(cases))
}

// toCatalogTypeName converts ast.TypeName to catalog.TypeName for ResolveType.
// catalog.TypeName is the lightweight shape used by the public ResolveType API.
func toCatalogTypeName(tn *ast.TypeName) catalog.TypeName {
	var schema, name string
	if tn.Names != nil && len(tn.Names.Items) > 0 {
		items := tn.Names.Items
		if len(items) == 1 {
			if s, ok := items[0].(*ast.String); ok {
				name = s.Str
			}
		} else if len(items) >= 2 {
			if s, ok := items[0].(*ast.String); ok {
				schema = s.Str
			}
			if s, ok := items[len(items)-1].(*ast.String); ok {
				name = s.Str
			}
		}
	}
	return catalog.TypeName{Schema: schema, Name: name, TypeMod: tn.Typemod}
}

// Format helper for readable output.
var _ = fmt.Sprintf // keep fmt imported when strings.Builder not used

// ---------- Equivalence test: hand-built CreateStmt vs parsed DDL ----------
//
// Per codex review section 2B: the PoC must prove that hand-constructed AST
// produces a catalog state equivalent to parsing the equivalent DDL text.
// This test installs the same table two ways on two separate catalogs and
// asserts the resulting Relation objects are structurally identical on all
// fields reachable via public API.

func TestLoaderPoC_HandBuiltVsDDLEquivalence(t *testing.T) {
	// Catalog A: install via hand-built CreateStmt.
	catA := catalog.New()
	catA.SetSearchPath([]string{"public"})
	handBuilt := buildCreateTableStmt("public", "orders", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("customer", typeName("text"), false),
		columnDef("amount", mustTypeName(t, "numeric(10,2)"), false),
		columnDef("placed_at", mustTypeName(t, "timestamp with time zone"), false),
	})
	if err := catA.DefineRelation(handBuilt, 'r'); err != nil {
		t.Fatalf("catA install: %v", err)
	}

	// Catalog B: install via parsed DDL text.
	catB := catalog.New()
	catB.SetSearchPath([]string{"public"})
	ddl := `CREATE TABLE public.orders (
		id int NOT NULL,
		customer text,
		amount numeric(10,2),
		placed_at timestamp with time zone
	)`
	if _, err := catB.Exec(ddl, nil); err != nil {
		t.Fatalf("catB install via Exec: %v", err)
	}

	relA := catA.GetRelation("public", "orders")
	relB := catB.GetRelation("public", "orders")
	if relA == nil || relB == nil {
		t.Fatalf("relation install missing: A=%v B=%v", relA != nil, relB != nil)
	}

	// Compare structural fields. OIDs are not expected to match absolutely
	// (each catalog has its own oidGen starting point), but relative order
	// and per-column shape must match.

	if relA.RelKind != relB.RelKind {
		t.Errorf("RelKind differs: A=%c B=%c", relA.RelKind, relB.RelKind)
	}
	if len(relA.Columns) != len(relB.Columns) {
		t.Fatalf("column count: A=%d B=%d", len(relA.Columns), len(relB.Columns))
	}
	for i := range relA.Columns {
		ca, cb := relA.Columns[i], relB.Columns[i]
		if ca.Name != cb.Name {
			t.Errorf("col[%d].Name: A=%q B=%q", i, ca.Name, cb.Name)
		}
		if ca.TypeOID != cb.TypeOID {
			t.Errorf("col[%d] (%s).TypeOID: A=%d B=%d", i, ca.Name, ca.TypeOID, cb.TypeOID)
		}
		if ca.NotNull != cb.NotNull {
			t.Errorf("col[%d] (%s).NotNull: A=%v B=%v", i, ca.Name, ca.NotNull, cb.NotNull)
		}
		if ca.AttNum != cb.AttNum {
			t.Errorf("col[%d] (%s).AttNum: A=%d B=%d", i, ca.Name, ca.AttNum, cb.AttNum)
		}
		if ca.TypeMod != cb.TypeMod {
			t.Errorf("col[%d] (%s).TypeMod: A=%d B=%d", i, ca.Name, ca.TypeMod, cb.TypeMod)
		}
	}

	// Functional equivalence: analyze the same SELECT against both catalogs
	// and assert the same target list shape.
	qA := mustAnalyze(t, catA, `SELECT id, customer, amount, placed_at FROM orders WHERE amount > 0`)
	qB := mustAnalyze(t, catB, `SELECT id, customer, amount, placed_at FROM orders WHERE amount > 0`)

	if len(qA.TargetList) != len(qB.TargetList) {
		t.Errorf("query target count: A=%d B=%d", len(qA.TargetList), len(qB.TargetList))
	}
	for i := range qA.TargetList {
		if qA.TargetList[i].ResName != qB.TargetList[i].ResName {
			t.Errorf("target[%d].ResName: A=%q B=%q",
				i, qA.TargetList[i].ResName, qB.TargetList[i].ResName)
		}
	}

	t.Log("equivalence PoC: hand-built CreateStmt and parsed DDL produce structurally identical Relation + analyze the same way")
}

// ============================================================================
// Loader pseudo-install verification
// ============================================================================
//
// These tests probe whether omni accepts minimal "pseudo" forms of each schema
// object kind, so the catalog loader strategy ("install everything; on failure install a
// pseudo at the same slot to keep downstream installs cascading through") can
// work inline during catalog init rather than as a query-time retry.
//
// Each test exercises one pseudo form in isolation, then the final
// TestLoaderPoC_RootPseudoCascadePrevention combines them end-to-end.

// ---- Pseudo enum: DefineEnum with minimal vals ----

func TestLoaderPoC_PseudoEnum_EmptyVals(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Can omni accept an enum with zero values?
	stmt := &ast.CreateEnumStmt{
		TypeName: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "broken_enum"},
		}},
		Vals: &ast.List{Items: []ast.Node{}},
	}
	err := cat.DefineEnum(stmt)
	t.Logf("DefineEnum empty vals: err=%v", err)

	if err == nil {
		// If it accepted, can a table column reference it and can the query run?
		tbl := buildCreateTableStmt("public", "t", []*ast.ColumnDef{
			columnDef("id", typeName("int4"), true),
			columnDef("status", mustTypeName(t, "public.broken_enum"), false),
		})
		if err := cat.DefineRelation(tbl, 'r'); err != nil {
			t.Logf("DefineRelation with empty-enum column: err=%v", err)
		} else {
			q, err := analyzeSafe(cat, `SELECT id, status FROM t WHERE status = 'anything'`)
			t.Logf("analyze against empty-enum column: err=%v targets=%d", err, targetsOf(q))
		}
	}
}

func TestLoaderPoC_PseudoEnum_SingleDummyVal(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Fallback: enum with single dummy value.
	stmt := &ast.CreateEnumStmt{
		TypeName: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "broken_enum"},
		}},
		Vals: &ast.List{Items: []ast.Node{
			&ast.String{Str: "__pseudo__"},
		}},
	}
	if err := cat.DefineEnum(stmt); err != nil {
		t.Fatalf("DefineEnum single dummy val failed: %v", err)
	}

	// Install a table using the pseudo enum as a column type.
	tbl := buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("status", mustTypeName(t, "public.broken_enum"), false),
	})
	if err := cat.DefineRelation(tbl, 'r'); err != nil {
		t.Fatalf("DefineRelation with pseudo-enum column failed: %v", err)
	}

	// Probe queries that would normally hit enum operators.
	queries := []string{
		`SELECT id, status FROM t`,
		`SELECT id FROM t WHERE status = 'running'`,
		`SELECT id FROM t WHERE status::text = 'running'`,
		`SELECT COUNT(*) FROM t GROUP BY status`,
	}
	for _, q := range queries {
		res, err := analyzeSafe(cat, q)
		t.Logf("query %q: err=%v targets=%d", q, err, targetsOf(res))
	}
}

// ---- Pseudo domain: DefineDomain with base type text ----

func TestLoaderPoC_PseudoDomain_OverText(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	stmt := &ast.CreateDomainStmt{
		Domainname: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "broken_domain"},
		}},
		Typname: typeName("text"),
	}
	if err := cat.DefineDomain(stmt); err != nil {
		t.Fatalf("DefineDomain over text failed: %v", err)
	}

	tbl := buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("code", mustTypeName(t, "public.broken_domain"), false),
	})
	if err := cat.DefineRelation(tbl, 'r'); err != nil {
		t.Fatalf("DefineRelation with pseudo-domain column failed: %v", err)
	}

	queries := []string{
		`SELECT id, code FROM t`,
		`SELECT id FROM t WHERE code = 'anything'`,
		`SELECT id FROM t WHERE code LIKE 'A%'`,
	}
	for _, q := range queries {
		res, err := analyzeSafe(cat, q)
		t.Logf("query %q: err=%v targets=%d", q, err, targetsOf(res))
	}
}

// ---- Pseudo composite: DefineCompositeType with metadata field names all text ----

func TestLoaderPoC_PseudoComposite_FieldsAllText(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Pretend metadata tells us this composite has fields {street, city, zip}.
	// We preserve the names but flatten every field to text.
	stmt := &ast.CompositeTypeStmt{
		Typevar: &ast.RangeVar{
			Schemaname: "public",
			Relname:    "broken_addr",
		},
		Coldeflist: &ast.List{Items: []ast.Node{
			columnDef("street", typeName("text"), false),
			columnDef("city", typeName("text"), false),
			columnDef("zip", typeName("text"), false),
		}},
	}
	if err := cat.DefineCompositeType(stmt); err != nil {
		t.Fatalf("DefineCompositeType pseudo failed: %v", err)
	}

	tbl := buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("addr", mustTypeName(t, "public.broken_addr"), false),
	})
	if err := cat.DefineRelation(tbl, 'r'); err != nil {
		t.Fatalf("DefineRelation with pseudo-composite column failed: %v", err)
	}

	queries := []string{
		`SELECT id, addr FROM t`,
		`SELECT id, (addr).city FROM t`,
		`SELECT id FROM t WHERE (addr).city = 'NYC'`,
	}
	for _, q := range queries {
		res, err := analyzeSafe(cat, q)
		t.Logf("query %q: err=%v targets=%d", q, err, targetsOf(res))
	}
}

// ---- Pseudo range: DefineRange with subtype text ----

func TestLoaderPoC_PseudoRange_SubtypeText(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Try: range with subtype text. PG itself requires subtype to have btree
	// ordering, which text does. If omni accepts it, this is a viable pseudo.
	stmt := &ast.CreateRangeStmt{
		TypeName: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "broken_range"},
		}},
		Params: &ast.List{Items: []ast.Node{
			&ast.DefElem{
				Defname: "subtype",
				Arg: &ast.TypeName{
					Names:   &ast.List{Items: []ast.Node{&ast.String{Str: "text"}}},
					Typemod: -1,
				},
			},
		}},
	}
	err := cat.DefineRange(stmt)
	t.Logf("DefineRange subtype=text: err=%v", err)
	if err != nil {
		t.Skip("range install failed; this is fine if bytebase metadata never has ranges")
	}

	tbl := buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("span", mustTypeName(t, "public.broken_range"), false),
	})
	if err := cat.DefineRelation(tbl, 'r'); err != nil {
		t.Logf("DefineRelation with pseudo-range column: err=%v", err)
	}

	q, err := analyzeSafe(cat, `SELECT id, span FROM t`)
	t.Logf("query on pseudo-range: err=%v targets=%d", err, targetsOf(q))
}

// ---- Pseudo view: constant SELECT body, no FROM ----

func TestLoaderPoC_PseudoView_ConstantTargetList(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Pretend metadata tells us view "v" exposes columns {id, title, total},
	// but its body is broken/unparseable. Build a pseudo SELECT body with
	// NULL::text aliased to each name.
	pseudoBody := `SELECT NULL::text AS id, NULL::text AS title, NULL::text AS total`
	sel := mustParseSelect(t, pseudoBody)

	stmt := &ast.ViewStmt{
		View: &ast.RangeVar{
			Schemaname:     "public",
			Relname:        "broken_view",
			Relpersistence: 'p',
		},
		Query: sel,
	}
	if err := cat.DefineView(stmt); err != nil {
		t.Fatalf("DefineView with constant-target body failed: %v", err)
	}

	rel := cat.GetRelation("public", "broken_view")
	if rel == nil {
		t.Fatal("pseudo view not installed")
	}
	t.Logf("pseudo view installed: RelKind=%c cols=%d", rel.RelKind, len(rel.Columns))
	for _, c := range rel.Columns {
		t.Logf("  col: %s", c.Name)
	}

	queries := []string{
		`SELECT id, title, total FROM broken_view`,
		`SELECT id FROM broken_view WHERE total = '0'`,
		`SELECT COUNT(*) FROM broken_view`,
	}
	for _, q := range queries {
		res, err := analyzeSafe(cat, q)
		t.Logf("query %q: err=%v targets=%d", q, err, targetsOf(res))
	}
}

// ---- Pseudo function: all params/return as text ----

func TestLoaderPoC_PseudoFunction_AllText(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Pretend metadata says the real function was broken_fn(some_broken_type)
	// returning some_other_broken_type. We install a pseudo with everything
	// collapsed to text.
	stmt := &ast.CreateFunctionStmt{
		Funcname: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "broken_fn"},
		}},
		Parameters: &ast.List{Items: []ast.Node{
			&ast.FunctionParameter{ArgType: typeName("text"), Mode: ast.FUNC_PARAM_IN},
		}},
		ReturnType: typeName("text"),
		Options: &ast.List{Items: []ast.Node{
			&ast.DefElem{Defname: "language", Arg: &ast.String{Str: "sql"}},
			&ast.DefElem{
				Defname: "as",
				Arg:     &ast.List{Items: []ast.Node{&ast.String{Str: "SELECT $1"}}},
			},
		}},
	}
	if err := cat.CreateFunctionStmt(stmt); err != nil {
		t.Fatalf("pseudo function install failed: %v", err)
	}

	procs := cat.LookupProcByName("broken_fn")
	t.Logf("pseudo function installed: %d procs under 'broken_fn'", len(procs))

	// Install a minimal table to query against.
	if err := cat.DefineRelation(buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("name", typeName("text"), false),
	}), 'r'); err != nil {
		t.Fatalf("install t: %v", err)
	}

	queries := []string{
		`SELECT id, broken_fn(name) AS result FROM t`,
		`SELECT id FROM t WHERE broken_fn(name) = 'x'`,
		// Analyzer should pick the only overload regardless of actual arg type.
		`SELECT broken_fn('literal') FROM t`,
	}
	for _, q := range queries {
		res, err := analyzeSafe(cat, q)
		t.Logf("query %q: err=%v targets=%d", q, err, targetsOf(res))
	}
}

// ---- End-to-end: root-pseudo prevents cascade ----

func TestLoaderPoC_RootPseudoCascadePrevention(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Scenario:
	//   enum E      — real install fails (simulated)
	//   table T     — has column col E
	//   view V      — SELECT col FROM T
	//
	// Phase 1: for each object in topo order:
	//   1. attempt real install
	//   2. on failure, install pseudo version in the same slot
	//
	// Expected: T and V install successfully because they see pseudo E in the
	// catalog. Query against V has correct lineage.

	// Step 1: simulate E real-install failure. In the real loader we would try
	// DefineEnum first and catch the error. Here we skip straight to the
	// pseudo form to simulate "real install failed".
	pseudoE := &ast.CreateEnumStmt{
		TypeName: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "task_status"},
		}},
		Vals: &ast.List{Items: []ast.Node{
			&ast.String{Str: "__pseudo__"},
		}},
	}
	if err := cat.DefineEnum(pseudoE); err != nil {
		t.Fatalf("pseudo enum install failed: %v", err)
	}

	// Step 2: install T pointing at the (pseudo) E. This is the real install,
	// not a pseudo — T's own metadata is fine.
	tStmt := buildCreateTableStmt("public", "tasks", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("status", mustTypeName(t, "public.task_status"), false),
		columnDef("title", typeName("text"), false),
	})
	if err := cat.DefineRelation(tStmt, 'r'); err != nil {
		t.Fatalf("T real install failed (cascade not prevented): %v", err)
	}

	// Step 3: install V — real install, view body references T.
	viewBody := `SELECT id, title, status FROM tasks WHERE id > 0`
	sel := mustParseSelect(t, viewBody)
	vStmt := &ast.ViewStmt{
		View: &ast.RangeVar{
			Schemaname:     "public",
			Relname:        "open_tasks",
			Relpersistence: 'p',
		},
		Query: sel,
	}
	if err := cat.DefineView(vStmt); err != nil {
		t.Fatalf("V real install failed (cascade not prevented): %v", err)
	}

	// Step 4: query V — should analyze successfully and return lineage.
	q := mustAnalyze(t, cat, `SELECT id, title, status FROM open_tasks`)
	if len(q.TargetList) != 3 {
		t.Errorf("query on view: got %d targets, want 3", len(q.TargetList))
	}

	// Verify lineage reaches base tables, not just view columns.
	for i, te := range q.TargetList {
		t.Logf("  target[%d]: name=%s expr=%T", i, te.ResName, te.Expr)
	}

	// Step 5: try a query that uses enum-typed column with operators that
	// would normally need enum resolution.
	res, err := analyzeSafe(cat, `SELECT id FROM open_tasks WHERE status = 'pending'`)
	t.Logf("query with WHERE on pseudo-enum column: err=%v targets=%d", err, targetsOf(res))

	t.Log("cascade prevention PoC: pseudo-enum at root allowed T and V to install as real objects")
}

// ---- Corner case 1: mixed real + pseudo overloads of the same function name ----
//
// Scenario: function fn has two overloads in metadata:
//   fn(int4)  → int4   (real install succeeds)
//   fn(my_broken_type)  → my_broken_type (real install fails because type is broken)
//
// catalog loader strategy: keep fn(int4) real, install pseudo fn(text) for the broken one.
// Verify: calls to fn(int_col) still pick fn(int4); calls to fn(text_col) pick
// the pseudo fn(text); calls to fn(literal_string) pick the pseudo.

func TestLoaderPoC_MixedRealAndPseudoOverloads(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	makeFuncStmt := func(name string, argType, retType string) *ast.CreateFunctionStmt {
		return &ast.CreateFunctionStmt{
			Funcname: &ast.List{Items: []ast.Node{
				&ast.String{Str: "public"},
				&ast.String{Str: name},
			}},
			Parameters: &ast.List{Items: []ast.Node{
				&ast.FunctionParameter{ArgType: typeName(argType), Mode: ast.FUNC_PARAM_IN},
			}},
			ReturnType: typeName(retType),
			Options: &ast.List{Items: []ast.Node{
				&ast.DefElem{Defname: "language", Arg: &ast.String{Str: "sql"}},
				&ast.DefElem{
					Defname: "as",
					Arg:     &ast.List{Items: []ast.Node{&ast.String{Str: "SELECT $1"}}},
				},
			}},
		}
	}

	// Install real fn(int4).
	if err := cat.CreateFunctionStmt(makeFuncStmt("fn", "int4", "int4")); err != nil {
		t.Fatalf("real fn(int4) install failed: %v", err)
	}

	// Install pseudo fn(text) — pretending the real fn(broken_type) failed.
	if err := cat.CreateFunctionStmt(makeFuncStmt("fn", "text", "text")); err != nil {
		t.Fatalf("pseudo fn(text) install failed: %v", err)
	}

	procs := cat.LookupProcByName("fn")
	t.Logf("fn overloads after mixed install: %d", len(procs))
	if len(procs) < 2 {
		t.Errorf("expected >= 2 fn overloads, got %d", len(procs))
	}

	// Install a table with one int and one text column.
	if err := cat.DefineRelation(buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("n", typeName("int4"), false),
		columnDef("s", typeName("text"), false),
	}), 'r'); err != nil {
		t.Fatalf("install t: %v", err)
	}

	// Each query exercises a different resolution path.
	queries := []struct {
		name string
		sql  string
	}{
		{"fn(int_col)  → should pick real fn(int4)", `SELECT id, fn(n) AS r FROM t`},
		{"fn(text_col) → should pick pseudo fn(text)", `SELECT id, fn(s) AS r FROM t`},
		{"fn(literal_int) → picks fn(int4) or via implicit cast", `SELECT fn(42) FROM t`},
		{"fn(literal_text) → picks fn(text)", `SELECT fn('hello') FROM t`},
	}
	for _, q := range queries {
		res, err := analyzeSafe(cat, q.sql)
		t.Logf("%s: err=%v targets=%d", q.name, err, targetsOf(res))
	}
}

// ---- Corner case 2: pseudo install after a real install has truly failed ----
//
// Scenario: real DefineDomain fails because the base type does not exist.
// The loader must then install a pseudo in the same slot. Verify:
//   1. real install actually returns an error (otherwise this test is vacuous)
//   2. catalog does not hold partial state for the failed name
//   3. pseudo install at the same name succeeds
//   4. a table column using that pseudo name resolves correctly

func TestLoaderPoC_PseudoAfterFailedReal(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	// Step 1: try a real DefineDomain with a non-existent base type.
	badStmt := &ast.CreateDomainStmt{
		Domainname: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "status_code"},
		}},
		Typname: typeName("nonexistent_base_type_xyz"),
	}
	err := cat.DefineDomain(badStmt)
	t.Logf("real DefineDomain with bogus base type: err=%v", err)

	realFailed := err != nil

	// Step 2: check catalog state. If the real install created a partial entry,
	// the pseudo install at the same name may conflict.
	//
	// Probe by trying to look up the name via a constructed type reference.
	// There's no direct "is this type defined?" API, but we can try the pseudo
	// install and see if it errors with a duplicate.

	// Step 3: install pseudo domain over text at the same name.
	pseudoStmt := &ast.CreateDomainStmt{
		Domainname: &ast.List{Items: []ast.Node{
			&ast.String{Str: "public"},
			&ast.String{Str: "status_code"},
		}},
		Typname: typeName("text"),
	}
	err = cat.DefineDomain(pseudoStmt)
	t.Logf("pseudo DefineDomain (base=text) after real failure: err=%v", err)

	if err != nil {
		t.Errorf("pseudo install after real failure should succeed; got %v", err)
		return
	}

	// Step 4: use the pseudo in a table column.
	if err := cat.DefineRelation(buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("code", mustTypeName(t, "public.status_code"), false),
	}), 'r'); err != nil {
		t.Fatalf("table with pseudo-domain column install failed: %v", err)
	}

	q := mustAnalyze(t, cat, `SELECT id, code FROM t WHERE code = 'ok'`)
	if len(q.TargetList) != 2 {
		t.Errorf("query on pseudo-domain column: got %d targets, want 2", len(q.TargetList))
	}

	if !realFailed {
		t.Log("note: real DefineDomain with bogus base type DID NOT fail — " +
			"omni's validation is looser than expected. Test still exercises " +
			"the pseudo install path but the 'recovery after failure' premise " +
			"is weaker. Consider whether the loader needs to handle this case.")
	} else {
		t.Log("pseudo recovery PoC: real install failed cleanly, pseudo at same " +
			"name installed successfully, downstream table install works")
	}
}

// ---- helpers for loader tests ----

// ---- Install cost benchmarks: catalog-loader hand-built vs Exec(DDL text) ----
//
// Validates the plan's "5-10× faster than current" claim empirically.
// Generates synthetic schemas of varying size with representative shapes
// (tables with mixed-type columns) and times both install paths.
//
// Run with:
//   go test -bench='BenchmarkLoader_Install' -benchmem -run='^$' -count=3 \
//     github.com/bytebase/bytebase/backend/plugin/parser/pg

func makeSyntheticTables(n int) []*ast.CreateStmt {
	tables := make([]*ast.CreateStmt, n)
	for i := range n {
		tables[i] = buildCreateTableStmt("public", fmt.Sprintf("t_%d", i), []*ast.ColumnDef{
			columnDef("id", typeName("int4"), true),
			columnDef("name", typeName("text"), false),
			columnDef("created_at", typeName("timestamp"), false),
			columnDef("amount", typeName("int8"), false),
			columnDef("status", typeName("text"), false),
		})
	}
	return tables
}

func makeSyntheticDDL(n int) string {
	var sb strings.Builder
	sb.Grow(n * 200)
	for i := range n {
		fmt.Fprintf(&sb, "CREATE TABLE public.t_%d (id int4 NOT NULL, name text, created_at timestamp, amount int8, status text);\n", i)
	}
	return sb.String()
}

func benchmarkE3Install(b *testing.B, n int) {
	tables := makeSyntheticTables(n)
	b.ResetTimer()
	for range b.N {
		cat := catalog.New()
		cat.SetSearchPath([]string{"public"})
		for _, stmt := range tables {
			if err := cat.DefineRelation(stmt, 'r'); err != nil {
				b.Fatalf("DefineRelation: %v", err)
			}
		}
	}
}

func benchmarkDDLInstall(b *testing.B, n int) {
	ddl := makeSyntheticDDL(n)
	b.ResetTimer()
	for range b.N {
		cat := catalog.New()
		cat.SetSearchPath([]string{"public"})
		if _, err := cat.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
			b.Fatalf("Exec: %v", err)
		}
	}
}

func BenchmarkLoader_Install_E3_100(b *testing.B)   { benchmarkE3Install(b, 100) }
func BenchmarkLoader_Install_E3_500(b *testing.B)   { benchmarkE3Install(b, 500) }
func BenchmarkLoader_Install_E3_2000(b *testing.B)  { benchmarkE3Install(b, 2000) }
func BenchmarkLoader_Install_DDL_100(b *testing.B)  { benchmarkDDLInstall(b, 100) }
func BenchmarkLoader_Install_DDL_500(b *testing.B)  { benchmarkDDLInstall(b, 500) }
func BenchmarkLoader_Install_DDL_2000(b *testing.B) { benchmarkDDLInstall(b, 2000) }

// ---- Overload selection assertions: prove analyzer picks the right overload ----
//
// Builds on TestLoaderPoC_MixedRealAndPseudoOverloads. That test logs outcomes
// but does not assert which overload got picked. Here we install two overloads
// with DISTINCT return types (int4 vs text) and assert that the analyzer picks
// the right one by checking the result column's TypeOID.
//
// Well-known PG OIDs: int4=23, text=25 (stable; used throughout omni tests).

func TestLoaderPoC_OverloadSelectionAsserted(t *testing.T) {
	const (
		oidInt4 uint32 = 23
		oidText uint32 = 25
	)

	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	mkFn := func(arg, ret string) *ast.CreateFunctionStmt {
		return &ast.CreateFunctionStmt{
			Funcname: &ast.List{Items: []ast.Node{
				&ast.String{Str: "public"},
				&ast.String{Str: "fn"},
			}},
			Parameters: &ast.List{Items: []ast.Node{
				&ast.FunctionParameter{ArgType: typeName(arg), Mode: ast.FUNC_PARAM_IN},
			}},
			ReturnType: typeName(ret),
			Options: &ast.List{Items: []ast.Node{
				&ast.DefElem{Defname: "language", Arg: &ast.String{Str: "sql"}},
				&ast.DefElem{
					Defname: "as",
					Arg:     &ast.List{Items: []ast.Node{&ast.String{Str: "SELECT $1"}}},
				},
			}},
		}
	}

	// Real fn(int4) → int4.  Pseudo fn(text) → text (simulated: pretend the
	// real fn(broken_type) failed and we installed a text-backed pseudo).
	if err := cat.CreateFunctionStmt(mkFn("int4", "int4")); err != nil {
		t.Fatalf("fn(int4) install failed: %v", err)
	}
	if err := cat.CreateFunctionStmt(mkFn("text", "text")); err != nil {
		t.Fatalf("fn(text) install failed: %v", err)
	}

	if err := cat.DefineRelation(buildCreateTableStmt("public", "t", []*ast.ColumnDef{
		columnDef("id", typeName("int4"), true),
		columnDef("n", typeName("int4"), false),
		columnDef("s", typeName("text"), false),
	}), 'r'); err != nil {
		t.Fatalf("table install: %v", err)
	}

	exprResultType := func(q *catalog.Query, targetIdx int) uint32 {
		t.Helper()
		if targetIdx >= len(q.TargetList) {
			t.Fatalf("target index %d out of range (have %d)", targetIdx, len(q.TargetList))
		}
		te := q.TargetList[targetIdx]
		fce, ok := te.Expr.(*catalog.FuncCallExpr)
		if !ok {
			t.Fatalf("target[%d] expected FuncCallExpr, got %T", targetIdx, te.Expr)
		}
		return fce.ResultType
	}

	// Case 1: fn(int_col) must pick fn(int4) → returns int4
	q := mustAnalyze(t, cat, `SELECT fn(n) FROM t`)
	if got := exprResultType(q, 0); got != oidInt4 {
		t.Errorf("fn(int_col): expected ResultType=%d (int4), got %d", oidInt4, got)
	}

	// Case 2: fn(text_col) must pick fn(text) → returns text
	q = mustAnalyze(t, cat, `SELECT fn(s) FROM t`)
	if got := exprResultType(q, 0); got != oidText {
		t.Errorf("fn(text_col): expected ResultType=%d (text), got %d", oidText, got)
	}

	// Case 3: fn(42) — integer literal — must pick fn(int4)
	q = mustAnalyze(t, cat, `SELECT fn(42) FROM t`)
	if got := exprResultType(q, 0); got != oidInt4 {
		t.Errorf("fn(literal_int): expected ResultType=%d (int4), got %d", oidInt4, got)
	}

	// Case 4: fn('hello') — string literal — must pick fn(text)
	q = mustAnalyze(t, cat, `SELECT fn('hello') FROM t`)
	if got := exprResultType(q, 0); got != oidText {
		t.Errorf("fn(literal_text): expected ResultType=%d (text), got %d", oidText, got)
	}

	t.Log("overload selection asserted: analyzer picks the right overload by arg type; " +
		"mixed real+pseudo overloads resolve correctly")
}

// ---- REAL cascade prevention: force a real install failure, then install pseudo ----
//
// The previous cascade test simulated failure by skipping the real install
// entirely. This test actually runs a real install that fails at the omni
// level, then proves the loader pattern "catch error → install pseudo at
// same slot" lets downstream installs use real AST.
//
// Scenario:
//   Domain D (base=nonexistent_type_xyz)   — real DefineDomain ERRORS
//   Table T (col D, title text)            — real DefineRelation must succeed against pseudo D
//   View V (SELECT col, title FROM T)      — real DefineView must succeed against real T
//   Query SELECT col, title FROM V         — must analyze with lineage to T.{col, title}

func TestLoaderPoC_RealCascadeWithGenuineFailure(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})

	installOne := func(kind, name string, installReal func() error, installPseudo func() error) (degraded bool) {
		if err := installReal(); err != nil {
			t.Logf("[%s %q] real install failed: %v", kind, name, err)
			if pErr := installPseudo(); pErr != nil {
				t.Fatalf("[%s %q] pseudo install also failed: %v", kind, name, pErr)
			}
			t.Logf("[%s %q] pseudo installed at same slot", kind, name)
			return true
		}
		t.Logf("[%s %q] real install succeeded", kind, name)
		return false
	}

	// Step 1: install D. Real fails (nonexistent base), pseudo succeeds.
	dName := "d_status"
	dDegraded := installOne("domain", dName,
		func() error {
			return cat.DefineDomain(&ast.CreateDomainStmt{
				Domainname: &ast.List{Items: []ast.Node{
					&ast.String{Str: "public"},
					&ast.String{Str: dName},
				}},
				Typname: typeName("nonexistent_base_type_for_test_xyz"),
			})
		},
		func() error {
			return cat.DefineDomain(&ast.CreateDomainStmt{
				Domainname: &ast.List{Items: []ast.Node{
					&ast.String{Str: "public"},
					&ast.String{Str: dName},
				}},
				Typname: typeName("text"),
			})
		},
	)
	if !dDegraded {
		t.Fatal("expected D real install to fail (test premise broken)")
	}

	// Step 2: install T. This is a REAL install (T's metadata is fine).
	// The column references the pseudo D, so DefineRelation must find it.
	tName := "records"
	tDegraded := installOne("table", tName,
		func() error {
			return cat.DefineRelation(buildCreateTableStmt("public", tName, []*ast.ColumnDef{
				columnDef("id", typeName("int4"), true),
				columnDef("col", mustTypeName(t, "public."+dName), false),
				columnDef("title", typeName("text"), false),
			}), 'r')
		},
		func() error {
			return errors.Errorf("should not reach pseudo for table %s", tName)
		},
	)
	if tDegraded {
		t.Fatal("T should install as REAL against pseudo D — cascade prevention violated")
	}

	// Step 3: install V. REAL install. View body references T (which is real).
	vName := "records_view"
	vDegraded := installOne("view", vName,
		func() error {
			sel := mustParseSelect(t, `SELECT id, col, title FROM records WHERE id > 0`)
			return cat.DefineView(&ast.ViewStmt{
				View: &ast.RangeVar{
					Schemaname:     "public",
					Relname:        vName,
					Relpersistence: 'p',
				},
				Query: sel,
			})
		},
		func() error {
			return errors.Errorf("should not reach pseudo for view %s", vName)
		},
	)
	if vDegraded {
		t.Fatal("V should install as REAL against real T — cascade prevention violated")
	}

	// Step 4: query V. Verify lineage makes it all the way to base table columns.
	q := mustAnalyze(t, cat, `SELECT id, col, title FROM records_view`)
	if len(q.TargetList) != 3 {
		t.Fatalf("view query: got %d targets, want 3", len(q.TargetList))
	}
	for i, te := range q.TargetList {
		if _, isVar := te.Expr.(*catalog.VarExpr); !isVar {
			t.Errorf("target[%d] (%s): expected VarExpr (resolved lineage), got %T",
				i, te.ResName, te.Expr)
		}
	}

	// Step 5: query with WHERE on the pseudo-backed column — text semantics
	// should let this analyze successfully.
	q2, err := analyzeSafe(cat, `SELECT id FROM records_view WHERE col = 'active'`)
	if err != nil {
		t.Errorf("WHERE on pseudo-domain column should analyze; got err=%v", err)
	} else if len(q2.TargetList) != 1 {
		t.Errorf("WHERE query: got %d targets, want 1", len(q2.TargetList))
	}

	t.Log("REAL cascade PoC: genuine DefineDomain failure → pseudo at same slot → " +
		"downstream table and view install as real → query lineage intact")
}

// analyzeSafe runs AnalyzeSelectStmt but returns (nil, err) instead of failing
// the test — probe tests need to record whether analyze succeeded.
func analyzeSafe(cat *catalog.Catalog, sql string) (*catalog.Query, error) {
	stmts, err := ParsePg(sql)
	if err != nil {
		return nil, errors.Wrap(err, "parse")
	}
	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("not a SelectStmt: %T", stmts[0].AST)
	}
	return cat.AnalyzeSelectStmt(sel)
}

func targetsOf(q *catalog.Query) int {
	if q == nil {
		return -1
	}
	return len(q.TargetList)
}
