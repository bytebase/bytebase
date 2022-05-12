package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

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
					Content: "",
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
					Content: "",
				},
			},
		},
		{
			statement: "SELECT a FROM t",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.StatementNoWhere,
					Title:   "Require WHERE clause",
					Content: "\"SELECT a FROM t\" requires WHERE clause",
				},
			},
		},
		{
			statement: "SELECT a FROM t WHERE a > 0",
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}

	runSchemaReviewRuleTests(t, tests, &WhereRequirementAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleStatementRequireWhere,
		Level:   api.SchemaRuleLevelWarning,
		Payload: "",
	}, &MockCatalogService{})
}
