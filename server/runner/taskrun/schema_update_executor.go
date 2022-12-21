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

	statement := payload.Statement
	if payload.SheetID > 0 {
		sheet, err := exec.store.GetSheet(ctx, &api.SheetFind{ID: &payload.SheetID, LoadFull: true}, api.SystemBotID)
		if err != nil {
			return true, nil, err
		}
		if sheet == nil {
			return true, nil, errors.Errorf("sheet ID %v not found", payload.SheetID)
		}
		statement = sheet.Statement
	}
	terminated, result, err := runMigration(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.stateCfg, exec.profile, task, db.Migrate, statement, payload.SchemaVersion, payload.VCSPushEvent)
	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, task.Instance, task.Database.Name, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", task.Instance.Name),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err),
		)
	}

	return terminated, result, err
}
