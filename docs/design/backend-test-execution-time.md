# Backend test execution time

Status: in progress

## Scope

Cut the execution time of `go test ./backend/...`. It takes 14–19.5 minutes in
CI and has grown 54% since May (p50 10.2 → 15.7 min).

Runner capacity and queue time are out of scope. The self-hosted VM exists
because the Bytebase binary is expensive to build, and the warm Go build cache
is the point.

## Measurements

One local pass of `go test -p=8 -json ./backend/...` on 20 vCPU / 31 GB, with
warm build and image caches, finishes in **14 m 19 s**. Individual package wall
times sum to **3180 s**, so packages overlap about 3.7×.

Per-operation costs are means of 5 sequential runs:

| Operation | Cost |
| --- | ---: |
| MySQL container start | 10 420 ms |
| Postgres container start | 4 328 ms |
| Server shutdown | 1 047 ms — 1 036 of it `httpServer.Shutdown` |
| Server boot | 703 ms — `CREATE DATABASE` 39, `NewServer`+`LATEST.sql` 381, serve/healthz/signup/login 266 |

A Postgres container costs 6.2× a full server boot, and a boot's own teardown
costs more than the boot. `GetTestPgContainer` has no pooling, so every call is
a new container. The suite makes roughly 400 Postgres containers, 37
non-Postgres ones, and 207 server boots per run. The non-Postgres and boot
counts are exact; the Postgres count is approximate, because `docker events`
caught 297 creations but only started recording after two dozen packages had
finished.

### What that costs each package

Four packages run their tests **serially**: `store`, `migrator` and
`component/review` never call `t.Parallel()`, and `api/v1` calls it 8 times
across 350 tests. Their test time and wall time match to within 1%, so their
container cost *is* wall clock.

| Package | Tests | Containers | Container time | Wall | Share of wall |
| --- | ---: | ---: | ---: | ---: | ---: |
| `backend/store` | 128 | 90 | 390 s | 459 s | **85%** |
| `backend/api/v1` | 350 | 100 | 433 s | 540 s | **80%** |
| `backend/migrator` | 18 | 16 | 69 s | 90 s | **77%** |
| `backend/component/review` | 53 | 41 | 178 s | 282 s | **63%** |

Those shares are arithmetic, not estimates. Remove a container start in one of
those packages and you remove a second of its wall clock.

The two other expensive packages have to be read differently:

- **`backend/tests`** — 635 s wall. It calls `t.Parallel()` 203 times and runs
  about 3.7× parallel inside, so costs overlap. Its fixtures come to roughly
  812 s of work (104 containers, plus 207 boot-and-shutdown cycles), which
  compresses to about 220 s of its wall.
- **`backend/plugin/schema/oracle`** — 315 s wall, and fixtures are not the
  story here. Its container starts in 14 s. The other 273 s is DDL work against
  a real Oracle.

### The floor

`backend/plugin/...` costs 866 s across 69 packages, and splits by who owns the
code: `plugin/schema/*` is 475 s in 6 packages, `plugin/db/*` is 339 s in 18.
The other 45 packages — pure parser and advisor tests, no containers — come to
51 s. Parsing is not what costs; engines are.

What any of this buys follows one rule:

```
wall ≈ max( slowest single package , sum of package walls ÷ 3.7 )
```

It reproduces the measured run: `max(635, 3180 ÷ 3.7)` is 859 s, against an
actual 859 s. Treat the 3.7 as an observed property of this schedule, not a
constant — it will drift as package times change.

`backend/tests` at 635 s is the slowest package. Fixing the four serial packages
makes the run floor-bound on it almost immediately, so everything after that has
to come out of `backend/tests` itself. Working through the efforts below should
land the local pass around 6 minutes.

## Design

Seven efforts, ordered by payoff per unit of work.

Efforts 1, 3, 4 and 5 all target `backend/tests`. Their figures are fixture
cost inside a package that runs 3.7× parallel, and effort 1's saving sits inside
effort 4's, so they do not simply add. Together they should take it from 635 s
of wall clock to roughly 330 s.

Effort 6 was rescoped and now runs ahead of effort 2 for `api/v1`; the numbering
is kept so existing references resolve.

### [✓] 1. Close idle connections before shutting the server down

**Three lines.** `httpServer.Shutdown` burns 1036 ms on every one of the 207
boots. It is waiting out an idle HTTP/2 connection that nobody will reuse: the
test client's `http2.Transport` is never closed. Runners exit in 0 ms.

```go
func (ctl *controller) Close(ctx context.Context) error {
    if ctl.client != nil {
        ctl.client.CloseIdleConnections()
    }
    ...
}
```

**Measured, not projected.** With those three lines applied, shutdown drops
from 1047 ms to **1 ms**, and a full boot-and-shutdown cycle goes from 1750 ms
to 676 ms. Across 207 boots that is ~217 s.

That 217 s is part of effort 4's total, not on top of it. This goes first only
because it is three lines and needs no restructuring, and it stops mattering
once effort 4 collapses the boot count.

**What it bought.** Over 5 cycles the shutdown went from 1058 ms to **1 ms**
and the boot-and-shutdown cycle from 1720 ms to **691 ms**, as projected. A
package-isolated `go test ./backend/tests/` — not the full-suite figure above —
went from **679 s to 611 s** across its 233 boots: 227 s off the summed test
time, compressed 3.3× by the package's own parallelism. Measure the efforts
below against 611 s.

### 2. One shared container per package, then turn parallelism on

**459 s of wall clock, 390 s of it container starts.** `store` starts 90
containers, and because it runs serially that cost is its wall clock almost
exactly.

`api/v1`, `component/review` and `migrator` were in this effort's original
scope. AGENTS.md now confines a real metadata Postgres to `store` and `tests`,
so all three move to effort 6 instead.

Two changes, in this order:

1. Give each package a `TestMain` that starts one Postgres and migrates a
   template database once. Each test then gets its own database with
   `CREATE DATABASE … TEMPLATE`. `backend/tests/main_test.go` is the reference.
2. Add `t.Parallel()`. Sharing the container is what makes parallelism
   affordable: a 44 ms database copy can run concurrently in a way a 4.3 s
   container start cannot.

Measured on the migrated Bytebase schema (9.5 MB): migrating the template costs
309 ms once, a plain `CREATE DATABASE` is 32 ms, and
`CREATE DATABASE … TEMPLATE` is **44 ms** — 98× cheaper than the 4328 ms
container it replaces.

One detail. Close the template connection after migrating, because `TEMPLATE`
refuses to copy a database that still has a session attached.

This section originally called for `api/v1` to share a Postgres "rather than
using a fake store". Effort 6 reverses that.

Expect tests that silently assumed a virgin database to fail. That is the point.
Fix them with per-test databases, not with cleanup hooks — those reintroduce
ordering coupling.

Step 1 alone takes `store` from 459 s to roughly 75 s. Step 2 should take it
below that; what remains is real work across 128 tests.

### 3. Pool the provisioned Postgres instances

**450 s of fixture time inside `backend/tests`.** 104 of its tests call
`provisionPgInstance` to get a managed instance for Bytebase to point at, and
each one is a fresh container.

Most of these tests only need an instance to exist and to hold their own
databases, which they already create. Those can borrow from a pool of about 8
long-lived containers and take a database each, leaving about 35 s.

Ten files are the exception: they call `UpdateInstance`, `DeleteInstance` or a
data-source mutation, so a shared instance would break for everyone else. The
pool needs checkout semantics — borrow a shared instance by default, take an
exclusive one when the test mutates instance config:

```
project_instance_test.go   project_instance_iam_collision_test.go
project_instance_archive_task_run_test.go   cross_project_delete_test.go
plan_update_test.go   data_source_test.go   sql_query_data_source_test.go
sql_export_data_source_test.go   instance_keytab_retention_test.go
mcp_forbidden_credential_mints_test.go
```

There is one place to fix this because `backend/tests` is the only package that
starts a Bytebase server — `server.NewServer` has exactly two call sites, the
real binary and `backend/tests/tests.go`. Keep it that way. A server-starting
test elsewhere would need its own workspace scaffolding and its own instance
pool, and would put a second package on the critical path.

### 4. Isolate per project, not per workspace

**362 s of fixture time inside `backend/tests`.**

Every test there gets its own server, and therefore its own workspace: a
database, a full `LATEST.sql` migration, signup, login, a license upload, two
rollout policies and a project. That is 703 ms up and 1047 ms down, 207 times
over, almost all of it rebuilding scaffolding no test asserts on.

Project is already Bytebase's tenancy boundary, so make it the test boundary:
one server for the whole package, one shared workspace, and a project per test —
plus its own principal where the test needs one. The collision tests already prove this
works. `setupCollidingProjects` puts two projects in one workspace and asserts
neither can see the other.

**Workspace-scoped singletons are the exception.** These files call
`UpdateSetting` on rows that exist once per workspace — the MCP capability and
masking toggles, workspace approval, maximum request expiration, email, AI. Two
parallel tests toggling one would race, so they keep a workspace of their own:

```
approval_test.go   mcp_capability_setting_test.go   mcp_masking_test.go
login_audit_test.go   review_run_lifecycle_test.go   sensitive_data_test.go
mcp_forbidden_credential_mints_test.go   maximum_request_expiration_test.go
webhook_test.go   (via webhook_helpers_test.go)
```

Workspace IAM grants are additive, so `addMemberToWorkspaceIAM` is safe provided
each test grants to its own principal.

`TestWebhookIntegration` needs separate attention: 144 s, the slowest test in
the repo, 24 subtests behind one boot with several fixed sleeps.

### 5. Postgres only for API and workflow tests

**Up to 320 s inside `backend/tests`**, of which 115 s is container starts and
the rest is the duplicated test bodies. `TestSQLReviewForMySQL` costs 52.7 s
where `TestSQLReviewForPostgreSQL` costs 17.9 s for the same assertion. The rule
is now in the root `AGENTS.md`.

`docker events` recorded 11 MySQL containers from `backend/tests`. The
candidates, one container each:

```
TestSQLReviewForMySQL              TestSyncerForMySQL
TestSQLExport                      TestSQLAdminQuery
TestSensitiveData                  TestAdminQueryAffectedRows
TestSQLQueryStopOnError            TestSQLAdminExecuteStopOnError
TestSQLExportDataSourceResolution  TestSQLQueryDataSourceResolution
TestActionCheckCommand             TestGetLatestSchema   (MySQL arm of an engine switch)
```

Where a Postgres equivalent proves the same workflow, delete the MySQL copy and
save the whole test. Where the MySQL path is the only coverage, port it to
Postgres and save only the container difference — which is why the payoff is a
ceiling, not a promise.

Three keep their engine. Each needs a comment saying why:

- `TestGhostSchemaUpdate` and `TestGitOpsRolloutGhostDirective` — gh-ost is
  MySQL-only.
- `TestTransactionMode` — per-engine transaction semantics are the behavior
  under test. It covers MySQL, Postgres, Oracle and MSSQL in one table, and
  boots no server.

### 6. Seams instead of containers

**912 s of wall clock, 680 s of it container starts.** AGENTS.md confines a real
metadata Postgres to `store` and `tests`. `api/v1`, `component/review` and
`migrator` are not on that list and start 157 containers between them.
`backend/api/mcp` is the proof this is livable: 168 tests in 39.5 s, 16 of its
18 files containerless.

Effort 2 declined a fake store on two grounds, and both fail here. `api/v1` is
not testing its API — `mcp_info_test.go` calls `service.GetMCPInfo` as a Go
method, so the container buys a `*store.Store` and nothing the test asserts. And
"a fake store" was the wrong unit: `store.Store` has 305 methods and `api/v1`
calls 181, but `api/mcp` needed five (`serverStore`, faked in 62 lines) and
`mcp_gate.go` needed one (`mcpSettingsReader`). Interfaces go beside the handler
that reads them, never over the store.

Fourteen files in `api/v1` start containers, ~90 tests. Read from names, not
bodies — verify the back two rows:

| Disposition | Tests | Shape |
| --- | ---: | --- |
| Move to `backend/store` | ~20 | `Test*Claims`, concurrent-update and serialization, `TestMixedIssuePatchRollsBackWhenLabelsFail`, retention filter boundaries, `TestIssueApprovalFiltersRunBeforePaging` |
| Pure function over plain data | ~40 | canonical-name tests, the four `TestCheckRelease*`, converters |
| Narrow interface and a fake | ~30 | `TestGetMCPInfoHandler`, the list-and-hide tests, `TestApproveIssueFailsClosedWhenIAMLookupFails` |

`component/review` splits the same way:
`TestApplyApprovalTemplateAndCreatePlanCheckRunDoNotDeadlock` is lock ordering
and moves to `store`; the CEL, template-matching and target-unfolding tests are
pure functions already, wearing a container for the package's sake.

Prefer the middle row: a handler is fetch, decide, convert, and only the fetch
needs a store.

`migrator` is done, by deletion rather than by moving. `TestLatestVersion` and
`TestVersionUnique` need no database and stay; the other 16 booted a container
each to apply one migration to a hand-built fixture and assert the transform.
Moving them to `backend/store` would have relocated the 90 s, not removed it, so
they were deleted outright. The package now runs in under 2 s.

That is 90 s bought at a real price, and the price should be stated plainly: the
migration SQL — including irreversible data transforms over customer metadata —
now has no test coverage, and no fake can restore it. Anyone reintroducing
coverage here should bring back the fixtures against a shared container rather
than one per test.

**Every fake needs a contract test** — one table run against both the fake and
the real store — or effort 2's objection is right. Since none of these packages
may hold a container, that suite lives in `backend/store`. No `mockgen`, no SQLite.

Convert `mcp_info_test.go` first — `mcpSettingsReader` exists and the test is 60
lines — and measure before doing the rest. Server boots go the same way:
`mcp_capability_setting_test.go` spends 6 to read and write one settings row,
the saved-query list and filter tests 12, GitOps `CheckRelease` 11 of 17. Keep
the 27 `TestCollision*` and `TestClaim*` tests where they are —
`backend/store/AGENTS.md` requires them for a bug class unit tests cannot catch.

### 7. Engine conformance to omni

**475 s of package wall time, and the largest change here.** The
`plugin/schema/*` packages assert engine fidelity: generate DDL, apply it to a
real engine, dump it back, compare. That belongs to omni.

The blocker is that the round-trip is expressed in Bytebase's metadata proto
(`storepb`, `plugin/schema`, `store/model`), so omni must own the metadata model
first. The ownership split above is the plan: all 475 s of `plugin/schema/*`
moves, of which Oracle alone is 315 s; all 339 s of `plugin/db/*` stays, because
that is our driver.

For most engines the metadata model is not the near blocker — omni has no engine
to move to. It ships a parser and AST for Oracle and for MSSQL and nothing more,
so `schema/oracle` (272 s) and `schema/mssql` (51 s) have no destination; both
already use omni, but only for the AST their own extractors walk. omni does own
a TiDB catalog and deparser, and `schema/tidb` imports neither. Only pg and
mysql delegate, and only in part: pg's SDL diff is a 140-line adapter over
`omni/pg/catalog`, while `get_database_definition.go` (4855 lines) and
`metadata_migration.go` (2956 lines) are ours; mysql's `GetDatabaseMetadata` is
a 46-line adapter, while its `GenerateMigration` (1121 lines) is not. Every
container test in `plugin/schema/*` calls one of those Bytebase-owned entry
points, so the tests follow the implementation and not the other way round.

What was already omni's is gone. 24 files and 150 tests in `schema/pg` called
`omni/pg/catalog` directly — `LoadSDL`, `Diff`, `GenerateMigration` — with no
Bytebase symbol in them. Read against omni's own `pg/catalog` suite, all but
four of their behaviors were already covered there by more cases and stronger
assertions: they matched substrings of the rendered SQL where omni inspects
typed migration ops. Two of them could not fail at all — the pair named for
EXCLUDE constraints has no EXCLUDE constraint in either fixture. The four
genuine gaps went into omni beside the cases they belong with; the rest were
deleted rather than moved. That is 3 s of the 403 s this effort is about, which
is the measure of how much of the cost sits behind the port rather than beside
it.

`plugin/db/starrocks` (158 s) is gone. Nearly all of it was two testcontainer
tests booting the `allin1` image — an FE and a BE in one container, with a
readiness wait measured in minutes — to assert materialized-view sync and dump
round-trip. That is engine fidelity, which belongs in omni, paid for on every PR
by every engineer. The two tests and the `GetStarRocksContainer` helpers were
deleted; the package now runs in under 2 s on its remaining unit tests.

### One loose end

`action/**` is in the workflow's `paths` filter, but `go test ./backend/...`
never runs the seven test files under `action/`. Add `./action/...` to the
command. They are unit tests and cost about 2 s.

## Reproducing

```bash
go test -p=8 -timeout 60m -json ./backend/... > test.json

# package wall time, descending
jq -r 'select(.Action=="pass" or .Action=="fail") | select(has("Test")|not)
       | "\(.Elapsed)\t\(.Package)"' test.json | sort -rn | head -20
```

Package wall time is the figure to trust. Summing per-test `Elapsed` is
misleading: parallel subtests overlap, so the sum can exceed the package's own
wall time by an order of magnitude — `plugin/schema/oracle` reports 3666 s of
subtest time inside a 315 s package.

Per-operation costs came from a temporary test in `backend/tests` that timed
`StartServerWithExternalPg` phase by phase, `GetPgContainer` and
`getMySQLContainer` in a loop, plus four `slog` probes inside `server.Shutdown`.
All reverted.
