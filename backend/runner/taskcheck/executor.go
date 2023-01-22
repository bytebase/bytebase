package taskcheck

import (
	"context"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// Executor is the task check executor.
type Executor interface {
	// Run will be called periodically by the task check scheduler
	Run(ctx context.Context, taskCheckRun *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error)
}
