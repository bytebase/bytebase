package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

const schemaSyncLeaderType = "SCHEMA_SYNC"

func TestLeaderLease(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	newStore := func(t *testing.T) *store.Store {
		t.Helper()
		leaseStore, err := store.New(ctx, pgURL, false)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, leaseStore.Close()) })
		return leaseStore
	}

	reset := func(t *testing.T) {
		t.Helper()
		_, err := db.ExecContext(ctx, "DELETE FROM leader_lease")
		require.NoError(t, err)
	}

	t.Run("acquire, renew, and contend", func(t *testing.T) {
		reset(t)
		generation, acquired, err := s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		require.NoError(t, err)
		require.True(t, acquired)
		require.EqualValues(t, 1, generation)

		generation, acquired, err = s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		require.NoError(t, err)
		require.False(t, acquired, "an unexpired lease contends even for its current holder")
		require.Zero(t, generation)

		generation, acquired, err = s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-b", time.Second)
		require.NoError(t, err)
		require.False(t, acquired)
		require.Zero(t, generation)

		renewed, err := s.RenewLeaderLease(ctx, schemaSyncLeaderType, "replica-a", 1, time.Second)
		require.NoError(t, err)
		require.True(t, renewed)

		var holder string
		var storedGeneration int64
		require.NoError(t, db.QueryRowContext(ctx, "SELECT replica_id, generation FROM leader_lease WHERE type = $1", schemaSyncLeaderType).Scan(&holder, &storedGeneration))
		require.Equal(t, "replica-a", holder)
		require.EqualValues(t, 1, storedGeneration, "renewing must preserve the generation")
	})

	t.Run("release retains the row and permits an immediate takeover", func(t *testing.T) {
		reset(t)
		generation, acquired, err := s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		require.NoError(t, err)
		require.True(t, acquired)
		require.EqualValues(t, 1, generation)

		released, err := s.ReleaseLeaderLease(ctx, schemaSyncLeaderType, "replica-a", generation)
		require.NoError(t, err)
		require.True(t, released)

		generation, acquired, err = s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-b", time.Second)
		require.NoError(t, err)
		require.True(t, acquired)
		require.EqualValues(t, 2, generation)

		var holder string
		var storedGeneration int64
		require.NoError(t, db.QueryRowContext(ctx, "SELECT replica_id, generation FROM leader_lease WHERE type = $1", schemaSyncLeaderType).Scan(&holder, &storedGeneration))
		require.Equal(t, "replica-b", holder)
		require.EqualValues(t, 2, storedGeneration)
	})

	t.Run("stale and non-holder generations cannot renew or release", func(t *testing.T) {
		reset(t)
		generation, acquired, err := s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		require.NoError(t, err)
		require.True(t, acquired)

		renewed, err := s.RenewLeaderLease(ctx, schemaSyncLeaderType, "replica-b", generation, time.Second)
		require.NoError(t, err)
		require.False(t, renewed)
		released, err := s.ReleaseLeaderLease(ctx, schemaSyncLeaderType, "replica-b", generation)
		require.NoError(t, err)
		require.False(t, released)

		_, err = db.ExecContext(ctx, "UPDATE leader_lease SET expires_at = clock_timestamp() - interval '1 second' WHERE type = $1", schemaSyncLeaderType)
		require.NoError(t, err)
		renewed, err = s.RenewLeaderLease(ctx, schemaSyncLeaderType, "replica-a", 1, time.Second)
		require.NoError(t, err)
		require.False(t, renewed, "an expired lease cannot be renewed")
		released, err = s.ReleaseLeaderLease(ctx, schemaSyncLeaderType, "replica-a", 1)
		require.NoError(t, err)
		require.True(t, released, "release must match the current generation even after expiry")
		generation, acquired, err = s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		require.NoError(t, err)
		require.True(t, acquired)
		require.EqualValues(t, 2, generation, "the same replica takes an expired lease with a new generation")

		renewed, err = s.RenewLeaderLease(ctx, schemaSyncLeaderType, "replica-a", 1, time.Second)
		require.NoError(t, err)
		require.False(t, renewed)
		released, err = s.ReleaseLeaderLease(ctx, schemaSyncLeaderType, "replica-a", 1)
		require.NoError(t, err)
		require.False(t, released)
	})

	t.Run("subsecond ttl is not truncated", func(t *testing.T) {
		reset(t)
		generation, acquired, err := s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", 900*time.Millisecond)
		require.NoError(t, err)
		require.True(t, acquired)
		require.EqualValues(t, 1, generation)

		var remainingSeconds float64
		require.NoError(t, db.QueryRowContext(ctx, "SELECT EXTRACT(EPOCH FROM expires_at - clock_timestamp()) FROM leader_lease WHERE type = $1", schemaSyncLeaderType).Scan(&remainingSeconds))
		require.Positive(t, remainingSeconds, "the TTL must retain subsecond precision")
	})

	t.Run("lease lifecycle spans independent database pools", func(t *testing.T) {
		reset(t)
		acquirer := newStore(t)
		renewer := newStore(t)
		releaser := newStore(t)
		takeover := newStore(t)

		generation, acquired, err := acquirer.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		require.NoError(t, err)
		require.True(t, acquired)
		require.EqualValues(t, 1, generation)

		renewed, err := renewer.RenewLeaderLease(ctx, schemaSyncLeaderType, "replica-a", generation, time.Second)
		require.NoError(t, err)
		require.True(t, renewed)

		released, err := releaser.ReleaseLeaderLease(ctx, schemaSyncLeaderType, "replica-a", generation)
		require.NoError(t, err)
		require.True(t, released)

		generation, acquired, err = takeover.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-b", time.Second)
		require.NoError(t, err)
		require.True(t, acquired)
		require.EqualValues(t, 2, generation)
	})

	t.Run("invalid leader type fails before database access", func(t *testing.T) {
		invalidStore := &store.Store{}
		_, _, err := invalidStore.TryAcquireLeaderLease(ctx, "", "replica-a", time.Second)
		require.Error(t, err)
		_, _, err = invalidStore.TryAcquireLeaderLease(ctx, "OTHER", "replica-a", time.Second)
		require.Error(t, err)
		_, err = invalidStore.RenewLeaderLease(ctx, "OTHER", "replica-a", 1, time.Second)
		require.Error(t, err)
		_, err = invalidStore.ReleaseLeaderLease(ctx, "OTHER", "replica-a", 1)
		require.Error(t, err)
	})

	t.Run("two first acquirers use distinct pooled connections and elect one leader", func(t *testing.T) {
		reset(t)
		s.GetDB().SetMaxOpenConns(2)
		defer s.GetDB().SetMaxOpenConns(0)
		assertRaceOneWinner(ctx, t, db, func() (int64, bool, error) {
			return s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		}, func() (int64, bool, error) {
			return s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-b", time.Second)
		})
	})

	t.Run("two expired takeovers elect one next generation", func(t *testing.T) {
		reset(t)
		_, err := db.ExecContext(ctx, `
			INSERT INTO leader_lease (type, replica_id, generation, expires_at)
			VALUES ($1, 'expired-replica', 41, clock_timestamp() - interval '1 second')
		`, schemaSyncLeaderType)
		require.NoError(t, err)
		s.GetDB().SetMaxOpenConns(2)
		defer s.GetDB().SetMaxOpenConns(0)
		assertRaceOneWinner(ctx, t, db, func() (int64, bool, error) {
			return s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-a", time.Second)
		}, func() (int64, bool, error) {
			return s.TryAcquireLeaderLease(ctx, schemaSyncLeaderType, "replica-b", time.Second)
		})

		var generation int64
		require.NoError(t, db.QueryRowContext(ctx, "SELECT generation FROM leader_lease WHERE type = $1", schemaSyncLeaderType).Scan(&generation))
		require.EqualValues(t, 42, generation)
	})
}

func assertRaceOneWinner(ctx context.Context, t *testing.T, db *sql.DB, first, second func() (int64, bool, error)) {
	t.Helper()
	lockTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer lockTx.Rollback()
	_, err = lockTx.ExecContext(ctx, "LOCK TABLE leader_lease IN ACCESS EXCLUSIVE MODE")
	require.NoError(t, err)

	type result struct {
		generation int64
		acquired   bool
		err        error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, acquire := range []func() (int64, bool, error){first, second} {
		wg.Go(func() {
			<-start
			generation, acquired, err := acquire()
			results <- result{generation: generation, acquired: acquired, err: err}
		})
	}
	close(start)

	require.Eventually(t, func() bool {
		var waiting int
		err := db.QueryRowContext(ctx, `
			SELECT count(*)
			FROM pg_stat_activity
			WHERE datname = current_database()
			  AND wait_event_type = 'Lock'
			  AND query LIKE '%leader_lease%'
		`).Scan(&waiting)
		return err == nil && waiting >= 2
	}, 2*time.Second, 10*time.Millisecond, "both acquirers must be observed blocked on PostgreSQL")

	require.NoError(t, lockTx.Commit())
	wg.Wait()
	close(results)

	wins := 0
	for result := range results {
		require.NoError(t, result.err)
		if result.acquired {
			wins++
		}
	}
	require.Equal(t, 1, wins)
}
