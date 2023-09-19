// Package dbfactory includes the database driver factory.
package dbfactory

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// DBFactory is the factory for building database driver.
type DBFactory struct {
	mysqlBinDir string
	pgBinDir    string
	mongoBinDir string
	dataDir     string
	secret      string
}

// New creates a new database driver factory.
func New(mysqlBinDir, mongoBinDir, pgBinDir, dataDir, secret string) *DBFactory {
	return &DBFactory{
		mysqlBinDir: mysqlBinDir,
		mongoBinDir: mongoBinDir,
		pgBinDir:    pgBinDir,
		dataDir:     dataDir,
		secret:      secret,
	}
}

// GetAdminDatabaseDriver gets the admin database driver using the instance's admin data source.
// Upon successful return, caller must call driver.Close(). Otherwise, it will leak the database connection.
func (d *DBFactory) GetAdminDatabaseDriver(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage) (db.Driver, error) {
	dataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if dataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.Title)
	}
	databaseName := ""
	if database != nil {
		databaseName = database.DatabaseName
	}
	datashare := false
	if database != nil && database.DataShare {
		datashare = true
	}
	if instance.Engine == db.Oracle && database != nil && database.ServiceName != "" {
		// For Oracle, we map CDB as instance and PDB as database.
		// The instance data source is the data source for CDB.
		// So, if the database is not nil, which means we want to connect the PDB, we need to override the database name, service name, and sid.
		dataSource = dataSource.Copy()
		dataSource.Database = database.DatabaseName
		dataSource.ServiceName = database.ServiceName
		dataSource.SID = ""
		databaseName = database.DatabaseName
	}
	schemaTenantMode := false
	if instance.Options != nil && instance.Options.SchemaTenantMode {
		schemaTenantMode = true
	}
	return d.GetDataSourceDriver(ctx, instance.Engine, dataSource, databaseName, instance.ResourceID, instance.UID, datashare, false /* readOnly */, schemaTenantMode)
}

// GetReadOnlyDatabaseDriver gets the read-only database driver using the instance's read-only data source.
// If the read-only data source is not defined, we will fallback to admin data source.
// Upon successful return, caller must call driver.Close(). Otherwise, it will leak the database connection.
func (d *DBFactory) GetReadOnlyDatabaseDriver(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage) (db.Driver, error) {
	dataSource := utils.DataSourceFromInstanceWithType(instance, api.RO)
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	// If there are no read-only data source, fall back to admin data source.
	if dataSource == nil {
		dataSource = adminDataSource
	}
	if dataSource == nil {
		return nil, common.Errorf(common.Internal, "data source not found for instance %q", instance.Title)
	}

	databaseName := ""
	if database != nil {
		databaseName = database.DatabaseName
	}
	if instance.Engine == db.Oracle && database != nil && database.ServiceName != "" {
		// For Oracle, we map CDB as instance and PDB as database.
		// The instance data source is the data source for CDB.
		// So, if the database is not nil, which means we want to connect the PDB, we need to override the database name, service name, and sid.
		dataSource = dataSource.Copy()
		dataSource.Database = database.DatabaseName
		dataSource.ServiceName = database.ServiceName
		dataSource.SID = ""
		databaseName = database.DatabaseName
	}
	schemaTenantMode := false
	if instance.Options != nil && instance.Options.SchemaTenantMode {
		schemaTenantMode = true
	}
	dataShare := false
	if database != nil {
		dataShare = database.DataShare
	}
	return d.GetDataSourceDriver(ctx, instance.Engine, dataSource, databaseName, instance.ResourceID, instance.UID, dataShare, true /* readOnly */, schemaTenantMode)
}

// GetDataSourceDriver returns the database driver for a data source.
func (d *DBFactory) GetDataSourceDriver(ctx context.Context, engine db.Type, dataSource *store.DataSourceMessage, databaseName, instanceID string, instanceUID int, datashare, readOnly bool, schemaTenantMode bool) (db.Driver, error) {
	dbBinDir := ""
	switch engine {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		// TODO(d): use maria mysqlbinlog for MariaDB.
		dbBinDir = d.mysqlBinDir
	case db.Postgres, db.RisingWave:
		dbBinDir = d.pgBinDir
	case db.MongoDB:
		dbBinDir = d.mongoBinDir
	}

	if databaseName == "" {
		databaseName = dataSource.Database
	}
	connectionDatabase := ""
	if datashare {
		connectionDatabase = dataSource.Database
	}
	password, err := common.Unobfuscate(dataSource.ObfuscatedPassword, d.secret)
	if err != nil {
		return nil, err
	}
	sslCA, err := common.Unobfuscate(dataSource.ObfuscatedSslCa, d.secret)
	if err != nil {
		return nil, err
	}
	sslCert, err := common.Unobfuscate(dataSource.ObfuscatedSslCert, d.secret)
	if err != nil {
		return nil, err
	}
	sslKey, err := common.Unobfuscate(dataSource.ObfuscatedSslKey, d.secret)
	if err != nil {
		return nil, err
	}
	sshPassword, err := common.Unobfuscate(dataSource.SSHObfuscatedPassword, d.secret)
	if err != nil {
		return nil, err
	}
	sshPrivateKey, err := common.Unobfuscate(dataSource.SSHObfuscatedPrivateKey, d.secret)
	if err != nil {
		return nil, err
	}
	sshConfig := db.SSHConfig{
		Host:       dataSource.SSHHost,
		Port:       dataSource.SSHPort,
		User:       dataSource.SSHUser,
		Password:   sshPassword,
		PrivateKey: sshPrivateKey,
	}
	driver, err := db.Open(
		ctx,
		engine,
		db.DriverConfig{
			DbBinDir:  dbBinDir,
			BinlogDir: common.GetBinlogAbsDir(d.dataDir, instanceUID),
		},
		db.ConnectionConfig{
			Username: dataSource.Username,
			Password: password,
			TLSConfig: db.TLSConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			},
			Host:                   dataSource.Host,
			Port:                   dataSource.Port,
			Database:               databaseName,
			ConnectionDatabase:     connectionDatabase,
			SRV:                    dataSource.SRV,
			AuthenticationDatabase: dataSource.AuthenticationDatabase,
			SID:                    dataSource.SID,
			ServiceName:            dataSource.ServiceName,
			SSHConfig:              sshConfig,
			ReadOnly:               readOnly,
			SchemaTenantMode:       schemaTenantMode,
		},
		db.ConnectionContext{
			InstanceID: instanceID,
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}
