package advisor

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

func GetBuiltinRules(engine storepb.Engine) []*storepb.SQLReviewRule {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_POSTGRES, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_ORACLE:
		return []*storepb.SQLReviewRule{
			{
				Type:   storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
				Level:  storepb.SQLReviewRule_WARNING,
				Engine: engine,
			},
		}
	default:
		return nil
	}
}
