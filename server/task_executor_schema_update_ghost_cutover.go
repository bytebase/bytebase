package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bytebase/bytebase/api"
	ghostsql "github.com/github/gh-ost/go/sql"
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
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("SchemaUpdateGhostCutoverTaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("stack"))
			terminated = true
			err = fmt.Errorf("encounter internal error when executing schema update cutover task")
		}
	}()

	taskDAGList, err := server.store.FindTaskDAGList(ctx, &api.TaskDAGFind{ToTaskID: task.ID})
	if err != nil {
		return true, nil, fmt.Errorf("failed to fetch taskDAG for schema update gh-ost cutover task, id: %v", task.ID)
	}
	if len(taskDAGList) != 1 {
		return true, nil, fmt.Errorf("expected to have one taskDAG in taskDAGList, get %v", len(taskDAGList))
	}
	syncTaskID := taskDAGList[0].FromTaskID
	taskList, err := server.store.FindTask(ctx, &api.TaskFind{ID: &syncTaskID}, true)
	if err != nil {
		return true, nil, fmt.Errorf("failed to get schema update gh-ost sync task for cutover task, error: %w", err)
	}
	if len(taskList) != 1 {
		return true, nil, fmt.Errorf("expected to have one task in taskList, get %v", len(taskList))
	}
	syncTask := taskList[0]

	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(syncTask.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update gh-ost sync payload: %w", err)
	}

	statement := strings.TrimSpace(payload.Statement)

	parser := ghostsql.NewParserFromAlterStatement(statement)
	if !parser.HasExplicitTable() {
		return true, nil, fmt.Errorf("table must be provided and table name must not be empty, or alterStatement must specify table name")
	}
	tableName := parser.GetExplicitTable()

	filename := fmt.Sprintf("/tmp/gh-ost.%v.%v.%v.%v.postponeFlag", syncTask.ID, task.Database.ID, task.Database.Name, tableName)

	if err := os.Remove(filename); err != nil {
		return true, nil, fmt.Errorf("failed to remove postpone flag file, error: %w", err)
	}

	return true, &api.TaskRunResultPayload{Detail: "cutover done"}, nil
}
