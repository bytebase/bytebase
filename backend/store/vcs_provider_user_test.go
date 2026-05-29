package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/store"
)

func TestVCSProviderUserTouchAndCount(t *testing.T) {
	ctx := context.Background()
	s, db := setupVCSProviderUserStore(ctx, t)

	const workspace = "default"
	user := &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITHUB,
		UserID:  "1001",
		Payload: &storepb.VCSProviderUserPayload{
			UserName:    "alice",
			DisplayName: "Alice",
		},
	}

	ok, err := s.TouchVCSProviderUser(ctx, workspace, user, store.VCSProviderUserActiveWindow, 1)
	require.NoError(t, err)
	require.True(t, ok)

	count, err := s.CountActiveVCSProviderUsers(ctx, workspace, store.VCSProviderUserActiveWindow)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var agedLastSeenAt time.Time
	require.NoError(t, db.QueryRowContext(ctx, `
		UPDATE vcs_provider_user
		SET last_seen_at = now() - interval '1 hour'
		WHERE workspace = $1 AND vcs_type = $2 AND user_id = $3
		RETURNING last_seen_at
	`, workspace, v1pb.VCSType_GITHUB.String(), "1001").Scan(&agedLastSeenAt))

	ok, err = s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITHUB,
		UserID:  "1001",
		Payload: &storepb.VCSProviderUserPayload{
			UserName:    "alice2",
			DisplayName: "Alice Cooper",
		},
	}, store.VCSProviderUserActiveWindow, 1)
	require.NoError(t, err)
	require.True(t, ok)

	users, err := s.ListActiveVCSProviderUsers(ctx, workspace, store.VCSProviderUserActiveWindow)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, workspace, users[0].Workspace)
	require.Equal(t, v1pb.VCSType_GITHUB, users[0].VCSType)
	require.Equal(t, "1001", users[0].UserID)
	require.Equal(t, "alice2", users[0].Payload.GetUserName())
	require.Equal(t, "Alice Cooper", users[0].Payload.GetDisplayName())
	require.Greater(t, users[0].LastSeenAt, agedLastSeenAt)
}

func TestVCSProviderUserInactiveUsersDoNotCountAndLimitRejectionKeepsRows(t *testing.T) {
	ctx := context.Background()
	s, db := setupVCSProviderUserStore(ctx, t)

	const workspace = "default"
	_, err := db.ExecContext(ctx, `
		INSERT INTO vcs_provider_user (workspace, vcs_type, user_id, last_seen_at, payload)
		VALUES
			($1, $2, 'active-1', now() - interval '1 day', '{"userName":"active"}'::jsonb),
			($1, $2, 'inactive-1', now() - interval '91 days', '{"userName":"inactive"}'::jsonb)
	`, workspace, v1pb.VCSType_GITHUB.String())
	require.NoError(t, err)

	count, err := s.CountActiveVCSProviderUsers(ctx, workspace, store.VCSProviderUserActiveWindow)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var inactiveLastSeenAt time.Time
	var inactivePayload string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT last_seen_at, payload::text
		FROM vcs_provider_user
		WHERE workspace = $1 AND vcs_type = $2 AND user_id = 'inactive-1'
	`, workspace, v1pb.VCSType_GITHUB.String()).Scan(&inactiveLastSeenAt, &inactivePayload))

	ok, err := s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITHUB,
		UserID:  "inactive-1",
		Payload: &storepb.VCSProviderUserPayload{
			UserName:    "inactive-updated",
			DisplayName: "Inactive Updated",
		},
	}, store.VCSProviderUserActiveWindow, 1)
	require.NoError(t, err)
	require.False(t, ok)

	var gotInactiveLastSeenAt time.Time
	var gotInactivePayload string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT last_seen_at, payload::text
		FROM vcs_provider_user
		WHERE workspace = $1 AND vcs_type = $2 AND user_id = 'inactive-1'
	`, workspace, v1pb.VCSType_GITHUB.String()).Scan(&gotInactiveLastSeenAt, &gotInactivePayload))
	require.Equal(t, inactiveLastSeenAt, gotInactiveLastSeenAt)
	require.JSONEq(t, inactivePayload, gotInactivePayload)

	ok, err = s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITLAB,
		UserID:  "new-1",
		Payload: &storepb.VCSProviderUserPayload{
			UserName: "new",
		},
	}, store.VCSProviderUserActiveWindow, 1)
	require.NoError(t, err)
	require.False(t, ok)

	var exists bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM vcs_provider_user
			WHERE workspace = $1 AND vcs_type = $2 AND user_id = 'new-1'
		)
	`, workspace, v1pb.VCSType_GITLAB.String()).Scan(&exists))
	require.False(t, exists)
}

func TestVCSProviderUserTouchInactiveUserWhenUnderLimit(t *testing.T) {
	ctx := context.Background()
	s, db := setupVCSProviderUserStore(ctx, t)

	const workspace = "default"
	_, err := db.ExecContext(ctx, `
		INSERT INTO vcs_provider_user (workspace, vcs_type, user_id, last_seen_at, payload)
		VALUES ($1, $2, 'inactive-1', now() - interval '91 days', '{"userName":"inactive"}'::jsonb)
	`, workspace, v1pb.VCSType_GITHUB.String())
	require.NoError(t, err)

	ok, err := s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITHUB,
		UserID:  "inactive-1",
		Payload: &storepb.VCSProviderUserPayload{
			UserName:    "active-again",
			DisplayName: "Active Again",
		},
	}, store.VCSProviderUserActiveWindow, 1)
	require.NoError(t, err)
	require.True(t, ok)

	users, err := s.ListActiveVCSProviderUsers(ctx, workspace, store.VCSProviderUserActiveWindow)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "inactive-1", users[0].UserID)
	require.Equal(t, "active-again", users[0].Payload.GetUserName())
}

func TestVCSProviderUserListActiveUsersSortedDesc(t *testing.T) {
	ctx := context.Background()
	s, db := setupVCSProviderUserStore(ctx, t)

	const workspace = "default"
	_, err := db.ExecContext(ctx, `
		INSERT INTO vcs_provider_user (workspace, vcs_type, user_id, last_seen_at, payload)
		VALUES
			($1, $2, 'old-active', now() - interval '2 days', '{"userName":"old"}'::jsonb),
			($1, $2, 'new-active', now() - interval '1 hour', '{"userName":"new"}'::jsonb),
			($1, $2, 'inactive', now() - interval '91 days', '{"userName":"inactive"}'::jsonb)
	`, workspace, v1pb.VCSType_GITHUB.String())
	require.NoError(t, err)

	users, err := s.ListActiveVCSProviderUsers(ctx, workspace, store.VCSProviderUserActiveWindow)
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, "new-active", users[0].UserID)
	require.Equal(t, "old-active", users[1].UserID)
	require.Greater(t, users[0].LastSeenAt, users[1].LastSeenAt)
}

func TestVCSProviderUserTouchStoresEmptyPayloadWhenNil(t *testing.T) {
	ctx := context.Background()
	s, _ := setupVCSProviderUserStore(ctx, t)

	const workspace = "default"
	ok, err := s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITHUB,
		UserID:  "1001",
	}, store.VCSProviderUserActiveWindow, 1)
	require.NoError(t, err)
	require.True(t, ok)

	users, err := s.ListActiveVCSProviderUsers(ctx, workspace, store.VCSProviderUserActiveWindow)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.NotNil(t, users[0].Payload)
	require.Empty(t, users[0].Payload.GetUserName())
	require.Empty(t, users[0].Payload.GetDisplayName())
}

func TestDeleteExpiredVCSProviderUsers(t *testing.T) {
	ctx := context.Background()
	s, db := setupVCSProviderUserStore(ctx, t)

	const workspace = "default"
	_, err := db.ExecContext(ctx, `
		INSERT INTO vcs_provider_user (workspace, vcs_type, user_id, last_seen_at, payload)
		VALUES
			($1, $2, 'active', now() - interval '89 days', '{"userName":"active"}'::jsonb),
			($1, $2, 'expired', now() - interval '91 days', '{"userName":"expired"}'::jsonb)
	`, workspace, v1pb.VCSType_GITHUB.String())
	require.NoError(t, err)

	rowsAffected, err := s.DeleteExpiredVCSProviderUsers(ctx, store.VCSProviderUserActiveWindow)
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)

	var userIDs []string
	rows, err := db.QueryContext(ctx, `
		SELECT user_id
		FROM vcs_provider_user
		WHERE workspace = $1
		ORDER BY user_id
	`, workspace)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var userID string
		require.NoError(t, rows.Scan(&userID))
		userIDs = append(userIDs, userID)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"active"}, userIDs)
}

func setupVCSProviderUserStore(ctx context.Context, t *testing.T) (*store.Store, *sql.DB) {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	return s, db
}
