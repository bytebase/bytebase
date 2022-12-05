package server

import (
	"context"
	"encoding/json"

	"github.com/github/gh-ost/go/logic"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/store"
)

// NewTaskCheckGhostSyncExecutor creates a task check gh-ost sync executor.
func NewTaskCheckGhostSyncExecutor(store *store.Store) TaskCheckExecutor {
	return &TaskCheckGhostSyncExecutor{
		store: store,
	}
}

// TaskCheckGhostSyncExecutor is the task check gh-ost sync executor.
type TaskCheckGhostSyncExecutor struct {
	store *store.Store
}

// Run will run the task check database connector executor once.
func (e *TaskCheckGhostSyncExecutor) Run(ctx context.Context, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	// gh-ost dry run could panic.
	// It may be bytebase who panicked, but that's rare. So
	// capture the error and send it into the result list.
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
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
	task, err := e.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return nil, common.Wrap(err, common.Internal)
	}
	if task == nil {
		return nil, common.Errorf(common.Internal, "failed to find task %d", taskCheckRun.TaskID)
	}

	if task.Instance == nil {
		return nil, common.Errorf(common.Internal, "failed to find instance %d", task.InstanceID)
	}

	if task.Database == nil {
		return nil, common.Errorf(common.Internal, "failed to find database %d", task.DatabaseID)
	}

	adminDataSource := api.DataSourceFromInstanceWithType(task.Instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %d", task.InstanceID)
	}

	instanceUserList, err := e.store.FindInstanceUserByInstanceID(ctx, task.InstanceID)
	if err != nil {
		return nil, common.Errorf(common.Internal, "failed to find instance user by instanceID %d", task.InstanceID)
	}

	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, common.Wrapf(err, common.Internal, "invalid database schema update gh-ost sync payload")
	}

	tableName, err := getTableNameFromStatement(payload.Statement)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to parse table name from statement, statement: %v", payload.Statement)
	}

	config := getGhostConfig(task, adminDataSource, instanceUserList, tableName, payload.Statement, true, 20000000)

	migrationContext, err := newMigrationContext(config)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create migration context")
	}

	migrator := logic.NewMigrator(migrationContext, "bb")

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
