# NotifyListener Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement PostgreSQL NOTIFY/LISTEN for HA-compatible cancel propagation of plan check runs and task runs.

**Architecture:** New runner listens on `bytebase_signal` channel. API services send pg_notify on cancel. Listener dispatches to in-memory cancel funcs via bus.

**Tech Stack:** Go, PostgreSQL NOTIFY/LISTEN, pgx v5

---

## Task 1: Create Signal Store Helper

**Files:**
- Create: `backend/store/signal.go`

**Step 1: Create the signal helper file**

```go
package store

import (
	"context"
	"encoding/json"
)

// Signal represents a notification payload sent via PostgreSQL NOTIFY.
type Signal struct {
	Type string `json:"type"`
	UID  int    `json:"uid"`
}

// Signal types.
const (
	SignalTypeCancelPlanCheckRun = "cancel_plan_check_run"
	SignalTypeCancelTaskRun      = "cancel_task_run"
)

// SendSignal sends a notification to the bytebase_signal channel.
func (s *Store) SendSignal(ctx context.Context, signalType string, uid int) error {
	payload, err := json.Marshal(&Signal{
		Type: signalType,
		UID:  uid,
	})
	if err != nil {
		return err
	}
	_, err = s.GetDB().ExecContext(ctx, "SELECT pg_notify('bytebase_signal', $1)", string(payload))
	return err
}
```

**Step 2: Verify it compiles**

Run: `go build ./backend/store/...`
Expected: Success (no output)

**Step 3: Commit**

```bash
but commit notify-listener -m "feat(store): add signal helper for pg_notify"
```

---

## Task 2: Create NotifyListener Runner

**Files:**
- Create: `backend/runner/notifylistener/listener.go`

**Step 1: Create the listener runner**

```go
// Package notifylistener listens for PostgreSQL NOTIFY signals for HA coordination.
package notifylistener

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	signalChannel    = "bytebase_signal"
	reconnectBackoff = 5 * time.Second
)

// Listener listens for PostgreSQL NOTIFY signals.
type Listener struct {
	db  *sql.DB
	bus *bus.Bus
}

// NewListener creates a new notify listener.
func NewListener(db *sql.DB, bus *bus.Bus) *Listener {
	return &Listener{
		db:  db,
		bus: bus,
	}
}

// Run starts the listener loop.
func (l *Listener) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	slog.Debug("Notify listener started")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := l.listen(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("notify listener error, reconnecting", log.BBError(err))
				time.Sleep(reconnectBackoff)
			}
		}
	}
}

func (l *Listener) listen(ctx context.Context) error {
	conn, err := l.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		pgxConn := driverConn.(*stdlib.Conn).Conn()

		_, err := pgxConn.Exec(ctx, "LISTEN "+signalChannel)
		if err != nil {
			return err
		}

		for {
			notification, err := pgxConn.WaitForNotification(ctx)
			if err != nil {
				return err
			}
			l.handleNotification(notification.Payload)
		}
	})
}

func (l *Listener) handleNotification(payload string) {
	var signal store.Signal
	if err := json.Unmarshal([]byte(payload), &signal); err != nil {
		slog.Warn("invalid signal payload", "payload", payload, log.BBError(err))
		return
	}

	switch signal.Type {
	case store.SignalTypeCancelPlanCheckRun:
		if cancel, ok := l.bus.RunningPlanCheckRunsCancelFunc.Load(signal.UID); ok {
			cancel.(context.CancelFunc)()
		}
	case store.SignalTypeCancelTaskRun:
		if cancel, ok := l.bus.RunningTaskRunsCancelFunc.Load(signal.UID); ok {
			cancel.(context.CancelFunc)()
		}
	default:
		slog.Warn("unknown signal type", "type", signal.Type)
	}
}
```

**Step 2: Verify it compiles**

Run: `go build ./backend/runner/notifylistener/...`
Expected: Success (no output)

**Step 3: Commit**

```bash
but commit notify-listener -m "feat(runner): add notifylistener for HA signal propagation"
```

---

## Task 3: Wire Listener into Server

**Files:**
- Modify: `backend/server/server.go:56-86` (add field to Server struct)
- Modify: `backend/server/server.go:196` (create listener in NewServer)
- Modify: `backend/server/server.go:236` (start listener in Run)

**Step 1: Add import and field to Server struct**

In `backend/server/server.go`, add import:

```go
"github.com/bytebase/bytebase/backend/runner/notifylistener"
```

Add field to Server struct (after line 62, after `approvalRunner`):

```go
notifyListener *notifylistener.Listener
```

**Step 2: Create listener in NewServer**

After line 196 (after `s.planCheckScheduler = plancheck.NewScheduler(...)`), add:

```go
s.notifyListener = notifylistener.NewListener(stores.GetDB(), s.bus)
```

**Step 3: Start listener in Run**

After line 236 (after `go s.dataCleaner.Run(ctx, &s.runnerWG)`), add:

```go
s.runnerWG.Add(1)
go s.notifyListener.Run(ctx, &s.runnerWG)
```

**Step 4: Verify it compiles**

Run: `go build ./backend/server/...`
Expected: Success (no output)

**Step 5: Commit**

```bash
but commit notify-listener -m "feat(server): wire notifylistener runner"
```

---

## Task 4: Add Signal to Plan Check Run Cancellation

**Files:**
- Modify: `backend/api/v1/plan_service.go:497-500`

**Step 1: Add SendSignal after local cancel**

In `CancelPlanCheckRun` function, after the existing cancel block (line 498-500) and before the `BatchCancelPlanCheckRuns` call (line 502), add:

```go
// Broadcast cancel signal to all replicas for HA.
if err := s.store.SendSignal(ctx, store.SignalTypeCancelPlanCheckRun, planCheckRun.UID); err != nil {
	slog.Warn("failed to send cancel signal", log.BBError(err))
}
```

Also add import for `log/slog` and `"github.com/bytebase/bytebase/backend/common/log"` if not present.

**Step 2: Verify it compiles**

Run: `go build ./backend/api/v1/...`
Expected: Success (no output)

**Step 3: Commit**

```bash
but commit notify-listener -m "feat(api): broadcast cancel signal for plan check runs"
```

---

## Task 5: Add Signal to Task Run Cancellation

**Files:**
- Modify: `backend/api/v1/rollout_service.go:930-936`

**Step 1: Add SendSignal in the cancel loop**

In `BatchCancelTaskRuns` function, modify the cancel loop (lines 930-936) to also send a signal:

Replace:
```go
for _, taskRun := range taskRuns {
	if taskRun.Status == storepb.TaskRun_RUNNING {
		if cancelFunc, ok := s.bus.RunningTaskRunsCancelFunc.Load(taskRun.ID); ok {
			cancelFunc.(context.CancelFunc)()
		}
	}
}
```

With:
```go
for _, taskRun := range taskRuns {
	if taskRun.Status == storepb.TaskRun_RUNNING {
		if cancelFunc, ok := s.bus.RunningTaskRunsCancelFunc.Load(taskRun.ID); ok {
			cancelFunc.(context.CancelFunc)()
		}
		// Broadcast cancel signal to all replicas for HA.
		if err := s.store.SendSignal(ctx, store.SignalTypeCancelTaskRun, taskRun.ID); err != nil {
			slog.Warn("failed to send cancel signal", log.BBError(err))
		}
	}
}
```

Also add import for `log/slog` if not present.

**Step 2: Verify it compiles**

Run: `go build ./backend/api/v1/...`
Expected: Success (no output)

**Step 3: Commit**

```bash
but commit notify-listener -m "feat(api): broadcast cancel signal for task runs"
```

---

## Task 6: Run Linter and Fix Issues

**Files:**
- Potentially all modified files

**Step 1: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/...`
Expected: No errors (or fix any reported issues)

**Step 2: Run golangci-lint again (may have max-issues limit)**

Run: `golangci-lint run --allow-parallel-runners ./backend/...`
Expected: No errors

**Step 3: Commit if fixes were needed**

```bash
but commit notify-listener -m "fix: address linter issues"
```

---

## Task 7: Build and Verify

**Step 1: Full build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Success (binary created)

**Step 2: Commit final state**

```bash
but commit notify-listener -m "chore: verify full build passes"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Signal store helper | `backend/store/signal.go` |
| 2 | NotifyListener runner | `backend/runner/notifylistener/listener.go` |
| 3 | Wire into server | `backend/server/server.go` |
| 4 | Plan check cancel signal | `backend/api/v1/plan_service.go` |
| 5 | Task run cancel signal | `backend/api/v1/rollout_service.go` |
| 6 | Lint fixes | Various |
| 7 | Full build verification | - |
