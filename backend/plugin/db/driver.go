// Package db provides the interfaces and libraries for database driver plugins.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
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
	Version string
	// Simplified database metadata.
	Databases []*storepb.DatabaseSchemaMetadata
	Metadata  *storepb.InstanceMetadata
}

// TableKey is the map key for table metadata.
type TableKey struct {
	Schema string
	Table  string
}

// ColumnKey is the map key for table metadata.
type ColumnKey struct {
	Schema string
	Table  string
	Column string
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

func (t MigrationType) NeedDump() bool {
	switch t {
	case Baseline, Migrate, MigrateSDL:
		return true
	default:
		return false
	}
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
	ProjectUID *int
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

	SheetUID *int
	Sheet    *string
}

// ConnectionConfig is the configuration for connections.
type ConnectionConfig struct {
	Host string
	Port string
	// More hosts and ports are required by elasticsearch.
	MultiHosts []string
	MultiPorts []string
	Username   string
	Password   string
	Database   string
	// It's only set for Redshift datashare database.
	DataShare bool
	TLSConfig TLSConfig
	// Only used for Hive.
	SASLConfig SASLConfig
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
	// AuthenticationPrivateKey is used by Snowflake and Databricks (databricks access token).
	AuthenticationPrivateKey string

	ConnectionContext ConnectionContext

	// AuthenticationType is for the database connection, we support normal username & password or Google IAM.
	AuthenticationType storepb.DataSourceOptions_AuthenticationType

	// AdditionalAddresses and ReplicaSet name are used for MongoDB.
	AdditionalAddresses []*storepb.DataSourceOptions_Address
	ReplicaSet          string
	DirectConnection    bool

	// Region is the location of where the DB is, works for AWS RDS.
	Region string

	// WarehouseID is used by Databricks.
	WarehouseID string

	RedisType      storepb.DataSourceOptions_RedisType
	MasterName     string
	MasterUsername string
	MasterPassword string

	// The maximum number of bytes for sql results in response body.
	MaximumSQLResultSize int64
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
	// UseDatabaseOwner is used by Postgres for using role of database owner.
	UseDatabaseOwner bool
}

// QueryContext is the context to query.
type QueryContext struct {
	// Schema is the specific schema for the query.
	// Mainly used for the search path of PostgreSQL.
	Schema string
	// Limit is the maximum row count returned. No limit enforced if limit <= 0
	Limit         int
	Explain       bool
	OperatorEmail string
	Option        *v1pb.QueryOption
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
	Open(ctx context.Context, dbType storepb.Engine, config ConnectionConfig) (Driver, error)
	// Remember to call Close to avoid connection leak
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	GetDB() *sql.DB
	// Execute will execute the statement.
	Execute(ctx context.Context, statement string, opts ExecuteOptions) (int64, error)
	// Used for execute readonly SELECT statement
	QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext QueryContext) ([]*v1pb.QueryResult, error)

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
	// Dump dumps the schema of database.
	Dump(ctx context.Context, out io.Writer, dbSchema *storepb.DatabaseSchemaMetadata) error
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
func Open(ctx context.Context, dbType storepb.Engine, driverConfig DriverConfig, connectionConfig ConnectionConfig) (Driver, error) {
	driversMu.RLock()
	f, ok := drivers[dbType]
	driversMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("db: unknown driver %v", dbType)
	}

	driver, err := f(driverConfig).Open(ctx, dbType, connectionConfig)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// ExecuteOptions is the options for execute.
type ExecuteOptions struct {
	CreateDatabase   bool
	CreateTaskRunLog func(time.Time, *storepb.TaskRunLog) error

	// Record the connection id first before executing.
	SetConnectionID    func(id string)
	DeleteConnectionID func()
}

func (o *ExecuteOptions) LogDatabaseSyncStart() {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type:              storepb.TaskRunLog_DATABASE_SYNC_START,
		DatabaseSyncStart: &storepb.TaskRunLog_DatabaseSyncStart{},
	})
	if err != nil {
		slog.Warn("failed to log database sync start", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogDatabaseSyncEnd(e string) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_DATABASE_SYNC_END,
		DatabaseSyncEnd: &storepb.TaskRunLog_DatabaseSyncEnd{
			Error: e,
		},
	})
	if err != nil {
		slog.Warn("failed to log database sync start", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogSchemaDumpStart() {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type:            storepb.TaskRunLog_SCHEMA_DUMP_START,
		SchemaDumpStart: &storepb.TaskRunLog_SchemaDumpStart{},
	})
	if err != nil {
		slog.Warn("failed to log schema dump start", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogSchemaDumpEnd(derr string) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_SCHEMA_DUMP_END,
		SchemaDumpEnd: &storepb.TaskRunLog_SchemaDumpEnd{
			Error: derr,
		},
	})
	if err != nil {
		slog.Warn("failed to log schema dump end", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogCommandExecute(commandIndexes []int32) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_COMMAND_EXECUTE,
		CommandExecute: &storepb.TaskRunLog_CommandExecute{
			CommandIndexes: commandIndexes,
		},
	})
	if err != nil {
		slog.Warn("failed to log command execute", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogCommandResponse(commandIndexes []int32, affectedRows int32, allAffectedRows []int32, rerr string) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_COMMAND_RESPONSE,
		CommandResponse: &storepb.TaskRunLog_CommandResponse{
			CommandIndexes:  commandIndexes,
			AffectedRows:    affectedRows,
			AllAffectedRows: allAffectedRows,
			Error:           rerr,
		},
	})
	if err != nil {
		slog.Warn("failed to log command response", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogTransactionControl(t storepb.TaskRunLog_TransactionControl_Type, rerr string) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TRANSACTION_CONTROL,
		TransactionControl: &storepb.TaskRunLog_TransactionControl{
			Type:  t,
			Error: rerr,
		},
	})
	if err != nil {
		slog.Warn("failed to log command transaction control", log.BBError(err))
	}
}

// ErrorWithPosition is the error with the position information.
type ErrorWithPosition struct {
	Err   error
	Start *storepb.TaskRunResult_Position
	End   *storepb.TaskRunResult_Position
}

func (e *ErrorWithPosition) Error() string {
	return e.Err.Error()
}

func (e *ErrorWithPosition) Unwrap() error {
	return e.Err
}
