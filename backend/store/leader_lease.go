package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
)

const schemaSyncLeaderLeaseType = "SCHEMA_SYNC"

// TryAcquireLeaderLease acquires an expired or previously unclaimed leader lease.
func (s *Store) TryAcquireLeaderLease(ctx context.Context, leaderType string, replicaID string, ttl time.Duration) (int64, bool, error) {
	if err := validateLeaderLeaseType(leaderType); err != nil {
		return 0, false, err
	}

	const query = `
		INSERT INTO leader_lease (type, replica_id, generation, expires_at)
		VALUES ($1, $2, 1, clock_timestamp() + $3::interval)
		ON CONFLICT (type) DO UPDATE
		SET replica_id = EXCLUDED.replica_id,
			generation = leader_lease.generation + 1,
			expires_at = clock_timestamp() + $3::interval
		WHERE leader_lease.expires_at <= clock_timestamp()
		RETURNING generation
	`

	var generation int64
	err := s.GetDB().QueryRowContext(ctx, query, leaderType, replicaID, ttl.String()).Scan(&generation)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, errors.Wrap(err, "try acquire leader lease")
	}
	return generation, true, nil
}

// RenewLeaderLease extends the lease only for its current, unexpired generation.
func (s *Store) RenewLeaderLease(ctx context.Context, leaderType string, replicaID string, generation int64, ttl time.Duration) (bool, error) {
	if err := validateLeaderLeaseType(leaderType); err != nil {
		return false, err
	}

	const query = `
		UPDATE leader_lease
		SET expires_at = clock_timestamp() + $4::interval
		WHERE type = $1
			AND replica_id = $2
			AND generation = $3
			AND expires_at > clock_timestamp()
		RETURNING generation
	`

	var renewedGeneration int64
	err := s.GetDB().QueryRowContext(ctx, query, leaderType, replicaID, generation, ttl.String()).Scan(&renewedGeneration)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "renew leader lease")
	}
	return true, nil
}

// ReleaseLeaderLease expires the lease while retaining its current generation.
func (s *Store) ReleaseLeaderLease(ctx context.Context, leaderType string, replicaID string, generation int64) (bool, error) {
	if err := validateLeaderLeaseType(leaderType); err != nil {
		return false, err
	}

	const query = `
		UPDATE leader_lease
		SET expires_at = clock_timestamp()
		WHERE type = $1
			AND replica_id = $2
			AND generation = $3
		RETURNING generation
	`

	var releasedGeneration int64
	err := s.GetDB().QueryRowContext(ctx, query, leaderType, replicaID, generation).Scan(&releasedGeneration)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "release leader lease")
	}
	return true, nil
}

func validateLeaderLeaseType(leaderType string) error {
	if leaderType != schemaSyncLeaderLeaseType {
		return errors.Errorf("unsupported leader lease type %q", leaderType)
	}
	return nil
}
