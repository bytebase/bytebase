package server

import (
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor"
)

// Schema review policy consists of a list of schema review rules.
// There is such a logical mapping in bytebase backend:
//   1. One schema review policy maps a TaskCheckRun.
//   2. Each schema reivew rule type maps an advisor.Type.
//   3. Each [db.Type][AdvisorType] maps an advisor.
//
// How to add a schema review rule:
//   1. Implement an advisor.(plugin/xxx)
//   2. Register this advisor in map[db.Type][AdvisorType].(plugin/advisor.go)
//   3. Map SchemaReviewRuleType to advisor.Type in this file.

func getAdvisorTypeByRule(ruleType api.SchemaReviewRuleType) (advisor.Type, error) {
	switch ruleType {
	case api.SchemaRuleStatementRequireWhere:
		return advisor.MySQLWhereRequirement, nil
	}
	return advisor.Fake, fmt.Errorf("unknown schema review rule type %v", ruleType)
}
