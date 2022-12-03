package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
)

// NewTaskCheckMigrationSchemaExecutor creates a task check migration schema executor.
func NewTaskCheckMigrationSchemaExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) TaskCheckExecutor {
	return &TaskCheckMigrationSchemaExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// TaskCheckMigrationSchemaExecutor is the task check migration schema executor.
type TaskCheckMigrationSchemaExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run will run the task check migration schema executor once.
func (e *TaskCheckMigrationSchemaExecutor) Run(ctx context.Context, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
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
				Title:     "Error",
				Content:   fmt.Sprintf("task not found for ID %v", taskCheckRun.TaskID),
			},
		}, nil
	}

	instance, err := e.store.GetInstanceByID(ctx, task.InstanceID)
	if err != nil {
		return []api.TaskCheckResult{}, err
	}

	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
	if err != nil {
		return []api.TaskCheckResult{}, err
	}
	defer driver.Close(ctx)

	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return []api.TaskCheckResult{}, err
	}

	if setup {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.MigrationSchemaMissing.Int(),
				Title:     "Error",
				Content:   fmt.Sprintf("Missing migration schema for instance %q", instance.Name),
			},
		}, nil
	}

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   fmt.Sprintf("Instance %q has setup migration schema", instance.Name),
		},
	}, nil
}
