package server

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

// NewDataUpdateTaskExecutor creates a data update (DML) task executor.
func NewDataUpdateTaskExecutor() TaskExecutor {
	return &DataUpdateTaskExecutor{}
}

// DataUpdateTaskExecutor is the data update (DML) task executor.
type DataUpdateTaskExecutor struct {
	completed int32
}

// RunOnce will run the data update (DML) task executor once.
func (*DataUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database data update payload")
	}

	return runMigration(ctx, server.store, server.dbFactory, server.RollbackRunner, server.ActivityManager, server.profile, task, db.Data, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent)
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *DataUpdateTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}
