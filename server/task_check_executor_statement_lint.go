package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

func NewTaskCheckStatementLintExecutor(logger *zap.Logger) TaskCheckExecutor {
	return &TaskCheckStatementLintExecutor{
		l: logger,
	}
}

type TaskCheckStatementLintExecutor struct {
	l *zap.Logger
}

func (exec *TaskCheckStatementLintExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return []api.TaskCheckResult{}, fmt.Errorf("invalid check statement lint payload: %w", err)
	}

	adviceList, err := advisor.Check(db.MySQL, advisor.Fake, advisor.AdvisorContext{Logger: exec.l}, payload.Statement)
	if err != nil {
		return []api.TaskCheckResult{}, fmt.Errorf("failed to lint statement: %w", err)
	}

	result = []api.TaskCheckResult{}
	for _, advice := range adviceList {
		status := api.TaskCheckStatusSuccess
		switch advice.Status {
		case advisor.Success:
			status = api.TaskCheckStatusSuccess
		case advisor.Warn:
			status = api.TaskCheckStatusWarn
		case advisor.Error:
			status = api.TaskCheckStatusError
		}

		result = append(result, api.TaskCheckResult{
			Status:  status,
			Title:   advice.Title,
			Content: advice.Content,
		})
	}

	return result, nil
}
