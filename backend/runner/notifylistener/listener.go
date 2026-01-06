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

		_, err := pgxConn.Exec(ctx, "LISTEN "+store.SignalChannel)
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
