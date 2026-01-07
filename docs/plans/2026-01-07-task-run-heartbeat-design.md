# Task Run Heartbeat Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Detect stale RUNNING task runs in HA deployments by tracking replica heartbeats.

**Architecture:** Each Bytebase replica sends heartbeats to a `replica_heartbeat` table using its `deployId`. Task runs record which replica claimed them. Data cleaner detects stale tasks (replica missing or heartbeat too old) and marks them FAILED.

**Tech Stack:** Go, PostgreSQL, existing runner infrastructure

---

## Task 1: Add Database Migration

**Files:**
- Create: `backend/migrator/migration/3.10/0000##replica_heartbeat.sql`
- Modify: `backend/migrator/migration/LATEST.sql:256-280`
- Modify: `backend/migrator/migrator_test.go` (update version)

**Step 1: Create migration file**

Create `backend/migrator/migration/3.10/0000##replica_heartbeat.sql`:

```sql
-- Create replica heartbeat table
CREATE TABLE replica_heartbeat (
    replica_id TEXT PRIMARY KEY,
    last_heartbeat TIMESTAMPTZ NOT NULL
);

-- Add replica_id column to task_run
ALTER TABLE task_run ADD COLUMN replica_id TEXT;

-- Mark existing RUNNING task runs as FAILED
UPDATE task_run
SET status = 'FAILED',
    result = '{"detail": "Marked as failed during heartbeat migration"}'
WHERE status = 'RUNNING';
```

**Step 2: Update LATEST.sql**

Add to `backend/migrator/migration/LATEST.sql` after task_run table definition (around line 269):

```sql
-- Add after: result jsonb NOT NULL DEFAULT '{}'
    replica_id TEXT
```

Add new table before task_run_log (around line 280):

```sql
CREATE TABLE replica_heartbeat (
    replica_id TEXT PRIMARY KEY,
    last_heartbeat TIMESTAMPTZ NOT NULL
);
```

**Step 3: Update migrator_test.go**

Update `TestLatestVersion` to include version "3.10".

**Step 4: Run migration locally**

```bash
# Verify migration syntax
psql -U bbdev bbdev -f backend/migrator/migration/3.10/0000##replica_heartbeat.sql
```

**Step 5: Commit**

```bash
but commit <branch> -m "chore: add replica heartbeat migration"
```

---

## Task 2: Add Store Methods for Heartbeat

**Files:**
- Create: `backend/store/replica_heartbeat.go`

**Step 1: Create store file**

Create `backend/store/replica_heartbeat.go`:

```go
package store

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

// UpsertReplicaHeartbeat updates or inserts a replica heartbeat.
func (s *Store) UpsertReplicaHeartbeat(ctx context.Context, replicaID string) error {
	q := qb.Q().Space(`
		INSERT INTO replica_heartbeat (replica_id, last_heartbeat)
		VALUES (?, now())
		ON CONFLICT (replica_id)
		DO UPDATE SET last_heartbeat = now()
	`, replicaID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to upsert replica heartbeat")
	}
	return nil
}

// DeleteStaleReplicaHeartbeats deletes heartbeat rows older than the given duration.
func (s *Store) DeleteStaleReplicaHeartbeats(ctx context.Context, olderThan time.Duration) (int64, error) {
	q := qb.Q().Space(`
		DELETE FROM replica_heartbeat
		WHERE last_heartbeat < now() - ?::INTERVAL
	`, olderThan.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to delete stale replica heartbeats")
	}
	return result.RowsAffected()
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/store/...
```

**Step 3: Commit**

```bash
but commit <branch> -m "feat: add replica heartbeat store methods"
```

---

## Task 3: Add Store Method for Stale Task Detection

**Files:**
- Modify: `backend/store/task_run.go`

**Step 1: Add FailStaleTaskRuns method**

Add to `backend/store/task_run.go`:

```go
// FailStaleTaskRuns marks RUNNING task runs as FAILED if their replica is dead.
// A replica is considered dead if:
// 1. Its replica_id is not in the replica_heartbeat table, OR
// 2. Its last_heartbeat is older than the staleness threshold
// Returns the number of task runs marked as failed.
func (s *Store) FailStaleTaskRuns(ctx context.Context, stalenessThreshold time.Duration) (int64, error) {
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?,
		    result = '{"detail": "Task run abandoned: owning replica stopped responding"}',
		    updated_at = now()
		WHERE status = ?
		  AND replica_id IS NOT NULL
		  AND (
		    replica_id NOT IN (SELECT replica_id FROM replica_heartbeat)
		    OR replica_id IN (
		      SELECT replica_id FROM replica_heartbeat
		      WHERE last_heartbeat < now() - ?::INTERVAL
		    )
		  )
	`, storepb.TaskRun_FAILED.String(), storepb.TaskRun_RUNNING.String(), stalenessThreshold.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to fail stale task runs")
	}
	return result.RowsAffected()
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/store/...
```

**Step 3: Commit**

```bash
but commit <branch> -m "feat: add store method to fail stale task runs"
```

---

## Task 4: Modify ClaimAvailableTaskRuns to Set replica_id

**Files:**
- Modify: `backend/store/task_run.go:220-258`

**Step 1: Update ClaimAvailableTaskRuns signature and query**

Modify `ClaimAvailableTaskRuns` in `backend/store/task_run.go`:

```go
// ClaimAvailableTaskRuns atomically claims all AVAILABLE task runs by updating them to RUNNING
// and returns the claimed task run and task UIDs. This combines list + claim into a single atomic operation.
// Uses FOR UPDATE SKIP LOCKED to allow concurrent schedulers to claim different tasks.
func (s *Store) ClaimAvailableTaskRuns(ctx context.Context, replicaID string) ([]*ClaimedTaskRun, error) {
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now(), replica_id = ?
		WHERE id IN (
			SELECT task_run.id FROM task_run
			WHERE task_run.status = ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, task_id
	`, storepb.TaskRun_RUNNING.String(), replicaID, storepb.TaskRun_AVAILABLE.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to claim task runs")
	}
	defer rows.Close()

	var claimed []*ClaimedTaskRun
	for rows.Next() {
		var c ClaimedTaskRun
		if err := rows.Scan(&c.TaskRunUID, &c.TaskUID); err != nil {
			return nil, err
		}
		claimed = append(claimed, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return claimed, nil
}
```

**Step 2: Update caller in running_scheduler.go**

Find the call to `ClaimAvailableTaskRuns` in `backend/runner/taskrun/running_scheduler.go` and add the replicaID parameter:

```go
// Change from:
claimed, err := s.store.ClaimAvailableTaskRuns(ctx)
// To:
claimed, err := s.store.ClaimAvailableTaskRuns(ctx, s.profile.DeployID)
```

**Step 3: Verify it compiles**

```bash
go build ./backend/...
```

**Step 4: Commit**

```bash
but commit <branch> -m "feat: set replica_id when claiming task runs"
```

---

## Task 5: Create Heartbeat Runner

**Files:**
- Create: `backend/runner/heartbeat/runner.go`

**Step 1: Create heartbeat runner**

Create `backend/runner/heartbeat/runner.go`:

```go
package heartbeat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	heartbeatInterval = 10 * time.Second
)

// Runner sends periodic heartbeats to indicate this replica is alive.
type Runner struct {
	store   *store.Store
	profile *config.Profile
}

// NewRunner creates a new heartbeat runner.
func NewRunner(store *store.Store, profile *config.Profile) *Runner {
	return &Runner{
		store:   store,
		profile: profile,
	}
}

// Run starts the heartbeat runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	slog.Debug("Heartbeat runner started", slog.String("replicaID", r.profile.DeployID))

	// Send heartbeat immediately on startup
	r.sendHeartbeat(ctx)

	for {
		select {
		case <-ticker.C:
			r.sendHeartbeat(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) sendHeartbeat(ctx context.Context) {
	if err := r.store.UpsertReplicaHeartbeat(ctx, r.profile.DeployID); err != nil {
		slog.Error("Failed to send heartbeat", log.BBError(err))
	}
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/runner/heartbeat/...
```

**Step 3: Commit**

```bash
but commit <branch> -m "feat: add heartbeat runner"
```

---

## Task 6: Add Stale Detection to Data Cleaner

**Files:**
- Modify: `backend/runner/cleaner/data_cleaner.go`

**Step 1: Add constants and stale detection ticker**

Modify `backend/runner/cleaner/data_cleaner.go`:

```go
const (
	cleanupInterval              = 1 * time.Hour
	staleDetectionInterval       = 30 * time.Second
	stalenessThreshold           = 1 * time.Minute
	heartbeatRetentionPeriod     = 1 * time.Hour
	exportArchiveRetentionPeriod = 24 * time.Hour
	oauth2ClientRetentionPeriod  = 30 * 24 * time.Hour // 30 days of inactivity
)
```

**Step 2: Update Run method with dual tickers**

Replace the `Run` method:

```go
// Run starts the DataCleaner.
func (c *DataCleaner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	cleanupTicker := time.NewTicker(cleanupInterval)
	defer cleanupTicker.Stop()

	staleTicker := time.NewTicker(staleDetectionInterval)
	defer staleTicker.Stop()

	slog.Debug("Data cleaner started",
		slog.Duration("cleanupInterval", cleanupInterval),
		slog.Duration("staleDetectionInterval", staleDetectionInterval))

	// Run cleanup immediately on startup
	c.cleanup(ctx)
	c.detectStaleTaskRuns(ctx)

	for {
		select {
		case <-cleanupTicker.C:
			c.cleanup(ctx)
		case <-staleTicker.C:
			c.detectStaleTaskRuns(ctx)
		case <-ctx.Done():
			return
		}
	}
}
```

**Step 3: Add stale detection and heartbeat cleanup methods**

Add these methods:

```go
func (c *DataCleaner) detectStaleTaskRuns(ctx context.Context) {
	rowsAffected, err := c.store.FailStaleTaskRuns(ctx, stalenessThreshold)
	if err != nil {
		slog.Error("Failed to detect stale task runs", log.BBError(err))
		return
	}
	if rowsAffected > 0 {
		slog.Info("Marked stale task runs as failed", slog.Int64("count", rowsAffected))
	}
}

func (c *DataCleaner) cleanupStaleHeartbeats(ctx context.Context) {
	rowsAffected, err := c.store.DeleteStaleReplicaHeartbeats(ctx, heartbeatRetentionPeriod)
	if err != nil {
		slog.Error("Failed to clean up stale replica heartbeats", log.BBError(err))
		return
	}
	if rowsAffected > 0 {
		slog.Info("Cleaned up stale replica heartbeats", slog.Int64("count", rowsAffected))
	}
}
```

**Step 4: Update cleanup method to include heartbeat cleanup**

```go
func (c *DataCleaner) cleanup(ctx context.Context) {
	c.cleanupExportArchives(ctx)
	c.cleanupOAuth2Data(ctx)
	c.cleanupWebRefreshTokens(ctx)
	c.cleanupStaleHeartbeats(ctx)
}
```

**Step 5: Verify it compiles**

```bash
go build ./backend/runner/cleaner/...
```

**Step 6: Commit**

```bash
but commit <branch> -m "feat: add stale task detection to data cleaner"
```

---

## Task 7: Register Heartbeat Runner in Server

**Files:**
- Modify: `backend/server/server.go`

**Step 1: Add import**

Add to imports:

```go
"github.com/bytebase/bytebase/backend/runner/heartbeat"
```

**Step 2: Add field to Server struct**

Add field (around line 64):

```go
heartbeatRunner    *heartbeat.Runner
```

**Step 3: Initialize runner**

Add after dataCleaner initialization (around line 202):

```go
// Heartbeat runner
s.heartbeatRunner = heartbeat.NewRunner(stores, profile)
```

**Step 4: Start runner**

Add after dataCleaner.Run (around line 239):

```go
s.runnerWG.Add(1)
go s.heartbeatRunner.Run(ctx, &s.runnerWG)
```

**Step 5: Verify it compiles**

```bash
go build ./backend/...
```

**Step 6: Commit**

```bash
but commit <branch> -m "feat: register heartbeat runner in server"
```

---

## Task 8: Run Linter and Fix Issues

**Step 1: Run golangci-lint**

```bash
golangci-lint run --allow-parallel-runners
```

**Step 2: Fix any issues**

Common issues to watch for:
- Unused imports
- Missing error checks
- Formatting issues

**Step 3: Run again until clean**

```bash
golangci-lint run --allow-parallel-runners
```

**Step 4: Commit fixes if any**

```bash
but commit <branch> -m "fix: address linter issues"
```

---

## Task 9: Manual Testing

**Step 1: Start the server**

```bash
PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug
```

**Step 2: Verify heartbeat table is populated**

```bash
psql -U bbdev bbdev -c "SELECT * FROM replica_heartbeat;"
```

**Step 3: Verify task runs get replica_id**

Create a task run and check:

```bash
psql -U bbdev bbdev -c "SELECT id, status, replica_id FROM task_run ORDER BY id DESC LIMIT 5;"
```

**Step 4: Test stale detection**

Manually insert a stale task run and verify it gets marked as FAILED:

```bash
# Insert a fake running task with unknown replica
psql -U bbdev bbdev -c "
  UPDATE task_run
  SET status = 'RUNNING', replica_id = 'fake-replica-id'
  WHERE id = (SELECT id FROM task_run LIMIT 1);
"

# Wait 30 seconds for stale detection to run
sleep 35

# Verify it was marked as FAILED
psql -U bbdev bbdev -c "
  SELECT id, status, result FROM task_run
  WHERE replica_id = 'fake-replica-id';
"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Database migration (replica_heartbeat table, task_run.replica_id) |
| 2 | Store methods for heartbeat upsert and cleanup |
| 3 | Store method for stale task detection |
| 4 | Modify ClaimAvailableTaskRuns to set replica_id |
| 5 | Create heartbeat runner |
| 6 | Add stale detection to data cleaner |
| 7 | Register heartbeat runner in server |
| 8 | Lint and fix issues |
| 9 | Manual testing |
