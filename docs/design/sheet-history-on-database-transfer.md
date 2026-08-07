# Database change history across project transfers

Companion to [`sheet-blob-scoping.md`](sheet-blob-scoping.md), which scopes sheet content to the
owning project. This document decides what a project sees of a database's change history after the
database is transferred into it: **the history list follows the database, the statement text does
not.**

## Background

`revision` and `changelog` have no project column. They key on `(instance, db_name)` and inherit
their project through `db.project`, so when `UPDATE db SET project = ?` runs, the history moves with
the database. This was never a design decision; it falls out of the schema. `convertToRevision` then
formats the sheet name with the database's new project
(`backend/api/v1/revision_service.go:304`), so a project-scoped sheet gate denies statements that
were readable moments earlier.

Four code paths move a database between projects, only one of which is a user-facing transfer:

| Path | Location | Scope |
|---|---|---|
| `UpdateDatabase` | `backend/store/database.go:332-380` | one database |
| `BatchUpdateDatabases` | `backend/store/database.go:397-560` | a batch |
| `updateInstanceLifecycle` | `backend/store/instance.go:373-455` | every database of a **workspace** instance, on archive |
| `DeleteProject` | `backend/store/project.go:722-729` | workspace-instance databases, reassigned to the default project |

Both instance-related paths concern **workspace** instances specifically, and for the same reason:
those instances are shared infrastructure, so their databases cannot simply be deleted along with a
project and must be reassigned somewhere.

`updateInstanceLifecycle` transfers databases only when archiving a workspace instance —
`MoveDatabasesToProjectID` is documented as "only valid when archiving a resource-scoped workspace
instance" (`backend/store/instance.go:44-46`), and `:434` rejects the project-instance case outright
with *cannot transfer databases while archiving project instance*. It also requires `Deleted` to be
set (`:288`), so it is an archive path rather than an ordinary update.

`DeleteProject` is the same shape and the one most easily missed: purge does not delete
workspace-instance databases, it reassigns them to the default project, and the `revision` delete at
`:702` covers only project instances — so those revisions survive a purge with their hashes intact.
Project-instance databases are deleted with their instance in both paths and raise no question.

## Requirements

**R1 — The applied-version ledger stays complete regardless of project.** `revision` carries a
`version` column and serves as Bytebase's equivalent of `flyway_schema_history`: the record of which
versioned migrations have already run against a database. Hiding revisions from the project
currently holding the database risks a versioned release re-running migrations that already applied.
This is a correctness constraint, not a preference.

**R2 — The audit record survives reorgs intact.** Regulated customers must answer "what changed on
this database, when, and by whom" for the full life of the database, across ownership changes. A
database moving between teams is an administrative event and must not create a gap in the compliance
record.

**R3 — The inheriting team can operate the database.** Whoever takes ownership needs to know what has
been applied. Handing a team a production database whose history is a black box is a worse outcome
than the disclosure it avoids.

**R4 — Projects remain a confidentiality boundary.** A database transfer that silently grants the
destination project read access to statements the source project authored is a policy violation, and
one performed by an operation that does not present itself as an access change. The principals who
gain access are not the person performing the transfer — who already holds rights on both sides —
but the destination project's other members.

R1 through R3 are satisfied by any design that keeps the history list complete. R4 is the only
requirement that discriminates, and it concerns statement text rather than the list.

## Current behavior

### Changelogs carry no SQL

The v1 `Changelog` message (`proto/v1/v1/changelog_service.proto:105`) carries `name`, `create_time`,
`status`, `schema` and `schema_size`, `task_run`, and `plan_title`. There is no statement and no
sheet field; fields 4–6, 9–10 and 12–14 are reserved, having been removed as the model evolved.
`backend/api/v1/changelog_service.go` populates no statement either.

A changelog therefore answers when a change ran, whether it succeeded, what the schema looked like
afterwards, and which rollout produced it — entirely from its own row. The `schema` snapshot is the
load-bearing part: consecutive changelogs diff to show what changed structurally without reference
to the SQL text. R2 rests almost entirely on this record, and no part of it depends on sheet scoping.

### Changelog SQL already stops at the project boundary

The statement behind a changelog is reachable only through `task_run` → `task.payload.sheetSha256`.
`task` is project-scoped with its own column and does not move with the database. The stored pointer
keeps naming the source project (`FormatTaskRun(database.ProjectID, ...)` at
`backend/runner/taskrun/database_migrate_executor.go:273`), and resolving it requires rights on that
project.

Changelogs already implement *metadata follows the database, SQL stays with the authoring project*.
That behavior predates this work.

### Revisions carry a hash directly

`RevisionPayload.sheet_sha256` names the sheet inline, so revisions — unlike changelogs — surface
statement content through the sheet gate. They are the only history record this decision affects.

## Design

Change history follows the database; statement content does not.

- Revision and changelog lists are returned in full after a transfer, including entries created under
  a previous project. Versions, timestamps, authors, status, schema snapshots, and the sheet *name*
  are all present.
- When a revision's sheet is not readable by the current project, the statement is withheld and the
  response says so explicitly — naming the owning project where that can be verified — rather than
  returning NotFound for the sheet.
- The sheet *name* is formatted from that same resolved owner, not from the database's current
  project. A name is emitted only when it is true.
- No sheet references are carried between projects. A database transfer grants no new read access,
  and the four reassignment paths above need no changes.

This satisfies R1 through R3 by keeping the list intact and R4 by leaving statement text with the
project that authored it. It also makes revisions consistent with how changelogs already behave,
rather than introducing two rules for two views of the same history.

The explicit withholding matters as much as the withholding itself. A destination team that sees
"statement owned by project `payments-core`" can make an informed request to the right owner. A team
that sees a list of failed content loads sees a broken product.

### Naming the owner

The owning project is derivable without new storage, from the revision's own provenance.
`RevisionPayload` carries `release` (`projects/{project}/releases/{release}`) and `task_run`
(`projects/{project}/plans/…/taskRuns/{taskRun}`), both embedding the authoring project in the
resource name; `backend/store/changelog.go:163-168` already parses a project out of a stored
`taskRun` this way. Resolution order:

1. The revision's `release`, else its `taskRun` — **corroborated by exactly the rule the backfill
   uses**, and resolving to a project in the caller's workspace.
2. `sheet_blob_ref` — which projects currently hold a ref for that hash, **joined through `project`
   and constrained to the caller's workspace**, and only when that leaves exactly one candidate.
   Useful when the revision carries no provenance, both fields being optional.
3. Anything else — report the owner as unknown, and never echo a project ID that failed either test.

**Provenance is a hint, not an assertion.** Both fields are stored verbatim from the caller:
`convertRevision` writes `Release: revision.Release` and `TaskRun: revision.TaskRun` straight through
(`backend/api/v1/revision_service.go:352-364`). Until the T6 fix lands, `createRevisions` resolves
`revision.Release` with no workspace or database-project comparison (`:207-223`), and it never
compares a `taskRun`'s task to `revision.Sheet` at all — so a revision can carry provenance naming
another tenant's project, or naming a perfectly real project in the caller's own workspace that had
nothing to do with the sheet.

**Naming therefore uses the same corroboration rule as the backfill, not a weaker one.** Identity,
age, and hash match, exactly as specified in
[the backfill section](sheet-blob-scoping.md#backfill). A design in which the backfill trusts
corroborated provenance while the response trusts raw provenance has two different notions of
provenance in it, and the weaker one would be the user-visible one.

| Provenance | Response |
|---|---|
| Corroborated, project in the caller's workspace | Name it |
| Corroborated, project in another workspace | Owner unknown |
| Present but not corroborated — stale, purged, or fabricated | Owner unknown |
| Absent | Fall through to step 2 |

The response may still distinguish *why* it is unavailable — no provenance, provenance that cannot
be verified, an owner outside the caller's workspace — since a reason is not an identifier. What it
must never do is echo a project ID that failed a check. Misattribution is not a harmless default: a
user told that project `payments-core` owns a statement it never authored will go and ask them for
it, and may be granted access on a false premise.

This applies to *naming*, not to the backfill's use of the same fields. Granting a ref to the project
that genuinely authored a sheet is correct regardless of which workspace the referencing revision
sits in — the backfill's corroboration check is about attributing ownership, while this is about what
a caller is allowed to be told.

Step 2 must not read the ref table unqualified. `sheet_blob_ref` has no workspace column, and
content-addressed dedup means one blob is shared by every project that authored identical SQL —
across tenants. Generated boilerplate makes that routine rather than rare: `getCreateDatabaseStatement`
emits `CREATE DATABASE %s;` for several engines, so a common statement will carry refs from many
workspaces. Naming an arbitrary one would leak a foreign tenant's project name from a design whose
purpose is closing cross-tenant leakage.

```sql
SELECT r.project
FROM sheet_blob_ref r
JOIN project p ON p.resource_id = r.project
WHERE r.sha256 = ? AND p.workspace = ?
```

Ambiguity resolves to unknown rather than to a guess. Two projects in the same workspace holding
refs for one hash is honest dedup, and neither is more the author than the other.

A purged authoring project resolves to unknown, not to a name. `DeleteProject` hard-deletes the
`release`, `task` and `plan` rows along with the project, so provenance naming it can no longer
corroborate, and `sheet_blob_ref WHERE project = A` is deleted too, so step 2 finds nothing either.
That is the correct outcome rather than a gap: the statement is permanently unavailable, there is
nobody left to ask, and the only thing withheld beyond the content is an ID that can no longer be
verified against any workspace.

Step 1 is attribution only and never implies access. A purged project holds no refs and cannot be
granted any, since `sheet_blob_ref.project` has a live foreign key, and the backfill skips
uncorroborated provenance for the same reason.

The response should distinguish the two outcomes rather than flattening them: a live project the
caller can ask, versus an authoring project that no longer exists and whose content is now
unreadable by anyone. The second is a permanent consequence of purging a project, consistent with
the zero-ref blobs described in the scoping doc.

Naming the owner discloses that some other project holds SQL with that hash. That is acceptable
within a workspace and strictly less than the content itself.

## Alternatives considered

**Carry sheet references on transfer.** On any project change, insert ref rows for the destination
covering every hash in that database's revisions. Preserves current behavior exactly and grants
narrowly — only hashes that database's own revisions reference. It fails R4: it widens access as a
side effect of an operation that is not about access, for principals who were not party to it.

It is also the most machinery of any option, and the machinery is where the risk lives. A carry
helper must be wired into all four paths, purge additionally requires carrying before the source
project's refs are deleted, and each path needs its own test because the failure is silent until a
later read. Three of the four paths were missed on first analysis; only `BatchUpdateDatabases` is an
obvious transfer, and the other three are consequences of unrelated operations.

**Withhold with no signal.** Carry nothing and let the gate return NotFound. Costs nothing and
satisfies R4, but produces a history list where every statement fails to load with no indication of
why or where to look. This is the default outcome if the decision is not made explicitly, which is
the main reason to make it explicitly.

**Project-scope revisions and changelogs.** Add a `project` column to both, set at creation, and
filter reads to the caller's project, so a database transferred into project B shows only history
created while it was in B. This is the cleanest authorization story — history becomes project-scoped
like `plan`, `task`, `issue`, and `release` — and it would also resolve the dangling `taskRun`
pointer noted below. It fails R1: a transferred database would appear to have no applied versions,
and a versioned release would re-run migrations that already ran. A reorg-triggered path to data
corruption outweighs the modelling improvement.

> This rejection rests on revisions being what determines whether a version is already applied.
> Confirm against `runVersionedRelease` in
> `backend/runner/taskrun/database_migrate_executor.go` before treating it as settled. If revisions
> do not drive version-skip logic, this alternative becomes considerably more attractive.

**Delete history on transfer.** Fails R1, R2, and R3 simultaneously. Recorded for completeness.

## Implementation

- A field on the revision response carrying the withheld-content reason and the owning project,
  populated by the resolution order above — provenance first, `sheet_blob_ref` second, unknown last.
- `convertToRevision` formats `Revision.sheet` from the resolved owner rather than from
  `database.ProjectID` (`backend/api/v1/revision_service.go:304`), and emits no sheet name when the
  owner does not resolve. Leaving it built from the current project would keep exactly the false
  resource name R7 exists to eliminate — a response asserting that project A owns the statement while
  handing the client `projects/B/sheets/{sha}` to follow, which then 404s. The owner is already
  computed for the withheld-content field, so this costs nothing extra.
- A UI affordance for the withheld state that names the owning project rather than rendering an
  error, and says the project no longer exists when that is the case.
- No *carry* logic in `UpdateDatabase`, `BatchUpdateDatabases`, `updateInstanceLifecycle`, or
  `DeleteProject` — nothing moves refs between projects. `DeleteProject` still needs its
  `DELETE FROM sheet_blob_ref WHERE project = ?` from the scoping doc: `sheet_blob_ref(project)` has
  a plain foreign key with no `ON DELETE CASCADE`, so purging a project that owns refs fails without
  it. "No changes" here means no transfer behavior, not no purge cleanup.

Tests:

- Create a revision under project A, move the database to project B, assert the revision list is
  complete under B while the statement is withheld with A named — A's provenance corroborates, and A
  is in the caller's workspace.
- Purge path: put a database on a *workspace* instance in project A, create a revision, purge A, then
  assert that under the default project the list is still complete, the statement is withheld, and
  the response reports the owner as unknown and names no project.
  This is the case most likely to regress — it is not a user-facing transfer, and it is the one
  where `sheet_blob_ref` has already been deleted, so it exercises the provenance path specifically.
- Instance archive: put a database on a **workspace** instance in project A, create a revision, then
  archive the instance with `MoveDatabasesToProjectID` set to project B, and assert the revision list
  is complete under B while the statement is withheld with A named. Archiving a *project* instance is
  the wrong path — `backend/store/instance.go:434` rejects it — so a test written that way would pass
  vacuously without exercising any transfer.
- A revision carrying neither `release` nor `taskRun` falls through to `sheet_blob_ref`, and to
  unknown when that is empty too.
- Cross-tenant leak guard, both paths: (a) seed identical SQL in two workspaces so one blob carries
  refs from both, then request a withheld statement from one and assert the response never names the
  other workspace's project; (b) seed a revision whose `release` provenance names a project in
  another workspace — the state T6 permits — and assert the response reports unknown rather than
  echoing it; (c) seed a revision naming a real project in the caller's *own* workspace that never
  authored the sheet, and assert the response reports unknown rather than misattributing it. Generated `CREATE DATABASE` boilerplate makes this collision realistic, so it
  is a regression test rather than a contrived one.
- Ambiguity: two projects in the caller's own workspace holding refs for one hash resolves to
  unknown, not to whichever row comes back first.
- Sheet name: after a transfer, `Revision.sheet` names the resolved owner and resolves to that
  project's readable sheet for a caller who holds rights there; when the owner is unknown, no sheet
  name is emitted rather than one pointing at the current project.

## Open items

**Revisions with no resolvable owner have no sheet name.** Where the owner resolves, the name is
correct and unchanged for every database that never moved. Where it does not, the response carries
no sheet name at all — a behavior change for clients that assume the field is always populated, and
the residual cost of refusing to emit a name that is not true.

**Workspace-level roles have no escape hatch.** The gate checks the ref table for the named project,
so workspace admins and DBAs are refused as well, despite holding `bb.sheets.get` broadly. Whether
workspace-scoped roles should bypass project scoping is a policy decision worth making deliberately
rather than inheriting from the mechanism.

**`plan_title` crosses the boundary today.** It is derived by parsing the source project out of the
stored `taskRun` string and joining through it (`backend/store/changelog.go:163-168`), so a changelog
in the destination project surfaces the source project's plan title. Minor and pre-existing, but it
is the one place changelog display crosses a project line, and it should be fixed or accepted
deliberately.
