package server

import (
	"time"

	"github.com/bytebase/bytebase/common"
)

const (
	// secretLength is the length for the secret used to sign the JWT auto token
	secreatLength = 32
)

// retrieved via the SettingService upon startup
type config struct {
	// secret used to sign the JWT auth token
	secret string
	// deploymentID used to initial the identify for a new deployment.
	deploymentID string
}

// Profile is the configuration to start main server.
type Profile struct {
	// Mode can be "prod" or "dev"
	Mode common.ReleaseMode
	// BackendHost is the listening backend host for server
	BackendHost string
	// BackendPort is the binding backend port for server.
	BackendPort int
	// FrontendHost is the listening frontend host for server.
	FrontendHost string
	// FrontendPort is the listening frontend host for server.
	FrontendPort int
	// DatastorePort is the binding port for database instance for storing Bytebase data.
	DatastorePort int
	// PgUser is the user we use to connect to bytebase's Postgres database.
	// The name of the database storing metadata is the same as pgUser.
	PgUser string
	// When we are running in readonly mode:
	// - The data file will be opened in readonly mode, no applicable migration or seeding will be applied.
	// - Requests other than GET will be rejected
	// - Any operations involving mutation will not start (e.g. Background schema syncer, task scheduler)
	Readonly bool
	// DataDir is the directory stores the data including Bytebase's own database, backups, etc.
	DataDir string
	// Debug decides the log level
	Debug bool
	// Demo decides that whether load demo data.
	Demo bool
	// DemoDataDir points to where to populate the initial data.
	DemoDataDir string
	// BackupRunnerInterval is the interval for backup runner.
	BackupRunnerInterval time.Duration
	// Version is the bytebase's version
	Version string
	// PgURL is the optional external PostgreSQL instance connection url
	PgURL string
	// SegmentKey is the key for segment.
	SegmentKey string
}

func (prof *Profile) useEmbedDB() bool {
	return len(prof.PgURL) == 0
}
