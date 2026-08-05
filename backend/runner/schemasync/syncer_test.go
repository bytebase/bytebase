package schemasync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
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

func TestTrySyncAllSkipsDatabasesInArchivedProjects(t *testing.T) {
	ctx := context.Background()
	s := setupSyncerStore(ctx, t)
	archived := true
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Delete:     &archived,
	}))

	lastSyncTime := timestamppb.New(time.Now())
	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "instance-a",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Activation:   true,
			SyncInterval: durationpb.New(time.Minute),
			LastSyncTime: lastSyncTime,
			Engine:       storepb.Engine_POSTGRES,
			DataSources:  []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "project-a",
		InstanceID:   "instance-a",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)

	syncer := NewSyncer(s, nil, nil)
	syncer.trySyncAll(ctx)

	queued := 0
	syncer.databaseSyncMap.Range(func(_, _ any) bool {
		queued++
		return true
	})
	require.Zero(t, queued)

	archived = false
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Delete:     &archived,
	}))
	syncer.trySyncAll(ctx)
	syncer.databaseSyncMap.Range(func(_, _ any) bool {
		queued++
		return true
	})
	require.Equal(t, 1, queued)
}

func TestCanScheduleInstanceSyncRequiresActiveProject(t *testing.T) {
	ctx := context.Background()
	s := setupSyncerStore(ctx, t)
	projectID := "project-a"
	syncer := NewSyncer(s, nil, nil)
	instance := &store.InstanceMessage{ProjectID: &projectID}

	archived := true
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: projectID,
		Workspace:  "default",
		Delete:     &archived,
	}))
	require.False(t, syncer.canScheduleInstanceSync(ctx, instance))

	archived = false
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: projectID,
		Workspace:  "default",
		Delete:     &archived,
	}))
	require.True(t, syncer.canScheduleInstanceSync(ctx, instance))
}

func setupSyncerStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}
