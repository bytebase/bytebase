package server

import (
	"context"
	"log"

	"github.com/bytebase/bytebase/api"
)

func NewDefaultTaskExecutor(logger *log.Logger) TaskExecutor {
	return &DefaultTaskExecutor{
		l: logger,
	}
}

type DefaultTaskExecutor struct {
	l *log.Logger
}

func (exec *DefaultTaskExecutor) Run(ctx context.Context, taskRun api.TaskRun) (terminated bool, err error) {
	return true, nil
}
