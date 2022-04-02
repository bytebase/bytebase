//go:build !release
// +build !release

package cmd

import (
	"time"

	"github.com/blang/semver/v4"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) Profile {
	return Profile{
		mode:                 "dev",
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bbdev",
		dataDir:              dataDir,
		demoDataDir:          "demo/test",
		backupRunnerInterval: 10 * time.Second,
		schemaVersion:        semver.MustParse("1.1.0"),
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
		demoDataDir:          "demo/test",
		backupRunnerInterval: 10 * time.Second,
		schemaVersion:        semver.MustParse("1.1.0"),
	}
}
