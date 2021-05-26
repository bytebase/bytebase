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

func (exec *DefaultTaskExecutor) Run(ctx context.Context, server *Server, taskRun api.TaskRun) (terminated bool, err error) {
	return true, nil
}
