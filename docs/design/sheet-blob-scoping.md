# Scoping `sheet_blob` to its project

Fixes T5 in `docs/design/v1-api-audit-2026-08.md`.

Sheet content is fetched by SHA256 with no scope predicate, so any principal holding
`bb.sheets.get` on any one project can read any sheet in the deployment — across projects and
across tenants — given the hash. A `sheet_blob_ref(project, sha256)` edge table makes ownership an
explicit stored fact, and an API-layer gate enforces it.

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
changes needed)" as a non-goal, true before the cutover and false after. The assumption survives in
the tree as a justification for skipping checks, at `backend/api/v1/release_service.go:331` and
`:405`.

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

The gate and `HasSheets` both query `WHERE project = ? AND sha256 IN (…)`, which the
project-leading primary key serves directly. The secondary index exists for the two paths that
start from a hash with no project in hand: the owner lookup for a withheld statement
([naming the owner](sheet-history-on-database-transfer.md#naming-the-owner)) and the zero-ref
verification query in [Rollout](#rollout). PostgreSQL can only full-scan a `(project, sha256)` btree
for a `sha256`-only predicate, so both would degrade as the table grows.

Those hash-first queries are a deliberate exception to the composite-PK predicate rule in
`AGENTS.md`: their whole purpose is to discover *which* project holds a hash, so the scope column
cannot be supplied. They are made safe by joining through `project` and constraining to the caller's
workspace instead — never by omitting scope entirely.

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
[the transfer decision](sheet-history-on-database-transfer.md) refuses, and invisible to shadow mode
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
| Corroborated — the named `release` or `task_run` row still exists under that project | Insert `(that project, sha)` |
| Present but not corroborated | No ref |
| Absent — both fields empty | No ref |

**There is no `db.project` fallback because shadow mode cannot see an over-grant.** Falling back
would be correct for every database that never moved, which is most of them, and wrong only for
transferred ones — a tempting trade until you notice the asymmetry in how the two errors surface.
Under-granting produces a shadow miss: a log line, during a full release in which content is still
returned, with the requesting project right there in the log. Over-granting produces nothing at all,
because the ref exists and the gate is satisfied. A guess that fails loudly and correctably beats a
guess that fails silently and permanently.

So the operational path for these rows is: no ref, shadow misses during the enforcement-deferred
release, operator reviews the log, and tops up `(requesting project, sha)` for the ones they confirm.
That is an informed grant rather than an inferred one, and the operator has context the migration
does not — whether a database was transferred, and who should see its history.

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
- **Temporal.** Require the corroborating row's `created_at` to be no later than the revision's.
  A revision is written after the task run it records completes, so a genuine reference always
  predates it, whereas a row that reused the ID after a purge is necessarily newer — the purge, the
  new project, and its rollout all happen after the old revision was written. `task_run`, `release`
  and `revision` all carry `created_at`, and `DEFAULT now()` means it cannot be backdated.
- **Hash match.** Require the corroborating row to actually reference *this* revision's
  `sheetSha256`. Identity and age establish that the row is the one the string names; only this
  establishes that the row has anything to do with the sheet being attributed to it.

**The hash match is what makes provenance mean something.** Without it, corroboration proves a task
run exists and is old enough, and nothing more. `createRevisions` validates a `taskRun` thoroughly —
project, plan, environment and task all have to line up, and `:173` even requires the task run to
belong to the database's own project — but it never compares the task's sheet to `revision.Sheet`
(`backend/api/v1/revision_service.go:167-192`), while `HasSheets` at `:158` is still deployment-wide.
So a revision in project B can name a genuine B task run alongside an unrelated project A hash, and
a corroboration that only checks identity would insert `(B, shaA)` — a laundered grant, permanent,
and invisible to shadow mode because the ref exists.

Matching the hash costs one more join and closes it. `task.payload` is a oneof
(`proto/store/store/task.proto:27-36`), so the task branch has two shapes: `sheetSha256` equal to
the revision's hash, or `release` naming a release whose `payload.files[]` contains it. The release
branch is the single-hop form of the same test.

Apply it to the release branch too, even though `createRevisions` already enforces
`fileSheet == revision.Sheet` there (`:222`). Uniformity is worth more than the saved join: it
removes any need to reason about which creation-time validation happened to run, on a path where T6
shows those validations can be steered.

The temporal test defeats ID reuse, the full chain narrows the window it must cover, and the hash
match defeats laundering. Residual risk after all three is limited to clock anomalies, and
uncorroborated rows fail closed to no ref — an unreadable statement rather than a wrong grant.

Their content then becomes unreadable, which is the intended outcome: the authoring project is gone,
so there is nobody left to ask. The withheld-statement response reports the owner as unknown rather
than echoing the parsed ID — an ID that failed corroboration cannot be verified against any
workspace, so it is not safe to surface. See
[naming the owner](sheet-history-on-database-transfer.md#naming-the-owner).

A revision with no provenance at all is treated the same way: no ref, surfaced as a shadow miss, and
granted only if an operator confirms it. Both populations appear together on the review list in
[Rollout](#rollout).

Two tables that look like sources are not. `ChangelogPayload` carries only `task_run` and
`git_commit`, and the v1 `Changelog` message exposes no statement or sheet field. `issue_comment`
has no sheet reference of any kind.

Guard every derivation with `EXISTS (SELECT 1 FROM sheet_blob …)` so a payload naming a hash with no
blob cannot violate the foreign key.

**Unreferenced blobs become unreadable.** A blob referenced by nothing has no derivable project. In
practice this is a sheet created but never attached to a plan or release; the UI attaches
immediately, and `createSheet` caches content client-side (`frontend/src/stores/app/sheet.ts:91`) so
a fresh draft does not re-fetch. Leaving orphans globally readable would reproduce the bug, so the
gap is accepted and noted in the migration comment.

**Observed references are not authorization.** The backfill reconstructs which project *referenced*
a hash, which is not the same as which project was *entitled* to it.
`convertPlanSpecChangeDatabaseConfig` (`backend/api/v1/plan_service.go:1157`) parses a
caller-supplied `projects/{project}/sheets/{sha}` and stores only the hash, and `HasSheets`
currently validates existence with no scope — so a plan in project B may already reference a hash
owned by project A, indistinguishable after the write from a legitimate one. Backfilling `(B, sha)`
would mint a permanent grant out of the access being closed, and shadow mode cannot see it because
the ref row exists.

No ground truth survives: the refactor that caused this bug deleted the sheet's creator and project
metadata. The population is therefore sized and reviewed rather than filtered — see
[Rollout](#rollout).

### Store API

`sheet_blob_ref` is an ACL fact enforced at the API layer; content retrieval and access control stay
separate operations. The runners are why that distinction is real rather than cosmetic: when
`database_migrate_executor` reads a sheet it is executing work authorized when the plan was created,
not making an access decision. Threading a project through it to satisfy a signature would be
authorization theater.

Every primitive is set-shaped (R5).

```go
// Content only. A hash fully determines content, so no scope is involved.
func (s *Store) GetSheetsFull(ctx, sha256Hexes ...string) (map[string]*SheetMessage, error)
func (s *Store) GetSheetsTruncated(ctx, sha256Hexes ...string) (map[string]*SheetMessage, error)

// Scoped: which of these hashes may this project read? One query.
func (s *Store) FilterSheetsForProject(ctx, projectID string, sha256Hexes ...string) (map[string]bool, error)

// Scoped: validation predicate, not a content read.
func (s *Store) HasSheets(ctx, projectID string, sha256Hexes ...string) (bool, error)

// Scoped: writes blobs and ref rows, both as set inserts, in one transaction.
func (s *Store) CreateSheets(ctx, projectID string, creates ...*SheetMessage) ([]*SheetMessage, error)
```

`CreateSheets` needs a transaction so a blob cannot land without its ref; it currently issues a bare
`ExecContext`. The ref insert mirrors the blob insert's array shape rather than looping.

Three of its four call sites have a project to hand — `sheet_service.go:54` and `:97`, and
`release_service.go:96`. The fourth does not: `rollout_service_task.go:143` stores generated
`CREATE DATABASE` boilerplate from inside `getTaskCreatesFromCreateDatabaseConfig`, which takes no
project, and neither does its caller `getTaskCreatesFromSpec`. Thread it down from
`rollout_service.go:1279`, which has the plan's project.

### Cache ordering

`sheetFullCache` stays keyed by hex hash alone (`backend/store/store.go:51,82,110`, a 10-entry LRU).
Content is a pure function of the hash, so a hash-keyed content cache is always correct, and the
dominant consumer — a runner reading the same sheet repeatedly — keeps full hit rate.

This is safe **only** because the scope check is a separate step that runs first. The cache read at
`backend/store/sheet.go:43` returns before any query executes, so a scope predicate placed inside
`getSheet` would be skipped on a cache hit. Enforcement must not live there, and the check and the
fetch must not be fused into one join.

Ten entries is sized for the runner pattern, not for batch reads, and a release with hundreds of
files will miss on nearly all of them. That is not a reason to grow it: the batch path collapses
those misses into a single query.

### API-layer gate

Moving enforcement out of the store means the compiler no longer finds missed call sites, and a
forgotten check is exactly how `release_service.go:398` and `plan_service.go:642` ended up unscoped.
Structure replaces the compiler: one package-level helper doing check-then-fetch — package-level
because most routed call sites live in `ReleaseService` and `RolloutService`, not `SheetService` —
taking a set and costing two queries whatever the input size.

1. One ref query returning the subset of requested hashes the project may read.
2. Cache lookups for survivors, then one content query for the misses.

A hash that does not survive yields NotFound rather than PermissionDenied, so the response does not
confirm the hash exists in another project. Fusing the two steps into a single join is rejected: it
would put the content read in the same statement as the permission check, which is exactly the
arrangement a cache hit can skip.

A test asserts nothing under `backend/api/v1/` calls a store content getter outside the gate.
Runner and component call sites keep the unscoped getters.

Three existing call patterns need reshaping to feed it. `validateAndSanitizeReleaseFiles`
(`release_service.go:377-413`) fetches once per file inside its loop — a pre-existing N+1 that a
per-file scope check would double. `rollout_service_task.go:214` fetches a loop-invariant hash once
per target database, masked today only by the LRU. `plan_service.go:746` is already correct and
worth copying: it collects across every spec and makes one call.

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

**Enforcement inside the store.** Attractive because a required parameter is compiler-checked at
every call site, which is stronger than a convention. Rejected because the content cache is keyed by
hash and returns before any query runs, so a predicate inside `getSheet` is skipped on a cache hit —
and because the runners would be forced to supply a project purely to satisfy a signature, when they
are executing already-authorized work rather than making an access decision.

## Rollout

Correctness rests on backfill completeness, which cannot be established by construction; a missed
source is a silent NotFound (R4). Treat that as demonstrated risk rather than hypothesis — two
drafts of this document got the source list wrong, first inventing two sources that do not exist,
then naming the wrong column and nesting for `plan_check_run`.

**Ship every scope check in shadow mode, not just the read gate.** The gate evaluates the check and
logs a miss — project, hash, call site — but still returns content. `HasSheets` needs the same
treatment and is easy to overlook, because it is a *validation* predicate rather than a read: it
backs plan, release and revision validation and fails the request outright with "some sheets are not
found". Left unshadowed, any pre-existing blob without a ref would turn into a spurious validation
error on a write path — a worse failure than a withheld read, since it blocks work rather than
hiding content. In shadow mode it falls back to the deployment-wide existence check and logs the
miss. Run one full release; flip to enforcing only once logs are clean.

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
absent provenance, or provenance the backfill could not corroborate. They will appear as shadow
misses during the deferred-enforcement release; this query is the same population identified up
front, so a top-up can be prepared before the log accumulates.

The test must repeat the corroboration check rather than asking whether the hash has any ref at all.
The design expects hashes to be shared, so a revision whose own provenance failed may still have a
ref from an unrelated project that authored identical SQL — and a ref-existence proxy would silently
drop exactly the rows this list exists to surface.

```sql
SELECT r.instance, r.db_name,
       (regexp_match(COALESCE(NULLIF(r.payload->>'release', ''),
                              r.payload->>'taskRun'), 'projects/([^/]+)/'))[1] AS named_project
FROM revision r
WHERE r.payload->>'sheetSha256' IS NOT NULL
  -- the corroboration test from the backfill, repeated verbatim:
  -- single-row identity, age, and the hash the row actually references.
  -- (project, release_id) is not a declared unique key, so require exactly one match.
  AND NOT (
    (SELECT count(*) FROM release rel
      WHERE rel.project    = (regexp_match(r.payload->>'release', 'projects/([^/]+)/'))[1]
        AND rel.release_id = (regexp_match(r.payload->>'release', 'releases/([^/]+)'))[1]) = 1
    AND EXISTS (
      SELECT 1 FROM release rel
      CROSS JOIN LATERAL jsonb_array_elements(COALESCE(rel.payload->'files', '[]'::jsonb)) AS f
      WHERE rel.project      = (regexp_match(r.payload->>'release', 'projects/([^/]+)/'))[1]
        AND rel.release_id   = (regexp_match(r.payload->>'release', 'releases/([^/]+)'))[1]
        AND rel.created_at  <= r.created_at
        AND f->>'sheetSha256' = r.payload->>'sheetSha256'
    )
  )
  AND NOT EXISTS (
    SELECT 1 FROM task_run tr
    JOIN task t ON t.project = tr.project AND t.id = tr.task_id
    WHERE tr.project     = (regexp_match(r.payload->>'taskRun', 'projects/([^/]+)/'))[1]
      AND tr.id          = (regexp_match(r.payload->>'taskRun', 'taskRuns/(\d+)$'))[1]::bigint
      AND tr.task_id     = (regexp_match(r.payload->>'taskRun', 'tasks/(\d+)/'))[1]::int
      AND tr.created_at <= r.created_at
      AND (
        -- task.payload.source is a oneof: a sheet hash, or a release naming one
        t.payload->>'sheetSha256' = r.payload->>'sheetSha256'
        OR EXISTS (
          SELECT 1 FROM release rel2
          CROSS JOIN LATERAL jsonb_array_elements(COALESCE(rel2.payload->'files', '[]'::jsonb)) AS f2
          WHERE rel2.project      = (regexp_match(t.payload->>'release', 'projects/([^/]+)/'))[1]
            AND rel2.release_id   = (regexp_match(t.payload->>'release', 'releases/([^/]+)'))[1]
            AND f2->>'sheetSha256' = r.payload->>'sheetSha256'
        )
      )
  );
```

A large count means either widespread project purges, reused project IDs, or a population of
revisions created without provenance — each worth understanding before enforcing, since all of them
end in unreadable statements.

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

No schema needed; worth landing separately and first.

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
3. **Add audit annotations** to `proto/v1/v1/sheet_service.proto`, which declares none while 20 other
   v1 services do, leaving sheet reads and enumeration unlogged.
4. **Drop `bb.sheets.update`** — granted to four roles, declared by no RPC, dead since sheets became
   immutable.
5. **Comment the GC hazard on `sheet_blob`.** Nothing deletes from it; the only foreign key that ever
   pointed at it (`task_run.sheet_sha256`, migration 3.14) was dropped in `7192406c5f`, so every
   surviving reference lives in JSONB and is invisible to referential integrity. A GC written the
   obvious way — delete blobs no FK points at — would empty the table. `sheet_blob_ref` makes a
   correct GC possible later.

## Tests

- **Cross-tenant read** — seed a blob in workspace W1, request it under a project in W2, expect
  NotFound. The regression test for the severe half.
- **Cross-project read** — same workspace, different project, expect NotFound. The regression test
  for R6.
- **Cache ordering** — warm the cache under the owning project, then request the same hash under a
  foreign project and assert NotFound. Fails if enforcement ever migrates into `getSheet` behind the
  cache read.
- **Gate confinement** — nothing under `backend/api/v1/` calls a store content getter outside the
  gate.
- **Query count** — a release with many files must issue a bounded number of queries. A correctness
  test passes just as happily against an N+1.
- **Backfill** — exercise all five sources; every referenced hash resolves under its project and no
  unreferenced hash does.
- **Multi-project hashes** — seed a plan in project B referencing a hash created in project A, run
  the backfill, assert the audit query reports it. The point is that it is detected, not blessed.
- **Project purge** — purge a project holding sheet refs and assert it succeeds. Direct regression
  test for the new foreign key.
- **Create versus purge** — `CreateSheets` concurrent with a purge of its project ends in a
  controlled NotFound, not a foreign-key error, in both lock-acquisition directions.
- **`HasSheets`** no longer answers for a foreign project's hash.
- **`TestCollision_Sheet`** per the composite-PK convention in `AGENTS.md`. The shared
  `setupCollidingProjects` fixture and `assertProjectUnchanged` helper do not currently cover sheets;
  extend them rather than writing a table-specific variant.

## Order

1. The five independent fixes, T6 first.
2. Create `sheet_blob_ref` and land the ref *writers* — the transactional `CreateSheets` and the
   purge delete. No scope check reads the table yet.
3. Run the backfill once the writers are live everywhere, then the verification queries; review the
   multi-project list and the ambiguous-provenance list.
4. Land the scope checks — the read gate and the scoped validators — both in shadow mode.
5. Re-run the backfill, then flip to enforcing one release later, once shadow logs are clean.

**Writers must precede the backfill, or new sheets fall into the gap.** A one-shot backfill scans
history; it cannot cover a sheet created after it runs. If the migration ships before `CreateSheets`
writes refs, every sheet created in between lands in `sheet_blob` with no ref and appears in no
historical source, so it becomes a zero-ref miss and a NotFound at enforcement. A rolling upgrade
widens the same gap even when both ship together, since old replicas keep serving the old
`CreateSheets` against the new schema until the rollout completes.

The backfill is idempotent (`ON CONFLICT DO NOTHING`), so the safe pattern is to run it after the
rollout has fully converged and to re-run it immediately before flipping to enforcing. Shadow-mode
misses are the backstop: anything the backfill missed shows up there as a log line rather than as a
production NotFound.
