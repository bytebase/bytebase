package advisor

import storepb "github.com/bytebase/bytebase/proto/generated-go/store"

const (
	BuiltinRulePriorBackupCheck SQLReviewRuleType = "builtin.prior-backup-check"
	BuiltinRuleObjectOwnerCheck SQLReviewRuleType = "builtin.object-owner-check"
)

func GetBuiltinRules(engine storepb.Engine) []*storepb.SQLReviewRule {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_ORACLE:
		return []*storepb.SQLReviewRule{
			{
				Type:    string(BuiltinRulePriorBackupCheck),
				Level:   storepb.SQLReviewRuleLevel_ERROR,
				Payload: "",
				Engine:  engine,
				Comment: "",
			},
		}
	case storepb.Engine_POSTGRES:
		return []*storepb.SQLReviewRule{
			{
				Type:    string(BuiltinRulePriorBackupCheck),
				Level:   storepb.SQLReviewRuleLevel_ERROR,
				Payload: "",
				Engine:  engine,
				Comment: "",
			},
			{
				Type:    string(BuiltinRuleObjectOwnerCheck),
				Level:   storepb.SQLReviewRuleLevel_WARNING,
				Payload: "",
				Engine:  engine,
				Comment: "",
			},
		}
	default:
		return nil
	}
}
