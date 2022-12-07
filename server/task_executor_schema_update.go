package server

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
)

// NewSchemaUpdateTaskExecutor creates a schema update (DDL) task executor.
func NewSchemaUpdateTaskExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, rollbackRunner *RollbackRunner, activityManager *ActivityManager, profile config.Profile) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		store:           store,
		dbFactory:       dbFactory,
		rollbackRunner:  rollbackRunner,
		activityManager: activityManager,
		profile:         profile,
	}
}

// SchemaUpdateTaskExecutor is the schema update (DDL) task executor.
type SchemaUpdateTaskExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	rollbackRunner  *RollbackRunner
	activityManager *ActivityManager
	profile         config.Profile
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaUpdateTaskExecutor) RunOnce(ctx context.Context, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}

	return runMigration(ctx, exec.store, exec.dbFactory, exec.rollbackRunner, exec.activityManager, exec.profile, task, db.Migrate, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}
