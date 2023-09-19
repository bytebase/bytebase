//go:build release

package cmd

import (
	"github.com/bytebase/bytebase/backend/common"
	server "github.com/bytebase/bytebase/backend/sql-server"
)

func activeProfile() server.Profile {
	return server.Profile{
		Mode:                common.ReleaseModeProd,
		BackendHost:         flags.host,
		BackendPort:         flags.port,
		Version:             version,
		GitCommit:           gitcommit,
		MetricConnectionKey: "46nEgegY1APWBz4NcQp0rCRJomDnLJyE",
		WorkspaceID:         flags.workspaceID,
	}
}
