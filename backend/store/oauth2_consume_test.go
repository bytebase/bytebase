package store_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestOAuth2AtomicConsume verifies the single-use issuance gate on
// authorization codes and refresh tokens. The DELETE ... RETURNING must make
// consumption atomic so concurrent redemptions of one grant can never both
// succeed (the double-mint race that the prior read-then-delete flow allowed),
// and a second sequential redemption must observe consumed=false.
func TestOAuth2AtomicConsume(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-test');
		INSERT INTO principal (name, email, password_hash) VALUES ('demo', 'demo@example.com', 'unused');
		INSERT INTO oauth2_client (client_id, workspace, client_secret_hash, config)
		VALUES ('client-A', NULL, 'unused-hash', '{}'::jsonb);
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	t.Run("authorization code is single-use", func(t *testing.T) {
		_, err := s.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
			Code:      "code-single-use",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "ws-test",
			Config:    &storepb.OAuth2AuthorizationCodeConfig{RedirectUri: "http://localhost/cb"},
			ExpiresAt: time.Now().Add(10 * time.Minute),
		})
		require.NoError(t, err)

		consumed, err := s.ConsumeOAuth2AuthorizationCode(ctx, "client-A", "code-single-use")
		require.NoError(t, err)
		require.True(t, consumed, "first consume must claim the code")

		consumed, err = s.ConsumeOAuth2AuthorizationCode(ctx, "client-A", "code-single-use")
		require.NoError(t, err)
		require.False(t, consumed, "second consume must observe the code already used, not error")
	})

	t.Run("consuming an unknown code returns false without error", func(t *testing.T) {
		consumed, err := s.ConsumeOAuth2AuthorizationCode(ctx, "client-A", "never-existed")
		require.NoError(t, err)
		require.False(t, consumed)
	})

	t.Run("concurrent redemption of one code claims it exactly once", func(t *testing.T) {
		_, err := s.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
			Code:      "code-racy",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "ws-test",
			Config:    &storepb.OAuth2AuthorizationCodeConfig{RedirectUri: "http://localhost/cb"},
			ExpiresAt: time.Now().Add(10 * time.Minute),
		})
		require.NoError(t, err)

		const racers = 16
		var wg sync.WaitGroup
		results := make([]bool, racers)
		errs := make([]error, racers)
		start := make(chan struct{})
		for i := range racers {
			wg.Go(func() {
				<-start
				results[i], errs[i] = s.ConsumeOAuth2AuthorizationCode(ctx, "client-A", "code-racy")
			})
		}
		close(start)
		wg.Wait()

		wins := 0
		for i := range racers {
			require.NoErrorf(t, errs[i], "racer %d must not error", i)
			if results[i] {
				wins++
			}
		}
		require.Equal(t, 1, wins, "exactly one concurrent redemption may claim the code")
	})

	t.Run("refresh token is single-use", func(t *testing.T) {
		_, err := s.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: "rt-single-use",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "ws-test",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		})
		require.NoError(t, err)

		consumed, err := s.ConsumeOAuth2RefreshToken(ctx, "client-A", "rt-single-use")
		require.NoError(t, err)
		require.True(t, consumed, "first consume must claim the refresh token")

		consumed, err = s.ConsumeOAuth2RefreshToken(ctx, "client-A", "rt-single-use")
		require.NoError(t, err)
		require.False(t, consumed, "second consume must observe the token already rotated, not error")
	})

	t.Run("concurrent refresh of one token claims it exactly once", func(t *testing.T) {
		_, err := s.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: "rt-racy",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "ws-test",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		})
		require.NoError(t, err)

		const racers = 16
		var wg sync.WaitGroup
		results := make([]bool, racers)
		errs := make([]error, racers)
		start := make(chan struct{})
		for i := range racers {
			wg.Go(func() {
				<-start
				results[i], errs[i] = s.ConsumeOAuth2RefreshToken(ctx, "client-A", "rt-racy")
			})
		}
		close(start)
		wg.Wait()

		wins := 0
		for i := range racers {
			require.NoErrorf(t, errs[i], "racer %d must not error", i)
			if results[i] {
				wins++
			}
		}
		require.Equal(t, 1, wins, "exactly one concurrent refresh may claim the token")
	})
}
