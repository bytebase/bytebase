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
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// InstanceMetadata is the metadata for an instance.
type InstanceMetadata struct {
	Version string
	// Simplified database metadata.
	Databases []*storepb.DatabaseSchemaMetadata
	Metadata  *storepb.Instance
}

// TableKey is the map key for table metadata.
type TableKey struct {
	Schema string
	Table  string
}

type TableKeyWithColumns struct {
	Schema  string
	Table   string
	Columns []*storepb.ColumnMetadata
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

type ConstraintKey struct {
	Schema     string
	Constraint string
}

type SequenceKey struct {
	Schema   string
	Sequence string
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[storepb.Engine]driverFunc)
)

// DriverConfig is the driver configuration.
type DriverConfig struct {
	// The directiory contains db specific utilites, mongosh for MongoDB.
	DBBinDir string
}

type driverFunc func(DriverConfig) Driver

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

// ConnectionConfig is the configuration for connections.
type ConnectionConfig struct {
	DataSource        *storepb.DataSource
	ConnectionContext ConnectionContext
	Password          string
}

// ConnectionContext is the context for connection.
// It's not used for establishing the db connection, but is useful for logging.
type ConnectionContext struct {
	EnvironmentID        string
	InstanceID           string
	EngineVersion        string
	OperationalComponent string
	// UseDatabaseOwner is used by Postgres for using role of database owner.
	UseDatabaseOwner bool
	DatabaseName     string
	// It's only set for Redshift datashare database.
	DataShare bool
	// ReadOnly is only supported for Postgres at the moment.
	ReadOnly bool
}

// QueryContext is the context to query.
type QueryContext struct {
	// Schema is the specific schema for the query.
	// Mainly used for the search path of PostgreSQL.
	Schema string
	// Container is the specific container for the query.
	// Mainly used for CosmosDB.
	Container string
	// Limit is the maximum row count returned. No limit enforced if limit <= 0
	Limit         int
	Explain       bool
	OperatorEmail string
	Option        *v1pb.QueryOption
	// The maximum number of bytes for sql results in response body.
	MaximumSQLResultSize int64
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
