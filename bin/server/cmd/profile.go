package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func getBaseProfile() server.Profile {
	var demoDataDir string
	if flags.demo {
		demoName := string(common.ReleaseModeDev)
		if flags.demoName != "" {
			demoName = flags.demoName
		}
		demoDataDir = fmt.Sprintf("demo/%s", demoName)
	}
	backupStorageBackend := api.BackupStorageBackendLocal
	if flags.backupBucket != "" {
		backupStorageBackend = api.BackupStorageBackendS3
	}
	// Using flags.port + 1 as our datastore port
	datastorePort := flags.port + 1

	return server.Profile{
		ExternalURL:          flags.externalURL,
		DatastorePort:        datastorePort,
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
		FeishuAPIURL:         flags.feishuAPIURL,
	}
}

// GetTestProfile will return a profile for testing.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports.
func GetTestProfile(dataDir string, port int, feishuAPIURL string) server.Profile {
	// Using flags.port + 1 as our datastore port
	datastorePort := port + 1
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		ExternalURL:          fmt.Sprintf("http://localhost:%d", port),
		DatastorePort:        datastorePort,
		PgUser:               "bbtest",
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
		BackupStorageBackend: api.BackupStorageBackendLocal,
		FeishuAPIURL:         feishuAPIURL,
	}
}

// GetTestProfileWithExternalPg will return a profile for testing with external Postgres.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports,
// pgURL for connect to Postgres.
func GetTestProfileWithExternalPg(dataDir string, port int, pgUser string, pgURL string, feishuAPIURL string) server.Profile {
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		ExternalURL:          fmt.Sprintf("http://localhost:%d", port),
		PgUser:               pgUser,
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
		BackupStorageBackend: api.BackupStorageBackendLocal,
		FeishuAPIURL:         feishuAPIURL,
		PgURL:                pgURL,
	}
}
