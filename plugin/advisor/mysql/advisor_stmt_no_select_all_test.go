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
					Title:   "No SELECT all",
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
	}

	runSchemaReviewRuleTests(t, tests, &NoSelectAllAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleStatementNoSelectAll,
		Level:   api.SchemaRuleLevelError,
		Payload: "",
	}, &MockCatalogService{})
}
