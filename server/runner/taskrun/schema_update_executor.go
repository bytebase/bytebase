package taskrun

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/server/component/state"
	"github.com/bytebase/bytebase/server/runner/schemasync"
	"github.com/bytebase/bytebase/store"
)

// NewSchemaUpdateExecutor creates a schema update (DDL) task executor.
func NewSchemaUpdateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &SchemaUpdateExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// SchemaUpdateExecutor is the schema update (DDL) task executor.
type SchemaUpdateExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaUpdateExecutor) RunOnce(ctx context.Context, task *api.Task) (bool, *api.TaskRunResultPayload, error) {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}

	terminated, result, err := runMigration(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.stateCfg, exec.profile, task, db.Migrate, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, task.Instance, task.Database.Name); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", task.Instance.Name),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err),
		)
	}

	return terminated, result, err
}
