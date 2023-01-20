package taskcheck

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/server/component/dbfactory"
	"github.com/bytebase/bytebase/backend/store"
)

// NewPITRMySQLExecutor creates a task check migration schema executor.
func NewPITRMySQLExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &PITRMySQLExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// PITRMySQLExecutor is the task check migration schema executor.
type PITRMySQLExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run will run the task check migration schema executor once.
func (e *PITRMySQLExecutor) Run(ctx context.Context, _ *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error) {
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

	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by ID %d", instanceID)
	}

	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
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
