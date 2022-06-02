//go:build !release
// +build !release

package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string) server.Profile {
	// Using flags.port + 1 as our datastore port
	datastorePort := flags.port + 1
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		BackendHost:          flags.host,
		BackendPort:          flags.port,
		FrontendHost:         flags.frontendHost,
		FrontendPort:         flags.frontendPort,
		DatastorePort:        datastorePort,
		PgUser:               "bbdev",
		Readonly:             flags.readonly,
		Debug:                flags.debug,
		Demo:                 flags.demo,
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
		Version:              version,
		PgURL:                flags.pgURL,
		MetricConnectionKey:  "3zcZLeX3ahvlueEJqNyJysGfVAErsjjT",
	}
}
