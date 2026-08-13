package loadtest

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
)

func snapshotMetrics(ctx context.Context, db *sql.DB) (Metrics, error) {
	m := Metrics{Timestamp: time.Now()}

	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity`).Scan(&m.TotalConnections); err != nil {
		return Metrics{}, errors.Wrapf(err, "count pg_stat_activity")
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE state = 'active'`).Scan(&m.ActiveConnections); err != nil {
		return Metrics{}, errors.Wrapf(err, "count active connections")
	}
	m.IdleConnections = m.TotalConnections - m.ActiveConnections

	if err := db.QueryRowContext(ctx, `SELECT setting::int FROM pg_settings WHERE name = 'max_connections'`).Scan(&m.MaxConnections); err != nil {
		return Metrics{}, errors.Wrapf(err, "read max_connections")
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_database WHERE datistemplate = false`).Scan(&m.DatabaseCount); err != nil {
		return Metrics{}, errors.Wrapf(err, "count pg_database")
	}

	var commit, rollback, blocksRead, blocksHit, tempFiles, tempBytes, deadlocks sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(sum(xact_commit), 0), COALESCE(sum(xact_rollback), 0), COALESCE(sum(blks_read), 0), COALESCE(sum(blks_hit), 0), COALESCE(sum(temp_files), 0), COALESCE(sum(temp_bytes), 0), COALESCE(sum(deadlocks), 0) FROM pg_stat_database`).Scan(&commit, &rollback, &blocksRead, &blocksHit, &tempFiles, &tempBytes, &deadlocks); err != nil {
		return Metrics{}, errors.Wrapf(err, "read pg_stat_database aggregates")
	}
	m.XactCommit = commit.Int64
	m.XactRollback = rollback.Int64
	m.BlocksRead = blocksRead.Int64
	m.BlocksHit = blocksHit.Int64
	m.TempFiles = tempFiles.Int64
	m.TempBytes = tempBytes.Int64
	m.Deadlocks = deadlocks.Int64

	return m, nil
}
