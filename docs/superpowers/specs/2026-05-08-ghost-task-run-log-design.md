# gh-ost Task Run Log Design

## Context

Bytebase task run logs currently show normal SQL execution through command entries and post-execution database sync through timed sync entries. The gh-ost migration path creates task run log write options, but the gh-ost execution itself does not emit task run logs. Users only see later database sync logs, which hides the longest and most operationally important part of an online schema migration.

The first change should make gh-ost visible and distinguishable in the task run log without adding progress streaming yet.

## Goals

- Show a dedicated gh-ost migration section in task run logs.
- Record start time, end time, statement context, and execution error.
- Keep release-file grouping behavior intact for versioned releases.
- Leave room for future gh-ost progress logs without overbuilding the first step.

## Non-Goals

- Do not stream gh-ost progress in the first implementation.
- Do not redesign the task run log viewer.
- Do not encode gh-ost progress as fake command responses or plain text logs.

## Design

Add a new task run log pair for gh-ost execution:

- Store proto: `GHOST_MIGRATION_START`
- Store proto: `GHOST_MIGRATION_END`
- v1 API entry type: `GHOST_MIGRATION`

The public API should collapse the store start/end pair into one timed entry, matching the existing pattern for database sync, prior backup, schema dump, and compute diff.

The v1 `GHOST_MIGRATION` payload should include:

- `statement`: the cleaned ALTER statement passed to gh-ost.
- `start_time`: when gh-ost execution starts.
- `end_time`: when gh-ost execution finishes.
- `error`: the failure message, if gh-ost fails or the task is canceled.

Do not include parsed table context in the first implementation. The user-facing distinction comes from the log type and label, while `statement` gives enough execution context. Table-specific fields can be added later if the progress UI needs them.

## Backend Flow

`runGhostMigration` already creates `db.ExecuteOptions` with `CreateTaskRunLog`. Pass those options into `executeGhostMigration`.

Inside `executeGhostMigration`:

1. Parse flags and clean Bytebase directives.
2. Trim the cleaned ALTER statement.
3. Parse the target table and create the gh-ost migration context.
4. Emit `GHOST_MIGRATION_START` after the statement and context are ready, before `migrator.Migrate()`.
5. Run gh-ost.
6. Emit `GHOST_MIGRATION_END` with an empty error on success.
7. Emit `GHOST_MIGRATION_END` with the error string on migration failure or task cancellation.

The existing database sync logs should remain after gh-ost completes. The task run log sequence should usually be:

1. `GHOST_MIGRATION`
2. `DATABASE_SYNC`

For release files, the existing `RELEASE_FILE_EXECUTE` marker remains first, so gh-ost logs are grouped under the correct release file.

## API Conversion

Extend `convertToTaskRunLogEntries` to collapse `GHOST_MIGRATION_START` and `GHOST_MIGRATION_END` into one v1 `GHOST_MIGRATION` entry.

The first implementation may follow the existing adjacent-pair converter behavior because no gh-ost progress entries will be emitted between start and end. When progress entries are added later, the converter should either pair by open entry type rather than adjacency or model progress as child entries under a migration entry.

## Frontend

Add a task run log label:

- `task-run.log-type.ghost-migration`: `gh-ost migration`

Render the entry with the same timed-section UI used by database sync and compute diff. The visible difference is the section name and type, not a special layout.

The detail text should behave like other timed sections:

- Running: reuse the existing timed running detail text.
- Completed: existing completed text.
- Failed: error string.

## Future Progress Design

A later phase can add `GHOST_MIGRATION_PROGRESS` or nested progress details with fields such as:

- copied rows
- estimated rows
- applied DML events
- backlog
- lag
- heartbeat lag
- state
- ETA

That phase should include log throttling and UI design. It should not be mixed into the minimal start/end implementation.

## Testing

Backend tests should verify that a gh-ost migration task run returns a task run log containing:

- one `GHOST_MIGRATION` entry
- a non-empty statement
- start and end timestamps
- no error on success
- a following `DATABASE_SYNC` entry

Frontend model tests should cover the new entry type as a timed section, including running, completed, and error states.

## Decisions

- First implementation stores only the cleaned gh-ost statement, not parsed table context.
- First implementation uses a gh-ost-specific section label and reuses existing timed detail text.
