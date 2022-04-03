//go:build release
// +build release

package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) Profile {
	demoDataDir := ""
	if isDemo {
		demoDataDir = fmt.Sprintf("demo/%s", common.ReleaseModeRelease)
	}
	return Profile{
		mode:                 common.ReleaseModeRelease,
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bb",
		dataDir:              dataDir,
		demoDataDir:          demoDataDir,
		backupRunnerInterval: 10 * time.Minute,
	}
}
