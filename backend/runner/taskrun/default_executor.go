package taskrun

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewDefaultExecutor creates a default task executor.
func NewDefaultExecutor() Executor {
	return &DefaultExecutor{}
}

// DefaultExecutor is the default task executor.
type DefaultExecutor struct {
}

// RunOnce will run the default task executor once.
func (*DefaultExecutor) RunOnce(_ context.Context, _ context.Context, task *store.TaskMessage, _ int) (terminated bool, result *storepb.TaskRunResult, err error) {
	slog.Info("Run default task type", slog.String("task", task.Name))

	return true, &storepb.TaskRunResult{Detail: fmt.Sprintf("No-op task %s", task.Name)}, nil
}
