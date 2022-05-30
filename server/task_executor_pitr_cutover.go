package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"go.uber.org/zap"
)

// NewPITRCutoverTaskExecutor creates a PITR cutover task executor.
func NewPITRCutoverTaskExecutor(instance *mysqlutil.Instance) TaskExecutor {
	return &PITRCutoverTaskExecutor{

		mysqlutil: instance,
	}
}

// PITRCutoverTaskExecutor is the PITR cutover task executor.
type PITRCutoverTaskExecutor struct {
	mysqlutil *mysqlutil.Instance
}

// RunOnce will run the PITR cutover task executor once.
func (exec *PITRCutoverTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR cutover task", zap.String("task", task.Name))

	// Currently api.TaskDatabasePITRCutoverPayload is empty, so we do not need to unmarshal from task.Payload.

	return exec.pitrCutover(ctx, task, server)
}

func (exec *PITRCutoverTaskExecutor) pitrCutover(ctx context.Context, task *api.Task, server *Server) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "")
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	issue, err := getIssueByPipelineID(ctx, server.store, task.PipelineID)
	if err != nil {
		return true, nil, err
	}

	connCfg, err := getConnectionConfig(ctx, task.Instance, task.Database.Name)
	if err != nil {
		return true, nil, err
	}

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		log.Error("failed to cast driver to mysql.Driver", zap.Stack("stack"))
		return true, nil, fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}
	mysqlRestore := restoremysql.New(mysqlDriver, exec.mysqlutil, connCfg)

	log.Info("Start swapping the original and PITR database",
		zap.String("instance", task.Instance.Name),
		zap.String("original_database", task.Database.Name),
	)
	pitrDatabaseName, pitrOldDatabaseName, err := mysqlRestore.SwapPITRDatabase(ctx, task.Database.Name, issue.CreatedTs)
	if err != nil {
		log.Error("failed to swap the original and PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("database", task.Database.Name),
			zap.Stack("stack"),
			zap.Error(err))
		return true, nil, fmt.Errorf("failed to swap the original and PITR database, error[%w]", err)
	}

	log.Info("Finish swapping the original and PITR database",
		zap.String("original_database", task.Database.Name),
		zap.String("pitr_database", pitrDatabaseName),
		zap.String("old_database", pitrOldDatabaseName))
	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Swapped PITR database for target database %q", task.Database.Name),
	}, nil
}
