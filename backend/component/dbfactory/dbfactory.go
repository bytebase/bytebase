// Package dbfactory includes the database driver factory.
package dbfactory

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	secretlib "github.com/bytebase/bytebase/backend/component/secret"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DBFactory is the factory for building database driver.
type DBFactory struct {
	mongoBinDir string
	store       *store.Store
}

// New creates a new database driver factory.
func New(store *store.Store, mongoBinDir string) *DBFactory {
	return &DBFactory{
		mongoBinDir: mongoBinDir,
		store:       store,
	}
}

// GetAdminDatabaseDriver gets the admin database driver using the instance's admin data source.
// Upon successful return, caller must call driver.Close(). Otherwise, it will leak the database connection.
func (d *DBFactory) GetAdminDatabaseDriver(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage, connectionContext db.ConnectionContext) (db.Driver, error) {
	dataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
	if dataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.ResourceID)
	}
	if database != nil {
		connectionContext.DatabaseName = database.DatabaseName
	}
	return d.GetDataSourceDriver(ctx, instance, dataSource, connectionContext)
}

// GetDataSourceDriver returns the database driver for a data source.
func (d *DBFactory) GetDataSourceDriver(ctx context.Context, instance *store.InstanceMessage, dataSource *storepb.DataSource, connectionContext db.ConnectionContext) (db.Driver, error) {
	dbBinDir := ""
	if instance.Metadata.GetEngine() == storepb.Engine_MONGODB {
		dbBinDir = d.mongoBinDir
	}

	password, err := secretlib.ReplaceExternalSecret(ctx, dataSource.GetPassword(), dataSource.GetExternalSecret())
	if err != nil {
		return nil, err
	}
	connectionContext.InstanceID = instance.ResourceID
	connectionContext.EngineVersion = instance.Metadata.GetVersion()

	driver, err := db.Open(
		ctx,
		instance.Metadata.GetEngine(),
		db.DriverConfig{
			DBBinDir: dbBinDir,
		},
		db.ConnectionConfig{
			DataSource:        dataSource,
			ConnectionContext: connectionContext,
			Password:          password,
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}
