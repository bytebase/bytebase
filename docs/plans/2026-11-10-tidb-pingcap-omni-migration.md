# TiDB: Remove the `github.com/pingcap/tidb` Dependency (migrate to omni)

**Goal:** Drop Bytebase's dependency on `github.com/pingcap/tidb` by moving every
TiDB importer onto `github.com/bytebase/omni`. omni's TiDB parser is at
bidirectional grammar parity with TiDB v8.5.5 and is already the sole TiDB
statement parser (`parseTiDBStatements`, registered in
[`tidb.go`](../../backend/plugin/parser/tidb/tidb.go)). What remains is to
rewrite the AST **consumers** that still walk pingcap's `ast` package so the
native parser, the `AsPingCapAST` bridge, and the pingcap go.mod entry can all
be deleted.

**Template:** The MSSQL, MySQL, PostgreSQL, and Oracle query-span migrations
(`docs/plans/2026-0{3,4,5}-*-query-span-omni-migration.md`) are the model. Those
moved from ANTLR/pg_query to omni. TiDB is easier in one respect — it migrates
from one object-AST (pingcap) to another (omni) — and harder in another: omni's
TiDB AST is a **Postgres-shaped rewrite**, so node shapes differ (see Reshape
notes below). It is not a mechanical identifier rename.

## Current architecture (post-dispatcher-retirement, PR #21936)

- `parseTiDBStatements` ([`omni.go`](../../backend/plugin/parser/tidb/omni.go))
  is the registered `ParseStatementsFunc`. omni is the only parser; there is no
  pingcap fallback for new statement parsing.
- An `*OmniAST` wraps each omni node. It implements `PingCapASTProvider`:
  `AsPingCapAST()` lazily re-parses the statement text with the native pingcap
  parser to serve **un-migrated consumers**. This bridge, plus `ParseTiDB` /
  `ParseTiDBForSyntaxCheck` and the whole pingcap line-tracking apparatus
  (`applyTiDBLineTracking`, `SetLineForMySQLCreateTableStmt`, the
  [`tokenizer`](../../backend/plugin/parser/tokenizer/tokenizer.go) methods that
  take `*tidbast.CreateTableStmt`), exist **only** to keep those consumers
  working. They are deletion targets once the consumers migrate, not migration
  targets themselves — omni nodes already carry their own positions.

## Importer inventory

Authoritative list: `grep -rl github.com/pingcap/tidb backend/`. Five
`db/*/sync.go` hits (`mysql`, `doris`, `oceanbase`, `starrocks`, `tidb`) are a
comment URL to a TiDB issue, **not imports** — ignore them. The real importers:

| Group | Files | pingcap surface | omni target | Status / notes |
|---|---|---|---|---|
| **db/tidb/stats** | `stats.go` | `DeleteStmt`, `UpdateStmt`, `Join`, `TableSource`, `ResultSetNode` (DML target counting) | `omni/tidb/ast` + `ParseTiDBOmni` | **DONE** (this PR). Fully self-contained; parsed with omni, walks `DeleteStmt.Tables`/`UpdateStmt.Tables`/`JoinClause`. |
| **advisor/tidb** | `utils.go`, `advisor_builtin_prior_backup_check.go` | `utils.go`: `ast.StmtNode` return + `ast.Rand`; `prior_backup_check.go`: `AsPingCapAST()` + `DDLNode` | `omni/tidb/ast` | **In progress.** `utils.go` is already ~90% on `omniast` (`OmniStmt` + dozens of `omni*` helpers). Remaining: `getTiDBNodes` returns pingcap `[]ast.StmtNode` for still-un-migrated advisors; finishes when the last advisor stops using it. `prior_backup_check` uses the bridge for authoritative DDL detection. |
| **parser/tidb — query-span** | `query_span_extractor.go` (+ helpers `util.go`, `query.go`, `limit.go`, `query_type.go`, `statement_type.go`, `resource_change.go`, `ast.go`) | 40+ expression/clause node types; visitor `Accept`/`Enter`/`Leave`; `format`/`Restore`; `parser.New` | `omni/tidb/ast` (`Walk`/`Visit`, `NodeToString`) + `ParseTiDBOmni` | **Not started. No omni-side gap blocks it** (see below). The largest unit. `resource_change.go` is already dual (imports both). |
| **parser/tidb — core/bridge** | `tidb.go`, `ast.go` | `tidbparser`, pingcap `ast`, `mysql`, `parser_driver`; the `AST`/`PingCapASTProvider` bridge | — (delete) | Deleted last, once no consumer calls `AsPingCapAST`/`ParseTiDB`. |
| **tokenizer** | `tokenizer.go` | `*tidbast.CreateTableStmt`, `Constraint*` consts | — (delete) | Only caller is `tidb.go`'s pingcap line-tracking. Dies with the bridge; not a migration target. |
| **schema/tidb** | `walk_through.go`, `get_database_metadata.go`, `generate_migration_test.go` | `parser.New`, `ast.*`, `types.FieldType`, `mysql.Type*`/sql-mode | `omni/tidb/catalog` (`metadata_load.go`) + `omni/tidb/ast` + `omni/metadata` | **Blocked on omni catalog parity**, being finished in a sibling omni session. May need a fresh omni pin re-bump once that lands. |

## OMNI-SIDE GAPS

**None block the migration.** A full read-coverage audit of the query-span
extractor against `omni/tidb/ast` found that omni expresses every table, column,
derived table, set operation, CTE, join, and DML target the extractor consumes.
All differences are structural **reshapes**, not absent data. The three
closest-to-a-gap items are all handleable on the Bytebase side with no omni PR:

1. **`DMLNode` / `DDLNode` marker interfaces** (`query_type.go`): omni ships no
   DML/DDL classification interface. Replace the `case tidbast.DMLNode:` /
   `case tidbast.DDLNode:` fall-throughs with explicit concrete-type
   enumeration (or a small local classifier).
2. **`Node.Text()`** (`util.go`, auto-derived column name fallback): omni nodes
   carry byte-offset `Loc` + a `NodeToString` deparser instead of retained
   source text. Slice the statement by `Loc`, or call `NodeToString`.
3. **`format` package / `Restore` / `parser_driver.ValueExpr`** (`limit.go`):
   not in omni by design; `ast.NodeToString` + `*ast.IntLit` replace them.

An optional omni ergonomics PR (a `DMLNode`/`DDLNode`-style classifier, a
source-span convenience) could reduce boilerplate but is not on the critical
path.

## Key reshapes (how the walk changes)

- **Identifiers:** pingcap `model.CIStr` (`.O` original / `.L` lowercased) → plain
  `string`. Drop `.O`; replace `.L` with explicit `strings.ToLower`.
- **Traversal:** pingcap mutating `Accept`/`Enter`/`Leave` → omni non-mutating
  `ast.Walk`/`ast.Inspect` with `Visitor.Visit(node) Visitor` (pre-order on
  `Visit(node)`, post-order on `Visit(nil)`). CTE scope push/pop moves into that
  pairing.
- **FROM/joins:** `SelectStmt.From.TableRefs` (one `*Join` root, leaves wrapped in
  `*TableSource`) → `SelectStmt.From []TableExpr` of `*TableRef` | `*JoinClause`
  | `*SubqueryExpr`; alias moves onto `TableRef.Alias`/`SubqueryExpr.Alias`;
  `NaturalJoin`/`Using` → `JoinClause.Type`/`JoinClause.Condition`.
- **Set operations:** distinct `*SetOprStmt`/`*SetOprSelectList` → folded into
  `SelectStmt` (`SetOp != SetOpNone`, `Left`/`Right` binary tree). Use
  `LeftmostQueryLeaf`.
- **Aggregates/window funcs:** `AggregateFuncExpr`/`WindowFuncExpr` → generic
  `FuncCallExpr` (`.Quantifier`/`.Star`/`.Over`).
- **DML targets:** `IsMultiTable`/`Tables`/`TableRefs` → `DeleteStmt.Tables` +
  `DeleteStmt.Using` / `UpdateStmt.Tables` as `[]TableExpr`; multi-table-ness is
  inferred from slice length / `Using` presence, not a bool.

## Recommended order (smallest / most independent first)

1. **`db/tidb/stats.go`** — DONE in this PR. Self-contained, independent of the
   bridge and of catalog parity; proves the `ParseTiDBOmni` + omni-AST-walk
   pattern. Verified by the pure `TestCountDMLTargets` (9 cases) +
   `TestGetAffectedRows*`.
2. **Finish `advisor/tidb`** — migrate the remaining advisors off
   `getTiDBNodes`/`AsPingCapAST`, then delete the pingcap return path. Already
   mostly done; unblocks removing the bridge's advisor consumers.
3. **`parser/tidb` query-span cluster** — the main effort. Build a
   package-internal omni extractor mirroring the pingcap one, prove it with a
   full-corpus golden harness against the existing
   `test-data/query-span/` fixtures (zero diffs), then cut over. No omni
   prerequisite. Independent of catalog parity. Migrate the helper files
   (`query_type.go`, `statement_type.go`, `resource_change.go`, `limit.go`,
   `util.go`, `query.go`) in the same effort.
4. **`schema/tidb`** — walk-through + `get_database_metadata`. **Blocked** on
   omni's TiDB catalog/review reaching parity (sibling omni session). Re-bump
   the omni pin when that lands, then port onto `omni/tidb/catalog` +
   `omni/metadata`, mirroring `schema/mysql`.
5. **Delete the bridge** — once nothing calls `AsPingCapAST`/`ParseTiDB`/
   `ParseTiDBForSyntaxCheck`, remove `tidb.go`'s pingcap parser,
   `applyTiDBLineTracking`, `SetLineForMySQLCreateTableStmt`, the `tokenizer`
   pingcap methods, the `AST`/`PingCapASTProvider` bridge, and finally the
   `github.com/pingcap/tidb` entries from `go.mod` + `go mod tidy`.

**Do not remove `pingcap/tidb` from `go.mod` until every importer above is
migrated.** Steps 1–3 can proceed now; step 4 waits on omni catalog parity.
