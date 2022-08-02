package mysql

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestMySQLNamingTableConvention(t *testing.T) {
	invalidTableName := advisor.RandomString(65)

	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE techBook(id int, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`techBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id int, name varchar(255))",
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
			Statement: fmt.Sprintf("CREATE TABLE %s(id int, name varchar(255))", invalidTableName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: fmt.Sprintf("`%s` mismatches table naming convention, its length should be within 64 characters", invalidTableName),
				},
			},
		},
		{
			Statement: "ALTER TABLE techBook RENAME TO TechBook",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`TechBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: "ALTER TABLE techBook RENAME TO tech_book",
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
			Statement: "RENAME TABLE techBook TO tech_book, literaryBook TO LiteraryBook",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`LiteraryBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: "RENAME TABLE techBook TO TechBook, literaryBook TO LiteraryBook",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`TechBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`LiteraryBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: "RENAME TABLE techBook TO tech_book, literaryBook TO literary_book",
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
	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^[a-z]+(_[a-z]+)*$",
		MaxLength: 64,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingTableConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleTableNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockMySQLDatabase)
}
