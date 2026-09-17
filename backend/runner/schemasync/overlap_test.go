package schemasync

import (
	"context"
	"io"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

// schemaReads gives each SyncDBSchema call on a ClickHouse instance the function that produces its result.
var schemaReads = make(chan func() *metadatapb.DatabaseSchemaMetadata)

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
		return read(), nil
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
	readSchema := func(name string) func() *metadatapb.DatabaseSchemaMetadata {
		return func() *metadatapb.DatabaseSchemaMetadata { return &metadatapb.DatabaseSchemaMetadata{Name: name} }
	}
	handRead := func(read func() *metadatapb.DatabaseSchemaMetadata) {
		select {
		case schemaReads <- read:
		case <-ctx.Done():
			require.FailNow(t, "no sync read the schema", ctx.Err())
		}
	}

	earlierDone, laterDone := make(chan error, 1), make(chan error, 1)
	finishEarlierRead := make(chan struct{})
	go runSync(earlierDone)
	handRead(func() *metadatapb.DatabaseSchemaMetadata {
		<-finishEarlierRead
		return &metadatapb.DatabaseSchemaMetadata{Name: "before-change"}
	})
	go runSync(laterDone)
	select {
	case schemaReads <- readSchema("after-change"):
		// The later sync read while the earlier one was reading: let it store its result first.
		require.NoError(t, <-laterDone)
		close(finishEarlierRead)
	case <-time.After(500 * time.Millisecond):
		// The later sync is waiting for the earlier one.
		close(finishEarlierRead)
		handRead(readSchema("after-change"))
		require.NoError(t, <-laterDone)
	}
	require.NoError(t, <-earlierDone)

	stored, err := stores.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "app"})
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, "after-change", stored.GetProto().GetName())
	require.Empty(t, syncer.databaseSyncLocks)
}

func TestLockDatabaseSyncStopsWaitingWhenCanceled(t *testing.T) {
	syncer := &Syncer{}
	database := &store.DatabaseMessage{InstanceID: "instance-a", DatabaseName: "app"}
	unlock, err := syncer.lockDatabaseSync(context.Background(), database)
	require.NoError(t, err)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = syncer.lockDatabaseSync(canceled, database)
	require.ErrorIs(t, err, context.Canceled)

	unlock()
	require.Empty(t, syncer.databaseSyncLocks)
	unlock, err = syncer.lockDatabaseSync(context.Background(), database)
	require.NoError(t, err)
	unlock()
}
