//go:build release
// +build release

package cmd

import (
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string) server.Profile {
	p := getBaseProfile()
	p.Mode = common.ReleaseModeProd
	p.PgUser = "bb"
	p.DataDir = dataDir
	p.BackupRunnerInterval = 10 * time.Minute
	p.MetricConnectionKey = "so9lLwj5zLjH09sxNabsyVNYSsAHn68F"
	return p
}
