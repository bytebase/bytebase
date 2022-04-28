package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"go.uber.org/zap"
)

// NewTaskCheckStatementAdvisorCompositeExecutor creates a task check statement advisor composite executor.
func NewTaskCheckStatementAdvisorCompositeExecutor(logger *zap.Logger) TaskCheckExecutor {
	return &TaskCheckStatementAdvisorCompositeExecutor{
		l: logger,
	}
}

// TaskCheckStatementAdvisorCompositeExecutor is the task check statement advisor composite executor with has sub-advisor.
type TaskCheckStatementAdvisorCompositeExecutor struct {
	l *zap.Logger
}

// Run will run the task check statement advisor composite executor once, and run its sub-advisor one-by-one.
func (exec *TaskCheckStatementAdvisorCompositeExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	if taskCheckRun.Type != api.TaskCheckDatabaseStatementAdvise {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("invalid check statement advisor composite type: %v", taskCheckRun.Type))
	}
	if !server.feature(api.FeatureSchemaReviewPolicy) {
		return nil, common.Errorf(common.NotAuthorized, fmt.Errorf(api.FeatureSchemaReviewPolicy.AccessErrorMessage()))
	}

	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("invalid check statement advise payload: %w", err))
	}

	policy, err := server.store.GetSchemaReviewPolicyByEnvID(ctx, payload.EnvironmentID)
	if err != nil {
		return nil, common.Errorf(common.Internal, fmt.Errorf("failed to get schema review policy: %w", err))
	}

	result = []api.TaskCheckResult{}
	for _, rule := range policy.RuleList {
		if rule.Level == api.SchemaRuleLevelDisabled {
			continue
		}
		advisorType, err := getAdvisorTypeByRule(rule.Type, payload.DbType)
		if err != nil {
			exec.l.Debug("not supported rule", zap.Error(err))
			continue
		}
		adviceList, err := advisor.Check(
			payload.DbType,
			advisorType,
			advisor.Context{
				Logger:    exec.l,
				Charset:   payload.Charset,
				Collation: payload.Collation,
				Rule:      rule,
			},
			payload.Statement,
		)
		if err != nil {
			return nil, common.Errorf(common.Internal, fmt.Errorf("failed to check statement: %w", err))
		}

		for _, advice := range adviceList {
			status := api.TaskCheckStatusSuccess
			switch advice.Status {
			case advisor.Success:
				continue
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
	}
	if len(result) == 0 {
		result = append(result, api.TaskCheckResult{
			Status:  api.TaskCheckStatusSuccess,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return result, nil

}
