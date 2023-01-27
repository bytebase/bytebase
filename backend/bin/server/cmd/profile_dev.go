//go:build !release
// +build !release

package cmd

import (
	"time"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func activeProfile(dataDir string) config.Profile {
	p := getBaseProfile(dataDir)
	p.Mode = common.ReleaseModeDev
	p.PgUser = "bbdev"
	p.BackupRunnerInterval = 10 * time.Second
	p.AppRunnerInterval = 30 * time.Second
	p.MetricConnectionKey = "3zcZLeX3ahvlueEJqNyJysGfVAErsjjT"
	return p
}
