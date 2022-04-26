package server

import (
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

// Schema review policy consists of a list of schema review rules.
// There is such a logical mapping in bytebase backend:
//                    this file     server/task_check_executor_statement_advisor.go
//                        |                        |
// schema review rule ----------> TaskCheckType ------> plugin.AdvisorType --+
//                                                                           +----> advisors
// DB type ------------------------------------------------------------------+  |
//                                                                        plugin/advisor.go
//
// But for unimplemented advisors, we should not generate corresponding TaskChecks.
// So we should also check DB type here.

// getTaskCheckTypeAndPayloadByRule gets the corresponding TaskCheckType and payload for each specific SchemaReviewRule.
func getTaskCheckTypeAndPayloadByRule(rule *api.SchemaReviewRule, base api.TaskCheckDatabaseStatementAdvisePayload) (api.TaskCheckType, string, error) {
	switch rule.Type {
	case api.SchemaRuleQueryRequireWhere:
		if base.DbType != db.MySQL && base.DbType != db.TiDB {
			return "", "", fmt.Errorf("schema review rule %v dosen't support %v", rule.Type, base.DbType)
		}
		base.Level = rule.Level
		payload, err := json.Marshal(base)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal statement advise payload: %v, err: %w", api.TaskCheckDatabaseStatementRequireWhere, err)
		}
		return api.TaskCheckDatabaseStatementRequireWhere, string(payload), nil
	}
	return "", "", fmt.Errorf("unknown schema review rule type %v", rule.Type)
}
