package pg

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestPostgreSQLNamingTableConvention(t *testing.T) {
	invalidTableName := advisor.RandomString(33)

	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE \"techBook\"(id int, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "\"techBook\" mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
			},
		},
		{
			Statement: fmt.Sprintf("CREATE TABLE \"%s\"(id int, name varchar(255))", invalidTableName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within 32 characters", invalidTableName),
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE _techBook(id int, name varchar(255))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "\"_techbook\" mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
			},
		},
		{
			// PostgreSQL uses lowercase by default unless using double quotes.
			Statement: "CREATE TABLE techBook(id int, name varchar(255))",
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
			Statement: "ALTER TABLE tech_book RENAME TO \"TechBook\"",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "\"TechBook\" mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
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
			Statement: `CREATE TABLE _techBook(id int, name varchar(255));
						ALTER TABLE tech_book RENAME TO "TechBook";`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "\"_techbook\" mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingTableConventionMismatch,
					Title:   "naming.table",
					Content: "\"TechBook\" mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    2,
				},
			},
		},
	}
	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^[a-z]+(_[a-z]+)*$",
		MaxLength: 32,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingTableConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleTableNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
