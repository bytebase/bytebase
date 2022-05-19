package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
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
	exec.l.Info("Run PITR restore task", zap.String("task", task.Name))

	payload := api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return true, nil, fmt.Errorf("invalid PITR restore payload[%s], error[%w]", task.Payload, err)
	}

	return exec.pitrRestore(ctx, task, server)
}

// TODO(dragonly): Should establish a BASELINE migration in the swap database task.
// And what's the right schema version in tenant mode?
func (exec *PITRRestoreTaskExecutor) pitrRestore(ctx context.Context, task *api.Task, server *Server) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", exec.l)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	if err := exec.doPITRRestore(ctx, task, server.store, driver); err != nil {
		return true, nil, err
	}

	exec.l.Info("created PITR database", zap.String("target database", task.Database.Name))

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Created PITR database for target database %q", task.Database.Name),
	}, nil
}

func (exec *PITRRestoreTaskExecutor) doPITRRestore(ctx context.Context, task *api.Task, store *store.Store, driver db.Driver) error {
	instance := task.Instance
	database := task.Database

	issue, err := getIssueByPipelineID(ctx, store, exec.l, task.PipelineID)
	if err != nil {
		return err
	}

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		exec.l.Error("failed to cast driver to mysql.Driver", zap.Stack("stack"))
		return fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	mysqlRestore := restoremysql.New(mysqlDriver)
	config := pluginmysql.BinlogInfo{}
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

func getIssueByPipelineID(ctx context.Context, store *store.Store, l *zap.Logger, pid int) (*api.Issue, error) {
	issue, err := store.GetIssueByPipelineID(ctx, pid)
	if err != nil {
		l.Error("failed to get issue by PipelineID", zap.Int("PipelineID", pid), zap.Error(err))
		return nil, fmt.Errorf("failed to get issue by PipelineID[%d], error[%w]", pid, err)
	}
	if issue == nil {
		l.Error("issue not found with PipelineID", zap.Int("PipelineID", pid))
		return nil, fmt.Errorf("issue not found with PipelineID[%d]", pid)
	}
	return issue, nil
}
