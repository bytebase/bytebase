package server

import (
	"context"
	"time"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// NewSchemaUpdateGhostDropOriginalTableTaskExecutor creates a schema update (gh-ost) drop original table task executor.
func NewSchemaUpdateGhostDropOriginalTableTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateGhostDropOriginalTableTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateGhostDropOriginalTableTaskExecutor is the schema update (gh-ost) drop original table task executor.
type SchemaUpdateGhostDropOriginalTableTaskExecutor struct {
	l *zap.Logger //nolint
}

// RunOnce will run SchemaUpdateGhostDropOriginalTable task once.
func (exec *SchemaUpdateGhostDropOriginalTableTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	time.Sleep(time.Second * 5)
	return true, &api.TaskRunResultPayload{Detail: "not implemented yet"}, nil
}
