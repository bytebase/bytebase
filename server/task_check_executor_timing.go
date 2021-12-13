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

const dataFormat = "2006-01-02 15:04:05"

// Run will run the task check timing executor once.
func (exec *TaskCheckTimingExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	taskFind := &api.TaskFind{
		ID: &taskCheckRun.TaskID,
	}
	task, err := server.TaskService.FindTask(ctx, taskFind)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, err)
	}
	exec.l.Info("Checking TIMING", zap.Int64("timestamp", task.NotBeforeTs))
	if time.Now().Before(time.Unix(task.NotBeforeTs, 0)) {
		return []api.TaskCheckResult{
			{
				Status:  api.TaskCheckStatusError,
				Code:    common.TaskTimingNotAllowed,
				Title:   "TimingNotAllowed",
				Content: fmt.Sprintf("Did not pass the earliest execution timing: %s", time.Unix(task.NotBeforeTs, 0).Format(dataFormat)),
			},
		}, nil
	} else {
		return []api.TaskCheckResult{
			{
				Status:  api.TaskCheckStatusSuccess,
				Code:    common.Ok,
				Title:   "OK",
				Content: fmt.Sprintf("Passed the earliest execution timing: %s", time.Unix(task.NotBeforeTs, 0).Format(dataFormat)),
			},
		}, nil
	}
}
