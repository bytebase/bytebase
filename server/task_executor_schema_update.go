package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// NewSchemaUpdateTaskExecutor creates a schema update (DDL) task executor.
func NewSchemaUpdateTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateTaskExecutor is the schema update (DDL) task executor.
type SchemaUpdateTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("SchemaUpdateTaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("stack"))
			terminated = true
			err = fmt.Errorf("encounter internal error when executing sql")
		}
	}()

	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update payload: %w", err)
	}

	return runMigration(ctx, exec.l, server, task, payload.MigrationType, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}
