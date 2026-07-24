package store

import (
	"context"
	"database/sql"
	"strconv"
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
	// AdvisoryLockKeyVCSProviderUser is used as the namespace for active VCS
	// provider user limit checks and upserts.
	AdvisoryLockKeyVCSProviderUser AdvisoryLockKey = 1004
	// AdvisoryLockKeyPlanIssueRollout serializes Plan review changes, linked
	// Bytebase Issue creation, and Rollout creation for the same Plan.
	AdvisoryLockKeyPlanIssueRollout AdvisoryLockKey = 1005
)

// AcquirePlanIssueRolloutAdvisoryLock serializes coordinated Plan, linked Issue,
// Rollout, and Plan Check Run transactions for one Plan.
func AcquirePlanIssueRolloutAdvisoryLock(ctx context.Context, tx *sql.Tx, projectID string, planUID int64) error {
	key := projectID + "/" + strconv.FormatInt(planUID, 10)
	return AcquireAdvisoryXactLockWithStringKey(ctx, tx, AdvisoryLockKeyPlanIssueRollout, key)
}

// TryAdvisoryXactLock attempts to acquire a transaction-level advisory lock.
// Returns true if acquired, false if already held by another transaction.
// The lock is automatically released when the transaction ends.
func TryAdvisoryXactLock(ctx context.Context, tx *sql.Tx, key AdvisoryLockKey) (bool, error) {
	var acquired bool
	if err := tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", int64(key)).Scan(&acquired); err != nil {
		return false, err
	}
	return acquired, nil
}

// TryAdvisoryXactLockWithStringKey attempts to acquire a transaction-level
// advisory lock scoped by namespace and string key. Returns true if acquired,
// false if already held by another transaction. The lock is automatically
// released when the transaction ends.
func TryAdvisoryXactLockWithStringKey(ctx context.Context, tx *sql.Tx, namespace AdvisoryLockKey, key string) (bool, error) {
	var acquired bool
	if err := tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1, hashtext($2))", int32(namespace), key).Scan(&acquired); err != nil {
		return false, err
	}
	return acquired, nil
}

// AcquireAdvisoryXactLock acquires a transaction-level advisory lock, blocking
// until the lock is available. The lock is automatically released when the
// transaction ends.
func AcquireAdvisoryXactLock(ctx context.Context, tx *sql.Tx, key AdvisoryLockKey) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", int64(key))
	return err
}

// AcquireAdvisoryXactLockWithStringKey acquires a transaction-level advisory
// lock scoped by namespace and string key, blocking until the lock is available.
// The lock is automatically released when the transaction ends.
func AcquireAdvisoryXactLockWithStringKey(ctx context.Context, tx *sql.Tx, namespace AdvisoryLockKey, key string) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1, hashtext($2))", int32(namespace), key)
	return err
}
