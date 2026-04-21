package schemasync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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
