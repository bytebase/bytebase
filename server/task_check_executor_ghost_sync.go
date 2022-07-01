package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/github/gh-ost/go/logic"
)

// NewTaskCheckGhostSyncExecutor creates a task check gh-ost sync executor.
func NewTaskCheckGhostSyncExecutor() TaskCheckExecutor {
	return &TaskCheckGhostSyncExecutor{}
}

// TaskCheckGhostSyncExecutor is the task check gh-ost sync executor.
type TaskCheckGhostSyncExecutor struct {
}

// Run will run the task check database connector executor once.
func (exec *TaskCheckGhostSyncExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	// gh-ost dry run could panic.
	// It may be bytebase who panicked, but that's rare. So
	// capture the error and send it into the result list.
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			result = []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.Internal.Int(),
					Title:     "gh-ost dry run failed",
					Content:   panicErr.Error(),
				},
			}
			err = nil
		}
	}()
	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err)
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

	instance := task.Instance
	if instance == nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("instance ID not found %v", task.InstanceID))
	}

	database := task.Database
	if database == nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("database ID not found %v", task.DatabaseID))
	}

	adminDataSource := api.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("admin data source not found for instance %d", instance.ID))
	}

	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("invalid database schema update gh-ost sync payload: %w", err))
	}

	databaseName := database.Name
	tableName, err := getTableNameFromStatement(payload.Statement)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("failed to parse table name from statement, statement: %v, error: %w", payload.Statement, err))
	}

	migrationContext, err := newMigrationContext(ghostConfig{
		host:                 instance.Host,
		port:                 instance.Port,
		user:                 adminDataSource.Username,
		password:             adminDataSource.Password,
		database:             databaseName,
		table:                tableName,
		alterStatement:       payload.Statement,
		socketFilename:       getSocketFilename(taskCheckRun.ID, task.Database.ID, databaseName, tableName),
		postponeFlagFilename: "",
		noop:                 true,
		// On the source and each replica, you must set the server_id system variable to establish a unique replication ID. For each server, you should pick a unique positive integer in the range from 1 to 2^32 − 1, and each ID must be different from every other ID in use by any other source or replica in the replication topology. Example: server-id=3.
		// https://dev.mysql.com/doc/refman/5.7/en/replication-options-source.html
		// Here we use serverID = offset + task.ID to avoid potential conflicts.
		serverID: 20000000 + uint(taskCheckRun.ID),
	})
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("failed to create migration context, error: %w", err))
	}

	migrator := logic.NewMigrator(migrationContext)

	if err := migrator.Migrate(); err != nil {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.Internal.Int(),
				Title:     "gh-ost dry run failed",
				Content:   err.Error(),
			},
		}, nil
	}

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   "gh-ost dry run succeeded",
		},
	}, nil
}
