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

// TaskPayload is the task payload.
type TaskPayload struct {
	Statement string `json:"statement,omitempty"`
	SheetID   int    `json:"sheetId,omitempty"`
}
