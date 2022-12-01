package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// NewTaskCheckDatabaseConnectExecutor creates a task check database connect executor.
func NewTaskCheckDatabaseConnectExecutor() TaskCheckExecutor {
	return &TaskCheckDatabaseConnectExecutor{}
}

// TaskCheckDatabaseConnectExecutor is the task check database connect executor.
type TaskCheckDatabaseConnectExecutor struct {
}

// Run will run the task check database connector executor once.
func (*TaskCheckDatabaseConnectExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return []api.TaskCheckResult{}, common.Wrap(err, common.Internal)
	}
	if task == nil {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.Internal.Int(),
				Title:     fmt.Sprintf("Failed to find task %v", taskCheckRun.TaskID),
				Content:   err.Error(),
			},
		}, nil
	}

	database, err := server.store.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
	if err != nil {
		return []api.TaskCheckResult{}, common.Wrap(err, common.Internal)
	}
	if database == nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, "database ID not found %v", task.DatabaseID)
	}

	driver, err := server.dbFactory.GetAdminDatabaseDriver(ctx, database.Instance, database.Name)
	if err != nil {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.DbConnectionFailure.Int(),
				Title:     fmt.Sprintf("Failed to connect %q", database.Name),
				Content:   err.Error(),
			},
		}, nil
	}
	defer driver.Close(ctx)

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   fmt.Sprintf("Successfully connected %q", database.Name),
		},
	}, nil
}
