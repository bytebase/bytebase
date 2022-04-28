package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

// NewTaskCheckMigrationSchemaExecutor creates a task check migration schema executor.
func NewTaskCheckMigrationSchemaExecutor(logger *zap.Logger) TaskCheckExecutor {
	return &TaskCheckMigrationSchemaExecutor{
		l: logger,
	}
}

// TaskCheckMigrationSchemaExecutor is the task check migration schema executor.
type TaskCheckMigrationSchemaExecutor struct {
	l *zap.Logger
}

// Run will run the task check migration schema executor once.
func (exec *TaskCheckMigrationSchemaExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err)
	}
	if task == nil {
		return []api.TaskCheckResult{
			{
				Status:  api.TaskCheckStatusError,
				Code:    common.Internal,
				Title:   "Error",
				Content: fmt.Sprintf("task not found for ID %v", taskCheckRun.TaskID),
			},
		}, nil
	}

	instance, err := server.store.GetInstanceByID(ctx, task.InstanceID)
	if err != nil {
		return []api.TaskCheckResult{}, err
	}

	driver, err := getAdminDatabaseDriver(ctx, instance, "", exec.l)
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
				Status:  api.TaskCheckStatusError,
				Code:    common.MigrationSchemaMissing,
				Title:   "Error",
				Content: fmt.Sprintf("Missing migration schema for instance %q", instance.Name),
			},
		}, nil
	}

	return []api.TaskCheckResult{
		{
			Status:  api.TaskCheckStatusSuccess,
			Code:    common.Ok,
			Title:   "OK",
			Content: fmt.Sprintf("Instance %q has setup migration schema", instance.Name),
		},
	}, nil
}
