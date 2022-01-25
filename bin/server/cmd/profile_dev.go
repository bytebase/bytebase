//go:build !release
// +build !release

package cmd

import (
	"fmt"
	"time"
)

func activeProfile(dataDir string, isDemo bool) profile {
	dsn := fmt.Sprintf("file:%s/bytebase_dev.db", dataDir)
	if isDemo {
		dsn = fmt.Sprintf("file:%s/bytebase_demo.db", dataDir)
	}
	return profile{
		mode:                 "dev",
		dsn:                  dsn,
		seedDir:              "seed/test",
		forceResetSeed:       true,
		backupRunnerInterval: 10 * time.Second,
	}
}

func GetTestProfile(dataDir string) profile {
	return profile{
		mode:                 "dev",
		dsn:                  fmt.Sprintf("file:%s/bytebase.db", dataDir),
		seedDir:              "seed/test",
		forceResetSeed:       true,
		backupRunnerInterval: 10 * time.Second,
	}
}
