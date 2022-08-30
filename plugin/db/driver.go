// Package db provides the interfaces and libraries for database driver plugins.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/vcs"
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

	// BytebaseDatabase is the database installed in the controlled database server.
	BytebaseDatabase = "bytebase"
)

// User is the database user.
type User struct {
	Name  string
	Grant string
}

// View is the database view.
type View struct {
	Name string
	// CreatedTs isn't supported for ClickHouse.
	CreatedTs  int64
	UpdatedTs  int64
	Definition string
	Comment    string
}

// Extension is the database extension.
type Extension struct {
	Name        string
	Version     string
	Schema      string
	Description string
}

// Index is the database index.
type Index struct {
	Name string
	// This could refer to a column or an expression.
	Expression string
	Position   int
	// Type isn't supported for SQLite.
	Type    string
	Unique  bool
	Primary bool
	// Visible isn't supported for Postgres, SQLite.
	Visible bool
	// Comment isn't supported for SQLite.
	Comment string
}

// Column the database table column.
type Column struct {
	Name     string
	Position int
	Default  *string
	// Nullable isn't supported for ClickHouse.
	Nullable bool
	Type     string
	// CharacterSet isn't supported for Postgres, ClickHouse, SQLite.
	CharacterSet string
	// Collation isn't supported for ClickHouse, SQLite.
	Collation string
	// Comment isn't supported for SQLite.
	Comment string
}

// Table is the database table.
type Table struct {
	Name string
	// CreatedTs isn't supported for ClickHouse, SQLite.
	CreatedTs int64
	// UpdatedTs isn't supported for SQLite.
	UpdatedTs int64
	Type      string
	// Engine isn't supported for Postgres, Snowflake, SQLite.
	Engine string
	// Collation isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	Collation string
	RowCount  int64
	// DataSize isn't supported for SQLite.
	DataSize int64
	// IndexSize isn't supported for ClickHouse, Snowflake, SQLite.
	IndexSize int64
	// DataFree isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	DataFree int64
	// CreateOptions isn't supported for Postgres, ClickHouse, Snowflake, SQLite.
	CreateOptions string
	// Comment isn't supported for SQLite.
	Comment    string
	ColumnList []Column
	// IndexList isn't supported for ClickHouse, Snowflake.
	IndexList []Index
}

// InstanceMeta is the metadata for an instance.
type InstanceMeta struct {
	Version      string
	UserList     []User
	DatabaseList []DatabaseMeta
}

// DatabaseMeta is the metadata for a database.
type DatabaseMeta struct {
	Name string
	// CharacterSet isn't supported for ClickHouse, Snowflake.
	CharacterSet string
	// Collation isn't supported for ClickHouse, Snowflake.
	Collation string
}

// Schema is the database schema.
type Schema struct {
	Name string
	// CharacterSet isn't supported for ClickHouse, Snowflake.
	CharacterSet string
	// Collation isn't supported for ClickHouse, Snowflake.
	Collation     string
	TableList     []Table
	ViewList      []View
	ExtensionList []Extension
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[Type]driverFunc)
)

// DriverConfig is the driver configuration.
type DriverConfig struct {
	PgInstanceDir string
	// We use resource directory to splice the path of embedded binary, likes binaries in mysqlutil package.
	ResourceDir string
	BinlogDir   string
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
// Refer to https://stackoverflow.com/a/6222235/19075342, but we support '.' for now.
const placeholderRegexp = `[^\\/?%*:|"<>]+`

// ParseMigrationInfo matches filePath against filePathTemplate
// If filePath matches, then it will derive MigrationInfo from the filePath.
// Both filePath and filePathTemplate are the full file path (including the base directory) of the repository.
func ParseMigrationInfo(filePath string, filePathTemplate string) (*MigrationInfo, error) {
	placeholderList := []string{
		"ENV_NAME",
		"VERSION",
		"DB_NAME",
		"TYPE",
		"DESCRIPTION",
	}

	// Escape "." characters to match literals instead of using it as a wildcard.
	filePathRegex := strings.ReplaceAll(filePathTemplate, `.`, `\.`)

	filePathRegex = strings.ReplaceAll(filePathRegex, `/*/`, `/[^/]*/`)
	filePathRegex = strings.ReplaceAll(filePathRegex, `**`, `.*`)

	for _, placeholder := range placeholderList {
		filePathRegex = strings.ReplaceAll(filePathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf(`(?P<%s>%s)`, placeholder, placeholderRegexp))
	}
	myRegex, err := regexp.Compile(filePathRegex)
	if err != nil {
		return nil, errors.Errorf("invalid file path template: %q", filePathTemplate)
	}
	if !myRegex.MatchString(filePath) {
		return nil, errors.Errorf("file path %q does not match file path template %q", filePath, filePathTemplate)
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
				case "migrate":
					mi.Type = Migrate
				default:
					return nil, errors.Errorf("file path %q contains invalid migration type %q, must be 'migrate' or 'data'", filePath, matchList[index])
				}
			case "DESCRIPTION":
				mi.Description = matchList[index]
			}
		}
	}

	if mi.Version == "" {
		return nil, errors.Errorf("file path %q does not contain {{VERSION}}, configured file path template %q", filePath, filePathTemplate)
	}
	if mi.Namespace == "" {
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

// MigrationHistory is the API message for migration history.
type MigrationHistory struct {
	ID int

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
	ID *int

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
}

// ConnectionContext is the context for connection.
// It's not used for establishing the db connection, but is useful for logging.
type ConnectionContext struct {
	EnvironmentName string
	InstanceName    string
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
	GetDBConnection(ctx context.Context, database string) (*sql.DB, error)
	// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
	// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
	Execute(ctx context.Context, statement string) error
	// Used for execute readonly SELECT statement
	// limit is the maximum row count returned. No limit enforced if limit <= 0
	Query(ctx context.Context, statement string, limit int) ([]interface{}, error)

	// Sync schema
	// SyncInstance syncs the instance metadata.
	SyncInstance(ctx context.Context) (*InstanceMeta, error)
	// SyncDBSchema syncs a single database schema.
	SyncDBSchema(ctx context.Context, database string) (*Schema, error)

	// Migration related
	// Check whether we need to setup migration (e.g. creating/upgrading the migration related tables)
	NeedsSetupMigration(ctx context.Context) (bool, error)
	// Create or upgrade migration related tables
	SetupMigrationIfNeeded(ctx context.Context) error
	// Execute migration will apply the statement and record the migration history, the schema after migration on success.
	// The migration type is determined by m.Type. Note, it can also perform data migration (DML) in addition to schema migration (DDL).
	// It returns the migration history id and the schema after migration on success.
	ExecuteMigration(ctx context.Context, m *MigrationInfo, statement string) (int64, string, error)
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

// Open opens a database specified by its database driver type and connection config.
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

	if err := driver.Ping(ctx); err != nil {
		_ = driver.Close(ctx)
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
