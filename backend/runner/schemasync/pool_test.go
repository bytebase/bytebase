package schemasync

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

func init() {
	db.Register(storepb.Engine_MYSQL, func() db.Driver { return &poolTestDriver{} })
}

// Only the target engine is stubbed; metadata reads and transactions use PostgreSQL.
type poolTestDriver struct {
	db.Driver
}

func (d *poolTestDriver) Open(context.Context, storepb.Engine, db.ConnectionConfig) (db.Driver, error) {
	return d, nil
}

func (*poolTestDriver) Close(context.Context) error { return nil }

func (*poolTestDriver) SyncDBSchema(context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return &storepb.DatabaseSchemaMetadata{Name: "app"}, nil
}

func (*poolTestDriver) Dump(context.Context, io.Writer, *storepb.DatabaseSchemaMetadata) error {
	return nil
}

func TestSyncDatabaseSchemaHoldsOnePoolConnection(t *testing.T) {
	ctx := context.Background()
	stores := setupSyncerStore(ctx, t)
	_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "instance-a",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_MYSQL,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	database, err := stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "instance-a", DatabaseName: "app", ProjectID: "project-a",
		Metadata: &storepb.DatabaseMetadata{BackupAvailable: true},
	})
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	syncer := NewSyncer(stores, dbfactory.New(stores, licenseService), licenseService, nil)

	// An absent bbdataarchive forces the backup lookup to read the metadata DB.
	// With one connection, a lookup inside UpdateDatabase's transaction cannot
	// proceed. This models a sync burst occupying every pool slot (BYT-10184).
	stores.GetDB().SetMaxOpenConns(1)
	syncCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	require.NoError(t, syncer.SyncDatabaseSchema(syncCtx, database))

	updated, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID: &database.InstanceID, DatabaseName: &database.DatabaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.False(t, updated.Metadata.BackupAvailable)
	require.Equal(t, storepb.SyncStatus_SYNC_STATUS_OK, updated.Metadata.SyncStatus)
	require.NotNil(t, updated.Metadata.LastSyncTime)
}
