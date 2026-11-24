package advisor

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// BuiltinWalkThroughCheckAdvisor is a no-op advisor for BUILTIN_WALK_THROUGH_CHECK.
// The actual walk-through logic runs in the dedicated block in SQLReviewCheck
// before the rule loop. This advisor exists only so the rule can be registered
// in the advisor registry without causing "unknown advisor" errors.
type BuiltinWalkThroughCheckAdvisor struct{}

func (*BuiltinWalkThroughCheckAdvisor) Check(_ context.Context, _ Context) ([]*storepb.Advice, error) {
	return nil, nil
}

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
