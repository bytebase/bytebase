package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestNoSelectAll(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "SELECT * FROM t",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.StatementSelectAll,
					Title:   "statement.select.no-select-all",
					Content: "\"SELECT * FROM t\" uses SELECT all",
				},
			},
		},
		{
			Statement: "SELECT a, b FROM t",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "SELECT a, b FROM (SELECT * from t1 JOIN t2) t",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.StatementSelectAll,
					Title:   "statement.select.no-select-all",
					Content: "\"SELECT a, b FROM (SELECT * from t1 JOIN t2) t\" uses SELECT all",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &NoSelectAllAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleStatementNoSelectAll,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
