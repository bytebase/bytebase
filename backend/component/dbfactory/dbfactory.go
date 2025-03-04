// Package dbfactory includes the database driver factory.
package dbfactory

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/secret"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DBFactory is the factory for building database driver.
type DBFactory struct {
	mysqlBinDir string
	pgBinDir    string
	mongoBinDir string
	dataDir     string
	secret      string
	store       *store.Store
}

// New creates a new database driver factory.
func New(store *store.Store, mysqlBinDir, mongoBinDir, pgBinDir, dataDir, secret string) *DBFactory {
	return &DBFactory{
		mysqlBinDir: mysqlBinDir,
		mongoBinDir: mongoBinDir,
		pgBinDir:    pgBinDir,
		dataDir:     dataDir,
		secret:      secret,
		store:       store,
	}
}

// GetAdminDatabaseDriver gets the admin database driver using the instance's admin data source.
// Upon successful return, caller must call driver.Close(). Otherwise, it will leak the database connection.
func (d *DBFactory) GetAdminDatabaseDriver(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage, connectionContext db.ConnectionContext) (db.Driver, error) {
	dataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
	if dataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.Title)
	}
	databaseName := ""
	if database != nil {
		databaseName = database.DatabaseName
	}
	datashare := false
	if database != nil && database.Metadata.GetDatashare() {
		datashare = true
	}
	return d.GetDataSourceDriver(ctx, instance, dataSource, databaseName, datashare, false /* readOnly */, connectionContext)
}

// GetDataSourceDriver returns the database driver for a data source.
func (d *DBFactory) GetDataSourceDriver(ctx context.Context, instance *store.InstanceMessage, dataSource *store.DataSourceMessage, databaseName string, datashare, readOnly bool, connectionContext db.ConnectionContext) (db.Driver, error) {
	dbBinDir := ""
	switch instance.Engine {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		// TODO(d): use maria mysqlbinlog for MariaDB.
		dbBinDir = d.mysqlBinDir
	case storepb.Engine_POSTGRES, storepb.Engine_RISINGWAVE:
		dbBinDir = d.pgBinDir
	case storepb.Engine_MONGODB:
		dbBinDir = d.mongoBinDir
	}

	if databaseName == "" {
		databaseName = dataSource.Options.GetDatabase()
	}
	password, err := common.Unobfuscate(dataSource.Options.GetObfuscatedPassword(), d.secret)
	if err != nil {
		return nil, err
	}
	sslCA, err := common.Unobfuscate(dataSource.Options.GetObfuscatedSslCa(), d.secret)
	if err != nil {
		return nil, err
	}
	sslCert, err := common.Unobfuscate(dataSource.Options.GetObfuscatedSslCert(), d.secret)
	if err != nil {
		return nil, err
	}
	sslKey, err := common.Unobfuscate(dataSource.Options.GetObfuscatedSslKey(), d.secret)
	if err != nil {
		return nil, err
	}
	sshPassword, err := common.Unobfuscate(dataSource.Options.GetSshObfuscatedPassword(), d.secret)
	if err != nil {
		return nil, err
	}
	sshPrivateKey, err := common.Unobfuscate(dataSource.Options.GetSshObfuscatedPrivateKey(), d.secret)
	if err != nil {
		return nil, err
	}
	authenticationPrivateKey, err := common.Unobfuscate(dataSource.Options.GetAuthenticationPrivateKeyObfuscated(), d.secret)
	if err != nil {
		return nil, err
	}
	masterPassword, err := common.Unobfuscate(dataSource.Options.GetMasterObfuscatedPassword(), d.secret)
	if err != nil {
		return nil, err
	}

	updatedPassword, err := secret.ReplaceExternalSecret(ctx, password, dataSource.Options.GetExternalSecret())
	if err != nil {
		return nil, err
	}
	password = updatedPassword
	sshConfig := db.SSHConfig{
		Host:       dataSource.Options.GetSshHost(),
		Port:       dataSource.Options.GetSshPort(),
		User:       dataSource.Options.GetSshUser(),
		Password:   sshPassword,
		PrivateKey: sshPrivateKey,
	}
	var dbSaslConfig db.SASLConfig
	switch t := dataSource.Options.GetSaslConfig().GetMechanism().(type) {
	case *storepb.SASLConfig_KrbConfig:
		dbSaslConfig = &db.KerberosConfig{
			Primary:  t.KrbConfig.Primary,
			Instance: t.KrbConfig.Instance,
			Realm: db.Realm{
				Name:                 t.KrbConfig.Realm,
				KDCHost:              t.KrbConfig.KdcHost,
				KDCPort:              t.KrbConfig.KdcPort,
				KDCTransportProtocol: t.KrbConfig.KdcTransportProtocol,
			},
			Keytab: t.KrbConfig.Keytab,
		}
	default:
		dbSaslConfig = nil
	}
	connectionContext.InstanceID = instance.ResourceID
	connectionContext.EngineVersion = instance.EngineVersion

	maximumSQLResultSize := d.store.GetMaximumSQLResultLimit(ctx)
	driver, err := db.Open(
		ctx,
		instance.Engine,
		db.DriverConfig{
			DbBinDir: dbBinDir,
		},
		db.ConnectionConfig{
			Username: dataSource.Options.GetUsername(),
			Password: password,
			TLSConfig: db.TLSConfig{
				UseSSL:  dataSource.Options.GetUseSsl(),
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			},
			Host:                      dataSource.Options.GetHost(),
			Port:                      dataSource.Options.GetPort(),
			Database:                  databaseName,
			DataShare:                 datashare,
			SRV:                       dataSource.Options.GetSrv(),
			AuthenticationDatabase:    dataSource.Options.GetAuthenticationDatabase(),
			SID:                       dataSource.Options.GetSid(),
			ServiceName:               dataSource.Options.ServiceName,
			SSHConfig:                 sshConfig,
			ReadOnly:                  readOnly,
			ConnectionContext:         connectionContext,
			AuthenticationPrivateKey:  authenticationPrivateKey,
			AuthenticationType:        dataSource.Options.GetAuthenticationType(),
			SASLConfig:                dbSaslConfig,
			AdditionalAddresses:       dataSource.Options.GetAdditionalAddresses(),
			ReplicaSet:                dataSource.Options.GetReplicaSet(),
			DirectConnection:          dataSource.Options.GetDirectConnection(),
			Region:                    dataSource.Options.GetRegion(),
			WarehouseID:               dataSource.Options.GetWarehouseId(),
			RedisType:                 dataSource.Options.GetRedisType(),
			MasterName:                dataSource.Options.GetMasterName(),
			MasterUsername:            dataSource.Options.GetMasterUsername(),
			MasterPassword:            masterPassword,
			ExtraConnectionParameters: dataSource.Options.GetExtraConnectionParameters(),
			MaximumSQLResultSize:      maximumSQLResultSize,
			ClientSecretCredential:    dataSource.Options.GetClientSecretCredential(),
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}
