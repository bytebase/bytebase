package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestInstanceSyncPreservesConcurrentConfiguration(t *testing.T) {
	t.Parallel()
	ctx, db, s := newInstanceProjectFixture(t)
	id := "sync-instance"
	metadata := testInstanceMetadata()
	metadata.Version = "old"
	metadata.DataSources[0].Password = "old-password"
	instance, err := s.CreateInstance(ctx, &store.InstanceMessage{ResourceID: id, Workspace: "default", Metadata: metadata})
	require.NoError(t, err)
	staleSync := proto.CloneOf(instance.Metadata)
	changed := proto.CloneOf(instance.Metadata)
	changed.Activation = true
	changed.Vendor = storepb.Vendor_NEON
	changed.DataSources[0].Password = "new-password"
	_, err = s.UpdateInstance(ctx, &store.UpdateInstanceMessage{ResourceID: &id, Workspace: "default", Metadata: changed})
	require.NoError(t, err)
	var credentialsBefore string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT metadata->'dataSources' FROM instance WHERE resource_id = $1`, id).Scan(&credentialsBefore))
	staleSync.Version = "17.1"
	staleSync.LastSyncTime = timestamppb.New(time.Unix(1000, 0))
	staleSync.MysqlLowerCaseTableNames = 1
	staleSync.Roles = []*storepb.InstanceRole{{Name: "app"}}
	updated, err := s.UpdateInstance(ctx, &store.UpdateInstanceMessage{ResourceID: &id, Workspace: "default", SyncMetadata: staleSync})
	require.NoError(t, err)
	require.True(t, updated.Metadata.Activation)
	require.Equal(t, storepb.Vendor_NEON, updated.Metadata.Vendor)
	require.Equal(t, "new-password", updated.Metadata.DataSources[0].Password)
	require.Equal(t, "17.1", updated.Metadata.Version)
	require.True(t, proto.Equal(staleSync.LastSyncTime, updated.Metadata.LastSyncTime))
	require.Equal(t, int32(1), updated.Metadata.MysqlLowerCaseTableNames)
	require.Len(t, updated.Metadata.Roles, 1)
	var credentialsAfter string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT metadata->'dataSources' FROM instance WHERE resource_id = $1`, id).Scan(&credentialsAfter))
	require.Equal(t, credentialsBefore, credentialsAfter)
	updated, err = s.UpdateInstance(ctx, &store.UpdateInstanceMessage{ResourceID: &id, Workspace: "default", SyncMetadata: &storepb.Instance{}})
	require.NoError(t, err)
	require.Empty(t, updated.Metadata.Version)
	require.Nil(t, updated.Metadata.LastSyncTime)
	require.Zero(t, updated.Metadata.MysqlLowerCaseTableNames)
	require.Empty(t, updated.Metadata.Roles)
	require.True(t, updated.Metadata.Activation)
	require.Equal(t, storepb.Vendor_NEON, updated.Metadata.Vendor)
	require.Equal(t, "new-password", updated.Metadata.DataSources[0].Password)
}
