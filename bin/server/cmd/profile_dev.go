//go:build !release
// +build !release

package cmd

import (
	"fmt"
	"time"
)

func activeProfile(dataDir string, port int, isDemo bool) profile {
	dsn := fmt.Sprintf("file:%s/bytebase_dev.db", dataDir)
	if isDemo {
		dsn = fmt.Sprintf("file:%s/bytebase_demo.db", dataDir)
	}
	return profile{
		mode:                 "dev",
		port:                 port,
		dsn:                  dsn,
		seedDir:              "seed/test",
		forceResetSeed:       true,
		backupRunnerInterval: 10 * time.Second,
	}
}

func GetTestProfile(dataDir string) profile {
	return profile{
		mode:                 "dev",
		port:                 1234,
		dsn:                  fmt.Sprintf("file:%s/bytebase_test.db", dataDir),
		seedDir:              "seed/test",
		forceResetSeed:       true,
		backupRunnerInterval: 10 * time.Second,
	}
}
