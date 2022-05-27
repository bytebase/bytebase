package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestNoSelectAll(t *testing.T) {
	tests := []test{
		{
			statement: "SELECT * FROM t",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.StatementSelectAll,
					Title:   "Not SELECT all",
					Content: "\"SELECT * FROM t\" uses SELECT all",
				},
			},
		},
		{
			statement: "SELECT a, b FROM t",
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
			statement: "SELECT a, b FROM (SELECT * from t1 JOIN t2) t",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.StatementSelectAll,
					Title:   "Not SELECT all",
					Content: "\"SELECT a, b FROM (SELECT * from t1 JOIN t2) t\" uses SELECT all",
				},
			},
		},
	}

	runSchemaReviewRuleTests(t, tests, &NoSelectAllAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleStatementNoSelectAll,
		Level:   api.SchemaRuleLevelError,
		Payload: "",
	}, &MockCatalogService{})
}
