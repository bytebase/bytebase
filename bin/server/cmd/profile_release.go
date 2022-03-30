//go:build release
// +build release

package cmd

import (
	"time"

	"github.com/blang/semver/v4"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) Profile {
	demoDataDir := ""
	if isDemo {
		demoDataDir = "demo/release"
	}
	return Profile{
		mode:                 "release",
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bb",
		dataDir:              dataDir,
		demoDataDir:          demoDataDir,
		backupRunnerInterval: 10 * time.Minute,
		schemaVersion:        semver.MustParse("1.0.0"),
	}
}
