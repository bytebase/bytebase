package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		{"SELECT 1", "EXPLAIN SELECT 1"},
		{"SELECT 1;", "EXPLAIN SELECT 1;"},
		{"DELETE FROM t", "EXPLAIN DELETE FROM t"},
		{"UPDATE t SET a = 1", "EXPLAIN UPDATE t SET a = 1"},
		// A statement that already asks for a plan is rebuilt around the statement
		// it plans, so no request plans a plan.
		{"EXPLAIN SELECT 1", "EXPLAIN SELECT 1"},
		{"EXPLAIN FORMAT=JSON SELECT 1", "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1 ORDER BY 1", "EXPLAIN SELECT 1 ORDER BY 1"},
		// An EXPLAIN with no statement to plan is left as the user wrote it: it
		// describes a table or another connection's query, and planning a plan
		// would only turn a working statement into a syntax error.
		{"EXPLAIN t", "EXPLAIN t"},
		{"DESCRIBE t", "DESCRIBE t"},
		{"EXPLAIN FOR CONNECTION 5", "EXPLAIN FOR CONNECTION 5"},
	}
	for _, tc := range tests {
		got, err := explainStatement(tc.statement, base.ExplainFormatDefault)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
}

// TestExplainStatementRejects locks what never becomes an EXPLAIN: a write with
// EXPLAIN options smuggled in front of it, which anything that simply prefixed
// EXPLAIN would turn into an EXPLAIN ANALYZE DELETE that runs the DELETE, and an
// EXPLAIN ANALYZE the caller wrote, which plans by executing what it plans.
func TestExplainStatementRejects(t *testing.T) {
	for _, statement := range []string{
		"ANALYZE DELETE FROM t",
		"EXPLAIN ANALYZE SELECT * FROM t",
		"EXPLAIN ANALYZE DELETE FROM t",
		"FORMAT=JSON SELECT 1",
		"SELECT 1; SELECT 2",
		"",
	} {
		_, err := explainStatement(statement, base.ExplainFormatDefault)
		require.Errorf(t, err, "%q", statement)
	}
}

// TestExplainStatementExecutableComment keeps the rebuild off text whose locations
// the lexer reports against a spliced buffer. The statement runs as written, which
// still never plans a plan.
func TestExplainStatementExecutableComment(t *testing.T) {
	const statement = "EXPLAIN /*!40001 SELECT 1 */"
	got, err := explainStatement(statement, base.ExplainFormatDefault)
	require.NoError(t, err)
	require.Equal(t, statement, got)
}
