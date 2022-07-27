package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// NewTaskCheckTimingExecutor creates a task check timing executor.
func NewTaskCheckTimingExecutor() TaskCheckExecutor {
	return &TaskCheckTimingExecutor{}
}

// TaskCheckTimingExecutor is the task check timing executor.
type TaskCheckTimingExecutor struct {
}

const dataFormat = "2006-01-02 15:04:05"

// Run will run the task check timing executor once.
func (*TaskCheckTimingExecutor) Run(_ context.Context, _ *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	payload := &api.TaskCheckEarliestAllowedTimePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Invalid, "invalid check timing payload: %w", err)
	}

	if payload.EarliestAllowedTs == 0 {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "OK",
				Content:   "Earliest allowed time unset",
			},
		}, nil
	}

	// EarliestAllowedTs is store in UTC+0000
	if time.Now().UTC().Before(time.Unix(payload.EarliestAllowedTs, 0).UTC()) {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.TaskTimingNotAllowed.Int(),
				Title:     "Not ready to run",
				Content:   fmt.Sprintf("Need to wait until the configured earliest running time: %s (UTC+0000)", time.Unix(payload.EarliestAllowedTs, 0).UTC().Format(dataFormat)),
			},
		}, nil
	}

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   fmt.Sprintf("Passed the configured earliest running time: %s (UTC+0000)", time.Unix(payload.EarliestAllowedTs, 0).UTC().Format(dataFormat)),
		},
	}, nil
}
