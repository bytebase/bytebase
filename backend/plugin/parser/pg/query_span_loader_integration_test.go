package pg

import (
	"context"
	"os"
	"os/exec"
	"runtime/debug"
	"testing"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ---------------------------------------------------------------------------
// Catalog loader integration tests
// ---------------------------------------------------------------------------
//
// These tests drive the full GetQuerySpan pipeline (parse → initCatalog via
// catalogLoader → AnalyzeSelectStmt → extractLineage) against hand-built
// DatabaseSchemaMetadata fixtures that reproduce the bugs the loader is designed to
// prevent, plus the cascade-prevention story.
//
// Each test asserts one of:
//   - A query against an unrelated object succeeds even when a bad object is
//     present in the schema (BYT-9215 / BYT-9261 class, blast-radius).
//   - A query against a broken-but-pseudo'd object still returns lineage.
//   - A cascade chain (bad root → dependent table → dependent view) installs
//     downstream objects as real, with query lineage intact.

func TestLoaderIntegration_BYT9215_BadQuotedIdentifier(t *testing.T) {
	// BYT-9215 class: a table name contains a character sequence the DDL
	// deparse-and-reparse loop chokes on. Under the old path this would kill
	// query span for the entire database. Under the loader, the bad table is
	// pseudo-installed and unrelated queries succeed.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{
				// A table with a quoted identifier containing an apostrophe —
				// the kind of input BYT-9215 reported failing under Exec(ddl).
				// Real install should succeed (no DDL roundtrip), but even if
				// it didn't, pseudo would catch it.
				{
					Name: "'weird'table",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
					},
				},
				// An unrelated healthy table.
				{
					Name: "accounts",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "email", Type: "text"},
					},
				},
			},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, email FROM accounts`)
	if len(span.Results) != 2 {
		t.Fatalf("accounts query: got %d results, want 2", len(span.Results))
	}
	for _, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result %q has empty sources (lineage lost)", r.Name)
		}
	}
}

func TestLoaderIntegration_BrokenEnumCascade(t *testing.T) {
	// Real failure chain: table references a user-defined type that does
	// not exist in metadata. Real install of the table fails (omni cannot
	// resolve the type). Pseudo install of the table succeeds with text
	// columns. Query against the table returns its metadata column names.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "tasks",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					// References an enum that is NOT declared in metadata —
					// buildCreateStmt will succeed (typeNameFromString works)
					// but DefineRelation fails (enum not in catalog).
					{Name: "status", Type: "public.nonexistent_enum"},
					{Name: "title", Type: "text"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, status, title FROM tasks`)
	if len(span.Results) != 3 {
		t.Fatalf("tasks query: got %d results, want 3", len(span.Results))
	}
	wantNames := []string{"id", "status", "title"}
	for i, want := range wantNames {
		if i >= len(span.Results) {
			break
		}
		if span.Results[i].Name != want {
			t.Errorf("result[%d]: got %q, want %q", i, span.Results[i].Name, want)
		}
		if len(span.Results[i].SourceColumns) == 0 {
			t.Errorf("result[%d] (%s): empty sources", i, span.Results[i].Name)
		}
	}
}

func TestLoaderIntegration_BrokenRootTableAndHealthyNeighbor(t *testing.T) {
	// A query against a healthy table must succeed even when an unrelated
	// table in the same schema references a broken type. This is the core
	// blast-radius claim: one bad object does not poison all queries.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{
				{
					Name: "broken",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "bad", Type: "public.nonexistent_type"},
					},
				},
				{
					Name: "healthy",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "label", Type: "text"},
					},
				},
			},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, label FROM healthy`)
	if len(span.Results) != 2 {
		t.Fatalf("healthy query: got %d results, want 2", len(span.Results))
	}
	for _, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result %q lost lineage because of unrelated broken table", r.Name)
		}
	}
}

func TestLoaderIntegration_ViewOverBrokenTableStillResolves(t *testing.T) {
	// Chain: enum missing → table T references it (degrades to pseudo) →
	// view V on T installs real (against pseudo T) → query on V resolves.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "records",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "status", Type: "public.nonexistent_enum"},
					{Name: "title", Type: "text"},
				},
			}},
			Views: []*storepb.ViewMetadata{{
				Name:       "records_view",
				Definition: "SELECT id, title, status FROM records",
				DependencyColumns: []*storepb.DependencyColumn{
					{Schema: "public", Table: "records", Column: "id"},
					{Schema: "public", Table: "records", Column: "title"},
					{Schema: "public", Table: "records", Column: "status"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, title, status FROM records_view`)
	if len(span.Results) != 3 {
		t.Fatalf("view query: got %d results, want 3", len(span.Results))
	}
	for i, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result[%d] (%s) lost lineage", i, r.Name)
		}
	}
}

func TestLoaderIntegration_SimpleHealthyPath(t *testing.T) {
	// Baseline sanity: a clean schema must produce exact lineage down to
	// (schema, table, column) — no degraded flags.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "users",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "email", Type: "text"},
					{Name: "created_at", Type: "timestamp with time zone"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, email FROM users WHERE id > 0`)
	if len(span.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[0], "users", "id")
	mustHaveExactSource(t, span.Results[1], "users", "email")
}

func TestLoaderIntegration_EnumWorksWhenDeclared(t *testing.T) {
	// When an enum IS declared in metadata, the table installs as real and
	// enum-typed columns resolve through their real type. This is the
	// positive control for TestLoaderIntegration_BrokenEnumCascade.
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			EnumTypes: []*storepb.EnumTypeMetadata{{
				Name:   "task_status",
				Values: []string{"pending", "running", "done"},
			}},
			Tables: []*storepb.TableMetadata{{
				Name: "tasks",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "status", Type: "public.task_status"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, status FROM tasks WHERE status = 'pending'`)
	if len(span.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[0], "tasks", "id")
	mustHaveExactSource(t, span.Results[1], "tasks", "status")
}

func TestLoaderIntegration_CorrelatedRangeFunctionSubquery(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{
				{
					Name: "compliance_case_record_audits",
					Columns: []*storepb.ColumnMetadata{
						{Name: "case_id", Type: "text"},
						{Name: "entity_id", Type: "text"},
						{Name: "reason", Type: "text"},
						{Name: "remark", Type: "text"},
						{Name: "case_record", Type: "jsonb"},
						{Name: "reviewer_by", Type: "text"},
						{Name: "deleted_at", Type: "timestamptz"},
					},
				},
				{
					Name: "compliance_cases",
					Columns: []*storepb.ColumnMetadata{
						{Name: "case_id", Type: "text"},
						{Name: "case_info", Type: "jsonb"},
						{Name: "deleted_at", Type: "timestamptz"},
					},
				},
			},
		}},
	}
	sql := `
select
  a.case_id,
  -- c.member_id,
  a.entity_id as uid,
  a.reason,
  a.remark,
  (
    select elem->>'matchedDateTimeValue'
    from jsonb_array_elements(a.case_record::jsonb->'secondaryFieldResults') elem
    where elem->>'typeId' = '******'
    limit 1
  ) as wc_dob,
  (
    select elem->>'matchedValue'
    from jsonb_array_elements(a.case_record::jsonb->'secondaryFieldResults') elem
    where elem->>'typeId' = '******'
    limit 1
  ) as wc_citizenship,
  c.case_info->>'birth_date' as kyc_dob,
  c.case_info->>'nationality' as kyc_citizenship,
  a.reviewer_by as review_by
from compliance_case_record_audits a
inner join compliance_cases c
  on c.case_id = a.case_id and c.deleted_at is null
where a.reviewer_by in ('****** ', '******')
  and a.deleted_at is null;
`
	span := mustGetQuerySpan(t, meta, sql)
	if len(span.Results) != 9 {
		t.Fatalf("got %d results, want 9", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[4], "compliance_case_record_audits", "case_record")
	mustHaveExactSource(t, span.Results[5], "compliance_case_record_audits", "case_record")
	mustHaveExactSource(t, span.Results[6], "compliance_cases", "case_info")
	mustHaveExactSource(t, span.Results[7], "compliance_cases", "case_info")
}

func TestAppendQueryDoesNotMutateSharedStack(t *testing.T) {
	outer := &catalog.Query{}
	parent := &catalog.Query{}
	current := &catalog.Query{}
	nested := &catalog.Query{}
	queryStack := []*catalog.Query{outer, parent, current}

	nextStack := appendQuery(queryStack[:2], nested)

	if queryStack[2] != current {
		t.Fatalf("appendQuery mutated caller stack: got %p, want %p", queryStack[2], current)
	}
	if len(nextStack) != 3 || nextStack[2] != nested {
		t.Fatalf("appendQuery returned unexpected stack: %+v", nextStack)
	}
}

func TestLoaderIntegration_BuiltinFunctionSelfReferenceDoesNotOverflow(t *testing.T) {
	if os.Getenv("BYTEBASE_TEST_FUNCTION_SELF_REFERENCE") == "1" {
		debug.SetMaxStack(1 << 20)
		varExpr := &catalog.VarExpr{RangeIdx: 0, AttNum: 1}
		q := &catalog.Query{
			TargetList: []*catalog.TargetEntry{{
				Expr: varExpr,
			}},
			RangeTable: []*catalog.RangeTableEntry{{
				Kind: catalog.RTEFunction,
				FuncExprs: []catalog.AnalyzedExpr{&catalog.FuncCallExpr{
					Args: []catalog.AnalyzedExpr{&catalog.OpExpr{
						Left: varExpr,
						Right: &catalog.ConstExpr{
							Value: "items",
						},
					}},
				}},
			}},
		}
		extractor := newOmniQuerySpanExtractor(loaderTestDB, []string{loaderTestSchema}, base.GetQuerySpanContext{})
		extractor.cat = catalog.New()
		extractor.walkExpr(q, q.TargetList[0].Expr, make(base.SourceColumnSet))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoaderIntegration_BuiltinFunctionSelfReferenceDoesNotOverflow$")
	cmd.Env = append(os.Environ(), "BYTEBASE_TEST_FUNCTION_SELF_REFERENCE=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("walkExpr overflowed on self-referential function RTE: %v\n%s", err, output)
	}
}

func TestLoaderIntegration_PlainColumnCycleDoesNotOverflow(t *testing.T) {
	if os.Getenv("BYTEBASE_TEST_PLAIN_COLUMN_CYCLE") == "1" {
		debug.SetMaxStack(1 << 20)
		varExpr := &catalog.VarExpr{RangeIdx: 0, AttNum: 1}
		q := &catalog.Query{
			TargetList: []*catalog.TargetEntry{{
				Expr: varExpr,
			}},
			RangeTable: []*catalog.RangeTableEntry{{
				Kind: catalog.RTESubquery,
			}},
		}
		q.RangeTable[0].Subquery = q
		_ = isUltimatelyPlainColumn(q, varExpr)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoaderIntegration_PlainColumnCycleDoesNotOverflow$")
	cmd.Env = append(os.Environ(), "BYTEBASE_TEST_PLAIN_COLUMN_CYCLE=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("isUltimatelyPlainColumn overflowed on cyclic subquery RTE: %v\n%s", err, output)
	}
}

func TestLoaderIntegration_PredicateQueryCycleDoesNotOverflow(t *testing.T) {
	if os.Getenv("BYTEBASE_TEST_PREDICATE_QUERY_CYCLE") == "1" {
		debug.SetMaxStack(1 << 20)
		q := &catalog.Query{}
		q.CTEList = []*catalog.CommonTableExprQ{{Query: q}}
		extractor := newOmniQuerySpanExtractor(loaderTestDB, []string{loaderTestSchema}, base.GetQuerySpanContext{})
		analyzer := &plpgsqlAnalyzer{
			extractor: extractor,
			scope:     newVariableScope(nil),
		}
		analyzer.collectQueryPredicateColumns(q)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoaderIntegration_PredicateQueryCycleDoesNotOverflow$")
	cmd.Env = append(os.Environ(), "BYTEBASE_TEST_PREDICATE_QUERY_CYCLE=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("collectQueryPredicateColumns overflowed on cyclic query graph: %v\n%s", err, output)
	}
}

func TestLoaderIntegration_SetOpQueryCycleDoesNotOverflow(t *testing.T) {
	if os.Getenv("BYTEBASE_TEST_SET_OP_QUERY_CYCLE") == "1" {
		debug.SetMaxStack(1 << 20)
		q := &catalog.Query{SetOp: catalog.SetOpUnion}
		q.LArg = q
		q.RArg = &catalog.Query{TargetList: []*catalog.TargetEntry{{
			Expr: &catalog.ConstExpr{
				Value: "1",
			},
		}}}
		selStmt := &ast.SelectStmt{}
		selStmt.Larg = selStmt
		selStmt.Rarg = &ast.SelectStmt{}
		extractor := newOmniQuerySpanExtractor(loaderTestDB, []string{loaderTestSchema}, base.GetQuerySpanContext{})
		extractor.extractLineage(q, selStmt)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoaderIntegration_SetOpQueryCycleDoesNotOverflow$")
	cmd.Env = append(os.Environ(), "BYTEBASE_TEST_SET_OP_QUERY_CYCLE=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("extractLineage overflowed on cyclic set-op query graph: %v\n%s", err, output)
	}
}

func TestLoaderIntegration_FallbackCTECycleDoesNotOverflow(t *testing.T) {
	if os.Getenv("BYTEBASE_TEST_FALLBACK_CTE_CYCLE") == "1" {
		debug.SetMaxStack(1 << 20)
		cteSel := &ast.SelectStmt{
			TargetList: &ast.List{Items: []ast.Node{&ast.ResTarget{
				Name: "x",
				Val: &ast.ColumnRef{Fields: &ast.List{Items: []ast.Node{
					&ast.String{Str: "c"},
					&ast.String{Str: "x"},
				}}},
			}}},
			FromClause: &ast.List{Items: []ast.Node{&ast.RangeVar{Relname: "c"}}},
		}
		extractor := newOmniQuerySpanExtractor(loaderTestDB, []string{loaderTestSchema}, base.GetQuerySpanContext{})
		extractor.cat = catalog.New()
		analyzer := &plpgsqlAnalyzer{
			extractor: extractor,
			scope:     newVariableScope(nil),
			cteMap:    map[string]*ast.SelectStmt{"c": cteSel},
		}
		analyzer.resolveThroughCTE(cteSel, "x", nil, make(base.SourceColumnSet))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoaderIntegration_FallbackCTECycleDoesNotOverflow$")
	cmd.Env = append(os.Environ(), "BYTEBASE_TEST_FALLBACK_CTE_CYCLE=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("resolveThroughCTE overflowed on cyclic fallback CTE: %v\n%s", err, output)
	}
}

// ---------- helpers ----------

// loaderTestDB / loaderTestSchema are the fixed database/schema names
// every catalog-loader integration test uses; keeping them as constants keeps the call
// sites terse and satisfies the unparam linter.
const (
	loaderTestDB     = "db"
	loaderTestSchema = "public"
)

func mustGetQuerySpan(t *testing.T, meta *storepb.DatabaseSchemaMetadata, sql string) *base.QuerySpan {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{meta})
	span, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: sql}, loaderTestDB, "", false)
	if err != nil {
		t.Fatalf("GetQuerySpan(%q): %v", sql, err)
	}
	if span == nil {
		t.Fatalf("GetQuerySpan(%q): nil span", sql)
	}
	return span
}

func mustHaveExactSource(t *testing.T, result base.QuerySpanResult, table, column string) {
	t.Helper()
	want := base.ColumnResource{
		Database: loaderTestDB,
		Schema:   loaderTestSchema,
		Table:    table,
		Column:   column,
	}
	for src := range result.SourceColumns {
		if src == want {
			return
		}
	}
	t.Errorf("result %q missing source %+v; have %+v", result.Name, want, result.SourceColumns)
}
