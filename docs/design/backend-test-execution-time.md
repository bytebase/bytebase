# Backend test execution time

Status: in progress

## Scope

Cut the execution time of `go test ./backend/...`. It takes 14–19.5 minutes in
CI and has grown 54% since May (p50 10.2 → 15.7 min).

Runner capacity and queue time are out of scope. The self-hosted VM exists
because the Bytebase binary is expensive to build, and the warm Go build cache
is the point.

## Measurements

Everything below is one local pass of `go test -p=8 -json ./backend/...` on
20 vCPU / 31 GB with warm build and image caches. **The run this design started
from finished in 14 m 19 s**, with individual package wall times summing to
**3180 s**, so packages overlapped about 3.7×. Those are the numbers the efforts
were sized against; see "Where it stands now" for what the same command costs
today.

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

### Where it stands now

Same command, same box, with efforts 1, 2, 6 and 7 partly landed: **272 s**
(4 m 32 s), package walls summing to 736 s and overlapping 2.7×. Every package
passes.

| Package | Then | Now | What moved it |
| --- | ---: | ---: | --- |
| `backend/tests` | 635 s | 174 s | effort 1, then effort 2's `t.Parallel()` |
| `backend/api/v1` | 540 s | 43 s | #21349, a shared Postgres — not this design |
| `backend/store` | 459 s | 12 s | effort 2, both steps |
| `backend/plugin/schema/oracle` | 315 s | 78 s | #21344 goldens, #21351 shared container |
| `backend/component/review` | 282 s | 17 s | #21349 |
| `backend/migrator` | 90 s | <2 s | effort 6, by deletion |

Two rows are worth reading twice. `api/v1` fell 12× without anyone writing a
fake, which removes the wall-clock argument for effort 6 — the case for seams
there is now about what the tests say, not what they cost. And
`plugin/schema/*` as a whole went from 475 s to 164 s, `plugin/db/*` from 339 s
to 145 s, which shrinks effort 7 from the largest item here to a middling one.

### The floor

`backend/plugin/...` cost 866 s across 69 packages at the start, and split by
who owns the code: `plugin/schema/*` was 475 s in 6 packages, `plugin/db/*`
339 s in 18. The other 45 packages — pure parser and advisor tests, no
containers — came to 51 s. Parsing is not what costs; engines are. Today the
same 70 packages come to 358 s.

What any of this buys follows one rule:

```
wall ≈ max( slowest single package , sum of package walls ÷ overlap )
```

It reproduced the starting run — `max(635, 3180 ÷ 3.7)` is 859 s against an
actual 859 s — and it reproduces the current one: `max(174, 736 ÷ 2.7)` is
273 s against an actual 272 s. Treat the overlap factor as an observed property
of a given schedule, not a constant.

**The gate has moved, and that is the most important thing on this page.** The
run used to be bound by its slowest package, so only `backend/tests` was worth
attacking. It is now bound by the sum term: 736 s of work over an overlap of
2.7, with the slowest package at 174 s and nowhere near binding. Every second
removed anywhere now buys about 0.37 s of wall, and no single package is the
gate. That makes the remaining efforts additive rather than a queue behind one
package — and it makes work outside `backend/tests` worth doing again.

## Design

Six efforts. The numbering is the order they were written in, kept so existing
references resolve, and it is no longer the order of payoff — effort 3, a
checkout pool for the 104 `provisionPgInstance` callers, is dropped outright, and
four of the rest have been re-sized since. **Read the sizing line at the top of
each section, not its position.**

What changed is that effort 2's second step overtook everything else. Adding
`t.Parallel()` to 34 tests took `backend/tests` from 615 s to a median 173 s on
its own — more than efforts 1, 4 and 5 together were ever projected to buy — and
by making the run sum-bound rather than bound by one package it changed what the
others are worth. Efforts 4, 6 and 7 deflated: their costs now overlap, or were
already collected by work that landed outside this design. Effort 5 did not,
because deleting a test removes work instead of moving it.

The standing order, by what is left rather than by number:

| | Effort | Worth |
| --- | --- | --- |
| 1 | 5, drop the duplicated MySQL bodies | ~320 s of work, ~2 min of suite wall |
| 2 | `TestWebhookIntegration` | 148 s in one test, now `backend/tests`'s floor |
| 3 | 7, engine conformance to omni | 164 s, behind a large ownership blocker |
| 4 | 6, seams instead of containers | seconds; do it for the tests, not the clock |
| 5 | 4, isolate per project | a memory ceiling, not a speed play |

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

**Step 2 again, in `backend/tests`, and this is where the package was hiding its
time.** 175 of its 209 tests already called `t.Parallel()`. The other 34 never
did, and they summed to **483.7 s — 79% of the package's 614.7 s wall**, running
strictly one at a time, because Go defers parallel tests to the end and runs
everything else back to back. The 175 that did were compressing 1717.7 s into
about 131 s, a 13× overlap. The machinery was already there; a fifth of the
tests were not using it.

Adding `t.Parallel()` to those 34 — one line each, no other change — takes the
package from **614.7 s to a median 173 s**, over ten runs spanning 155–236 s
with no failures. The spread is real: twenty concurrent servers is a noisy
schedule, and the worst run is still 2.6× better than the best serial one.

Nothing structural was in the way. 32 of the 34 boot a server, and each already
gets its own server, workspace and database, so there is nothing to share. Every
piece of package-level state is already guarded — `nextPort` and
`nextDatabaseNumber` behind `mu`, `externalPgHost`/`Port` written once in
`TestMain` and read-only thereafter — and the package has no `t.Setenv` or
`t.Chdir` anywhere, either of which would panic under `t.Parallel()`. Of the 34,
exactly one contains a `time.Sleep` and it is inside a comment:
`waitForWebhookCount` replaced it with a poll-until-deadline. The two explicit
deadlines are 5-minute context timeouts against tests that run in about ten
seconds.

Ten of the 34 have subtests, and `tparallel`'s all-or-nothing rule forces a
choice on each. Two of them — the account email validation pair — boot a server
*inside every subtest*, so their subtests are independent and now run parallel
too. The other eight boot one server in the parent and drive it through the
subtests in order, which is `backend/tests`'s existing
`//nolint:tparallel // Subtests share one server lifecycle.` case; each now
carries that directive with its own reason. Note what this does not cost:
the parent stays parallel with the rest of the package either way, and that is
where the entire saving came from. Making `TestTransactionMode`'s four engine
subtests parallel would buy nothing, because the package floor is
`TestWebhookIntegration`, not the sum of everything else.

**`TestWebhookIntegration` is now the package floor.** At 148 s it is most of the
173 s, and the rest of the package finishes around it. It is the one place left
in `backend/tests` where a single test is worth attacking on its own.

### 4. Isolate per project, not per workspace

**Re-sized: about 7 s of wall, not 362 s.** The 362 s below is real work — 207
boots at 703 ms up and, after effort 1, ~1 ms down — but effort 2's step 2 made
it overlap. Spread across twenty workers it is worth single-digit seconds of
wall clock, and the run is no longer bound by this package anyway.

What survives is a resource argument, not a speed one: twenty concurrent servers
is what sets the memory ceiling on how far `-parallel` can go, and one server for
the whole package would lift it. Do this when a runner cannot take the
concurrency, not to buy time. The original sizing follows.

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

### 5. Postgres only for `backend/store` and `backend/tests`

**Up to 320 s inside `backend/tests`**, of which 115 s is container starts and
the rest is the duplicated test bodies. `TestSQLReviewForMySQL` costs 52.7 s
where `TestSQLReviewForPostgreSQL` costs 17.9 s for the same assertion.

Unlike effort 4, this one did not deflate. Deleting a test removes work rather
than moving it, and the run is now bound by the sum term, so the 320 s converts
at the whole-run rate of about 0.37 s of wall per second removed — roughly two
minutes off the suite. That makes this the largest remaining item on the page.

Two packages may hold a real database, and no others. `backend/store` is where a
metadata query is asserted against one, and `backend/tests` is the only package
that boots a server. Everywhere else — `api/v1` above all — the handler's
decide-and-convert half is tested over a fake or over plain data, which is effort
6; an API test does not earn a container by being an API test. Inside the two
packages that keep one, the engine is Postgres: engine dialects and DDL fidelity
belong in omni. The root `AGENTS.md` carries that second half of the rule today,
in the words "test API and workflow behavior against Postgres only".

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

**Re-sized: the seconds are already gone, the argument is not.** This section
was written against 822 s of wall clock, 611 s of it container starts. #21349
then gave `api/v1`, `component/review`, `auth`, `oauth2`, `lsp` and `mcp` a
shared Postgres through `testcontainer.MetadataMain`, and those two packages
went to 43 s and 17 s without a single fake being written. All six now sit within
a couple of seconds of the 6.9 s floor that a package holding one container
cannot go below, and `api/v1` is 43 s of which about 36 s is its own tests.

So do this for what the tests say, not what they cost. A container in `api/v1`
still buys a `*store.Store` that the assertions never look at, and effort 5 still
says the metadata database belongs in `store` and `tests`. The classification
below stands; only the payoff line has changed. `backend/api/mcp` remains the
proof it is livable: 168 tests in 19 s, 16 of its 18 files containerless.

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

**Re-sized: 164 s of package wall time, no longer the largest change here.**
#21344 replaced Oracle's migration round-trip with recorded goldens and #21351
gave each engine package one shared container, taking `plugin/schema/*` from
475 s to 164 s and Oracle from 315 s to 78 s. The ownership argument is
unaffected — these tests still assert engine fidelity, which is omni's — but the
port now buys 164 s, not 475 s, against the same large blocker described below.

The blocker is that the round-trip is expressed in Bytebase's metadata proto
(`storepb`, `plugin/schema`, `store/model`), so omni must own the metadata model
first. The ownership split above is the plan: all of `plugin/schema/*` moves,
all of `plugin/db/*` stays, because that is our driver. The figures in the rest
of this section — 475 s, 315 s for Oracle, 339 s for `plugin/db/*` — are the
pre-#21351 ones the analysis was done against; the split they describe is
unchanged, the totals are now 164 s and 145 s.

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
