package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

// TaskCheckExecutor is the task check executor.
type TaskCheckExecutor interface {
	// Run will be called periodically by the task check scheduler
	Run(ctx context.Context, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error)
}
