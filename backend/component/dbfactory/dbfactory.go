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
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.ResourceID)
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
func (d *DBFactory) GetDataSourceDriver(ctx context.Context, instance *store.InstanceMessage, dataSource *storepb.DataSource, databaseName string, datashare, readOnly bool, connectionContext db.ConnectionContext) (db.Driver, error) {
	dbBinDir := ""
	switch instance.Metadata.GetEngine() {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		dbBinDir = d.mysqlBinDir
	case storepb.Engine_POSTGRES, storepb.Engine_RISINGWAVE:
		dbBinDir = d.pgBinDir
	case storepb.Engine_MONGODB:
		dbBinDir = d.mongoBinDir
	}

	if databaseName == "" {
		databaseName = dataSource.GetDatabase()
	}
	password, err := common.Unobfuscate(dataSource.GetObfuscatedPassword(), d.secret)
	if err != nil {
		return nil, err
	}
	sslCA, err := common.Unobfuscate(dataSource.GetObfuscatedSslCa(), d.secret)
	if err != nil {
		return nil, err
	}
	sslCert, err := common.Unobfuscate(dataSource.GetObfuscatedSslCert(), d.secret)
	if err != nil {
		return nil, err
	}
	sslKey, err := common.Unobfuscate(dataSource.GetObfuscatedSslKey(), d.secret)
	if err != nil {
		return nil, err
	}
	sshPassword, err := common.Unobfuscate(dataSource.GetSshObfuscatedPassword(), d.secret)
	if err != nil {
		return nil, err
	}
	sshPrivateKey, err := common.Unobfuscate(dataSource.GetSshObfuscatedPrivateKey(), d.secret)
	if err != nil {
		return nil, err
	}
	authenticationPrivateKey, err := common.Unobfuscate(dataSource.GetAuthenticationPrivateKeyObfuscated(), d.secret)
	if err != nil {
		return nil, err
	}
	masterPassword, err := common.Unobfuscate(dataSource.GetMasterObfuscatedPassword(), d.secret)
	if err != nil {
		return nil, err
	}

	updatedPassword, err := secret.ReplaceExternalSecret(ctx, password, dataSource.GetExternalSecret())
	if err != nil {
		return nil, err
	}
	password = updatedPassword
	sshConfig := db.SSHConfig{
		Host:       dataSource.GetSshHost(),
		Port:       dataSource.GetSshPort(),
		User:       dataSource.GetSshUser(),
		Password:   sshPassword,
		PrivateKey: sshPrivateKey,
	}
	var dbSaslConfig db.SASLConfig
	switch t := dataSource.GetSaslConfig().GetMechanism().(type) {
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
	connectionContext.EngineVersion = instance.Metadata.GetVersion()

	maximumSQLResultSize := d.store.GetMaximumSQLResultLimit(ctx)
	driver, err := db.Open(
		ctx,
		instance.Metadata.GetEngine(),
		db.DriverConfig{
			DbBinDir: dbBinDir,
		},
		db.ConnectionConfig{
			Username: dataSource.GetUsername(),
			Password: password,
			TLSConfig: db.TLSConfig{
				UseSSL:  dataSource.GetUseSsl(),
				SslCA:   sslCA,
				SslCert: sslCert,
				SslKey:  sslKey,
			},
			Host:                      dataSource.GetHost(),
			Port:                      dataSource.GetPort(),
			Database:                  databaseName,
			DataShare:                 datashare,
			SRV:                       dataSource.GetSrv(),
			AuthenticationDatabase:    dataSource.GetAuthenticationDatabase(),
			SID:                       dataSource.GetSid(),
			ServiceName:               dataSource.ServiceName,
			SSHConfig:                 sshConfig,
			ReadOnly:                  readOnly,
			ConnectionContext:         connectionContext,
			AuthenticationPrivateKey:  authenticationPrivateKey,
			AuthenticationType:        dataSource.GetAuthenticationType(),
			SASLConfig:                dbSaslConfig,
			AdditionalAddresses:       dataSource.GetAdditionalAddresses(),
			ReplicaSet:                dataSource.GetReplicaSet(),
			DirectConnection:          dataSource.GetDirectConnection(),
			Region:                    dataSource.GetRegion(),
			WarehouseID:               dataSource.GetWarehouseId(),
			RedisType:                 dataSource.GetRedisType(),
			MasterName:                dataSource.GetMasterName(),
			MasterUsername:            dataSource.GetMasterUsername(),
			MasterPassword:            masterPassword,
			ExtraConnectionParameters: dataSource.GetExtraConnectionParameters(),
			MaximumSQLResultSize:      maximumSQLResultSize,
			ClientSecretCredential:    dataSource.GetClientSecretCredential(),
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}
