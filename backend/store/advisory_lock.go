package store

import (
	"context"
	"database/sql"
)

// AdvisoryLockKey defines lock identifiers for distributed coordination.
// Each scheduler/component needing cluster-wide mutex gets a unique key.
type AdvisoryLockKey int64

const (
	// AdvisoryLockKeyPendingScheduler is used by the pending task run scheduler
	// to ensure only one replica promotes PENDING → AVAILABLE at a time.
	AdvisoryLockKeyPendingScheduler AdvisoryLockKey = 1001
	// AdvisoryLockKeyMigration is used by the schema migrator to ensure only
	// one replica runs database migrations at a time.
	AdvisoryLockKeyMigration AdvisoryLockKey = 1002
)

// AdvisoryLock holds a dedicated connection for a session-level advisory lock.
type AdvisoryLock struct {
	conn *sql.Conn
	key  AdvisoryLockKey
}

// TryAdvisoryLock attempts to acquire a session-level advisory lock using a
// dedicated connection. Returns (lock, true) if acquired, (nil, false) if
// already held by another session. Caller must call lock.Release() when done.
func (s *Store) TryAdvisoryLock(ctx context.Context, key AdvisoryLockKey) (*AdvisoryLock, bool, error) {
	conn, err := s.dbConnManager.GetDB().Conn(ctx)
	if err != nil {
		return nil, false, err
	}

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", int64(key)).Scan(&acquired); err != nil {
		conn.Close()
		return nil, false, err
	}

	if !acquired {
		conn.Close()
		return nil, false, nil
	}

	return &AdvisoryLock{conn: conn, key: key}, true, nil
}

// Release releases the advisory lock and returns the connection to the pool.
func (l *AdvisoryLock) Release() error {
	if l.conn == nil {
		return nil
	}
	// Unlock then close; closing also releases but explicit unlock is cleaner
	_, _ = l.conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", int64(l.key))
	return l.conn.Close()
}

// AcquireAdvisoryLock acquires a session-level advisory lock, blocking until
// the lock is available. Returns a release function that must be called when done.
// This is useful for migrations where we want to wait rather than fail fast.
func AcquireAdvisoryLock(ctx context.Context, db *sql.DB, key AdvisoryLockKey) (func(), error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	// pg_advisory_lock blocks until the lock is acquired
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", int64(key)); err != nil {
		conn.Close()
		return nil, err
	}

	release := func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", int64(key))
		conn.Close()
	}
	return release, nil
}
