# Query Span Catalog Loader — E3: Eager Per-Object Install with Inline Root-Pseudo Fallback

**Status:** Draft v2.2 (post codex round-2 + main-line re-alignment)
**Author:** junyi
**Date:** 2026-04-15
**Supersedes:** v5 (lazy per-query closure), v1/v2/v3 of this document
**PoC:** `backend/plugin/parser/pg/query_span_v5_poc_test.go` — 30 tests + 6 benchmarks, all passing
**Related issues:** BYT-9215, BYT-9261, BYT-9021, BYT-9117 (mysql reuse)

---

## One-sentence summary

Install every schema object eagerly at catalog init in topological order, using hand-constructed AST nodes fed directly to omni's `DefineX` API. When a real install fails, immediately install a text-backed pseudo version at the same slot so downstream objects install as real. The DDL render path is deleted.

## Problem

`query_span_omni.go:86` calls `schema.GetDatabaseDefinition(POSTGRES, meta)` to render the full schema as a single DDL blob, then `cat.Exec(ddl, ContinueOnError)`. omni's `catalog.Exec` at `omni/pg/catalog/exec.go:33` calls `pgparser.Parse(sql)` on the whole blob first — batch parse. If any one statement has a construct pgparser rejects (BYT-9215: `::` in quoted identifier, BYT-9261: `->>` in index expression), the entire parse fails and `Exec` returns a top-level error before any `ContinueOnError` branch is reached.

Effect: one bad object in one schema makes query span fail for **every** query on that database. Masking silently degrades to `extractFallbackColumns`, which produces empty-source results that bypass most masking rules.

## Why closure is hard (and why E3 sidesteps it)

A previous iteration (v5) tried to solve this lazily: parse the query, extract its object references, install only those, avoid the full schema cost. The investigation showed accurate closure is not computable:

- Function and operator overloads require knowing argument types, which require loading the closure (chicken and egg).
- Composite `(col).field` access, domain base types, inheritance children, partition children — none visible from query syntax alone.
- Dynamic SQL in PL/pgSQL function bodies — fundamentally unknowable.

v5's escape hatch was "install all candidates by name, not signature" — a conservative superset. But this layered on metadata-assisted closure expansion, retry loops, and several other under-specified mechanisms.

E3 abandons the closure problem entirely: **install everything**. The analyzer reads what it needs; the rest is idle catalog weight. Install cost is O(schema), but per-object hand-built install is fast enough (measured 1.5-2.3× faster than current DDL path) that this is not a regression.

## Core insight: cascade through dependency order, pseudo at the root

Consider enum `E` with a broken definition, table `T` with column of type `E`, view `V` over `T`.

**Naive skip-on-failure** (what v5 fail-fast would do):
- `E` install fails → skip
- `T` install fails (can't resolve `E`) → skip
- `V` install fails (can't resolve `T`) → skip
- Any query touching anything in the chain fails.

**E3 inline pseudo**:
- `E` real install fails → immediately install `pseudo(E)` (text-backed enum) at the same slot → `degraded[E] = err`
- `T` real install: references `pseudo(E)` in the catalog, succeeds as **real** object with correct column names
- `V` real install: references real `T`, succeeds as **real** object
- Query on `V` runs with full lineage to base-table columns; the only degradation is that `E`-typed columns behave like `text` for type resolution.

**For cascade prevention to work we need two things**:
1. Topological order — process `E` before `T` before `V`. Otherwise `T`'s failure is indistinguishable from `T` itself being broken.
2. Inline pseudo — fail real, immediately install pseudo at the same slot, in the same loop. No retry, no second pass.

Empirical validation: `TestV5PoC_E3_RealCascadeWithGenuineFailure` forces a genuine `DefineDomain` SQLSTATE 42704 failure, installs a text-backed pseudo at the same name, then installs downstream table and view as real `DefineRelation` / `DefineView` calls, queries the view, and asserts every target is `*catalog.VarExpr` (full lineage to base-table columns). Not a simulation.

## Proposal

### Phase 1 — eager loader

```go
func (l *e3Loader) Load(ctx context.Context, meta *storepb.DatabaseSchemaMetadata) error {
    objects := l.collectObjects(meta)             // flatten metadata into []ObjectEntry
    sccs := l.tarjanSCC(objects)                  // condense cycles
    sorted := l.topoSort(sccs)                    // SCC-level topo sort
    for _, sccGroup := range sorted {
        for _, obj := range lexSort(sccGroup) {   // intra-SCC: deterministic lex order
            if err := l.installReal(obj); err != nil {
                l.degraded[obj.Key()] = err
                if pErr := l.installPseudo(obj); pErr != nil {
                    l.trulyBroken[obj.Key()] = pErr
                }
            }
        }
    }
    return nil
}
```

Single forward sweep. When `Load` returns, the catalog contains every metadata object as either real (degraded may still be false if nothing failed downstream) or pseudo. `trulyBroken` is for objects whose even-pseudo install failed — vanishingly rare, captured for observability.

### Phase 2 — analyze

```go
func (e *omniQuerySpanExtractor) ExtractQuerySpan(ctx context.Context) (*base.QuerySpan, error) {
    if err := e.initCatalog(ctx); err != nil { return nil, err }
    stmt := parseQuery(...)
    analyzed, err := e.cat.AnalyzeSelectStmt(stmt)
    if err != nil {
        return e.extractFallbackSpan(stmt, err)  // classified, see §Fallback classifier
    }
    return e.extractLineage(analyzed, stmt, e.degraded)
}
```

`extractLineage` consults the `degraded` set to mark lineage entries whose source is a pseudo object.

### Phase 3 — what E3 does not do

No closure discovery. No `ExtractCatalogReferences`. No install-retry loop. No lazy install. These were v5 concepts. E3 deletes them.

## Fallback classifier (pure function, used by tests)

The classifier is kept because tests need to assert which reason a given query hit. It is **not** wired into runtime telemetry, metrics, or a flip gate. It is a pure function returning an enum.

```go
type fallbackReason int

const (
    reasonNone fallbackReason = iota
    reasonExpectedPseudoSemantic      // pseudo-induced operator / type mismatch
    reasonUndefinedReference          // analyzer says object not found
    reasonAnalyzerUnsupported         // non-*catalog.Error internal analyzer failure
)

func classifyAnalyzeError(err error) fallbackReason {
    var cErr *catalog.Error
    if !errors.As(err, &cErr) {
        return reasonAnalyzerUnsupported
    }
    switch cErr.Code {
    case catalog.CodeUndefinedFunction, catalog.CodeAmbiguousFunction,
         catalog.CodeDatatypeMismatch, catalog.CodeFeatureNotSupported,
         catalog.CodeAmbiguousColumn:
        return reasonExpectedPseudoSemantic
    case catalog.CodeUndefinedTable, catalog.CodeUndefinedColumn,
         catalog.CodeUndefinedObject, catalog.CodeUndefinedSchema:
        return reasonUndefinedReference
    default:
        return reasonAnalyzerUnsupported
    }
}
```

Three buckets are enough for test assertions. We dropped the `reasonLoaderBugSuspect` / `reasonTrulyBroken` / `reasonUserOrSyncLag` distinction from v3 because it was introduced to feed a runtime flip gate we no longer have. Tests that want to distinguish "loader bug" from "user query referenced non-existent object" do so by constructing the fixture they know the answer to.

The classifier lives in `query_span_e3_classify.go` and is called from all three fallback sites (see §Fallback call-site audit) to give tests a consistent assertion surface. The extractor stores the last classified reason as `e.lastFallbackReason` for tests to check.

## Fallback call-site audit

Three sites today fall through to `extractFallbackColumns` / `fallbackExtractLineage` on any analyzer error, unconditionally:

1. `query_span_omni.go:276-295` — top-level `AnalyzeSelectStmt` in `ExtractQuerySpan`.
2. `query_span_omni_plpgsql.go:85-115` — `analyzeSQLBody` (SQL function body inlining).
3. `query_span_omni_plpgsql.go:405-450` — `analyzeEmbeddedSQL` (embedded SQL in PL/pgSQL blocks).

All three adopt `classifyAnalyzeError` as a pre-fallback step. The fallback itself still runs; the reason is recorded for test inspection. PL/pgSQL is not "zero changes" — it is a shared touchpoint, ~30 LOC per site.

`tryUserFuncTableSource` (`query_span_omni.go:1350+`) remains as a pre-fallback path for RETURNS TABLE functions.

## Dependency graph from metadata

Topological order derives from `storepb.DatabaseSchemaMetadata`. Dependency edges:

| Object | Depends on |
|---|---|
| Schema | none |
| Enum | Schema |
| Domain | Schema |
| Composite type | Schema |
| Range type | Schema |
| Table | Schema; each column's user-defined type reference (extracted from `columns[i].type` string) |
| View | Schema; every entry in `dependency_columns.{schema,table}` |
| MatView | Same as view |
| Function | Schema; each parameter type *if* user-defined; return type *if* user-defined |

**Unavailable metadata** doesn't block E3:

- **Inheritance**: `sync.go:777` uses `INFORMATION_SCHEMA.COLUMNS`, which returns parent-merged columns for child tables. The child's `columns[]` is complete. E3 installs each child as a standalone table.
- **Partition**: same reasoning. Query span doesn't care about partition bounds.
- **Composite / Domain / Range internal type refs**: `storepb` doesn't carry these. E3 treats them as leaves in the dep graph. If real install fails because of an unresolved internal reference, pseudo catches it.

### Cycle handling — Tarjan SCC + lex ordering

Metadata-level cycles are rare (mutually recursive views via `CREATE OR REPLACE`). E3:

1. Tarjan SCC on the dep graph.
2. Topo sort the condensed DAG of SCCs.
3. For each SCC of size > 1:
   - Sort members by `(schema, name)` lexicographically.
   - Process in order. The first member's real install fails (its body references the SCC's other members) → pseudo fallback.
   - Subsequent members reference the first (now pseudo), so their real installs succeed.

For an SCC of size N, exactly one member ends up pseudo; the other N-1 are real with degraded lineage back to the cycle root. Strictly better than "all SCC members become pseudo."

Lex ordering guarantees determinism for test reproducibility.

## Type string grammar

`column.Type` in `storepb.ColumnMetadata` is produced by `sync.go:820-872`. Full surface:

| # | Form | Example | Extractor output |
|---|---|---|---|
| 1 | Built-in scalar | `integer`, `text`, `boolean`, `json`, `jsonb`, `uuid`, `date`, `interval` | `[]` |
| 2 | Built-in with length | `character(10)`, `character varying(255)`, `bit(8)`, `bit varying(8)` | `[]` |
| 3 | Built-in numeric | `numeric(10,2)`, `decimal(8)`, `numeric(5)` | `[]` |
| 4 | Built-in time | `timestamp(3) with time zone`, `time(6) without time zone` | `[]` |
| 5 | USER-DEFINED | `public.task_status`, `myschema.my_domain` | `[{public, task_status}]` |
| 6 | ARRAY of built-in | `_text`, `_int4` (raw `udt_name`, PG internal form) | `[]` |
| 7 | ARRAY of user-defined | `_task_status` (schema stripped by sync.go:834) | `[]` (cannot topo-sort; installs fail → pseudo) |
| 8 | System-schema-qualified | `pg_catalog.int4`, `information_schema.cardinal_number` | `[]` |

Form 7 is an existing sync issue. E3 is resilient to it (pseudo catches). E3 does not fix it.

### Extractor

```go
func extractUserTypeRefs(typeStr string) []UserTypeRef {
    if typeStr == "" { return nil }
    base := stripParens(typeStr)                  // "numeric(10,2)" → "numeric"
    base = stripTimeZoneSuffix(base)              // "timestamp(3) with time zone" → "timestamp"
    if isBuiltin(base) { return nil }             // allow-list
    if strings.HasPrefix(base, "_") { return nil }// PG internal array form
    if schema, name, ok := splitQualified(base); ok {
        if isSystemSchema(schema) { return nil }
        return []UserTypeRef{{Schema: schema, Name: name}}
    }
    return nil
}
```

### Soundness rule

False negatives (missing a user type ref) are acceptable — pseudo catches. False positives (marking a built-in as user type) are not — they corrupt the topo order.

### Golden corpus (used in tests)

```
"integer"                                → []
"text"                                   → []
"bigint"                                 → []
"numeric(10,2)"                          → []
"character varying(255)"                 → []
"timestamp(3) with time zone"            → []
"json" / "jsonb" / "uuid" / "interval"   → []
"public.task_status"                     → [{public, task_status}]
"myschema.my_domain"                     → [{myschema, my_domain}]
"_text"                                  → []
"_task_status"                           → []
"pg_catalog.int4"                        → []
"information_schema.cardinal_number"     → []
""                                       → []
```

Built-in allow-list: ~50 entries, hardcoded from PG standard types. Additions require a golden test.

## Pseudo install forms (PoC-verified)

Every pseudo form is covered by a green test in `query_span_v5_poc_test.go`.

| Object | Pseudo AST | PoC test |
|---|---|---|
| Enum | `CreateEnumStmt{Vals: empty}` | `TestV5PoC_E3_PseudoEnum_EmptyVals` |
| Domain | `CreateDomainStmt{Typname: text}` | `TestV5PoC_E3_PseudoDomain_OverText` |
| Composite | `CompositeTypeStmt{Coldeflist: [_broken text]}` | `TestV5PoC_E3_PseudoComposite_FieldsAllText` |
| Range | `CreateRangeStmt{subtype: text}` | `TestV5PoC_E3_PseudoRange_SubtypeText` |
| Table | `CreateStmt{cols: [(name, text) for name in metadata]}` | Implicit; pseudo is `DefineRelation` with all-text columns |
| View | `ViewStmt{Query: SELECT NULL::text AS c1, NULL::text AS c2 ...}` | `TestV5PoC_E3_PseudoView_ConstantTargetList` |
| MatView | `CreateTableAsStmt{Query: SELECT NULL::text AS ...}` | Variant of pseudo view |
| Function | `CreateFunctionStmt{params: [text], returns: text, body: SELECT $1}` | `TestV5PoC_E3_PseudoFunction_AllText`; selection asserted by `TestV5PoC_E3_OverloadSelectionAsserted` |

**Cascade prevention end-to-end**: `TestV5PoC_E3_RealCascadeWithGenuineFailure`. Real `DefineDomain` 42704 failure → pseudo at same slot → downstream table + view install as real objects → query on view has full lineage to base-table columns.

**Composite field access limitation**: `storepb` doesn't carry composite field names, so pseudo composite uses a single `_broken` field. Queries using `(col).field` fall back to `extractFallbackColumns`. This is no worse than the current path (which also fails BYT-9215-class queries). Medium-term follow-up: add `CompositeTypeMetadata` to proto + sync.

## `storepb` gaps and how E3 handles them

| Gap | E3 handling | Fidelity impact |
|---|---|---|
| `DomainMetadata` missing | Pseudo domain over text | Domain constraints not visible (query span doesn't use them) |
| `RangeTypeMetadata` missing | Pseudo range subtype=text | Range-specific operators degrade to fallback |
| `CompositeTypeMetadata` missing | Pseudo composite with single field | `(col).field` falls back; top-level `SELECT col` works |
| `TableMetadata.inherits_parent` missing | Child's columns already parent-merged by sync | Zero impact |

Only composite field access is a real fidelity loss, and it is no worse than today's `initCatalog` failure on BYT-9215/9261.

## `initCatalog` shrinks

Before (`query_span_omni.go` around line 86):
```go
func (e *omniQuerySpanExtractor) initCatalog(ctx context.Context) error {
    ddl := schema.GetDatabaseDefinition(POSTGRES, e.meta)
    _, err := e.cat.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true})
    // fallback handling when err is non-nil
}
```

After:
```go
func (e *omniQuerySpanExtractor) initCatalog(ctx context.Context) error {
    e.cat = catalog.New()
    e.cat.SetSearchPath(e.searchPath)
    loader := newE3Loader(e.cat, e.meta)
    if err := loader.Load(ctx); err != nil {
        return err  // only catastrophic errors, e.g. ctx cancellation
    }
    e.degraded = loader.degraded
    e.trulyBroken = loader.trulyBroken
    return nil
}
```

The DDL render path is deleted outright. BYT-9215 and BYT-9261 cannot recur by construction: no DDL text is parsed during install. The only SQL parser invocations in Phase 1 are for view/matview bodies (SELECT statements on a stable code path).

## Search path (out of scope)

Two pre-existing search-path issues were scoped into v2/v3 drafts via codex review iterations, but they are orthogonal to BYT-9215 / BYT-9261 and not caused or worsened by E3. They are moved to §Follow-ups:

- `ExtractAccessTablesOption.DefaultSchema` is a single string, forcing callers to pass `e.searchPath[0]` and drop the rest of the search path.
- Fallback path `query_span_omni.go:963` resolves unqualified relations with `searchPath[0]` only.

Neither is a regression E3 introduces, neither blocks BYT-9215 / BYT-9261 remediation, and fixing either would pull unrelated call sites (including `resource_change.go`) into the E3 PR. E3 preserves current search-path behavior unchanged.

`$user` expansion is also unchanged: `query_span.go:26` calls `meta.GetSearchPath()` (user-less variant at `backend/store/model/database.go:222`). Plumbing `currentUser` through `GetQuerySpanContext` is a separate product decision.

## File layout

New files under `backend/plugin/parser/pg/`:

```
query_span_e3_type_name.go    — typeNameFromString (cheat parser) + extractUserTypeRefs + allow-list
query_span_e3_builders.go     — buildCreateStmt, buildViewStmt, buildCreateEnumStmt, ...
query_span_e3_pseudo.go       — pseudoCreateStmt, pseudoViewStmt, pseudoEnum, ...
query_span_e3_loader.go       — e3Loader{} + Load() + collectObjects + Tarjan SCC + topoSort
query_span_e3_classify.go     — classifyAnalyzeError + fallbackReason
query_span_e3_test.go         — loader, topo, cycle, pseudo, classifier, type-grammar unit tests
query_span_e3_integration_test.go — BYT repros + cascade + search path + PL/pgSQL integration tests
```

Modified:
```
query_span_omni.go            — initCatalog body replaced; fallback at :276 uses classifier
query_span_omni_plpgsql.go    — analyzeSQLBody and analyzeEmbeddedSQL fallbacks use classifier
```

Kept unchanged:
- `query_span_v5_poc_test.go` — 30 tests + 6 benchmarks, locked in as regression baseline
- `extractLineage` core — operates on the same analyzer output
- `base.QuerySpan` contract (additive only: new `Degraded` bit on `SourceColumn`)

## The `Degraded` field on result columns

`base.QuerySpan.Results[i].SourceColumns[j]` gains a boolean `Degraded` bit:

- `Degraded=false`: fully resolved to a real table column.
- `Degraded=true`: source is a pseudo object; column name is correct but type-level fidelity reduced.

Masking policy: conservative — `Degraded=true` sources are still subject to masking (treated as "real enough"). This is a strict improvement over today, where catalog failures produce empty sources that silently bypass masking.

## Key decisions

**D1. Eager per-object install, not lazy closure.** Sound by construction; closure discovery is heuristic. Measured 1.5-2.3× faster than current DDL path on synthetic schemas.

**D2. Topological order via Tarjan SCC + intra-SCC lex ordering.** Makes skip-on-failure composable with downstream resolution. Cycles handled by pseudo-ing one member (lex-first), rest install real.

**D3. Inline pseudo at the failed slot, not downstream or at query time.** Preserves catalog state invariant: after processing obj X's slot, X is in the catalog (real or pseudo).

**D4. Pseudo types are text-based.** text has the most permissive operator resolution and implicit-cast surface in PG. PoC validates every tested query pattern.

**D5. No catalog cache.** Current path is per-query O(schema); E3 is the same shape but faster. Caching is a post-E3 optimization if profiling demands it.

**D6. Ignore inheritance and partition parent metadata.** `sync.go:777` flattens.

**D7. Composite field access is a known degradation.** `storepb` lacks composite field names today. Medium-term follow-up: add `CompositeTypeMetadata`.

**D8. No proto changes in the E3 PR.** Ship the blast-radius fix independent of metadata enrichment.

**D9. No omni PR required.** v2/v3 briefly considered a prerequisite omni `ErrorIdent` PR. Dropped: the value it unlocked (runtime flip gate precision) is not on the main line now that shipping is by test-driven PR review, not automated gate.

**D10. MySQL reuse (BYT-9117).** Same loader + pseudo pattern, different `build*Stmt` functions. Scheduled as follow-up.

**D11. Classifier is a pure function for test assertions.** Not a runtime gate, not a telemetry sink, not wired into counters. Three reason buckets are enough for tests to distinguish expected degradation from genuine bugs.

## Tests

**The test matrix replaces the shadow-diff harness.** No runtime comparison of two code paths. Instead, a comprehensive set of unit + integration tests exercises every E3 path and every known BYT-issue class. Any regression surfaces in CI, not in production.

### Retention

- All 30 existing PoC tests + 6 benchmarks in `query_span_v5_poc_test.go`. Regression baseline.
- Existing `query_span_test.go` cases run unchanged; they are now covered by the E3 path.
- Existing testcontainer tests (`query_span_ddl_debug_testcontainer_test.go`, `query_span_typecast_testcontainer_test.go`) run unchanged.

### New unit tests (`query_span_e3_test.go`)

- **Loader**:
  - `collectObjects` extracts the expected `ObjectEntry` set from fixture `DatabaseSchemaMetadata`.
  - `tarjanSCC` finds cycles and returns SCC groups in deterministic order.
  - `topoSort` of SCCs produces a valid topological order.
  - `installReal` + `installPseudo` interaction on a fixture with one broken object.
- **Classifier**: each SQLSTATE case in `classifyAnalyzeError` returns the expected reason. Non-`*catalog.Error` → `reasonAnalyzerUnsupported`.
- **Type grammar**: every row in §Type string grammar golden corpus is asserted against `extractUserTypeRefs`.
- **Pseudo builders**: each `pseudoXStmt` produces an AST that installs via its corresponding `DefineX` call without error. Structural assertions on resulting catalog state.

### New integration tests (`query_span_e3_integration_test.go`)

- **BYT-9215 repro**: construct `DatabaseSchemaMetadata` with a table name `"'lib"."address"`. Assert E3 loader installs successfully (real or pseudo), a query on an unrelated table succeeds with full lineage, and a query on the broken object degrades gracefully.
- **BYT-9261 repro**: table with a `->>` index expression in metadata. Same assertion pattern.
- **Broken enum cascade**: `E` with invalid metadata, `T(col E)`, `V` over `T`. After load, `E` is pseudo, `T` and `V` are real. Query on `V` has `Degraded=true` on the `E`-typed column only.
- **Cyclic view definitions**: two mutually recursive views. After load, one is pseudo (the lex-first), the other is real. Query on either returns lineage; the one referencing the pseudo has `Degraded=true`.
- **Pseudo function selection**: table with int column + text column; pseudo function `fn(text)` coexists with real `fn(int4)`. Assert query `fn(int_col)` picks the int overload, `fn(text_col)` picks the pseudo.
- **Search path — full path resolution**: table `foo` only in second schema of a two-element search path. Assert primary path finds it.
- **Search path — fallback path consistency**: same table under a deliberate analyzer failure. Assert fallback also finds it (`query_span_omni.go:963` walks full path).
- **PL/pgSQL body with broken dependency**: function body references a pseudo-composite. Assert `analyzeSQLBody` routes to `reasonExpectedPseudoSemantic` and returns fallback lineage.
- **PL/pgSQL embedded SQL with type mismatch**: same pattern for `analyzeEmbeddedSQL`.

### Benchmarks

- `BenchmarkV5PoC_Install_*` remain in place (kernel benchmarks).
- Optional additions once full loader exists: benchmark the full `Load()` (including topo, classify, pseudo, PL/pgSQL) on a fixture schema.

## Rollout

Simple:

1. Write plan v2.2 (this document). Done.
2. Implement loader PR(s) per §File layout. Single PR or split into "builders + pseudo + classifier" + "loader + integration" — reviewer's call.
3. All tests in §Tests green.
4. PR review.
5. Merge.
6. Monitor user reports. Fix forward if regressions surface.

No feature flag. No shadow mode. No flip gate. `git revert` is the rollback mechanism if needed. Standard bytebase shipping practice.

### Follow-ups (independent PRs)

- `CompositeTypeMetadata` in proto + sync → recovers `(col).field` lineage fidelity.
- `DomainMetadata` / `RangeTypeMetadata` → when data shows they matter.
- `$user` search-path plumbing → product decision.
- BYT-9117 mysql reuse.
- Four latent `base.QuerySpan` fields (`PredicateColumns`, `PredicatePaths`, `NotFoundError`, `FunctionNotSupportedError`) → tracked independently; orthogonal to load path.

## Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Pseudo composite loses `(col).field` lineage | Medium | Falls back to `extractFallbackColumns`. No worse than current path. |
| `extractUserTypeRefs` false positives corrupt topo | Medium | Golden tests on full grammar (§Type string grammar). Hard contract C5. |
| `extractUserTypeRefs` false negatives miss user types | Low | Install fails → pseudo catches. |
| Function signature parsing (overloads by signature string) misparses | Medium | Golden tests on common forms. |
| Tarjan SCC non-determinism | Low | Intra-SCC lex ordering. Unit test asserts order. |
| Install cost on huge schemas (>10k objects) | Low | Measured 2000 tables at ~4 ms. Extrapolates to ~20 ms at 10k. |
| `transformFuncCall` / `transformAExpr` drop schema qualification | Medium | E3 installs all function/operator candidates globally; drop is masked at query time. File omni tracking issue separately. |
| PL/pgSQL analyzer picks up degraded state unexpectedly | Low | Same `cat`; integration test covers function body using a degraded table. |
| Cycle semantics untested against real mutually recursive views | Medium | Integration fixture with two mutually recursive views asserts one pseudo, one real. |
| Inherited columns not actually complete in metadata | Low | Add one integration test with an inheritance chain. |
| E3 introduces a regression that reaches main | Medium | Test matrix is the safety net (unit + integration + BYT repros + testcontainer). Fix forward if it slips. |

## Alternatives considered

**A. v5 lazy closure + DefineX.** Requires `ExtractCatalogReferences` + closure expansion + retry loop. Four layers gone in E3. Superseded.

**B. ANTLR-approach manual walker.** ~3000 LOC of hand-written walker replacing `AnalyzeSelectStmt`. Overkill: the broken path is `initCatalog`, not the analyzer. Discarded.

**C. Alt D stopgap (per-statement Exec).** Valid as immediate stopgap if E3 will take >1 week, but with E3 starting immediately it adds a PR and a commit target that E3 will obsolete. Skipped.

**D. Catalog cache.** Optimization on top of E3. Deferred until profiling demands it.

**E. Pseudo the cascade victims instead of the root.** Larger degraded set. Rejected in favor of root-pseudo (D3).

**F. All SCC members as pseudo.** Cycle size N → N members pseudo. Rejected in favor of one-pseudo-rest-real (D2).

**G. Runtime feature flag + shadow diff harness + automated flip gate.** v1/v2/v3 iteration path. Turned out to be over-engineered for bytebase's shipping practice. Replaced by test matrix + standard PR review + fix-forward.

**H. omni prerequisite PR adding structured `ErrorIdent`.** Useful only for automated flip gates. Dropped with the flip gate. Can be proposed independently later if classifier accuracy for operational diagnosis warrants it.

## Hard contracts

**C1. View/matview body must be `*ast.SelectStmt`, not `*ast.RawStmt`.** omni's `DefineView` / `ExecCreateTableAs` type-assert directly.

**C2. Partitioned tables pass relkind `'r'` to `DefineRelation`.** omni flips to `'p'` internally when `Partspec` is set.

**C3. Function overloads install sequentially.** Each overload is one `CreateFunctionStmt`. New signatures append; duplicate signatures are rejected.

**C4. `typeNameFromString` input must be from trusted metadata.**

**C5. `extractUserTypeRefs` must be sound, not precise.** False negatives acceptable (pseudo catches). False positives are not (corrupt dep graph).

**C6. Pseudo install must never depend on user objects.** Every pseudo form is built from built-in types only.

**C7. Loader must hold no long-lived state.** Fresh `catalog.New()` per extractor.

**C8. All 17 type string shapes in §Type string grammar have golden tests.** Including the PG internal array form `_name` and system-schema qualified forms.

**C9. Classifier is shared.** `classifyAnalyzeError` is one function called from three fallback sites. Never inline-classify at a call site.

**C10. Tarjan SCC intra-SCC ordering is lexicographic on `(schema, name)`.** Determinism required for test reproducibility.

## Complexity honesty

| Dimension | v5 | **E3** |
|---|---|---|
| Per-query install cost | O(closure) | O(schema), measured 1.5-2.3× faster than current |
| Loader code | `ExtractCatalogReferences` + `expandClosure` + `installClosure` + retry | `collectObjects` + `tarjanSCC` + `topoSort` + `installReal`/`installPseudo` |
| Implementation LOC estimate | ~1000 | ~1000 (loader + classifier + search path unification + type-string extractor) |
| Fidelity regression risk | Medium (missed refs) | Low (only composite `(col).field`) |
| PL/pgSQL coupling | Fallback not narrowed | Classifier at 3 sites, observability only |
| Metadata proto changes | None | None |
| omni PR required | None | None |
| Runtime machinery | Fallback flag, shadow mode | None |
| Rollout | Multi-week soak | Merge when tests green |

## Appendix: metadata audit

Performed 2026-04-15 via exploration of `proto/store/store/database.proto` and `backend/plugin/db/pg/sync.go`.

| Field | Exists? | Location | Notes |
|---|---|---|---|
| `TableMetadata.columns[i].type` | ✅ | `database.proto:510` | String; user types as `"schema.name"` per `sync.go:1019` |
| `ViewMetadata.dependency_columns` | ✅ | `database.proto:587` | `{schema, table, column}` per view col |
| `MaterializedViewMetadata.dependency_columns` | ✅ | `database.proto:625` | Same shape |
| `EnumTypeMetadata` | ✅ | `database.proto:124-134` | `{name, values[]}` |
| `FunctionMetadata` | ✅ | `database.proto:645-672` | Per signature |
| `TableMetadata.partitions` | ✅ (parent→children) | `database.proto:360-361` | Reverse direction from topo needs; irrelevant |
| `TableMetadata.inherits_parent` | ❌ | — | Not needed: sync flattens |
| `CompositeTypeMetadata` | ❌ | — | Pseudo composite with `_broken` field |
| `DomainMetadata` | ❌ | — | Pseudo domain over text |
| `RangeTypeMetadata` | ❌ | — | Pseudo range subtype=text |

Of the four missing metadata kinds, only `CompositeTypeMetadata` causes query span fidelity loss (`(col).field` access), and it is fully recoverable via an independent metadata PR.
