package taskrun

import (
	"context"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewSchemaUpdateGhostExecutor creates a schema update (gh-ost) task executor.
func NewSchemaUpdateGhostExecutor() Executor {
	return &SchemaUpdateGhostExecutor{}
}

// SchemaUpdateGhostExecutor is the schema update (gh-ost) task executor.
type SchemaUpdateGhostExecutor struct {
}

// TODO(p0ny): implement.
func (*SchemaUpdateGhostExecutor) RunOnce(ctx context.Context, taskContext context.Context, task *store.TaskMessage, _ int) (terminated bool, result *storepb.TaskRunResult, err error) {
	return true, &storepb.TaskRunResult{
		Detail: "TBD",
	}, nil
}
