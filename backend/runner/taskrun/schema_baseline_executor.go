package taskrun

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/server/component/activity"
	"github.com/bytebase/bytebase/backend/server/component/config"
	"github.com/bytebase/bytebase/backend/server/component/dbfactory"
	"github.com/bytebase/bytebase/backend/server/component/state"
	"github.com/bytebase/bytebase/backend/store"
)

// NewSchemaBaselineExecutor creates a schema baseline task executor.
func NewSchemaBaselineExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &SchemaBaselineExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// SchemaBaselineExecutor is the schema baseline task executor.
type SchemaBaselineExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaBaselineExecutor) RunOnce(ctx context.Context, task *api.Task) (bool, *api.TaskRunResultPayload, error) {
	payload := &api.TaskDatabaseSchemaBaselinePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema baseline payload")
	}

	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		EnvironmentID: &task.Database.Instance.Environment.ResourceID,
		InstanceID:    &task.Database.Instance.ResourceID,
		DatabaseName:  &task.Database.Name,
	})
	if err != nil {
		return true, nil, err
	}

	terminated, result, err := runMigration(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.stateCfg, exec.profile, task, db.Baseline, "" /* statement */, payload.SchemaVersion, nil /* vcsPushEvent */)
	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", task.Instance.Name),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err),
		)
	}

	return terminated, result, err
}
