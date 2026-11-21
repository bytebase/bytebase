package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeMaskingStatementSeam(t *testing.T) {
	// OpFind -> MaskableAPIFind with sorted PredicateFields.
	got, err := AnalyzeMaskingStatement(`db.users.find({name: "alice", email: "alice@example.com"})`)
	require.NoError(t, err)
	require.Equal(t, &MaskingAnalysis{
		API:             MaskableAPIFind,
		Operation:       "find",
		Collection:      "users",
		PredicateFields: []string{"email", "name"},
	}, got)

	// OpFindOne -> MaskableAPIFindOne.
	got, err = AnalyzeMaskingStatement(`db["users"].findOne({"contact.phone": "123"})`)
	require.NoError(t, err)
	require.Equal(t, &MaskingAnalysis{
		API:             MaskableAPIFindOne,
		Operation:       "findOne",
		Collection:      "users",
		PredicateFields: []string{"contact.phone"},
	}, got)

	// OpAggregate with $lookup -> JoinedCollections conversion.
	got, err = AnalyzeMaskingStatement(`db.users.aggregate([{$lookup: {from: "orders", localField: "_id", foreignField: "userId", as: "orders"}}])`)
	require.NoError(t, err)
	require.Equal(t, &MaskingAnalysis{
		API:        MaskableAPIAggregate,
		Operation:  "aggregate",
		Collection: "users",
		JoinedCollections: []JoinedCollection{
			{Collection: "orders", AsField: "orders"},
		},
	}, got)

	// OpAggregate with unsupported stage -> MaskableAPIUnsupportedRead.
	got, err = AnalyzeMaskingStatement(`db.users.aggregate([{$group: {_id: "$status"}}])`)
	require.NoError(t, err)
	require.Equal(t, &MaskingAnalysis{
		API:              MaskableAPIUnsupportedRead,
		Operation:        "aggregate",
		Collection:       "users",
		UnsupportedStage: "$group",
	}, got)

	// Other read op -> MaskableAPIUnsupportedRead.
	got, err = AnalyzeMaskingStatement(`db.users.countDocuments({})`)
	require.NoError(t, err)
	require.Equal(t, MaskableAPIUnsupportedRead, got.API)

	// Non-read and multi-statement return nil, nil; syntax error propagates.
	got, err = AnalyzeMaskingStatement(`db.users.insertOne({name: "alice"})`)
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = AnalyzeMaskingStatement(`db.users.find({}); db.users.findOne({})`)
	require.NoError(t, err)
	require.Nil(t, got)

	_, err = AnalyzeMaskingStatement(`db.users.find({`)
	require.Error(t, err)
}
