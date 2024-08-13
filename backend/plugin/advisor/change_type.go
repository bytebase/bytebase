package advisor

import storepb "github.com/bytebase/bytebase/proto/generated-go/store"

var sqlEditorAllowlist = map[SQLReviewRuleType]bool{
	SchemaRuleStatementRequireWhere: true,
}

func isRuleAllowed(rule SQLReviewRuleType, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) bool {
	if changeType != storepb.PlanCheckRunConfig_SQL_EDITOR {
		return true
	}

	if _, ok := sqlEditorAllowlist[rule]; ok {
		return true
	}
	return false
}
