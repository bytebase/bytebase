package server

import (
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor"
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

func getAdvisorTypeByRule(ruleType api.SchemaReviewRuleType) (advisor.Type, error) {
	switch ruleType {
	case api.SchemaRuleStatementRequireWhere:
		return advisor.MySQLWhereRequirement, nil
	}
	return advisor.Fake, fmt.Errorf("unknown schema review rule type %v", ruleType)
}
