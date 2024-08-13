package advisor

import storepb "github.com/bytebase/bytebase/proto/generated-go/store"

var sqlEditorAllowlist = map[SQLReviewRuleType]bool{
	SchemaRuleStatementRequireWhere: true,
}

// skipRuleInSQLEditor will skip the sql review check in SQL Editor.
func skipRuleInSQLEditor(rule SQLReviewRuleType, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) bool {
	if changeType != storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED {
		return false
	}

	if _, ok := sqlEditorAllowlist[rule]; ok {
		return false
	}
	return true
}
