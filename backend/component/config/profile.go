// Package config includes all the server configurations in a component.
package config

import (
	"sync/atomic"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

	// LastActiveTS is the service last active timestamp, any API calls will refresh this value.
	LastActiveTS atomic.Int64
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
	// RuntimeEnableAuditLogStdout enables audit logging to stdout in structured JSON format.
	// can be set in runtime via workspace setting
	RuntimeEnableAuditLogStdout atomic.Bool
}

// UseEmbedDB returns whether to use embedDB.
func (prof *Profile) UseEmbedDB() bool {
	return len(prof.PgURL) == 0
}

var saasFeatureControlMap = map[string]bool{
	storepb.SettingName_AI.String(): true,
}

// IsFeatureUnavailable returns if the feature is unavailable in SaaS mode.
func (prof *Profile) IsFeatureUnavailable(feature string) bool {
	return prof.SaaS && saasFeatureControlMap[feature]
}
