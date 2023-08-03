package taskrun

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

// NewDataUpdateExecutor creates a data update (DML) task executor.
func NewDataUpdateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterpriseAPI.LicenseService, stateCfg *state.State, profile config.Profile) Executor {
	return &DataUpdateExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		license:         license,
		stateCfg:        stateCfg,
		profile:         profile,
	}
}

// DataUpdateExecutor is the data update (DML) task executor.
type DataUpdateExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	license         enterpriseAPI.LicenseService
	stateCfg        *state.State
	profile         config.Profile
}

// RunOnce will run the data update (DML) task executor once.
func (exec *DataUpdateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database data update payload")
	}

	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, err
	}
	return runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.activityManager, exec.license, exec.stateCfg, exec.profile, task, db.Data, statement, payload.SchemaVersion, &payload.SheetID, payload.VCSPushEvent)
}
