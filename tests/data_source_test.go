package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestDataSource(t *testing.T) {
	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "test",
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	a.NoError(err)
	a.Equal(1, len(instance.DataSourceList))

	adminDataSource := instance.DataSourceList[0]
	a.Equal(api.AdminDataSourceName, adminDataSource.Name)
	a.Equal(api.Admin, adminDataSource.Type)

	instanceID := instance.ID

	// create read-only data source.
	err = ctl.createDataSource(api.DataSourceCreate{
		InstanceID: instanceID,
		DatabaseID: adminDataSource.DatabaseID,
		Name:       api.ReadOnlyDataSourceName,
		Type:       api.RO,
		Username:   "ro_ds",
		Password:   "",
		SslCa:      "",
		SslCert:    "",
		SslKey:     "",
	})
	a.NoError(err)

	instance, err = ctl.getInstanceByID(instanceID)
	a.NoError(err)
	a.NotEqual(nil, instance)
	a.Equal(2, len(instance.DataSourceList))

	readOnlyDataSource := instance.DataSourceList[1]
	a.Equal(api.ReadOnlyDataSourceName, readOnlyDataSource.Name)
	a.Equal(api.RO, readOnlyDataSource.Type)

	dataSourceNewName := "updated_ro_ds"
	dataSourceNewPassword := "bytebase"
	// create read-only data source.
	err = ctl.patchDataSource(adminDataSource.DatabaseID, readOnlyDataSource.ID, api.DataSourcePatch{
		Username: &dataSourceNewName,
		Password: &dataSourceNewPassword,
	})
	a.NoError(err)

	dataSourceNewHost := "127.0.0.1"
	dataSourceNewPort := "8000"

	// update read-only data source read replica fields without enterprise license.
	err = ctl.removeLicense()
	a.NoError(err)
	err = ctl.patchDataSource(adminDataSource.DatabaseID, readOnlyDataSource.ID, api.DataSourcePatch{
		Host: &dataSourceNewHost,
		Port: &dataSourceNewPort,
	})
	a.Equal(err.Error(), "http response error code 403 body \"{\\\"message\\\":\\\"Read replica connection is a ENTERPRISE feature, please upgrade to access it.\\\"}\\n\"")

	err = ctl.setLicense()
	a.NoError(err)
	err = ctl.patchDataSource(adminDataSource.DatabaseID, readOnlyDataSource.ID, api.DataSourcePatch{
		Host: &dataSourceNewHost,
		Port: &dataSourceNewPort,
	})
	a.NoError(err)

	err = ctl.deleteDataSource(adminDataSource.DatabaseID, readOnlyDataSource.ID)
	a.NoError(err)

	instance, err = ctl.getInstanceByID(instanceID)
	a.NoError(err)
	a.NotEqual(nil, instance)
	a.Equal(1, len(instance.DataSourceList))
}
