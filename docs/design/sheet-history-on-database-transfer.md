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
- When a revision's sheet is not readable by the caller, the statement is withheld and the UI says
  so explicitly — naming the owning project when the revision carries one — rather than rendering a
  failed content load. The sheet read itself still returns NotFound, per the scoping doc's
  no-existence-confirmation rule; the *revision* response is what carries the truthful owner.
- The sheet *name* is formatted from the stored authoring project, not from the database's current
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

The owner is a stored fact, not a read-time derivation. `RevisionPayload.project` carries the
authoring project: the server stamps it at creation from the database's then-current project —
after validating that the project holds a ref for the revision's hash — and the runners stamp it
the same way when recording completed task runs. `convertToRevision` formats `Revision.sheet` from
the stamp and nothing else; an empty stamp means no sheet name.

Rows predating the field are backfilled once, by migration, from the revision's own provenance —
`release` (`projects/{project}/releases/{release}`), else `taskRun`
(`projects/{project}/plans/…/taskRuns/{taskRun}`) — **corroborated by exactly the rule the backfill
uses**: single-row identity, age, and hash match, as specified in
[the backfill section](sheet-blob-scoping.md#backfill). Corroborated rows get the stamp and a ref;
everything else stays unstamped.

**Provenance is a hint, not an assertion, which is why the backfill corroborates it.** Both fields
are stored verbatim from the caller. Before the T6 fix, `createRevisions` resolved a
`revision.Release` with no database-project comparison and never compared a `taskRun`'s task to
`revision.Sheet` — so a historical revision can carry provenance naming another tenant's project,
or naming a perfectly real project in the caller's own workspace that had nothing to do with the
sheet. A backfill that trusted raw provenance would turn either into a stamp; corroboration is what
makes the stored fact trustworthy. New writers need no corroboration because the server stamps from
its own knowledge, and the T6 checks now reject cross-project provenance at creation.

| Row | Sheet name in the response |
|---|---|
| Stamped at creation, or backfill-corroborated | Named under the stamp |
| Backfill could not corroborate — stale, purged, or fabricated provenance | No name |
| No provenance to corroborate | No name |

An unstamped row's response must never echo a project ID that failed the check — a reason is not an
identifier, and misattribution is not a harmless default: a user told that project `payments-core`
owns a statement it never authored will go and ask them for it, and may be granted access on a
false premise. The stamp makes this cheap to honor: naming reads a stored field rather than
re-running a resolution that could disagree with the one that granted refs.

**`sheet_blob_ref` is not a fallback for authorship.** An earlier draft resolved a provenance-less
revision by finding which projects hold a ref for its hash and naming the one candidate in the
caller's workspace. That is not evidence about *this revision*: a ref proves only that some project
authored identical content. Content-addressed dedup makes identical content the normal case rather
than a coincidence — `getCreateDatabaseStatement` emits `CREATE DATABASE %s;` for several engines,
so exactly-one collisions on boilerplate are plausible rather than contrived, and the answer would
be a project that had nothing to do with the revision.

Naming it would send the user to request access from a project that never authored the statement,
which is the misattribution this section already rejects for uncorroborated provenance. Applying the
corroboration standard to provenance and a weaker inference to the ref table would be the same
inconsistency in a different place.

So there is no ref-table fallback. A revision names an owner when it carries a stamp, and carries a
stamp only when the server knew the owner at creation or the backfill corroborated one.

**Purge timing splits into two outcomes, deliberately.** A project purged *before* the migration
leaves its surviving revisions unstampable — provenance naming it can no longer corroborate, since
`DeleteProject` hard-deletes the `release`, `task` and `plan` rows — so those rows carry no name.
A project purged *after* its revisions were stamped leaves the stamp behind: the revision keeps
naming the purged project, the name 404s on read (purge deleted the refs), and the UI reports the
statement as owned by a project the caller cannot read. The dangling name is accepted rather than
scrubbed — purge does not rewrite revision payloads, the name was true when stamped, and access
fails closed regardless. The residual is misattribution-by-reuse: a new project created under the
purged ID inherits the dangling names' *labels* (never their content — it holds no refs for those
hashes). The corroboration age test keeps the same reuse from minting stamps in the backfill.

Attribution never implies access. A purged project holds no refs and cannot be granted any, since
`sheet_blob_ref.project` has a live foreign key.

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

- No new response field. The withheld state is derivable client-side from fields that already
  exist: a revision with `sheet_sha256` but no `sheet` name has no known owner, and a named sheet
  whose `GetSheet` fails is owned by the named project. `RevisionDetailPanel` renders each as an
  informational alert — naming the owning project in the second case — rather than an error.
- `RevisionPayload.project` (`proto/store/store/revision.proto`) stores the authoring project.
  `convertRevision` and the task-run executor stamp it at creation; migration 3.22.5 backfills
  corroborated rows.
- `convertToRevision` formats `Revision.sheet` from the stamp rather than from
  `database.ProjectID`, and emits no sheet name when the stamp is empty. Leaving it built from the
  current project would keep exactly the false resource name the scoping doc's R7 exists to
  eliminate — a response handing the client `projects/B/sheets/{sha}` to follow, which then 404s.
- No *carry* logic in `UpdateDatabase`, `BatchUpdateDatabases`, `updateInstanceLifecycle`, or
  `DeleteProject` — nothing moves refs between projects. `DeleteProject` still needs its
  `DELETE FROM sheet_blob_ref WHERE project = ?` from the scoping doc: `sheet_blob_ref(project)` has
  a plain foreign key with no `ON DELETE CASCADE`, so purging a project that owns refs fails without
  it. "No changes" here means no transfer behavior, not no purge cleanup.

Tests, as implemented:

- `TestSheetHistoryOnDatabaseTransfer` — a revision authored under project A, database moved to B:
  the list is complete under B, the sheet stays named under A, a caller with rights on A reads the
  statement through that name, and B gains no read access of its own.
- `TestSheetHistoryAfterOwnerPurge` — the authoring project is purged after stamping, the database
  lands in the default project: the list is complete, the dangling name still renders, and reading
  it returns NotFound. This is the case most likely to regress — not a user-facing transfer, and
  the one where `sheet_blob_ref` has already been deleted.
- The backfill matrix (`TestMigration3_22_5_ScopeSheetBlob`) carries the naming negatives that were
  previously listed as read-path tests, since naming is now decided where the stamp is written: a
  revision with no provenance stays unstamped even when another project holds a ref for the
  identical hash; provenance naming a ghost project, a project that never authored the hash, or a
  reused ID (the age test) stays unstamped rather than misattributed. The instance-archive transfer
  path needs no dedicated test because no per-path logic exists — naming reads the stamp and no
  path carries refs, so all four reassignment paths share one behavior by construction.

## Open items

**Revisions with no stamp have no sheet name.** Where the stamp exists, the name is correct and
unchanged for every database that never moved. Where it does not, the response carries no sheet
name at all — a behavior change for clients that assume the field is always populated, and the
residual cost of refusing to emit a name that is not true.

**Stamped names can dangle after a purge.** A post-migration purge leaves stamped revisions naming
a project that no longer exists; the name 404s and the UI reports an owner the caller cannot read.
Accepted deliberately — see [Naming the owner](#naming-the-owner) — with ID reuse inheriting labels
but never content.

**Workspace-level roles have no escape hatch.** The gate checks the ref table for the named project,
so workspace admins and DBAs are refused as well, despite holding `bb.sheets.get` broadly. Whether
workspace-scoped roles should bypass project scoping is a policy decision worth making deliberately
rather than inheriting from the mechanism.

**`plan_title` crosses the boundary today.** It is derived by parsing the source project out of the
stored `taskRun` string and joining through it (`backend/store/changelog.go:163-168`), so a changelog
in the destination project surfaces the source project's plan title. Minor and pre-existing, but it
is the one place changelog display crosses a project line, and it should be fixed or accepted
deliberately.
