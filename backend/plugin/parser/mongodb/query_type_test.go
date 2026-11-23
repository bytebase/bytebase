package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQueryTypeSeam(t *testing.T) {
	// Exercises omni mongo/analysis.ClassifyStatement -> base.OmniQueryType mapping and safety fallbacks.
	tests := []struct {
		statement string
		want      base.QueryType
	}{
		{`db.users.find({})`, base.Select},
		{`db.users.aggregate([{$match: {}}])`, base.Select},
		{`db.users.aggregate([{$out: "output"}])`, base.DML},
		{`db.users.aggregate([{$merge: {into: "output"}}])`, base.DML},
		{`db.runCommand({find: "users"})`, base.Select},
		{`db.users.getIndexes()`, base.SelectInfoSchema},
		{`show dbs`, base.SelectInfoSchema},
		{`db.runCommand({serverStatus: 1})`, base.SelectInfoSchema},
		{`db.someUnknownDbMethod()`, base.DML},
		{`db.users.insertOne({a: 1})`, base.DML},
		{`db.createCollection("test")`, base.DDL},
		{`db.users.find({}).explain()`, base.Explain},
		{`db.users.someUnknownMethod()`, base.DML},
		{`db.users.find({`, base.DML},
	}

	for _, tc := range tests {
		require.Equal(t, tc.want, getQueryType(tc.statement), tc.statement)
	}
}
