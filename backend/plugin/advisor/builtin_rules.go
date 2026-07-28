package advisor

import (
	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func GetBuiltinRules(engine storepb.Engine) []*storepb.SQLReviewRule {
	rules := []*storepb.SQLReviewRule{
		{
			Type:   storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: engine,
			Payload: &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: common.MaxSheetCheckSize,
				},
			},
		},
	}

	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_POSTGRES, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_ORACLE:
		rules = append(rules, &storepb.SQLReviewRule{
			Type:   storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: engine,
		})
	default:
	}

	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_TIDB, storepb.Engine_POSTGRES, storepb.Engine_OCEANBASE:
		rules = append(rules, &storepb.SQLReviewRule{
			Type:   storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: engine,
		})
	default:
	}

	return rules
}
