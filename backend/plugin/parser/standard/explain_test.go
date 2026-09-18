package standard

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
		{"explained_view_query()", "EXPLAIN explained_view_query()"},
		// Without an AST the user's own EXPLAIN is left as written, which is still
		// never the plan of a plan.
		{"EXPLAIN SELECT 1", "EXPLAIN SELECT 1"},
		{"explain PIPELINE SELECT 1", "explain PIPELINE SELECT 1"},
		{"/* plan it */ EXPLAIN SELECT 1", "/* plan it */ EXPLAIN SELECT 1"},
		{"-- plan it\nEXPLAIN SELECT 1", "-- plan it\nEXPLAIN SELECT 1"},
	}
	for _, tc := range tests {
		got, err := explainStatement(tc.statement, base.ExplainFormatDefault)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
}
