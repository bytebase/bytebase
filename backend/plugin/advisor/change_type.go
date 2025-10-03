package advisor

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

var sqlEditorAllowlist = map[SQLReviewRuleType]bool{
	SchemaRuleStatementRequireWhereForSelect: true,
}

func isRuleAllowed(rule SQLReviewRuleType, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) bool {
	if changeType == storepb.PlanCheckRunConfig_SQL_EDITOR {
		return sqlEditorAllowlist[rule]
	}
	return true
}
