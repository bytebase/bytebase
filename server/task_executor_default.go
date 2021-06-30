package server

import (
	"context"

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

func (exec *DefaultTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, err error) {
	exec.l.Info("Run default task type", zap.String("task", task.Name))

	return true, nil
}
