package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

// TestCollision_LoginAttempt is the composite-PK collision gate for
// login_attempt (PRIMARY KEY (identity, kind)) — see
// backend/store/AGENTS.md and docs/pre-pr-checklist.md step 3. The table is not project-scoped: identity
// is itself the scope column, so the shared project fixture does not apply.
// What collision safety means here is that no store operation may cross
// buckets: rows sharing an identity (other kind) or sharing a kind (other
// identity) must be untouched by claims, clears, and purges on a neighbor.
func TestCollision_LoginAttempt(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	const window = 10 * time.Minute
	const victim = "victim@example.com"
	const neighbor = "neighbor@example.com"

	// Fill the victim's PASSWORD bucket to its limit and park one attempt in
	// each neighboring bucket that shares one PK column with it.
	for range 3 {
		granted, err := s.ClaimAttempt(ctx, victim, storepb.LoginAttemptKind_PASSWORD, 3, window)
		require.NoError(t, err)
		require.True(t, granted)
	}
	for _, seed := range []struct {
		identity string
		kind     storepb.LoginAttemptKind
	}{
		{victim, storepb.LoginAttemptKind_MFA},        // same identity, other kind
		{neighbor, storepb.LoginAttemptKind_PASSWORD}, // same kind, other identity
	} {
		granted, err := s.ClaimAttempt(ctx, seed.identity, seed.kind, 3, window)
		require.NoError(t, err)
		require.True(t, granted)
	}

	attempts := func(identity string, kind storepb.LoginAttemptKind) int {
		t.Helper()
		var count int
		require.NoError(t, container.GetDB().QueryRowContext(ctx, `
			SELECT COALESCE((SELECT attempts FROM login_attempt WHERE identity = $1 AND kind = $2), 0)
		`, identity, kind.String()).Scan(&count))
		return count
	}

	// The victim's lock must not leak into either neighbor.
	granted, err := s.ClaimAttempt(ctx, victim, storepb.LoginAttemptKind_PASSWORD, 3, window)
	require.NoError(t, err)
	require.False(t, granted, "the victim bucket is at its limit")
	require.Equal(t, 1, attempts(victim, storepb.LoginAttemptKind_MFA))
	require.Equal(t, 1, attempts(neighbor, storepb.LoginAttemptKind_PASSWORD))

	// Clearing the victim must delete exactly the victim's row.
	require.NoError(t, s.ClearLoginAttempt(ctx, victim, storepb.LoginAttemptKind_PASSWORD))
	require.Zero(t, attempts(victim, storepb.LoginAttemptKind_PASSWORD))
	require.Equal(t, 1, attempts(victim, storepb.LoginAttemptKind_MFA))
	require.Equal(t, 1, attempts(neighbor, storepb.LoginAttemptKind_PASSWORD))

	// The time-scoped purge must not use the lock as a shortcut: backdate only
	// the neighbor and confirm the purge takes it and nothing else.
	_, err = container.GetDB().ExecContext(ctx, `
		UPDATE login_attempt SET last_attempt_at = last_attempt_at - make_interval(secs => $3)
		WHERE identity = $1 AND kind = $2
	`, neighbor, storepb.LoginAttemptKind_PASSWORD.String(), (2 * time.Hour).Seconds())
	require.NoError(t, err)
	deleted, err := s.DeleteStaleLoginAttempts(ctx, time.Hour)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	require.Zero(t, attempts(neighbor, storepb.LoginAttemptKind_PASSWORD))
	require.Equal(t, 1, attempts(victim, storepb.LoginAttemptKind_MFA))
}
