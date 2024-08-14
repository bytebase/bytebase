package advisor

import storepb "github.com/bytebase/bytebase/proto/generated-go/store"

const (
	BuiltinRulePriorBackupCheck SQLReviewRuleType = "builtin.prior-backup-check"
)

func GetBuiltinRules(engine storepb.Engine) []*storepb.SQLReviewRule {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_POSTGRES:
		return []*storepb.SQLReviewRule{
			{
				Type:    string(BuiltinRulePriorBackupCheck),
				Level:   storepb.SQLReviewRuleLevel_ERROR,
				Payload: "",
				Engine:  engine,
				Comment: "",
			},
		}
	default:
		return nil
	}
}
