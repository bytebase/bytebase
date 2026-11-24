// Package db provides the interfaces and libraries for database driver plugins.
package db

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
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
	Databases []*metadatapb.DatabaseSchemaMetadata
	Metadata  *storepb.Instance
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[storepb.Engine]driverFunc)
)

type driverFunc func() Driver

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
	Vendor        storepb.Vendor
	// TenantMode indicates whether to use database owner role for PostgreSQL tenant mode.
	TenantMode   bool
	DatabaseName string
	// It's only set for Redshift datashare database.
	DataShare bool
	// ReadOnly asks the driver to open a read-only database session. Three
	// honor it, all by setting default_transaction_read_only on the connection
	// they open: postgres, cockroachdb, and redshift outside a datashare
	// database. Every other driver ignores it.
	ReadOnly bool
	// TaskRunUID is set when executing a task run, used to set application_name for connection identification.
	TaskRunUID *int64
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
	// SkipMasking skips data masking when the query is authorized by an access grant with unmask=true.
	SkipMasking bool
	// MaskingEnabled means the results will be masked. Masking finds the columns to mask
	// from the statement, so a driver must withhold rows the statement does not trace
	// to columns, such as those a procedure call returns.
	MaskingEnabled bool
	// DataSourceType records which data source runs the query; the Oracle private-link guard
	// exempts the admin one, whose ALL_DB_LINKS the sync recorded.
	DataSourceType storepb.DataSourceType
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
	// Execute will execute the statement.
	Execute(ctx context.Context, statement string, opts ExecuteOptions) (int64, error)
	// Used for execute readonly SELECT statement
	QueryConn(ctx context.Context, statement string, queryContext QueryContext) ([]*v1pb.QueryResult, error)

	// Sync schema
	// SyncInstance syncs the instance metadata.
	SyncInstance(ctx context.Context) (*InstanceMetadata, error)
	// SyncDBSchema syncs a single database schema.
	SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error)

	// Dump dumps the schema of database.
	Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error
}

// TaskRunSessionReader is the capability a driver offers when its engine can
// describe one of its own sessions by application_name, with the sessions it
// blocks and the sessions blocking it (pg: pg_stat_activity and
// pg_blocking_pids; cockroachdb: crdb_internal.cluster_queries and
// crdb_internal.cluster_sessions).
type TaskRunSessionReader interface {
	TaskRunSession(ctx context.Context, applicationName string) (*v1pb.TaskRunSession, error)
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

	return f().Open(ctx, dbType, connectionConfig)
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

func (o *ExecuteOptions) log(action string, entry *storepb.TaskRunLog) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	if err := o.CreateTaskRunLog(time.Now(), entry); err != nil {
		slog.Warn("failed to log "+action, log.BBError(err))
	}
}

func (o *ExecuteOptions) LogComputeDiffStart() {
	o.log("compute diff start", &storepb.TaskRunLog{
		Type:             storepb.TaskRunLog_COMPUTE_DIFF_START,
		ComputeDiffStart: &storepb.TaskRunLog_ComputeDiffStart{},
	})
}

func (o *ExecuteOptions) LogComputeDiffEnd(e string) {
	o.log("compute diff end", &storepb.TaskRunLog{
		Type:           storepb.TaskRunLog_COMPUTE_DIFF_END,
		ComputeDiffEnd: &storepb.TaskRunLog_ComputeDiffEnd{Error: e},
	})
}

func (o *ExecuteOptions) LogDatabaseSyncStart() {
	o.log("database sync start", &storepb.TaskRunLog{
		Type:              storepb.TaskRunLog_DATABASE_SYNC_START,
		DatabaseSyncStart: &storepb.TaskRunLog_DatabaseSyncStart{},
	})
}

func (o *ExecuteOptions) LogDatabaseSyncEnd(e string) {
	o.log("database sync end", &storepb.TaskRunLog{
		Type:            storepb.TaskRunLog_DATABASE_SYNC_END,
		DatabaseSyncEnd: &storepb.TaskRunLog_DatabaseSyncEnd{Error: e},
	})
}

// LogCommandExecute logs the execution of a command.
func (o *ExecuteOptions) LogCommandExecute(commandRange *storepb.Range, commandText string) {
	ce := &storepb.TaskRunLog_CommandExecute{}
	if o != nil && o.LogCommandStatement {
		ce.Statement = commandText
	} else {
		ce.Range = commandRange
	}
	o.log("command execute", &storepb.TaskRunLog{
		Type:           storepb.TaskRunLog_COMMAND_EXECUTE,
		CommandExecute: ce,
	})
}

func (o *ExecuteOptions) LogCommandResponse(affectedRows int64, allAffectedRows []int64, rerr string) {
	o.log("command response", &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_COMMAND_RESPONSE,
		CommandResponse: &storepb.TaskRunLog_CommandResponse{
			AffectedRows:    affectedRows,
			AllAffectedRows: allAffectedRows,
			Error:           rerr,
		},
	})
}

func (o *ExecuteOptions) LogGhostMigrationStart() {
	o.log("gh-ost migration start", &storepb.TaskRunLog{
		Type:                storepb.TaskRunLog_GHOST_MIGRATION_START,
		GhostMigrationStart: &storepb.TaskRunLog_GhostMigrationStart{},
	})
}

func (o *ExecuteOptions) LogGhostMigrationEnd(rerr string) {
	o.log("gh-ost migration end", &storepb.TaskRunLog{
		Type:              storepb.TaskRunLog_GHOST_MIGRATION_END,
		GhostMigrationEnd: &storepb.TaskRunLog_GhostMigrationEnd{Error: rerr},
	})
}

func (o *ExecuteOptions) LogRetryInfo(err error, retryCount int) {
	if o == nil || o.CreateTaskRunLog == nil {
		return
	}
	o.log("retry info", &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_RETRY_INFO,
		RetryInfo: &storepb.TaskRunLog_RetryInfo{
			Error:          err.Error(),
			RetryCount:     int32(retryCount),
			MaximumRetries: int32(o.MaximumRetries),
		},
	})
}

func (o *ExecuteOptions) LogTransactionControl(t storepb.TaskRunLog_TransactionControl_Type, rerr string) {
	o.log("command transaction control", &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TRANSACTION_CONTROL,
		TransactionControl: &storepb.TaskRunLog_TransactionControl{
			Type:  t,
			Error: rerr,
		},
	})
}
