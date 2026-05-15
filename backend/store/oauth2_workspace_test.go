package store_test

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

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestOAuth2WorkspaceBinding exercises the workspace round-trip on
// oauth2_authorization_code and oauth2_refresh_token via the real store.
// This is the persistence half of the consent → token workspace binding
// flow (the in-memory propagation through handleAuthorizePost /
// handleAuthorizationCodeGrant / issueTokens / GenerateOAuth2AccessToken
// is covered by the unit tests in backend/api/oauth2/token_test.go and
// the helper-package unit tests below).
//
// Covers:
//   - Non-empty workspace persists across write/read
//   - Legacy empty-workspace path: a code or refresh token with no workspace
//     (pre-3.18.2 row) survives the migration's nullable column and reads
//     back as "" so the handler's fallback chain to client.Workspace /
//     singleton can take over.
func TestOAuth2WorkspaceBinding(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	// Seed the minimal fixtures needed by the FK constraints on the OAuth2
	// tables: a workspace and a principal.
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

	t.Run("auth code round-trips workspace bound at consent time", func(t *testing.T) {
		_, err := s.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
			Code:      "code-with-ws",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "ws-test",
			Config:    &storepb.OAuth2AuthorizationCodeConfig{RedirectUri: "http://localhost/cb"},
			ExpiresAt: time.Now().Add(10 * time.Minute),
		})
		require.NoError(t, err)

		got, err := s.GetOAuth2AuthorizationCode(ctx, "client-A", "code-with-ws")
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "ws-test", got.Workspace,
			"workspace written at consent time must round-trip through the DB into the issued code")
	})

	t.Run("auth code with empty workspace stays empty for legacy fallback", func(t *testing.T) {
		// Simulates a pre-3.18.2 auth code created before the workspace
		// column existed. The migration added the column as nullable, so
		// existing rows have NULL and Get returns "".
		_, err := s.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
			Code:      "code-no-ws",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "", // explicitly empty
			Config:    &storepb.OAuth2AuthorizationCodeConfig{RedirectUri: "http://localhost/cb"},
			ExpiresAt: time.Now().Add(10 * time.Minute),
		})
		require.NoError(t, err)

		got, err := s.GetOAuth2AuthorizationCode(ctx, "client-A", "code-no-ws")
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got.Workspace,
			"empty workspace must persist as NULL and read back as \"\" so the handler's fallback chain (client.Workspace → singleton) can run")
	})

	t.Run("refresh token preserves workspace across refresh", func(t *testing.T) {
		_, err := s.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: "rt-hash-1",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "ws-test",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		})
		require.NoError(t, err)

		got, err := s.GetOAuth2RefreshToken(ctx, "client-A", "rt-hash-1")
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "ws-test", got.Workspace,
			"refresh token must carry the workspace binding so a /token refresh re-issues for the same workspace")
	})

	t.Run("refresh token with empty workspace stays empty", func(t *testing.T) {
		_, err := s.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: "rt-hash-2",
			ClientID:  "client-A",
			UserEmail: "demo@example.com",
			Workspace: "",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		})
		require.NoError(t, err)

		got, err := s.GetOAuth2RefreshToken(ctx, "client-A", "rt-hash-2")
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got.Workspace)
	})
}
