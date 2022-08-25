package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnDisallowChangingType(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: ``,
			Want:      []advisor.Advice{
				//	{
				//		Status:  ,
				//		Code:    ,
				//		Title:   ,
				//		Content: ,
				//		Line:    ,
				//	},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &ColumnDisallowChangingTypeAdvisor{}, &advisor.SQLReviewRule{
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
