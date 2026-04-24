# Backup Schema Sync Async Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move `backupData`'s backup-phase schema metadata refresh off the migration task run critical path by enqueueing the existing async schema sync.

**Architecture:** Keep the current `backupData` control flow and replace only the final synchronous schema sync calls with `SyncDatabaseAsync`. Non-Postgres still targets the backup database; Postgres still targets the source database because backup tables live in a backup schema inside that database. The async queue is best-effort in HA, which is acceptable because this sync only refreshes metadata and does not feed task result, changelog, revision, or sync history persistence.

**Tech Stack:** Go, existing `backend/runner/taskrun` executor, existing `backend/runner/schemasync.Syncer.SyncDatabaseAsync`.

---

### Task 1: Replace Backup Schema Sync With Async Enqueue

**Files:**
- Modify: `backend/runner/taskrun/database_migrate_executor.go:889-903`

- [ ] **Step 1: Record the current synchronous call sites**

Run:

```bash
rg -n "SyncDatabaseSchema\\(ctx, (backupDatabase|database)\\)" backend/runner/taskrun/database_migrate_executor.go
```

Expected output before implementation:

```text
890:		if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, backupDatabase); err != nil {
897:		if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
```

- [ ] **Step 2: Replace the blocking calls with async enqueue**

Edit `backend/runner/taskrun/database_migrate_executor.go` and replace the block at the end of `backupData` with:

```go
	if database.Engine != storepb.Engine_POSTGRES {
		exec.schemaSyncer.SyncDatabaseAsync(backupDatabase)
	} else {
		exec.schemaSyncer.SyncDatabaseAsync(database)
	}
```

Remove the `slog.Error` blocks that only handled synchronous sync failures. `SyncDatabaseAsync` does not return an error, and execution failures are handled later by the schema syncer when it drains the async queue.

- [ ] **Step 3: Verify no backupData synchronous sync remains**

Run:

```bash
rg -n "SyncDatabaseSchema\\(ctx, (backupDatabase|database)\\)" backend/runner/taskrun/database_migrate_executor.go
```

Expected output after implementation: no matches.

- [ ] **Step 4: Verify both async branch targets are present**

Run:

```bash
rg -n "SyncDatabaseAsync\\((backupDatabase|database)\\)" backend/runner/taskrun/database_migrate_executor.go
```

Expected output after implementation:

```text
890:		exec.schemaSyncer.SyncDatabaseAsync(backupDatabase)
892:		exec.schemaSyncer.SyncDatabaseAsync(database)
```

- [ ] **Step 5: Format the modified Go file**

Run:

```bash
gofmt -w backend/runner/taskrun/database_migrate_executor.go
```

- [ ] **Step 6: Run targeted tests**

Run:

```bash
go test -v -count=1 ./backend/runner/taskrun
```

Expected: package passes.

- [ ] **Step 7: Commit the implementation**

Run:

```bash
git add backend/runner/taskrun/database_migrate_executor.go
git commit -m "fix: async backup schema sync"
```

---

### Task 2: Repository Verification

**Files:**
- Verify: `backend/runner/taskrun/database_migrate_executor.go`

- [ ] **Step 1: Run Go lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: no issues. If lint reports issues, run the fix command in Step 2.

- [ ] **Step 2: Run Go lint auto-fix if needed**

Run only if Step 1 reports fixable issues:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

Expected: the final lint command reports no issues.

- [ ] **Step 3: Build backend**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: build succeeds.

- [ ] **Step 4: Confirm final diff is scoped**

Run:

```bash
git diff --stat HEAD~1..HEAD
git status --short
```

Expected:

```text
backend/runner/taskrun/database_migrate_executor.go | ...
```

`git status --short` may still show the pre-existing untracked `frontend/.vscode/settings.json`; do not stage or modify it.
