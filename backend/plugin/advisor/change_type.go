package advisor

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

var sqlEditorAllowlist = map[SQLReviewRuleType]bool{
	SchemaRuleStatementRequireWhereForSelect: true,
}

func isRuleAllowed(rule SQLReviewRuleType, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) bool {
	switch changeType {
	case storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED:
		return false
	case storepb.PlanCheckRunConfig_DDL:
		return true
	case storepb.PlanCheckRunConfig_DDL_GHOST:
		return true
	case storepb.PlanCheckRunConfig_DML:
		return true
	case storepb.PlanCheckRunConfig_SDL:
		return true
	case storepb.PlanCheckRunConfig_SQL_EDITOR:
		return sqlEditorAllowlist[rule]
	default:
		return false
	}
}
