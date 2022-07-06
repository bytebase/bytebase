package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

// NewSchemaUpdateGhostDropOriginalTableTaskExecutor creates a schema update (gh-ost) drop original table task executor.
func NewSchemaUpdateGhostDropOriginalTableTaskExecutor() TaskExecutor {
	return &SchemaUpdateGhostDropOriginalTableTaskExecutor{}
}

// SchemaUpdateGhostDropOriginalTableTaskExecutor is the schema update (gh-ost) drop original table task executor.
type SchemaUpdateGhostDropOriginalTableTaskExecutor struct {
}

// RunOnce will run SchemaUpdateGhostDropOriginalTable task once.
func (exec *SchemaUpdateGhostDropOriginalTableTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	return true, &api.TaskRunResultPayload{Detail: "not implemented yet"}, nil
}
