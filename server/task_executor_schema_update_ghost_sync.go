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
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/store"
)

// NewSchemaUpdateGhostSyncTaskExecutor creates a schema update (gh-ost) sync task executor.
func NewSchemaUpdateGhostSyncTaskExecutor(store *store.Store, taskScheduler *TaskScheduler) TaskExecutor {
	return &SchemaUpdateGhostSyncTaskExecutor{
		store:         store,
		taskScheduler: taskScheduler,
	}
}

// SchemaUpdateGhostSyncTaskExecutor is the schema update (gh-ost) sync task executor.
type SchemaUpdateGhostSyncTaskExecutor struct {
	store         *store.Store
	taskScheduler *TaskScheduler
}

// RunOnce will run SchemaUpdateGhostSync task once.
func (exec *SchemaUpdateGhostSyncTaskExecutor) RunOnce(ctx context.Context, _ *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}
	return exec.runGhostMigration(ctx, exec.store, exec.taskScheduler, task, payload.Statement)
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
		return "", errors.Errorf("failed to parse table name from statement, statement: %v", statement)
	}
	return parser.GetExplicitTable(), nil
}

type sharedGhostState struct {
	migrationContext *base.MigrationContext
	errCh            <-chan error
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

	// vendor related
	isAWS bool
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
		throttleHTTPIntervalMillis          = 100
		throttleHTTPTimeoutMillis           = 1000
	)
	statement := strings.Join(strings.Fields(config.alterStatement), " ")
	migrationContext := base.NewMigrationContext()
	migrationContext.InspectorConnectionConfig.Key.Hostname = config.host
	port := 3306
	if config.port != "" {
		configPort, err := strconv.Atoi(config.port)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert port from string to int")
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
	if config.isAWS {
		migrationContext.AssumeRBR = true
	}
	// set defaults
	migrationContext.AllowedRunningOnMaster = allowedRunningOnMaster
	migrationContext.ConcurrentCountTableRows = concurrentCountTableRows
	migrationContext.HooksStatusIntervalSec = hooksStatusIntervalSec
	migrationContext.CutOverType = base.CutOverAtomic
	migrationContext.ThrottleHTTPIntervalMillis = throttleHTTPIntervalMillis
	migrationContext.ThrottleHTTPTimeoutMillis = throttleHTTPTimeoutMillis

	if migrationContext.AlterStatement == "" {
		return nil, errors.Errorf("alterStatement must be provided and must not be empty")
	}
	parser := ghostsql.NewParserFromAlterStatement(migrationContext.AlterStatement)
	migrationContext.AlterStatementOptions = parser.GetAlterStatementOptions()

	if migrationContext.DatabaseName == "" {
		if !parser.HasExplicitSchema() {
			return nil, errors.Errorf("database must be provided and database name must not be empty, or alterStatement must specify database name")
		}
		migrationContext.DatabaseName = parser.GetExplicitSchema()
	}
	if migrationContext.OriginalTableName == "" {
		if !parser.HasExplicitTable() {
			return nil, errors.Errorf("table must be provided and table name must not be empty, or alterStatement must specify table name")
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

func getGhostConfig(task *api.Task, dataSource *api.DataSource, userList []*api.InstanceUser, tableName string, statement string, noop bool, serverIDOffset uint) ghostConfig {
	var isAWS bool
	for _, user := range userList {
		if user.Name == "'rdsadmin'@'localhost'" && strings.Contains(user.Grant, "SUPER") {
			isAWS = true
			break
		}
	}
	return ghostConfig{
		host:                 task.Instance.Host,
		port:                 task.Instance.Port,
		user:                 dataSource.Username,
		password:             dataSource.Password,
		database:             task.Database.Name,
		table:                tableName,
		alterStatement:       statement,
		socketFilename:       getSocketFilename(task.ID, task.Database.ID, task.Database.Name, tableName),
		postponeFlagFilename: getPostponeFlagFilename(task.ID, task.Database.ID, task.Database.Name, tableName),
		noop:                 noop,
		// On the source and each replica, you must set the server_id system variable to establish a unique replication ID. For each server, you should pick a unique positive integer in the range from 1 to 2^32 − 1, and each ID must be different from every other ID in use by any other source or replica in the replication topology. Example: server-id=3.
		// https://dev.mysql.com/doc/refman/5.7/en/replication-options-source.html
		// Here we use serverID = offset + task.ID to avoid potential conflicts.
		serverID: serverIDOffset + uint(task.ID),
		// https://github.com/github/gh-ost/blob/master/doc/rds.md
		isAWS: isAWS,
	}
}

func (*SchemaUpdateGhostSyncTaskExecutor) runGhostMigration(ctx context.Context, store *store.Store, taskScheduler *TaskScheduler, task *api.Task, statement string) (terminated bool, result *api.TaskRunResultPayload, err error) {
	syncDone := make(chan struct{})
	// set buffer size to 1 to unblock the sender because there is no listner if the task is canceled.
	// see PR #2919.
	migrationError := make(chan error, 1)

	statement = strings.TrimSpace(statement)

	tableName, err := getTableNameFromStatement(statement)
	if err != nil {
		return true, nil, err
	}

	adminDataSource := api.DataSourceFromInstanceWithType(task.Instance, api.Admin)
	if adminDataSource == nil {
		return true, nil, common.Errorf(common.Internal, "admin data source not found for instance %d", task.Instance.ID)
	}

	instanceUserList, err := store.FindInstanceUserByInstanceID(ctx, task.InstanceID)
	if err != nil {
		return true, nil, common.Errorf(common.Internal, "failed to find instance user by instanceID %d", task.InstanceID)
	}

	config := getGhostConfig(task, adminDataSource, instanceUserList, tableName, statement, false, 10000000)

	migrationContext, err := newMigrationContext(config)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to init migrationContext for gh-ost")
	}

	migrator := logic.NewMigrator(migrationContext, "bb")

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func(childCtx context.Context) {
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()
		createdTs := time.Now().Unix()
		for {
			select {
			case <-ticker.C:
				var (
					totalUnit     = atomic.LoadInt64(&migrationContext.RowsEstimate) + atomic.LoadInt64(&migrationContext.RowsDeltaEstimate)
					completedUnit = migrationContext.GetTotalRowsCopied()
					updatedTs     = time.Now().Unix()
				)
				taskScheduler.taskProgress.Store(task.ID, api.Progress{
					TotalUnit:     totalUnit,
					CompletedUnit: completedUnit,
					CreatedTs:     createdTs,
					UpdatedTs:     updatedTs,
				})
				// Since we are using postpone flag file to postpone cutover, it's gh-ost mechanism to set migrationContext.IsPostponingCutOver to 1 after synced and before postpone flag file is removed. We utilize this mechanism here to check if synced.
				if atomic.LoadInt64(&migrationContext.IsPostponingCutOver) > 0 {
					close(syncDone)
					return
				}
			case <-childCtx.Done():
				return
			}
		}
	}(childCtx)

	go func() {
		if err := migrator.Migrate(); err != nil {
			log.Error("failed to run gh-ost migration", zap.Error(err))
			migrationError <- err
			return
		}
		migrationError <- nil
		// we send to migrationError channel anyway because:
		// 1. before syncDone, the gh-ost sync task will receive it.
		// 2. after syncDone, the gh-ost cutover task will receive it.
	}()

	select {
	case <-syncDone:
		taskScheduler.sharedTaskState.Store(task.ID, sharedGhostState{migrationContext: migrationContext, errCh: migrationError})
		return true, &api.TaskRunResultPayload{Detail: "sync done"}, nil
	case err := <-migrationError:
		return true, nil, err
	case <-ctx.Done():
		migrationContext.PanicAbort <- errors.New("task canceled")
		return true, nil, errors.New("task canceled")
	}
}
