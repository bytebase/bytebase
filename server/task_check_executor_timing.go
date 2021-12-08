package server

import (
	"context"
	"fmt"
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
	"time"
)

// NewTaskCheckTimingExecutor creates a task check timing executor.
func NewTaskCheckTimingExecutor(logger *zap.Logger) TaskCheckExecutor {
	return &TaskCheckTimingExecutor{
		l: logger,
	}
}

// TaskCheckTimingExecutor is the task check timing executor.
type TaskCheckTimingExecutor struct {
	l *zap.Logger
}

// Run will run the task check timing executor once.
func (exec *TaskCheckTimingExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	taskFind := &api.TaskFind{
		ID: &taskCheckRun.TaskID,
	}
	task, err := server.TaskService.FindTask(ctx, taskFind)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err)
	}
	if time.Now().Before(time.Unix(task.NotBeforeTs, 0)) {
		return []api.TaskCheckResult{
			{
				Status:  api.TaskCheckStatusError,
				Code:    common.TimingNotAllowed,
				Title:   "TimingNotAllowed",
				Content: fmt.Sprintf("Did not pass the earliest execution timing"),
			},
		}, nil
	} else {
		return []api.TaskCheckResult{
			{
				Status:  api.TaskCheckStatusSuccess,
				Code:    common.Ok,
				Title:   "OK",
				Content: fmt.Sprintf("Passed the earliest execution timing"),
			},
		}, nil
	}
}
