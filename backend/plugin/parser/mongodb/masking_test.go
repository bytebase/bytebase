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
			description: "aggregate with shape-preserving stages",
			statement:   `db.users.aggregate([{$match: {name: "alice"}}])`,
			want: &MaskingAnalysis{
				API:             MaskableAPIAggregate,
				Operation:       "aggregate",
				Collection:      "users",
				PredicateFields: []string{"name"},
			},
		},
		{
			description: "aggregate with multiple shape-preserving stages",
			statement:   `db.users.aggregate([{$match: {status: "active"}}, {$sort: {name: 1}}, {$limit: 10}])`,
			want: &MaskingAnalysis{
				API:             MaskableAPIAggregate,
				Operation:       "aggregate",
				Collection:      "users",
				PredicateFields: []string{"status"},
			},
		},
		{
			description: "aggregate match with logical operators",
			statement:   `db.users.aggregate([{$match: {$or: [{age: {$gt: 18}}, {name: "alice"}]}}])`,
			want: &MaskingAnalysis{
				API:             MaskableAPIAggregate,
				Operation:       "aggregate",
				Collection:      "users",
				PredicateFields: []string{"age", "name"},
			},
		},
		{
			description: "aggregate with addFields and unset",
			statement:   `db.users.aggregate([{$addFields: {fullName: "test"}}, {$unset: "ssn"}])`,
			want: &MaskingAnalysis{
				API:        MaskableAPIAggregate,
				Operation:  "aggregate",
				Collection: "users",
			},
		},
		{
			description: "aggregate with empty pipeline",
			statement:   `db.users.aggregate([])`,
			want: &MaskingAnalysis{
				API:        MaskableAPIAggregate,
				Operation:  "aggregate",
				Collection: "users",
			},
		},
		{
			description: "aggregate with no arguments",
			statement:   `db.users.aggregate()`,
			want: &MaskingAnalysis{
				API:        MaskableAPIAggregate,
				Operation:  "aggregate",
				Collection: "users",
			},
		},
		{
			description: "aggregate with $group is unsupported",
			statement:   `db.users.aggregate([{$group: {_id: "$status"}}])`,
			want: &MaskingAnalysis{
				API:              MaskableAPIUnsupportedRead,
				Operation:        "aggregate",
				Collection:       "users",
				UnsupportedStage: "$group",
			},
		},
		{
			description: "aggregate with $project is unsupported",
			statement:   `db.users.aggregate([{$match: {name: "alice"}}, {$project: {name: 1}}])`,
			want: &MaskingAnalysis{
				API:              MaskableAPIUnsupportedRead,
				Operation:        "aggregate",
				Collection:       "users",
				UnsupportedStage: "$project",
			},
		},
		{
			description: "aggregate with $lookup basic form is supported",
			statement:   `db.users.aggregate([{$lookup: {from: "orders", localField: "_id", foreignField: "userId", as: "orders"}}])`,
			want: &MaskingAnalysis{
				API:        MaskableAPIAggregate,
				Operation:  "aggregate",
				Collection: "users",
				JoinedCollections: []JoinedCollection{
					{AsField: "orders", Collection: "orders"},
				},
			},
		},
		{
			description: "aggregate with $lookup pipeline form is unsupported",
			statement:   `db.users.aggregate([{$lookup: {from: "orders", pipeline: [{$match: {status: "active"}}], as: "orders"}}])`,
			want: &MaskingAnalysis{
				API:              MaskableAPIUnsupportedRead,
				Operation:        "aggregate",
				Collection:       "users",
				UnsupportedStage: "$lookup",
			},
		},
		{
			description: "aggregate with $graphLookup is supported",
			statement:   `db.users.aggregate([{$graphLookup: {from: "employees", startWith: "$reportsTo", connectFromField: "reportsTo", connectToField: "name", as: "reportingHierarchy"}}])`,
			want: &MaskingAnalysis{
				API:        MaskableAPIAggregate,
				Operation:  "aggregate",
				Collection: "users",
				JoinedCollections: []JoinedCollection{
					{AsField: "reportingHierarchy", Collection: "employees"},
				},
			},
		},
		{
			description: "aggregate with $out is unsupported",
			statement:   `db.users.aggregate([{$match: {status: "active"}}, {$out: "activeUsers"}])`,
			want: &MaskingAnalysis{
				API:              MaskableAPIUnsupportedRead,
				Operation:        "aggregate",
				Collection:       "users",
				UnsupportedStage: "$out",
			},
		},
		{
			description: "aggregate with $unwind is shape-preserving",
			statement:   `db.users.aggregate([{$unwind: "$tags"}])`,
			want: &MaskingAnalysis{
				API:        MaskableAPIAggregate,
				Operation:  "aggregate",
				Collection: "users",
			},
		},
		{
			description: "aggregate with $match and $unwind",
			statement:   `db.users.aggregate([{$match: {status: "active"}}, {$unwind: "$tags"}])`,
			want: &MaskingAnalysis{
				API:             MaskableAPIAggregate,
				Operation:       "aggregate",
				Collection:      "users",
				PredicateFields: []string{"status"},
			},
		},
		{
			description: "aggregate with $replaceRoot is unsupported",
			statement:   `db.users.aggregate([{$replaceRoot: {newRoot: "$contact"}}])`,
			want: &MaskingAnalysis{
				API:              MaskableAPIUnsupportedRead,
				Operation:        "aggregate",
				Collection:       "users",
				UnsupportedStage: "$replaceRoot",
			},
		},
		{
			description: "aggregate with $count is unsupported",
			statement:   `db.users.aggregate([{$count: "total"}])`,
			want: &MaskingAnalysis{
				API:              MaskableAPIUnsupportedRead,
				Operation:        "aggregate",
				Collection:       "users",
				UnsupportedStage: "$count",
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
