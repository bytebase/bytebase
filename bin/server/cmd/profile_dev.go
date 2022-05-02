//go:build !release
// +build !release

package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) server.Profile {
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		BackendHost:          flags.host,
		BackendPort:          port,
		FrontendHost:         flags.frontendHost,
		FrontendPort:         flags.frontendPort,
		DatastorePort:        datastorePort,
		PgUser:               "bbdev",
		Readonly:             flags.readonly,
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
		Version:              version,
		PgURL:                flags.pgURL,
	}
}

// GetTestProfile will return a profile for testing.
func GetTestProfile(dataDir string, port, datastorePort int) server.Profile {
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		BackendPort:          port,
		DatastorePort:        datastorePort,
		PgUser:               "bbtest",
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
	}
}
