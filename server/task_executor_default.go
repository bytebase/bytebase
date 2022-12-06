package server

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
)

// NewDefaultTaskExecutor creates a default task executor.
func NewDefaultTaskExecutor() TaskExecutor {
	return &DefaultTaskExecutor{}
}

// DefaultTaskExecutor is the default task executor.
type DefaultTaskExecutor struct {
}

// RunOnce will run the default task executor once.
func (*DefaultTaskExecutor) RunOnce(_ context.Context, _ *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run default task type", zap.String("task", task.Name))

	return true, &api.TaskRunResultPayload{Detail: fmt.Sprintf("No-op task %s", task.Name)}, nil
}
