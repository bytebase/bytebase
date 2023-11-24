// Package db provides the interfaces and libraries for database driver plugins.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	// SlowQueryMaxLen is the max length of slow query.
	SlowQueryMaxLen = 2048
	// SlowQueryMaxSamplePerFingerprint is the max number of slow query samples per fingerprint.
	SlowQueryMaxSamplePerFingerprint = 100
	// SlowQueryMaxSamplePerDay is the max number of slow query samples per day.
	SlowQueryMaxSamplePerDay = 10000
)

// User is the database user.
type User struct {
	Name  string
	Grant string
}

// InstanceMetadata is the metadata for an instance.
type InstanceMetadata struct {
	Version       string
	InstanceRoles []*storepb.InstanceRoleMetadata
	// Simplified database metadata.
	Databases []*storepb.DatabaseSchemaMetadata
	Metadata  *storepb.InstanceMetadata
}

// TableKey is the map key for table metadata.
type TableKey struct {
	Schema string
	Table  string
}

// IndexKey is the map key for table metadata.
type IndexKey struct {
	Schema string
	Table  string
	Index  string
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[storepb.Engine]driverFunc)
)

// DriverConfig is the driver configuration.
type DriverConfig struct {
	// The directiory contains db specific utilites (e.g. mysqldump for MySQL, pg_dump for PostgreSQL, mongosh for MongoDB).
	DbBinDir string

	// NOTE, introducing db specific fields is the last resort.
	// MySQL specific
	BinlogDir string
}

type driverFunc func(DriverConfig) Driver

// MigrationSource is the migration engine.
type MigrationSource string

const (
	// UI is the migration source type for UI.
	UI MigrationSource = "UI"
	// VCS is the migration source type for VCS.
	VCS MigrationSource = "VCS"
	// LIBRARY is the migration source type for LIBRARY.
	LIBRARY MigrationSource = "LIBRARY"
)

// MigrationType is the type of a migration.
type MigrationType string

const (
	// Baseline is the migration type for BASELINE.
	// Used for establishing schema baseline, this is used when
	// 1. Onboard the database into Bytebase since Bytebase needs to know the current database schema.
	// 2. Had schema drift and need to re-establish the baseline.
	Baseline MigrationType = "BASELINE"
	// Migrate is the migration type for MIGRATE.
	// Used for DDL change including CREATE DATABASE.
	Migrate MigrationType = "MIGRATE"
	// MigrateSDL is the migration type via state-based schema migration.
	// Used for schema change including CREATE DATABASE.
	MigrateSDL MigrationType = "MIGRATE_SDL"
	// Branch is the migration type for BRANCH.
	// Used when restoring from a backup (the restored database branched from the original backup).
	Branch MigrationType = "BRANCH"
	// Data is the migration type for DATA.
	// Used for DML change.
	Data MigrationType = "DATA"
)

// GetVersionTypeSuffix returns the suffix used for schema version string from GitOps.
func (t MigrationType) GetVersionTypeSuffix() string {
	switch t {
	case Migrate:
		return "ddl"
	case Data:
		return "dml"
	case MigrateSDL:
		return "sdl"
	case Baseline:
		return "baseline"
	}
	return ""
}

// MigrationStatus is the status of migration.
type MigrationStatus string

const (
	// Pending is the migration status for PENDING.
	Pending MigrationStatus = "PENDING"
	// Done is the migration status for DONE.
	Done MigrationStatus = "DONE"
	// Failed is the migration status for FAILED.
	Failed MigrationStatus = "FAILED"
)

// MigrationInfo is the API message for migration info.
type MigrationInfo struct {
	// fields for instance change history
	// InstanceID nil is metadata database.
	InstanceID *int
	DatabaseID *int
	IssueUID   *int
	CreatorID  int

	ReleaseVersion string
	Version        model.Version
	Namespace      string
	Database       string
	Environment    string
	Source         MigrationSource
	Type           MigrationType
	Status         MigrationStatus
	Description    string
	Creator        string
	// Payload contains JSON-encoded string of VCS push event if the migration is triggered by a VCS push event.
	Payload *storepb.InstanceChangeHistoryPayload
}

// placeholderRegexp is the regexp for placeholder.
// Refer to https://stackoverflow.com/a/6222235/19075342, but we support "." for now.
const placeholderRegexp = `[^\\/?%*:|"<>]+`

// ParseMigrationInfo matches filePath against filePathTemplate
// If filePath matches, then it will derive MigrationInfo from the filePath.
// Both filePath and filePathTemplate are the full file path (including the base directory) of the repository.
// It returns (nil, nil) if it doesn't look like a migration file path.
func ParseMigrationInfo(filePath, filePathTemplate string, allowOmitDatabaseName bool) (*MigrationInfo, error) {
	placeholderList := []string{
		"ENV_ID",
		"VERSION",
		"DB_NAME",
		"TYPE",
		"DESCRIPTION",
	}

	// Escape "." characters to match literals instead of using it as a wildcard.
	filePathRegex := strings.ReplaceAll(filePathTemplate, `.`, `\.`)

	// We need using for-loop to handle the continuous "*" case.
	// For example, if filePathTemplate is "foo/*/*/bar", then we need to convert it to "foo/[^/]+/[^/]+/bar",
	// but strings.ReplaceAll(filePathRegex, `/*/`, `/[^/]+/`) will only replace the first "*" to "/[^/]+/"(i.e "foo/[^/]+/*/bar")
	tempFilePathRegex := strings.ReplaceAll(filePathRegex, `/*/`, `/[^/]+/`)
	for tempFilePathRegex != filePathRegex {
		filePathRegex = tempFilePathRegex
		tempFilePathRegex = strings.ReplaceAll(filePathRegex, `/*/`, `/[^/]+/`)
	}

	// After the previous for-loop, filePathRegex will not include any "/*/" anymore, so we can safely replace all ** to .*.
	filePathRegex = strings.ReplaceAll(filePathRegex, `**`, `.*`)

	for _, placeholder := range placeholderList {
		filePathRegex = strings.ReplaceAll(filePathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf(`(?P<%s>%s)`, placeholder, placeholderRegexp))
	}
	myRegex, err := regexp.Compile(filePathRegex)
	if err != nil {
		return nil, errors.Errorf("invalid file path template: %q", filePathTemplate)
	}
	if !myRegex.MatchString(filePath) {
		// File path does not match file path template.
		return nil, nil
	}

	mi := &MigrationInfo{
		Source: VCS,
		Type:   Migrate,
	}
	matchList := myRegex.FindStringSubmatch(filePath)
	for _, placeholder := range placeholderList {
		index := myRegex.SubexpIndex(placeholder)
		if index >= 0 {
			switch placeholder {
			case "ENV_ID":
				mi.Environment = matchList[index]
			case "VERSION":
				mi.Version = model.Version{Version: matchList[index]}
			case "DB_NAME":
				mi.Namespace = matchList[index]
				mi.Database = matchList[index]
			case "TYPE":
				switch matchList[index] {
				case "data":
					mi.Type = Data
				case "dml":
					mi.Type = Data
				case "migrate":
					mi.Type = Migrate
				case "ddl":
					mi.Type = Migrate
				default:
					return nil, errors.Errorf("file path %q contains invalid migration type %q, must be 'migrate'('ddl') or 'data'('dml')", filePath, matchList[index])
				}
			case "DESCRIPTION":
				mi.Description = matchList[index]
			}
		}
	}

	if mi.Version.Version == "" {
		return nil, errors.Errorf("file path %q does not contain {{VERSION}}, configured file path template %q", filePath, filePathTemplate)
	}
	if mi.Namespace == "" && !allowOmitDatabaseName {
		return nil, errors.Errorf("file path %q does not contain {{DB_NAME}}, configured file path template %q", filePath, filePathTemplate)
	}

	if mi.Description == "" {
		switch mi.Type {
		case Baseline:
			mi.Description = fmt.Sprintf("Create %s baseline", mi.Database)
		case Data:
			mi.Description = fmt.Sprintf("Create %s data change", mi.Database)
		default:
			mi.Description = fmt.Sprintf("Create %s schema migration", mi.Database)
		}
	} else {
		// Replace _ with space
		mi.Description = strings.ReplaceAll(mi.Description, "_", " ")
		// Capitalize first letter
		description := []rune(mi.Description)
		description[0] = unicode.ToUpper(description[0])
		mi.Description = string(description)
	}

	return mi, nil
}

// ParseSchemaFileInfo attempts to parse the given schema file path to extract
// the schema file info.
// It returns (nil, nil) if it doesn't look like a schema file path.
func ParseSchemaFileInfo(baseDirectory, schemaPathTemplate, file string) (*MigrationInfo, error) {
	if schemaPathTemplate == "" {
		return nil, nil
	}

	// Escape "." characters to match literals instead of using it as a wildcard.
	schemaFilePathRegex := strings.ReplaceAll(schemaPathTemplate, ".", `\.`)

	placeholders := []string{
		"ENV_ID",
		"DB_NAME",
	}
	for _, placeholder := range placeholders {
		schemaFilePathRegex = strings.ReplaceAll(schemaFilePathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9+-=/_#?!$. ]+)", placeholder))
	}

	// NOTE: We do not want to use filepath.Join here because we always need "/" as the path separator.
	re, err := regexp.Compile(path.Join(baseDirectory, schemaFilePathRegex))
	if err != nil {
		return nil, errors.Wrap(err, "compile schema file path regex")
	}
	match := re.FindStringSubmatch(file)
	if len(match) == 0 {
		return nil, nil
	}

	info := make(map[string]string)
	// Skip the first item because it is always the empty string, see docstring of
	// the SubexpNames() method.
	for i, name := range re.SubexpNames()[1:] {
		info[name] = match[i+1]
	}
	return &MigrationInfo{
		Source:      VCS,
		Type:        Migrate,
		Environment: info["ENV_ID"],
		Database:    info["DB_NAME"],
	}, nil
}

// ConnectionConfig is the configuration for connections.
type ConnectionConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	// The database used to connect.
	// It's only set for Redshift datashare database.
	ConnectionDatabase string
	TLSConfig          TLSConfig
	// ReadOnly is only supported for Postgres at the moment.
	ReadOnly bool
	// SRV is only supported for MongoDB now.
	SRV bool
	// AuthenticationDatabase is only supported for MongoDB now.
	AuthenticationDatabase string
	// SID and ServiceName are Oracle only.
	SID         string
	ServiceName string
	SSHConfig   SSHConfig
	// SchemaTenantMode is the Oracle specific mode.
	// If true, bytebase will treat the schema as a database.
	SchemaTenantMode bool
}

// SSHConfig is the configuration for connection over SSH.
type SSHConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	PrivateKey string
}

// ConnectionContext is the context for connection.
// It's not used for establishing the db connection, but is useful for logging.
type ConnectionContext struct {
	EnvironmentID string
	InstanceID    string
	EngineVersion string
}

// QueryContext is the context to query.
type QueryContext struct {
	// Limit is the maximum row count returned. No limit enforced if limit <= 0
	Limit               int
	ReadOnly            bool
	SensitiveSchemaInfo *base.SensitiveSchemaInfo
	// EnableSensitive will set to be true if the database instance has license.
	EnableSensitive bool

	// CurrentDatabase is for MySQL
	CurrentDatabase string
	// ShareDB is for Redshift.
	ShareDB bool
}

// DatabaseRoleMessage is the API message for database role.
type DatabaseRoleMessage struct {
	// The role unique name.
	Name string
	// The connection count limit for this role.
	ConnectionLimit int32
	// The expiration for the role's password.
	ValidUntil *string
	// The role attribute.
	Attribute *string
}

// DatabaseRoleUpsertMessage is the API message for upserting a database role.
type DatabaseRoleUpsertMessage struct {
	// The role unique name.
	Name string
	// A password is only significant if the client authentication method requires the user to supply a password when connecting to the database.
	Password *string
	// Connection limit can specify how many concurrent connections a role can make. -1 (the default) means no limit.
	ConnectionLimit *int32
	// The VALID UNTIL clause sets a date and time after which the role's password is no longer valid. If this clause is omitted the password will be valid for all time.
	ValidUntil *string
	// The role attribute.
	Attribute *string
}

// Driver is the interface for database driver.
type Driver interface {
	// General execution
	// A driver might support multiple engines (e.g. MySQL driver can support both MySQL and TiDB),
	// So we pass the dbType to tell the exact engine.
	Open(ctx context.Context, dbType storepb.Engine, config ConnectionConfig, connCtx ConnectionContext) (Driver, error)
	// Remember to call Close to avoid connection leak
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	GetType() storepb.Engine
	GetDB() *sql.DB
	// Execute will execute the statement.
	Execute(ctx context.Context, statement string, createDatabase bool, opts ExecuteOptions) (int64, error)
	// Used for execute readonly SELECT statement
	QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *QueryContext) ([]*v1pb.QueryResult, error)
	// RunStatement will execute the statement and return the result, for both SELECT and non-SELECT statements.
	RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error)

	// Sync schema
	// SyncInstance syncs the instance metadata.
	SyncInstance(ctx context.Context) (*InstanceMetadata, error)
	// SyncDBSchema syncs a single database schema.
	SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error)

	// Sync slow query logs
	// SyncSlowQuery syncs the slow query logs.
	// The returned map is keyed by database name, and the value is list of slow query statistics grouped by query fingerprint.
	SyncSlowQuery(ctx context.Context, logDateTs time.Time) (map[string]*storepb.SlowQueryStatistics, error)
	// CheckSlowQueryLogEnabled checks if the slow query log is enabled.
	CheckSlowQueryLogEnabled(ctx context.Context) error

	// Role
	// CreateRole creates the role.
	CreateRole(ctx context.Context, upsert *DatabaseRoleUpsertMessage) (*DatabaseRoleMessage, error)
	// UpdateRole updates the role.
	UpdateRole(ctx context.Context, roleName string, upsert *DatabaseRoleUpsertMessage) (*DatabaseRoleMessage, error)
	// FindRole finds the role by name.
	FindRole(ctx context.Context, roleName string) (*DatabaseRoleMessage, error)
	// ListRole lists the role.
	ListRole(ctx context.Context) ([]*DatabaseRoleMessage, error)
	// DeleteRole deletes the role by name.
	DeleteRole(ctx context.Context, roleName string) error

	// Dump and restore
	// Dump the database.
	// The returned string is the JSON encoded metadata for the logical dump.
	// For MySQL, the payload contains the binlog filename and position when the dump is generated.
	Dump(ctx context.Context, out io.Writer, schemaOnly bool) (string, error)
	// Restore the database from src, which is a full backup.
	Restore(ctx context.Context, src io.Reader) error
}

// Register makes a database driver available by the provided type.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(dbType storepb.Engine, f driverFunc) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if f == nil {
		panic("db: Register driver is nil")
	}
	if _, dup := drivers[dbType]; dup {
		panic(fmt.Sprintf("db: Register called twice for driver %s", dbType))
	}
	drivers[dbType] = f
}

// Open opens a database specified by its database driver type and connection config without verifying the connection.
func Open(ctx context.Context, dbType storepb.Engine, driverConfig DriverConfig, connectionConfig ConnectionConfig, connCtx ConnectionContext) (Driver, error) {
	driversMu.RLock()
	f, ok := drivers[dbType]
	driversMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("db: unknown driver %v", dbType)
	}

	driver, err := f(driverConfig).Open(ctx, dbType, connectionConfig, connCtx)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// ExecuteOptions is the options for execute.
type ExecuteOptions struct {
	BeginFunc          func(ctx context.Context, conn *sql.Conn) error
	EndTransactionFunc func(tx *sql.Tx) error
	// ChunkedSubmission is the flag to indicate if we should use chunked submission for the statement.
	// If true, we will submit each statement chunk, otherwise we will submit all statements in a batch.
	// For both cases, we will use one transaction to wrap the statements.
	ChunkedSubmission     bool
	UpdateExecutionStatus func(*v1pb.TaskRun_ExecutionDetail)
}
