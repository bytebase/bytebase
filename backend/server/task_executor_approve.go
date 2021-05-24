package server

import (
	"context"
	"log"

	"github.com/bytebase/bytebase/api"
)

func NewApproveTaskExecutor(logger *log.Logger) TaskExecutor {
	return &ApproveTaskExecutor{
		l: logger,
	}
}

type ApproveTaskExecutor struct {
	l *log.Logger
}

func (exec *ApproveTaskExecutor) Run(ctx context.Context, taskRun api.TaskRun) (terminated bool, err error) {
	return true, nil
}
