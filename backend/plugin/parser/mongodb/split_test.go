package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitSQL(t *testing.T) {
	testCases := []struct {
		description string
		statement   string
		wantCount   int
		wantTexts   []string
		wantNil     bool
	}{
		{
			description: "single statement",
			statement:   `db.users.find({})`,
			wantCount:   1,
			wantTexts:   []string{`db.users.find({})`},
		},
		{
			description: "two statements with semicolons",
			statement:   `db.users.find({});db.products.find({})`,
			wantCount:   2,
			wantTexts:   []string{`db.users.find({});`, `db.products.find({})`},
		},
		{
			description: "two statements with newlines",
			statement:   "db.users.find({})\ndb.products.find({})",
			wantCount:   2,
			wantTexts:   []string{`db.users.find({})`, `db.products.find({})`},
		},
		{
			description: "two statements with semicolon and newline",
			statement:   "db.users.find({});\ndb.products.insertOne({name: \"test\"})",
			wantCount:   2,
			wantTexts:   []string{"db.users.find({});", `db.products.insertOne({name: "test"})`},
		},
		{
			description: "three statements mixed separators",
			statement:   "show dbs;\ndb.users.find({})\ndb.products.drop()",
			wantCount:   3,
			wantTexts:   []string{"show dbs;", "db.users.find({})", "db.products.drop()"},
		},
		{
			description: "single statement with trailing semicolon",
			statement:   `db.users.find({});`,
			wantCount:   1,
			wantTexts:   []string{`db.users.find({});`},
		},
		{
			description: "empty input",
			statement:   "",
			wantNil:     true,
		},
		{
			description: "whitespace only",
			statement:   "   \n\t  ",
			wantNil:     true,
		},
		{
			description: "show commands",
			statement:   "show dbs\nshow collections",
			wantCount:   2,
			wantTexts:   []string{"show dbs", "show collections"},
		},
		{
			description: "multiline aggregate is one statement",
			statement: `db.users.aggregate([
  { $match: { status: "A" } },
  { $group: { _id: "$cust_id" } }
])`,
			wantCount: 1,
		},
		{
			description: "method chaining is one statement",
			statement:   `db.users.find({}).sort({ name: 1 }).limit(10)`,
			wantCount:   1,
		},
		{
			description: "bracket notation collection",
			statement:   `db["my-collection"].find({})`,
			wantCount:   1,
			wantTexts:   []string{`db["my-collection"].find({})`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, err := SplitSQL(tc.statement)
			require.NoError(t, err)

			if tc.wantNil {
				require.Nil(t, result)
				return
			}

			require.Len(t, result, tc.wantCount)

			if tc.wantTexts != nil {
				for i, wantText := range tc.wantTexts {
					require.Equal(t, wantText, result[i].Text, "statement %d text mismatch", i)
				}
			}

			// Verify all statements have position info.
			for i, stmt := range result {
				require.NotNil(t, stmt.Start, "statement %d should have Start", i)
				require.NotNil(t, stmt.End, "statement %d should have End", i)
				require.NotNil(t, stmt.Range, "statement %d should have Range", i)
				require.Greater(t, stmt.Range.End, stmt.Range.Start, "statement %d Range.End should be > Range.Start", i)
			}
		})
	}
}
