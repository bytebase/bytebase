package taskcheck

import (
	"context"
	"encoding/json"

	"github.com/github/gh-ost/go/logic"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewGhostSyncExecutor creates a task check gh-ost sync executor.
func NewGhostSyncExecutor(store *store.Store) Executor {
	return &GhostSyncExecutor{
		store: store,
	}
}

// GhostSyncExecutor is the task check gh-ost sync executor.
type GhostSyncExecutor struct {
	store *store.Store
}

// Run will run the task check database connector executor once.
func (e *GhostSyncExecutor) Run(ctx context.Context, _ *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error) {
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
	if task.Instance == nil {
		return nil, common.Errorf(common.Internal, "failed to find instance %d", task.InstanceID)
	}
	if task.Database == nil {
		return nil, common.Errorf(common.Internal, "failed to find database %d", task.DatabaseID)
	}
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %d not found", task.InstanceID)
	}

	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %d", task.InstanceID)
	}

	instanceUsers, err := e.store.ListInstanceUsers(ctx, &store.FindInstanceUserMessage{InstanceUID: task.InstanceID})
	if err != nil {
		return nil, common.Errorf(common.Internal, "failed to find instance user by instanceID %d", task.InstanceID)
	}

	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, common.Wrapf(err, common.Internal, "invalid database schema update gh-ost sync payload")
	}

	tableName, err := utils.GetTableNameFromStatement(payload.Statement)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to parse table name from statement, statement: %v", payload.Statement)
	}

	config := utils.GetGhostConfig(task, adminDataSource, instanceUsers, tableName, payload.Statement, true, 20000000)

	migrationContext, err := utils.NewMigrationContext(config)
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
