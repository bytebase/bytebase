package store_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testpg"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestLoginAttemptClaim verifies the per-identity lockout slots
// (docs/design/login-attempt-lockout.md): a claim atomically takes one of N
// attempt slots before the credential is checked, a locked identity claims
// nothing (so the lock expires exactly D after the Nth attempt), the counter
// forgets after D of quiet, and success deletes the row.
func TestLoginAttemptClaim(t *testing.T) {
	ctx := context.Background()
	db, s, _ := testpg.New(t)

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
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_EMAIL_CODE, 5, window)
			require.NoError(t, err)
			require.True(t, granted, "claim %d must be granted", i+1)
		}
		for range 2 {
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_EMAIL_CODE, 5, window)
			require.NoError(t, err)
			require.False(t, granted, "claims beyond N must be refused")
		}
	})

	t.Run("locked claims do not extend the lock", func(t *testing.T) {
		const identity = "locked@example.com"
		for range 3 {
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		lockedAt := lastAttemptAt(identity, storepb.LoginAttemptKind_PASSWORD)
		granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.False(t, granted)
		require.Equal(t, lockedAt, lastAttemptAt(identity, storepb.LoginAttemptKind_PASSWORD),
			"a refused claim must not touch last_attempt_at, so the lock lasts exactly D from the Nth attempt")
	})

	t.Run("counter forgets after a quiet window", func(t *testing.T) {
		const identity = "idle@example.com"
		for range 3 {
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.False(t, granted, "identity must be locked before the window passes")

		backdate(identity, storepb.LoginAttemptKind_PASSWORD, window+time.Second)
		for range 3 {
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted, "the counter must restart at one after D of quiet")
		}
		granted, err = s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.False(t, granted)
	})

	t.Run("clear deletes the row", func(t *testing.T) {
		const identity = "cleared@example.com"
		for range 3 {
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		require.NoError(t, s.ClearLoginAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD))
		for range 3 {
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
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
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_MFA, 3, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_MFA, 3, window)
		require.NoError(t, err)
		require.False(t, granted)

		granted, err = s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.True(t, granted, "another kind for the same identity must have its own slots")

		granted, err = s.ClaimAttempt(ctx, "other-bucketed@example.com", storepb.LoginAttemptKind_MFA, 3, window)
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
				granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, n, window)
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
		_, err := s.ClaimAttempt(ctx, "", storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.Error(t, err, "an empty identity must never write a row")
		_, err = s.ClaimAttempt(ctx, "someone@example.com", storepb.LoginAttemptKind_LOGIN_ATTEMPT_KIND_UNSPECIFIED, 3, window)
		require.Error(t, err, "an unspecified kind must never write a row")
		_, err = s.ClaimAttempt(ctx, strings.Repeat("a", 2049), storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.Error(t, err, "a structurally oversized identity must never write a row")
	})

	t.Run("purge deletes only stale rows", func(t *testing.T) {
		const staleIdentity = "stale@example.com"
		const freshIdentity = "fresh@example.com"
		for _, identity := range []string{staleIdentity, freshIdentity} {
			granted, err := s.ClaimAttempt(ctx, identity, storepb.LoginAttemptKind_PASSWORD, 3, window)
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

// TestSendBudgetClaim covers the outbound send budget: the same ClaimAttempt as
// the lockout, under the window rule EMAIL_CODE_SEND selects.
func TestSendBudgetClaim(t *testing.T) {
	ctx := context.Background()
	db, s, _ := testpg.New(t)

	const window = time.Hour
	const kind = storepb.LoginAttemptKind_EMAIL_CODE_SEND

	backdateWindow := func(key string, by time.Duration) {
		t.Helper()
		_, err := db.ExecContext(ctx, `
			UPDATE login_attempt SET last_attempt_at = last_attempt_at - make_interval(secs => $3)
			WHERE identity = $1 AND kind = $2
		`, key, kind.String(), by.Seconds())
		require.NoError(t, err)
	}

	t.Run("grants exactly max per window", func(t *testing.T) {
		const key = "sender-a"
		for i := range 3 {
			granted, err := s.ClaimAttempt(ctx, key, kind, 3, window)
			require.NoError(t, err)
			require.True(t, granted, "grant %d is within the window", i)
		}
		granted, err := s.ClaimAttempt(ctx, key, kind, 3, window)
		require.NoError(t, err)
		require.False(t, granted, "the window is full")
	})

	// Why EMAIL_CODE_SEND anchors its window to the first claim. Under a
	// credential kind's rule every grant restarts the window, so a steady trickle
	// never resets it and "3 per hour" would become "3 ever, until an hour of
	// silence", refusing traffic that never approached the rate.
	t.Run("a steady trickle does not accumulate across windows", func(t *testing.T) {
		const key = "sender-trickle"
		// Six sends, each arriving well inside the window but with the window
		// itself rolling over between every pair. A lockout counter would reach
		// its limit on the third; a fixed window starts over each time.
		for i := range 6 {
			granted, err := s.ClaimAttempt(ctx, key, kind, 3, window)
			require.NoError(t, err, "send %d", i)
			require.True(t, granted, "send %d must not be refused: no window ever held more than two", i)
			if i%2 == 1 {
				backdateWindow(key, window+time.Minute)
			}
		}
	})

	t.Run("a full window reopens once it expires", func(t *testing.T) {
		const key = "sender-expiry"
		for range 2 {
			granted, err := s.ClaimAttempt(ctx, key, kind, 2, window)
			require.NoError(t, err)
			require.True(t, granted)
		}
		granted, err := s.ClaimAttempt(ctx, key, kind, 2, window)
		require.NoError(t, err)
		require.False(t, granted)

		backdateWindow(key, window+time.Minute)
		granted, err = s.ClaimAttempt(ctx, key, kind, 2, window)
		require.NoError(t, err)
		require.True(t, granted, "a new window starts once the old one expires")
	})

	t.Run("refusals do not extend the window", func(t *testing.T) {
		const key = "sender-refusal"
		granted, err := s.ClaimAttempt(ctx, key, kind, 1, window)
		require.NoError(t, err)
		require.True(t, granted)
		var opened time.Time
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT last_attempt_at FROM login_attempt WHERE identity = $1 AND kind = $2
		`, key, kind.String()).Scan(&opened))

		for range 5 {
			granted, err := s.ClaimAttempt(ctx, key, kind, 1, window)
			require.NoError(t, err)
			require.False(t, granted)
		}
		var after time.Time
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT last_attempt_at FROM login_attempt WHERE identity = $1 AND kind = $2
		`, key, kind.String()).Scan(&after))
		require.True(t, opened.Equal(after), "hammering a full window must not push its expiry out")
	})

	t.Run("senders and kinds are independent", func(t *testing.T) {
		granted, err := s.ClaimAttempt(ctx, "sender-x", kind, 1, window)
		require.NoError(t, err)
		require.True(t, granted)
		granted, err = s.ClaimAttempt(ctx, "sender-y", kind, 1, window)
		require.NoError(t, err)
		require.True(t, granted, "one sender's full window must not bind another")
		granted, err = s.ClaimAttempt(ctx, "sender-x", storepb.LoginAttemptKind_PASSWORD, 1, window)
		require.NoError(t, err)
		require.True(t, granted, "a send budget must not consume a credential bucket")
	})

	t.Run("an unkeyed claim is refused outright", func(t *testing.T) {
		_, err := s.ClaimAttempt(ctx, "", kind, 1, window)
		require.Error(t, err)
	})
}
