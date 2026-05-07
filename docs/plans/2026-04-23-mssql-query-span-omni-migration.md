# MSSQL query_span_extractor omni migration — full plan

Date: 2026-04-23
Owner: jy
Scope: `backend/plugin/parser/tsql/query_span_extractor.go` (3743 LOC) + `query_span_predicate.go` (already has omni counterpart)

## Goal

End state: every code path in MSSQL query-span extraction runs on omni AST. `query_span_extractor.go` and `query_span_predicate.go` deleted. `tsql/omni.go`'s ANTLR fallback removed. No regression on `TestGetQuerySpan` YAML corpus or downstream consumers (advisors, masking, lineage).

**Merge policy**: all phases happen on this branch in isolation. The branch is merged *only after* the full migration is complete and the ANTLR path is gone. No intermediate release, no feature flag on main, no dual-path in production.

Non-goals: migrating non-query-span tsql files (resource_change, backup/restore, completion, diagnose, get_database_metadata). Those are tracked separately.

## Architecture

Single extractor. No dual-parse. Omni produces the AST; extractor consumes it.

```
GetQuerySpan
  → ParseTSQLOmni(stmt) → []omnimssql.Statement
  → pre-pass: collect DECLARE @t TABLE into gCtx.TempTables
  → omniQuerySpanExtractor.run(stmts)
      → classifyQueryType(stmts[0].AST)   (already omni)
      → if not SELECT: return span with only Type + access tables
      → collectAccessTables(stmts[0].AST)
      → isMixedQuery check
      → extractFromSelectStmt(sel) → PseudoTable
      → predicate-columns collection (uses existing omni helper + subquery-output merge)
      → return QuerySpan
```

Struct layout stays the same as `querySpanExtractor`: `ctx, defaultDatabase, defaultSchema, ignoreCaseSensitive, gCtx, ctes, outerTableSources, tableSourcesFrom, predicateColumns`. The current Phase-0 `omniQuerySpanExtractor` embeds `*querySpanExtractor` for helper reuse; at cutover we rename the embedded struct to the sole `querySpanExtractor` and drop the wrapper.

## Shared helpers (keep, no parallel copies)

These take strings and are reused as-is:

- `tsqlFindTableSchemaByParts(linkedServer, db, schema, table)` — already extracted Phase 0
- `tsqlIsFieldSensitive(db, schema, table, column)` — pure string
- `tsqlGetAllFieldsOfTableInFromOrOuterCTE(db, schema, table)` — pure string
- `isIdentifierEqual(a, b)` — pure
- `unionTableSources(...TableSource)` — pure
- `isMixedQuery`, `isSystemResource` — pure
- `getColumnsFromCreateView(definition, db, schema)` — recurses into query-span on the view body; once the extractor is omni, this becomes self-hosting

Added during migration:

- `resolveOmniExpression(ast.ExprNode) (base.QuerySpanResult, error)` — the omni equivalent of `getQuerySpanResultFromExpr`
- `extractOmniFromSubquery(sel *ast.SelectStmt) (*base.PseudoTable, error)` — clone-scope descent
- `collectOmniAccessTables(ast.Node) base.SourceColumnSet` — mirrors the ANTLR accessTableListener

## Coverage map (every ANTLR entry point → omni destination)

| ANTLR function | LOC | Omni replacement | Phase |
|---|---:|---|---|
| `getQuerySpan` | ~85 | `getOmniQuerySpan` | 11 (cutover) |
| `tsqlSelectOnlyListener` walk | ~40 | explicit statement-level iteration | 11 |
| `EnterDeclare_statement` (temp table) | ~25 | `preCollectTempTables(stmts)` | 8 |
| `EnterDml_clause` | ~15 | dispatch inside `getOmniQuerySpan` | 11 |
| `extractTSqlSensitiveFieldsFromSelectStatementStandalone` | ~110 | folded into `extractFromSelectStmt` (With + set ops) | 1/5/6 |
| `extractTSqlSensitiveFieldsFromSelectStatement` | ~15 | same | 1 |
| `extractTSqlSensitiveFieldsFromQueryExpression` | ~50 | `extractFromSelectStmt` handles Larg/Rarg | 5 |
| `extractTSqlSensitiveFieldsFromQuerySpecification` | ~75 | `extractFromQuerySpec` | 1/2 |
| `extractTSqlSensitiveFieldsFromTableSources` | ~20 | `extractFromClause` | 2 |
| `extractTSqlSensitiveFieldsFromTableSource` | ~65 | `extractTableSourceWithJoins` | 2 |
| `extractTSqlSensitiveFieldsFromTableSourceItem` | ~75 | `extractTableSourceItem` | 2/3/7 |
| `extractTSqlSensitiveFieldsFromDerivedTable` | ~35 | `extractDerivedTable` | 3 |
| `extractTSqlSensitiveFieldsFromTableValueConstructor` | ~20 | `extractValuesClause` | 3 |
| `extractTSqlSensitiveFieldsFromSubquery` | ~5 | `extractFromSubquery` | 4 |
| `tsqlFindTableSchema` | ~95 | use `tsqlFindTableSchemaByParts` (shared) | 0 ✅ |
| `getColumnsFromCreateView` | ~35 | unchanged (self-hosting after cutover) | 11 |
| `tsqlGetAllFieldsOfTableInFromOrOuterCTE` | ~40 | shared | 0 ✅ |
| `tsqlIsFullColumnNameSensitive` | ~10 | replaced by direct `tsqlIsFieldSensitive` calls from `*ColumnRef` | 1 |
| `tsqlIsFieldSensitive` | ~90 | shared | 0 ✅ |
| `getQuerySpanResultFromExpr` | ~2525 | `resolveOmniExpression` | 1 (largest) |
| `unionTableSources` | ~20 | shared | 0 ✅ |
| `getAccessTables` + listener | ~80 | `collectOmniAccessTables` | 9 |
| `isMixedQuery` + `isSystemResource` | ~30 | shared | 0 ✅ |
| `splitTableNameIntoNormalizedParts` | ~30 | unused after omni (TableRef fields directly) | drop at 11 |
| `normalizeFullTableNameFallback` | ~30 | unused after omni | drop at 11 |
| `unquote` | ~15 | unused (omni already unquotes) | drop at 11 |
| `getSelectBodyFromCreateView` | ~30 | unused once omni parses views directly | drop at 11 |

## Phases

### Phase 0 — Foundation (DONE)

✅ `tsqlFindTableSchemaByParts` extracted
✅ `collectOmniPredicateColumnRefs` + test
✅ `omniQuerySpanExtractor` scaffold: single-table SELECT, WHERE predicates, bare/qualified/`*`/`t.*`/`AS alias` in target list
✅ 25 tests under `TestOmniQuerySpan_SupportedShapes/NotFound/UnsupportedShapes`

### Phase 1 — Expression resolver `resolveOmniExpression`

The largest single piece. Mirrors ANTLR's `getQuerySpanResultFromExpr` (2525 LOC). Each omni ExprNode type gets a case:

| Omni node | Source columns behavior |
|---|---|
| `*ColumnRef` | `tsqlIsFieldSensitive` lookup; IsPlainField=true |
| `*StarExpr` | expand via `tsqlGetAllFieldsOfTableInFromOrOuterCTE`; caller treats as multiple |
| `*Literal`, `*VariableRef`, `*StarExpr` (no qualifier in scalar context) | empty sources |
| `*BinaryExpr`, `*UnaryExpr`, `*BetweenExpr`, `*LikeExpr`, `*IsExpr`, `*InExpr` | merge sources of sub-exprs |
| `*CaseExpr`, `*IifExpr`, `*CoalesceExpr`, `*NullifExpr` | merge all branches |
| `*CastExpr`, `*ConvertExpr`, `*TryCastExpr`, `*TryConvertExpr` | unwrap |
| `*ParenExpr`, `*CollateExpr`, `*AtTimeZoneExpr` | unwrap |
| `*FuncCallExpr` | merge all Args; recurse into OverClause (partition/order exprs contribute) |
| `*MethodCallExpr` | merge Args |
| `*SubqueryExpr` (scalar subquery) | clone extractor with outerTableSources, extract, use first result column's sources; also merge subquery's predicate/source columns into outer's predicateColumns |
| `*SubqueryComparisonExpr` | Left sources + subquery output sources |
| `*ExistsExpr` | subquery's predicate columns (no value column) |
| `*FullTextPredicate` | Columns + Value sources |
| `*ResTarget`, `*SelectAssign` | unwrap Val; Name becomes result Name |
| `*GroupingSetsExpr`, `*RollupExpr`, `*CubeExpr` | merge args (appear in GROUP BY, not SELECT usually) |

Tests: one focused subtest per node type (≈ 30). Includes: arithmetic, nested CASE, CAST chain, COALESCE of columns, function with column args, aggregate with OVER (partition by different col than select), scalar subquery, correlated scalar subquery.

Special attention: IsPlainField flag semantics — true iff the expression is exactly a `*ColumnRef` (or `*ParenExpr` wrapping one). Check ANTLR behavior for edge cases.

### Phase 2 — Multi-table FROM + JOIN + ON

Handlers:

- `extractFromClause(*ast.List) ([]TableSource, error)` — iterate items, each is a table tree (JoinClause, TableRef, AliasedTableRef, etc.). Concatenate into tableSourcesFrom.
- `extractTableSourceWithJoins(node ast.Node) ([]TableSource, error)`
  - `*ast.JoinClause`: recurse Left + Right, concat; ON condition gets extracted to collect predicate columns (needs a separate pre-walk; the predicate helper already covers this in `collectOmniSelectPredicateColumnRefs`)
  - other → `extractTableSourceItem`

Scope note: JOIN `Left` and `Right` contribute to the same scope. Multiple comma-joined items in FromClause also all in same scope.

Tests: INNER, LEFT, RIGHT, FULL, CROSS JOIN; 3-table chain `a JOIN b ON ... JOIN c ON ...`; comma join with mixed JOIN; join with a derived table; USING clause (omni `JoinClause.Using` is a *List).

### Phase 3 — Derived tables + column alias lists

- `*AliasedTableRef` wrapping a `*SubqueryExpr` → derived table: recurse with clone-extractor that inherits tableSourcesFrom as outerTableSources; apply alias as PseudoTable.Name; apply column alias list positionally (rename each column).
- `*AliasedTableRef` wrapping a `*ValuesClause` → VALUES-derived table: each row is a tuple of expressions; column count from first row; Alias name list renames.
- Bare `*ValuesClause` in FromClause → same as above minus alias.
- Column alias list on a `*AliasedTableRef{Table: *TableRef}` → rename physical columns (rare but legal for TVFs).

Length-mismatch check: error mirrors ANTLR's `"number of column alias %d does not match the number of columns %d"`.

Tests: `(SELECT a,b FROM t) AS x`; `(SELECT a,b FROM t) AS x(c1,c2)`; `(VALUES (1,2),(3,4)) AS v(a,b)`; nested derived table; derived table with ORDER BY / GROUP BY inside.

### Phase 4 — Subqueries in expressions + scope management

- Scalar subquery `(SELECT ...)` appearing as ExprNode: resolveOmniExpression calls `extractFromSubquery(sq.Query.(*SelectStmt))` with a cloned extractor whose outerTableSources includes the outer scope's tableSourcesFrom. The scalar's "source" for outer attribution: first result column's SourceColumns. Also merge clone's predicateColumns into outer.
- `*ExistsExpr`: same clone-scope; contributes only predicate columns (no result value).
- `*InExpr.Subquery`, `*SubqueryComparisonExpr.Subquery`: same clone-scope; result's SourceColumns get merged into outer predicateColumns (this is exactly the "subquery output as outer predicate" that the current predicate helper deliberately skips — it gets filled in here).

After this phase, the predicate helper's limitation goes away because scope-cloned extraction captures both direct refs AND subquery output refs.

Tests: correlated subquery (inner references outer alias); nested 2/3-level correlations; EXISTS with predicate on inner + correlation to outer; IN subquery where outer and inner share column names.

### Phase 5 — Set operations

`*ast.SelectStmt.Op` ∈ {Union, Intersect, Except} with Larg/Rarg:

- Recurse into Larg → anchor results
- Recurse into Rarg → new results
- Length must match; for each column, merge SourceColumns. Anchor name wins.
- `All` flag doesn't affect column set, only runtime semantics

Tests: `A UNION B`; `A UNION ALL B`; `A INTERSECT B EXCEPT C` (chained — omni represents as a nested SelectStmt tree); set op inside a derived table.

### Phase 6 — CTE (WITH clause)

Pre-process WithClause before the body:

- For each `*ast.CommonTableExpr`:
  - Clone extractor with empty tableSourcesFrom (CTE can see previous CTEs via q.ctes but not the outer FROM)
  - Extract CTE body (recursion: CTE body is a `*ast.SelectStmt`)
  - Apply column alias list if `.Columns` is non-empty (positional rename)
  - Build a PseudoTable with CTE name + result columns
  - Push to `q.ctes` (order matters: later CTEs can see earlier)

Recursive CTE: not explicitly handled by ANTLR code either; we match that behavior.

Tests: simple CTE; CTE with explicit column list; chain of CTEs where CTE2 uses CTE1; CTE shadowing a physical table name; deeply nested CTE references.

### Phase 7 — Special table sources

- **PIVOT**: `*ast.PivotExpr{Source, AggFunc, ForCol, InValues, Alias}` — result columns = source's non-pivot columns (need to subtract the FOR column) + one column per IN value, each pivoted aggregate. Source columns for pivoted columns = the AggFunc's source columns. This is new semantic work; ANTLR returned "not supported yet", so the omni extractor preserves that behavior with a typed unsupported error. Real PIVOT support is a separate follow-up.
- **UNPIVOT**: similar — emulate current ANTLR (unsupported).
- **CROSS APPLY / OUTER APPLY**: `*JoinClause{Type: CrossApply|OuterApply}`. The Right is a table expression (often a TVF or subquery) that can reference Left columns (correlated). Treat as a join where Right is evaluated in a scope that includes Left. Contribution to tableSourcesFrom: both sides.
- **Table-valued function call (TVF)**: `*FuncCallExpr` appearing in FROM position. Look up return signature — complicated, requires schema metadata for user TVFs. The omni extractor hardcodes common system TVFs (`OPENJSON`, `STRING_SPLIT`) and returns a typed unsupported error for unknown TVFs.
- **Temp tables**: `#t` → PseudoTable from gCtx.TempTables; `@t` (TableVarRef) → same lookup.
- **TableVarMethodCallRef** (XML `.nodes()`): emits a table of XML fragments; columns named after the method's alias columns; no physical source.
- **CHANGETABLE**: return PseudoTable (stats only, no sensitive data flow). Matches ANTLR behavior.

Tests: temp table declared + used; @t variable table used; XML nodes method; CROSS APPLY with TVF; OUTER APPLY; `OPENJSON(...)`; CHANGETABLE.

### Phase 8 — Multi-statement pre-pass for temp tables

Current ANTLR runs the listener across ALL statements in `GetQuerySpan`, picking up `DECLARE @t TABLE` from previous statements before processing the final SELECT. Omni needs the same:

- `preCollectTempTables(stmts []omnimssql.Statement)` walks every stmt
- For each `*ast.DeclareStmt` with `VariableDecl.IsTable == true`, extract columns from `VariableDecl.TableDef`, build a PseudoTable, put into `gCtx.TempTables[name]`

Also handle `CREATE TABLE #temp (...)`: omni `*CreateTableStmt` where name starts with `#`. Same pre-collect.

Tests: `DECLARE @t TABLE(a INT, b INT); SELECT a FROM @t`; `CREATE TABLE #tmp (id INT); SELECT id FROM #tmp`.

### Phase 9 — Access tables

`collectOmniAccessTables(root ast.Node) base.SourceColumnSet`:

- `ast.Inspect` to walk the whole AST
- Collect `*ast.TableRef` seen in table-source positions (FromClause, JoinClause, PivotExpr.Source, DerivedTable, CTE body FROM, subquery FROM)
- NOT inside: FuncCallExpr.Name (which happens to be *TableRef for qualified function names — but those are functions, not tables)

Distinguishing is tricky. Approach: rather than inspect-everywhere, walk only through table-source-bearing fields explicitly. Start from SelectStmt, descend into FromClause + subqueries + CTEs. Skip TargetList, WhereClause expressions (unless we enter a subquery's FromClause via a SubqueryExpr).

Parity check: run both ANTLR's `getAccessTables` and omni's `collectOmniAccessTables` on every fixture, assert set-equality.

### Phase 10 — INTO, FOR XML/JSON, OPTION, hints

- `SELECT INTO target_table`: target becomes a write resource; not part of sensitive field extraction but appears in access tables as a target (current ANTLR doesn't specially mark this; we match)
- `FOR XML`, `FOR JSON`: result becomes a single-column text blob; current ANTLR extractor doesn't specially handle this (just runs regular extraction); we match. Possible follow-up: single-column collapse. Note for later.
- `OPTION (...)` / table hints: ignore; they don't affect column flow
- `TABLESAMPLE`: ignore; row-level only

Tests: SELECT INTO #tmp FROM t (temp destination); FOR XML AUTO (ensure no crash, output matches current behavior).

### Phase 11 — Cutover

1. `GetQuerySpan` stops using ANTLR. Entry point switches to `getOmniQuerySpan`.
2. Delete `query_span_predicate.go`.
3. Delete all ANTLR-context-taking methods from `querySpanExtractor` (they're duplicated now as omni versions).
4. Rename `omniQuerySpanExtractor` → `querySpanExtractor`; merge structs.
5. Remove unused ANTLR-only helpers: `normalizeFullTableNameFallback`, `splitTableNameIntoNormalizedParts`, `unquote`, `getSelectBodyFromCreateView`, `NormalizeTSQLIdentifier` (if only used here — grep).
6. `omni.go`'s `AsANTLRAST()` lazy ANTLR fallback: check if any other tsql file still uses it; drop if not.

All YAML fixtures must pass. Parser invariant tests (`TestOmniQuerySpanParserInvariants`) must still pass. New extractor tests all green. Lint clean. Build clean.

### Phase 12 — Cleanup

- Delete the now-unused `query_span_extractor.go` ANTLR code (once fully replaced)
- Keep parser AST shape coverage as `query_span_parser_invariant_test.go`.
- Update `AGENTS.md` / skill docs if they still reference the old file layout
- Open follow-up tickets for items deferred: PIVOT/UNPIVOT result-column inference, TVF metadata, SELECT INTO target marking, FOR XML single-column collapse

## Test strategy

### Existing-fixture parity (gating at cutover only)
`TestGetQuerySpan` YAML corpus (29 cases across 6 files) MUST pass unchanged *after* Phase 11 cutover. Before cutover, the omni extractor is a separate code path that only runs via its own unit tests; ANTLR remains primary. Between phases, we don't need to keep the fixtures green via the omni path — that happens in one shot at cutover.

### Per-phase unit tests
Each phase adds a focused block in `query_span_extractor_omni_test.go`. Target 15-30 cases per phase, grouped by phase name:

- Phase 1: `TestOmniQuerySpan_Expressions` — one subtest per ExprNode type + edge cases
- Phase 2: `TestOmniQuerySpan_Joins`
- Phase 3: `TestOmniQuerySpan_DerivedTables`
- Phase 4: `TestOmniQuerySpan_Subqueries`
- Phase 5: `TestOmniQuerySpan_SetOps`
- Phase 6: `TestOmniQuerySpan_CTE`
- Phase 7: `TestOmniQuerySpan_SpecialTableSources`
- Phase 8: `TestOmniQuerySpan_TempTables`
- Phase 9: `TestOmniQuerySpan_AccessTables`
- Phase 10: `TestOmniQuerySpan_IntoForClauses`

### Differential test (development aid, not CI gate)
Add a test-only helper `diffQuerySpan(ctx, stmt, ...)` that runs both ANTLR and omni paths on the same input and reports structural diffs (result count, names, source column sets, predicate columns, access tables, NotFoundError parity). Run it locally across the full YAML corpus as a progress thermometer: `TestQuerySpan_AntlrOmniParity`. Used to track "how much of the corpus omni currently matches". Expected to show diffs until Phase 10 is complete. At Phase 11, it must show 0 diffs — that's the cutover gate. Delete the test as part of Phase 12 cleanup.

### Oracle test (optional, post-cutover)
Add testcontainer MSSQL. For each YAML fixture, execute the SQL against MSSQL with the metadata we mocked, capture `sys.dm_exec_describe_first_result_set` or similar. Verify our extracted column names match the server's view of the result. Not a blocker for cutover but valuable as a follow-up.

## How the two paths coexist on-branch (no dispatch)

No flag, no env var, no runtime dispatch. The omni extractor lives as a parallel package-internal implementation in `query_span_extractor_omni.go`. `GetQuerySpan` keeps calling the ANTLR path as it does today, unchanged.

Validation of omni progress during development:
1. **Unit tests** (`TestOmniQuerySpan_*`) — direct calls into `newOmniQuerySpanExtractor(...).getOmniQuerySpan()`, bypassing `GetQuerySpan`. This is where every phase's coverage is proven.
2. **Parity harness** (`TestQuerySpan_AntlrOmniParity`, dev-only) — table-driven over the YAML corpus; calls both ANTLR via `GetQuerySpan` and omni directly; reports diffs. Useful as a thermometer; not wired to CI until cutover.

Unsupported omni constructs return typed unsupported errors from the production path.

At Phase 11, `GetQuerySpan` is rewritten to call the omni extractor directly (no fallback). The development-only unsupported sentinel is deleted along with the ANTLR extraction code.

## Risk register

| # | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| R1 | Omni parser chokes on an unusual fixture form | Low (probe says 29/29) | High | Probe test already in place; extend before each phase |
| R2 | Subquery scope bug: outer ref resolving to wrong alias | Medium | High | Mirror ANTLR's clone-extractor pattern byte-for-byte; explicit test cases for 3-level correlated subquery with alias shadowing |
| R3 | IsPlainField flag diverges (ANTLR marks `(col)` as plain but `col+0` as not) | Medium | Medium | Mirror exact ANTLR logic in resolveOmniExpression; diff test will catch |
| R4 | CTE scoping / ordering differs (omni might reorder CTEs) | Low | Medium | Use CTE items in omni order (matches SQL order); test chain of 3 CTEs |
| R5 | Temp table pre-pass misses a form (CREATE TABLE #t with constraints, SELECT INTO #t) | Medium | Medium | Handle both DeclareStmt and CreateTableStmt; test both |
| R6 | Access-tables set not identical to ANTLR's accessTableListener | Medium | Medium | Diff test on every fixture + ad-hoc corpus |
| R7 | View body parsing recursion — omni-in-omni infinite loop if view definition is malformed | Low | Low | Existing code uses `newQ.getQuerySpan(q.ctx, selectBody)`; recursion limit handled by parser; same pattern reused |
| R8 | PIVOT/UNPIVOT regression: current ANTLR returns empty for pivot, omni might produce something different | Low | Low | Emulate ANTLR's current behavior (unsupported) in Phase 7; PIVOT support is a separate follow-up |
| R9 | Linked server (`srv.db.schema.obj`) currently errors; omni preserves Server field, we might accidentally start resolving it | Low | Medium | Keep the `if server != ""` early return in tsqlFindTableSchemaByParts |
| R10 | CONTAINS/FREETEXT now a dedicated `*FullTextPredicate` node (new since 569d6f7); ensure resolveOmniExpression / predicate helpers handle it | Low (already tested) | Medium | Predicate helper already handles it; resolveOmniExpression must include the case |

## Timeline estimate

Rough engineer-days (focused, interruptible):

- Phase 1 (expressions): 2d — the big one, ~30 node types, IsPlainField edge cases
- Phase 2 (joins): 0.5d
- Phase 3 (derived tables + column alias): 0.5d
- Phase 4 (subqueries + scope): 1d — scope cloning is tricky
- Phase 5 (set ops): 0.5d
- Phase 6 (CTE): 0.5d
- Phase 7 (special table sources): 1d — CROSS APPLY with correlated TVF needs care
- Phase 8 (multi-statement pre-pass): 0.5d
- Phase 9 (access tables parity): 0.5d
- Phase 10 (INTO / FOR / hints): 0.25d
- Phase 11 (cutover + cleanup): 0.75d
- Phase 12 (housekeeping): 0.25d

Total: ~8 engineer-days. Parity test fixed after each phase, keeping CI green.

## Open questions

1. **Should parser AST shape coverage stay?** Resolved: keep the useful parser AST coverage as `query_span_parser_invariant_test.go`.
2. **PIVOT full support in this migration or later?** Current ANTLR is a no-op ("not supported"). Recommend: keep no-op parity in Phase 7; real PIVOT support = new project.
3. **TVF return-shape metadata**: do we add a gCtx function for it (similar to ListDatabaseNamesFunc)? Recommend: yes, but defer to a separate ticket; the migration uses an empty fallback meanwhile.
4. **Order of `tableSourcesFrom` across comma-join vs JOIN**: ANTLR current code `append`s as it walks left-to-right, JOIN unwraps from inside. Match exactly.
5. **Case-sensitivity flag threading**: The `ignoreCaseSensitive` parameter in `GetQuerySpan` is user-controlled. All identifier comparisons inside omni-extracted data must honor it. Easy (already inherited via embedded `*querySpanExtractor`), just a reminder.

## Definition of done (merge gate)

- [ ] Phases 1-10 each: code + per-phase unit tests + lint clean + `go build` ok (on-branch, incremental)
- [ ] Parity harness `TestQuerySpan_AntlrOmniParity` shows 0 diffs across the full YAML corpus
- [ ] Phase 11 cutover: `GetQuerySpan` calls omni extractor directly; no fallback, no env var; `query_span_predicate.go` deleted; ANTLR-context-taking methods removed
- [ ] `TestGetQuerySpan` (29 fixtures) passes unchanged
- [ ] Phase 12: no reference to `antlr4-go` remains in `query_span*.go`; `omni.go`'s `AsANTLRAST` fallback reviewed and dropped if no other tsql file uses it; parity harness deleted
- [ ] `go build ./backend/bin/server/main.go` passes
- [ ] `golangci-lint run ./backend/plugin/parser/tsql/...` clean
- [ ] Only then: merge branch
