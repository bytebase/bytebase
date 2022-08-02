package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnNoNULL(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: `Column "id" in "public"."book" can not have NULL value`,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: `Column "name" in "public"."book" can not have NULL value`,
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int, name varchar(255), PRIMARY KEY (id))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: `Column "name" in "public"."book" can not have NULL value`,
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int PRIMARY KEY, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: `Column "name" in "public"."book" can not have NULL value`,
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int NOT NULL, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: `Column "name" in "public"."book" can not have NULL value`,
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int PRIMARY KEY, name varchar(255) NOT NULL)",
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
			Statement: "ALTER TABLE book ADD COLUMN id int",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: `Column "id" in "public"."book" can not have NULL value`,
				},
			},
		},
		{
			Statement: "ALTER TABLE book ADD COLUMN id int PRIMARY KEY, ADD COLUMN name varchar(255) NOT NULL",
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
			Statement: "ALTER TABLE book ALTER COLUMN id SET NOT NULL",
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
			Statement: "ALTER TABLE book ALTER COLUMN id DROP NOT NULL",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: `Column "id" in "public"."book" can not have NULL value`,
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &ColumnNoNullAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleColumnNotNull,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockPostgreSQLDatabase)
}
