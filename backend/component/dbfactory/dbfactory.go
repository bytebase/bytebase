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
func (d *DBFactory) GetAdminDatabaseDriver(ctx context.Context, instance *store.InstanceMessage, databaseName string) (db.Driver, error) {
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.Title)
	}

	dbBinDir := ""
	switch instance.Engine {
	case db.MySQL, db.TiDB:
		dbBinDir = d.mysqlBinDir
	case db.Postgres:
		dbBinDir = d.pgBinDir
	case db.MongoDB:
		dbBinDir = d.mongoBinDir
	}

	if databaseName == "" {
		databaseName = adminDataSource.Database
	}
	password, err := common.Unobfuscate(adminDataSource.ObfuscatedPassword, d.secret)
	if err != nil {
		return nil, err
	}
	sslCA, err := common.Unobfuscate(adminDataSource.ObfuscatedSslCa, d.secret)
	if err != nil {
		return nil, err
	}
	sslCert, err := common.Unobfuscate(adminDataSource.ObfuscatedSslCert, d.secret)
	if err != nil {
		return nil, err
	}
	sslKey, err := common.Unobfuscate(adminDataSource.ObfuscatedSslKey, d.secret)
	if err != nil {
		return nil, err
	}
	driver, err := getDatabaseDriver(
		ctx,
		instance.Engine,
		db.DriverConfig{
			DbBinDir:  dbBinDir,
			BinlogDir: common.GetBinlogAbsDir(d.dataDir, instance.UID),
		},
		db.ConnectionConfig{
			Username: adminDataSource.Username,
			Password: password,
			TLSConfig: db.TLSConfig{
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			},
			Host:                   adminDataSource.Host,
			Port:                   adminDataSource.Port,
			Database:               databaseName,
			SRV:                    adminDataSource.SRV,
			AuthenticationDatabase: adminDataSource.AuthenticationDatabase,
		},
		db.ConnectionContext{
			EnvironmentID: instance.EnvironmentID,
			InstanceID:    instance.ResourceID,
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// GetReadOnlyDatabaseDriver gets the read-only database driver using the instance's read-only data source.
// If the read-only data source is not defined, we will fallback to admin data source.
// Upon successful return, caller must call driver.Close(). Otherwise, it will leak the database connection.
func (d *DBFactory) GetReadOnlyDatabaseDriver(ctx context.Context, instance *store.InstanceMessage, databaseName string) (db.Driver, error) {
	dataSource := utils.DataSourceFromInstanceWithType(instance, api.RO)
	// If there are no read-only data source, fall back to admin data source.
	if dataSource == nil {
		dataSource = utils.DataSourceFromInstanceWithType(instance, api.Admin)
	}
	if dataSource == nil {
		return nil, common.Errorf(common.Internal, "data source not found for instance %q", instance.Title)
	}

	host, port := dataSource.Host, dataSource.Port
	if dataSource.Host != "" || dataSource.Port != "" {
		host, port = dataSource.Host, dataSource.Port
	}

	dbBinDir := ""
	switch instance.Engine {
	case db.MySQL, db.TiDB:
		dbBinDir = d.mysqlBinDir
	case db.Postgres:
		dbBinDir = d.pgBinDir
	}

	driver, err := getDatabaseDriver(
		ctx,
		instance.Engine,
		db.DriverConfig{
			DbBinDir:  dbBinDir,
			BinlogDir: common.GetBinlogAbsDir(d.dataDir, instance.UID),
		},
		db.ConnectionConfig{
			Username: dataSource.Username,
			Password: dataSource.ObfuscatedPassword,
			Host:     host,
			Port:     port,
			Database: databaseName,
			TLSConfig: db.TLSConfig{
				SslCA:   dataSource.ObfuscatedSslCa,
				SslCert: dataSource.ObfuscatedSslCert,
				SslKey:  dataSource.ObfuscatedSslKey,
			},
			ReadOnly: true,
		},
		db.ConnectionContext{
			EnvironmentID: instance.EnvironmentID,
			InstanceID:    instance.ResourceID,
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// Retrieve db.Driver connection with standard parameters for all type data source.
func getDatabaseDriver(ctx context.Context, engine db.Type, driverConfig db.DriverConfig, connectionConfig db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	driver, err := db.Open(
		ctx,
		engine,
		driverConfig,
		connectionConfig,
		connCtx,
	)
	if err != nil {
		return nil, common.Wrapf(err, common.DbConnectionFailure, "failed to connect database at %s:%s with user %q", connectionConfig.Host, connectionConfig.Port, connectionConfig.Username)
	}
	return driver, nil
}
