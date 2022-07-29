//go:build !release
// +build !release

package cmd

import (
	"github.com/bytebase/bytebase/common"
	server "github.com/bytebase/bytebase/sql-server"
)

func activeProfile(dataDir string) server.Profile {
	return server.Profile{
		Mode:                common.ReleaseModeDev,
		BackendHost:         flags.host,
		BackendPort:         flags.port,
		Debug:               flags.debug,
		Version:             version,
		DataDir:             dataDir,
		GitCommit:           gitcommit,
		MetricConnectionKey: "",
	}
}
