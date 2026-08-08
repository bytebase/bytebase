# Scoping `sheet_blob` to its project

Fixes T5 in `docs/design/v1-api-audit-2026-08.md`.

Sheet content is fetched by SHA256 with no scope predicate, so any principal holding
`bb.sheets.get` on any one project can read any sheet in the deployment — across projects and
across tenants — given the hash. A `sheet_blob_ref(project, sha256)` edge table makes ownership an
explicit stored fact, and the store's project-scoped sheet accessors enforce it.

## Background

`sheet_blob` is `(sha256 PRIMARY KEY, content)` — no project column, no workspace column.
`getSheet` selects `WHERE sha256 = decode(?, 'hex')` with no scope predicate
(`backend/store/sheet.go:60-71`), while the ACL authorizes `bb.sheets.get` against the
caller-named project, correctly pinned to the caller's workspace (`backend/api/v1/acl.go:397`).
Authorization is per-tenant; the data fetch is deployment-global.

This is a regression, not an oversight. Commit `bee2080737` (#18552, Dec 2025) dropped the
project-scoped `sheet` table — which carried `project text NOT NULL REFERENCES
project(resource_id)` and `idx_sheet_project` — and promoted the `sheet_blob` dedup side-table to
primary store. Its design doc recorded "Change authorization model (already project-scoped, no
changes needed)" as a non-goal, true before the cutover and false after. The assumption survived in
the tree as stale comments justifying skipped checks in `release_service.go` until this work
deleted them.

## Requirements

**R1 — Close cross-tenant read.** A principal in workspace W1 must not read sheet content belonging
to workspace W2. In Bytebase Cloud this is tenant isolation; it is the severe half of the finding
and is non-negotiable.

**R2 — Preserve authoring governance.** Customers restrict who may author change SQL under a given
project. `CreateSheet` is gated on `bb.sheets.create` against the parent project, and that must
remain expressible.

**R3 — Preserve content-addressed dedup.** Identical SQL must not be stored twice. This was the
point of the refactor that introduced the bug and must not be undone to fix it.

**R4 — No silent breakage of existing reads.** The fix rests on a backfill whose completeness
cannot be established by construction. A missed reference source must not become a production
NotFound with no signal.

**R5 — Bounded round-trips.** Requests routinely carry many sheets; a release can have hundreds of
files. The scope check must not be O(n) in queries.

**R6 — Projects remain a confidentiality boundary.** Many customers separate projects specifically
so teams cannot read each other's SQL — regulated separation where a PCI- or HIPAA-scoped project
is walled off and audited, and agency or multi-end-customer workspaces where one workspace holds
databases belonging to different clients.

**R7 — The resource name states the truth.** Across this API the prefix in a resource name
identifies the resource's actual owner, never a scope the caller selected. A scope that disagrees
with the name is the defect being fixed, not a fix for it.

## Decision

**A sheet is owned by a project.** The resource is the pair (project, content): two projects that
independently author identical SQL own two distinct sheets that happen to share storage.
Content-addressed dedup becomes a storage optimization invisible at the API layer.

Everything else follows from that one commitment. `projects/{project}/sheets/{sha}` is an honest
name, `parent = projects/{project}` on `CreateSheet` is a real parent, project-level
`bb.sheets.create` is coherent, read is project-scoped, and no part of the public contract changes.

The alternative — treating the resource as the content, shared tenant-wide — is examined under
[Alternatives](#alternatives-considered). It fails R2 and R6 and requires a breaking rename.

### Why the name forces this

The API has one consistent rule and it is unbroken. Project-owned resources use a single
project-prefixed pattern: `AccessGrant` (`projects/{project}/accessGrants/{access_grant}`),
`DatabaseGroup`, `Plan`, `PlanCheckRun`, `Issue`, `Release`. Workspace-global resources use a bare
collection with the workspace implicit from the JWT: `Group` (`groups/{group}`), `IdP`
(`idps/{idp}`), `ReviewConfig` (`reviewConfigs/{reviewConfig}`). Resources with two legitimate
addressings declare both patterns — `Database`, `DatabaseChangelog`, `DatabaseCatalog`,
`DatabaseMetadata`, `InstanceRole`, `Instance` all carry `instances/{instance}/…` alongside
`projects/{project}/instances/{instance}/…`, and `Policy` carries three.

In every case the prefix names the **actual owner**. A project instance really is owned by that
project; a database's `projects/{p}/…` form names the project in `db.project`, not one the caller
picked. Nowhere else in this API does a name prefix identify a caller-supplied scope. `Sheet`
declaring `projects/{project}/sheets/{sheet}` while the lookup ignores the project *is* T5, stated
as a naming defect rather than an authorization one.

The create side is equally strict: a create request's `parent` is always exactly the prefix of the
created resource's name. Workspace-scoped resources therefore have no parent at all —
`CreateGroup` is `POST /v1/groups`, `CreateReviewConfig` is `POST /v1/reviewConfigs`,
`CreateIdentityProvider` is `POST /v1/idps`. `CreatePolicyRequest.parent` is `workspaces/{workspace}`
or `environments/{environment}` or `projects/{project}`, and the resulting `Policy` name is that
parent plus `/policies/{policy}`.

So project-parented creation and a workspace-scoped resource cannot coexist. Keeping
`parent = projects/{project}` while naming the resource `sheets/{sha}` would make the parent a
permission argument wearing a parent's name; dropping the parent to match the name moves
`bb.sheets.create` to the workspace and loses R2. Project ownership is the only shape in which
creation, naming, and access all agree.

## Design

### An ownership edge, not a column

Adding `project` directly to `sheet_blob` would break dedup (R3): the same statement in two projects
would need two rows, and the migration would have to split existing shared rows. Keep `sheet_blob`
as pure content storage and record ownership beside it.

```sql
CREATE TABLE sheet_blob_ref (
    project text  NOT NULL REFERENCES project(resource_id),
    sha256  bytea NOT NULL REFERENCES sheet_blob(sha256),
    PRIMARY KEY (project, sha256)
);

CREATE INDEX idx_sheet_blob_ref_sha256 ON sheet_blob_ref(sha256);
```

`project` alone is a sufficient scope column: `project.resource_id` is a single-column primary key,
so project IDs are one global namespace across every workspace, and a project resolves to exactly
one workspace through `project.workspace`.

The scoped accessors, `MissingSheetsForProject` and the source-level missing-ref audit all query
`WHERE project = ? AND sha256 …`, which the project-leading primary key serves directly. The
secondary index exists for the one query that starts from a hash with no project in hand: the
zero-ref verification in [Post-upgrade audits](#post-upgrade-audits), which PostgreSQL can only
satisfy by full-scanning a `(project, sha256)` btree. That is an operator query rather than a hot
path, but it runs over the whole table and a future `sheet_blob` GC would need the same access
shape.

That query is a deliberate exception to the composite-PK predicate rule in `AGENTS.md`: it asks
which blobs have no ref at all, so there is no scope column to supply. It is an offline audit, not a
request-path read, which is what makes the exception acceptable — no request-path query omits scope.

Migration slot `backend/migrator/migration/3.22/0005##scope_sheet_blob.sql`; `TestLatestVersion` in
`backend/migrator/migrator_test.go` needs updating, and `LATEST.sql` mirrored.

### Backfill

Five reference sources, and only five. `grep -rn "sheet_sha256" proto/store/` returns exactly these
messages; re-run it before writing the migration, and for each hit confirm two things the grep does
not give you — the column name in `LATEST.sql` (not always `payload`) and the nesting of the field
inside its message (not always top-level). `protojson` camelCases keys.

| Source | Column | Path to the hash | Route to project |
|---|---|---|---|
| `plan` | `config` | `specs[].changeDatabaseConfig.sheetSha256` | `plan.project` |
| `task` | `payload` | `sheetSha256` | `task.project` |
| `release` | `payload` | `files[].sheetSha256` | `release.project` |
| `plan_check_run` | `result` | `results[].sheetSha256` | `plan_check_run.project` |
| `revision` | `payload` | `sheetSha256` | corroborated `release` → else corroborated `taskRun` → else **no ref** (see below) |

`plan_check_run` punishes assumption: the column is `result`, not `payload`, and the hash sits on
`PlanCheckRunResult.Result` inside the repeated `results` array.

**`revision` must not route through `db.project`.** The first four sources are project-scoped rows
that never move, so their `project` column is the authoring project. `revision` is not: it has no
project column and inherits one through `db.project`, which is the database's *current* project. A
database transferred before this migration would backfill its revisions to the destination, granting
that project's members access to SQL the source authored — precisely the grant
[the transfer decision](sheet-history-on-database-transfer.md) refuses, and invisible to any audit
because the ref row exists.

Derive the authoring project from the revision's own provenance instead. `RevisionPayload` carries
`release` (`projects/{project}/releases/{release}`) and `task_run`
(`projects/{project}/plans/…/taskRuns/{taskRun}`), both with the project embedded in the resource
name. Parse it out, preferring `release` and falling back to `taskRun`; `backend/store/changelog.go:163-168`
already does exactly this with `regexp_match(payload->>'taskRun', 'projects/([^/]+)/')`.

**A revision produces a ref only when its provenance is corroborated**: the `release` or `task_run`
row it names must still exist under the project it names. There is no fallback — not to
`db.project`, not to anything. Revisions with absent or uncorroborated provenance get no ref, and
their content is unreadable until an operator grants it deliberately.

| Provenance | Action |
|---|---|
| Corroborated — the named `release` or `task_run` row still exists under that project | Stamp `payload.project`, insert `(that project, sha)` |
| Present but not corroborated | No stamp, no ref |
| Absent — both fields empty | No stamp, no ref |

**Corroboration is a one-time event, and its result is stored.** The migration stamps each
corroborated revision's `payload.project` — the same field new revision writers set at creation
from the database's then-current project — and derives the ref from the stamp. Read time never
re-corroborates: `convertToRevision` formats the sheet name from the stamp, and an empty stamp
means no name. The stored fact is what keeps naming and access consistent after transfers; see
[naming the owner](sheet-history-on-database-transfer.md#naming-the-owner).

**There is no `db.project` fallback because an over-grant is invisible.** Falling back would be
correct for every database that never moved, which is most of them, and wrong only for transferred
ones — a tempting trade until you notice the asymmetry in how the two errors surface.
Under-granting produces a NotFound and a row on the ambiguous-provenance audit, with the affected
revision right there in the list. Over-granting produces nothing at all, because the ref exists and
the gate is satisfied, and no audit could flag it — the ref row looks legitimate. A guess that
fails loudly and correctably beats a guess that fails silently and permanently.

So the operational path for these rows is: no ref, a row on the ambiguous-provenance audit, and an
operator who tops up `(requesting project, sha)` for the ones they confirm. That is an informed
grant rather than an inferred one, and the operator has context the migration does not — whether a
database was transferred, and who should see its history.

**Corroboration is checked on the referenced row, not on the project.** `DeleteProject` hard-deletes
the `project` row and `CreateProject` accepts a caller-supplied `ProjectId`
(`backend/api/v1/project_service.go:210`), so a purged ID can be reused; a surviving revision naming
the *old* project would otherwise resolve to an unrelated *new* project of the same name and grant it
the purged project's SQL. Checking the referenced row distinguishes the cases, because ordinary
release deletion is soft (`backend/store/release.go:223`) and leaves the row, while purge
hard-deletes it along with `plan`, `task` and `plan_check_run`
(`backend/store/project.go:578`, `:641`, `:651`, `:671`).

`revision` is the only source that survives a purge of its authoring project at all: a
workspace-instance database is reassigned to the default project and keeps its revisions, whose
`release` and `taskRun` strings still name the deleted project.

**Match the whole reference, and require it to predate the revision.** Row identity alone is not
enough, because every ID in these names restarts per project. `task_run` is
`PRIMARY KEY (project, id)` and `task` is `(project, id)`, so a reused project ID whose first
rollout allocates low IDs reproduces exactly the IDs an old revision names — and the first rollout
in a fresh project is precisely when those low IDs get allocated. Checking only the `task` row from
a `taskRun` name is the weakest version of this and would pass routinely.

Three constraints together close it:

- **Full chain, on a key that identifies one row.** For `taskRun`, match the `task_run` row on
  `(project, id)` — its primary key — *and* require its `task_id` to equal the task ID in the name.
  For `release`, `(project, release_id)` is **not** a declared unique key: `release` is
  `PRIMARY KEY (project, train, iteration)` and `idx_release_project_release_id` is a plain
  `CREATE INDEX`, not a unique one. Corroboration must therefore require `(project, release_id)` to
  match *exactly one* row rather than merely to match some row; provenance that does not identify a
  single release has not identified anything, and `EXISTS` would accept whichever duplicate happens
  to predate the revision and carry the hash. This is the composite-PK identification rule in
  `AGENTS.md` applied to an alternate key that turns out not to be one.

  **A release is resolved in two places, and both need the same test.** Directly, from the
  revision's own `release`; and indirectly, when a corroborating task's `payload.release` names one.
  The indirect path is easy to leave on a bare `EXISTS` — it sits nested inside the task branch and
  reads like a lookup rather than a corroboration — but a duplicate release carrying the hash would
  corroborate a task whose intended release did not, which is the same over-grant by a longer route.
  Exactly-one, age, and hash apply wherever `(project, release_id)` is resolved.
- **Temporal.** Require the corroborating row's `created_at` to be no later than the revision's.
  A revision is written after the task run it records completes, so a genuine reference always
  predates it, whereas a row that reused the ID after a purge is necessarily newer — the purge, the
  new project, and its rollout all happen after the old revision was written. `task_run`, `release`
  and `revision` all carry `created_at`, and `DEFAULT now()` means it cannot be backdated.
- **Hash match.** Require the corroborating row to actually reference *this* revision's
  `sheetSha256`. Identity and age establish that the row is the one the string names; only this
  establishes that the row has anything to do with the sheet being attributed to it.

**The hash match is what makes provenance mean something.** Without it, corroboration proves a task
run exists and is old enough, and nothing more. Pre-fix `createRevisions` validated a `taskRun`'s
identity — project, plan, environment and task all had to line up — but it never compared the
task's sheet to `revision.Sheet`, and its sheet-existence check was deployment-wide. So a
historical revision in project B can name a genuine B task run alongside an unrelated project A
hash, and a corroboration that only checks identity would insert `(B, shaA)` — a laundered grant,
permanent, and invisible to any audit because the ref exists.

Matching the hash costs one more join and closes it. `task.payload` is a oneof
(`proto/store/store/task.proto:27-36`), so the task branch has two shapes: `sheetSha256` equal to
the revision's hash, or `release` naming a release whose `payload.files[]` contains it. The release
branch is the single-hop form of the same test.

Apply it to the release branch too, even though `createRevisions` already enforces
`fileSheet == revision.Sheet` there. Uniformity is worth more than the saved join: it removes any
need to reason about which creation-time validation happened to run, on a path where T6 shows those
validations can be steered.

The temporal test defeats ID reuse, the full chain narrows the window it must cover, and the hash
match defeats laundering. Residual risk after all three is limited to clock anomalies, and
uncorroborated rows fail closed to no ref — an unreadable statement rather than a wrong grant.

Their content then becomes unreadable, which is the intended outcome: the authoring project is gone,
so there is nobody left to ask. An unstamped revision carries no sheet name rather than one that
echoes the parsed ID — an ID that failed corroboration cannot be verified against any workspace, so
it is not safe to surface. See
[naming the owner](sheet-history-on-database-transfer.md#naming-the-owner).

A revision with no provenance at all is treated the same way: no stamp, no ref, and a grant only if
an operator confirms it. Both populations appear together on the ambiguous-provenance list in
[Post-upgrade audits](#post-upgrade-audits).

Two tables that look like sources are not. `ChangelogPayload` carries only `task_run` and
`git_commit`, and the v1 `Changelog` message exposes no statement or sheet field. `issue_comment`
has no sheet reference of any kind. Four *historical* locations also held hashes and are all dead:
`task_run.sheet_sha256` (column added and dropped within 3.14), `plan_check_run.config` (column
gone), and `changelog.payload.sheetSha256` / `issue_comment.planSpecUpdate.*SheetSha256` (proto
fields removed; the JSONB keys linger in old rows, invisible under `DiscardUnknown` unmarshaling,
with zero readers).

**The migration must not abort on data, whatever is stored.** Every data-dependent hazard is
guarded: hex-shape fences ahead of every `decode()`; `jsonb_typeof` checks ahead of every
`jsonb_array_elements` (COALESCE alone passes a stored JSON `null` through); `EXISTS` against
`sheet_blob` so a payload naming a hash with no blob cannot violate the foreign key; `EXISTS`
against `project` on the stamped-refs insert so a pre-existing bogus `payload.project` cannot
either; and bounded digit matches (`\d{1,18}`) so a malformed name cannot overflow the bigint
casts. Each guard has a migration-test probe seeding the hostile shape. The migration runs as one
transaction under the migrator's advisory lock, so any environmental failure rolls back completely
and retries at next startup — fail-stop, never fail-corrupt.

**Scale is addressed with indexed temp tables, not CTEs.** The corroboration intermediates are
`CREATE TEMP TABLE … ON COMMIT DROP`, indexed and `ANALYZE`d before use. CTE materializations carry
no statistics and no indexes, which sent the planner into nested loops — measured at ~15 minutes
over a ~1M-row synthetic deployment as CTEs, 19.5 seconds total as temp tables, with every
statement linear in table size. `ON COMMIT DROP` keeps the intermediates inside the migration
transaction, so atomicity is unchanged.

**Unreferenced blobs become unreadable.** A blob referenced by nothing has no derivable project. In
practice this is a sheet created but never attached to a plan or release; the UI attaches
immediately, and `createSheet` caches content client-side (`frontend/src/stores/app/sheet.ts:91`) so
a fresh draft does not re-fetch. Leaving orphans globally readable would reproduce the bug, so the
gap is accepted and noted in the migration comment.

**Observed references are not authorization.** The backfill reconstructs which project *referenced*
a hash, which is not the same as which project was *entitled* to it.
`convertPlanSpecChangeDatabaseConfig` parses a caller-supplied `projects/{project}/sheets/{sha}`
and stores only the hash, and the pre-fix existence check had no scope — so a historical plan in
project B may reference a hash owned by project A, indistinguishable after the write from a
legitimate one. Backfilling `(B, sha)` mints a permanent grant out of the access being closed, and
no audit can prove it wrong because the ref row exists.

No ground truth survives: the refactor that caused this bug deleted the sheet's creator and project
metadata. The population is therefore sized and reviewed rather than filtered — see
[Post-upgrade audits](#post-upgrade-audits).

### Store API

`sheet_blob_ref` is an ACL fact enforced by the store's scoped accessors; content retrieval and
access control stay separate steps inside them. The runners are why the split is real rather than
cosmetic: when `database_migrate_executor` reads a sheet it is executing work authorized when the
plan or release was created, not making an access decision. Threading a project through it to
satisfy a signature would be authorization theater, so the unscoped getter stays — exported,
documented as runner-only, and confined out of `backend/api/v1` by a static test.

Every primitive is set-shaped (R5), and everything else is unexported so the compiler enforces the
split:

```go
// Unscoped, runners and components only. A hash fully determines content.
func (s *Store) GetSheetFull(ctx, sha256Hex string) (*SheetMessage, error)

// Scoped reads: filter the hashes through the project's refs, then fetch
// content only for the survivors. Two queries whatever the input size.
func (s *Store) GetSheetsForProject(ctx, projectID string, sha256Hexes []string, raw bool) (map[string]*SheetMessage, error)
func (s *Store) GetSheetForProject(ctx, projectID, sha256Hex string, raw bool) (*SheetMessage, error)

// Scoped validation predicate, not a content read: the hashes, in input
// order, for which the project holds no ref. Callers name the first missing
// sheet in their error.
func (s *Store) MissingSheetsForProject(ctx, projectID string, sha256Hexes ...string) ([]string, error)

// Scoped creation: writes blobs and ref rows, both as set inserts, in one
// transaction behind the project purge fence.
func (s *Store) CreateSheets(ctx, projectID string, creates ...*SheetMessage) ([]*SheetMessage, error)

// Unexported internals: filterSheetsForProject (the ref query),
// getSheetsFull (cache-aware batch fetch), getSheets (single content query).
```

A malformed hash is filtered before any SQL runs — it can never match a blob, so it is treated as
absent rather than becoming a decode error. `CreateSheets` is a transaction so a blob cannot land
without its ref, with the blob inserted first because the ref's foreign key requires it; both
inserts are set-shaped `ON CONFLICT DO NOTHING`. Its four call sites all pass the owning project:
the sheet service's two create RPCs, `CreateRelease` for statement-carrying files, and the
create-database task path, which threads the plan's project down from `GetPipelineCreate`.

### Cache ordering

`sheetFullCache` stays keyed by hex hash alone (a 10-entry LRU in `backend/store/store.go`).
Content is a pure function of the hash, so a hash-keyed content cache is always correct, and the
dominant consumer — a runner reading the same sheet repeatedly — keeps full hit rate.

This is safe **only** because the ref check is a separate step that runs first, inside
`GetSheetsForProject`, ahead of the cache-aware fetch. A cache hit returns before any query
executes, so a scope predicate placed with the content fetch would be skipped. The check and the
fetch must not be fused into one join, and must not be reordered; the store method carries that
warning, and the cache-ordering integration test fails if it ever regresses.

Ten entries is sized for the runner pattern, not for batch reads, and a release with hundreds of
files will miss on nearly all of them. That is not a reason to grow it: the batch path collapses
those misses into a single query.

### The gate lives in the store

Enforcement sits in the scoped store accessors, and the compiler does most of the confinement: the
raw batch getters and the ref filter are unexported, so nothing outside the store can fetch content
around the check. The one exported unscoped getter, `GetSheetFull`, exists for the runners, and a
static AST test asserts nothing under `backend/api/v1/` calls it — the single boundary the compiler
cannot enforce, since runners live in other packages.

A hash the project may not read yields NotFound rather than PermissionDenied, so the response does
not confirm the hash exists in another project.

**Every v1 consumer, enumerated.** All user-facing paths route through the scoped accessors:

| Call site | Path | Routes to |
|---|---|---|
| `sheet_service.go` | `GetSheet` | `GetSheetForProject` |
| `release_service_check.go` | `loadReleaseFileStatements`, via `CheckRelease` | `GetSheetsForProject` |
| `rollout_service.go` | `PreviewTaskRunRollback` | `GetSheetForProject` |
| `rollout_service_task.go` | `getSheetContentBySha256` (ghost directive) | `GetSheetForProject` |
| `release_service.go` | `CreateRelease` validation | `MissingSheetsForProject` |
| `revision_service.go` | `BatchCreateRevisions` validation | `MissingSheetsForProject` |
| `plan_service.go` | `validateSpecs` | `MissingSheetsForProject` |

`PreviewTaskRunRollback` is the one most easily missed — it is a content read that does not look
like one, reached through a rollback-preview route rather than through anything named for sheets.

Two batching notes that fell out of the rewiring: `validateAndSanitizeReleaseFiles` is now a pure
function — `CreateRelease` persists only hashes, so its old full-content prefetch (potentially
megabytes per file) is gone entirely, replaced by one ref check; only `CheckRelease`, which runs
advisors over the SQL, hydrates statements, in one batched scoped read. The ghost-directive check
fetches its loop-invariant hash once instead of once per target database.

### Creation-time validation only

References are validated when the referencing object is created, and never re-checked per use:

- `CreateRelease` checks every referenced sheet against the project's refs before persisting, and
  creates sheets (with refs) for inline statements. Releases are immutable — `UpdateRelease` is
  `Unimplemented` — so the check cannot go stale.
- `validateSpecs` runs at `CreatePlan` and `UpdatePlan` and checks the specs' direct hashes; for a
  release reference it verifies the release exists in the same project, whose files were themselves
  validated at `CreateRelease`.
- `BatchCreateRevisions` checks the revision's hash against the database's project.

What makes one-time checks durable is the deletion story: sheets are never deleted, and refs are
deleted only by project purge, which deletes every referencing row — plan, task, release,
plan_check_run — in the same transaction. There is no state in which a previously-valid stored
reference silently loses its ref while the referencing object survives.

Historical objects created before enforcement need no revalidation either, because the backfill
resolves them: every stored `(project, hash)` reference in the four sources received a ref, so
pre-existing plans and releases keep working, and any cross-project reference among them is a ref
row the multi-project audit surfaces for review rather than a runtime failure. A stored reference
that got no ref (a hash whose blob never existed) fails closed at the scoped reads on the rollout
path.

### Project purge

The `project` foreign key makes `sheet_blob_ref` a dependent of `project`, and purge ends with
`DELETE FROM project WHERE resource_id = ?` (`backend/store/project.go:826`) after clearing every
other dependent table. Add `DELETE FROM sheet_blob_ref WHERE project = ?` after the `db` delete
(`:704`) and before `project_webhook` (`:739`), matching the position established under
[lock ordering](#transaction-lock-ordering). Without it, purging any project fails on the foreign
key.

Blobs whose only ref belonged to the purged project are left with zero refs, consistent with today's
behavior — nothing has ever deleted from `sheet_blob` — and the state a future GC would collect.

### Database transfer

No sheet references are carried when a database changes project. Change history follows the
database; statement content stays with the project that authored it. The four code paths that move a
database between projects therefore need no changes here.
[`sheet-history-on-database-transfer.md`](sheet-history-on-database-transfer.md) records that
decision and its requirements.

## Alternatives considered

**Workspace scope.** Make sheets shared within a tenant: `sheet_blob_ref(workspace, sha256)`, the
resource renamed to a bare `sheets/{sheet}` collection, and `CreateSheet` parentless with
`bb.sheets.create` checked at workspace level. This is internally consistent and it is attractive on
cost — databases cannot move between workspaces (`UpdateProject`'s mask has no `workspace` case), so
the transfer question never arises; the ref table would reference `workspace` rather than `project`,
so purge would not touch it; and the laundering problem would shrink to cross-workspace references,
which cannot be created legitimately and should audit to zero.

It fails R2 and R6. Parentless creation moves authoring governance to the workspace, so "this user
may write change SQL for project A only" stops being expressible. Cross-project read inside a
workspace stays open, which is a policy violation for regulated-separation and agency customers, and
it leaves sheets inconsistent with `worksheet`, which already applies `PRIVATE` / `PROJECT_READ` /
`PROJECT_WRITE` visibility to SQL text.

It is also a breaking API change across four proto sites — the `Sheet` pattern
(`proto/v1/v1/sheet_service.proto:92`), `GetSheetRequest.name` (`:83`), `Release_File.sheet`
(`proto/v1/v1/release_service.proto:331`), `Revision.sheet`
(`proto/v1/v1/revision_service.proto:183`) — plus the shape of `CreateSheet`. Storage is unaffected,
since only the hash persists and the resource name is constructed on read, so this is a contract
break rather than a data migration; it would still need the `breaking` label and a dual-accept
deprecation window.

The asymmetry decides it. Project scope costs no API change in either direction and can be relaxed
later behind a stable contract. Workspace scope costs one breaking change to adopt and a second, in
the opposite direction, to reverse.

**Workspace scope keeping the project prefix.** Rejected outright. It would make `Sheet` the only
resource in the API whose name prefix is a caller-chosen scope, and it aliases — the same sheet
answering to `projects/A/sheets/X` and `projects/B/sheets/X`, with `name` echoing whichever was
asked. That is the decorative prefix the audit named, preserved deliberately.

**A `project` column on `sheet_blob`.** Fails R3. The same statement in two projects would need two
rows, and the migration would have to split existing shared rows.

**Enforcement as a predicate inside the content getter.** Rejected, though store-level enforcement
was ultimately adopted in a different shape. The rejected form puts the scope predicate in the
content query itself, which the hash-keyed cache skips on a hit; the adopted form keeps the ref
check a separate first step inside the scoped accessors, ahead of the cache, which preserves the
compiler-checked-parameter advantage without the cache hazard. The runners are not forced to supply
a project to satisfy a signature: the unscoped getter remains for them, confined out of the API
layer by a static test rather than by threading authorization theater through executors.

## Post-upgrade audits

Correctness rests on backfill completeness, which cannot be established by construction; a missed
source is a silent NotFound (R4). Treat that as demonstrated risk rather than hypothesis — two
drafts of this document got the source list wrong, first inventing two sources that do not exist,
then naming the wrong column and nesting for `plan_check_run`.

**Enforcement ships on, with no shadow mode.** An earlier draft staged enforcement behind a
shadow-logging release. That staging was dropped along with the per-use checks it instrumented:
with creation-time-only validation there is no per-request check to shadow, and the risk it hedged
is addressed directly instead. Completeness is proven rather than observed — the proto sweep pins
exactly five live hash locations, all covered; the four historical locations are verified dead
(columns dropped or fields removed, zero readers); and the migration's output is asserted exactly,
scenario by scenario, in `TestMigration3_22_5_ScopeSheetBlob`. What remains after that proof is the
designed-unreadable population — orphan drafts and uncorroborated revisions — which shadow logging
would only have observed at request time anyway. The audits below identify it exhaustively up
front, which strictly dominates a traffic-dependent log.

The queries carry the same robustness guards as the migration — `jsonb_typeof` ahead of every array
expansion, a hex-shape test ahead of every `decode()`, bounded digit matches — so they cannot error
on a malformed stored payload.

**Verify the backfill first.** Blobs with zero refs should be a small, non-growing population:

```sql
SELECT count(*) FROM sheet_blob b
WHERE NOT EXISTS (SELECT 1 FROM sheet_blob_ref r WHERE r.sha256 = b.sha256);
```

**Review multi-project hashes before enforcing.** Each is either honest dedup or a laundered
reference, and the data cannot distinguish them, so this is a review list rather than a filter:

```sql
SELECT encode(sha256, 'hex'), count(DISTINCT project), array_agg(DISTINCT project)
FROM sheet_blob_ref GROUP BY sha256 HAVING count(DISTINCT project) > 1;
```

A long list means cross-project references are already widespread and the plan needs revisiting,
since the backfill would make them permanent. Capture it at a known-good point: this design is
public, including the rule that turns a reference into an ownership edge.

**Review revisions that produced no ref.** These are the rows whose statements are now unreadable —
absent provenance, or provenance the backfill could not corroborate. The migration stamped every
corroborated row's `payload.project`, so the population is simply the unstamped rows that carry a
hash — no need to repeat the corroboration test, since its outcome is stored:

```sql
SELECT r.instance, r.db_name, r.resource_id,
       (regexp_match(COALESCE(NULLIF(r.payload->>'release', ''),
                              r.payload->>'taskRun'), 'projects/([^/]+)/'))[1] AS named_project
FROM revision r
WHERE r.payload->>'sheetSha256' ~ '^[0-9a-fA-F]{64}$'
  AND r.payload->>'project' IS NULL;
```

`named_project` is the project the provenance *claims*, surfaced for the operator's investigation
only — it failed corroboration, so it must not be granted (or shown to end users) on the query's
say-so. To grant one after review: stamp `payload.project` and insert the ref in one transaction,
which is exactly what the migration does for corroborated rows.

A large count means either widespread project purges, reused project IDs, or a population of
revisions created without provenance — each worth understanding, since all of them end in
unreadable statements.

## Transaction lock ordering

`sheet_blob_ref` is a project-owned branch and takes a position in the canonical sibling order in
`backend/store/README.md`, updating that list, `DeleteProject`, and `DeleteInstance` together. It is
a direct child of `project` with no descendants, and every table that can reference a hash sits
earlier, so it belongs immediately after `db` and before `project_webhook` — the same position as
the purge delete.

```text
... -> changelog -> sync_history -> revision -> db_schema -> db
-> sheet_blob_ref -> project_webhook -> service_account -> ...
```

`CreateSheets` becomes a transaction spanning `sheet_blob` and `sheet_blob_ref`. Insert the blob
first: the ref's foreign key requires it, so the order is forced. Both are `ON CONFLICT DO NOTHING`
— new-row-only inserts, not upserts that can update an existing row, so the README's rule 4 upsert
clause does not apply.

As a writer of purge-managed data, `CreateSheets` must declare a lifecycle policy. **It requires an
active project**: a sheet ref is a new resource with no deleted-project continuation case. The API
create paths check the project before calling the store, but that check is not serialized with
purge, so a concurrent purge would surface as a raw foreign-key violation rather than a controlled
NotFound. `CreateSheets` therefore takes the same transaction-scoped purge fence that database
creation and task-run creation already take; `withDatabasePurgeFence`
(`backend/store/database.go:798-820`) is the pattern to mirror.

Both need the deterministic real-PostgreSQL regression tests that section mandates, asserting
terminal outcomes in both lock-acquisition directions rather than merely the absence of SQLSTATE
`40P01`.

## Independent fixes

No schema needed; landed with the change.

1. **The T6 hash echo.** `backend/api/v1/revision_service.go:203` takes `projectID` from the
   attacker-supplied `revision.File` and uses it as a store key without comparing it to
   `database.ProjectID`. `FindReleaseMessage` has no `Workspace` field
   (`backend/store/release.go:30-37`), so this reaches any project in any workspace, and `:223`
   echoes the real hash. Add the comparison, drop the hash from the error. The same missing
   comparison appears for `revision.TaskRun` at `:167-186`. This is the cheapest way to obtain a hash
   today and should go first.
2. **Delete the stale comments** at `release_service.go:331` and `:405`
   (`// Sheets are now project-agnostic, no need to check projectID`) — the original wrong assumption,
   sitting where it will talk the next reader into repeating it.
3. **Add audit annotations** to `proto/v1/v1/sheet_service.proto`'s create RPCs, which declare none
   while 20 other v1 services do, leaving sheet authoring unlogged. `GetSheet` stays unaudited
   deliberately: it is a high-volume content fetch, and the convention across the API audits
   mutations plus the SQL query/export paths, not plain Gets.
4. **Drop `bb.sheets.update`** — granted to four roles, declared by no RPC, dead since sheets became
   immutable.
5. **Comment the GC hazard on `sheet_blob`.** Nothing deletes from it; the only foreign key that ever
   pointed at it (`task_run.sheet_sha256`, migration 3.14) was dropped in `7192406c5f`, so every
   surviving reference lives in JSONB and is invisible to referential integrity. A GC written the
   obvious way — delete blobs no FK points at — would empty the table. `sheet_blob_ref` makes a
   correct GC possible later.

**Audit the sources directly rather than inferring from refs.** A reference to a *pre-existing*
blob made through a stored payload invokes no `CreateSheets` and writes no ref, and the
multi-project audit cannot see it because that audit reads `sheet_blob_ref`. For each of the four
project-scoped sources, find `(project, hash)` pairs with no matching ref:

```sql
SELECT DISTINCT pl.project, spec->'changeDatabaseConfig'->>'sheetSha256' AS sha
FROM plan pl
CROSS JOIN LATERAL jsonb_array_elements(
    CASE WHEN jsonb_typeof(pl.config->'specs') = 'array' THEN pl.config->'specs' ELSE '[]'::jsonb END) AS spec
WHERE spec->'changeDatabaseConfig'->>'sheetSha256' ~ '^[0-9a-fA-F]{64}$'
  AND NOT EXISTS (
    SELECT 1 FROM sheet_blob_ref r
    WHERE r.project = pl.project
      AND r.sha256  = decode(spec->'changeDatabaseConfig'->>'sheetSha256', 'hex')
  )
-- UNION ALL the same shape for task, release, plan_check_run
;
```

Every row is something that will 404 on a scoped read. Immediately after the upgrade this should
return only references whose blob never existed — the backfill covers everything else — and it is
exhaustive rather than traffic-dependent, needing no assumption about when a reference was created.

`revision` is deliberately excluded: uncorroborated revisions produce no ref *by design*, so they
would flood this query with expected rows. They have their own list above.

## Tests

Integration coverage is four tests in `backend/tests/sheet_scope_test.go`, one per independent
decision, plus the migration matrix and the lock-order pair:

- **Cross-project read and cache ordering** — `TestSheetProjectScope`: a sheet created in project A
  is readable there (raw and truncated), NotFound under project B — with the raw foreign read
  running *after* the owner's raw read warmed the hash-keyed content cache, so the test fails if
  enforcement ever moves behind the cache. The same test pins the CreateRelease minting gate
  (naming A's sheet from B is refused), the owning project's own reference succeeding, and
  CheckRelease hydrating real content (a sheet holding broken SQL must surface a syntax-error
  advice — only hydrated content can). Cross-*tenant* reads are subsumed mechanically: refs are
  per-project and the scoped accessors consult nothing else, so there is no workspace-dependent
  code path to test separately.
- **Transfer semantics** — `TestSheetHistoryOnDatabaseTransfer`: the runtime half of the companion
  doc's decision. History follows the database; the sheet stays named under the stamped authoring
  project; the destination gains no read access.
- **Purge semantics** — `TestSheetHistoryAfterOwnerPurge`: after the authoring project is purged,
  the revision list survives, the dangling sheet name still renders, and reading it 404s — the
  contract the frontend's withheld-statement panel is built on.
- **Write non-interference** — `TestCollision_SheetWrite` per the composite-PK convention in
  `AGENTS.md`, through the shared `setupCollidingProjects` fixture: a sheet write in one project
  whose content (and therefore sha256) equals another project's leaves that project's refs
  bit-identical.
- **Backfill** — `TestMigration3_22_5_ScopeSheetBlob`: all five sources, the full corroboration
  matrix (identity, exactly-one, age, hash, preference, fallback — every negative asserted by an
  exact ref-set match), the stamp outcomes per scenario, the three audits, and a probe per
  abort-hazard guard (dangling blob, ghost project, bigint overflow, stale stamp, JSON null and
  malformed array shapes).
- **Create versus purge** — the deterministic lock-order pair mandated by
  `backend/store/README.md`: `CreateSheets` concurrent with a purge of its project ends in a
  controlled NotFound, not a foreign-key error, in both lock-acquisition directions.
- **Gate confinement** — a static AST test asserts nothing under `backend/api/v1/` calls
  `GetSheetFull`; the rest of the split is compiler-enforced by unexported store internals.

## Order

Everything ships in one release: the independent fixes, the schema, the ref writers, the backfill,
and enforcement. The migration creates the table, backfills, and stamps atomically at startup
before the binary serves traffic, and every scoped code path is live from the first request. The
staging that an earlier draft spread across three releases collapsed along with shadow mode — with
creation-time-only validation and a proven-complete backfill there is no instrumentation period to
stage.

Two windows remain, both narrow and both documented:

- **Rolling upgrade.** Old replicas keep serving the pre-fix writers while the first new replica
  migrates. A sheet created by an old replica in that window lands in `sheet_blob` with no ref (an
  orphan draft — readable again the moment anything references it through a new-binary create
  path, since `CreateSheets` upserts), and a revision written by an old replica lands unstamped —
  its content stays readable through the release/task-source refs, and it appears on the
  ambiguous-provenance audit for a manual stamp. Neither corrupts anything; both self-identify.
- **Re-running the backfill.** It is idempotent (`ON CONFLICT DO NOTHING`, stamp merge writes the
  same value), but do not re-run it blindly to "top up": any reference created *after* enforcement
  went live was validated by the creation-time gates, so a re-run adds nothing legitimate — and
  the migration's version tracking prevents accidental re-execution anyway.

Run the three audits after the upgrade; grant survivors deliberately, per
[Post-upgrade audits](#post-upgrade-audits).
