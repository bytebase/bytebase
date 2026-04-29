package schemasync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
