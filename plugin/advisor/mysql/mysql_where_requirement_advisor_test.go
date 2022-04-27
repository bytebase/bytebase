package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func runSchemaReviewRuleTests(t *testing.T, tests []test, adv advisor.Advisor, rule *api.SchemaReviewRule) {
	logger, _ := zap.NewDevelopmentConfig().Build()
	ctx := advisor.Context{
		Logger:    logger,
		Charset:   "",
		Collation: "",
		Rule:      rule,
	}
	for _, tc := range tests {
		adviceList, err := adv.Check(ctx, tc.statement)
		require.NoError(t, err)
		assert.Equal(t, tc.want, adviceList)
	}
}

func TestWhereRequirement(t *testing.T) {
	tests := []test{
		{
			statement: "DELETE FROM t1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.StatementNoWhere,
					Title:   "Require WHERE clause",
					Content: "\"DELETE FROM t1\" requires WHERE clause",
				},
			},
		},
		{
			statement: "UPDATE t1 SET a = 1",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.StatementNoWhere,
					Title:   "Require WHERE clause",
					Content: "\"UPDATE t1 SET a = 1\" requires WHERE clause",
				},
			},
		},
		{
			statement: "DELETE FROM t1 WHERE a > 0",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "Pass rule: statements require 'WHERE' clause",
				},
			},
		},
		{
			statement: "UPDATE t1 SET a = 1 WHERE a > 10",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "Pass rule: statements require 'WHERE' clause",
				},
			},
		},
	}

	runSchemaReviewRuleTests(t, tests, &WhereRequirementAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleStatementRequireWhere,
		Level:   api.SchemaRuleLevelWarning,
		Payload: "",
	})
}
