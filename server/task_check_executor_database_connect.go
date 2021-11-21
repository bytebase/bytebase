package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

func NewTaskCheckDatabaseConnectExecutor(logger *zap.Logger) TaskCheckExecutor {
	return &TaskCheckDatabaseConnectExecutor{
		l: logger,
	}
}

type TaskCheckDatabaseConnectExecutor struct {
	l *zap.Logger
}

func (exec *TaskCheckDatabaseConnectExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	taskFind := &api.TaskFind{
		ID: &taskCheckRun.TaskID,
	}
	task, err := server.TaskService.FindTask(ctx, taskFind)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err)
	}

	databaseFind := &api.DatabaseFind{
		ID: task.DatabaseID,
	}
	database, err := server.ComposeDatabaseByFind(ctx, databaseFind)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err)
	}

	driver, err := GetDatabaseDriver(ctx, database.Instance, database.Name, exec.l)
	if err != nil {
		return []api.TaskCheckResult{
			{
				Status:  api.TaskCheckStatusError,
				Code:    common.DbConnectionFailure,
				Title:   fmt.Sprintf("Failed to connect %q", database.Name),
				Content: err.Error(),
			},
		}, nil
	}
	defer driver.Close(ctx)

	return []api.TaskCheckResult{
		{
			Status:  api.TaskCheckStatusSuccess,
			Code:    common.Ok,
			Title:   "OK",
			Content: fmt.Sprintf("Successfully connected %q", database.Name),
		},
	}, nil
}
