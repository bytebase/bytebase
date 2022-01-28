//go:build release
// +build release

package cmd

import (
	"fmt"
	"time"
)

func activeProfile(dataDir string, port int, isDemo bool) Profile {
	dsn := fmt.Sprintf("file:%s/bytebase.db", dataDir)
	seedDir := "seed/release"
	forceResetSeed := false
	if isDemo {
		dsn = fmt.Sprintf("file:%s/bytebase_demo.db", dataDir)
		seedDir = "seed/test"
		forceResetSeed = true
	}
	return Profile{
		mode:                 "release",
		port:                 port,
		dsn:                  dsn,
		seedDir:              seedDir,
		forceResetSeed:       forceResetSeed,
		backupRunnerInterval: 10 * time.Minute,
	}
}
