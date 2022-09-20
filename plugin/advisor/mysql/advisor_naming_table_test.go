package mysql

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
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
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book_copy(id int, name varchar(255))",
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
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book RENAME TO TechBook",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`TechBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
			},
		},
		{
			Statement: "ALTER TABLE tech_book RENAME TO tech_book_copy",
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
			Statement: "RENAME TABLE tech_book TO tech_book_copy, tech_book_copy TO LiteraryBook",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`LiteraryBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE literary_book(a int);RENAME TABLE tech_book TO TechBook, literary_book TO LiteraryBook",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`TechBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`LiteraryBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE literary_book(a int);RENAME TABLE tech_book TO tech_book_copy, literary_book TO literary_book_copy",
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
