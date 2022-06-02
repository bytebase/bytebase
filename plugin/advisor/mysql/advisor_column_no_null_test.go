package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnNoNull(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE TABLE book(id int, name varchar(255))",
			want: []advisor.Advice{
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
			statement: "CREATE TABLE book(id int PRIMARY KEY, name varchar(255))",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` can not have NULL value",
				},
			},
		},
		{
			statement: "CREATE TABLE book(id int NOT NULL, name varchar(255))",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` can not have NULL value",
				},
			},
		},
		{
			statement: "CREATE TABLE book(id int PRIMARY KEY, name varchar(255) NOT NULL)",
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
			statement: "ALTER TABLE book ADD COLUMN (id int, name varchar(255) NOT NULL)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`id` can not have NULL value",
				},
			},
		},
		{
			statement: "ALTER TABLE book ADD COLUMN (id int PRIMARY KEY, name varchar(255) NOT NULL)",
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
			statement: "ALTER TABLE book CHANGE COLUMN id name varchar(255)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.ColumnCanNotNull,
					Title:   "column.no-null",
					Content: "`book`.`name` can not have NULL value",
				},
			},
		},
		{
			statement: "ALTER TABLE book CHANGE COLUMN id name varchar(255) NOT NULL",
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

	runSchemaReviewRuleTests(t, tests, &ColumnNoNullAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleColumnNotNull,
		Level:   api.SchemaRuleLevelWarning,
		Payload: "",
	}, &MockCatalogService{})
}
