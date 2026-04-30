# MySQL Query Span Omni Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate MySQL query-span extraction from ANTLR parse-tree walking to omni MySQL AST without runtime fallback.

**Architecture:** Follow the MSSQL migration model: build a package-internal omni extractor that mirrors the existing extractor's behavior, prove it with probe tests, focused behavior tests, and a full-corpus golden harness, then cut over once fixture output is unchanged. Do not use the PostgreSQL catalog-analyzer model for this migration; MySQL already has a hand-written extractor whose behavior can be translated directly onto omni AST.

**Tech Stack:** Go, `github.com/bytebase/omni/mysql/ast`, `backend/plugin/parser/base` query-span model, existing YAML fixtures under `backend/plugin/parser/mysql/test-data/query-span/`.

## Current Status

- Probe coverage exists in `backend/plugin/parser/mysql/query_span_omni_probe_test.go`.
- Omni parses 30/30 current MySQL query-span fixture statements.
- Probe assertions cover aliases, stars, joins, derived tables, CTEs, set operations, correlated subqueries, `JSON_TABLE`, and query-type root nodes.
- `backend/plugin/parser/mysql/query_span_extractor_omni.go` is the production MySQL query-span extractor.
- Strict golden coverage is 30/30 matched and 0 diffs across the current MySQL query-span fixture corpus.
- Post-review systematic regression tests cover derived-table column alias lists, `IN (SELECT ...)` result lineage, explicit long-tail expression nodes, legacy DML roots (`CALL`, `DO`, `HANDLER`), and `TABLE` / `VALUES` select-family roots.
- The omni expression extractor now fails on unsupported expression node types instead of silently returning empty lineage.
- Public `GetQuerySpan` is omni-backed through `querySpanExtractor.getQuerySpan`.
- `query_span*.go` production code no longer depends on `ParseMySQL`, `GetANTLRAST`, `antlr4-go`, or `github.com/bytebase/parser/mysql`.

## Non-Negotiables

- No runtime feature flag.
- No environment switch.
- No fallback from omni to ANTLR after cutover.
- No partial production rollout.
- The ANTLR path may exist only as a development reference before cutover.
- The cutover happens only after the omni path has zero parity diffs across the current fixture corpus.
- Existing YAML fixture expectations remain unchanged at cutover.

## Final Shape

The final MySQL query-span implementation should look like this:

```text
GetQuerySpan
  -> newQuerySpanExtractor(...)
  -> q.getQuerySpan(ctx, stmt.Text)
      -> ParseMySQLOmni(stmt)
      -> collectOmniAccessTables(root)
      -> isMixedQuery(...)
      -> classifyOmniQueryType(root, allSystems)
      -> if non-SELECT: return type + access tables
      -> extractFromSelectRoot(*ast.SelectStmt | *ast.TableStmt | *ast.ValuesStmt)
          -> processCTEs
          -> extractFromSetOp
          -> extractFromClause
          -> collect predicate/source tables as legacy behavior requires
          -> extractTargetList
      -> return QuerySpan
```

Final file ownership:

- `backend/plugin/parser/mysql/query_span.go`
  - Stays as the public registration and `GetQuerySpan` wrapper.
  - Calls the sole omni-backed `querySpanExtractor.getQuerySpan`.

- `backend/plugin/parser/mysql/query_span_extractor.go`
  - Final home for shared extractor state and string/metadata helpers.
  - Keeps helpers that do not depend on ANTLR contexts:
    - `getAllTableColumnSources`
    - `getFieldColumnSource`
    - `filterClusterName`
    - `findTableSchema`
    - `getColumnsForView`
    - `isMixedQuery`
    - `isSystemResource`
  - Deletes ANTLR listener and ANTLR context methods after cutover.

- `backend/plugin/parser/mysql/query_span_extractor_omni.go`
  - Final home for all omni AST extraction logic.
  - May be split if it grows too large:
    - `query_span_omni_expr.go`
    - `query_span_omni_from.go`
    - `query_span_omni_access.go`
  - Starts package-internal while ANTLR remains reference.
  - Becomes the production path at cutover.

- `backend/plugin/parser/mysql/query_type.go`
  - Current ANTLR `queryTypeListener` is deleted or replaced by `classifyOmniQueryType`.

- `backend/plugin/parser/mysql/omni.go`
  - `AsANTLRAST()` fallback is removed only after no MySQL caller needs it.
  - This migration removes query-span's dependency; other MySQL modules must be checked separately before deleting fallback globally.

- Tests:
  - `query_span_omni_probe_test.go` stays through cutover, then can remain as a parser regression test.
  - `query_span_extractor_omni_test.go` stays as focused unit coverage.
  - `query_span_omni_parity_test.go` remains as a golden harness after cutover. It must compare the internal omni extractor directly against YAML expectations, not `GetQuerySpan` against the internal omni path.

## Coverage Matrix

The current fixture corpus is the minimum gate. Add focused tests for each bucket before implementing that bucket.

| Bucket | Required behavior |
|---|---|
| Query type | `SELECT`, `TABLE`, `VALUES`, `EXPLAIN`, `EXPLAIN ANALYZE SELECT`, `SHOW`, `SET`, DDL, DML, all-system SELECT, legacy DML roots (`CALL`, `DO`, `HANDLER`) |
| Access tables | top-level FROM, joined tables, derived-table body tables, CTE body tables, subqueries in SELECT/WHERE, system-table suppression |
| Target list | constants, bare columns, qualified columns, aliases, expression names, `*`, `table.*` |
| Expressions | column refs, literals, binary/unary ops, functions, aggregates, `CASE`, `CAST`, `BETWEEN`, `IN` value list, `IN` subquery, `LIKE`, `IS`, `EXISTS`, scalar subquery, `CONVERT`, `COLLATE`, `MATCH`, `ROW`, `MEMBER OF`, `INTERVAL`, window args |
| FROM | `TableRef`, aliases, comma joins, `JoinClause`, `ON`, `USING`, nested joins |
| Derived tables | subquery table source, derived alias, derived column alias list, nested derived tables |
| Set operations | `UNION`, `UNION ALL`, recursive CTE union, positional source-column merge |
| CTE | non-recursive CTE, nested CTE, explicit CTE column list, recursive CTE anchor/recursive merge |
| JSON_TABLE | JSON table columns, `priorTableInFrom`, source lineage from JSON expression owner |
| Views | `getColumnsForView` recurses through omni extractor and applies view output columns |
| Case sensitivity | existing `ignoreCaseSensitive` behavior preserved |
| Not found | missing database/table/column errors map to `QuerySpan.NotFoundError` as today |
| StarRocks | current two StarRocks fixtures keep matching |

## ANTLR To Omni Mapping

| Current ANTLR item | Final omni item |
|---|---|
| `getQuerySpan` | `getOmniQuerySpan` renamed back to `getQuerySpan` at cutover |
| `queryTypeListener` | `classifyOmniQueryType(ast.Node, allSystems)` |
| `selectOnlyListener` | direct root dispatch in `getOmniQuerySpan` |
| `extractContext` | direct `extractFromSelectStmt` / statement type switch |
| `extractSelectStatement` | `extractFromSelectStmt(*ast.SelectStmt)` |
| `extractQueryExpression` | `extractFromSelectStmt` including CTE + set-op handling |
| `extractQueryExpressionParens` | unnecessary; omni AST is already normalized |
| `extractQueryExpressionBody` | `extractFromSetOp` or simple select body handling |
| `extractQueryPrimary` | `extractFromSelectStmt`, `extractTableStmt`, `extractValuesStmt` as needed |
| `extractExplicitTable` | `resolveTableRef` / `extractTableStmt` |
| `extractTableValueConstructor` | `extractValuesStmt` if fixtures require it |
| `extractQuerySpecification` | `extractFromSelectStmt` simple-select branch |
| `extractSelectItemList` | `extractTargetList([]ast.ExprNode, fromSources)` |
| `extractSelectItem` | `extractTarget` |
| `extractSourceColumnSetFromExpr` | `resolveExpression(ast.ExprNode)` |
| `extractSourceColumnSetFromExprList` | `mergeExpressionSources(...ast.ExprNode)` |
| `extractTableWild` | `expandStar(*ast.ColumnRef)` |
| `extractTableSourcesFromFromClause` | `extractFromClause([]ast.TableExpr)` |
| `extractTableReferenceList` | `extractFromClause` |
| `extractTableReference` | `extractTableSource(ast.TableExpr)` |
| `extractJoinedTable` | `extractJoin(*ast.JoinClause)` |
| `extractTableFactor` | `extractTableSource` type switch |
| `extractTableFunction` | `extractJsonTable(*ast.JsonTableExpr)` |
| `extractTableReferenceListParens` | unnecessary unless omni exposes nested table lists |
| `extractSubquery` | `extractSubqueryAsPseudo(*ast.SelectStmt)` |
| `extractDerivedTable` | `extractDerivedTable(*ast.SubqueryExpr)` |
| `extractSingleTable` | `resolveTableRef(*ast.TableRef)` |
| `extractSingleTableParens` | unnecessary |
| `extractCommonTableExpression` | `processCTEs([]*ast.CommonTableExpr)` |
| `extractRecursiveCTE` | `extractRecursiveCTE(*ast.CommonTableExpr)` |
| `extractNonRecursiveCTE` | `extractNonRecursiveCTE(*ast.CommonTableExpr)` |
| `recursiveCTEExtractListener` | direct recursive CTE anchor/recursive branch extraction |
| `getAccessTables` / `accessTableListener` | `collectOmniAccessTables(ast.Node, defaultDatabase)` |
| `extractTableRefs` / `resourceExtractListener` | explicit table-source traversal for access tables |

## Target API

Final extractor methods:

```go
func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error)
func (q *querySpanExtractor) extractFromSelectStmt(sel *ast.SelectStmt) (*base.PseudoTable, error)
func (q *querySpanExtractor) extractFromSetOp(sel *ast.SelectStmt) (*base.PseudoTable, error)
func (q *querySpanExtractor) processCTEs(ctes []*ast.CommonTableExpr) error
func (q *querySpanExtractor) extractTableSource(expr ast.TableExpr) ([]base.TableSource, error)
func (q *querySpanExtractor) resolveTableRef(ref *ast.TableRef) (base.TableSource, error)
func (q *querySpanExtractor) resolveExpression(expr ast.ExprNode) (base.QuerySpanResult, error)
func (q *querySpanExtractor) cloneForSubquery() *querySpanExtractor
func collectOmniAccessTables(root ast.Node, defaultDatabase string) base.SourceColumnSet
```

During migration, these may live on `*omniQuerySpanExtractor`. At cutover, merge or rename so there is only one production extractor type.

## Execution Plan

Every implementation task follows TDD:

1. Write the focused failing test.
2. Run the focused test and confirm it fails for the expected missing behavior.
3. Implement the smallest behavior.
4. Run the focused test.
5. Run golden harness and record matched/diff count.
6. Run existing `TestGetQuerySpan` to protect the reference path before cutover.

### Phase 0: Probe And Scaffold

Status: done.

Files:

- `backend/plugin/parser/mysql/query_span_omni_probe_test.go`
- `backend/plugin/parser/mysql/query_span_extractor_omni.go`
- `backend/plugin/parser/mysql/query_span_extractor_omni_test.go`
- `backend/plugin/parser/mysql/query_span_omni_parity_test.go`

Proof:

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/mysql -run '^(TestMySQLOmniQuerySpanMigrationProbe|TestOmniQuerySpanScaffold_QueryTypesAndAccessTables|TestMySQLOmniQuerySpanGoldenHarness|TestGetQuerySpan)$'
```

### Phase 1: Simple SELECT Result Columns

Status: done.

Goal: Make basic target-list fixtures match.

Tests first:

- `SELECT 1`
- `SELECT a FROM t`
- `SELECT a AS x FROM t`
- `SELECT a, t.b, db.t.c FROM t`
- `SELECT * FROM t`
- `SELECT *, a FROM t`

Implementation:

- Add `extractFromSelectStmt` simple branch.
- Add `extractTargetList`.
- Add `resolveExpression` for:
  - `*ast.ColumnRef`
  - literals
  - `*ast.ResTarget`
- Add `expandStar`.

Expected parity movement:

- Several `standard.yaml` simple SELECT cases should move from diff to match.

### Phase 2: Expression Source Merging

Status: done.

Goal: Match expression lineage and names for non-subquery expressions.

Tests first:

- arithmetic: `a-b AS c1`
- comparison: `a=b AS c2`
- function: `MAX(a)`
- nested function args
- `CASE`
- `CAST`
- `BETWEEN`
- `IN` value list
- `LIKE`
- `IS NULL`
- window function args if omni fixtures or added tests require them

Implementation:

- Extend `resolveExpression`.
- Add `mergeExpressionSources`.
- Preserve `IsPlainField` semantics:
  - plain column ref: true
  - literal constant: true, matching current MySQL behavior for `SELECT 1`
  - expression over columns/functions/subqueries: false

### Phase 3: FROM, JOIN, Alias, And Scope

Status: done.

Goal: Resolve table aliases and joined table sources.

Tests first:

- `FROM t AS x`
- `x.a`
- `JOIN ... ON`
- `JOIN ... USING(a)`
- comma join
- nested join
- alias shadowing between table aliases and physical names

Implementation:

- Add `extractFromClause`.
- Add `extractTableSource`.
- Add `extractJoin`.
- Add alias-aware table source wrappers if current `base.TableSource` APIs require pseudo renaming.
- Ensure `tableSourceFrom` ordering matches current ANTLR behavior.

### Phase 4: Derived Tables And Subqueries

Status: done.

Goal: Match derived table and scalar/correlated subquery behavior.

Tests first:

- `(SELECT a,b FROM t) AS x`
- `(SELECT a,b FROM t) AS x(c1,c2)`
- scalar subquery constant
- scalar subquery from table
- correlated scalar subquery
- `WHERE a IN (SELECT a FROM t)`
- `EXISTS (SELECT 1 FROM t WHERE ...)`

Implementation:

- Add `cloneForSubquery`.
- Add `extractDerivedTable`.
- Add `extractSubqueryAsPseudo`.
- Merge subquery source columns into the correct result or access-table set according to current parity.
- Fix `collectOmniAccessTables` to include subqueries in target lists and predicates, not only FROM.

### Phase 5: Set Operations

Status: done.

Goal: Match positional lineage for `UNION` and recursive CTE set-op bodies.

Tests first:

- `SELECT a FROM t UNION SELECT b FROM t2`
- `UNION ALL`
- column count mismatch behavior if current code errors
- set-op inside derived table

Implementation:

- Add `extractFromSetOp`.
- Merge result source columns positionally.
- Preserve anchor-side result names.

### Phase 6: CTEs

Status: done.

Goal: Match all existing CTE fixtures, including recursive cases.

Tests first:

- simple CTE
- nested CTE
- explicit CTE column aliases
- recursive CTE with explicit aliases
- recursive CTE without aliases

Implementation:

- Add `processCTEs`.
- Add `extractNonRecursiveCTE`.
- Add `extractRecursiveCTE`.
- Preserve current shadowing behavior: nearest CTE wins over physical table when database is unspecified.

### Phase 7: JSON_TABLE

Status: done.

Goal: Match JSON table fixture behavior.

Tests first:

- current `JSON_TABLE` fixture.
- JSON table columns derive source from owner JSON expression.
- `priorTableInFrom` resolves `t.doc` inside `JSON_TABLE(t.doc, ...)`.

Implementation:

- Add `extractJsonTable`.
- Add JSON table column pseudo results.
- Thread `priorTableInFrom` exactly as current extractor does.

### Phase 8: Views And Metadata Recursion

Status: done.

Goal: Keep view-derived columns working with omni recursion.

Tests first:

- existing StarRocks view fixtures.
- MySQL view metadata with `SELECT *`.
- view with aliases.

Implementation:

- Change `getColumnsForView` to call omni extractor once the needed phases are ready.
- Keep old behavior until cutover if needed for reference path.

### Phase 9: NotFound And Mixed System/User Behavior

Status: done.

Goal: Preserve error recovery and masking-relevant behavior.

Tests first:

- missing table
- missing column
- mixed `mysql.user` and user table query
- all-system table query returns `SelectInfoSchema` and empty source columns

Implementation:

- Route `ResourceNotFoundError` into `QuerySpan.NotFoundError`.
- Preserve `MixUserSystemTablesError`.
- Ensure all-system access tables are suppressed in returned span.

### Phase 10: Strict Golden Gate

Status: done.

Goal: Make the fixture harness strict and reach zero diffs against checked-in YAML expectations.

Steps:

1. Change the harness to fail when diffs exist.
2. Run it against all current fixtures.
3. Fix remaining diffs.
4. After cutover, keep the harness as a golden check and ensure it does not compare `GetQuerySpan` to the same internal omni path.

Proof:

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/mysql -run '^TestMySQLOmniQuerySpanGoldenHarness$'
```

Current result: `30/30 matched, 0 diffs`.

### Phase 10.5: Post-Review Systematic Correction

Status: done.

Goal: Close the review findings and add regression coverage for the migration failure modes that let them through.

Root causes:

1. The migration moved from ANTLR's recursive parse-tree traversal to omni's explicit AST type switches, but some omni child fields were not wired into lineage extraction.
2. Probe tests verified AST shape availability, not extractor behavior.
3. The post-cutover golden harness initially used `GetQuerySpan` as the reference even though `GetQuerySpan` already delegated to omni.
4. The fixture corpus was too small to cover long-tail root statements and expression nodes from the old extractor.

Regression tests:

- `TestOmniQuerySpanSystematicMigrationRegressions/derived_table_column_aliases_are_applied`
- `TestOmniQuerySpanSystematicMigrationRegressions/in_subquery_sources_are_part_of_result_lineage`
- `TestOmniQuerySpanSystematicMigrationRegressions/explicit_expression_nodes_do_not_drop_lineage`
- `TestOmniQuerySpanSystematicMigrationRegressions/legacy_dml_roots_stay_dml`
- `TestOmniQuerySpanSystematicMigrationRegressions/table_and_values_roots_return_select_results`
- `TestMySQLOmniQuerySpanGoldenHarness`

Implementation corrections:

- Apply `SubqueryExpr.Columns` to derived table pseudo columns with length validation.
- Merge `InExpr.Select` result sources into the expression lineage.
- Add explicit expression handlers for `EXISTS`, `CONVERT`, `COLLATE`, `MATCH`, `ROW`, `MEMBER OF`, `INTERVAL`, and `DEFAULT`.
- Return an error for unsupported omni expression node types instead of silently returning empty lineage.
- Classify `CALL`, `DO`, and `HANDLER` roots as DML.
- Extract `TABLE` and `VALUES` roots as select-family results.

### Phase 11: Cutover

Status: done.

Goal: Make omni the only MySQL query-span production path.

Steps:

1. Change public `GetQuerySpan` path to use the omni extractor.
2. Rename or merge `omniQuerySpanExtractor` into `querySpanExtractor`.
3. Delete ANTLR context methods from query-span code.
4. Delete `selectOnlyListener`, `accessTableListener`, `resourceExtractListener`, and `recursiveCTEExtractListener`.
5. Delete ANTLR `queryTypeListener` or leave only if non-query-span code still uses it.
6. Keep `ParseMySQL` only for other modules that still require it.
7. Run no-ANTLR query-span grep:

```bash
rg 'ParseMySQL\(|GetANTLRAST|antlr4-go|github.com/bytebase/parser/mysql' backend/plugin/parser/mysql/query_span*.go
```

Expected: no production query-span dependency. Test-only parity code may be deleted or excluded.

### Phase 12: Cleanup

Status: done.

Goal: Remove temporary migration scaffolding.

Steps:

1. Decide whether to keep `query_span_omni_probe_test.go` as a permanent parser-shape regression test.
2. Keep `query_span_omni_parity_test.go` as a golden harness after `TestGetQuerySpan` is omni-backed and strict fixture tests pass.
3. Update this plan with final status.
4. Run full required Go checks.

## Global Verification

Focused query-span proof:

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/mysql -run '^(TestGetQuerySpan|TestMySQLOmniQuerySpanMigrationProbe|TestOmniQuerySpan|TestMySQLOmniQuerySpanGoldenHarness)'
```

Package proof:

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/mysql
```

Repository gates before PR:

```bash
gofmt -w <modified go files>
golangci-lint run --allow-parallel-runners
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

## Cutover Gate

- Probe test passes.
- Focused omni extractor tests pass.
- Strict golden harness reports 0 diffs.
- Existing `TestGetQuerySpan` passes unchanged after switching to omni.
- No production query-span dependency on ANTLR remains.
- `golangci-lint run --allow-parallel-runners` passes.
- Server build passes.

Only after this gate is satisfied should the branch be merged.
