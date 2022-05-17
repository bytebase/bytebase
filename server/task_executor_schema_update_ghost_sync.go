package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

type ghostConfig struct {
	host           string
	port           string
	user           string
	password       string
	database       string
	table          string
	alterStatement string
	noop           bool
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
	// TODO(p0ny): change file name according to design doc.
	migrationContext.ServeSocketFile = fmt.Sprintf("/tmp/gh-ost.%s.%s.sock", migrationContext.DatabaseName, migrationContext.OriginalTableName)
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

// RunOnce will run SchemaUpdateGhostSync task once.
func (exec *SchemaUpdateGhostSyncTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("SchemaUpdateGhostSyncTaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("stack"))
			terminated = true
			err = fmt.Errorf("encounter internal error when executing migration using gh-ost")
		}
	}()

	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update gh-ost sync payload: %w", err)
	}
	return runGhostMigration(ctx, exec.l, server, task, db.Migrate, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}

func runGhostMigration(ctx context.Context, l *zap.Logger, server *Server, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := preMigration(ctx, l, server, task, migrationType, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := executeSync(ctx, l, task, mi, statement)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, l, server, task, vcsPushEvent, mi, migrationID, schema)
}

func executeSync(ctx context.Context, l *zap.Logger, task *api.Task, mi *db.MigrationInfo, statement string) (migrationHistoryID int64, updatedSchema string, resErr error) {
	statement = strings.TrimSpace(statement)
	databaseName := db.BytebaseDatabase

	driver, err := getAdminDatabaseDriver(ctx, task.Instance, task.Database.Name, l)
	if err != nil {
		return -1, "", err
	}
	defer driver.Close(ctx)
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return -1, "", fmt.Errorf("failed to check migration setup for instance %q: %w", task.Instance.Name, err)
	}
	if setup {
		return -1, "", common.Errorf(common.MigrationSchemaMissing, fmt.Errorf("missing migration schema for instance %q", task.Instance.Name))
	}

	executor := driver.(util.MigrationExecutor)

	var prevSchemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, mi.Database, &prevSchemaBuf, true); err != nil {
		return -1, "", err
	}

	insertedID, err := util.BeginMigration(ctx, executor, mi, prevSchemaBuf.String(), statement, databaseName)
	if err != nil {
		return -1, "", err
	}
	startedNs := time.Now().UnixNano()

	defer func() {
		if err := util.EndMigration(ctx, l, executor, startedNs, insertedID, updatedSchema, databaseName, resErr == nil /*isDone*/); err != nil {
			l.Error("failed to update migration history record",
				zap.Error(err),
				zap.Int64("migration_id", migrationHistoryID),
			)
		}
	}()

	err = executeGhost(task.Instance, task.Database.Name, statement)
	if err != nil {
		return -1, "", err
	}

	var afterSchemaBuf bytes.Buffer
	if _, err := executor.Dump(ctx, mi.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		return -1, "", util.FormatError(err)
	}

	return insertedID, afterSchemaBuf.String(), nil
}

func executeGhost(instance *api.Instance, databaseName string, statement string) error {
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
		alterStatement: statement,
		noop:           false,
	})
	if err != nil {
		return fmt.Errorf("failed to init migrationContext for gh-ost, error: %w", err)
	}

	migrator := logic.NewMigrator(migrationContext)
	if err = migrator.Migrate(); err != nil {
		return fmt.Errorf("failed to run gh-ost, error: %w", err)
	}
	return nil
}
