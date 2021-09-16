package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

type TaskCheckExecutor interface {
	// Run will be called periodically by the task check scheduler
	Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error)
}
