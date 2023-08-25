package taskcheck

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func runExecutorOnce(ctx context.Context, exec Executor, taskCheckRun *store.TaskCheckRunMessage, task *store.TaskMessage) (result []api.TaskCheckResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			log.Error("TaskChecker PANIC RECOVER", zap.Error(panicErr), zap.Stack("panic-stack"))
			result = nil
			err = errors.Errorf("encounter internal error when executing check")
		}
	}()

	return exec.Run(ctx, taskCheckRun, task)
}

// Executor is the task check executor.
type Executor interface {
	// Run will be called periodically by the task check scheduler
	Run(ctx context.Context, taskCheckRun *store.TaskCheckRunMessage, task *store.TaskMessage) (result []api.TaskCheckResult, err error)
}

// TaskPayload is the task payload.
type TaskPayload struct {
	SheetID int `json:"sheetId,omitempty"`
}
