//go:build release

package cmd

import (
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func activeProfile(dataDir string) *config.Profile {
	p := getBaseProfile(dataDir)
	p.RuntimeDebug.Store(p.Debug)
	p.Mode = common.ReleaseModeProd
	// Enable metric if it's not explicitly disabled and it's not running in demo mode.
	if !flags.disableMetric && !p.Demo {
		p.MetricConnectionKey = "so9lLwj5zLjH09sxNabsyVNYSsAHn68F"
	}
	return p
}
