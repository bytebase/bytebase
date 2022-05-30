package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
)

// NewSchemaUpdateTaskExecutor creates a schema update (DDL) task executor.
func NewSchemaUpdateTaskExecutor() TaskExecutor {
	return &SchemaUpdateTaskExecutor{}
}

// SchemaUpdateTaskExecutor is the schema update (DDL) task executor.
type SchemaUpdateTaskExecutor struct {
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update payload: %w", err)
	}

	return runMigration(ctx, server, task, payload.MigrationType, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}
