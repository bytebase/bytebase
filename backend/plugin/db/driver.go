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
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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

type driverFunc func() Driver

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
	EnvironmentID string
	InstanceID    string
	EngineVersion string
	// TenantMode indicates whether to use database owner role for PostgreSQL tenant mode.
	TenantMode   bool
	DatabaseName string
	// It's only set for Redshift datashare database.
	DataShare bool
	// ReadOnly is only supported for Postgres at the moment.
	ReadOnly bool
	// MessageBuffer is used for logging messages from the database server.
	MessageBuffer []*v1pb.QueryResult_Message
	// TaskRunUID is set when executing a task run, used to set application_name for connection identification.
	TaskRunUID *int
}

// AppendMessage appends a message to the message buffer.
func (c *ConnectionContext) AppendMessage(message *v1pb.QueryResult_Message) {
	c.MessageBuffer = append(c.MessageBuffer, message)
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
	Timeout              *durationpb.Duration
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
	Dump(ctx context.Context, out io.Writer, dbMetadata *storepb.DatabaseSchemaMetadata) error
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
func Open(ctx context.Context, dbType storepb.Engine, connectionConfig ConnectionConfig) (Driver, error) {
	driversMu.RLock()
	f, ok := drivers[dbType]
	driversMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("db: unknown driver %v", dbType)
	}

	driver, err := f().Open(ctx, dbType, connectionConfig)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// ExecuteOptions is the options for execute.
type ExecuteOptions struct {
	CreateDatabase   bool
	CreateTaskRunLog func(time.Time, *storepb.TaskRunLog) error
	// If true, log the statement of the command.
	// else only log the command index.
	LogCommandStatement bool

	// The maximum number of retries for lock timeout statements.
	MaximumRetries int
}

func (o *ExecuteOptions) LogComputeDiffStart() {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type:             storepb.TaskRunLog_COMPUTE_DIFF_START,
		ComputeDiffStart: &storepb.TaskRunLog_ComputeDiffStart{},
	})
	if err != nil {
		slog.Warn("failed to log compute diff start", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogComputeDiffEnd(e string) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_COMPUTE_DIFF_END,
		ComputeDiffEnd: &storepb.TaskRunLog_ComputeDiffEnd{
			Error: e,
		},
	})
	if err != nil {
		slog.Warn("failed to log compute diff end", log.BBError(err))
	}
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

// LogCommandExecute logs the execution of a command.
func (o *ExecuteOptions) LogCommandExecute(commandRange *storepb.Range, commandText string) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	ce := &storepb.TaskRunLog_CommandExecute{}
	if o.LogCommandStatement {
		ce.Statement = commandText
	} else {
		ce.Range = commandRange
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type:           storepb.TaskRunLog_COMMAND_EXECUTE,
		CommandExecute: ce,
	})
	if err != nil {
		slog.Warn("failed to log command execute", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogCommandResponse(affectedRows int64, allAffectedRows []int64, rerr string) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err := o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_COMMAND_RESPONSE,
		CommandResponse: &storepb.TaskRunLog_CommandResponse{
			AffectedRows:    affectedRows,
			AllAffectedRows: allAffectedRows,
			Error:           rerr,
		},
	})
	if err != nil {
		slog.Warn("failed to log command response", log.BBError(err))
	}
}

func (o *ExecuteOptions) LogRetryInfo(err error, retryCount int) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	err = o.CreateTaskRunLog(time.Now(), &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_RETRY_INFO,
		RetryInfo: &storepb.TaskRunLog_RetryInfo{
			Error:          err.Error(),
			RetryCount:     int32(retryCount),
			MaximumRetries: int32(o.MaximumRetries),
		},
	})
	if err != nil {
		slog.Warn("failed to log retry info", log.BBError(err))
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
