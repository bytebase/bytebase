package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
)

// NewTaskCheckDatabaseConnectExecutor creates a task check database connect executor.
func NewTaskCheckDatabaseConnectExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) TaskCheckExecutor {
	return &TaskCheckDatabaseConnectExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// TaskCheckDatabaseConnectExecutor is the task check database connect executor.
type TaskCheckDatabaseConnectExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run will run the task check database connector executor once.
func (e *TaskCheckDatabaseConnectExecutor) Run(ctx context.Context, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	task, err := e.store.GetTaskByID(ctx, taskCheckRun.TaskID)
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

	database, err := e.store.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
	if err != nil {
		return []api.TaskCheckResult{}, common.Wrap(err, common.Internal)
	}
	if database == nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, "database ID not found %v", task.DatabaseID)
	}

	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, database.Instance, database.Name)
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
