# Scoping `sheet_blob` — fixing the global-by-SHA256 read

Fixes T5 in `docs/design/v1-api-audit-2026-08.md`.

**In short.** Sheet content is fetched by SHA256 with no scope predicate, so anyone holding
`bb.sheets.get` on any one project can read any sheet in the deployment — across projects and
across tenants — if they know the hash. The fix adds a `sheet_blob_ref(project, sha256)` edge
table, keeps enforcement at the API layer behind a single batched gate, and backfills ownership
from existing references. The two things most likely to go wrong are backfill completeness
(a missed source is a silent NotFound in production) and the fact that a backfilled reference
records what a project *touched*, not what it was *entitled* to. Both have explicit sections.

## Problem

`sheet_blob` is `(sha256 PRIMARY KEY, content)` — no project column, no workspace column.
`getSheet` selects `WHERE sha256 = decode(?, 'hex')` with no scope predicate
(`backend/store/sheet.go:60-71`), while the ACL authorizes `bb.sheets.get` against the
caller-named project, correctly pinned to the caller's workspace
(`backend/api/v1/acl.go:397`).

Authorization is per-tenant; the data fetch is deployment-global. Anyone holding
`bb.sheets.get` on any one project reads any blob in the deployment, across projects and
across workspaces, given the hash.

This is a regression, not an oversight. Commit `bee2080737` (#18552, Dec 2025) dropped the
project-scoped `sheet` table — which carried `project text NOT NULL REFERENCES
project(resource_id)` — and promoted the `sheet_blob` dedup side-table to primary store. Its
design doc's Non-Goals recorded "Change authorization model (already project-scoped, no
changes needed)", which was true before the cutover and false after. The assumption is still
in the tree as a justification for skipping checks, at `backend/api/v1/release_service.go:331`
and `:405`.

## The model is per-project

This is not a new decision — it is the model the rest of the code already implements, and
sheets are the one thing that fell out of it.

`validateSpecs` rejects a release whose project differs from the plan's project
(`backend/api/v1/plan_service.go:759`). Three lines above, the sheet check is
`HasSheets(ctx, sheetSha256s...)` with no project, and `:642` discards the project segment of
a caller-supplied sheet name outright. Same function, two resources, opposite treatment. The
dropped `sheet` table carried `project text NOT NULL REFERENCES project(resource_id)`, and the
resource name is still `projects/{project}/sheets/{sha}`.

So the fix restores project scoping rather than inventing a scope.

Scoping on `project` alone is sufficient for tenant isolation too: `project.resource_id` is a
global primary key carrying a `workspace` column, and the ACL already pins the caller-named
project to the caller's workspace (`backend/api/v1/acl.go:397`). One predicate closes both
boundaries.

## Design decision: an ownership edge, not a column

Adding `project` directly to `sheet_blob` would break the content-addressed dedup that
motivated the refactor — the same statement in two projects would need two rows, and the
migration would have to split existing shared rows.

Instead, keep `sheet_blob` as pure content storage and add an edge table recording which
projects may read a given hash. Two projects that independently author identical SQL share one
blob and hold one ref row each. Dedup survives; "who can read this hash" becomes an explicit
stored fact rather than an implicit one.

## Schema and migration

### Table

`backend/migrator/migration/3.22/0005##scope_sheet_blob.sql` (current head is 3.22.4;
`TestLatestVersion` in `backend/migrator/migrator_test.go` needs updating).

```sql
CREATE TABLE sheet_blob_ref (
    project text NOT NULL REFERENCES project(resource_id),
    sha256 bytea NOT NULL REFERENCES sheet_blob(sha256),
    PRIMARY KEY (project, sha256)
);
```

Mirror into `backend/migrator/migration/LATEST.sql`.

### Project purge

The `project` foreign key makes `sheet_blob_ref` a dependent of `project`, and project purge
ends with `DELETE FROM project WHERE resource_id = ?` (`backend/store/project.go:826`) after
clearing every other dependent table. Add

```sql
DELETE FROM sheet_blob_ref WHERE project = ?
```

to that sequence at the position established under transaction lock ordering below — after the
`db` delete (`:704`) and before `project_webhook` (`:739`). Without it, purging any project fails
on the foreign key, a hard breakage of a shipped path.

This delete is not sufficient on its own. Purge also reassigns workspace-instance databases to
the default project, and their revisions survive with their hashes, so the refs must be carried
to the destination *before* this delete runs or that history becomes unreadable. See "Purge is
the ordering trap" under database project reassignment.

Blobs whose only ref belonged to the purged project are left behind with zero refs. That is
consistent with today's behavior (nothing has ever deleted from `sheet_blob`) and is the state
a future GC would collect; see the GC note under independent fixes.

### Backfill

Derive `(project, sha256)` from every surviving reference. All of them live inside JSONB
payloads, and `protojson` camelCases the keys — `sheetSha256`, not `sheet_sha256`.

The authoritative source list is `grep -rn "sheet_sha256" proto/store/`, which returns exactly
five messages. Re-run it before writing the migration rather than trusting this table — and for
each hit, confirm two things the grep does not tell you: the **column name** in `LATEST.sql`
(it is not always `payload`) and the **nesting** of the field inside its message (it is not
always top-level).

| Source | Column | Path to the hash | Route to project |
|---|---|---|---|
| `plan` | `config` | `specs[].changeDatabaseConfig.sheetSha256` | `plan.project` |
| `task` | `payload` | `sheetSha256` | `task.project` |
| `release` | `payload` | `files[].sheetSha256` | `release.project` |
| `plan_check_run` | `result` | `results[].sheetSha256` | `plan_check_run.project` |
| `revision` | `payload` | `sheetSha256` | `db(instance, db_name)` → `db.project` |

`plan_check_run` is the one that punishes assumption: the column is `result`, not `payload`, and
the hash lives on `PlanCheckRunResult.Result` inside the repeated `results` array, so the path is
`result->'results'` with a `jsonb_array_elements` expansion — the same shape as `plan` and
`release`, not the flat shape of `task` and `revision`.

The first four carry `project` directly. Only `revision` needs the join through `db`, because
it keys on `(instance, db_name)` and inherits its project from the database.

Two tables that look like sources are not. `ChangelogPayload` carries only `task_run` and
`git_commit` — a changelog reaches SQL transitively through its task run, and the v1
`Changelog` message exposes no statement or sheet field. `issue_comment` has no sheet reference
of any kind.

Representative statement for the plan source; the others follow the same shape and are
`UNION`ed into one insert:

```sql
INSERT INTO sheet_blob_ref (project, sha256)
SELECT DISTINCT pl.project, decode(x.sha, 'hex')
FROM plan pl
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(pl.config->'specs', '[]'::jsonb)) spec
CROSS JOIN LATERAL (SELECT spec->'changeDatabaseConfig'->>'sheetSha256' AS sha) x
WHERE x.sha IS NOT NULL
  AND EXISTS (SELECT 1 FROM sheet_blob b WHERE b.sha256 = decode(x.sha, 'hex'))
ON CONFLICT DO NOTHING;
```

The `EXISTS` guard keeps the foreign key satisfiable if any payload ever names a hash with no
blob.

**Known gap.** A blob referenced by nothing has no derivable project — there is no information
anywhere that ties it to one. Those rows get no ref and become unreadable. In practice this is
a sheet created but never attached to a plan or release; the UI attaches immediately, and
`createSheet` caches the content client-side (`frontend/src/stores/app/sheet.ts:91`) so a fresh
draft does not re-fetch. The alternative — leaving orphans globally readable — reproduces the
bug, so accept the gap and note it in the migration comment.

**Observed references are not authorization.** This is the sharpest limitation of the whole
approach and it needs a decision before the migration runs.

The backfill reconstructs which project *referenced* a hash. That is not the same as which
project was *entitled* to it, and today's unscoped paths let the two diverge.
`convertPlanSpecChangeDatabaseConfig` (`backend/api/v1/plan_service.go:1157`) parses a
caller-supplied `projects/{project}/sheets/{sha}` and stores only the hash, discarding the
project; `HasSheets` currently validates existence with no scope. So a plan in project B can
already hold a reference to a hash owned by project A, and after the write nothing distinguishes
it from a legitimate one. Backfilling `(B, sha)` would mint a permanent grant out of exactly the
access this change exists to close.

Shadow mode does not catch this. The ref row exists, so the gate neither logs nor denies — it
is invisible to the safety net described below.

There is no clean automated fix: the refactor that created this bug deleted the sheet's creator
and project metadata, so no ground truth for "who owned this hash" survives anywhere. What can
be done is to size the population and decide deliberately:

```sql
SELECT sha256, count(DISTINCT project) AS projects, array_agg(DISTINCT project)
FROM sheet_blob_ref
GROUP BY sha256
HAVING count(DISTINCT project) > 1;
```

Run this immediately after the backfill, before enforcement. A hash claimed by more than one
project is either honest dedup — two projects independently authored identical SQL, and both
claims are real — or a laundered cross-project reference. They are indistinguishable from the
data, so this is a review list, not an automatic filter. In most deployments it should be short
or empty; if it is long, that is itself a finding worth reporting before proceeding.

One timing note: this design is public on the PR, including the rule that a reference becomes an
ownership edge. Between publication and the migration there is a window in which references could
be planted deliberately. Capture the multi-project list at a known-good point rather than only
after upgrade.

## Store and API surface

### Store API

`sheet_blob_ref` is an ACL fact, enforced at the API layer. Content retrieval and access
control are separate operations and the store API keeps them separate.

The runners are the reason this distinction is real rather than cosmetic. When
`database_migrate_executor` reads a sheet, it is executing work that was authorized when the
plan was created; it is not making an access-control decision. Threading a project through it
purely to satisfy a signature would be authorization theater.

Every primitive is set-shaped. Requests routinely carry many sheets — a release can have
hundreds of files — so nothing here may be O(n) in round-trips. See "Batch shape" below.

```go
// Content only. A hash fully determines the content, so no scope is involved.
// Returns a map keyed by hex hash; an absent key means no such blob.
func (s *Store) GetSheetsFull(ctx context.Context, sha256Hexes ...string) (map[string]*SheetMessage, error)
func (s *Store) GetSheetsTruncated(ctx context.Context, sha256Hexes ...string) (map[string]*SheetMessage, error)

// Scoped: which of these hashes may this project read? One query.
func (s *Store) FilterSheetsForProject(ctx context.Context, projectID string, sha256Hexes ...string) (map[string]bool, error)

// Scoped: a validation predicate, not a content read. Already batched today.
func (s *Store) HasSheets(ctx context.Context, projectID string, sha256Hexes ...string) (bool, error)

// Scoped: writes blobs and ref rows, both as set inserts.
func (s *Store) CreateSheets(ctx context.Context, projectID string, creates ...*SheetMessage) ([]*SheetMessage, error)
```

The singular `GetSheetFull`/`GetSheetTruncated` can stay as thin wrappers for the genuinely
single-sheet callers in the runners, but the batch form is the primitive.

`HasSheets` gains the join and stops being a deployment-wide existence oracle:

```sql
SELECT COUNT(*)
FROM sheet_blob b
JOIN sheet_blob_ref r ON r.sha256 = b.sha256
WHERE b.sha256 IN (...) AND r.project = ?
```

`CreateSheets` writes blobs and ref rows in one transaction. It currently issues a bare
`ExecContext`; it needs a transaction so a blob cannot land without its ref. The ref insert
mirrors the existing blob insert's array shape — one statement, not a loop:

```sql
INSERT INTO sheet_blob_ref (project, sha256)
SELECT ?, unnest(CAST(? AS BYTEA[]))
ON CONFLICT DO NOTHING
```

Three of its four call sites have a project to hand — `sheet_service.go:54` and `:97` (the
resolved parent) and `release_service.go:96`. The fourth does not:
`rollout_service_task.go:143` stores generated `CREATE DATABASE` boilerplate from inside
`getTaskCreatesFromCreateDatabaseConfig(ctx, s *store.Store, spec, c)`, which takes no project,
and neither does its caller `getTaskCreatesFromSpec`. Thread it down from
`rollout_service.go:1279`, which has the plan's project — two signature changes.

### Cache

`sheetFullCache` stays keyed by hex hash alone (`backend/store/store.go:51,82,110`, a 10-entry
LRU). Because content is a pure function of the hash, a hash-keyed content cache is always
correct, and the dominant consumer — a runner reading the same sheet repeatedly — keeps full
hit rate.

This is safe only because the ACL check is a separate step that runs first. The cache read at
`backend/store/sheet.go:43` returns before any query executes, so a scope predicate placed
inside `getSheet` would be skipped on a cache hit. Enforcement must not live there.

Ten entries is sized for the runner pattern — one sheet read repeatedly — not for batch reads,
and a release with hundreds of files will miss on nearly all of them. That is fine and not a
reason to grow it: the batch path collapses those misses into a single query, so the cache is an
optimization for the repeated-single-sheet case rather than load-bearing for throughput.

### API-layer gate

Moving enforcement out of the store means the compiler no longer finds missed call sites — and
a forgotten check is exactly how `release_service.go:398` and `plan_service.go:642` ended up
unscoped. Replace the compiler with structure:

Add one helper in `backend/api/v1/` that does check-then-fetch. It has to be a package-level
function, not a method — three of the four routed call sites live in `ReleaseService` and
`RolloutService`, not `SheetService` — and it takes a set, because its heaviest caller iterates
over release files:

```go
func sheetsForProject(ctx context.Context, s *store.Store, projectID string, sha256Hexes []string, raw bool) (map[string]*store.SheetMessage, error)
```

It filters the requested hashes through `sheet_blob_ref` for `projectID`, and any hash that does
not survive yields NotFound — NotFound rather than PermissionDenied, so the response does not
confirm that a hash exists somewhere else. Only the surviving hashes reach the content getter.
Every v1 sheet read routes through it:

- `backend/api/v1/sheet_service.go:136,138` — `project.ResourceID`, already resolved and
  workspace-checked at `:119-131`
- `backend/api/v1/release_service.go:398` — where `validateAndSanitizeReleaseFiles` currently
  discards the project outright
- `backend/api/v1/rollout_service.go:1233`, `rollout_service_task.go:463` — the resolved project

`HasSheets` call sites pass a project directly: `release_service.go:144`,
`revision_service.go:158` (`database.ProjectID`), `plan_service.go:746` (`plan.ProjectID` — the
same value already used to validate the release at `:759`).

Runner and component call sites keep the unscoped content getters:
`backend/runner/taskrun/*`, `backend/runner/plancheck/*`, `backend/component/review/evaluator.go:710`.

Hold the line with a test asserting that nothing under `backend/api/v1/` calls any store sheet
content getter directly except the gate helper. Six call sites today, so it is cheap.

### Batch shape

The gate must cost a bounded number of round-trips regardless of how many sheets a request
carries. A release with two hundred files must not produce two hundred queries — still less four
hundred, which a naive check-then-fetch per file would give.

`sheetsForProject` is therefore two queries, whatever the input size:

1. **One ref query** returning the subset of requested hashes readable by this project:

   ```sql
   SELECT encode(sha256, 'hex')
   FROM sheet_blob_ref
   WHERE project = ?
     AND sha256 IN (SELECT decode(unnest(CAST(? AS TEXT[])), 'hex'))
   ```

   Anything requested but absent from the result is NotFound. This runs first, which is what
   preserves the cache-ordering invariant above.

2. **Cache lookups** for the surviving hashes, then **one content query** for the misses, using
   the same `unnest` shape.

Note that fusing both into a single join would be tempting and is wrong: it would make the
content read the same statement as the permission check, which is exactly the arrangement that
lets a cache hit skip the check. Keeping them as two steps is deliberate.

Three existing call patterns need reshaping to feed it:

- **`validateAndSanitizeReleaseFiles`** (`release_service.go:377-413`) calls `GetSheetFull` once
  per file inside its loop. It already has all the files up front, so hoist a single collect pass
  for every `f.Sheet`, one `sheetsForProject` call, then loop over the map. This path is O(n)
  today, before this change — the fix is not new debt, it is a pre-existing N+1 that the scope
  check would otherwise double.
- **`rollout_service_task.go:214`** calls `getSheetContentBySha256(ctx, s, c.SheetSha256)` inside
  a loop over target databases, but `c.SheetSha256` is loop-invariant — a spec targeting a
  hundred databases fetches the same hash a hundred times. Hoist it above the loop. Only the
  10-entry LRU keeps this from being a hundred queries today, which is a fragile reason for it to
  be fast.
- **`plan_service.go:746`** is already correct and worth copying: it collects `sheetSha256s`
  across every spec and makes one `HasSheets` call.

The write paths are already set-shaped and stay that way: `CreateSheets` uses array `unnest` for
both inserts, the backfill is `INSERT ... SELECT`, and the reassignment helper below is a single
`INSERT ... SELECT` rather than a per-revision loop.

## Database project reassignment

The case where a hash legitimately needs to become readable under a second project — and the
easiest part of this design to get wrong, because it happens in more places than it looks.

`revision` has no project column: it keys on `(instance, db_name)` and inherits its project
through `db`. So whenever a database's `project` changes, its revision history silently changes
project too, and `convertToRevision` then formats the sheet name with the *new* project
(`backend/api/v1/revision_service.go:304`). Under a strict check, reads that worked before the
move start returning NotFound.

Revisions are the only row type affected. `changelog` also keys on `(instance, db_name)` but
carries no sheet reference at all, so it moves without implicating any hash.

### The invariant

**Whenever a database's project changes, carry its revision sheet refs to the destination project
in the same transaction, before the change commits.**

State it as an invariant rather than patching call sites, because there are four of them and a
fifth is easy to add later without noticing:

| Path | Location | What moves |
|---|---|---|
| `UpdateDatabase` | `backend/store/database.go:332-380` | one database |
| `BatchUpdateDatabases` | `backend/store/database.go:397-560` | a batch of databases |
| `updateInstanceLifecycle` | `backend/store/instance.go:438-455` | every database of an instance, when a project instance is archived |
| `DeleteProject` | `backend/store/project.go:722-729` | workspace-instance databases, reassigned to the default project |

Only the second is an explicit user-facing "transfer". The other three are consequences of other
operations, which is precisely why enumerating them matters — an earlier draft of this section
covered `BatchUpdateDatabases` alone and would have broken revision history on the other three.

### One helper, four callers

Extract a single store-internal helper — `carryRevisionSheetRefs(ctx, tx, destProject, databases)`
— and call it from all four. One statement, no loop over revisions, and it stays one statement
for a batch:

```sql
INSERT INTO sheet_blob_ref (project, sha256)
SELECT ?, decode(r.payload->>'sheetSha256', 'hex')
FROM revision r
WHERE (r.instance, r.db_name) IN (...)
  AND r.payload->>'sheetSha256' IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM sheet_blob b
    WHERE b.sha256 = decode(r.payload->>'sheetSha256', 'hex')
  )
ON CONFLICT DO NOTHING
```

The `EXISTS` guard is the same one the backfill uses, and it matters more here. A revision naming
a hash with no blob would otherwise violate the new foreign key and abort the entire enclosing
operation — turning a scoping change into a regression on database transfer, instance archive,
and project purge, none of which read sheet blobs today or have any reason to start failing.

Ref rows for the source project stay — the content was genuinely authored there.

### Purge is the ordering trap

`DeleteProject` both reassigns databases *and* deletes the purged project's refs, so the two
steps interact. The reassignment (`project.go:722-729`) moves only workspace-instance databases
to the default project; project-instance databases were already deleted with their instance, and
the `revision` delete at `:702` only covers project instances — so revisions on workspace
instances survive the purge with their hashes intact.

If the ref delete runs first, those surviving revisions lose their only ref and never gain a
replacement: listing revision history for a database that outlived its project returns NotFound
for sheets that read fine a moment earlier. **Carry the refs, then delete.**

Fusing the two into one statement removes the chance of the predicates drifting apart:

```sql
WITH moved AS (
  UPDATE db SET project = ?
  FROM instance
  WHERE db.instance = instance.resource_id
    AND db.project = ?
    AND instance.project IS NULL
  RETURNING db.instance, db.name
)
INSERT INTO sheet_blob_ref (project, sha256)
SELECT ?, decode(r.payload->>'sheetSha256', 'hex')
FROM revision r
JOIN moved m ON m.instance = r.instance AND m.name = r.db_name
WHERE r.payload->>'sheetSha256' IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM sheet_blob b
    WHERE b.sha256 = decode(r.payload->>'sheetSha256', 'hex')
  )
ON CONFLICT DO NOTHING;
```

Then `DELETE FROM sheet_blob_ref WHERE project = ?` for the purged project.

This keeps the resource name honest rather than weakening the predicate to match the looser
behavior.

## Transaction lock ordering

Two new multi-table write paths fall under the canonical ordering in
`backend/store/README.md#transaction-row-lock-ordering`, which `AGENTS.md` requires be settled
before the code is written.

### Position in the canonical sibling order

The README requires a new project-owned branch to establish its position in the canonical order
before implementation, and to update that list, `DeleteProject`, and `DeleteInstance` together.
`sheet_blob_ref` is a direct child of `project` with no descendants and no dependents, and every
table that can reference a hash (`plan`, `task`, `release`, `plan_check_run`, `revision`) sits
earlier in the list. Place it immediately after `db`:

```text
... -> changelog -> sync_history -> revision -> db_schema -> db
-> sheet_blob_ref -> project_webhook -> service_account -> ...
```

That is also where the purge delete belongs — between the `db` delete
(`backend/store/project.go:704`) and `project_webhook` (`:739`) — so the purge sequence and the
canonical list stay consistent.

### Lifecycle policy

`sheet_blob_ref` is purge-managed data, so the README requires its writers to state whether they
need an **active** project or merely an **existing** one, and to serialize that against project
deletion.

**`CreateSheets` requires an active project.** A sheet ref is a new resource; there is no
deleted-project continuation case for it. The API create paths check the project before calling
the store, but that check is not serialized with purge — a concurrent purge between the check and
the insert would surface as a raw foreign-key violation rather than a controlled NotFound. So
`CreateSheets` takes the same transaction-scoped purge fence before any row lock that database
creation, batch database updates, and task-run creation already take.

There is a precedent to mirror rather than invent: `withDatabasePurgeFence`
(`backend/store/database.go:798-820`) is exactly this pattern for database writers — it resolves
ownership, locks every project fence `FOR UPDATE`, and only then runs the write.

The reassignment paths need nothing extra, because all four already hold that lock before any ref
insert would run:

| Path | Project lock |
|---|---|
| `UpdateDatabase` | `withDatabasePurgeFence` (`database.go:808-816`) |
| `BatchUpdateDatabases` | `database.go:509-517` |
| `updateInstanceLifecycle` | `instance.go:444-451` |
| `DeleteProject` | the purge transaction itself |

Each fails NotFound on a missing project, so the controlled outcome is already in place.

### Ordering of the write paths

- **`CreateSheets`** spans `sheet_blob` and `sheet_blob_ref`. Insert the blob first, then the
  ref: the ref's foreign key requires the blob to exist, so the order is forced. Both are
  `ON CONFLICT DO NOTHING` — new-row-only inserts, not upserts that can update an existing row,
  so the README's rule 4 upsert clause does not apply to them. The foreign-key checks on
  `project` and `sheet_blob` do count as locks and are covered by the purge fence above.
- **The reassignment paths** insert refs inside a transaction that has already taken
  `FOR UPDATE` on every involved project. Note this is *not* an instance of the child-to-parent
  rule — that rule governs locking **existing** rows, and these refs are new rows with nothing to
  lock beforehand. The insert is safe because the transaction already holds a stronger lock on the
  parent than the foreign-key check needs, not because parent-then-child is the required order. It
  is not; an earlier draft of this doc claimed otherwise.

Both need the deterministic real-PostgreSQL regression tests that section mandates, asserting
terminal outcomes in both lock-acquisition directions rather than merely the absence of
SQLSTATE `40P01`, and covering the create-versus-purge race named above.

## Rollout safety

This is a read-path change to a shipped multi-tenant product whose correctness rests entirely on
backfill completeness, and completeness cannot be established by construction. A missed source
means sheets silently return NotFound in production with no signal.

Treat that as a live risk rather than a hypothetical. Two drafts of this document got the source
list wrong — the first invented two sources that do not exist, the second named the wrong column
and nesting for `plan_check_run` — and a draft of the reassignment section covered one of its
four paths. Each would have shipped as silent NotFounds. The list is now verified; the failure
mode it illustrates is the one to design against.

**Ship the gate in shadow mode.** It evaluates the ref check and logs a miss — project, hash,
call site — but still returns the content. Run it for a full release. Flip to denying only once
the log is clean.

**Verify the backfill before that.** After the migration, for each of the five sources, compare
the count of distinct `(project, sha256)` pairs derivable from it against the ref rows present,
and count blobs with zero refs:

```sql
SELECT count(*) FROM sheet_blob b
WHERE NOT EXISTS (SELECT 1 FROM sheet_blob_ref r WHERE r.sha256 = b.sha256);
```

A non-zero result is expected — it is the orphan-draft population noted under the backfill — but
it should be small and should not grow once the shadow period begins. A large number means a
source was missed.

## Independent fixes, no schema needed

These are worth landing separately and first — none needs a migration.

1. **The T6 hash echo.** `backend/api/v1/revision_service.go:203` takes `projectID` from the
   attacker-supplied `revision.File` and uses it as a store key without comparing it to
   `database.ProjectID`. `FindReleaseMessage` has no `Workspace` field
   (`backend/store/release.go:30-37`), so this reaches any project in any workspace, and `:223`
   echoes the real hash. Add the comparison; drop the hash from the error string. The same
   missing comparison appears for `revision.TaskRun` at `:167-186`.
2. **Delete the two stale comments** at `release_service.go:331` and `:405`
   (`// Sheets are now project-agnostic, no need to check projectID`). They are now false and
   will mislead the next reader into repeating the mistake.
3. **Add audit annotations** to `proto/v1/v1/sheet_service.proto`. It declares none, while 20
   other v1 services do — so sheet reads and enumeration are currently unlogged.
4. **Drop `bb.sheets.update`.** Granted to four roles in `backend/store/predefined_roles.go`,
   declared by no RPC; dead since sheets became immutable.
5. **Comment the GC hazard on `sheet_blob`.** Nothing deletes from it today — not the cleaner,
   not project purge, and there is no `DeleteSheet` RPC. The only foreign key that ever pointed
   at it (`task_run.sheet_sha256`, migration 3.14) was dropped in `7192406c5f`, so every
   surviving reference now lives in JSONB and is invisible to referential integrity. A future GC
   written the obvious way — "delete blobs no FK points at" — would empty the table.
   `sheet_blob_ref` makes a correct GC possible later, but that is out of scope here.

## Tests

- **Cross-project read** — seed a blob in project A, request it under project B, expect
  NotFound. This is the regression test for the whole finding. Run it with A and B in the same
  workspace *and* in different workspaces; the second is the tenant-isolation case.
- **Cache ordering** — request a blob under its owning project to warm the cache, then request
  the same hash under a foreign project and assert NotFound. This is the test that fails if
  enforcement ever migrates back into `getSheet` behind the cache read.
- **Gate coverage** — assert nothing under `backend/api/v1/` calls a store sheet content getter
  outside the gate helper.
- **Query count** — create a release with many files and assert the gate issues a bounded number
  of queries, not one per file. A count assertion is the only thing that actually holds the batch
  shape; a correctness test passes just as happily against an N+1 implementation.
- **Backfill** — build a metadata DB exercising each of the five reference sources, run the
  migration, assert every referenced hash resolves under its project and no unreferenced hash
  does.
- **Project purge** — purge a project that owns sheet refs and assert it succeeds. This is the
  direct regression test for the new foreign key.
- **Reassignment, one case per path** — the invariant has four callers and each needs its own
  test, since the failure is silent and only shows up on a later read:
  - `UpdateDatabase` and `BatchUpdateDatabases` — create a revision under project A, move the
    database to project B, assert the statement is still readable under B.
  - `updateInstanceLifecycle` — archive a project instance with a destination project, assert
    revision history stays readable under the destination.
  - `DeleteProject` — put a database on a *workspace* instance in project A, create a revision,
    purge project A, then assert the statement is still readable under the default project the
    database was reassigned to. This is the ordering trap: it passes if refs are carried before
    the source delete and fails if they are not.
- **Blobless revision** — a revision naming a hash with no blob must not abort any of the four
  reassignment paths, covering the `EXISTS` guard.
- **Create versus purge** — the deterministic race required by the lifecycle policy: a
  `CreateSheets` concurrent with a purge of its project must end in a controlled NotFound, not a
  foreign-key error, in both lock-acquisition directions.
- **Multi-project hashes** — seed a plan in project B referencing a hash created in project A
  (the pre-existing unscoped path), run the backfill, and assert the audit query reports it.
  The point is that the row is *detected*, not that it is silently blessed.
- **`HasSheets`** no longer answers for a foreign project's hash.
- **`TestCollision_Sheet`** in `backend/tests/`, per the composite-PK convention in `AGENTS.md`.
  Note that the shared `setupCollidingProjects` fixture and `assertProjectUnchanged` helper do
  not currently cover sheets — extend them first rather than writing a table-specific variant.
- **Lock ordering** — the deterministic real-PostgreSQL tests required by the section above, for
  `CreateSheets` and for the reassignment paths.

## Order

1. The five no-schema fixes above — T6 first, since it is the cheapest way to obtain a hash.
2. Migration, backfill, purge delete, backfill verification query.
3. Run the multi-project audit query and review its output. This is a gate, not a report: a long
   list means cross-project references are already widespread and the plan needs revisiting
   before enforcement, since the backfill would otherwise make them permanent.
4. Store API split, gate helper, call sites, the reassignment helper and all four of its callers,
   tests — gate shipping in shadow mode.
5. Flip the gate to denying, one release later, once the shadow log is clean.
