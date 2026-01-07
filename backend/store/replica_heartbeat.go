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
