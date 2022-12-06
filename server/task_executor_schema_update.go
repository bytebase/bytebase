package server

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

// NewSchemaUpdateTaskExecutor creates a schema update (DDL) task executor.
func NewSchemaUpdateTaskExecutor() TaskExecutor {
	return &SchemaUpdateTaskExecutor{}
}

// SchemaUpdateTaskExecutor is the schema update (DDL) task executor.
type SchemaUpdateTaskExecutor struct {
	completed int32
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update payload")
	}

	return runMigration(ctx, server.store, server.dbFactory, server.RollbackRunner, server.ActivityManager, server.profile, task, db.Migrate, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *SchemaUpdateTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}
