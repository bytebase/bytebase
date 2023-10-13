package taskrun

import (
	"context"
	"fmt"
	"log/slog"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

// NewDefaultExecutor creates a default task executor.
func NewDefaultExecutor() Executor {
	return &DefaultExecutor{}
}

// DefaultExecutor is the default task executor.
type DefaultExecutor struct {
}

// RunOnce will run the default task executor once.
func (*DefaultExecutor) RunOnce(_ context.Context, _ context.Context, task *store.TaskMessage, _ int) (terminated bool, result *api.TaskRunResultPayload, err error) {
	slog.Info("Run default task type", slog.String("task", task.Name))

	return true, &api.TaskRunResultPayload{Detail: fmt.Sprintf("No-op task %s", task.Name)}, nil
}
