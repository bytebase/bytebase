package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// Schema review policy consists of a list of schema review rules.
// There is such a logical mapping in bytebase backend:
//   1. One schema review policy maps a TaskCheckRun.
//   2. Each schema review rule type maps an advisor.Type.
//   3. Each [db.Type][AdvisorType] maps an advisor.
//
// How to add a schema review rule:
//   1. Implement an advisor.(plugin/xxx)
//   2. Register this advisor in map[db.Type][AdvisorType].(plugin/advisor.go)
//   3. Map SchemaReviewRuleType to advisor.Type in getAdvisorTypeByRule(current file).

// NewTaskCheckStatementAdvisorCompositeExecutor creates a task check statement advisor composite executor.
func NewTaskCheckStatementAdvisorCompositeExecutor() TaskCheckExecutor {
	return &TaskCheckStatementAdvisorCompositeExecutor{}
}

// TaskCheckStatementAdvisorCompositeExecutor is the task check statement advisor composite executor with has sub-advisor.
type TaskCheckStatementAdvisorCompositeExecutor struct {
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

	policy, err := server.store.GetSchemaReviewPolicyNormalByID(ctx, payload.PolicyID)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			return []api.TaskCheckResult{
				{
					Status:  api.TaskCheckStatusWarn,
					Code:    common.TaskCheckEmptySchemaReviewPolicy,
					Title:   "Empty schema review policy or disabled",
					Content: "",
				},
			}, nil
		}
		return nil, common.Errorf(common.Internal, fmt.Errorf("failed to get schema review policy: %w", err))
	}

	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return nil, common.Errorf(common.Internal, fmt.Errorf("failed to get task by id: %w", err))
	}

	result = []api.TaskCheckResult{}
	for _, rule := range policy.RuleList {
		if rule.Level == advisor.SchemaRuleLevelDisabled {
			continue
		}
		advisorType, err := getAdvisorTypeByRule(rule.Type, payload.DbType)
		if err != nil {
			log.Debug("not supported rule", zap.Error(err))
			continue
		}
		adviceList, err := advisor.Check(
			payload.DbType,
			advisorType,
			advisor.Context{
				Charset:   payload.Charset,
				Collation: payload.Collation,
				Rule:      rule,
				Catalog:   store.NewCatalog(task.DatabaseID, server.store),
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
	// There may be multiple syntax errors, return one only.
	if len(result) > 0 && result[0].Title == advisor.SyntaxErrorTitle {
		return result[:1], nil
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

func getAdvisorTypeByRule(ruleType advisor.SchemaReviewRuleType, engine db.Type) (advisor.Type, error) {
	switch ruleType {
	case advisor.SchemaRuleStatementRequireWhere:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLWhereRequirement, nil
		}
	case advisor.SchemaRuleStatementNoLeadingWildcardLike:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLNoLeadingWildcardLike, nil
		}
	case advisor.SchemaRuleStatementNoSelectAll:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLNoSelectAll, nil
		}
	case advisor.SchemaRuleSchemaBackwardCompatibility:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLMigrationCompatibility, nil
		}
	case advisor.SchemaRuleTableNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLNamingTableConvention, nil
		}
	case advisor.SchemaRuleIDXNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLNamingIndexConvention, nil
		}
	case advisor.SchemaRuleUKNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLNamingUKConvention, nil
		}
	case advisor.SchemaRuleFKNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLNamingFKConvention, nil
		}
	case advisor.SchemaRuleColumnNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLNamingColumnConvention, nil
		}
	case advisor.SchemaRuleRequiredColumn:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLColumnRequirement, nil
		}
	case advisor.SchemaRuleColumnNotNull:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLColumnNoNull, nil
		}
	case advisor.SchemaRuleTableRequirePK:
		switch engine {
		case db.MySQL, db.TiDB:
			return advisor.MySQLTableRequirePK, nil
		}
	case advisor.SchemaRuleMySQLEngine:
		if engine == db.MySQL {
			return advisor.MySQLUseInnoDB, nil
		}
	}
	return advisor.Fake, fmt.Errorf("unknown schema review rule type %v for %v", ruleType, engine)
}
