package taskcheck

import (
	"context"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

// Executor is the task check executor.
type Executor interface {
	// Run will be called periodically by the task check scheduler
	Run(ctx context.Context, taskCheckRun *store.TaskCheckRunMessage, task *store.TaskMessage) (result []api.TaskCheckResult, err error)
}
