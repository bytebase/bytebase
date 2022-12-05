package server

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

// NewSchemaBaselineTaskExecutor creates a schema baseline task executor.
func NewSchemaBaselineTaskExecutor() TaskExecutor {
	return &SchemaBaselineTaskExecutor{}
}

// SchemaBaselineTaskExecutor is the schema baseline task executor.
type SchemaBaselineTaskExecutor struct {
	completed int32
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaBaselineTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)
	payload := &api.TaskDatabaseSchemaBaselinePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema baseline payload")
	}

	return runMigration(ctx, server.store, server.dbFactory, server.RollbackRunner, server.ActivityManager, server.profile, task, db.Baseline, payload.Statement, payload.SchemaVersion, nil /* vcsPushEvent */)
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *SchemaBaselineTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (*SchemaBaselineTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}
