package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/api"
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
					Title:   "No leading wildcard LIKE",
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
					Title:   "No leading wildcard LIKE",
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
					Title:   "No leading wildcard LIKE",
					Content: "\"SELECT * FROM t WHERE a LIKE '%acc' OR a LIKE '%abc'\" uses leading wildcard LIKE",
				},
			},
		},
	}

	runSchemaReviewRuleTests(t, tests, &NoLeadingWildcardLikeAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleStatementNoLeadingWildcardLike,
		Level:   api.SchemaRuleLevelError,
		Payload: "",
	}, &MockCatalogService{})
}
