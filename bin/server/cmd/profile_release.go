//go:build release
// +build release

package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) server.Profile {
	demoDataDir := ""
	if isDemo {
		demoDataDir = fmt.Sprintf("demo/%s", common.ReleaseModeProd)
	}
	return server.Profile{
		Mode:                 common.ReleaseModeProd,
		Port:                 port,
		DatastorePort:        datastorePort,
		PgUser:               "bb",
		DataDir:              dataDir,
		DemoDataDir:          demoDataDir,
		BackupRunnerInterval: 10 * time.Minute,
	}
}
