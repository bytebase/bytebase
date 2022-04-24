package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// NewSchemaUpdateGhostDropOriginalTableTaskExecutor creates a schema update (gh-ost) drop original table task executor.
func NewSchemaUpdateGhostDropOriginalTableTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateGhostDropOriginalTableTaskExecutor is the schema update (gh-ost) drop original table task executor.
type SchemaUpdateGhostDropOriginalTableTaskExecutor struct {
	l *zap.Logger //nolint
}

// RunOnce will run SchemaUpdateGhostDropOriginalTable task once.
func (exec *SchemaUpdateGhostDropOriginalTableTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	return true, nil, fmt.Errorf("not implemented yet")
}
