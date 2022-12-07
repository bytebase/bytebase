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

// NewSchemaBaselineTaskExecutor creates a schema baseline task executor.
func NewSchemaBaselineTaskExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, rollbackRunner *RollbackRunner, activityManager *activity.Manager, profile config.Profile) TaskExecutor {
	return &SchemaBaselineTaskExecutor{
		store:           store,
		dbFactory:       dbFactory,
		rollbackRunner:  rollbackRunner,
		activityManager: activityManager,
		profile:         profile,
	}
}

// SchemaBaselineTaskExecutor is the schema baseline task executor.
type SchemaBaselineTaskExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	rollbackRunner  *RollbackRunner
	activityManager *activity.Manager
	profile         config.Profile
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaBaselineTaskExecutor) RunOnce(ctx context.Context, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaBaselinePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema baseline payload")
	}

	return runMigration(ctx, exec.store, exec.dbFactory, exec.rollbackRunner, exec.activityManager, exec.profile, task, db.Baseline, payload.Statement, payload.SchemaVersion, nil /* vcsPushEvent */)
}
