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
	}
}

// GetTestProfile will return a profile for testing.
// We require port as an argument so that test can run in parallel in different ports.
func GetTestProfile(dataDir string, port int) server.Profile {
	// Using port + 1 as our datastore port
	datastorePort := port + 1
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		BackendHost:          flags.host,
		BackendPort:          port,
		DatastorePort:        datastorePort,
		PgUser:               "bbtest",
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
	}
}

// GetTestProfileWithExternalPg return a profile for testing with external postgres.
// We require port as an argument so that test acan run in parallel in different ports,
// and pgURL as an argument so that bytebase can connect external postgres with pgURL.
func GetTestProfileWithExternalPg(dataDir, pgURL string, port int) server.Profile {
	// Using port + 1 as our datastore port
	datastorePort := port + 1
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		BackendHost:          flags.host,
		BackendPort:          port,
		DatastorePort:        datastorePort,
		PgURL:                pgURL,
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
	}
}
