package pg

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingColumnConvention(t *testing.T) {
	invalidColumnName := advisor.RandomString(33)

	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int, \"creatorId\" int)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "\"book\".\"creatorId\" mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: fmt.Sprintf("CREATE TABLE book(id int, %s int)", invalidColumnName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: fmt.Sprintf("\"book\".\"%s\" mismatches column naming convention, its length should be within 32 characters", invalidColumnName),
				},
			},
		},
		{
			// PostgreSQL uses lowercase by default unless using double quotes.
			Statement: "CREATE TABLE book(id int, creator_Id int)",
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
			Statement: "CREATE TABLE book(id int, creator_id int)",
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
			Statement: `CREATE TABLE book(id int, creator_id int);
						ALTER TABLE book ADD COLUMN "creatorId" int`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "\"book\".\"creatorId\" mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(id int, creator_id int);
						ALTER TABLE book ADD COLUMN "creator" int`,
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
			Statement: `CREATE TABLE book(id int, creator_id int);
						ALTER TABLE book RENAME COLUMN creator_id TO "creatorId"`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "\"book\".\"creatorId\" mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(id int, creator_id int);
						ALTER TABLE book RENAME COLUMN creator_id TO "creator"`,
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
		MaxLength: 32,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingColumnConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleColumnNaming,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
