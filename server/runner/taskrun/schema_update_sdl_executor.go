package taskrun

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/differ"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/server/component/state"
	"github.com/bytebase/bytebase/server/runner/schemasync"
	"github.com/bytebase/bytebase/store"
)

// NewSchemaUpdateSDLExecutor creates a schema update (SDL) task executor.
func NewSchemaUpdateSDLExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &SchemaUpdateSDLExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// SchemaUpdateSDLExecutor is the schema update (SDL) task executor.
type SchemaUpdateSDLExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run the schema update (SDL) task executor once.
func (exec *SchemaUpdateSDLExecutor) RunOnce(ctx context.Context, task *api.Task) (bool, *api.TaskRunResultPayload, error) {
	payload := &api.TaskDatabaseSchemaUpdateSDLPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		EnvironmentID: &task.Database.Instance.Environment.ResourceID,
		InstanceID:    &task.Database.Instance.ResourceID,
		DatabaseName:  &task.Database.Name,
	})
	if err != nil {
		return true, nil, err
	}
	ddl, err := exec.computeDatabaseSchemaDiff(ctx, exec.dbFactory, task.Database, payload.Statement)
	if err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema diff")
	}
	terminated, result, err := runMigration(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.stateCfg, exec.profile, task, db.MigrateSDL, ddl, payload.SchemaVersion, payload.VCSPushEvent)

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", task.Instance.Name),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err),
		)
	}

	return terminated, result, err
}

// computeDatabaseSchemaDiff computes the diff between current database schema
// and the given schema. It returns an empty string if there is no applicable
// diff.
func (*SchemaUpdateSDLExecutor) computeDatabaseSchemaDiff(ctx context.Context, dbFactory *dbfactory.DBFactory, database *api.Database, newSchemaStr string) (string, error) {
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, database.Instance, database.Name)
	if err != nil {
		return "", errors.Wrap(err, "get admin driver")
	}
	defer func() {
		_ = driver.Close(ctx)
	}()

	var schema bytes.Buffer
	_, err = driver.Dump(ctx, database.Name, &schema, true /* schemaOnly */)
	if err != nil {
		return "", errors.Wrap(err, "dump old schema")
	}

	var engine parser.EngineType
	switch database.Instance.Engine {
	case db.Postgres:
		engine = parser.Postgres
	case db.MySQL:
		engine = parser.MySQL
	default:
		return "", errors.Errorf("unsupported database engine %q", database.Instance.Engine)
	}

	diff, err := differ.SchemaDiff(engine, schema.String(), newSchemaStr)
	if err != nil {
		return "", errors.New("compute schema diff")
	}
	return diff, nil
}
