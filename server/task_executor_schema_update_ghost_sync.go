package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/github/gh-ost/go/base"
	"github.com/github/gh-ost/go/logic"
	ghostsql "github.com/github/gh-ost/go/sql"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
)

// NewSchemaUpdateGhostSyncTaskExecutor creates a schema update (gh-ost) sync task executor.
func NewSchemaUpdateGhostSyncTaskExecutor() TaskExecutor {
	return &SchemaUpdateGhostSyncTaskExecutor{}
}

// SchemaUpdateGhostSyncTaskExecutor is the schema update (gh-ost) sync task executor.
type SchemaUpdateGhostSyncTaskExecutor struct {
	completed int32
}

// RunOnce will run SchemaUpdateGhostSync task once.
func (exec *SchemaUpdateGhostSyncTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update gh-ost sync payload: %w", err)
	}
	return runGhostMigration(ctx, server, task, payload.Statement)
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *SchemaUpdateGhostSyncTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress
func (exec *SchemaUpdateGhostSyncTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}

func getSocketFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.sock", taskID, databaseID, databaseName, tableName)
}

func getPostponeFlagFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.postponeFlag", taskID, databaseID, databaseName, tableName)
}

func getTableNameFromStatement(statement string) (string, error) {
	// Trim the statement for the parser.
	// This in effect removes all leading and trailing spaces, substitute multiple spaces with one.
	statement = strings.Join(strings.Fields(statement), " ")
	parser := ghostsql.NewParserFromAlterStatement(statement)
	if !parser.HasExplicitTable() {
		return "", fmt.Errorf("failed to parse table name from statement, statement: %v", statement)
	}
	return parser.GetExplicitTable(), nil
}

type ghostConfig struct {
	// serverID should be unique
	serverID             uint
	host                 string
	port                 string
	user                 string
	password             string
	database             string
	table                string
	alterStatement       string
	socketFilename       string
	postponeFlagFilename string
	noop                 bool
}

func newMigrationContext(config ghostConfig) (*base.MigrationContext, error) {
	const (
		allowedRunningOnMaster              = true
		concurrentCountTableRows            = true
		timestampAllTable                   = true
		hooksStatusIntervalSec              = 60
		heartbeatIntervalMilliseconds       = 100
		niceRatio                           = 0
		chunkSize                           = 1000
		dmlBatchSize                        = 10
		maxLagMillisecondsThrottleThreshold = 1500
		defaultNumRetries                   = 60
		cutoverLockTimeoutSeconds           = 3
		exponentialBackoffMaxInterval       = 64
	)
	statement := strings.Join(strings.Fields(config.alterStatement), " ")
	migrationContext := base.NewMigrationContext()
	migrationContext.InspectorConnectionConfig.Key.Hostname = config.host
	port := 3306
	if config.port != "" {
		configPort, err := strconv.Atoi(config.port)
		if err != nil {
			return nil, fmt.Errorf("failed to convert port from string to int, error: %w", err)
		}
		port = configPort
	}
	migrationContext.InspectorConnectionConfig.Key.Port = port
	migrationContext.CliUser = config.user
	migrationContext.CliPassword = config.password
	migrationContext.DatabaseName = config.database
	migrationContext.OriginalTableName = config.table
	migrationContext.AlterStatement = statement
	migrationContext.Noop = config.noop
	migrationContext.ReplicaServerId = config.serverID
	// set defaults
	migrationContext.AllowedRunningOnMaster = allowedRunningOnMaster
	migrationContext.ConcurrentCountTableRows = concurrentCountTableRows
	migrationContext.HooksStatusIntervalSec = hooksStatusIntervalSec
	migrationContext.CutOverType = base.CutOverAtomic

	if migrationContext.AlterStatement == "" {
		return nil, fmt.Errorf("alterStatement must be provided and must not be empty")
	}
	parser := ghostsql.NewParserFromAlterStatement(migrationContext.AlterStatement)
	migrationContext.AlterStatementOptions = parser.GetAlterStatementOptions()

	if migrationContext.DatabaseName == "" {
		if !parser.HasExplicitSchema() {
			return nil, fmt.Errorf("database must be provided and database name must not be empty, or alterStatement must specify database name")
		}
		migrationContext.DatabaseName = parser.GetExplicitSchema()
	}
	if migrationContext.OriginalTableName == "" {
		if !parser.HasExplicitTable() {
			return nil, fmt.Errorf("table must be provided and table name must not be empty, or alterStatement must specify table name")
		}
		migrationContext.OriginalTableName = parser.GetExplicitTable()
	}
	migrationContext.ServeSocketFile = config.socketFilename
	migrationContext.PostponeCutOverFlagFile = config.postponeFlagFilename
	migrationContext.TimestampAllTable = timestampAllTable
	migrationContext.SetHeartbeatIntervalMilliseconds(heartbeatIntervalMilliseconds)
	migrationContext.SetNiceRatio(niceRatio)
	migrationContext.SetChunkSize(chunkSize)
	migrationContext.SetDMLBatchSize(dmlBatchSize)
	migrationContext.SetMaxLagMillisecondsThrottleThreshold(maxLagMillisecondsThrottleThreshold)
	migrationContext.SetDefaultNumRetries(defaultNumRetries)
	migrationContext.ApplyCredentials()
	if err := migrationContext.SetCutOverLockTimeoutSeconds(cutoverLockTimeoutSeconds); err != nil {
		return nil, err
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(exponentialBackoffMaxInterval); err != nil {
		return nil, err
	}
	return migrationContext, nil
}

func runGhostMigration(_ context.Context, _ *Server, task *api.Task, statement string) (terminated bool, result *api.TaskRunResultPayload, err error) {
	syncDone := make(chan struct{})
	syncError := make(chan error)

	go func() {
		statement = strings.TrimSpace(statement)
		err := executeGhost(task, statement, syncDone)
		if err != nil {
			log.Error("failed to execute schema update gh-ost sync executeSync", zap.Error(err))
			// There could be an error in gh-ost migration after the syncDone channel returns which causes the outer function returns, too.
			// Then there's no consumer of syncError, so we must make sending to syncError non-blocking.
			select {
			case syncError <- fmt.Errorf("failed to execute schema update gh-ost sync executeSync, error: %w", err):
			default:
			}
			return
		}
	}()

	select {
	case <-syncDone:
		return true, &api.TaskRunResultPayload{Detail: "sync done"}, nil
	case err := <-syncError:
		return true, nil, err
	}
}

func executeGhost(task *api.Task, statement string, syncDone chan<- struct{}) error {
	instance := task.Instance
	databaseName := task.Database.Name

	tableName, err := getTableNameFromStatement(statement)
	if err != nil {
		return err
	}

	adminDataSource := api.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return common.Errorf(common.Internal, "admin data source not found for instance %d", instance.ID)
	}

	migrationContext, err := newMigrationContext(ghostConfig{
		host:                 instance.Host,
		port:                 instance.Port,
		user:                 adminDataSource.Username,
		password:             adminDataSource.Password,
		database:             databaseName,
		table:                tableName,
		alterStatement:       statement,
		socketFilename:       getSocketFilename(task.ID, task.Database.ID, databaseName, tableName),
		postponeFlagFilename: getPostponeFlagFilename(task.ID, task.Database.ID, databaseName, tableName),
		noop:                 false,
		// On the source and each replica, you must set the server_id system variable to establish a unique replication ID. For each server, you should pick a unique positive integer in the range from 1 to 2^32 − 1, and each ID must be different from every other ID in use by any other source or replica in the replication topology. Example: server-id=3.
		// https://dev.mysql.com/doc/refman/5.7/en/replication-options-source.html
		// Here we use serverID = offset + task.ID to avoid potential conflicts.
		serverID: 10000000 + uint(task.ID),
	})
	if err != nil {
		return fmt.Errorf("failed to init migrationContext for gh-ost, error: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	migrator := logic.NewMigrator(migrationContext)

	go func(ctx context.Context, migrationContext *base.MigrationContext) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Since we are using postpone flag file to postpone cutover, it's gh-ost mechanism to set migrationContext.IsPostponingCutOver to 1 after synced and before postpone flag file is removed. We utilize this mechanism here to check if synced.
				if atomic.LoadInt64(&migrationContext.IsPostponingCutOver) > 0 {
					close(syncDone)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, migrationContext)

	if err = migrator.Migrate(); err != nil {
		return fmt.Errorf("failed to run gh-ost, error: %w", err)
	}
	return nil
}
