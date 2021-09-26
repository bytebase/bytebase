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
		ID: &taskCheckRun.TaskId,
	}
	task, err := server.TaskService.FindTask(ctx, taskFind)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err.Error())
	}

	databaseFind := &api.DatabaseFind{
		ID: task.DatabaseId,
	}
	database, err := server.ComposeDatabaseByFind(ctx, databaseFind)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err.Error())
	}

	driver, err := GetDatabaseDriver(database.Instance, database.Name, exec.l)
	if err != nil {
		return []api.TaskCheckResult{}, err
	}
	defer driver.Close(ctx)

	if err := driver.Ping(ctx); err != nil {
		return []api.TaskCheckResult{
			{
				Status:  api.TaskCheckStatusError,
				Code:    common.DbConnectionFailure,
				Title:   fmt.Sprintf("failed to connect %q", database.Name),
				Content: err.Error(),
			},
		}, nil
	}

	return []api.TaskCheckResult{
		{
			Status:  api.TaskCheckStatusSuccess,
			Code:    common.Ok,
			Title:   fmt.Sprintf("Successfully connected %q", database.Name),
			Content: "",
		},
	}, nil
}
