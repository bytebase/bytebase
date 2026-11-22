# Engine code moves to omni

2026-11-12 · Proposed

Decides which code under `backend/plugin` belongs in omni, the contracts that keep the two
repositories apart, and the order of the moves. Written on `main` at `fb411c9453`; "today" in this
document means that state.

## Problem

Omni (`github.com/bytebase/omni`) is Bytebase's engine repository: per engine, a recursive-descent
parser, a typed AST, a catalog that simulates DDL, completion, and review rules, verified by
scenario corpora and by oracle tests against real engines in containers. Bytebase consumes it
through a module pin.

The boundary between the two repositories is historical, not principled. Omni owns parsing and the
metadata type; Bytebase still owns most of what is computed from them, and the drivers that talk
to the engines — roughly 195k non-test Go lines of engine-specific code under `backend/plugin`
(advisor 58k, parser 63k, schema 35k, drivers 38k):

- **Split changes.** One logical engine change — a parser fix plus the review rule or query-span
  change that depends on it — is two PRs in two repositories, with a pin bump between them. The
  fix lands in omni; the judgment that should have moved with it waits in Bytebase.
- **Duplicated machinery.** Omni's catalogs already carry `metadata_load.go`, `migration_*.go`,
  `sdl.go` and full diff engines for MySQL and PostgreSQL; `backend/plugin/schema` carries a
  parallel hand-written implementation of the same conversions (`get_database_definition.go`,
  `differ.go` + `generate_migration.go`) that predates them.
- **Tests in the wrong repository.** The root `AGENTS.md` already rules that "engine dialect and
  DDL fidelity tests belong in omni", yet the advisor YAML corpora, query-span fixtures, and
  schema golden files that encode engine behavior live in Bytebase, out of reach of omni's corpus
  and oracle gates.
- **Every new engine is built twice.** Omni builds the engine (parser → catalog → completion);
  Bytebase then adds a `parser/<engine>` package of adapters and semantic features, an
  `advisor/<engine>` package, a `schema/<engine>` package, and a `db/<engine>` driver before the
  engine reaches users.
- **The least-tested engine surface is the driver.** `sync.go` — system-catalog queries per engine
  and version, mapped into `omni/metadata` — and result type mapping are pure engine behavior,
  but they sit outside omni's container matrix, so their regressions ship silently: a driver bump
  that returns empty type names, moves fractional digits between `DecimalSize` fields, or flips a
  security default passes Bytebase CI.

The direction is already in motion — omni owns the metadata proto, SQL Review V2 runs on omni's
`review` contract, and the newest engines (Redshift, Snowflake, Trino, GoogleSQL, …) keep their
analysis layers in omni. This document names the end state and the path for the rest.

## Where the boundary stands today

**Parsing and AST — moved.** Every production parse goes through omni. ANTLR survives only in a
comment, an indirect `go.mod` entry, and guard tests that assert it stays out
(`backend/plugin/parser/tsql/backup_test.go`).

**Metadata — moved.** `omni/metadata` (generated from omni's `proto/omni/metadata/database.proto`)
is the database schema snapshot type, imported at ~230 sites across `backend`; the store proto no
longer defines it. Bytebase persists it and wraps it in `backend/store/model` for indexed lookup.

**Walk-through and simulation — moved.** `backend/plugin/schema/<engine>/walk_through.go` adapts
omni catalogs (`New`, `Exec`, `LoadSQL`); the advisor's builtin walk-through check and SQL Review
V2's `WALK_THROUGH` rule both run on them.

**SQL review — two generations, mid-migration.**

- V1: `backend/plugin/advisor`, a registry keyed by `storepb.Engine` ×
  `storepb.SQLReviewRule_Type` (105 rule types, ~380 rule files: mysql 82, pg 60, oceanbase 60,
  mariadb 59, tidb 51, oracle 28, mssql 26, snowflake 16, redshift 1, bigquery 1). Rules walk omni
  ASTs but emit `storepb.Advice` and read `storepb.SQLReviewRule` payloads. Callers:
  `backend/runner/plancheck/statement_advise_executor.go` and
  `backend/api/v1/release_service_check.go`.
- V2: omni's `review` package is the declared contract ("the contract between Bytebase SQL Review
  V2 and the engine review packages"), with per-engine `Review` functions in omni
  (`mysql/review`, `pg/review`, `mariadb/review`, `oceanbase/review`, `tidb/review`,
  `mssql/review`, `oracle/review`, `snowflake/review`) covering the 11 `ReviewRuleType` rules.
  `backend/runner/reviewrun` resolves policy and store facts and calls one `Review` per
  (spec, engine).

The two generations stay entirely separate, in omni as in Bytebase. V1 judgment that moves to
omni goes to its own tree — the `v1review` contract and `<engine>/v1review` packages — which
neither imports `review` nor shares a type with it, and nothing V1 ports joins the V2 standard
rule set.

**Query analysis — Bytebase-side for the big engines.** `backend/plugin/parser/base` defines 13
per-engine capability registries: query span, splitter, changed resources, statement types/ranges,
query classification, completion, diagnose, explain, result limit, DML→SELECT backup transform,
restore SQL. The implementations in `backend/plugin/parser/<engine>` walk omni ASTs but live in
Bytebase for mysql (6.3k lines), pg (10.1k), mariadb (6.3k), oceanbase (4.3k), tidb (4.5k), tsql
(7.1k), plsql (5.8k), snowflake (2.6k). For the newest engines omni already owns this layer
(`redshift/analysis`, `snowflake/analysis`, `trino/analysis`, `googlesql/analysis`, and the rest),
and the Bytebase package is a thin adapter — `backend/plugin/parser/redshift` is the reference
shape.

**Schema conversion — split and partly duplicated.** `backend/plugin/schema/<engine>` holds
metadata→DDL deparse (`get_database_definition.go`, ~2k lines per engine), DDL→metadata
(`get_database_metadata.go`), a Bytebase-local metadata differ (`differ.go` → `schema.MetadataDiff`)
with per-engine `generate_migration.go`, and SDL support. The newer SDL/metadata-diff paths
(`sdl_migration.go` for pg, cockroachdb, mysql) already delegate to omni
(`catalog.LoadSDL`/`LoadSQL` → `catalog.Diff` → `catalog.GenerateMigration`). Callers:
`backend/api/v1/database_service.go` and the drivers' `dump.go` (dumps render synced metadata
through `schema.GetDatabaseDefinition`).

**Drivers — Bytebase, entangled with product types.** `backend/plugin/db/<engine>` owns
connections, metadata sync, query, execute, explain, and dump. The entanglement is concrete:
`db.ConnectionConfig` embeds the raw `*storepb.DataSource`; `QueryConn` builds
`[]*v1pb.QueryResult` inside the driver; `SyncDBSchema` already returns
`*metadatapb.DatabaseSchemaMetadata` (omni's type); `Dump` renders synced metadata through the
`schema` registry. The network and credential path sits beside the protocol code:
`db/util/aws.go` (RDS IAM tokens), `db/util/ssh.go` (tunnels), cloud dialers (`cloudsqlconn`),
and secret resolution upstream in `backend/component/secret`. Omni has no driver layer, but it
already links real drivers (`pgx`, `go-sql-driver/mysql`, `go-mssqldb`, `go-ora`) and runs real
engines in containers for its oracle gates.

**Engine coverage.** Omni's 18 engine trees cover 20 of the 24 `storepb.Engine` values, with
aliasing resolved on the Bytebase side (cockroachdb→pg, bigquery/spanner→googlesql,
dynamodb→partiql, mongodb→mongo). No omni tree exists for clickhouse, databricks, hive, redis;
their parser-layer code rides `parser/standard` or stays in Bytebase until an omni tree exists,
and Phase 4 starts their trees at the driver layer.

## Goals

1. Omni is the single home for code whose correctness is decided by an engine's grammar,
   semantics, or wire behavior: parsing, analysis (query span, classification, changed resources,
   splitting), review judgment, DDL simulation, diff, migration and definition generation,
   completion context, and the driver conversation — connect, sync, query, execute, dump.
2. Bytebase keeps everything decided by product configuration and stored state: policy
   resolution, stores, credential and network-path resolution (secret managers, IAM identities,
   tunnels, TLS material), connection lifecycle and scheduling, API conversion, engine dispatch.
   After this plan, the Bytebase repository speaks to target databases only through omni.
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
  credential, a `*tls.Config`, a `DialContext`); omni owns everything after the dial: session
  setup, sync queries, statement execution, result decoding. Engines whose client is a cloud SDK
  (BigQuery, Spanner, DynamoDB, Cosmos DB, Databricks) take credential material as values and
  bring their SDK into omni; injected dialers keep the IAM and tunnel machinery out of omni for
  the TCP engines.
- **Engine dispatch stays in Bytebase.** The `storepb.Engine` enum, the registries in
  `parser/base` / `schema` / `advisor`, and the fork/alias table (mariadb→its own omni tree,
  cockroachdb→pg, bigquery/spanner→googlesql, dynamodb→partiql) are the Bytebase-side seam. Omni
  stays package-per-engine with no engine enum.

## Target state under backend/plugin

| today | end state |
|---|---|
| `advisor/<engine>` (~380 rule files) | thin adapters; the judgment is omni `<engine>/v1review` functions behind `omni/v1review` |
| `advisor` root (registry, `sql_review.go`) | stays as V1's product surface; deleted only if the review V2 track retires V1 |
| `parser/<engine>` capability implementations | omni `<engine>/analysis`; Bytebase package shrinks to the adapter shape `parser/redshift` has today |
| `parser/base` | stays: registries, dispatch, product types; semantic types it owns today (query span, resources) move to an omni contract package and are re-exported during transition |
| `schema/<engine>` deparse/differ/migration | omni `<engine>/catalog` (`metadata_load`, `Diff`, `GenerateMigration`, deparse); Bytebase keeps the registry and `storepb` conversion |
| `schema` comparers, `differ.go` | replaced by omni catalog diff |
| `db/<engine>`, `db/transaction`, `db/util` engine helpers | omni `<engine>/driver`: connect, sync, query and result decoding, execute, explain |
| `db` root, `db/util/aws.go`, `db/util/ssh.go` | stay: dispatch, `storepb.DataSource` → omni config conversion, IAM dialers, tunnels |
| YAML corpora, golden fixtures for the above | move to omni next to the code, under its gates |

New omni contract surface:

- `omni/analysis` (new): the types one query analysis returns — query span (source columns,
  predicate columns, sensitive-field lineage), statement classification and types, changed
  resources, statement ranges — plus per-engine entry points in `<engine>/analysis`. The existing
  `<engine>/analysis` packages are the draft of this contract; writing `omni/analysis` promotes
  their shared shapes to a stable boundary the way `omni/review` did for rules.
- `omni/v1review` (new, separate from `omni/review`): the V1 contract. A rule is one function per
  engine, such as `<engine>/v1review.NamingTable(stmts, cfg)`, taking the statement nodes Bytebase
  already parsed and a plain config built from the rule's `SQLReviewRule` payload, and returning
  judgment-only findings (a condition code, the judged name, a byte range). The registry, payloads,
  severity levels, and advice wording stay in Bytebase.
- Completion keeps the cut the capability guide already fixed: omni emits parser-native
  token/rule/scope/intent signals; Bytebase resolves live metadata into product `Candidate`
  values.
- Driver contract (new, Phase 4): a neutral connection config — endpoint, resolved credentials,
  `*tls.Config`, optional `DialContext`, engine version — built by Bytebase from
  `storepb.DataSource`; omni-owned result and instance types in place of the
  `[]*v1pb.QueryResult` drivers build today, converted in `backend/api/v1` exactly as
  `v1.DatabaseMetadata` is converted from `omni/metadata`; execute logging as a callback
  interface, so task-run log rendering stays product.

## Migration plan

Ordered by value per unit of risk. Phases overlap; within each, an engine is one unit of work, and
a fix ported to a fork engine lands in the same PR as its parent
(`fix(mysql,mariadb): ...`), matching omni's own fork rule.

### Phase 1 — V1 review rules (advisor → omni `<engine>/v1review`)

The largest tree (58k lines) and the best-prepared path: rules already walk omni ASTs, so the walk
logic ports nearly verbatim; what changes is the frame. The walk moves to omni and returns
judgment-only findings; the advisor keeps its registration, payload handling, level, and advice
wording, and renders the findings as `storepb.Advice`. V1's callers, policies, and UI do not
change.

1. Create `omni/v1review` and land the first configurable rule end to end: `NAMING_TABLE`, in
   every engine with an omni review tree, with each advisor converted to an adapter
   (omni#825 and its Bytebase pin-bump PR).
2. Port by rule class across all engines at once — naming, column, statement, index, system — the
   way review feedback is already audited by class, so one semantic decision is made once and
   tested uniformly, instead of engine by engine rediscovering it.
3. Rules that read a live connection (`rule_statement_dml_dry_run`,
   `rule_statement_affected_row_limit`, `rule_insert_row_limit`,
   `advisor_disallow_offline_ddl`, `advisor_statement_object_owner_check`) split: Bytebase runs
   EXPLAIN / catalog queries and passes results as facts; omni judges. The fact computations
   themselves (dry-run EXPLAIN, affected-row estimation) join omni's driver layer in Phase 4;
   Bytebase still runs them, because connections never cross the v1review contract.
4. A port holds strict V1 parity: the same forms, name text, length unit, and anchor, defects
   included. The gate is an old-vs-new harness that runs the HEAD advisor and the adapter over the
   engine's YAML fixtures plus an edge corpus and requires zero diff, mutation-checked so it can
   fail. The behaviors the fixtures did not cover are appended to them. A change to what V1
   reports, including a V1 bug fix, lands later as its own commit. The judgment's own cases become
   omni `<engine>/v1review` tests.
5. `backend/plugin/advisor` shrinks to V1's product shell — registry, payloads, adapters — and
   stays as long as V1 does.

### Phase 2 — query analysis (parser/`<engine>` → omni `<engine>/analysis`)

Per engine, in dependency order of consumers (query span feeds masking; splitter feeds everything):

1. Create `<engine>/analysis` in omni and move the implementations: query span extractor,
   classification (`query.go`, `statement_type.go`), changed resources, statement ranges, split,
   limit rewrite, explain generation, diagnose, DML→SELECT backup transform and restore SQL.
2. The Bytebase package keeps only registration and conversion into `parser/base` types — the
   `parser/redshift` shape. `parser/base` registries keep their signatures, so the consumers
   outside `backend/plugin` (`backend/api/v1`, lsp, plancheck, export, taskrun, sheet) do not
   change.
3. Move the semantic types to `omni/analysis` once two engines prove the shapes; alias them from
   `parser/base` until consumers migrate.
4. Engine order: start with snowflake (omni `snowflake/analysis` already exists; Bytebase side is
   2.6k lines), then the mysql family (mysql, mariadb, oceanbase, tidb — shared shape, four
   trees), then pg (largest, 10.1k lines, plus plpgsql), then tsql and plsql.

`parser/standard`, `tokenizer`, and the engines with no omni tree (clickhouse, databricks, hive,
redis) stay in Bytebase until an omni tree exists for them; they are small and inert.

### Phase 3 — schema conversion (schema/`<engine>` → omni `<engine>/catalog`)

The goal is convergence, not relocation: omni catalogs already own simulation, diff, migration,
and SDL for MySQL and PostgreSQL, verified by oracle tests that caught bugs the Bytebase golden
files cannot (dependent view rebuilds, serial/sequence association).

1. Metadata→DDL deparse (`get_database_definition.go` and the per-object definition funcs) moves
   into omni next to each catalog's existing deparse code; dumps and
   `backend/api/v1/database_service.go` keep calling through the `schema` registry.
2. The Bytebase differ (`differ.go` + per-engine `generate_migration.go`) retires in favor of
   `catalog.LoadMetadata` → `catalog.Diff` → `catalog.GenerateMigration`, the path
   `sdl_migration.go` already walks for pg and mysql. Golden fixtures move, and the `record`
   regeneration process documented in `backend/plugin/schema/AGENTS.md` moves with them; fixture
   flips are classified against real-engine behavior before acceptance, never assumed to be
   improvements.
3. Fork gaps close as part of the move: `schema/mariadb` exists today only because omni's
   `mariadb/catalog` lacks `LoadMetadata`/`Diff`/`GenerateMigration`; the move builds those out in
   omni rather than preserving the simulate-on-mysql workaround.
4. Engines whose omni catalog does not exist yet (oracle, mssql, trino, …) keep their Bytebase
   `schema/<engine>` package until omni's L5 layer reaches them; the registry hides the
   difference from callers.

### Phase 4 — drivers (db/`<engine>` → omni `<engine>/driver`)

Last because its contract surface is the widest (connection config, result types, execute
callbacks) and because Phase 3 already moved what dump renders. The product keeps the lifecycle —
`dbfactory` pooling, `schemasync` scheduling, task execution — and calls omni through the `db`
registry it already uses.

1. Fix the contract first: the neutral connection config and omni-owned result/instance types
   (Contracts above). `ConnectionConfig` stops embedding `*storepb.DataSource`; the conversion,
   secret resolution, IAM token acquisition, and tunnel setup (`db/util/aws.go`, `db/util/ssh.go`,
   `cloudsqlconn`) stay in Bytebase and arrive as values and dial funcs.
2. Move per engine in two slices: **sync + dump** first (read-only, and the highest-value tests:
   system-catalog queries per engine × version finally land inside omni's container matrix), then
   **query + execute** (result decoding, type mapping, transaction modes, explain).
3. Driver dependencies and their fork replaces (`azcosmos`) move to omni's `go.mod`, and the
   driver-bump audit discipline moves with them — omni's matrix is where a bump's behavior diff
   (type names, connection counts, security defaults) is provable.
4. Engines with no omni tree (clickhouse, databricks, hive, redis) get driver-only trees. The
   driver layer does not depend on L1–L6, so starting a tree at the driver inverts the usual
   build order harmlessly.
5. `backend/plugin/db` shrinks to the registry seam — `Open` dispatch plus config and result
   conversion — and the per-engine directories delete.

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

- Engine-behavior tests move with the code: each ported V1 rule's judgment gains omni
  `<engine>/v1review` tests, while the advisor YAML corpora stay in Bytebase as the adapters'
  parity gate; query-span and schema golden files become omni analysis/catalog scenarios, gaining omni's
  corpus gates and real-engine oracle runs that Bytebase CI never had.
- Driver behavior gains what Bytebase CI never gave it: an engine × version container matrix in
  omni for sync goldens, result/type-mapping corpora, and driver-dependency bumps — the regression
  class where a bumped driver silently changes type names, per-request connection counts, or
  security defaults while every reply stays equal.
- Bytebase keeps product tests at the seam: one smoke per capability per engine through the
  registries (dispatch, conversion, config plumbing), plus the existing workflow tests
  (`backend/tests`, plancheck, reviewrun) unchanged — workflow E2E still runs real target engines,
  but to prove rollouts and policies, not engine behavior.
- During a port, the differential corpus is the gate; after it, the omni pin bump diff plus the
  Bytebase seam tests are.

## Risks

- **Behavior drift in ports.** Fixtures encode the old implementation, including its bugs; parity
  runs plus omni's real-engine oracles are the mitigation, and a port is not "done" on green
  fixtures alone.
- **Version skew.** Two repositories mean the contract must tolerate one side moving first;
  name-keyed identifiers and ignore-unknown semantics are load-bearing, and a contract change that
  breaks them needs a two-step rollout.
- **Pin-bump blast radius grows.** Today a bump mainly changes parsing, simulation, and the
  eleven V2 rules; after this plan it also changes V1 review judgment, masking lineage (query span),
  generated DDL, and every driver conversation at once. Mitigations: audited
  bumps (above), smaller and more frequent bumps, and omni's own gates running per PR.
- **The V1 config surface.** 105 rule types with payloads must be expressible in `v1review`
  without importing proto. The contract shares config types by payload kind (`NamingConfig` for
  every naming rule) rather than per rule, so the six payload messages bound it.
- **Omni CI cost.** Moving ~195k lines plus fixtures grows omni's suite, which already runs every
  package on `docs/` changes and holds real-engine containers; package layout and corpus placement
  must keep gates per-engine.
- **Hotfix latency through the pin.** Once drivers move, a target-engine incident (a sync bug
  breaking an instance) is fixed in omni and shipped through a pin bump; two repositories sit in
  the critical path of a customer fix. Mitigations: small frequent bumps so a hotfix bump carries
  little else, and omni `main` kept releasable at all times.
- **Omni's dependency surface grows.** Engine client SDKs (BigQuery, Spanner, DynamoDB, Cosmos
  DB, Databricks) and driver fork replaces move into omni's `go.mod`, with the Renovate and
  bump-audit load that follows. The injected-dialer rule exists to cap this: IAM, tunnel, and
  secret-manager dependencies stay in Bytebase.
- **Result conversion cost.** Query results are decoded into omni's type and converted to
  `v1pb.QueryResult` at the API boundary; for large result sets that is an extra pass. Bounded by
  the existing result size caps, and shareable later if it ever shows up in profiles.
- **Two-repo debugging.** A production issue now bisects across a pin; mitigated by small bumps
  and by omni's one-engine-one-tree layout keeping blame local.

## Alternatives

- **Status quo.** The costs in Problem compound: every omni engine milestone (L7 in omni's
  capability guide) re-creates a Bytebase integration layer that this plan would have already
  deleted.
- **Keep the drivers in Bytebase** (this document's first draft). Rejected: sync, result and type
  mapping, dump, and execute semantics are engine behavior — exactly the split this plan ends
  everywhere else — and they are the engine code least covered by tests today. The purity
  argument did not survive inspection either: omni already links four real drivers and runs real
  engines for its oracle gates. What genuinely must stay product-side is the credential and
  network path, and injecting it as values gives that a narrower home than keeping 38k lines of
  driver code around it.
- **Merge the repositories.** Rejected: omni's value is an engine-pure module with its own
  verification discipline (corpus gates, oracle containers, scenario-driven development) and no
  product coupling; a monorepo would re-entangle what the metadata and review contracts just
  separated, and Bytebase's release cadence would drag engine work.
- **A third contracts module.** Rejected: `omni/metadata` and `omni/review` already play this role
  inside omni, and a separate module adds a second version pair to skew-manage for no isolation
  gain.
