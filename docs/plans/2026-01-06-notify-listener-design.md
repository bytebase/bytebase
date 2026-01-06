# PostgreSQL NOTIFY/LISTEN for HA Cancel Propagation

## Problem

In HA setup, cancel requests for plan check runs and task runs may hit a different replica than the one executing the job. The `context.CancelFunc` only exists in memory on the executing replica, so the cancel request fails silently.

## Solution

Use PostgreSQL NOTIFY/LISTEN to broadcast cancel signals across all replicas.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Replica A                                │
│  ┌──────────────┐    ┌─────────────┐    ┌───────────────────┐  │
│  │ API Service  │───▶│ pg_notify() │    │ NotifyListener    │  │
│  │ (cancel req) │    └──────┬──────┘    │ Runner            │  │
│  │              │           │           │  - LISTEN channel │  │
│  │ local call ──┼───────────┼───────────┼─▶ lookup in bus   │  │
│  └──────────────┘           │           │  - call CancelFunc│  │
│                             │           └───────────────────┘  │
└─────────────────────────────┼──────────────────────────────────┘
                              │ PostgreSQL
                              │ NOTIFY 'bytebase_signal'
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Replica B                                │
│                             │           ┌───────────────────┐  │
│                             └──────────▶│ NotifyListener    │  │
│                                         │ Runner            │  │
│                                         │  - receives NOTIFY│  │
│                                         │  - lookup in bus  │  │
│                                         │  - call CancelFunc│  │
│                                         └───────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Design Details

### Channel & Payload

- **Channel:** `bytebase_signal`
- **Payload format:**
  ```json
  {"type": "cancel_plan_check_run", "uid": 123}
  {"type": "cancel_task_run", "uid": 456}
  ```

The `type` field encodes both action and resource. Extensible for future signal types.

### NotifyListener Runner

**Location:** `backend/runner/notifylistener/listener.go`

```go
type Listener struct {
    db  *sql.DB
    bus *bus.Bus
}

type Signal struct {
    Type string `json:"type"`
    UID  int    `json:"uid"`
}

func (l *Listener) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            if err := l.listen(ctx); err != nil {
                slog.Error("notify listener error, reconnecting", "error", err)
                time.Sleep(5 * time.Second)
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

    err = conn.Raw(func(driverConn any) error {
        pgxConn := driverConn.(*stdlib.Conn).Conn()

        _, err := pgxConn.Exec(ctx, "LISTEN bytebase_signal")
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
    return err
}

func (l *Listener) handleNotification(payload string) {
    var signal Signal
    if err := json.Unmarshal([]byte(payload), &signal); err != nil {
        slog.Warn("invalid signal payload", "payload", payload)
        return
    }

    switch signal.Type {
    case "cancel_plan_check_run":
        if cancel, ok := l.bus.RunningPlanCheckRunsCancelFunc.Load(signal.UID); ok {
            cancel.(context.CancelFunc)()
        }
    case "cancel_task_run":
        if cancel, ok := l.bus.RunningTaskRunsCancelFunc.Load(signal.UID); ok {
            cancel.(context.CancelFunc)()
        }
    }
}
```

### Signal Sender Helper

**Location:** `backend/store/signal.go`

```go
func (s *Store) SendSignal(ctx context.Context, signalType string, uid int) error {
    payload, _ := json.Marshal(map[string]any{
        "type": signalType,
        "uid":  uid,
    })
    _, err := s.db.ExecContext(ctx,
        "SELECT pg_notify('bytebase_signal', $1)",
        string(payload))
    return err
}
```

### API Changes

**Cancel flow (plan check runs and task runs):**

1. **Local fast path** - Call cancel func directly if found in local sync.Map
2. **Broadcast** - Send pg_notify to all replicas
3. **DB update** - Update status in database (source of truth)

```go
// Example: CancelPlanCheckRun in plan_service.go
func (s *PlanService) CancelPlanCheckRun(ctx context.Context, ...) {
    // 1. Local fast path
    if cancel, ok := s.bus.RunningPlanCheckRunsCancelFunc.Load(uid); ok {
        cancel.(context.CancelFunc)()
    }

    // 2. Broadcast to all replicas
    s.store.SendSignal(ctx, "cancel_plan_check_run", uid)

    // 3. Update DB status
    s.store.BatchCancelPlanCheckRuns(ctx, []int{uid})
}
```

### Error Handling

| Scenario | Behavior |
|----------|----------|
| UID not found in local sync.Map | No-op (expected - another replica has it) |
| Invalid JSON payload | Log warning, skip |
| Connection dropped | Reconnect with 5s backoff |
| Duplicate cancel (local + NOTIFY) | Safe - CancelFunc is idempotent |

## Files to Change

### New Files

| File | Purpose |
|------|---------|
| `backend/runner/notifylistener/listener.go` | NotifyListener runner |
| `backend/store/signal.go` | SendSignal helper |

### Modified Files

| File | Change |
|------|--------|
| `backend/api/v1/plan_service.go` | Add SendSignal call in CancelPlanCheckRun |
| `backend/api/v1/rollout_service.go` | Add SendSignal call in task run cancel |
| `backend/server/server.go` | Initialize and start NotifyListener runner |

## Notes

- No database schema changes required - uses PostgreSQL built-in NOTIFY/LISTEN
- Channel name `bytebase_signal` is generic for future extensibility
- Runner uses dedicated connection via `db.Conn()` since LISTEN requires persistent connection
