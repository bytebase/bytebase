package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestInstanceSchemaPurgeContention races a sync's UpsertInstanceSchema with
// each purge that deletes the row. A sync that loses to a purge fails, and a
// sync that lands between the purge's instance_schema and instance deletes
// fails the purge; both are the accepted best-effort outcomes of a writer
// racing a purge, and a retry completes the purge.
func TestInstanceSchemaPurgeContention(t *testing.T) {
	t.Parallel()
	for i, p := range []struct {
		name  string
		seed  string
		purge func(*storePostgresFixture) error
	}{
		{
			name:  "instance purge",
			seed:  `INSERT INTO instance (resource_id, workspace, deleted) VALUES ('oracle', 'default', TRUE);`,
			purge: func(f *storePostgresFixture) error { return f.store.DeleteInstance(f.ctx, "default", "oracle") },
		},
		{
			name:  "project purge",
			seed:  `INSERT INTO instance (resource_id, workspace, project) VALUES ('oracle', 'default', 'project-a');`,
			purge: func(f *storePostgresFixture) error { return f.store.DeleteProjects(f.ctx, "default", "project-a") },
		},
	} {
		barrierID := 9991 + 3*i

		t.Run(p.name+" waits for the sync", func(t *testing.T) {
			t.Parallel()
			f := newStorePostgresFixture(t, p.seed)
			require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL")))
			barrier := newMaintenanceLockBarrier(f.ctx, t, f.db, barrierID)
			installMaintenanceLockBarrier(t, f.db, barrierID, "AFTER UPDATE ON instance_schema FOR EACH STATEMENT")

			synced := make(chan error, 1)
			go func() { synced <- f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL", "PLAN_TABLE")) }()
			waitForMaintenanceBarrier(f.ctx, t, f.db, barrierID)
			syncPID := maintenanceBarrierWaitingPID(f.ctx, t, f.db, barrierID)
			purged := make(chan error, 1)
			go func() { purged <- p.purge(f) }()
			waitForBackendBlockedByPID(f.ctx, t, f.db, syncPID)
			barrier.release(t)

			requireMaintenanceResult(t, synced)
			requireMaintenanceResult(t, purged)
			requireNoInstanceSchema(t, f)
		})

		t.Run("a sync waits for the "+p.name+" and fails", func(t *testing.T) {
			t.Parallel()
			f := newStorePostgresFixture(t, p.seed)
			require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL")))
			barrier := newMaintenanceLockBarrier(f.ctx, t, f.db, barrierID+1)
			installMaintenanceLockBarrier(t, f.db, barrierID+1, "AFTER DELETE ON instance_schema FOR EACH STATEMENT")

			purged := make(chan error, 1)
			go func() { purged <- p.purge(f) }()
			waitForMaintenanceBarrier(f.ctx, t, f.db, barrierID+1)
			purgePID := maintenanceBarrierWaitingPID(f.ctx, t, f.db, barrierID+1)
			synced := make(chan error, 1)
			go func() { synced <- f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL", "PLAN_TABLE")) }()
			waitForBackendBlockedByPID(f.ctx, t, f.db, purgePID)
			barrier.release(t)

			requireMaintenanceResult(t, purged)
			requireMaintenanceError(t, synced, "instance_schema_instance_fkey")
			requireNoInstanceSchema(t, f)
		})

		t.Run("a sync between the "+p.name+"'s deletes fails it", func(t *testing.T) {
			t.Parallel()
			f := newStorePostgresFixture(t, p.seed)
			barrier := newMaintenanceLockBarrier(f.ctx, t, f.db, barrierID+2)
			installMaintenanceLockBarrier(t, f.db, barrierID+2, "AFTER DELETE ON instance_schema FOR EACH STATEMENT")

			purged := make(chan error, 1)
			go func() { purged <- p.purge(f) }()
			waitForMaintenanceBarrier(f.ctx, t, f.db, barrierID+2)
			synced := make(chan error, 1)
			go func() { synced <- f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL")) }()
			requireMaintenanceResult(t, synced)
			barrier.release(t)

			requireMaintenanceError(t, purged, "instance_schema_instance_fkey")
			got, err := f.store.GetInstanceSchema(f.ctx, "oracle")
			require.NoError(t, err)
			require.NotNil(t, got, "the failed purge rolled back; the sync's row stands")
			require.NoError(t, p.purge(f))
			requireNoInstanceSchema(t, f)
		})
	}
}

func requireMaintenanceError(t *testing.T, result <-chan error, contains string) {
	t.Helper()
	select {
	case err := <-result:
		require.ErrorContains(t, err, contains)
	case <-time.After(maintenanceLockWait):
		t.Fatal("timed out waiting for maintenance operation")
	}
}

func requireNoInstanceSchema(t *testing.T, f *storePostgresFixture) {
	t.Helper()
	got, err := f.store.GetInstanceSchema(f.ctx, "oracle")
	require.NoError(t, err)
	require.Nil(t, got)
}
