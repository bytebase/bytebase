// Package config includes all the server configurations in a component.
package config

import (
	"sync/atomic"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
)

// Profile is the configuration to start main server.
// Profile must not be copied, its fields must not be modified unless mentioned otherwise.
type Profile struct {
	// Mode can be "prod" or "dev"
	Mode common.ReleaseMode
	// ExternalURL is the URL user visits Bytebase.
	ExternalURL string
	// DatastorePort is the binding port for database instance for storing Bytebase metadata.
	// Only applicable when using embedded PG (PgURL is empty).
	DatastorePort int
	// SampleDatabasePort is the start binding port for sample database instance.
	// If SampleDatabasePort is not 0, then we start 2 sample instance at SampleDatabasePort and SampleDatabasePort+1.
	SampleDatabasePort int
	// Port is the binding port for the server.
	Port int
	// When we are running in SaaS mode, some features are not allowed to edit by users.
	SaaS bool
	// When enabled output logs in json format
	EnableJSONLogging bool
	// DataDir is the directory stores the data including Bytebase's own database, backups, etc.
	DataDir string
	// Demo mode.
	Demo bool
	// HA replica mode.
	HA bool

	// Version is the bytebase's server version
	Version string
	// Git commit hash of the build
	GitCommit string
	// PgURL is the optional external PostgreSQL instance connection url
	PgURL string
	// MetricConnectionKey is the connection key for metric.
	MetricConnectionKey string

	// LastActiveTS is the service last active timestamp, any API calls will refresh this value.
	LastActiveTS int64
	// Unique ID per Bytebase instance run.
	DeployID string
	// Whether the server is running in a docker container.
	IsDocker bool

	// can be set in runtime
	RuntimeDebug atomic.Bool
	// RuntimeMemoryProfileThreshold is the memory threshold in bytes for the server to trigger a pprof memory profile.
	// can be set in runtime
	// 0 means no threshold.
	RuntimeMemoryProfileThreshold atomic.Uint64
}

// UseEmbedDB returns whether to use embedDB.
func (prof *Profile) UseEmbedDB() bool {
	return len(prof.PgURL) == 0
}

var saasFeatureControlMap = map[string]bool{
	string(base.SettingPluginAgent): true,
	string(base.SettingWorkspaceID): true,
}

// IsFeatureUnavailable returns if the feature is unavailable in SaaS mode.
func (prof *Profile) IsFeatureUnavailable(feature string) bool {
	return prof.SaaS && saasFeatureControlMap[feature]
}
