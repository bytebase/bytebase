package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

// NewDataUpdateTaskExecutor creates a data update (DML) task executor.
func NewDataUpdateTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

// NewDataUpdateTaskExecutor is the data update (DML) task executor.
type DataUpdateTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run the data update (DML) task executor once.
func (exec *DataUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DataUpdateTaskExecutor PANIC RECOVER", zap.Error(panicErr))
			terminated = true
			err = fmt.Errorf("encounter internal error when executing sql")
		}
	}()

	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database data update payload: %w", err)
	}

	return runMigration(ctx, exec.l, server, task, db.Data, payload.Statement, payload.VCSPushEvent)
}
