package sqlserver

import "github.com/bytebase/bytebase/backend/common"

// Profile is the configuration to start main server.
type Profile struct {
	// Mode can be "prod" or "dev"
	Mode common.ReleaseMode
	// BackendHost is the listening backend host for server
	BackendHost string
	// BackendPort is the binding backend port for server.
	BackendPort int
	// Debug decides the log level
	Debug bool
	// Version is the bytebase's version
	Version string
	// Git commit hash of the build
	GitCommit string
	// MetricConnectionKey is the connection key for metric.
	MetricConnectionKey string
	// WorkspaceID is the identifier for SQL Service, used by metric.
	WorkspaceID string
}
