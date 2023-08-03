package cmd

import (
	"time"

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

	sampleDatabasePort := 0
	if !flags.disableSample {
		// Using flags.port + 3 as our sample database port if not disabled.
		sampleDatabasePort = flags.port + 3
	}

	return config.Profile{
		ExternalURL:               flags.externalURL,
		GrpcPort:                  flags.port + 1, // Using flags.port + 1 as our gRPC server port.
		DatastorePort:             flags.port + 2, // Using flags.port + 2 as our datastore port.
		SampleDatabasePort:        sampleDatabasePort,
		Readonly:                  flags.readonly,
		SaaS:                      flags.saas,
		DataDir:                   dataDir,
		ResourceDir:               common.GetResourceDir(dataDir),
		Debug:                     flags.debug,
		DemoName:                  flags.demoName,
		Version:                   version,
		GitCommit:                 gitcommit,
		PgURL:                     flags.pgURL,
		BackupStorageBackend:      backupStorageBackend,
		BackupRegion:              flags.backupRegion,
		BackupBucket:              flags.backupBucket,
		BackupCredentialFile:      flags.backupCredential,
		FeishuAPIURL:              feishu.APIPath,
		LastActiveTs:              time.Now().Unix(),
		DevelopmentUseV2Scheduler: flags.developmentUseV2Scheduler,
	}
}
