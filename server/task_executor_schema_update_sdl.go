package server

import (
	"bytes"
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/differ"
	"github.com/bytebase/bytebase/server/component/dbfactory"
)

// NewSchemaUpdateSDLTaskExecutor creates a schema update (SDL) task executor.
func NewSchemaUpdateSDLTaskExecutor() TaskExecutor {
	return &SchemaUpdateSDLTaskExecutor{}
}

// SchemaUpdateSDLTaskExecutor is the schema update (SDL) task executor.
type SchemaUpdateSDLTaskExecutor struct {
	completed int32
}

// RunOnce will run the schema update (SDL) task executor once.
func (exec *SchemaUpdateSDLTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)
	payload := &api.TaskDatabaseSchemaUpdateSDLPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}

	ddl, err := exec.computeDatabaseSchemaDiff(ctx, server.dbFactory, task.Database, payload.Statement)
	if err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema diff")
	}
	return runMigration(ctx, server.store, server.dbFactory, server.RollbackRunner, server.ActivityManager, server.profile, task, db.MigrateSDL, ddl, payload.SchemaVersion, payload.VCSPushEvent)
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *SchemaUpdateSDLTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
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
