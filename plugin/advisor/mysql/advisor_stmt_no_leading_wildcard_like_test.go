package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestNoLeadingWildcardLike(t *testing.T) {
	tests := []test{
		{
			statement: "SELECT * FROM t WHERE a LIKE 'abc%'",
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
			statement: "SELECT * FROM t WHERE a LIKE '%abc'",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.StatementLeadingWildcardLike,
					Title:   "statement.where.no-leading-wildcard-like",
					Content: "\"SELECT * FROM t WHERE a LIKE '%abc'\" uses leading wildcard LIKE",
				},
			},
		},
		{
			statement: "SELECT * FROM t WHERE a LIKE 'abc' OR a LIKE '%abc'",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.StatementLeadingWildcardLike,
					Title:   "statement.where.no-leading-wildcard-like",
					Content: "\"SELECT * FROM t WHERE a LIKE 'abc' OR a LIKE '%abc'\" uses leading wildcard LIKE",
				},
			},
		},
		{
			statement: "SELECT * FROM t WHERE a LIKE '%acc' OR a LIKE '%abc'",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.StatementLeadingWildcardLike,
					Title:   "statement.where.no-leading-wildcard-like",
					Content: "\"SELECT * FROM t WHERE a LIKE '%acc' OR a LIKE '%abc'\" uses leading wildcard LIKE",
				},
			},
		},
		{
			statement: "SELECT * FROM (SELECT * FROM t WHERE a LIKE '%acc' OR a LIKE '%abc') t1",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.StatementLeadingWildcardLike,
					Title:   "statement.where.no-leading-wildcard-like",
					Content: "\"SELECT * FROM (SELECT * FROM t WHERE a LIKE '%acc' OR a LIKE '%abc') t1\" uses leading wildcard LIKE",
				},
			},
		},
	}

	runSchemaReviewRuleTests(t, tests, &NoLeadingWildcardLikeAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleStatementNoLeadingWildcardLike,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, &MockCatalogService{})
}
