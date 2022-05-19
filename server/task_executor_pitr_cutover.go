package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
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

	issue, err := getIssueByPipelineID(ctx, server.store, exec.l, task.PipelineID)
	if err != nil {
		return true, nil, err
	}

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		exec.l.Error("failed to cast driver to mysql.Driver", zap.Stack("stack"))
		return true, nil, fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}
	mysqlRestore := restoremysql.New(mysqlDriver)

	pitrDatabaseName := restoremysql.GetPITRDatabaseName(task.Database.Name, issue.CreatedTs)
	pitrOldDatabaseName := restoremysql.GetPITROldDatabaseName(task.Database.Name, issue.CreatedTs)
	exec.l.Info("Start swapping the original and PITR database",
		zap.String("instance", task.Instance.Name),
		zap.String("original_database", task.Database.Name),
		zap.String("pitr_database", pitrDatabaseName),
	)
	if err := mysqlRestore.SwapPITRDatabase(ctx, task.Database.Name, issue.CreatedTs); err != nil {
		exec.l.Error("failed to swap the original and PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("database", task.Database.Name),
			zap.Stack("stack"),
			zap.Error(err))
		return true, nil, fmt.Errorf("failed to swap the original and PITR database, error[%w]", err)
	}

	exec.l.Info("Finish swapping the original and PITR database",
		zap.String("target_database", task.Database.Name),
		zap.String("old_database", pitrOldDatabaseName))
	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Swapped PITR database for target database %q", task.Database.Name),
	}, nil
}
