package cmd

import (
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/feishu"
)

func getBaseProfile(dataDir string) config.Profile {
	backupStorageBackend := api.BackupStorageBackendLocal
	if flags.backupBucket != "" {
		backupStorageBackend = api.BackupStorageBackendS3
	}

	return config.Profile{
		ExternalURL:          flags.externalURL,
		GrpcPort:             flags.port + 1, // Using flags.port + 1 as our gRPC server port.
		DatastorePort:        flags.port + 2, // Using flags.port + 2 as our datastore port.
		SampleDatabasePort:   flags.port + 3, // Using flags.port + 3 as our sample database port.
		Readonly:             flags.readonly,
		DataDir:              dataDir,
		ResourceDir:          common.GetResourceDir(dataDir),
		Debug:                flags.debug,
		DemoName:             flags.demoName,
		DisallowSignup:       flags.disallowSignup,
		Version:              version,
		GitCommit:            gitcommit,
		PgURL:                flags.pgURL,
		DisableMetric:        flags.disableMetric,
		BackupStorageBackend: backupStorageBackend,
		BackupRegion:         flags.backupRegion,
		BackupBucket:         flags.backupBucket,
		BackupCredentialFile: flags.backupCredential,
		FeishuAPIURL:         feishu.APIPath,
	}
}
