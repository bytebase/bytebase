package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestLoginAttemptClaim verifies the per-identity lockout slots
// (docs/design/login-attempt-lockout.md): a claim atomically takes one of N
// attempt slots before the credential is checked, a locked identity claims
// nothing (so the lock expires exactly D after the Nth attempt), the counter
// forgets after D of quiet, and success deletes the row.
func TestLoginAttemptClaim(t *testing.T) {
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

	const window = 10 * time.Minute

	backdate := func(identity string, kind storepb.LoginAttemptKind, by time.Duration) {
		t.Helper()
		result, err := db.ExecContext(ctx, `
			UPDATE login_attempt SET last_attempt_at = last_attempt_at - make_interval(secs => $3)
			WHERE identity = $1 AND kind = $2
		`, identity, kind.String(), by.Seconds())
		require.NoError(t, err)
		rows, err := result.RowsAffected()
		require.NoError(t, err)
		require.EqualValues(t, 1, rows)
	}
	lastAttemptAt := func(identity string, kind storepb.LoginAttemptKind) time.Time {
		t.Helper()
		var at time.Time
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT last_attempt_at FROM login_attempt WHERE identity = $1 AND kind = $2
		`, identity, kind.String()).Scan(&at))
		return at
	}

	t.Run("grants exactly N slots then locks", func(t *testing.T) {
		const identity = "victim@example.com"
		for i := range 5 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_EMAIL_CODE, 5, window)
			require.NoError(t, err)
			require.True(t, granted, "claim %d must be granted", i+1)
		}
		for range 2 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_EMAIL_CODE, 5, window)
			require.NoError(t, err)
			require.False(t, granted, "claims beyond N must be refused")
		}
	})

	t.Run("locked claims do not extend the lock", func(t *testing.T) {
		const identity = "locked@example.com"
		for range 3 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		lockedAt := lastAttemptAt(identity, storepb.LoginAttemptKind_PASSWORD)
		granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.False(t, granted)
		require.Equal(t, lockedAt, lastAttemptAt(identity, storepb.LoginAttemptKind_PASSWORD),
			"a refused claim must not touch last_attempt_at, so the lock lasts exactly D from the Nth attempt")
	})

	t.Run("counter forgets after a quiet window", func(t *testing.T) {
		const identity = "idle@example.com"
		for range 3 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.False(t, granted, "identity must be locked before the window passes")

		backdate(identity, storepb.LoginAttemptKind_PASSWORD, window+time.Second)
		for range 3 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted, "the counter must restart at one after D of quiet")
		}
		granted, err = s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.False(t, granted)
	})

	t.Run("clear deletes the row", func(t *testing.T) {
		const identity = "cleared@example.com"
		for range 3 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		require.NoError(t, s.ClearLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD))
		for range 3 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted, "success must forget every prior failure")
		}
	})

	t.Run("clear of an unknown identity is a no-op", func(t *testing.T) {
		require.NoError(t, s.ClearLoginAttempt(ctx, "never-seen@example.com", storepb.LoginAttemptKind_MFA))
	})

	t.Run("kinds and identities are independent buckets", func(t *testing.T) {
		const identity = "bucketed@example.com"
		for range 3 {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_MFA, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_MFA, 3, window)
		require.NoError(t, err)
		require.False(t, granted)

		granted, err = s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.True(t, granted, "another kind for the same identity must have its own slots")

		granted, err = s.ClaimLoginAttempt(ctx, "other-bucketed@example.com", storepb.LoginAttemptKind_MFA, 3, window)
		require.NoError(t, err)
		require.True(t, granted, "another identity must have its own slots")
	})

	t.Run("concurrent claims grant exactly N slots", func(t *testing.T) {
		const identity = "raced@example.com"
		const n = 5
		var wg sync.WaitGroup
		grants := make(chan bool, 20)
		for range 20 {
			wg.Go(func() {
				granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, n, window)
				require.NoError(t, err)
				grants <- granted
			})
		}
		wg.Wait()
		close(grants)
		grantedCount := 0
		for granted := range grants {
			if granted {
				grantedCount++
			}
		}
		require.Equal(t, n, grantedCount, "the row lock must serialize claims so exactly N slots are granted")
	})

	t.Run("unkeyed claims are refused outright", func(t *testing.T) {
		_, err := s.ClaimLoginAttempt(ctx, "", storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.Error(t, err, "an empty identity must never write a row")
		_, err = s.ClaimLoginAttempt(ctx, "someone@example.com", storepb.LoginAttemptKind_LOGIN_ATTEMPT_KIND_UNSPECIFIED, 3, window)
		require.Error(t, err, "an unspecified kind must never write a row")
		_, err = s.ClaimLoginAttempt(ctx, strings.Repeat("a", 2049), storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.Error(t, err, "a structurally oversized identity must never write a row")
	})

	t.Run("purge deletes only stale rows", func(t *testing.T) {
		const staleIdentity = "stale@example.com"
		const freshIdentity = "fresh@example.com"
		for _, identity := range []string{staleIdentity, freshIdentity} {
			granted, err := s.ClaimLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		backdate(staleIdentity, storepb.LoginAttemptKind_PASSWORD, time.Hour+time.Second)

		deleted, err := s.DeleteStaleLoginAttempts(ctx, time.Hour)
		require.NoError(t, err)
		require.EqualValues(t, 1, deleted)

		var count int
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM login_attempt WHERE identity = $1
		`, freshIdentity).Scan(&count))
		require.Equal(t, 1, count, "rows attempted within the retention window must survive the purge")
	})
}

// TestLoginAttemptBucketsClaim covers the all-or-nothing multi-bucket claim used
// by the outbound sign-in-code budget. A limit composed of several buckets must
// not be partially spendable: if claiming them one at a time, every refusal by a
// later bucket would still consume the earlier ones, so a caller who can no
// longer send anything could keep draining the buckets ahead of the refusal —
// locking them for their whole window at no cost.
func TestLoginAttemptBucketsClaim(t *testing.T) {
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

	const window = time.Hour
	const kind = storepb.LoginAttemptKind_EMAIL_CODE_SEND

	attemptsOf := func(identity string) int {
		t.Helper()
		var attempts int
		err := db.QueryRowContext(ctx, `
			SELECT attempts FROM login_attempt WHERE identity = $1 AND kind = $2
		`, identity, kind.String()).Scan(&attempts)
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		require.NoError(t, err)
		return attempts
	}

	t.Run("a refused bucket charges none of the others", func(t *testing.T) {
		// The refusing bucket must sort AFTER the one that has headroom: buckets
		// are claimed in identity order, so this is the ordering under which a
		// non-atomic claim would already have charged `wide` by the time
		// `narrow` refuses. Named the other way round the test passes against
		// the very implementation it exists to reject.
		const wide = "aaa-wide-bucket"
		const narrow = "zzz-narrow-bucket"
		buckets := []store.LoginAttemptBucket{{Identity: wide, Max: 10}, {Identity: narrow, Max: 2}}

		for i := range 2 {
			granted, err := s.ClaimLoginAttemptBuckets(ctx, kind, window, buckets)
			require.NoError(t, err)
			require.True(t, granted, "claim %d is within every bucket", i)
		}
		require.Equal(t, 2, attemptsOf(wide))
		require.Equal(t, 2, attemptsOf(narrow))

		// The narrow bucket is now full. Every further claim must be refused
		// without moving the wide bucket, however many times it is retried.
		for range 5 {
			granted, err := s.ClaimLoginAttemptBuckets(ctx, kind, window, buckets)
			require.NoError(t, err)
			require.False(t, granted)
		}
		require.Equal(t, 2, attemptsOf(wide), "a refused claim must not spend the buckets ahead of the refusal")
		require.Equal(t, 2, attemptsOf(narrow), "a refused claim leaves the refusing bucket untouched too")
	})

	t.Run("buckets are claimed in identity order from either direction", func(t *testing.T) {
		// Same two buckets, opposite request order, claimed concurrently. The
		// store sorts by identity — full primary-key order, since kind is fixed —
		// so both transactions take the rows in the same order and neither can
		// wait on the other's first lock. Asserts terminal state, not just the
		// absence of a deadlock error: every claim must be granted and counted.
		const first = "aaa-bucket"
		const second = "zzz-bucket"
		ascending := []store.LoginAttemptBucket{{Identity: first, Max: 100}, {Identity: second, Max: 100}}
		descending := []store.LoginAttemptBucket{{Identity: second, Max: 100}, {Identity: first, Max: 100}}

		const perDirection = 20
		var wg sync.WaitGroup
		errs := make(chan error, 2*perDirection)
		for _, order := range [][]store.LoginAttemptBucket{ascending, descending} {
			for range perDirection {
				wg.Go(func() {
					granted, err := s.ClaimLoginAttemptBuckets(ctx, kind, window, order)
					if err != nil {
						errs <- err
						return
					}
					if !granted {
						errs <- errors.Errorf("claim refused with headroom left")
					}
				})
			}
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err, "opposing claim orders must not deadlock or refuse")
		}
		require.Equal(t, 2*perDirection, attemptsOf(first))
		require.Equal(t, 2*perDirection, attemptsOf(second))
	})

	t.Run("an unkeyed bucket is refused outright", func(t *testing.T) {
		_, err := s.ClaimLoginAttemptBuckets(ctx, kind, window, []store.LoginAttemptBucket{{Identity: "ok", Max: 1}, {Identity: "", Max: 1}})
		require.Error(t, err, "an empty identity must never write a row")
		require.Equal(t, 0, attemptsOf("ok"), "validation must run before any bucket is charged")
	})

	t.Run("no buckets is a grant", func(t *testing.T) {
		granted, err := s.ClaimLoginAttemptBuckets(ctx, kind, window, nil)
		require.NoError(t, err)
		require.True(t, granted, "a caller with no budget configured is not rate limited")
	})
}
