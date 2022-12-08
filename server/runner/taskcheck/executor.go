package taskcheck

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

// Executor is the task check executor.
type Executor interface {
	// Run will be called periodically by the task check scheduler
	Run(ctx context.Context, taskCheckRun *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error)
}
