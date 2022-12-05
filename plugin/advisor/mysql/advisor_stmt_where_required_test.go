package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestWhereRequirement(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "INSERT INTO tech_book(id) values (1)",
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
			Statement: "DELETE FROM tech_book",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.StatementNoWhere,
					Title:   "statement.where.require",
					Content: "\"DELETE FROM tech_book\" requires WHERE clause",
					Line:    1,
				},
			},
		},
		{
			Statement: "UPDATE tech_book SET id = 1",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.StatementNoWhere,
					Title:   "statement.where.require",
					Content: "\"UPDATE tech_book SET id = 1\" requires WHERE clause",
					Line:    1,
				},
			},
		},
		{
			Statement: "DELETE FROM tech_book WHERE id > 0",
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
			Statement: "UPDATE tech_book SET id = 1 WHERE id > 10",
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
			Statement: "SELECT id FROM tech_book",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.StatementNoWhere,
					Title:   "statement.where.require",
					Content: "\"SELECT id FROM tech_book\" requires WHERE clause",
					Line:    1,
				},
			},
		},
		{
			Statement: "SELECT id FROM tech_book WHERE id > 0",
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
			Statement: "SELECT id FROM tech_book WHERE id > (SELECT max(id) FROM tech_book)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.StatementNoWhere,
					Title:   "statement.where.require",
					Content: "\"SELECT id FROM tech_book WHERE id > (SELECT max(id) FROM tech_book)\" requires WHERE clause",
					Line:    1,
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &WhereRequirementAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleStatementRequireWhere,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
