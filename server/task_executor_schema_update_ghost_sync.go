package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/github/gh-ost/go/base"
	"github.com/github/gh-ost/go/logic"
	ghostsql "github.com/github/gh-ost/go/sql"
	"go.uber.org/zap"
)

// NewSchemaUpdateGhostSyncTaskExecutor creates a schema update (gh-ost) sync task executor.
func NewSchemaUpdateGhostSyncTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateGhostSyncTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateGhostSyncTaskExecutor is the schema update (gh-ost) sync task executor.
type SchemaUpdateGhostSyncTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run SchemaUpdateGhostSync task once.
func (exec *SchemaUpdateGhostSyncTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update gh-ost sync payload: %w", err)
	}
	return runGhostMigration(ctx, exec.l, server, task, db.Migrate, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}

func getSocketFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.sock", taskID, databaseID, databaseName, tableName)
}

func getPostponeFlagFilename(taskID int, databaseID int, databaseName string, tableName string) string {
	return fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.postponeFlag", taskID, databaseID, databaseName, tableName)
}

func getTableNameFromStatement(statement string) (string, error) {
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
		hooksStatusIntervalSec              = 60
		replicaServerID                     = 99999
		heartbeatIntervalMilliseconds       = 100
		niceRatio                           = 0
		chunkSize                           = 1000
		dmlBatchSize                        = 10
		maxLagMillisecondsThrottleThreshold = 1500
		defaultNumRetries                   = 60
		cutoverLockTimoutSeconds            = 3
		exponentialBackoffMaxInterval       = 64
	)
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
	migrationContext.AlterStatement = config.alterStatement
	migrationContext.Noop = config.noop
	migrationContext.ReplicaServerId = config.serverID
	// set defaults
	migrationContext.AllowedRunningOnMaster = allowedRunningOnMaster
	migrationContext.ConcurrentCountTableRows = concurrentCountTableRows
	migrationContext.HooksStatusIntervalSec = hooksStatusIntervalSec
	migrationContext.ReplicaServerId = replicaServerID
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
	// TODO(p0ny): set OkToDropTable to false and drop table in dropOriginalTable Task.
	migrationContext.OkToDropTable = true
	migrationContext.SetHeartbeatIntervalMilliseconds(heartbeatIntervalMilliseconds)
	migrationContext.SetNiceRatio(niceRatio)
	migrationContext.SetChunkSize(chunkSize)
	migrationContext.SetDMLBatchSize(dmlBatchSize)
	migrationContext.SetMaxLagMillisecondsThrottleThreshold(maxLagMillisecondsThrottleThreshold)
	migrationContext.SetDefaultNumRetries(defaultNumRetries)
	migrationContext.ApplyCredentials()
	if err := migrationContext.SetCutOverLockTimeoutSeconds(cutoverLockTimoutSeconds); err != nil {
		return nil, err
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(exponentialBackoffMaxInterval); err != nil {
		return nil, err
	}
	return migrationContext, nil
}

func runGhostMigration(ctx context.Context, l *zap.Logger, server *Server, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := preMigration(ctx, l, server, task, migrationType, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}

	waitSync := &sync.WaitGroup{}
	waitSync.Add(1)

	go func(waitSync *sync.WaitGroup) {
		migrationID, schema, err := executeSync(ctx, l, task, mi, statement, waitSync)
		if err != nil {
			l.Error("failed to execute schema update gh-ost sync executeSync", zap.Error(err))
			return
		}
		_, _, err = postMigration(ctx, l, server, task, vcsPushEvent, mi, migrationID, schema)
		if err != nil {
			l.Error("failed to execute schema update gh-ost sync postMigration", zap.Error(err))
		}
	}(waitSync)

	waitSync.Wait()

	return true, &api.TaskRunResultPayload{Detail: "sync done"}, nil
}

func executeSync(ctx context.Context, l *zap.Logger, task *api.Task, mi *db.MigrationInfo, statement string, waitSync *sync.WaitGroup) (migrationHistoryID int64, updatedSchema string, resErr error) {
	statement = strings.TrimSpace(statement)

	driver, err := getAdminDatabaseDriver(ctx, task.Instance, task.Database.Name, l)
	if err != nil {
		return -1, "", err
	}
	defer driver.Close(ctx)
	needsSetup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return -1, "", fmt.Errorf("failed to check migration setup for instance %q: %w", task.Instance.Name, err)
	}
	if needsSetup {
		return -1, "", common.Errorf(common.MigrationSchemaMissing, fmt.Errorf("missing migration schema for instance %q", task.Instance.Name))
	}

	executor := driver.(util.MigrationExecutor)

	var prevSchemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, mi.Database, &prevSchemaBuf, true); err != nil {
		return -1, "", err
	}

	insertedID, err := util.BeginMigration(ctx, executor, mi, prevSchemaBuf.String(), statement, db.BytebaseDatabase)
	if err != nil {
		return -1, "", err
	}
	startedNs := time.Now().UnixNano()

	defer func() {
		if err := util.EndMigration(ctx, l, executor, startedNs, insertedID, updatedSchema, db.BytebaseDatabase, resErr == nil /*isDone*/); err != nil {
			l.Error("failed to update migration history record",
				zap.Error(err),
				zap.Int64("migration_id", migrationHistoryID),
			)
		}
	}()
	if err = executeGhost(l, task, startedNs, statement, waitSync); err != nil {
		return -1, "", err
	}

	var afterSchemaBuf bytes.Buffer
	if _, err := executor.Dump(ctx, mi.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		return -1, "", util.FormatError(err)
	}

	return insertedID, afterSchemaBuf.String(), nil
}

func executeGhost(l *zap.Logger, task *api.Task, startedNs int64, statement string, waitSync *sync.WaitGroup) error {
	instance := task.Instance
	databaseName := task.Database.Name

	tableName, err := getTableNameFromStatement(statement)
	if err != nil {
		return err
	}

	adminDataSource := api.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return common.Errorf(common.Internal, fmt.Errorf("admin data source not found for instance %d", instance.ID))
	}

	migrationContext, err := newMigrationContext(ghostConfig{
		host:           instance.Host,
		port:           instance.Port,
		user:           adminDataSource.Username,
		password:       adminDataSource.Password,
		database:       databaseName,
		table:          tableName,
		alterStatement: statement,
		socketFilename: getSocketFilename(task.ID, task.Database.ID, databaseName, tableName),
		noop:           false,
		// serverID should be unique
		serverID: 10000000 + uint(task.ID),
	})
	if err != nil {
		return fmt.Errorf("failed to init migrationContext for gh-ost, error: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	migrator := logic.NewMigrator(migrationContext)

	go func(ctx context.Context, migrationContext *base.MigrationContext, l *zap.Logger, waitSync *sync.WaitGroup) {
		ticker := time.NewTicker(1 * time.Second)
		defer waitSync.Done()
		for {
			select {
			case <-ticker.C:
				if atomic.LoadInt64(&migrationContext.IsPostponingCutOver) > 0 {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, migrationContext, l, waitSync)

	if err = migrator.Migrate(); err != nil {
		return fmt.Errorf("failed to run gh-ost, error: %w", err)
	}
	return nil
}
