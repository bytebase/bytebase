package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	"go.uber.org/zap"
)

// NewPITRDeleteTaskExecutor creates a PITR delete task executor.
func NewPITRDeleteTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &PITRDeleteTaskExecutor{
		l: logger,
	}
}

// PITRDeleteTaskExecutor is the PITR delete task executor.
type PITRDeleteTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run the PITR delete task executor once.
func (exec *PITRDeleteTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	exec.l.Info("Run PITR delete task", zap.String("task", task.Name))

	// api.TaskDatabasePITRDeletePayload is empty, do not need to unmarshal json

	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", exec.l)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)
	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		exec.l.Error("failed to cast driver to mysql.Driver", zap.Stack("stack"))
		return true, nil, fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	issue, err := server.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil || issue == nil {
		exec.l.Error("failed to get issue by PipelineID",
			zap.Int("PipelineID", task.PipelineID),
			zap.Error(err))
		return true, nil, fmt.Errorf("failed to get issue by PipelineID[%d], error[%w]", task.PipelineID, err)
	}

	mysqlRestore := restoremysql.New(mysqlDriver)
	if err := mysqlRestore.DeletePITRDatabases(ctx, task.Database.Name, issue.CreatedTs); err != nil {
		exec.l.Error("failed to delete the original database after PITR swap",
			zap.String("instance", task.Instance.Name),
			zap.String("original database", task.Database.Name))
		return true, nil, fmt.Errorf("failed to delete the original database after PITR swap, error[%w]", err)
	}

	exec.l.Info("delete original PITR database", zap.String("target database", task.Database.Name))
	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Delete original PITR database for target database %q", task.Database.Name),
	}, nil
}
