// Package config includes all the server configurations in a component.
package config

import (
	"time"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// Profile is the configuration to start main server.
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
	// GrpcPort is the binding port for gRPC server.
	GrpcPort int
	// PgUser is the user we use to connect to bytebase's Postgres database.
	// The name of the database storing metadata is the same as pgUser.
	PgUser string
	// When we are running in readonly mode:
	// - The data file will be opened in readonly mode, no applicable migration or seeding will be applied.
	// - Requests other than GET will be rejected
	// - Any operations involving mutation will not start (e.g. Background schema syncer, task scheduler)
	Readonly bool
	// When we are running in SaaS mode, some features are not allowed to edit by users.
	SaaS bool
	// DataDir is the directory stores the data including Bytebase's own database, backups, etc.
	DataDir string
	// ResourceDir is the directory stores the resources including embedded postgres, mysqlutil, mongoutil and etc.
	ResourceDir string
	// Debug decides the log level
	Debug bool
	// DemoName specifies the demo name. Empty string means no demo.
	DemoName string
	// AppRunnerInterval is the interval for application runner.
	AppRunnerInterval time.Duration
	// BackupRunnerInterval is the interval for backup runner.
	BackupRunnerInterval time.Duration
	// BackupStorageBackend is the backup storage backend.
	BackupStorageBackend api.BackupStorageBackend

	// Cloud backup related fields
	BackupRegion         string
	BackupBucket         string
	BackupCredentialFile string

	// IM integration related fields
	// FeishuAPIURL is the URL of Feishu API server.
	FeishuAPIURL string

	// Version is the bytebase's server version
	Version string
	// Git commit hash of the build
	GitCommit string
	// PgURL is the optional external PostgreSQL instance connection url
	PgURL string
	// MetricConnectionKey is the connection key for metric.
	MetricConnectionKey string
	// EnableMetric will enable the metric collector.
	EnableMetric bool

	// Test only flag to skip generating onboarding data.
	TestOnlySkipOnboardingData bool

	// LastActiveTs is the service last active timestamp, any API calls will refresh this value.
	LastActiveTs int64

	// Development flags
	DevelopmentUseV2Scheduler bool
}

// UseEmbedDB returns whether to use embedDB.
func (prof *Profile) UseEmbedDB() bool {
	return len(prof.PgURL) == 0
}

var saasFeatureControlMap = map[string]bool{
	string(api.SettingWorkspaceProfile): true,
}

// IsFeatureUnavailable returns if the feature is unavailable in SaaS mode.
func (prof *Profile) IsFeatureUnavailable(feature string) bool {
	return prof.SaaS && saasFeatureControlMap[feature]
}
