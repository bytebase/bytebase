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
		Port:                 port,
		DatastorePort:        datastorePort,
		PgUser:               "bbdev",
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
	}
}

// GetTestProfile will return a profile for testing.
func GetTestProfile(dataDir string, port, datastorePort int) server.Profile {
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		Port:                 port,
		DatastorePort:        datastorePort,
		PgUser:               "bbtest",
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
	}
}
