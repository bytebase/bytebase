package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnNoNull(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`id` can not have NULL value",
				},
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` can not have NULL value",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int PRIMARY KEY, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` can not have NULL value",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int NOT NULL, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` can not have NULL value",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int PRIMARY KEY, name varchar(255) NOT NULL)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE book ADD COLUMN (id int, name varchar(255) NOT NULL)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`id` can not have NULL value",
				},
			},
		},
		{
			Statement: "ALTER TABLE book ADD COLUMN (id int PRIMARY KEY, name varchar(255) NOT NULL)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE book CHANGE COLUMN id name varchar(255)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` can not have NULL value",
				},
			},
		},
		{
			Statement: "ALTER TABLE book CHANGE COLUMN id name varchar(255) NOT NULL",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}

	advisor.RunSchemaReviewRuleTests(t, tests, &ColumnNoNullAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleColumnNotNull,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, &advisor.MockCatalogService{})
}
