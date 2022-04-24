package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// NewSchemaUpdateGhostSyncTaskExecutor creates a schema update (gh-ost) sync task executor.
func NewSchemaUpdateGhostSyncTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateGhostSyncTaskExecutor is the schema update (gh-ost) sync task executor.
type SchemaUpdateGhostSyncTaskExecutor struct {
	l *zap.Logger //nolint
}

// RunOnce will run SchemaUpdateGhostSync task once.
func (exec *SchemaUpdateGhostSyncTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	return true, nil, fmt.Errorf("not implemented yet")
}
