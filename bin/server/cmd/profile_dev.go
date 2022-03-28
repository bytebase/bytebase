//go:build !release
// +build !release

package cmd

import (
	"fmt"
	"time"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) Profile {
	dsn := fmt.Sprintf("file:%s/bytebase_dev.db", dataDir)
	if isDemo {
		dsn = fmt.Sprintf("file:%s/bytebase_demo.db", dataDir)
	}
	return Profile{
		mode:                 "dev",
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bbdev",
		dataDir:              dataDir,
		dsn:                  dsn,
		seedDir:              "seed/test",
		forceResetSeed:       true,
		backupRunnerInterval: 10 * time.Second,
		schemaVersion:        10002,
	}
}

// GetTestProfile will return a profile for testing.
func GetTestProfile(dataDir string, port, datastorePort int) Profile {
	return Profile{
		mode:                 "dev",
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bbtest",
		dataDir:              dataDir,
		dsn:                  fmt.Sprintf("file:%s/bytebase_test.db", dataDir),
		seedDir:              "seed/test",
		forceResetSeed:       true,
		backupRunnerInterval: 10 * time.Second,
		schemaVersion:        10002,
	}
}
