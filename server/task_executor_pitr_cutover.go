package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// NewPITRCutoverTaskExecutor creates a PITR cutover task executor.
func NewPITRCutoverTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &PITRCutoverTaskExecutor{
		l: logger,
	}
}

// PITRCutoverTaskExecutor is the PITR cutover task executor.
type PITRCutoverTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run the PITR cutover task executor once.
func (exec *PITRCutoverTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	exec.l.Info("Run PITR cutover task", zap.String("task", task.Name))

	// Currently api.TaskDatabasePITRCutoverPayload is empty, so we do not need to unmarshal from task.Payload.

	return exec.pitrCutover(ctx, task, server)
}

func (exec *PITRCutoverTaskExecutor) pitrCutover(ctx context.Context, task *api.Task, server *Server) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", exec.l)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	if err := exec.doPITRCutover(ctx, task, server.store, driver); err != nil {
		return true, nil, err
	}

	exec.l.Info("swap PITR database done", zap.String("target database", task.Database.Name))

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Swapped PITR database for target database %q", task.Database.Name),
	}, nil
}

func (exec *PITRCutoverTaskExecutor) doPITRCutover(ctx context.Context, task *api.Task, store *store.Store, driver db.Driver) error {
	instance := task.Instance
	database := task.Database

	issue, err := store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil || issue == nil {
		exec.l.Error("failed to get issue by PipelineID",
			zap.Int("PipelineID", task.PipelineID),
			zap.Error(err))
		return fmt.Errorf("failed to get issue by PipelineID[%d], error[%w]", task.PipelineID, err)
	}

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		exec.l.Error("failed to cast driver to mysql.Driver", zap.Stack("stack"))
		return fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	mysqlRestore := restoremysql.New(mysqlDriver)

	exec.l.Debug("Start swapping the original and PITR database",
		zap.String("instance", instance.Name),
		zap.String("database", database.Name),
	)

	if err := mysqlRestore.SwapPITRDatabase(ctx, database.Name, issue.CreatedTs); err != nil {
		exec.l.Error("failed to swap the original and PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("database", database.Name),
			zap.Stack("stack"),
			zap.Error(err))
		return fmt.Errorf("failed to swap the original and PITR database, error[%w]", err)
	}

	return nil
}
