package server

import (
	"time"

	"github.com/bytebase/bytebase/common"
)

// Profile is the configuration to start main server.
type Profile struct {
	// Mode can be "prod" or "dev"
	Mode common.ReleaseMode
	// Port is the binding port for server.
	Port int
	// DatastorePort is the binding port for database instance for storing Bytebase data.
	DatastorePort int
	// PgUser is the user we use to connect to bytebase's Postgres database.
	// The name of the database storing metadata is the same as pgUser.
	PgUser string
	// DataDir is the directory stores the data including Bytebase's own database, backups, etc.
	DataDir string
	// DemoDataDir points to where to populate the initial data.
	DemoDataDir string
	// BackupRunnerInterval is the interval for backup runner.
	BackupRunnerInterval time.Duration
}
