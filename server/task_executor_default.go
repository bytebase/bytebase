package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

func NewDefaultTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &DefaultTaskExecutor{
		l: logger,
	}
}

type DefaultTaskExecutor struct {
	l *zap.Logger
}

func (exec *DefaultTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	exec.l.Info("Run default task type", zap.String("task", task.Name))

	return true, &api.TaskRunResultPayload{Detail: fmt.Sprintf("No-op task %s", task.Name)}, nil
}
