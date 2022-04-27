package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

// NewTaskCheckStatementAdvisorExecutor creates a task check statement advisor executor.
func NewTaskCheckStatementAdvisorExecutor(logger *zap.Logger) TaskCheckExecutor {
	return &TaskCheckStatementAdvisorExecutor{
		l: logger,
	}
}

// TaskCheckStatementAdvisorExecutor is the task check statement advisor executor.
type TaskCheckStatementAdvisorExecutor struct {
	l *zap.Logger
}

// Run will run the task check statement advisor executor once.
func (exec *TaskCheckStatementAdvisorExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	var advisorType advisor.Type
	switch taskCheckRun.Type {
	case api.TaskCheckDatabaseStatementFakeAdvise:
		advisorType = advisor.Fake
	case api.TaskCheckDatabaseStatementSyntax:
		advisorType = advisor.MySQLSyntax
	case api.TaskCheckDatabaseStatementCompatibility:
		if !server.feature(api.FeatureBackwardCompatibility) {
			return []api.TaskCheckResult{}, common.Errorf(common.NotAuthorized, fmt.Errorf(api.FeatureBackwardCompatibility.AccessErrorMessage()))
		}
		advisorType = advisor.MySQLMigrationCompatibility
	case api.TaskCheckDatabaseStatementSchemaReview:
		if !server.feature(api.FeatureSchemaReviewPolicy) {
			return []api.TaskCheckResult{}, common.Errorf(common.NotAuthorized, fmt.Errorf(api.FeatureBackwardCompatibility.AccessErrorMessage()))
		}
		return exec.RunSchemaReview(ctx, server, taskCheckRun)
	}

	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Invalid, fmt.Errorf("invalid check statement advise payload: %w", err))
	}

	adviceList, err := advisor.Check(
		payload.DbType,
		advisorType,
		advisor.Context{
			Logger:    exec.l,
			Charset:   payload.Charset,
			Collation: payload.Collation,
		},
		payload.Statement,
	)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("failed to check statement: %w", err))
	}

	return generateResultList(adviceList), nil
}

// RunSchemaReview will run the schema review check according to schema review policy.
func (exec *TaskCheckStatementAdvisorExecutor) RunSchemaReview(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Invalid, fmt.Errorf("invalid check statement advise payload: %w", err))
	}

	if payload.DbType != db.MySQL && payload.DbType != db.TiDB {
		return []api.TaskCheckResult{}, common.Errorf(common.NotImplemented, fmt.Errorf("not implemented schema review for %s yet", payload.DbType))
	}

	policy, err := server.store.GetSchemaReviewPolicyByEnvID(ctx, payload.EnvironmentID)
	if err != nil {
		return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("failed to get schema review policy: %w", err))
	}

	result = []api.TaskCheckResult{}
	for _, rule := range policy.RuleList {
		if rule.Level == api.SchemaRuleLevelDisabled {
			continue
		}
		advisorType, err := getAdvisorTypeByRule(rule.Type)
		if err != nil {
			exec.l.Debug("rule not support", zap.Error(err))
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
			return []api.TaskCheckResult{}, common.Errorf(common.Internal, fmt.Errorf("failed to check statement: %w", err))
		}
		result = append(result, generateResultList(adviceList)...)
	}
	return result, nil
}

func generateResultList(adviceList []advisor.Advice) []api.TaskCheckResult {
	result := []api.TaskCheckResult{}
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
	return result
}
