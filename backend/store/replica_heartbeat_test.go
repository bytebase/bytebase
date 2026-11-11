package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

func TestDeleteReplicaHeartbeat(t *testing.T) {
	ctx := context.Background()
	_, stores, _ := testcontainer.NewMetadataDB(t)

	require.NoError(t, stores.UpsertReplicaHeartbeat(ctx, "replica-a"))
	require.NoError(t, stores.UpsertReplicaHeartbeat(ctx, "replica-b"))
	count, err := stores.CountActiveReplicas(ctx, time.Minute)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	require.NoError(t, stores.DeleteReplicaHeartbeat(ctx, "replica-a"))
	count, err = stores.CountActiveReplicas(ctx, time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	require.NoError(t, stores.DeleteReplicaHeartbeat(ctx, "replica-a"))
	require.NoError(t, stores.DeleteReplicaHeartbeat(ctx, "replica-unknown"))
	count, err = stores.CountActiveReplicas(ctx, time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
