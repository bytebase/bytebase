package taskcheck

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
)

// NewDatabaseConnectExecutor creates a task check database connect executor.
func NewDatabaseConnectExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &DatabaseConnectExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// DatabaseConnectExecutor is the task check database connect executor.
type DatabaseConnectExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run will run the task check database connector executor once.
func (e *DatabaseConnectExecutor) Run(ctx context.Context, _ *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error) {
	database, err := e.store.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
	if err != nil {
		return []api.TaskCheckResult{}, common.Wrap(err, common.Internal)
	}
	if database == nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, "database ID not found %v", task.DatabaseID)
	}

	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, database.Instance, database.Name)
	if err == nil {
		err = driver.Ping(ctx)
	}
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
