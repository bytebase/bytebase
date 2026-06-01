package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/store"
)

func TestTryAdvisoryXactLockWithStringKeyScopesByKey(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx1.Rollback()
	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx2.Rollback()
	tx3, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx3.Rollback()

	acquired, err := store.TryAdvisoryXactLockWithStringKey(ctx, tx1, store.AdvisoryLockKeyVCSProviderUser, "workspace-a")
	require.NoError(t, err)
	require.True(t, acquired)

	acquired, err = store.TryAdvisoryXactLockWithStringKey(ctx, tx2, store.AdvisoryLockKeyVCSProviderUser, "workspace-a")
	require.NoError(t, err)
	require.False(t, acquired)

	acquired, err = store.TryAdvisoryXactLockWithStringKey(ctx, tx3, store.AdvisoryLockKeyVCSProviderUser, "workspace-b")
	require.NoError(t, err)
	require.True(t, acquired)
}
