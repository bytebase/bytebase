package taskcheck

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/api"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/server/component/dbfactory"
	"github.com/bytebase/bytebase/backend/store"
)

// NewMigrationSchemaExecutor creates a task check migration schema executor.
func NewMigrationSchemaExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &MigrationSchemaExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// MigrationSchemaExecutor is the task check migration schema executor.
type MigrationSchemaExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run will run the task check migration schema executor once.
func (e *MigrationSchemaExecutor) Run(ctx context.Context, _ *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error) {
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
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
				Content:   fmt.Sprintf("Missing migration schema for instance %q", instance.Title),
			},
		}, nil
	}

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   fmt.Sprintf("Instance %q has setup migration schema", instance.Title),
		},
	}, nil
}
