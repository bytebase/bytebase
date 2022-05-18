package server

import (
	"context"

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
	l *zap.Logger //nolint
}

// RunOnce will run SchemaUpdateGhostCutover task once.
func (exec *SchemaUpdateGhostCutoverTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	return true, &api.TaskRunResultPayload{Detail: "not implemented yet"}, nil
}
