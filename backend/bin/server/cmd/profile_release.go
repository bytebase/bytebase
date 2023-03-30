//go:build release
// +build release

package cmd

import (
	"time"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func activeProfile(dataDir string) config.Profile {
	p := getBaseProfile(dataDir)
	p.Mode = common.ReleaseModeProd
	p.PgUser = "bb"
	p.BackupRunnerInterval = 10 * time.Minute
	p.AppRunnerInterval = 30 * time.Second
	p.MetricConnectionKey = "so9lLwj5zLjH09sxNabsyVNYSsAHn68F"
	return p
}
