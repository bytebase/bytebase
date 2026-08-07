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
| `updateInstanceLifecycle` | `backend/store/instance.go:438-455` | every database of an instance, on project-instance archive |
| `DeleteProject` | `backend/store/project.go:722-729` | workspace-instance databases, reassigned to the default project |

The last is easy to miss: purge does not delete workspace-instance databases, it reassigns them to
the default project, and the `revision` delete at `:702` covers only project instances — so those
revisions survive a purge with their hashes intact.

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
  response says so explicitly, naming the owning project, rather than returning NotFound for the
  sheet.
- No sheet references are carried between projects. A database transfer grants no new read access,
  and the four reassignment paths above need no changes.

This satisfies R1 through R3 by keeping the list intact and R4 by leaving statement text with the
project that authored it. It also makes revisions consistent with how changelogs already behave,
rather than introducing two rules for two views of the same history.

The explicit withholding matters as much as the withholding itself. A destination team that sees
"statement owned by project `payments-core`" can make an informed request to the right owner. A team
that sees a list of failed content loads sees a broken product.

The owning project is derivable without new storage: `sheet_blob_ref` records which projects hold a
ref for a hash, so the withheld response can name them. This discloses that some other project holds
SQL with that hash, which is acceptable within a workspace and is strictly less than the content
itself.

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

- A field on the revision response carrying the withheld-content reason and the owning project, plus
  the handler change to populate it from `sheet_blob_ref`.
- A UI affordance for the withheld state that names the owning project rather than rendering an
  error.
- No changes to `UpdateDatabase`, `BatchUpdateDatabases`, `updateInstanceLifecycle`, or
  `DeleteProject`.

Tests: create a revision under project A, move the database to project B, then assert the revision
list is complete under B while the statement is withheld with A named. Repeat for the purge path,
where a workspace-instance database is reassigned to the default project — the case most likely to
regress, since it is not a user-facing transfer.

## Open items

**Sheet name attribution.** `convertToRevision` formats the sheet name with the database's current
project, which is incorrect after a transfer: it names the destination as owner of a sheet the source
authored. Deriving the owner from `sheet_blob_ref` covers the withheld-content response, but the
`name` field itself remains wrong. Recording the authoring project on the revision at creation would
fix it for new rows; rows predating the migration cannot recover it.

**Workspace-level roles have no escape hatch.** The gate checks the ref table for the named project,
so workspace admins and DBAs are refused as well, despite holding `bb.sheets.get` broadly. Whether
workspace-scoped roles should bypass project scoping is a policy decision worth making deliberately
rather than inheriting from the mechanism.

**`plan_title` crosses the boundary today.** It is derived by parsing the source project out of the
stored `taskRun` string and joining through it (`backend/store/changelog.go:163-168`), so a changelog
in the destination project surfaces the source project's plan title. Minor and pre-existing, but it
is the one place changelog display crosses a project line, and it should be fixed or accepted
deliberately.
