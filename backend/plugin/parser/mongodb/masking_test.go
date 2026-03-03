package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeMaskingStatement(t *testing.T) {
	tests := []struct {
		description string
		statement   string
		want        *MaskingAnalysis
		wantError   bool
	}{
		{
			description: "find with dot access and direct predicate field",
			statement:   `db.users.find({email: "alice@example.com"})`,
			want: &MaskingAnalysis{
				API:             MaskableAPIFind,
				Operation:       "find",
				Collection:      "users",
				PredicateFields: []string{"email"},
			},
		},
		{
			description: "findOne with bracket access and dot path key",
			statement:   `db["users"].findOne({"contact.phone": "123"})`,
			want: &MaskingAnalysis{
				API:             MaskableAPIFindOne,
				Operation:       "findOne",
				Collection:      "users",
				PredicateFields: []string{"contact.phone"},
			},
		},
		{
			description: "find with getCollection access and logical operators",
			statement: `db.getCollection("users").find({
				$or: [
					{email: "a@example.com"},
					{contact: {phone: "123"}}
				],
				$and: [
					{"profile.ssn": "111"},
					{name: "alice"}
				],
				$nor: [{status: "inactive"}]
			})`,
			want: &MaskingAnalysis{
				API:        MaskableAPIFind,
				Operation:  "find",
				Collection: "users",
				PredicateFields: []string{
					"contact",
					"contact.phone",
					"email",
					"name",
					"profile.ssn",
					"status",
				},
			},
		},
		{
			description: "aggregate is unsupported read api",
			statement:   `db.users.aggregate([{$match: {name: "alice"}}])`,
			want: &MaskingAnalysis{
				API:        MaskableAPIUnsupportedRead,
				Operation:  "aggregate",
				Collection: "users",
			},
		},
		{
			description: "countDocuments is unsupported read api",
			statement:   `db.users.countDocuments({})`,
			want: &MaskingAnalysis{
				API:        MaskableAPIUnsupportedRead,
				Operation:  "countDocuments",
				Collection: "users",
			},
		},
		{
			description: "distinct is unsupported read api",
			statement:   `db.users.distinct("name")`,
			want: &MaskingAnalysis{
				API:        MaskableAPIUnsupportedRead,
				Operation:  "distinct",
				Collection: "users",
			},
		},
		{
			description: "write method is not relevant to masking analyzer",
			statement:   `db.users.insertOne({name: "alice"})`,
			want:        nil,
		},
		{
			description: "multiple statements are ignored",
			statement:   `db.users.find({name: "alice"}); db.users.findOne({name: "bob"})`,
			want:        nil,
		},
		{
			description: "parse error returns error",
			statement:   `db.users.find({`,
			wantError:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			got, err := AnalyzeMaskingStatement(tc.statement)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
