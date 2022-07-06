package cmd

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

// GetTestProfile will return a profile for testing.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports.
func GetTestProfile(dataDir string, port int) server.Profile {
	// Using flags.port + 1 as our datastore port
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

// GetTestProfileWithExternalPg will return a profile for testing with external Postgres.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports,
// pgURL for connect to Postgres.
func GetTestProfileWithExternalPg(dataDir string, port int, pgUser string, pgURL string) server.Profile {
	return server.Profile{
		Mode:                 common.ReleaseModeDev,
		BackendHost:          flags.host,
		BackendPort:          port,
		PgUser:               pgUser,
		DataDir:              dataDir,
		DemoDataDir:          fmt.Sprintf("demo/%s", common.ReleaseModeDev),
		BackupRunnerInterval: 10 * time.Second,
		PgURL:                pgURL,
	}
}
