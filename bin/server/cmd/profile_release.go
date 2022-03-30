//go:build release
// +build release

package cmd

import (
	"fmt"
	"time"
)

func activeProfile(dataDir string, port, datastorePort int, isDemo bool) Profile {
	seedDir := ""
	if isDemo {
		seedDir = "seed/release"
	}
	return Profile{
		mode:                 "release",
		port:                 port,
		datastorePort:        datastorePort,
		pgUser:               "bb",
		dataDir:              dataDir,
		seedDir:              seedDir,
		backupRunnerInterval: 10 * time.Minute,
		schemaVersion:        semver.MustParse("1.0.0"),
	}
}
