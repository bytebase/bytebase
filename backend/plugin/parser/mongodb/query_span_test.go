package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpan(t *testing.T) {
	tests := []struct {
		description string
		statement   string
		wantType    base.QueryType
	}{
		{
			description: "find is Select",
			statement:   `db.users.find({})`,
			wantType:    base.Select,
		},
		{
			description: "findOne is Select",
			statement:   `db.users.findOne({name: "John"})`,
			wantType:    base.Select,
		},
		{
			description: "insertOne is DML",
			statement:   `db.users.insertOne({name: "John"})`,
			wantType:    base.DML,
		},
		{
			description: "insertMany is DML",
			statement:   `db.users.insertMany([{name: "A"}, {name: "B"}])`,
			wantType:    base.DML,
		},
		{
			description: "updateOne is DML",
			statement:   `db.users.updateOne({name: "John"}, {$set: {age: 30}})`,
			wantType:    base.DML,
		},
		{
			description: "deleteOne is DML",
			statement:   `db.users.deleteOne({name: "John"})`,
			wantType:    base.DML,
		},
		{
			description: "createCollection is DDL",
			statement:   `db.createCollection("test")`,
			wantType:    base.DDL,
		},
		{
			description: "dropDatabase is DDL",
			statement:   `db.dropDatabase()`,
			wantType:    base.DDL,
		},
		{
			description: "createIndex is DDL",
			statement:   `db.users.createIndex({name: 1})`,
			wantType:    base.DDL,
		},
		{
			description: "drop collection is DDL",
			statement:   `db.users.drop()`,
			wantType:    base.DDL,
		},
		{
			description: "find with explain is Explain",
			statement:   `db.users.find({}).explain()`,
			wantType:    base.Explain,
		},
		{
			description: "show dbs is SelectInfoSchema",
			statement:   `show dbs`,
			wantType:    base.SelectInfoSchema,
		},
		{
			description: "show collections is SelectInfoSchema",
			statement:   `show collections`,
			wantType:    base.SelectInfoSchema,
		},
		{
			description: "getCollectionNames is SelectInfoSchema",
			statement:   `db.getCollectionNames()`,
			wantType:    base.SelectInfoSchema,
		},
		{
			description: "aggregate without $out is Select",
			statement:   `db.users.aggregate([{$match: {age: {$gt: 25}}}])`,
			wantType:    base.Select,
		},
		{
			description: "aggregate with $out is DML",
			statement:   `db.users.aggregate([{$out: "output"}])`,
			wantType:    base.DML,
		},
		{
			description: "aggregate with $merge is DML",
			statement:   `db.users.aggregate([{$merge: {into: "output"}}])`,
			wantType:    base.DML,
		},
		{
			description: "countDocuments is Select",
			statement:   `db.users.countDocuments({})`,
			wantType:    base.Select,
		},
		{
			description: "distinct is Select",
			statement:   `db.users.distinct("name")`,
			wantType:    base.Select,
		},
		{
			description: "stats is SelectInfoSchema",
			statement:   `db.users.stats()`,
			wantType:    base.SelectInfoSchema,
		},
		{
			description: "getIndexes is SelectInfoSchema",
			statement:   `db.users.getIndexes()`,
			wantType:    base.SelectInfoSchema,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: tc.statement}, "", "", false)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Equal(t, tc.wantType, span.Type)
		})
	}
}
