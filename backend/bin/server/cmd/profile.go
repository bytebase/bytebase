package cmd

import (
	"fmt"

	"github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/feishu"
)

func getBaseProfile() config.Profile {
	var demoDataDir string
	if flags.demo {
		demoName := "default"
		if flags.demoName != "" {
			demoName = flags.demoName
		}
		demoDataDir = fmt.Sprintf("demo/%s", demoName)
	}
	backupStorageBackend := api.BackupStorageBackendLocal
	if flags.backupBucket != "" {
		backupStorageBackend = api.BackupStorageBackendS3
	}

	return config.Profile{
		ExternalURL:          flags.externalURL,
		GrpcPort:             flags.port + 1, // Using flags.port + 1 as our gRPC server port.
		DatastorePort:        flags.port + 2, // Using flags.port + 2 as our datastore port.
		Readonly:             flags.readonly,
		Debug:                flags.debug,
		Demo:                 flags.demo,
		DemoDataDir:          demoDataDir,
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
