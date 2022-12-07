package server

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/differ"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
)

// NewSchemaUpdateSDLTaskExecutor creates a schema update (SDL) task executor.
func NewSchemaUpdateSDLTaskExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, rollbackRunner *RollbackRunner, activityManager *ActivityManager, profile config.Profile) TaskExecutor {
	return &SchemaUpdateSDLTaskExecutor{
		store:           store,
		dbFactory:       dbFactory,
		rollbackRunner:  rollbackRunner,
		activityManager: activityManager,
		profile:         profile,
	}
}

// SchemaUpdateSDLTaskExecutor is the schema update (SDL) task executor.
type SchemaUpdateSDLTaskExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	rollbackRunner  *RollbackRunner
	activityManager *ActivityManager
	profile         config.Profile
}

// RunOnce will run the schema update (SDL) task executor once.
func (exec *SchemaUpdateSDLTaskExecutor) RunOnce(ctx context.Context, _ *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaUpdateSDLPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}

	ddl, err := exec.computeDatabaseSchemaDiff(ctx, exec.dbFactory, task.Database, payload.Statement)
	if err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema diff")
	}
	return runMigration(ctx, exec.store, exec.dbFactory, exec.rollbackRunner, exec.activityManager, exec.profile, task, db.MigrateSDL, ddl, payload.SchemaVersion, payload.VCSPushEvent)
}

// computeDatabaseSchemaDiff computes the diff between current database schema
// and the given schema. It returns an empty string if there is no applicable
// diff.
func (*SchemaUpdateSDLTaskExecutor) computeDatabaseSchemaDiff(ctx context.Context, dbFactory *dbfactory.DBFactory, database *api.Database, newSchemaStr string) (string, error) {
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
