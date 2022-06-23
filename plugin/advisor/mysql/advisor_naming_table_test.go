package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestMySQLNamingTableConvention(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE techBook(id int, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
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
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: "ALTER TABLE techBook RENAME TO TechBook",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
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
					Code:    common.Ok,
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
					Code:    common.NamingTableConventionMismatch,
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
					Code:    common.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "`TechBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
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
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}
	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format: "^[a-z]+(_[a-z]+)*$",
	})
	require.NoError(t, err)
	advisor.RunSchemaReviewRuleTests(t, tests, &NamingTableConventionAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleTableNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, &advisor.MockCatalogService{})
}
