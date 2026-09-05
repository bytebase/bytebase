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
land the local pass around 7.5 minutes.

## Design

Six efforts, ordered by payoff per unit of work. Two have moved since this was
written, and the numbering is kept through both so existing references resolve:
effort 3 — a checkout pool for the 104 `provisionPgInstance` callers — is
dropped, and effort 6 was rescoped and now runs ahead of effort 2 for `api/v1`.

Efforts 1, 4 and 5 all target `backend/tests`. Their figures are fixture cost
inside a package that runs 3.7× parallel, and effort 1's saving sits inside
effort 4's, so they do not simply add. Together they should take it from 635 s
of wall clock to roughly 450 s: dropping effort 3 leaves its 450 s of container
time in place, which is about 120 s of wall at that compression, on top of the
330 s the four efforts were originally worth.

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

### [✓] 2. One shared container per package, then turn parallelism on

**459 s of wall clock, 390 s of it container starts.** `store` starts 90
containers, and because it runs serially that cost is its wall clock almost
exactly.

`api/v1`, `component/review` and `migrator` were in this effort's original
scope. Effort 5 confines a real metadata Postgres to `store` and `tests`, so all
three move to effort 6 instead.

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

**What it bought, measured on one Mac, same tests both sides.** `backend/store`
went from **382.95 s to 12.0 s** — 90 container starts down to 1, and 20 files
off `GetTestPgContainer`. `main_test.go` holds the whole mechanism in one
helper: `newTestDB` returns a raw handle for seeding, a Store for the code under
test, and the URL for tests that open their own connections, all on a database
copied from the template.

The copies are never dropped, deliberately. They are physical — 140 tests hold
roughly 1.3 GB — but they live inside the package's own container, which
`TestMain` terminates on the way out, so the space comes back with it. Dropping
each database costs about 54 ms, or 7 s across the package, to reclaim disk that
is already reclaimed.

**What step 2 bought, measured on one Linux box (20 vCPU), same tests both
sides.** `t.Parallel()` on all 139 tests took `backend/store` from **16.2 s to
10.9 s**, means of runs spanning 16.0–16.3 and 10.8–10.9. Those are not the
12.0 s above; that was the Mac. Two things that would have blocked this were
checked first and held: Postgres advisory locks are per-database, so the lock and
claim tests cannot reach each other across per-test databases, and eight
concurrent `CREATE DATABASE … TEMPLATE` from one template all succeed.

**What is left is nearly all floor.** The package costs 1.3 s with no container
at all and 6.9 s for a single test that needs one, so 5.6 s of the 10.9 s is the
one container start and template migration that every test now queues behind
inside `metaOnce`. Subtracting that 6.9 s from each side, the 138 other tests'
own work went from 9.3 s to 4.0 s. Nothing further is worth spending here: the
next second has to come off the container, not off the tests.

**Nine tests keep serial subtests, and two of them give up top-level parallelism
for it.** `TestAuditLogRetentionFilteringEndToEnd` sets one workspace license row
per subtest and reads it back, and `TestLoginAttemptClaim`'s purge subtest
asserts a table-wide delete count that only holds while nothing else writes the
table; both carry a comment saying so. The other seven parallelize at both levels
because their subtests are keyed apart — distinct identities, codes, token
hashes — or take a database each. `tparallel` enforces all-or-nothing per test,
which is why those two go serial rather than parallel with serial subtests.

Peak concurrent connections to the shared container is 38 against Postgres's
default 100, at `-parallel=20`. A runner with many more cores should have that
re-measured before it is trusted.

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

### [✓] 5. Postgres only for `backend/store` and `backend/tests`

**Done, and the headline example in the original sizing was wrong.** This section
said `TestSQLReviewForMySQL` cost 52.7 s where `TestSQLReviewForPostgreSQL` cost
17.9 s "for the same assertion". They were not the same assertion:
`sql_review_mysql.yaml` held 21 cases against `sql_review_pg.yaml`'s 9. Deleting
it as a duplicate would have dropped real rule coverage.

The principle that resolved it is sharper than the one written here.
**`backend/tests` tests workflows; engines belong to the layer that owns them.**
Sorting the expensive tests by that rule gave four outcomes rather than one:

- **Same workflow run twice.** Five tests had literally mirrored cases —
  "MySQL - Second statement fails" beside "PostgreSQL - Second statement fails" —
  for behavior that is ours, not a dialect's. MySQL arm dropped, one unique case
  ported: **127.9 s → 59.9 s**.
- **MySQL-only, engine incidental.** Three tests about data source resolution and
  masking, ported to Postgres. The translation was not mechanical — Postgres
  grants are per object, `BIN_TO_UUID` became a `uuid` cast, and the catalog
  needed a `public` schema.
- **Engine wearing workflow clothing.** `TestSQLReviewForMySQL` was rule coverage
  in the wrong place: of the 41 rules its fixture asserts, 39 have a file under
  `plugin/advisor/mysql/test` carrying real `want:` expectations, and
  `TestMySQLRules` drives ~60 rules through them with no container, in 2.3 s.
  **Two did not, and checking that the file merely exists would have missed it.**
  `RunSQLReviewRuleTest` builds its context with `Driver: nil`, so the two rules
  that need a live `EXPLAIN` — `STATEMENT_AFFECTED_ROW_LIMIT` and
  `STATEMENT_DML_DRY_RUN` — have yaml files holding statements and no
  expectations at all. The first is covered anyway by two driver-backed tests
  against a fake `database/sql` EXPLAIN driver; the second was not, so deleting
  the workflow test would have dropped its only coverage, and
  `TestMySQLDMLDryRunAdvisor` was written against that same fake driver to
  replace it. Deleted, with
  `TestSyncerForMySQL` (its Postgres twin sits in the same file) and
  `TestGetLatestSchema`'s MySQL arm (it asserted a literal dump, including MySQL's
  `SET @OLD_UNIQUE_CHECKS` preamble). `TestTransactionMode` is deleted, 41 s. It had
  four engine cases whose expectations were identical — `expectRollbackOn` was `true`
  in all four and `skipTransaction` was never set, so both fields that existed to
  express engine variation were dead, along with two unreachable branches.

  **This leaves a known gap, recorded here rather than left silent.** Nothing now
  covers the execution half of the `-- txn-mode` directive:
  `executeInTransactionMode` and `executeInAutoCommitMode` appear in no test, so
  `txn-mode = on` could stop wrapping and every test would stay green. Parsing is
  still covered by `plugin/parser/base`. Anyone reintroducing coverage should
  guard the mode switch on one engine, and the case actually worth an engine
  matrix is DDL inside `txn-mode = on`, where MySQL and Oracle implicitly commit
  and defeat it — which is the reason the directive exists, and which no version
  of the deleted test ever covered.

- **Engine is the workflow.** `TestGhostSchemaUpdate` and
  `TestGitOpsRolloutGhostDirective` keep MySQL, and now say why in a comment:
  gh-ost is MySQL-only. `TestActionCheckCommand`'s database-group subtest keeps it
  too — the port was tried and reverted, because the declarative check returns
  three errors against the same schema on Postgres. Worth chasing separately.

MySQL in `backend/tests` is now those three tests and nothing else.

Two packages may hold a real database, and no others. `backend/store` is where a
metadata query is asserted against one, and `backend/tests` is the only package
that boots a server. Everywhere else — `api/v1` above all — the handler's
decide-and-convert half is tested over a fake or over plain data, which is effort
6; an API test does not earn a container by being an API test. Inside the two
packages that keep one, the engine is Postgres: engine dialects and DDL fidelity
belong in omni. The root `AGENTS.md` carries that second half of the rule today,
in the words "test API and workflow behavior against Postgres only".

`docker events` recorded 11 MySQL containers from `backend/tests` when this was
written. Each of those twelve candidate tests is accounted for above; the one
that is not is `TestActionCheckCommand`, which keeps its container.

The original plan here said "where a Postgres equivalent proves the same
workflow, delete the MySQL copy". That held for `TestSyncerForMySQL`, whose twin
was already in the same file. It did not hold for `TestSQLReviewForMySQL`: the
copy was not equivalent, and the thing that made it deletable was finding the
coverage a layer down rather than a layer across. **Check the layer below before
calling a test a duplicate.**

### 6. Seams instead of containers

**822 s of wall clock, 611 s of it container starts**, after `migrator` was
settled by deletion below. Effort 5 confines a real metadata Postgres to
`store` and `tests`; `api/v1` and `component/review` are not on that list and
start 141 containers between them. `backend/api/mcp` is the proof this is
livable: 168 tests in 39.5 s, 16 of its 18 files containerless.

Effort 2 declined a fake store on two grounds, and both fail here. `api/v1` is
not testing its API — `mcp_info_test.go` calls `service.GetMCPInfo` as a Go
method, so the container buys a `*store.Store` and nothing the test asserts. And
"a fake store" was the wrong unit: `store.Store` has 305 methods and `api/v1`
calls 181, but `api/mcp` needed five (`serverStore`, faked in 62 lines) and
`mcp_gate.go` needed one (`mcpSettingsReader`). Interfaces go beside the handler
that reads them, never over the store.

Fourteen files in `api/v1` start containers, 90 tests, classified from their
bodies:

| Disposition | Tests | Shape |
| --- | ---: | --- |
| Move to `backend/store` | 18 | the 7 lockout and claims tests, 6 concurrency and transaction tests, 2 retention-filter boundaries, `TestIssueApprovalFiltersRunBeforePaging` |
| Replace with a fake | 66 | canonical-name assertions through real services, the list-and-hide tests, validation rejections, `TestGetMCPInfoHandler`, `TestApproveIssueFailsClosedWhenIAMLookupFails` |
| Pure function over plain data | 6 | `TestExtractDomain`, `TestLDAPLoginIdentity`, the MFA temp-token shapes |

The last two rows are one boundary, not two: a fake test becomes a pure one as
soon as its decide-half is extracted, so 66 is a ceiling on the fakes and 6 a
floor on the pure functions.

The rest of `backend/api` adds 14, all fakes — `oauth2` 6, `mcp` 5, `auth` 2,
`lsp` 1. `oauth2`'s consent-ceiling tests seed five workspaces and their MCP
settings in raw SQL to exercise one ceiling read each. The 6 pure tests need no
work: verified by call closure, none reaches a container.

**Outside `api/`, seven more packages hold a metadata Postgres**, 73 tests
between them, classified the same way by call closure:

| Package | → `store` | → fake |
| --- | ---: | ---: |
| `component/review` | 12 | 29 |
| `runner/schemasync` | 5 | 2 |
| `runner/taskrun` | 5 | 4 |
| `runner/plancheck` | 1 | 5 |
| `server` | 2 | 2 |
| `component/recovery` | 0 | 5 |
| `enterprise` | 0 | 1 |

`component/review`'s 12 are the largest store-bound group anywhere — approval
lock ordering, concurrent approvals, staleness races — and share helpers with
their own 29, so fakes-first applies there too. `runner/schemasync`'s five hold
`AdvisoryLockKeySchemaSyncer` in a live transaction, which nothing but Postgres
can do. `plugin/db/*` and `plugin/schema/*` are out of scope: target engines,
not metadata, and effort 7's business.

Two places already have the right shape: 39 of `backend/store`'s 140 tests never
touch a database (the CEL-to-SQL builders assert generated SQL rather than
executing it), and 52 tests across the seven packages above are already
container-free.

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

**Fakes before moves.** The 18 do not move first, because their helpers are
shared with the tests that stay: 9 of the 11 helpers the 8 issue tests need are
also used by the 27 `issue_service_test.go` tests bound for fakes, so moving now
would duplicate them across two packages. Converting the 27 first lets those
helpers diverge naturally, and the 8 then move with helpers of their own. The
exception is a file whose tests all move — `audit_log_service_test.go` had two
tests and four file-local helpers, so it moved whole (**done**: now
`backend/store/audit_log_retention_test.go`, 0.14 s and 0.11 s against 4.3 s
each before). Its one unexported dependency, `convertToAuditLogs`, stayed behind
as a pure converter test with no database.

**Done anyway, duplication accepted.** The 8 issue tests moved to
`backend/store/issue_service_concurrency_test.go` with copies of the 9 shared
helpers; `api/v1` keeps its own for the 27 that stay, which get rewritten against
a fake regardless. All 8 run in 0.04–0.07 s against 4.3 s. Two snags, neither
needing a production change: they asserted on `IssueService.bus`, unexported, so
the helper now returns the bus it built; and `errDraftIssueNotSubmitted` is
unexported, but `IsDraftIssueNotSubmittedError` sits exported beside it.

Five of the 18 call unexported `api/v1` methods — `getAndVerifyUser`,
`verifyEmailCode`, `completeMFALogin`, `getOrCreateUserWithIDP` — and cannot
move without exporting them. `TestLoginAttemptRetentionOutlivesLockouts` is not
a database test at all; it compares two constants.

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
