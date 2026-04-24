# Backup Schema Sync Async Design

## Context

`backupData` in `backend/runner/taskrun/database_migrate_executor.go` creates prior-backup tables before a migration runs. After creating those backup tables, it currently calls `SyncDatabaseSchema` synchronously:

- non-Postgres engines sync the backup database;
- Postgres syncs the source database because backup tables live in the backup schema inside that database.

This sync only refreshes Bytebase database metadata. Its result is not used to persist the task run result, changelog, revision, or sync history. When the sync is slow, it extends the visible prior-backup phase even though the backup tables have already been created.

Bytebase can run in HA mode. The existing async schema sync path uses an in-memory per-replica queue (`SyncDatabaseAsync`/`databaseSyncMap`) drained by the schema syncer. That queue is best-effort: if the executing replica exits before the queue is drained, the queued refresh can be lost.

## Goal

Remove the backup-phase schema sync from the migration task run critical path while preserving the same eventual metadata refresh target.

## Non-Goals

- Do not add a durable schema-sync request table.
- Do not add task-run `DATABASE_SYNC` log entries for this path.
- Do not change post-migration schema sync behavior, changelog updates, revision behavior, or sync-history behavior.

## Design

At the end of `backupData`, replace the blocking `SyncDatabaseSchema` call with `SyncDatabaseAsync`.

Target selection remains unchanged:

- for non-Postgres engines, enqueue `backupDatabase`;
- for Postgres, enqueue `database`.

The enqueue is intentionally best-effort in HA. This is acceptable because the backup-phase sync is a metadata refresh only. If the async request is lost due to replica shutdown, later periodic sync or manual database sync can repair the metadata.

No task-run database-sync log events are emitted for this path because the actual sync is no longer part of the task run execution. The prior-backup task-run log span should represent backup table creation, not asynchronous metadata refresh.

## Implementation Notes

Keep the existing `if/else` structure at the single call site and replace the synchronous calls with async enqueues:

```go
if database.Engine != storepb.Engine_POSTGRES {
	exec.schemaSyncer.SyncDatabaseAsync(backupDatabase)
} else {
	exec.schemaSyncer.SyncDatabaseAsync(database)
}
```

The existing `SyncDatabaseAsync` method already ignores nil and deleted databases.

## Error Handling

The async enqueue itself does not fail. Sync execution failures are handled by the existing schema syncer path, which records sync failure state on database metadata when the queued sync runs.

No migration task failure should be introduced by the async metadata refresh.

## Testing

Add focused unit coverage for the backup schema sync target behavior:

- Postgres enqueues the source database.
- Non-Postgres enqueues the backup database.
- Nil backup database is tolerated by the async enqueue path because `SyncDatabaseAsync` already handles nil.

Run:

```bash
go test -v -count=1 ./backend/runner/taskrun
```

For final verification after implementation, follow repository Go workflow: `gofmt`, `golangci-lint run --allow-parallel-runners` until clean, relevant tests, and backend build.
