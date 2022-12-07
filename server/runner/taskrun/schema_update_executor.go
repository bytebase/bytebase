package taskrun

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

// NewSchemaUpdateExecutor creates a schema update (DDL) task executor.
func NewSchemaUpdateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, profile config.Profile) Executor {
	return &SchemaUpdateExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		profile:         profile,
	}
}

// SchemaUpdateExecutor is the schema update (DDL) task executor.
type SchemaUpdateExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	profile         config.Profile
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaUpdateExecutor) RunOnce(ctx context.Context, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}

	return runMigration(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.profile, task, db.Migrate, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}
