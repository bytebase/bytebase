package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestExplainStatement pins the statement that asks MongoDB for a plan. It is a
// cursor method on the end of the query, and its verbosity is the one that does
// not run what it plans.
func TestExplainStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		{`db.orders.find({a: 1})`, `db.orders.find({a: 1}).explain("queryPlanner")`},
		{`db.orders.find({}).sort({a: -1}).limit(10)`, `db.orders.find({}).sort({a: -1}).limit(10).explain("queryPlanner")`},
		{`db.orders.aggregate([{$match: {a: 1}}])`, `db.orders.aggregate([{$match: {a: 1}}]).explain("queryPlanner")`},
		{`db.orders.countDocuments({})`, `db.orders.countDocuments({}).explain("queryPlanner")`},
		{`db.orders.distinct("a")`, `db.orders.distinct("a").explain("queryPlanner")`},
		// Leading comments and a trailing semicolon are outside the statement.
		{"// plan it\ndb.orders.find({}) ;", `db.orders.find({}).explain("queryPlanner")`},
		// A statement that asks for its own plan keeps its query and loses its
		// verbosity, so a request to plan never runs what it plans.
		{`db.orders.find({}).explain()`, `db.orders.find({}).explain("queryPlanner")`},
		{`db.orders.find({}).explain("executionStats")`, `db.orders.find({}).explain("queryPlanner")`},
		{`db.orders.find({}).explain( "allPlansExecution" )`, `db.orders.find({}).explain("queryPlanner")`},
		{`db.orders.find({}).sort({a: 1}).explain("executionStats")`, `db.orders.find({}).sort({a: 1}).explain("queryPlanner")`},
	}
	for _, tc := range tests {
		got, err := explainStatement(tc.statement, base.ExplainFormatJSON)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
	// What this builds has to plan again to the same thing: the Visualize path
	// explains a statement that already explains.
	for _, tc := range tests {
		got, err := explainStatement(tc.want, base.ExplainFormatJSON)
		require.NoErrorf(t, err, "%q", tc.want)
		require.Equalf(t, tc.want, got, "%q", tc.want)
	}
}

// TestExplainStatementRejects locks what MongoDB will not plan. Unlike an
// engine that prefixes EXPLAIN, returning the statement unchanged here would
// run it, so anything unplannable is refused instead.
func TestExplainStatementRejects(t *testing.T) {
	for _, statement := range []string{
		`db.orders.insertOne({a: 1})`,
		`db.orders.updateOne({a: 1}, {$set: {b: 2}})`,
		`db.orders.deleteMany({})`,
		`db.orders.drop()`,
		// findOne has no plan of its own in the client that runs it.
		`db.orders.findOne({a: 1})`,
		// Not a query over a collection at all.
		`db.getCollectionNames()`,
		`db.orders.find({}); db.orders.find({})`,
		``,
	} {
		_, err := explainStatement(statement, base.ExplainFormatJSON)
		require.Errorf(t, err, "%q", statement)
	}
}

// TestDescribeExplain pins what a result says about a plan the caller asked for
// itself, and in particular whether producing it ran the query.
func TestDescribeExplain(t *testing.T) {
	tests := []struct {
		statement string
		format    string
		executed  bool
		isExplain bool
	}{
		{statement: `db.orders.find({}).explain()`, format: "json", isExplain: true},
		{statement: `db.orders.find({}).explain("queryPlanner")`, format: "json", isExplain: true},
		{statement: `db.orders.find({}).explain("executionStats")`, format: "json", executed: true, isExplain: true},
		{statement: `db.orders.find({}).explain("allPlansExecution")`, format: "json", executed: true, isExplain: true},
		{statement: `db.orders.aggregate([]).explain()`, format: "json", isExplain: true},
		{statement: `db.orders.find({})`},
		{statement: `db.orders.insertOne({a: 1})`},
		{statement: `not a statement`},
	}
	for _, tc := range tests {
		format, executed, isExplain := DescribeExplain(tc.statement)
		require.Equalf(t, tc.isExplain, isExplain, "%q", tc.statement)
		require.Equalf(t, tc.format, format, "%q", tc.statement)
		require.Equalf(t, tc.executed, executed, "%q", tc.statement)
	}
}
