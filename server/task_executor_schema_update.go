package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/bytebase/bytebase/api"
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
		return true, nil, fmt.Errorf("invalid database schema update payload: %w", err)
	}

	return runMigration(ctx, server, task, payload.MigrationType, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *SchemaUpdateTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (*SchemaUpdateTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}
