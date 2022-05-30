package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

// NewDataUpdateTaskExecutor creates a data update (DML) task executor.
func NewDataUpdateTaskExecutor() TaskExecutor {
	return &SchemaUpdateTaskExecutor{}
}

// DataUpdateTaskExecutor is the data update (DML) task executor.
type DataUpdateTaskExecutor struct {
}

// RunOnce will run the data update (DML) task executor once.
func (exec *DataUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database data update payload: %w", err)
	}

	return runMigration(ctx, server, task, db.Data, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}
