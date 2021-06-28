// +build release

package cmd

import (
	"fmt"

	"go.uber.org/zap"
)

func activeProfile(dataDir string, isDemo bool) profile {
	dsn := fmt.Sprintf("file:%s/bytebase.db", dataDir)
	seedDir := "seed/release"
	if isDemo {
		dsn = fmt.Sprintf("file:%s/bytebase_demo.db", dataDir)
		seedDir = "seed/test"
	}
	return profile{
		mode:    "release",
		dsn:     dsn,
		seedDir: seedDir,
	}
}
