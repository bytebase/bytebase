//go:build !release
// +build !release

package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) Profile {
	return Profile{
		mode:                 common.ReleaseModeDev,
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bbdev",
		dataDir:              dataDir,
		demoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		backupRunnerInterval: 10 * time.Second,
	}
}

// GetTestProfile will return a profile for testing.
func GetTestProfile(dataDir string, port, datastorePort int) Profile {
	return Profile{
		mode:                 common.ReleaseModeDev,
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bbtest",
		dataDir:              dataDir,
		demoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		backupRunnerInterval: 10 * time.Second,
	}
}
