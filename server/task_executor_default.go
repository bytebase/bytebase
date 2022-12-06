package server

import (
	"context"
	"fmt"
	"sync/atomic"

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
	completed int32
}

// RunOnce will run the default task executor once.
func (exec *DefaultTaskExecutor) RunOnce(_ context.Context, _ *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run default task type", zap.String("task", task.Name))
	defer atomic.StoreInt32(&exec.completed, 1)

	return true, &api.TaskRunResultPayload{Detail: fmt.Sprintf("No-op task %s", task.Name)}, nil
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *DefaultTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}
