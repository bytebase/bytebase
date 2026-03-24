# PostgreSQL QuerySpan Migration to Omni Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the ANTLR-based PostgreSQL QuerySpan extraction (~3,868 lines) with omni's semantic analysis infrastructure, reducing code by ~80% while maintaining full test compatibility with the existing 46 query_span + 31 query_type YAML test cases.

**Architecture:** The new implementation uses omni's `Catalog.AnalyzeSelectStmt()` to produce a semantically analyzed `Query` struct (with resolved column references, type information, and provenance tracking), then walks the analyzed tree to extract column lineage. Schema metadata is loaded into omni's catalog via `schema.GetDatabaseDefinition() → catalog.Exec(schemaDDL)`, reusing the pattern already established in `walk_through_omni.go`.

**Tech Stack:** Go, omni (`github.com/bytebase/omni/pg/catalog`), existing bytebase metadata infrastructure

**Scope exclusion:** PL/pgSQL function body analysis is tracked separately in [BYT-9082](https://linear.app/bytebase/issue/BYT-9082). This plan defers to that issue for function body parsing. During migration, function calls will fall back to the existing ANTLR-based function analysis until BYT-9082 is complete.

---

## Background

### Current state
- `query_span.go` — Entry point, calls ANTLR-based extractor
- `query_span_extractor.go` — 3,868 lines of ANTLR tree walking (~70 methods)
- `access_tables_antlr.go` — ANTLR listener for extracting accessed tables
- `query_type.go` — Already migrated to omni AST (no work needed)
- `access_tables.go` — Already has omni-based `ExtractAccessTables()` (no work needed)

### Target state
- `query_span.go` — Entry point calls omni-based extractor
- `query_span_omni.go` — New file: ~400-600 lines using omni's `AnalyzeSelectStmt()` + lineage walker
- `access_tables_antlr.go` — Deleted (replaced by existing `access_tables.go`)
- `query_span_extractor.go` — Deleted after migration complete

### Key files to read before starting
- `omni/pg/catalog/query.go` — Query, TargetEntry, VarExpr, RangeTableEntry types
- `omni/pg/catalog/analyze.go` — `AnalyzeSelectStmt()` (line 3580) and internal `analyzeSelectStmt()`
- `omni/pg/catalog/query_span_test.go` — Proof-of-concept lineage walker (our verification test)
- `bytebase/backend/plugin/schema/pg/walk_through_omni.go` — Pattern for loading metadata into omni catalog
- `bytebase/backend/plugin/schema/pg/get_database_definition.go` — Generates DDL from metadata proto
- `bytebase/backend/plugin/parser/pg/query_span_extractor.go` — What we're replacing
- `bytebase/backend/plugin/parser/pg/test-data/query_span.yaml` — 46 test cases (golden data)
- `bytebase/backend/plugin/parser/pg/test-data/query_type.yaml` — 31 test cases (already passing)

---

## Task 1: Create the omni-based QuerySpan extractor scaffold

**Files:**
- Create: `bytebase/backend/plugin/parser/pg/query_span_omni.go`
- Modify: `bytebase/backend/plugin/parser/pg/query_span.go`

This task creates the new extractor struct and wires up the entry point. The new extractor will use omni's catalog for analysis instead of ANTLR tree walking.

**Step 1: Write the new extractor struct and constructor**

Create `query_span_omni.go` with:

```go
package pg

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// omniQuerySpanExtractor extracts query span using omni's semantic analysis.
type omniQuerySpanExtractor struct {
	ctx             context.Context
	gCtx            base.GetQuerySpanContext
	defaultDatabase string
	searchPath      []string
	metaCache       map[string]*model.DatabaseMetadata
	cat             *catalog.Catalog
}

func newOmniQuerySpanExtractor(
	defaultDatabase string,
	searchPath []string,
	gCtx base.GetQuerySpanContext,
) *omniQuerySpanExtractor {
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}
	return &omniQuerySpanExtractor{
		defaultDatabase: defaultDatabase,
		searchPath:      searchPath,
		gCtx:            gCtx,
		metaCache:       make(map[string]*model.DatabaseMetadata),
	}
}
```

**Step 2: Wire up the entry point**

Modify `query_span.go` to call the new extractor. For now, keep both paths and add a feature flag or simply replace:

```go
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	if gCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.New("GetDatabaseMetadataFunc is not set")
	}
	_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database %q", database)
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = []string{schema}
	}
	extractor := newOmniQuerySpanExtractor(database, searchPath, gCtx)
	return extractor.getQuerySpan(ctx, stmt.Text)
}
```

**Step 3: Run query_type tests (should still pass since they use omni already)**

Run: `go test -v -count=1 -run ^TestGetQuerySpan$ github.com/bytebase/bytebase/backend/plugin/parser/pg`
Expected: Tests will fail since `getQuerySpan` is not yet implemented — that's OK.

**Step 4: Commit**

```bash
git add bytebase/backend/plugin/parser/pg/query_span_omni.go bytebase/backend/plugin/parser/pg/query_span.go
git commit -m "feat(pg): scaffold omni-based QuerySpan extractor"
```

---

## Task 2: Implement catalog loading from metadata

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

This task implements loading database schema metadata into omni's catalog, reusing the existing `GetDatabaseDefinition → catalog.Exec` pattern from `walk_through_omni.go`.

**Step 1: Add catalog initialization method**

```go
func (e *omniQuerySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := e.metaCache[database]; ok {
		return meta, nil
	}
	_, meta, err := e.gCtx.GetDatabaseMetadataFunc(e.ctx, e.gCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	e.metaCache[database] = meta
	return meta, nil
}

// initCatalog creates an omni catalog loaded with the database schema.
func (e *omniQuerySpanExtractor) initCatalog() error {
	meta, err := e.getDatabaseMetadata(e.defaultDatabase)
	if err != nil {
		return err
	}

	schemaDDL, err := schema.GetDatabaseDefinition(
		storepb.Engine_POSTGRES,
		schema.GetDefinitionContext{},
		meta.GetProto(),
	)
	if err != nil {
		return errors.Wrap(err, "failed to generate schema DDL")
	}

	e.cat = catalog.New()
	e.cat.SetSearchPath(e.searchPath)

	if schemaDDL != "" {
		if _, err := e.cat.Exec(schemaDDL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
			return errors.Wrap(err, "failed to load schema into catalog")
		}
	}
	return nil
}
```

**Step 2: Write a test that verifies catalog loading**

Add to a new test file or inline: create metadata proto, call `initCatalog()`, verify a table can be found via `cat.GetRelation("public", "t")`.

**Step 3: Run the test**

Run: `go test -v -count=1 -run ^TestOmniCatalogLoading$ github.com/bytebase/bytebase/backend/plugin/parser/pg`
Expected: PASS

**Step 4: Commit**

```bash
git add bytebase/backend/plugin/parser/pg/query_span_omni.go
git commit -m "feat(pg): implement catalog loading from metadata for QuerySpan"
```

---

## Task 3: Implement the core getQuerySpan pipeline

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

This is the main pipeline: parse → classify query type → analyze SELECT → extract lineage.

**Step 1: Implement getQuerySpan**

```go
func (e *omniQuerySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	e.ctx = ctx

	// Step 1: Parse with omni.
	omniStmts, err := ParsePg(stmt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}
	if len(omniStmts) != 1 {
		return nil, errors.Errorf("expected 1 statement, got %d", len(omniStmts))
	}

	// Step 2: Extract accessed tables using omni (already migrated).
	accessTables, err := ExtractAccessTables(stmt)
	if err != nil {
		return nil, err
	}
	accessesMap := make(base.SourceColumnSet)
	for _, resource := range accessTables {
		accessesMap[resource] = true
	}

	// Step 3: Check for mixed system/user tables.
	allSystems, mixed := isMixedQuery(accessesMap)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	// Step 4: Classify query type (already uses omni).
	queryType, isExplainAnalyze := classifyQueryType(omniStmts[0].AST, allSystems)

	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	if isExplainAnalyze {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Step 5: Initialize catalog and analyze SELECT.
	selStmt, ok := omniStmts[0].AST.(*ast.SelectStmt)
	if !ok {
		// Not a SELECT — should not happen after queryType check.
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	if err := e.initCatalog(); err != nil {
		return nil, errors.Wrap(err, "failed to init catalog")
	}

	query, err := e.cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		// Graceful degradation: return what we have.
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Step 6: Extract lineage from analyzed query.
	results := e.extractLineage(query)
	allSourceCols := e.extractAllSourceColumns(query)
	for col := range allSourceCols {
		accessesMap[col] = true
	}

	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessesMap,
		Results:       results,
	}, nil
}
```

**Step 2: Add stub methods for extractLineage and extractAllSourceColumns**

Return empty results for now — they will be implemented in the next tasks.

**Step 3: Run the query_type tests (31 tests)**

Run: `go test -v -count=1 -run ^TestGetQuerySpan$ github.com/bytebase/bytebase/backend/plugin/parser/pg`
Expected: query_type tests PASS (they don't check results), query_span tests FAIL (lineage not yet extracted)

**Step 4: Commit**

```bash
git add bytebase/backend/plugin/parser/pg/query_span_omni.go
git commit -m "feat(pg): implement core getQuerySpan pipeline with omni"
```

---

## Task 4: Implement the column lineage walker

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

Port the lineage walker from `omni/pg/catalog/query_span_test.go` into production code, adapted to produce `base.QuerySpanResult` instead of test-only types.

**Step 1: Implement extractLineage**

This walks `Query.TargetList` and for each `TargetEntry`, collects the source `ColumnResource` set by walking the expression tree. The key mapping is:

- `VarExpr` → resolve through `RangeTable[RangeIdx]` to get schema/table/column
- `RTERelation` → look up `Catalog.GetRelationByOID()` → `Relation.Schema.Name` + `Relation.Name`
- `RTESubquery` → recurse into `Subquery.TargetList[colIdx]`
- `RTECTE` → recurse into `CTEList[CTEIndex].Query.TargetList[colIdx]`
- `RTERelation` with `RelKind='v'` (view) → recurse into `Relation.AnalyzedQuery` (Gap 1 fix)

Key differences from the test walker:
- Output `base.QuerySpanResult` with `Name`, `SourceColumns` (as `base.SourceColumnSet`), `IsPlainField`
- Use `e.defaultDatabase` for the `Database` field in `ColumnResource`
- Handle set operations: merge lineage from `Query.LArg` and `Query.RArg`

**Step 2: Implement extractAllSourceColumns**

Walk the entire Query (TargetList + JoinTree.Quals + JoinExprNode.Quals + HavingQual) to collect all accessed columns. Return as `base.SourceColumnSet`.

**Step 3: Run a subset of query_span tests**

Run: `go test -v -count=1 -run ^TestGetQuerySpan$ github.com/bytebase/bytebase/backend/plugin/parser/pg`
Expected: Simple tests (column refs, star expansion, JOINs, subqueries, CTEs) should start passing.

**Step 4: Commit**

```bash
git add bytebase/backend/plugin/parser/pg/query_span_omni.go
git commit -m "feat(pg): implement column lineage walker for omni QuerySpan"
```

---

## Task 5: Handle set operations (UNION/INTERSECT/EXCEPT)

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

Set operations produce a top-level `Query` with `SetOp != SetOpNone` and `LArg`/`RArg` branches. The top-level `TargetList` has placeholder `VarExpr` entries without real provenance. We need to merge lineage from both branches.

**Step 1: Add set operation handling to extractLineage**

When `query.SetOp != catalog.SetOpNone`:
- Recursively get lineage from `query.LArg` and `query.RArg`
- For each output column position, merge source columns from both branches
- Column names come from the left branch (PG convention)
- For EXCEPT: only include left branch sources (right branch filters, doesn't contribute to output)

**Step 2: Run tests with UNION/INTERSECT/EXCEPT cases**

Expected: Set operation test cases pass.

**Step 3: Commit**

```bash
git commit -m "feat(pg): handle set operations in omni QuerySpan lineage"
```

---

## Task 6: Handle view through-lineage

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

When `resolveVar` encounters an `RTERelation` where the underlying `Relation.RelKind == 'v'` (view), it should recurse into `Relation.AnalyzedQuery` to trace lineage to the base tables.

**Step 1: Extend resolveVar for views**

In the `RTERelation` case of `resolveVar`:
```go
case catalog.RTERelation:
	rel := e.cat.GetRelationByOID(rte.RelOID)
	if rel == nil || rel.Schema == nil {
		return
	}
	// View through-lineage: recurse into view definition.
	if rel.RelKind == 'v' && rel.AnalyzedQuery != nil {
		if colIdx >= 0 && colIdx < len(rel.AnalyzedQuery.TargetList) {
			te := rel.AnalyzedQuery.TargetList[colIdx]
			e.walkExpr(rel.AnalyzedQuery, te.Expr, seen, result)
		}
		return
	}
	// Physical table: terminal case.
	// ... existing code ...
```

Also handle materialized views (`RelKind == 'm'`) the same way.

**Step 2: Run view-related test cases**

Expected: View lineage traces through to base tables.

**Step 3: Commit**

```bash
git commit -m "feat(pg): implement view through-lineage in omni QuerySpan"
```

---

## Task 7: Handle system functions as table sources

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

Omni's analyzer handles system functions in FROM clause via `RTEFunction`. However, the column names for system functions may differ from what QuerySpan expects. The key system functions are:

- `generate_series` → single column "generate_series"
- `generate_subscripts` → single column "generate_subscripts"
- `unnest` → N columns of "unnest" (one per array argument)
- `jsonb_each`/`json_each` → "key", "value"
- `jsonb_array_elements`/`json_array_elements` → "value"
- `json_to_record`/`jsonb_to_record` → columns from alias clause
- `json_to_recordset`/`jsonb_to_recordset` → columns from alias clause

**Step 1: Verify how omni handles these in RTEFunction**

Read `omni/pg/catalog/analyze.go` to check how `transformRangeFunction` creates RTEs for these functions. The column names should already be set correctly by omni's analysis.

**Step 2: If omni handles column names correctly, no code needed**

For `RTEFunction`, the lineage walker already returns no source columns (function results have no base-table provenance). The column names come from `rte.ColNames`. Verify by running the unnest/generate_series test cases.

**Step 3: If adjustments needed, add special handling**

Only add code if test cases fail — omni likely already handles the column names.

**Step 4: Run system function test cases**

Run: `go test -v -count=1 -run ^TestGetQuerySpan$ github.com/bytebase/bytebase/backend/plugin/parser/pg`
Expected: unnest, generate_series, json function tests pass.

**Step 5: Commit**

```bash
git commit -m "feat(pg): handle system function table sources in omni QuerySpan"
```

---

## Task 8: Handle user-defined functions (bridge to ANTLR for BYT-9082)

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

Until BYT-9082 (PL/pgSQL parser in omni) is complete, user-defined function calls need a bridge. When the lineage walker encounters a function call referencing a UDF, fall back to the existing ANTLR-based function analysis.

**Step 1: Detect UDF calls in the expression walker**

When `walkExpr` encounters a `FuncCallExpr`, check if it's a user-defined function by looking up the catalog's `UserProc` registry.

**Step 2: For UDF table sources (RTEFunction)**

Look up the function definition in metadata, then either:
- For SQL language functions: parse the body with `pg.Parse()`, analyze with `AnalyzeSelectStmt()`, extract lineage
- For PL/pgSQL functions: fall back to the existing `querySpanExtractor.findFunctionDefine()` logic

**Step 3: Run function-related test cases**

Expected: Function test cases pass (8 test cases).

**Step 4: Commit**

```bash
git commit -m "feat(pg): bridge UDF analysis to ANTLR for omni QuerySpan"
```

---

## Task 9: Handle the sourceColumns (accessed tables) collection

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

The existing QuerySpan collects `SourceColumns` as a set of `ColumnResource` with empty `Column` field (table-level access). This is used for data masking to know which tables are accessed.

**Step 1: Collect table-level access from RangeTable**

Walk `Query.RangeTable` to collect all `RTERelation` entries. For each, emit a `ColumnResource{Database: db, Schema: schema, Table: table, Column: ""}`.

**Step 2: Also collect from function body analysis**

If function analysis produces additional source columns, merge them.

**Step 3: Run tests checking sourcecolumns field**

Expected: The `sourcecolumns` field in YAML matches.

**Step 4: Commit**

```bash
git commit -m "feat(pg): collect accessed tables for omni QuerySpan sourceColumns"
```

---

## Task 10: Handle edge cases and error recovery

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/query_span_omni.go`

**Step 1: ResourceNotFoundError handling**

When omni's `AnalyzeSelectStmt` fails because a table/column doesn't exist, catch the error and return a partial `QuerySpan` with `NotFoundError` set, matching current behavior.

**Step 2: FunctionNotSupportedError handling**

When a function can't be analyzed, return partial results with the error flag set.

**Step 3: EXPLAIN statement handling**

`EXPLAIN SELECT ...` should extract the inner SELECT. `EXPLAIN ANALYZE` returns just the access map without results. Check that `classifyQueryType` already handles this correctly.

**Step 4: Run all 77 test cases**

Run: `go test -v -count=1 -run ^TestGetQuerySpan$ github.com/bytebase/bytebase/backend/plugin/parser/pg`
Expected: All 77 tests pass (46 query_span + 31 query_type).

**Step 5: Commit**

```bash
git commit -m "feat(pg): handle edge cases and error recovery in omni QuerySpan"
```

---

## Task 11: Clean up legacy ANTLR code

**Files:**
- Delete: `bytebase/backend/plugin/parser/pg/query_span_extractor.go` (3,868 lines)
- Delete: `bytebase/backend/plugin/parser/pg/access_tables_antlr.go` (96 lines)
- Modify: `bytebase/backend/plugin/parser/pg/query_span.go` (remove old extractor references)

**Step 1: Remove old extractor file**

Delete `query_span_extractor.go` entirely. If any helper functions are still needed by the UDF bridge (Task 8), extract them to a separate file first.

**Step 2: Remove ANTLR access table extractor**

Delete `access_tables_antlr.go` — replaced by `access_tables.go` which uses omni.

**Step 3: Check for remaining ANTLR imports in the package**

Run: `grep -r "antlr4-go/antlr" bytebase/backend/plugin/parser/pg/query_span*.go`
Expected: No matches (all ANTLR usage removed from query span files).

**Step 4: Run all tests one final time**

Run: `go test -v -count=1 -run ^TestGetQuerySpan$ github.com/bytebase/bytebase/backend/plugin/parser/pg`
Expected: All 77 tests pass.

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners bytebase/backend/plugin/parser/pg/...`
Expected: No issues.

**Step 6: Build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Builds successfully.

**Step 7: Commit**

```bash
git add -A bytebase/backend/plugin/parser/pg/
git commit -m "refactor(pg): remove legacy ANTLR QuerySpan extractor"
```

---

## Task Summary

| Task | Description | Lines changed (est.) | Risk |
|------|-------------|---------------------|------|
| 1 | Scaffold extractor + wire entry point | +80 | Low |
| 2 | Catalog loading from metadata | +50 | Low |
| 3 | Core getQuerySpan pipeline | +80 | Medium |
| 4 | Column lineage walker | +200 | High — core logic |
| 5 | Set operations | +40 | Medium |
| 6 | View through-lineage | +20 | Low |
| 7 | System functions | +20 (verify) | Low |
| 8 | UDF bridge to ANTLR | +100 | High — complex bridging |
| 9 | SourceColumns collection | +30 | Low |
| 10 | Edge cases + error recovery | +50 | Medium |
| 11 | Delete legacy code | -3,964 | Low — just deletion |

**Net result:** ~670 lines added, ~3,964 lines deleted = **~3,300 lines removed**

## Test Strategy

All 77 existing YAML test cases serve as the acceptance criteria. The test runner (`query_span_test.go`) is untouched — it calls `GetQuerySpan()` which is the entry point we're re-wiring. Tests should pass identically after migration.

Run the full test suite after each task. If a test fails, investigate before moving on — do not accumulate failures.
