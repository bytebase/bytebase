package server

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
)

// NewDataUpdateTaskExecutor creates a data update (DML) task executor.
func NewDataUpdateTaskExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, profile config.Profile) TaskExecutor {
	return &DataUpdateTaskExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		profile:         profile,
	}
}

// DataUpdateTaskExecutor is the data update (DML) task executor.
type DataUpdateTaskExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	profile         config.Profile
}

// RunOnce will run the data update (DML) task executor once.
func (exec *DataUpdateTaskExecutor) RunOnce(ctx context.Context, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database data update payload")
	}

	return runMigration(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.profile, task, db.Data, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}
