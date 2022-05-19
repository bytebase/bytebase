//go:build release
// +build release

package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string) server.Profile {
	demoDataDir := ""
	if flags.demo {
		demoDataDir = fmt.Sprintf("demo/%s", common.ReleaseModeProd)
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
		Version:              version,
		PgURL:                flags.pgURL,
		MetricConnectionKey:  "",
	}
}
