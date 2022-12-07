package taskcheck

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

// taskCheckExecutor is the task check executor.
type taskCheckExecutor interface {
	// Run will be called periodically by the task check scheduler
	Run(ctx context.Context, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error)
}
