//go:build release

package cmd

import (
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func activeProfile(dataDir string) *config.Profile {
	p := getBaseProfile(dataDir)
	p.Mode = common.ReleaseModeProd
	// Set metric connection key. Actual collection is controlled by workspace setting.
	if !p.Demo {
		p.MetricConnectionKey = "so9lLwj5zLjH09sxNabsyVNYSsAHn68F"
	}
	return p
}
