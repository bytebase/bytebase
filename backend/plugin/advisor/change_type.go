package advisor

import storepb "github.com/bytebase/bytebase/proto/generated-go/store"

var ChangeTypeWhiteListForRules = map[SQLReviewRuleType][]storepb.PlanCheckRunConfig_ChangeDatabaseType{
	SchemaRuleStatementCheckSetRoleVariable: {storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED},
}

// SkipRuleInChangeType will skip the sql review check for rule with specific change type.
func SkipRuleInChangeType(rule SQLReviewRuleType, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) bool {
	whitelist, ok := ChangeTypeWhiteListForRules[rule]
	if !ok {
		return false
	}

	for _, w := range whitelist {
		if w == changeType {
			return true
		}
	}
	return false
}
