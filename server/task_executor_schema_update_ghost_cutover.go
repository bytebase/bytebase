package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// NewSchemaUpdateGhostCutoverTaskExecutor creates a schema update (gh-ost) cutover task executor.
func NewSchemaUpdateGhostCutoverTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateGhostCutoverTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateGhostCutoverTaskExecutor is the schema update (gh-ost) cutover task executor.
type SchemaUpdateGhostCutoverTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run SchemaUpdateGhostCutover task once.
func (exec *SchemaUpdateGhostCutoverTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {

	taskDAG, err := server.store.GetTaskDAGByToTaskID(ctx, task.ID)
	if err != nil {
		return true, nil, fmt.Errorf("failed to get a single taskDAG for schema update gh-ost cutover task, id: %v, error: %w", task.ID, err)
	}
	syncTaskID := taskDAG.FromTaskID
	syncTask, err := server.store.GetTaskByID(ctx, syncTaskID)
	if err != nil {
		return true, nil, fmt.Errorf("failed to get schema update gh-ost sync task for cutover task, error: %w", err)
	}
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(syncTask.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update gh-ost sync payload: %w", err)
	}

	tableName, err := getTableNameFromStatement(payload.Statement)
	if err != nil {
		return true, nil, fmt.Errorf("failed to parse table name from statement, error: %w", err)
	}

	socketFilename := getSocketFilename(syncTaskID, task.Database.ID, task.Database.Name, tableName)
	postponeFilename := getPostponeFlagFilename(syncTaskID, task.Database.ID, task.Database.Name, tableName)

	if err := os.Remove(postponeFilename); err != nil {
		return true, nil, fmt.Errorf("failed to remove postpone flag file, error: %w", err)
	}

	ticker := time.NewTicker(time.Second * 1)

	for range ticker.C {
		if _, err := os.Stat(socketFilename); err != nil {
			break
		}
	}

	return true, &api.TaskRunResultPayload{Detail: "cutover done"}, nil
}
