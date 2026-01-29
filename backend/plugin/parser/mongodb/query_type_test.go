package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQueryType(t *testing.T) {
	tests := []struct {
		description string
		statement   string
		want        base.QueryType
	}{
		// Select commands.
		{
			description: "find",
			statement:   `db.users.find({})`,
			want:        base.Select,
		},
		{
			description: "findOne",
			statement:   `db.users.findOne({name: "John"})`,
			want:        base.Select,
		},
		{
			description: "countDocuments",
			statement:   `db.users.countDocuments({})`,
			want:        base.Select,
		},
		{
			description: "estimatedDocumentCount",
			statement:   `db.users.estimatedDocumentCount()`,
			want:        base.Select,
		},
		{
			description: "distinct",
			statement:   `db.users.distinct("name")`,
			want:        base.Select,
		},
		{
			description: "aggregate without $out/$merge",
			statement:   `db.users.aggregate([{$match: {}}])`,
			want:        base.Select,
		},

		// DML commands.
		{
			description: "insertOne",
			statement:   `db.users.insertOne({name: "John"})`,
			want:        base.DML,
		},
		{
			description: "insertMany",
			statement:   `db.users.insertMany([{a: 1}, {a: 2}])`,
			want:        base.DML,
		},
		{
			description: "updateOne",
			statement:   `db.users.updateOne({}, {$set: {a: 1}})`,
			want:        base.DML,
		},
		{
			description: "updateMany",
			statement:   `db.users.updateMany({}, {$set: {a: 1}})`,
			want:        base.DML,
		},
		{
			description: "deleteOne",
			statement:   `db.users.deleteOne({})`,
			want:        base.DML,
		},
		{
			description: "deleteMany",
			statement:   `db.users.deleteMany({})`,
			want:        base.DML,
		},
		{
			description: "replaceOne",
			statement:   `db.users.replaceOne({a: 1}, {a: 2})`,
			want:        base.DML,
		},
		{
			description: "findOneAndUpdate",
			statement:   `db.users.findOneAndUpdate({}, {$set: {a: 1}})`,
			want:        base.DML,
		},
		{
			description: "findOneAndReplace",
			statement:   `db.users.findOneAndReplace({a: 1}, {a: 2})`,
			want:        base.DML,
		},
		{
			description: "findOneAndDelete",
			statement:   `db.users.findOneAndDelete({a: 1})`,
			want:        base.DML,
		},

		// Ambiguous aggregate with $out / $merge.
		{
			description: "aggregate with $out",
			statement:   `db.users.aggregate([{$match: {}}, {$out: "output"}])`,
			want:        base.DML,
		},
		{
			description: "aggregate with $merge",
			statement:   `db.users.aggregate([{$match: {}}, {$merge: {into: "output"}}])`,
			want:        base.DML,
		},

		// DDL commands.
		{
			description: "createCollection",
			statement:   `db.createCollection("test")`,
			want:        base.DDL,
		},
		{
			description: "dropDatabase",
			statement:   `db.dropDatabase()`,
			want:        base.DDL,
		},
		{
			description: "drop collection",
			statement:   `db.users.drop()`,
			want:        base.DDL,
		},
		{
			description: "createIndex",
			statement:   `db.users.createIndex({name: 1})`,
			want:        base.DDL,
		},
		{
			description: "createIndexes",
			statement:   `db.users.createIndexes([{key: {name: 1}}])`,
			want:        base.DDL,
		},
		{
			description: "dropIndex",
			statement:   `db.users.dropIndex("name_1")`,
			want:        base.DDL,
		},
		{
			description: "dropIndexes",
			statement:   `db.users.dropIndexes()`,
			want:        base.DDL,
		},
		{
			description: "renameCollection",
			statement:   `db.users.renameCollection("newName")`,
			want:        base.DDL,
		},

		// Info commands (SelectInfoSchema).
		{
			description: "show dbs",
			statement:   `show dbs`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "show collections",
			statement:   `show collections`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "getCollectionNames",
			statement:   `db.getCollectionNames()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "getCollectionInfos",
			statement:   `db.getCollectionInfos()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "db.stats()",
			statement:   `db.stats()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "collection stats",
			statement:   `db.users.stats()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "serverStatus",
			statement:   `db.serverStatus()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "serverBuildInfo",
			statement:   `db.serverBuildInfo()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "db.version()",
			statement:   `db.version()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "hostInfo",
			statement:   `db.hostInfo()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "listCommands",
			statement:   `db.listCommands()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "getIndexes",
			statement:   `db.users.getIndexes()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "dataSize",
			statement:   `db.users.dataSize()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "storageSize",
			statement:   `db.users.storageSize()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "totalIndexSize",
			statement:   `db.users.totalIndexSize()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "totalSize",
			statement:   `db.users.totalSize()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "isCapped",
			statement:   `db.users.isCapped()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "validate",
			statement:   `db.users.validate()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "latencyStats",
			statement:   `db.users.latencyStats()`,
			want:        base.SelectInfoSchema,
		},

		// Explain.
		{
			description: "find with explain",
			statement:   `db.users.find({}).explain()`,
			want:        base.Explain,
		},
		{
			description: "find with sort and explain",
			statement:   `db.users.find({}).sort({a: 1}).explain()`,
			want:        base.Explain,
		},

		// Bulk operations.
		{
			description: "initializeOrderedBulkOp",
			statement:   `db.users.initializeOrderedBulkOp().insert({a: 1}).execute()`,
			want:        base.DML,
		},

		// RS statements.
		{
			description: "rs.status()",
			statement:   `rs.status()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "rs.initiate()",
			statement:   `rs.initiate()`,
			want:        base.DML,
		},
		{
			description: "rs.help()",
			statement:   `rs.help()`,
			want:        base.SelectInfoSchema,
		},

		// SH statements.
		{
			description: "sh.status()",
			statement:   `sh.status()`,
			want:        base.SelectInfoSchema,
		},
		{
			description: "sh.addShard()",
			statement:   `sh.addShard("host:27017")`,
			want:        base.DML,
		},
		{
			description: "sh.help()",
			statement:   `sh.help()`,
			want:        base.SelectInfoSchema,
		},

		// runCommand.
		{
			description: "runCommand find",
			statement:   `db.runCommand({find: "users"})`,
			want:        base.Select,
		},
		{
			description: "runCommand insert",
			statement:   `db.runCommand({insert: "users", documents: [{a: 1}]})`,
			want:        base.DML,
		},
		{
			description: "runCommand create",
			statement:   `db.runCommand({create: "newColl"})`,
			want:        base.DDL,
		},
		{
			description: "runCommand serverStatus",
			statement:   `db.runCommand({serverStatus: 1})`,
			want:        base.SelectInfoSchema,
		},

		// Generic/unknown methods.
		{
			description: "unknown collection method",
			statement:   `db.users.someUnknownMethod()`,
			want:        base.DML,
		},
		{
			description: "unknown db method",
			statement:   `db.someUnknownDbMethod()`,
			want:        base.DML,
		},

		// Parse error with partial AST: parser cannot recover "find" from the
		// incomplete input, so it falls back to the default DML.
		{
			description: "incomplete find statement",
			statement:   `db.users.find({`,
			want:        base.DML,
		},

		// Completely unparseable.
		{
			description: "completely unparseable input",
			statement:   `@@@###`,
			want:        base.DML,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			got := GetQueryType(tc.statement)
			require.Equal(t, tc.want, got, "statement: %s", tc.statement)
		})
	}
}
