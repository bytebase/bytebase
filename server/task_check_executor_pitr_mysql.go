package server

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db/mysql"
)

// NewTaskCheckPITRMySQLExecutor creates a task check migration schema executor.
func NewTaskCheckPITRMySQLExecutor() TaskCheckExecutor {
	return &TaskCheckPITRMySQLExecutor{}
}

// TaskCheckPITRMySQLExecutor is the task check migration schema executor.
type TaskCheckPITRMySQLExecutor struct {
}

// Run will run the task check migration schema executor once.
func (*TaskCheckPITRMySQLExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get task by ID %d", taskCheckRun.TaskID)
	}
	if task == nil {
		return nil, errors.Wrapf(err, "task with ID %d not found", taskCheckRun.TaskID)
	}

	payload := api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return nil, errors.Wrapf(err, "invalid PITR restore payload: %s", task.Payload)
	}

	if payload.BackupID != nil {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "OK",
				Content:   "Ready to do backup restore",
			},
		}, nil
	}

	instanceID := task.InstanceID
	databaseName := task.Database.Name
	if payload.TargetInstanceID != nil {
		instanceID = *payload.TargetInstanceID
		databaseName = *payload.DatabaseName
	}

	instance, err := server.store.GetInstanceByID(ctx, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by ID %d", instanceID)
	}

	driver, err := server.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		return nil, errors.Errorf("Failed to cast driver to mysql.Driver")
	}

	if err := mysqlDriver.CheckServerVersionForPITR(ctx); err != nil {
		return wrapTaskCheckError(err), nil
	}

	if err := mysqlDriver.CheckEngineInnoDB(ctx, databaseName); err != nil {
		return wrapTaskCheckError(err), nil
	}

	if err := mysqlDriver.CheckBinlogRowFormat(ctx); err != nil {
		return wrapTaskCheckError(err), nil
	}

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   "Ready to do PITR",
		},
	}, nil
}

func wrapTaskCheckError(err error) []api.TaskCheckResult {
	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusError,
			Namespace: api.BBNamespace,
			Code:      common.Internal.Int(),
			Title:     "Error",
			Content:   err.Error(),
		},
	}
}
