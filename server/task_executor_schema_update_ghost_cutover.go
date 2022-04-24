package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// NewSchemaUpdateGhostCutoverTaskExecutor creates a schema update (gh-ost) cutover task executor.
func NewSchemaUpdateGhostCutoverTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateGhostCutoverTaskExecutorExecutor is the schema update (gh-ost) cutover task executor.
type SchemaUpdateGhostCutoverTaskExecutorExecutor struct {
	l *zap.Logger //nolint
}

// RunOnce will run SchemaUpdateGhostCutover task once.
func (exec *SchemaUpdateGhostCutoverTaskExecutorExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	return true, nil, fmt.Errorf("not implemented yet")
}
