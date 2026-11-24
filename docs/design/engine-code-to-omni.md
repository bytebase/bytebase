# Engine code moves to omni

2026-11-12 · Implemented (updated 2026-11-24)

Defines which code under `backend/plugin` belongs in omni, the contracts that keep the two
repositories apart, and the boundary established across all four migration phases. Originally
proposed on `main` at `fb411c9453` (~195k non-test Go lines under `backend/plugin`); "today" in
this document reflects the completed migration (omni pin `8dc5a3627452`, ~24.5k non-test Go lines
across 132 files).

## Problem

Omni (`github.com/bytebase/omni`) is Bytebase's engine repository: per engine, a recursive-descent
parser, a typed AST, analysis and completion, a catalog that simulates and diffs DDL, review
rules, and wire-protocol drivers, verified by scenario corpora and by oracle tests against real
engines in containers. Bytebase consumes it through a module pin.

Historically, the boundary between the two repositories was accidental rather than principled.
Omni owned parsing and the metadata proto, while Bytebase still carried roughly 195k non-test Go
lines of engine-specific code under `backend/plugin` (advisor 58k, parser 63k, schema 35k, drivers
38k):

- **Split changes.** One logical engine change — a parser fix plus the review rule or query-span
  change that depends on it — required two PRs in two repositories with a pin bump between them.
- **Duplicated machinery.** Omni's catalogs carried `metadata_load.go`, `migration_*.go`, `sdl.go`,
  and full diff engines while `backend/plugin/schema` carried parallel hand-written deparse and
  differ implementations that predated them.
- **Tests in the wrong repository.** The root `AGENTS.md` states that "engine dialect and DDL
  fidelity tests belong in omni", yet query-span fixtures, schema golden files, and driver behavior
  lived in Bytebase, outside omni's corpus and oracle gates.
- **Every new engine was built twice.** Omni built parser → catalog → completion; Bytebase then
  rebuilt `parser/<engine>`, `advisor/<engine>`, `schema/<engine>`, and `db/<engine>` before the
  engine reached users.
- **The least-tested engine surface was the driver.** System-catalog sync queries per engine and
  version and result type mapping are pure engine behavior, yet sitting outside omni's container
  matrix let driver-bump regressions (empty type names, decimal metadata shifts, flipped security
  defaults) slip past Bytebase CI.

All four phases of this migration have now landed, shrinking `backend/plugin/{advisor,parser,schema,db}`
from ~195k to ~24.5k non-test Go lines of thin adapters and product glue. This document records
the boundary as implemented and the conventions that keep it clean.

## Where the boundary stands today

Non-test Go footprint under `backend/plugin`:

- `backend/plugin/advisor`: 23 files, 5,866 lines (down from ~58k)
- `backend/plugin/parser`: 52 files, 7,828 lines (down from ~63k; includes `parser/base` at
  3,354 lines across 19 files plus 22 engine adapter packages and `parser/standard`)
- `backend/plugin/schema`: 20 files, 3,754 lines (down from ~35k; includes
  `schema/internal/omnicatalog` at 1,277 lines across 6 files plus 10 engine adapter packages)
- `backend/plugin/db`: 37 files, 7,059 lines (down from ~38k; includes `db/util` at 2,331 lines
  across 9 files, `db/transaction` at 35 lines, root files, and 23 engine adapter directories)

**Parsing and AST — in omni.** Every production parse goes through omni (25 `omni/<engine>/parser`
packages plus `omni/cockroachdb/analysis`, which encapsulates `cockroachdb-parser` inside omni as
a tracked follow-up to replace with a native parser). ANTLR (`github.com/antlr4-go/antlr/v4`) is
`// indirect` in `go.mod` and kept out of Bytebase by guard tests
(`backend/plugin/parser/tsql/backup_test.go` and
`backend/plugin/parser/{pg,plsql,redshift,tsql}/completion_test.go`).

**Metadata — in omni.** `omni/metadata` (generated from omni's `proto/omni/metadata/database.proto`)
is the database schema snapshot type, imported across ~183 files in `backend`; the store proto no
longer defines it. Bytebase persists it and wraps it in `backend/store/model` for indexed lookup.

**Walk-through and simulation — in omni.** `backend/plugin/schema/internal/omnicatalog/walk_through.go`
and the per-engine schema adapters drive omni catalogs (`New`, `Exec`, `LoadMetadata`) using
`omni/analysis.WalkThroughStatement`; the advisor's builtin walk-through check and SQL Review V2's
`WALK_THROUGH` rule both run on them.

**SQL review — both generations on omni, separate contracts.**

- V1: All 10 engines' V1 review judgment lives in `omni/<engine>/v1review` (`googlesql`,
  `mariadb`, `mssql`, `mysql`, `oceanbase`, `oracle`, `pg`, `redshift`, `snowflake`, `tidb`) behind
  the shared `omni/v1review` contract. `backend/plugin/advisor` is a thin product shell: the
  `storepb.Engine` × `storepb.SQLReviewRule_Type` (105 rule types) registry, `V1Engine`
  (`v1review_engine.go`) and MySQL-family wiring (`v1review_mysql_family.go`), payload decoding,
  live-connection probing (`prober.go`), PostgreSQL SDL style/integrity wrappers
  (`advisor/pg/sdl_check.go`, `advisor/pg/sdl_integrity_check.go` delegating to
  `omni/pg/analysis`), advice rendering, and 362 YAML parity fixtures under
  `backend/plugin/advisor/*/test/*.yaml`. Callers remain
  `backend/runner/plancheck/statement_advise_executor.go` and
  `backend/api/v1/release_service_check.go`.
- V2: `omni/review` is the declared contract between Bytebase SQL Review V2 and omni's 8 engine
  review packages (`mysql/review`, `pg/review`, `mariadb/review`, `oceanbase/review`,
  `tidb/review`, `mssql/review`, `oracle/review`, `snowflake/review`) covering the 11
  `ReviewRuleType` rules. `backend/runner/reviewrun` resolves policy and store facts and invokes
  one `Review` per `(spec, engine)`.

The two generations stay completely separate, in omni as in Bytebase: `omni/v1review` neither
imports `omni/review` nor shares a type with it, and nothing V1 ported joins the V2 standard rule
set.

**Query analysis and completion — in omni.** All 22 engine parser packages under
`backend/plugin/parser/` (plus `parser/standard` for ClickHouse, Databricks, and Hive) delegate
query span, statement splitting, changed resources, statement types and ranges, query
classification, completion, diagnostics, explain generation, result-limit rewriting, DML→SELECT
backup transforms, and restore SQL to `omni/<engine>/analysis` (23 packages) and
`omni/<engine>/completion` (11 packages). The shared `omni/analysis` contract owns `QueryType`,
`StatementType`, `CandidateType`, `WalkThroughStatement`, `ResolvedSpan`, `ColumnResource`,
`ChangedResource`, `Statement`, and analysis error types. `backend/plugin/parser/base` keeps the
unified `Engine` capability registry (`base.Register`) and translates omni contract values into
product/LSP types;
`parser/tokenizer`, `split_test_runner.go`, regex `ValidateSQLForEditor` implementations, and
duplicated analysis/completion corpora have been deleted.

**Schema conversion — in omni, with explicit legacy shims.** All 10 engine packages under
`backend/plugin/schema/` delegate metadata↔DDL conversion, metadata diff, SDL migration, and
walk-through simulation to `omni/<engine>/catalog` (and `omni/{clickhouse,trino}/deparse`) through
`backend/plugin/schema/internal/omnicatalog`. The old Bytebase-local metadata differ (`differ.go`)
and hand-written per-engine `generate_migration.go` / deparse implementations have been deleted,
and MariaDB now runs on its own `omni/mariadb/catalog` fork rather than simulating on MySQL.
Callers remain `backend/api/v1/database_service.go` and the drivers' dump paths. Golden YAML
corpora live in `omni/<engine>/catalog/testdata/`; Bytebase retains focused seam tests and four
explicit, documented legacy compatibility shims:

- `backend/plugin/schema/pg/legacy_normalize.go`: repairs historical persisted PostgreSQL
  `IndexMetadata.Expressions` rows until a one-time metadata migrator removes the need.
- `backend/plugin/schema/mssql/mssql.go`: `stripUnrenderedFacts`, `restoreLegacyModuleText`, and
  `reconcileSkipDump` (handling `SkipDump`) preserve pre-cutover SQL Server dump/diff output until
  each fix ships with its own release note.
- `backend/plugin/schema/oracle/oracle.go`: `stripUnrenderedFacts` keeps identity clauses and index
  visibility out of Oracle dumps and migrations until enabled.
- `backend/plugin/schema/mariadb/mariadb.go`: `clearImplicitTableCharsets` preserves
  omission of unspecified table charsets/collations so migrations stay valid on older MariaDB
  releases.

**Drivers — in omni; product retains network, auth, and lifecycle glue.** All 23 driver packages
under `backend/plugin/db/` (covering all 24 `storepb.Engine` values — `db/mysql` registers both
`MYSQL` and `MARIADB`) delegate wire protocol, metadata sync, query, execute, explain, and dump to
`omni/<engine>/driver` (24 engine driver packages) behind the shared `omni/driver` contract and
`backend/plugin/db/util` (`OmniAdapter`, result/config/execute-log conversion).
Bytebase keeps the product seam: `db.ConnectionConfig` / `*storepb.DataSource` translation into
`omni/driver.Config`, SSH tunnels (`backend/plugin/db/util/ssh.go`), TLS material resolution
(`backend/plugin/db/util/ssl.go`), AWS IAM / RDS auth (`backend/plugin/db/util/aws.go`), Google
Cloud SQL dialers (`cloudsqlconn`), secret resolution (`backend/component/secret`),
`v1pb.QueryResult` and task-run log conversion (`backend/plugin/db/util/omnidriver.go`), and
`db.Prober` / `db.RowEstimator` wiring.

Every target-engine client SDK and parser dependency that belonged only to `backend/plugin` is now
`// indirect` in `go.mod` — including `go-ora/v2`, `go-mssqldb`, `gosnowflake/v2`,
`clickhouse-go/v2`, `mongo-driver/v2`, `go-redis/v9`, `azcosmos` (whose fork `replace` was also
removed), `cassandra-gocql-driver/v2`, `gohive/v2`, `trino-go-client`,
`cloud.google.com/go/bigquery`, `cloud.google.com/go/spanner`, `databricks-sdk-go`,
`aws-sdk-go-v2/service/dynamodb`, `antlr4-go/antlr/v4`, and `cockroachdb-parser`. Unused engine
testcontainers were pruned from `backend/common/testcontainer/`, leaving only PostgreSQL, TLS
PostgreSQL, MySQL, and SSH test harnesses.

Direct engine-related dependencies that intentionally remain in `go.mod`:

- `github.com/jackc/pgx/v5` (and `github.com/jackc/pgconn` in `backend/bin/server/cmd/root.go`):
  Bytebase's own PostgreSQL metadata store (`backend/store`, `backend/runner/notifylistener`,
  `backend/resources/postgres`, tests), plus `*pgconn.PgError` decoding in
  `backend/plugin/db/pg/pg.go`.
- `github.com/lib/pq`: Bytebase's metadata migrator (`backend/migrator`) and SQL identifier/literal
  quoting in `backend/component/export/sql.go`.
- `github.com/go-sql-driver/mysql` (with the `gh-ost` replace): `gh-ost` online-migration
  supervision outside `backend/plugin/db` (`backend/component/ghost`,
  `backend/runner/plancheck/ghost_sync_executor.go`,
  `backend/runner/taskrun/database_migrate_executor.go`).

**Engine coverage.** Omni now provides 26 engine trees covering all 24 `storepb.Engine` values
(including dedicated trees for `clickhouse`, `databricks`, `hive`, `redis`, `bigquery`, `spanner`,
`cockroachdb`, and `dynamodb`, alongside shared language trees `googlesql` and `partiql`), plus 5
shared contract packages (`omni/metadata`, `omni/review`, `omni/v1review`, `omni/analysis`,
`omni/driver`).

## Goals

1. Omni is the single home for code whose correctness is decided by an engine's grammar,
   semantics, or wire behavior: parsing, analysis (query span, classification, changed resources,
   splitting), review judgment, DDL simulation, diff, migration and definition generation,
   completion context, and the driver conversation — connect, sync, query, execute, dump.
2. Bytebase keeps everything decided by product configuration and stored state: policy
   resolution, stores, credential and network-path resolution (secret managers, IAM identities,
   tunnels, TLS material), connection lifecycle and scheduling, API conversion, engine dispatch.
   Bytebase speaks to target databases only through omni.
3. One contract style across the boundary, generalized from `omni/review`: omni never imports
   Bytebase, name-keyed identifiers tolerant of version skew, facts passed in — judgment returned.
4. Every move is an independent, revertible PR pair with parity evidence, following the playbook
   that retired ANTLR (PG took 21 PRs across 5 packages; TiDB ended with the bridge deleted in
   #21949 and the fork pin dropped in #21979).

## Non-goals

- **Bytebase's own store.** `backend/store` keeps speaking PostgreSQL to Bytebase's own metadata
  database. "Nothing to do with databases" means target databases; the product's state store is
  product.
- **Online-migration execution.** gh-ost supervision (`backend/component/ghost`) stays with
  rollout execution; its eligibility judgment is already omni's `ONLINE_MIGRATION` rule. Whether
  gh-ost itself becomes an omni-owned capability is a later question, out of scope here.
- **Changing product behavior.** Review findings, SDL semantics, editor features keep their
  behavior; this plan relocates implementations.
- **SQL Review V2, and any V1→V2 migration.** This plan moves V1's judgment to omni and leaves
  V2's rule set, policy, and contract as they are. Whether V1 rule types ever map onto V2, and
  when plancheck and the release check would cut over, is the review V2 track's decision.
- **Merging the repositories** (see Alternatives).
- Non-engine plugins (`ai`, `aireview`, `webhook`, `idp`, `mailer`, `sharepoint`, `stripe`) are
  untouched.

## The boundary rule

A piece of code moves to omni when a database engine's documented or observed behavior decides
whether it is correct — that is, when its tests would be stronger under omni's corpus, container,
and oracle gates than under Bytebase's product tests. It stays in Bytebase when a product
decision or stored state decides.

The contract style is the one `omni/review` already states:

- **No Bytebase imports, ever.** Omni's shared vocabulary is its own: `omni/metadata` for schema
  snapshots, contract packages for everything else. `storepb` types (`Advice`, `Engine`,
  `ChangedResourceTable`, rule payloads) stop at the Bytebase adapter.
- **Name-keyed across versions.** Identifiers cross the boundary as strings equal to proto enum
  names (as `review.Rule` does), so the two sides tolerate being on different versions: a name one
  side does not know is ignored or dropped, never an error.
- **Facts in, judgment out.** Bytebase resolves whatever needs the store or a connection (synced
  metadata, whether a backup database exists, EXPLAIN output, affected-row estimates, object
  ownership) and passes plain values; omni derives findings from SQL plus those values. A syntax
  error is a finding, not an error; an error means the call could not be evaluated.
- **Omni speaks the protocol; Bytebase supplies the path.** The driver boundary splits the same
  way: Bytebase resolves who may reach what over which path — secrets from secret managers, IAM
  tokens, SSH tunnels, TLS material — and passes the result as plain values and funcs (a resolved
  credential, a `*tls.Config` callback, a `Dial` func, a `TokenProvider`); omni owns everything
  after the dial: session setup, sync queries, statement execution, result decoding. Engines whose
  client is a cloud SDK (BigQuery, Spanner, DynamoDB, Cosmos DB, Databricks) take credential
  material as values and keep their SDK in omni; injected dialers keep the IAM and tunnel
  machinery out of omni for TCP engines.
- **Engine dispatch stays in Bytebase.** The `storepb.Engine` enum, the registries in
  `parser/base` / `schema` / `advisor` / `db`, and per-layer aliasing (e.g., `cockroachdb` sharing
  `schema/pg`, `bigquery`/`spanner` sharing `googlesql/analysis`, `dynamodb` using
  `partiql/analysis`) are the Bytebase-side seam. Omni stays package-per-engine with no engine
  enum.

## Target state under backend/plugin

| before (`fb411c9453`, ~195k lines) | current state (~24.5k lines) |
|---|---|
| `advisor/<engine>` (~380 rule files, 58k lines) | thin `V1Engine` adapters (10 engines, 5.9k lines total); judgment lives in `omni/<engine>/v1review` behind `omni/v1review` |
| `advisor` root (registry, `sql_review.go`) | stays as V1's product shell (`v1review_engine.go`, `v1review_mysql_family.go`, `prober.go`); deleted only if the review V2 track retires V1 |
| `parser/<engine>` capability implementations (63k lines) | thin adapters (22 engine packages + `parser/standard`, 7.8k lines across 52 files including `parser/base`) delegating to `omni/<engine>/analysis` and `omni/<engine>/completion` |
| `parser/base` | stays: unified `Engine` capability registry (`base.Register`), generic `NodeAST[N]` wrapper, dispatch, product/LSP conversions; shared semantic types (`QueryType`, `StatementType`, `CandidateType`, `WalkThroughStatement`, `ResolvedSpan`, etc.) live in `omni/analysis` |
| `schema/<engine>` deparse/differ/migration (35k lines) | thin adapters (10 engines, 3.8k lines total including `schema/internal/omnicatalog`) delegating to `omni/<engine>/catalog` and `omni/{clickhouse,trino}/deparse` via `schema.Register`, plus explicit legacy compatibility shims |
| `schema` comparers, `differ.go` | deleted; replaced by omni catalog diff via `schema/internal/omnicatalog` |
| `db/<engine>`, `db/transaction`, `db/util` engine helpers (38k lines) | thin adapters (23 engine directories, 7.1k lines total including `db/util`) delegating protocol, sync, query, execute, explain, and dump to `omni/<engine>/driver` via `db/util.OmniAdapter` |
| `db` root, `db/util/aws.go`, `db/util/ssh.go`, `db/util/ssl.go` | stay: registry dispatch, `storepb.DataSource` → `omni/driver.Config` translation, IAM/RDS auth, Cloud SQL dialers, SSH tunnels, TLS material resolution, proto result and task-run log conversion |
| YAML corpora, golden fixtures for the above | analysis, completion, schema, and driver corpora live in omni under its gates; advisor YAML parity fixtures (362 files) remain in `backend/plugin/advisor/*/test/*.yaml` |

Shared omni contract surface (all five packages now in production):

- `omni/metadata`: protobuf schema snapshot types shared across sync, schema conversion, analysis,
  and review.
- `omni/review`: the SQL Review V2 contract across 8 engine review packages.
- `omni/v1review` (separate from `omni/review`): the V1 review contract. Each rule is a function
  per engine, such as `<engine>/v1review.NamingTable(stmts, cfg)`, taking parsed statement nodes
  and plain configs built from the rule's `SQLReviewRule` payload, returning judgment-only
  `v1review.Finding` values (condition code, judged name, byte range). Bytebase keeps the
  registry, payloads, severity levels, live probes, and advice wording.
- `omni/analysis`: the shared query-analysis contract — `QueryType`, `StatementType`,
  `CandidateType`, `WalkThroughStatement`, `ResolvedSpan`, `ColumnResource`, `ChangedResource`,
  `Statement`, and analysis error types — implemented by 23 `<engine>/analysis` packages and 11
  `<engine>/completion` packages, while Bytebase resolves live metadata into product `Candidate`
  values.
- `omni/driver`: the driver contract — `Config` (endpoint, resolved credentials, optional
  `TokenProvider`, `TLS` callback, `Dial` func, `Params`, `EngineVersion`), `Driver`, `Syncer`,
  `Dumper`, `Querier`, `Session`, `Executor`, `ExecuteLogger`, `Instance`, and `Result`/`Value`
  types — implemented by 24 `<engine>/driver` packages. Bytebase converts `omni/driver.Result` to
  `v1pb.QueryResult` and streams `ExecuteLogger` callbacks into task-run logs in
  `backend/plugin/db/util/omnidriver.go`.

## Migration plan

Ordered by value per unit of risk. Phases overlapped; within each, an engine was one unit of work,
and fixes ported to fork engines landed alongside their parents (`fix(mysql,mariadb): ...`),
matching omni's own fork rule.

### Phase 1 — V1 review rules (advisor → omni `<engine>/v1review`)

**Status: Complete.**

All 10 engines' V1 review rules (`bigquery`/`googlesql`, `mariadb`, `mssql`, `mysql`, `oceanbase`,
`oracle`, `pg`, `redshift`, `snowflake`, `tidb`) now delegate judgment to `omni/<engine>/v1review`
behind `omni/v1review`, shrinking `backend/plugin/advisor` from ~58k to 5.9k non-test Go lines
across 23 files:

- Shared adapter plumbing lives in `backend/plugin/advisor/v1review_engine.go` (`V1Engine`) and
  `backend/plugin/advisor/v1review_mysql_family.go`, with live-connection fact gathering behind
  `backend/plugin/advisor/prober.go` (`db.Prober` / `db.RowEstimator`) so connections never cross
  the `omni/v1review` contract.
- Each engine keeps a thin `rules.go` registration file (plus PostgreSQL's live-rule and SDL
  rendering wrappers in `advisor/pg/{live_rules,sdl_check,sdl_integrity_check}.go`), while the 362
  YAML parity fixtures remain under `backend/plugin/advisor/*/test/*.yaml` as the adapter parity
  gate alongside omni's `<engine>/v1review` judgment tests.

### Phase 2 — query analysis (parser/`<engine>` → omni `<engine>/analysis`)

**Status: Complete.**

All 22 engine parser packages under `backend/plugin/parser/` (plus `parser/standard` for
ClickHouse, Databricks, and Hive) now delegate parsing, analysis, and completion to
`omni/<engine>/analysis` and `omni/<engine>/completion`, shrinking `backend/plugin/parser` from
~63k to 7.8k non-test Go lines across 52 files:

- `omni/analysis` owns the shared contract types (`QueryType`, `StatementType`, `CandidateType`,
  `WalkThroughStatement`, `ResolvedSpan`, `ColumnResource`, `ChangedResource`, `Statement`, error
  types), and `backend/plugin/parser/base` retains the unified `Engine` registry (`base.Register`)
  and product/LSP conversions so consumers (`backend/api/v1`, lsp, plancheck, export, taskrun,
  sheet) keep their existing call sites.
- `backend/plugin/parser/tokenizer`, `split_test_runner.go`, regex `ValidateSQLForEditor`
  implementations, and duplicated analysis/completion test corpora were deleted.
- Remaining omni-side follow-up: `omni/cockroachdb/analysis` still encapsulates
  `github.com/cockroachdb/cockroachdb-parser` inside omni (`// indirect` in Bytebase's `go.mod`).

### Phase 3 — schema conversion (schema/`<engine>` → omni `<engine>/catalog`)

**Status: Complete (with explicit legacy compatibility shims).**

All 10 engine packages under `backend/plugin/schema/` now delegate metadata↔DDL conversion,
metadata diff, SDL migration, and walk-through simulation to `omni/<engine>/catalog` (and
`omni/{clickhouse,trino}/deparse`) via `backend/plugin/schema/internal/omnicatalog`, shrinking
`backend/plugin/schema` from ~35k to 3.8k non-test Go lines:

- The Bytebase-local metadata differ (`differ.go`) and hand-written deparse/migration generators
  were deleted, and MariaDB moved onto its own `omni/mariadb/catalog` fork instead of simulating on
  MySQL.
- Golden YAML corpora moved to `omni/<engine>/catalog/testdata/`; `backend/plugin/schema/AGENTS.md`
  documents the seam-test boundary and the remaining `record` process for local fixtures.
- Intentional Bytebase-only compatibility shims remain where changing dump/migration output
  requires a separate product migration or release note:
  `backend/plugin/schema/pg/legacy_normalize.go`, `backend/plugin/schema/mssql/mssql.go`
  (`stripUnrenderedFacts`, `restoreLegacyModuleText`, `reconcileSkipDump`),
  `backend/plugin/schema/oracle/oracle.go` (`stripUnrenderedFacts`), and
  `backend/plugin/schema/mariadb/mariadb.go` (`clearImplicitTableCharsets`).

### Phase 4 — drivers (db/`<engine>` → omni `<engine>/driver`)

**Status: Complete.**

All 23 driver packages under `backend/plugin/db/` (registering all 24 `storepb.Engine` values;
`db/mysql` covers both `MYSQL` and `MARIADB`) now delegate wire protocol, metadata sync, query,
execute, explain, and dump to `omni/<engine>/driver` behind `omni/driver`, shrinking
`backend/plugin/db` from ~38k to 7.1k non-test Go lines:

- `backend/plugin/db/util` (`OmniAdapter`, `omnidriver.go`, `config.go`) owns the
  shared lifecycle, held-session query serialization, `v1pb.QueryResult` conversion, and
  `ExecuteLogger` task-run log plumbing.
- Per-engine directories remain as thin (~100–480 line) product adapters that translate
  `db.ConnectionConfig` / `*storepb.DataSource` into `omni/driver.Config`, wire engine-specific
  credential/network paths (SSH tunnels, TLS material, AWS IAM / RDS auth, AWS SigV4 signing, Cloud
  SQL dialers, Entra ID / OAuth token providers), and expose `db.Prober` / `db.RowEstimator` where
  V1 live review uses them.
- Target-engine driver SDK dependencies (including `opensearch-go/v5` and the old `azcosmos`
  replace) were removed or moved to `// indirect` in `go.mod`, and unused engine testcontainers were
  pruned from `backend/common/testcontainer/`. Bytebase keeps direct driver dependencies only
  outside the target-driver boundary (metadata store `pgx/v5`, migrator/export `lib/pq`, and
  `gh-ost` `go-sql-driver/mysql`).

### The playbook for every move

- **Bridge before burning.** Both implementations stay callable during a port. Parity is proved by
  a differential corpus — Bytebase's existing fixtures replayed through the omni implementation —
  and every flip is classified against the engine's real verdict, because a tightening that loses
  valid behavior looks identical to one that fixed a false accept.
- **One capability × one engine family per PR pair** (omni PR, then a Bytebase PR that bumps the
  pin and deletes the old path). Each is independently revertible.
- **Pin bumps are audited as behavior changes.** A bump ships every omni change since the last
  pin, including parser tightenings that newly reach validators and review rules. The bump PR
  audits the omni commit range, not just `go.mod`, and compose-verification (building the merged
  set) runs before release when several moves land in one cycle.
- **Local development** uses a `go.mod` replace to a sibling omni checkout; a replace never lands.
  The landed pin always points at omni `main`.

## Testing

- Engine-behavior tests live in omni alongside the code: each engine's `<engine>/v1review` tests,
  analysis/completion corpora, catalog/deparse golden files (`omni/<engine>/catalog/testdata/`),
  and real-engine oracle tests run under omni's per-engine gates.
- Driver behavior runs in omni's engine × version container matrix for sync goldens,
  result/type-mapping corpora, and driver-dependency bumps — catching the regression class where a
  bumped driver silently changes type names, per-request connection counts, or security defaults.
- Bytebase keeps product tests at the seam: advisor YAML parity fixtures
  (`backend/plugin/advisor/*/test/*.yaml`), registry dispatch / conversion / auth / TLS / SSH seam
  tests in `backend/plugin/{parser,schema,db}`, and workflow E2E tests (`backend/tests`, plancheck,
  reviewrun) against PostgreSQL and MySQL testcontainers (`backend/common/testcontainer/`).

## Risks

- **Behavior drift in ports and pin bumps.** Fixtures encode historical behavior, bugs included;
  differential parity runs plus omni's real-engine oracles mitigate regressions when modifying
  rules, analysis, or catalog logic.
- **Version skew.** Two repositories mean the contract must tolerate one side moving first;
  name-keyed identifiers and ignore-unknown semantics are load-bearing, and a contract change that
  breaks them needs a two-step rollout.
- **Pin-bump blast radius.** A single omni bump can simultaneously change parsing, V1 and V2
  review judgment, masking lineage (query span), generated DDL, and target-engine driver behavior.
  Mitigations: audited bumps (above), smaller and more frequent bumps, and omni's own per-engine
  gates.
- **The V1 config surface.** 105 rule types with payloads cross `omni/v1review` without importing
  proto; sharing config types by payload kind (`NamingConfig`, `TemplateConfig`, `CaseConfig`,
  etc.) keeps the contract bounded.
- **Omni CI cost.** Hosting all engine parsers, analyzers, catalogs, review rules, drivers, and
  oracle containers in omni requires keeping test gates scoped per engine.
- **Hotfix latency through the pin.** A target-engine incident (for example, a sync bug breaking an
  instance) is fixed in omni and shipped through a Bytebase pin bump, placing two repositories on
  the critical path. Mitigations: small frequent bumps so a hotfix bump carries little else, and
  omni `main` kept releasable at all times.
- **Omni's dependency surface.** Engine client SDKs (BigQuery, Spanner, DynamoDB, Cosmos DB,
  Databricks, etc.) and driver bumps live in omni's `go.mod`, with the Renovate and bump-audit
  load that follows. The injected-dialer rule caps this by keeping IAM, tunnel, and secret-manager
  dependencies in Bytebase.
- **Result conversion cost.** Query results are decoded into `omni/driver.Result` and converted to
  `v1pb.QueryResult` in `backend/plugin/db/util/omnidriver.go`; bounded by the existing result size
  caps, and shareable later if it ever shows up in profiles.
- **Two-repo debugging.** A production issue bisects across a pin; mitigated by small bumps and by
  omni's one-engine-one-tree layout keeping blame local.

## Alternatives

- **Status quo.** Rejected: the costs in Problem compounded with every new engine milestone,
  re-creating a Bytebase integration layer that the omni contracts eliminate.
- **Keep the drivers in Bytebase** (this document's first draft). Rejected: sync, result and type
  mapping, dump, and execute semantics are engine behavior — the exact split this migration ended
  everywhere else — and they were the engine code least covered by tests. Omni already ran real
  engines for its oracle gates; injecting credentials, TLS, and dialers as values gave product
  glue a much narrower home than keeping ~38k lines of driver code around it.
- **Merge the repositories.** Rejected: omni's value is an engine-pure module with its own
  verification discipline (corpus gates, oracle containers, scenario-driven development) and no
  product coupling; a monorepo would re-entangle what the contracts separated, and Bytebase's
  release cadence would drag engine work.
- **A third contracts module.** Rejected: `omni/metadata`, `omni/review`, `omni/v1review`,
  `omni/analysis`, and `omni/driver` already play this role inside omni, and a separate module
  would add a second version pair to skew-manage for no isolation gain.
