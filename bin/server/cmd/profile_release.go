//go:build release
// +build release

package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string, backupStorageBackend api.BackupStorageBackend) server.Profile {
	demoDataDir := ""
	if flags.demo {
		demoName := string(common.ReleaseModeProd)
		if flags.demoName != "" {
			demoName = flags.demoName
		}
		demoDataDir = fmt.Sprintf("demo/%s", demoName)
	}
	// Using flags.port + 1 as our datastore port
	datastorePort := flags.port + 1

	return server.Profile{
		Mode:                 common.ReleaseModeProd,
		BackendHost:          flags.host,
		BackendPort:          flags.port,
		FrontendHost:         flags.frontendHost,
		FrontendPort:         flags.frontendPort,
		DatastorePort:        datastorePort,
		PgUser:               "bb",
		Readonly:             flags.readonly,
		Debug:                flags.debug,
		Demo:                 flags.demo,
		DataDir:              dataDir,
		DemoDataDir:          demoDataDir,
		BackupRunnerInterval: 10 * time.Minute,
		BackupStorageBackend: backupStorageBackend,
		Version:              version,
		GitCommit:            gitcommit,
		PgURL:                flags.pgURL,
		MetricConnectionKey:  "so9lLwj5zLjH09sxNabsyVNYSsAHn68F",
	}
}
