package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

// NewTaskCheckStatementAdvisorSimpleExecutor creates a task check statement simple advisor executor.
func NewTaskCheckStatementAdvisorSimpleExecutor() TaskCheckExecutor {
	return &TaskCheckStatementAdvisorSimpleExecutor{}
}

// TaskCheckStatementAdvisorSimpleExecutor is the task check statement advisor simple executor.
type TaskCheckStatementAdvisorSimpleExecutor struct {
}

// Run will run the task check statement advisor executor once.
func (exec *TaskCheckStatementAdvisorSimpleExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	var advisorType advisor.Type
	switch taskCheckRun.Type {
	case api.TaskCheckDatabaseStatementFakeAdvise:
		advisorType = advisor.Fake
	case api.TaskCheckDatabaseStatementSyntax:
		advisorType = advisor.MySQLSyntax
	}

	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("invalid check statement advise payload: %w", err))
	}

	adviceList, err := advisor.Check(
		payload.DbType,
		advisorType,
		advisor.Context{
			Charset:   payload.Charset,
			Collation: payload.Collation,
		},
		payload.Statement,
	)
	if err != nil {
		return nil, common.Errorf(common.Internal, fmt.Errorf("failed to check statement: %w", err))
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
			Code:    advice.Code,
			Title:   advice.Title,
			Content: advice.Content,
		})
	}

	return result, nil
}
