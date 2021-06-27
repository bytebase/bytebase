// +build !release

package cmd

import (
	"fmt"

	"go.uber.org/zap"
)

func activeProfile(dataDir string, isDemo bool) profile {
	dsn := fmt.Sprintf("file:%s/bytebase_dev.db", dataDir)
	if isDemo {
		dsn = fmt.Sprintf("file:%s/bytebase_demo.db", dataDir)
	}
	return profile{
		demo:      isDemo,
		logConfig: zap.NewDevelopmentConfig(),
		dsn:       dsn,
		seedDir:   "seed/test",
	}
}
