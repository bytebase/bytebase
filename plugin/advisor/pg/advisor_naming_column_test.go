//go:build !release
// +build !release

package pg

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingColumnConvention(t *testing.T) {
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
		Format: "^[a-z]+(_[a-z]+)*$",
	})
	require.NoError(t, err)
	advisor.RunSchemaReviewRuleTests(t, tests, &NamingColumnConventionAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleColumnNaming,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, &advisor.MockCatalogService{})
}
