package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/plugin/db/util"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// NewPITRRestoreTaskExecutor creates a PITR restore task executor.
func NewPITRRestoreTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &PITRRestoreTaskExecutor{
		l: logger,
	}
}

// PITRRestoreTaskExecutor is the PITR restore task executor.
type PITRRestoreTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run the PITR restore task executor once.
func (exec *PITRRestoreTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("PITRRestoreTaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("stack"))
			terminated = true
			err = fmt.Errorf("encounter internal error when executing sql")
		}
	}()
	exec.l.Info("Run PITR restore task", zap.String("task", task.Name))

	payload := api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return true, nil, fmt.Errorf("invalid PITR restore payload[%s], error[%w]", task.Payload, err)
	}

	return exec.pitrRestore(ctx, task, server)
}

func (exec *PITRRestoreTaskExecutor) pitrRestore(ctx context.Context, task *api.Task, server *Server) (terminated bool, result *api.TaskRunResultPayload, err error) {
	schemaVersion := common.DefaultMigrationVersion()
	migrationInfo, err := preMigration(ctx, exec.l, server, task, db.Baseline, "", schemaVersion, nil)
	if err != nil {
		exec.l.Error("failed in preMigration stage", zap.Error(err))
		return true, nil, fmt.Errorf("failed in preMigration stage, error[%w]", err)
	}

	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", exec.l)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	needsSetup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return true, nil, fmt.Errorf("failed to check migration setup for instance %q, error[%w]", task.Instance.Name, err)
	}
	if needsSetup {
		return true, nil, common.Errorf(common.MigrationSchemaMissing, fmt.Errorf("missing migration schema for instance %q", task.Instance.Name))
	}

	executor := driver.(util.MigrationExecutor)
	var prevSchemaBuf bytes.Buffer
	if err := driver.Dump(ctx, migrationInfo.Database, &prevSchemaBuf, true); err != nil {
		return true, nil, err
	}

	// Insert a pending baseline migration record into the migration_history table.
	// The whole PITR process can be regarded as a data change followed by a schema sync.
	migrationHistoryID, err := util.BeginMigration(ctx, executor, migrationInfo, prevSchemaBuf.String(), "", migrationInfo.Database)
	if err != nil {
		return true, nil, err
	}
	startedNs := time.Now().UnixNano()

	var updatedSchemaBuf bytes.Buffer
	if err := executor.Dump(ctx, migrationInfo.Database, &updatedSchemaBuf, true /*schemaOnly*/); err != nil {
		return true, nil, util.FormatError(err)
	}
	updatedSchema := updatedSchemaBuf.String()

	if err := exec.doPITRRestore(ctx, task, server.store, driver); err != nil {
		return true, nil, err
	}

	// TODO(dragonly): Fix this using shared context between tasks.
	if err := util.EndMigration(ctx, exec.l, executor, startedNs, migrationHistoryID, updatedSchema, db.BytebaseDatabase, true /*isDone*/); err != nil {
		exec.l.Error("failed to update migration history record",
			zap.Int64("migration_id", migrationHistoryID),
			zap.Error(err),
		)
	}

	exec.l.Info("created PITR database", zap.String("target database", migrationInfo.Database))

	return true, &api.TaskRunResultPayload{
		Detail:      fmt.Sprintf("Created PITR database for target database %q", migrationInfo.Database),
		MigrationID: migrationHistoryID,
		Version:     migrationInfo.Version,
	}, nil
}

func (exec *PITRRestoreTaskExecutor) doPITRRestore(ctx context.Context, task *api.Task, store *store.Store, driver db.Driver) error {
	instance := task.Instance
	database := task.Database

	issue, err := store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil || issue == nil {
		exec.l.Error("failed to get issue by PipelineID",
			zap.Int("PipelineID", task.PipelineID),
			zap.Any("issue", *issue),
			zap.Error(err))
		return fmt.Errorf("failed to get issue by PipelineID[%d], error[%w]", task.PipelineID, err)
	}

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		exec.l.Error("failed to cast driver to mysql.Driver", zap.Stack("stack"))
		return fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	mysqlRestore := restoremysql.New(mysqlDriver)
	config := restoremysql.BinlogConfig{}
	// TODO(dragonly): Search and put the file io of the logical backup file here.
	// Currently, let's just use the empty backup dump as a placeholder.
	var buf bytes.Buffer
	buf.WriteString("-- This is a fake backup dump")

	exec.l.Debug("Start creating and restoring PITR database",
		zap.String("instance", instance.Name),
		zap.String("database", database.Name),
	)

	// RestorePITR will create the pitr database.
	// Since it's ephemeral and will be renamed to the original database soon, we will reuse the original
	// database's migration history, and append a new BASELINE migration.
	if err := mysqlRestore.RestorePITR(ctx, bufio.NewScanner(&buf), config, database.Name, issue.CreatedTs); err != nil {
		exec.l.Error("failed to perform a PITR restore in the PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("database", database.Name),
			zap.Stack("stack"),
			zap.Error(err))
		return fmt.Errorf("failed to perform a PITR restore in the PITR database, error[%w]", err)
	}

	return nil
}
