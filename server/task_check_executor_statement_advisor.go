package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor"
	"go.uber.org/zap"
)

func NewTaskCheckStatementAdvisorExecutor(logger *zap.Logger) TaskCheckExecutor {
	return &TaskCheckStatementAdvisorExecutor{
		l: logger,
	}
}

type TaskCheckStatementAdvisorExecutor struct {
	l *zap.Logger
}

func (exec *TaskCheckStatementAdvisorExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return []api.TaskCheckResult{}, fmt.Errorf("invalid check statement advise payload: %w", err)
	}

	var advisorType advisor.AdvisorType
	switch taskCheckRun.Type {
	case api.TaskCheckDatabaseStatementFakeAdvise:
		advisorType = advisor.Fake
	case api.TaskCheckDatabaseStatementSyntax:
		advisorType = advisor.MySQLSyntax
	}

	adviceList, err := advisor.Check(
		payload.DbType,
		advisorType,
		advisor.AdvisorContext{
			Logger:    exec.l,
			Charset:   payload.Charset,
			Collation: payload.Collation,
		},
		payload.Statement,
	)
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
