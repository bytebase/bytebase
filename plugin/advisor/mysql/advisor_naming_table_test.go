package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingTableConvention(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE TABLE techBook(id int, name varchar(255))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
					Title:   "Mismatch table naming convention",
					Content: "`techBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id int, name varchar(255))",
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
			statement: "ALTER TABLE techBook RENAME TO TechBook",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
					Title:   "Mismatch table naming convention",
					Content: "`TechBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: "ALTER TABLE techBook RENAME TO tech_book",
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
			statement: "RENAME TABLE techBook TO tech_book, literaryBook TO LiteraryBook",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
					Title:   "Mismatch table naming convention",
					Content: "`LiteraryBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: "RENAME TABLE techBook TO TechBook, literaryBook TO LiteraryBook",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
					Title:   "Mismatch table naming convention",
					Content: "`TechBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Error,
					Code:    common.NamingTableConventionMismatch,
					Title:   "Mismatch table naming convention",
					Content: "`LiteraryBook` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: "RENAME TABLE techBook TO tech_book, literaryBook TO literary_book",
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
	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "^[a-z]+(_[a-z]+)*$",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingTableConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleTableNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	}, &MockCatalogService{})
}
