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

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Type is the type of a database.
// nolint
type Type string

const (
	// ClickHouse is the database type for CLICKHOUSE.
	ClickHouse Type = "CLICKHOUSE"
	// MySQL is the database type for MYSQL.
	MySQL Type = "MYSQL"
	// Postgres is the database type for POSTGRES.
	Postgres Type = "POSTGRES"
	// Snowflake is the database type for SNOWFLAKE.
	Snowflake Type = "SNOWFLAKE"
	// SQLite is the database type for SQLite.
	SQLite Type = "SQLITE"
	// TiDB is the database type for TiDB.
	TiDB Type = "TIDB"
	// MongoDB is the database type for MongoDB.
	MongoDB Type = "MONGODB"
	// Spanner is the database type for Spanner.
	Spanner Type = "SPANNER"
	// UnknownType is the database type for UNKNOWN.
	UnknownType Type = "UNKNOWN"

	// BytebaseDatabase is the database installed in the controlled database server.
	BytebaseDatabase = "bytebase"
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
	Databases []*storepb.DatabaseMetadata
}

// TableKey is the map key for table metadata.
type TableKey struct {
	Schema string
	Table  string
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[Type]driverFunc)
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

// MigrationInfoPayload is the API message for migration info payload.
type MigrationInfoPayload struct {
	VCSPushEvent *vcs.PushEvent `json:"pushEvent,omitempty"`
}

// MigrationInfo is the API message for migration info.
type MigrationInfo struct {
	ReleaseVersion string
	Version        string
	Namespace      string
	Database       string
	Environment    string
	Source         MigrationSource
	Type           MigrationType
	Status         MigrationStatus
	Description    string
	Creator        string
	IssueID        string
	// Payload contains JSON-encoded string of VCS push event if the migration is triggered by a VCS push event.
	Payload        string
	CreateDatabase bool
	// UseSemanticVersion is whether version is a semantic version.
	// When UseSemanticVersion is set, version should be set to the format specified in Semantic Versioning 2.0.0 (https://semver.org/).
	// For example, for setting non-semantic version "hello", the values should be Version = "hello", UseSemanticVersion = false, SemanticVersionSuffix = "".
	// For setting semantic version "1.2.0", the values should be Version = "1.2.0", UseSemanticVersion = true, SemanticVersionSuffix = "20060102150405" (common.DefaultMigrationVersion).
	UseSemanticVersion bool
	// SemanticVersionSuffix should be set to timestamp format of "20060102150405" (common.DefaultMigrationVersion) if UseSemanticVersion is set.
	// Since stored version should be unique, we have to append a suffix if we allow users to baseline to the same semantic version for fixing schema drift.
	SemanticVersionSuffix string
	// Force is used to execute migration disregarding any migration history with PENDING or FAILED status.
	// This applies to BASELINE and MIGRATE types of migrations because most of these migrations are retry-able.
	// We don't use force option for DATA type of migrations yet till there's customer needs.
	Force bool
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
		"ENV_NAME",
		"VERSION",
		"DB_NAME",
		"TYPE",
		"DESCRIPTION",
	}

	// Escape "." characters to match literals instead of using it as a wildcard.
	filePathRegex := strings.ReplaceAll(filePathTemplate, `.`, `\.`)

	filePathRegex = strings.ReplaceAll(filePathRegex, `/*/`, `/[^/]+/`)
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
			case "ENV_NAME":
				mi.Environment = matchList[index]
			case "VERSION":
				mi.Version = matchList[index]
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

	if mi.Version == "" {
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
		mi.Description = strings.ToUpper(mi.Description[:1]) + mi.Description[1:]
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
		"ENV_NAME",
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
		Environment: info["ENV_NAME"],
		Database:    info["DB_NAME"],
	}, nil
}

// MigrationHistory is the API message for migration history.
type MigrationHistory struct {
	ID string

	Creator   string
	CreatedTs int64
	Updater   string
	UpdatedTs int64

	ReleaseVersion        string
	Namespace             string
	Sequence              int
	Source                MigrationSource
	Type                  MigrationType
	Status                MigrationStatus
	Version               string
	Description           string
	Statement             string
	Schema                string
	SchemaPrev            string
	ExecutionDurationNs   int64
	IssueID               string
	Payload               string
	UseSemanticVersion    bool
	SemanticVersionSuffix string
}

// MigrationHistoryFind is the API message for finding migration histories.
type MigrationHistoryFind struct {
	ID *string

	Database *string
	Source   *MigrationSource
	Version  *string
	// If specified, then it will only fetch "Limit" most recent migration histories
	Limit *int
}

// ConnectionConfig is the configuration for connections.
type ConnectionConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	Database  string
	TLSConfig TLSConfig
	// ReadOnly is only supported for Postgres at the moment.
	ReadOnly bool
	// StrictUseDb will only set as true if the user gives only a database instead of a whole instance to access.
	StrictUseDb bool
	// SRV is only supported for MongoDB now.
	SRV bool
	// AuthenticationDatabase is only supported for MongoDB now.
	AuthenticationDatabase string
}

// ConnectionContext is the context for connection.
// It's not used for establishing the db connection, but is useful for logging.
type ConnectionContext struct {
	EnvironmentID string
	InstanceID    string
}

// QueryContext is the context to query.
type QueryContext struct {
	// Limit is the maximum row count returned. No limit enforced if limit <= 0
	Limit                 int
	ReadOnly              bool
	SensitiveDataMaskType SensitiveDataMaskType
	SensitiveSchemaInfo   *SensitiveSchemaInfo

	// CurrentDatabase is for MySQL
	CurrentDatabase string
}

// DatabaseRoleAttributeMessage is the attribute for role. Docs: https://www.postgresql.org/docs/current/role-attributes.html
type DatabaseRoleAttributeMessage struct {
	// A database superuser bypasses all permission checks, except the right to log in.
	SuperUser bool
	// A role is given permission to inherit the privileges of roles it is a member of. To create a role without the permission, use "noInherit = true"
	NoInherit bool
	// A role must be explicitly given permission to create more roles (except for superusers, since those bypass all permission checks).
	CreateRole bool
	// A role must be explicitly given permission to create databases (except for superusers, since those bypass all permission checks).
	CreateDb bool
	// Only roles that have the LOGIN attribute can be used as the initial role name for a database connection.
	CanLogin bool
	// A role must explicitly be given permission to initiate streaming replication (except for superusers, since those bypass all permission checks).
	Replication bool
	// A role must be explicitly given permission to bypass every row-level security (RLS) policy (except for superusers, since those bypass all permission checks).
	BypassRls bool
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
	Attribute *DatabaseRoleAttributeMessage
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
	Attribute *DatabaseRoleAttributeMessage
}

// Driver is the interface for database driver.
type Driver interface {
	// General execution
	// A driver might support multiple engines (e.g. MySQL driver can support both MySQL and TiDB),
	// So we pass the dbType to tell the exact engine.
	Open(ctx context.Context, dbType Type, config ConnectionConfig, connCtx ConnectionContext) (Driver, error)
	// Remember to call Close to avoid connection leak
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	GetType() Type
	GetDBConnection(ctx context.Context, database string) (*sql.DB, error)
	// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
	// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
	Execute(ctx context.Context, statement string, createDatabase bool) (int64, error)
	// Used for execute readonly SELECT statement
	Query(ctx context.Context, statement string, queryContext *QueryContext) ([]interface{}, error)

	// Sync schema
	// SyncInstance syncs the instance metadata.
	SyncInstance(ctx context.Context) (*InstanceMetadata, error)
	// SyncDBSchema syncs a single database schema.
	SyncDBSchema(ctx context.Context, database string) (*storepb.DatabaseMetadata, error)

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

	// Migration related
	// Check whether we need to setup migration (e.g. creating/upgrading the migration related tables)
	NeedsSetupMigration(ctx context.Context) (bool, error)
	// Create or upgrade migration related tables
	SetupMigrationIfNeeded(ctx context.Context) error
	// Execute migration will apply the statement and record the migration history, the schema after migration on success.
	// The migration type is determined by m.Type. Note, it can also perform data migration (DML) in addition to schema migration (DDL).
	// It returns the migration history id and the schema after migration on success.
	ExecuteMigration(ctx context.Context, m *MigrationInfo, statement string) (string, string, error)
	// Find the migration history list and return most recent item first.
	FindMigrationHistoryList(ctx context.Context, find *MigrationHistoryFind) ([]*MigrationHistory, error)

	// Dump and restore
	// Dump the database, if dbName is empty, then dump all databases.
	// The returned string is the JSON encoded metadata for the logical dump.
	// For MySQL, the payload contains the binlog filename and position when the dump is generated.
	Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error)
	// Restore the database from src, which is a full backup.
	Restore(ctx context.Context, src io.Reader) error
}

// Register makes a database driver available by the provided type.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(dbType Type, f driverFunc) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if f == nil {
		panic("db: Register driver is nil")
	}
	if _, dup := drivers[dbType]; dup {
		panic("db: Register called twice for driver " + dbType)
	}
	drivers[dbType] = f
}

// Open opens a database specified by its database driver type and connection config without verifying the connection.
func Open(ctx context.Context, dbType Type, driverConfig DriverConfig, connectionConfig ConnectionConfig, connCtx ConnectionContext) (Driver, error) {
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

// FormatParamNameInQuestionMark formats the param name in question mark.
// For example, it will be WHERE hello = ? AND world = ?.
func FormatParamNameInQuestionMark(paramNames []string) string {
	if len(paramNames) == 0 {
		return ""
	}
	for i, param := range paramNames {
		if !strings.Contains(param, "?") {
			paramNames[i] = param + " = ?"
		}
	}
	return fmt.Sprintf("WHERE %s ", strings.Join(paramNames, " AND "))
}

// FormatParamNameInNumberedPosition formats the param name in numbered positions.
func FormatParamNameInNumberedPosition(paramNames []string) string {
	if len(paramNames) == 0 {
		return ""
	}
	var parts []string
	for i, param := range paramNames {
		idx := fmt.Sprintf("$%d", i+1)
		param = param + "=" + idx
		parts = append(parts, param)
	}
	return fmt.Sprintf("WHERE %s ", strings.Join(parts, " AND "))
}

// SensitiveDataMaskType is the mask type for sensitive data.
type SensitiveDataMaskType string

const (
	// SensitiveDataMaskTypeDefault is the sensitive data type to hide data with a default method.
	// The default method is subject to change.
	SensitiveDataMaskTypeDefault SensitiveDataMaskType = "DEFAULT"
)

// SensitiveSchemaInfo is the schema info using to extract sensitive fields.
type SensitiveSchemaInfo struct {
	DatabaseList []DatabaseSchema
}

// DatabaseSchema is the database schema using to extract sensitive fields.
type DatabaseSchema struct {
	Name      string
	TableList []TableSchema
}

// TableSchema is the table schema using to extract sensitive fields.
type TableSchema struct {
	Name       string
	ColumnList []ColumnInfo
}

// ColumnInfo is the column info using to extract sensitive fields.
type ColumnInfo struct {
	Name      string
	Sensitive bool
}

// SensitiveField is the struct about SELECT fields.
type SensitiveField struct {
	Name      string
	Sensitive bool
}
