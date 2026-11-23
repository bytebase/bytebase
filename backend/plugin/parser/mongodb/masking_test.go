package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestAnalyzeMaskingStatementSeam(t *testing.T) {
	// OpFind -> MongoDBMaskableAPIFind with sorted PredicateFields.
	got, err := analyzeMaskingStatement(`db.users.find({name: "alice", email: "alice@example.com"})`)
	require.NoError(t, err)
	require.Equal(t, &base.MongoDBAnalysis{
		API:             base.MongoDBMaskableAPIFind,
		Operation:       "find",
		Collection:      "users",
		PredicateFields: []string{"email", "name"},
	}, got)

	// OpFindOne -> MongoDBMaskableAPIFindOne.
	got, err = analyzeMaskingStatement(`db["users"].findOne({"contact.phone": "123"})`)
	require.NoError(t, err)
	require.Equal(t, &base.MongoDBAnalysis{
		API:             base.MongoDBMaskableAPIFindOne,
		Operation:       "findOne",
		Collection:      "users",
		PredicateFields: []string{"contact.phone"},
	}, got)

	// OpAggregate with $lookup -> JoinedCollections conversion.
	got, err = analyzeMaskingStatement(`db.users.aggregate([{$lookup: {from: "orders", localField: "_id", foreignField: "userId", as: "orders"}}])`)
	require.NoError(t, err)
	require.Equal(t, &base.MongoDBAnalysis{
		API:        base.MongoDBMaskableAPIAggregate,
		Operation:  "aggregate",
		Collection: "users",
		JoinedCollections: []base.MongoDBJoinedCollection{
			{Collection: "orders", AsField: "orders"},
		},
	}, got)

	// OpAggregate with unsupported stage -> MongoDBMaskableAPIUnsupportedRead.
	got, err = analyzeMaskingStatement(`db.users.aggregate([{$group: {_id: "$status"}}])`)
	require.NoError(t, err)
	require.Equal(t, &base.MongoDBAnalysis{
		API:              base.MongoDBMaskableAPIUnsupportedRead,
		Operation:        "aggregate",
		Collection:       "users",
		UnsupportedStage: "$group",
	}, got)

	// Other read op -> MongoDBMaskableAPIUnsupportedRead.
	got, err = analyzeMaskingStatement(`db.users.countDocuments({})`)
	require.NoError(t, err)
	require.Equal(t, base.MongoDBMaskableAPIUnsupportedRead, got.API)

	// Non-read and multi-statement return nil, nil; syntax error propagates.
	got, err = analyzeMaskingStatement(`db.users.insertOne({name: "alice"})`)
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = analyzeMaskingStatement(`db.users.find({}); db.users.findOne({})`)
	require.NoError(t, err)
	require.Nil(t, got)

	_, err = analyzeMaskingStatement(`db.users.find({`)
	require.Error(t, err)
}
