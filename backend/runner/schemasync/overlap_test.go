package schemasync

import (
	"context"
	"io"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

// schemaReads gives each SyncDBSchema call on a ClickHouse instance the function that produces its result.
var schemaReads = make(chan func() (*metadatapb.DatabaseSchemaMetadata, error))

func init() {
	db.Register(storepb.Engine_CLICKHOUSE, func() db.Driver { return &scriptedReadDriver{} })
}

type scriptedReadDriver struct {
	db.Driver
}

func (d *scriptedReadDriver) Open(context.Context, storepb.Engine, db.ConnectionConfig) (db.Driver, error) {
	return d, nil
}

func (*scriptedReadDriver) Close(context.Context) error { return nil }

func (*scriptedReadDriver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	select {
	case read := <-schemaReads:
		return read()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (*scriptedReadDriver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}

func TestOverlappingSyncsStoreTheLaterRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stores := setupSyncerStore(ctx, t)
	_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "instance-a",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_CLICKHOUSE,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	database, err := stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "instance-a", DatabaseName: "app", ProjectID: "project-a",
	})
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	syncer := NewSyncer(stores, dbfactory.New(stores, licenseService), licenseService, nil)

	runSync := func(done chan<- error) { done <- syncer.SyncDatabaseSchema(ctx, database) }
	readSchema := func(name string) func() (*metadatapb.DatabaseSchemaMetadata, error) {
		return func() (*metadatapb.DatabaseSchemaMetadata, error) {
			return &metadatapb.DatabaseSchemaMetadata{Name: name}, nil
		}
	}
	handRead := func(read func() (*metadatapb.DatabaseSchemaMetadata, error)) {
		select {
		case schemaReads <- read:
		case <-ctx.Done():
			require.FailNow(t, "no sync read the schema", ctx.Err())
		}
	}

	earlierDone, laterDone := make(chan error, 1), make(chan error, 1)
	finishEarlierRead := make(chan struct{})
	go runSync(earlierDone)
	handRead(func() (*metadatapb.DatabaseSchemaMetadata, error) {
		<-finishEarlierRead
		return &metadatapb.DatabaseSchemaMetadata{Name: "before-change"}, nil
	})
	go runSync(laterDone)
	handRead(readSchema("after-change"))
	require.NoError(t, <-laterDone)
	close(finishEarlierRead)
	require.NoError(t, <-earlierDone)

	stored, err := stores.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "app"})
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, "after-change", stored.GetProto().GetName())
}

func TestFailedSyncRecordsTheFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stores := setupSyncerStore(ctx, t)
	_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "instance-a",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_CLICKHOUSE,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	database, err := stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "instance-a", DatabaseName: "app", ProjectID: "project-a",
	})
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	syncer := NewSyncer(stores, dbfactory.New(stores, licenseService), licenseService, nil)

	syncer.SyncDatabaseAsync(database)
	go func() {
		select {
		case schemaReads <- func() (*metadatapb.DatabaseSchemaMetadata, error) {
			return nil, errors.New("target unreachable")
		}:
		case <-ctx.Done():
		}
	}()
	require.Error(t, syncer.syncQueuedDatabases(ctx))

	updated, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID: &database.InstanceID, DatabaseName: &database.DatabaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, storepb.SyncStatus_SYNC_STATUS_FAILED, updated.Metadata.GetSyncStatus())
	require.Contains(t, updated.Metadata.GetSyncError(), "target unreachable")
	require.NotNil(t, updated.Metadata.GetLastSyncTime())
}

func TestFailedSyncLeavesALaterSyncAlone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stores := setupSyncerStore(ctx, t)
	_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "instance-a",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_CLICKHOUSE,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	database, err := stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "instance-a", DatabaseName: "app", ProjectID: "project-a",
	})
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	syncer := NewSyncer(stores, dbfactory.New(stores, licenseService), licenseService, nil)

	// A sync that started after this one stored its result while it was reading.
	storedAt := time.Now().Add(time.Minute)
	_, err = stores.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		MetadataUpdates: []func(*storepb.DatabaseMetadata){
			func(md *storepb.DatabaseMetadata) {
				md.LastSyncTime = timestamppb.New(storedAt)
				md.SyncStatus = storepb.SyncStatus_SYNC_STATUS_OK
			},
		},
	})
	require.NoError(t, err)

	syncer.SyncDatabaseAsync(database)
	go func() {
		select {
		case schemaReads <- func() (*metadatapb.DatabaseSchemaMetadata, error) {
			return nil, errors.New("target unreachable")
		}:
		case <-ctx.Done():
		}
	}()
	require.Error(t, syncer.syncQueuedDatabases(ctx))

	updated, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID: &database.InstanceID, DatabaseName: &database.DatabaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, storepb.SyncStatus_SYNC_STATUS_OK, updated.Metadata.GetSyncStatus())
	require.True(t, storedAt.Equal(updated.Metadata.GetLastSyncTime().AsTime()))
}

func TestTrySyncAllWaitsOutTheInterval(t *testing.T) {
	ctx := context.Background()
	stores := setupSyncerStore(ctx, t)
	_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "instance-a",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Activation:   true,
			SyncInterval: durationpb.New(time.Hour),
			LastSyncTime: timestamppb.New(time.Now()),
			Engine:       storepb.Engine_CLICKHOUSE,
			DataSources:  []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	database, err := stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID: "project-a", InstanceID: "instance-a", DatabaseName: "app",
		Metadata: &storepb.DatabaseMetadata{LastSyncTime: timestamppb.New(time.Now())},
	})
	require.NoError(t, err)

	syncer := NewSyncer(stores, nil, nil, nil)
	syncer.trySyncAll(ctx)
	require.Zero(t, queuedDatabases(syncer))

	_, err = stores.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		MetadataUpdates: []func(*storepb.DatabaseMetadata){
			func(md *storepb.DatabaseMetadata) {
				md.LastSyncTime = timestamppb.New(time.Now().Add(-2 * time.Hour))
			},
		},
	})
	require.NoError(t, err)
	syncer.trySyncAll(ctx)
	require.Equal(t, 1, queuedDatabases(syncer))
}

func queuedDatabases(syncer *Syncer) int {
	queued := 0
	syncer.databaseSyncMap.Range(func(_, _ any) bool {
		queued++
		return true
	})
	return queued
}
