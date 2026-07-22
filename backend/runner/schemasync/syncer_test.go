package schemasync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

// TestSyncDatabaseSchemaNilDatabase pins the guard that prevents a nil
// *store.DatabaseMessage from panicking the syncer. Callers may receive
// (nil, nil) from store.GetDatabase when the referenced database is not
// tracked by Bytebase; the syncer must return a descriptive error instead
// of dereferencing the nil pointer. Regression test for BYT-9309.
func TestSyncDatabaseSchemaNilDatabase(t *testing.T) {
	s := &Syncer{}

	_, err := s.doSyncDatabaseSchema(context.Background(), nil, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil database")

	err = s.SyncDatabaseSchema(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil database")

	_, err = s.SyncDatabaseSchemaToHistory(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil database")
}

func TestGetOrDefaultSyncIntervalUsesEffectiveActivation(t *testing.T) {
	ctx := context.Background()
	customInterval := 10 * time.Minute
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Activation:   false,
			SyncInterval: durationpb.New(customInterval),
		},
	}
	s := &Syncer{}

	require.Equal(t, defaultSyncInterval, s.getOrDefaultSyncInterval(ctx, instance))

	instance.Metadata.Activation = true
	require.Equal(t, customInterval, s.getOrDefaultSyncInterval(ctx, instance))
}

func TestMergeInstanceMetadataPreservesLastSyncTimeForBasicSync(t *testing.T) {
	lastSyncTime := timestamppb.New(time.Unix(100, 0))

	metadata := mergeInstanceMetadata(
		&storepb.Instance{
			Version:      "old-version",
			LastSyncTime: lastSyncTime,
		},
		&db.InstanceMetadata{
			Version: "new-version",
			Metadata: &storepb.Instance{
				MysqlLowerCaseTableNames: 1,
			},
		},
		false,
	)

	require.Equal(t, lastSyncTime.AsTime(), metadata.LastSyncTime.AsTime())
	require.Equal(t, "new-version", metadata.Version)
	require.Equal(t, int32(1), metadata.MysqlLowerCaseTableNames)
}

func TestMergeInstanceMetadataUpdatesLastSyncTimeForFullSync(t *testing.T) {
	lastSyncTime := timestamppb.New(time.Unix(100, 0))
	fullSyncTime := timestamppb.New(time.Unix(200, 0))

	metadata := mergeInstanceMetadata(
		&storepb.Instance{
			LastSyncTime: lastSyncTime,
		},
		&db.InstanceMetadata{
			Metadata: &storepb.Instance{
				LastSyncTime: fullSyncTime,
			},
		},
		true,
	)

	require.Equal(t, fullSyncTime.AsTime(), metadata.LastSyncTime.AsTime())
}
