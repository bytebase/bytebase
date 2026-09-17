package redshift

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
		{"WITH x AS (SELECT 1) SELECT * FROM x", "EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x"},
		// A statement that already asks for a plan is rebuilt around the statement
		// it plans, so no request plans a plan.
		{"EXPLAIN SELECT 1", "EXPLAIN SELECT 1"},
		{"EXPLAIN VERBOSE SELECT 1", "EXPLAIN SELECT 1"},
		{"EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x", "EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x"},
		{"EXPLAIN SELECT 1 ORDER BY 1", "EXPLAIN SELECT 1 ORDER BY 1"},
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
		"EXPLAIN ANALYZE DELETE FROM t",
		"VERBOSE SELECT 1",
		"SELECT 1; SELECT 2",
		"",
	} {
		_, err := explainStatement(statement, base.ExplainFormatDefault)
		require.Errorf(t, err, "%q", statement)
	}
}
