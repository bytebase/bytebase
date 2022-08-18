//go:build !release
// +build !release

package cmd

import (
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string) server.Profile {
	p := getBaseProfile()
	p.Mode = common.ReleaseModeDev
	p.PgUser = "bbdev"
	p.DataDir = dataDir
	p.BackupRunnerInterval = 10 * time.Second
	p.MetricConnectionKey = "3zcZLeX3ahvlueEJqNyJysGfVAErsjjT"
	return p
}
