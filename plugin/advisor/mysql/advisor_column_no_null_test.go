package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnNoNull(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: `CREATE TABLE book(
				id int,
				name varchar(255))`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCannotNull,
					Title:   "column.no-null",
					Content: "`book`.`id` cannot have NULL value",
					Line:    2,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCannotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` cannot have NULL value",
					Line:    3,
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int PRIMARY KEY, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCannotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` cannot have NULL value",
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int NOT NULL, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCannotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` cannot have NULL value",
					Line:    1,
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
			Statement: `
				CREATE TABLE book(a int NOT NULL);
				ALTER TABLE book ADD COLUMN (id int, name varchar(255) NOT NULL);
				`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCannotNull,
					Title:   "column.no-null",
					Content: "`book`.`id` cannot have NULL value",
					Line:    3,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE book(a int NOT NULL);
				ALTER TABLE book ADD COLUMN (id int PRIMARY KEY, name varchar(255) NOT NULL);
			`,
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
			Statement: `
				CREATE TABLE book(id int NOT NULL);
				ALTER TABLE book CHANGE COLUMN id name varchar(255);
				`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.ColumnCannotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` cannot have NULL value",
					Line:    3,
				},
			},
		},
		{
			Statement: `
				CREATE TABLE book(id int NOT NULL);
				ALTER TABLE book CHANGE COLUMN id name varchar(255) NOT NULL;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &ColumnNoNullAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleColumnNotNull,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
