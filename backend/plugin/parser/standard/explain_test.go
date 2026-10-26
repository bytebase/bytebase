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

// TestClickhouseExplainStatement pins the statement that asks ClickHouse for
// the structured plan the viewer draws, and for the index accounting that is
// what a reader opens a ClickHouse plan for.
func TestClickhouseExplainStatement(t *testing.T) {
	tests := []struct {
		statement string
		format    base.ExplainFormat
		want      string
	}{
		{"SELECT 1", base.ExplainFormatJSON, "EXPLAIN json = 1, indexes = 1 SELECT 1"},
		{"SELECT 1", base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
		// Only JSON has a reader; the rest get the engine's own plan.
		{"SELECT 1", base.ExplainFormatText, "EXPLAIN SELECT 1"},
		{"SELECT 1", base.ExplainFormatXML, "EXPLAIN SELECT 1"},
		// Without an AST, a statement that already asks for a plan runs as the
		// user wrote it, so a typed EXPLAIN keeps the plan kind it named.
		{"EXPLAIN PIPELINE SELECT 1", base.ExplainFormatJSON, "EXPLAIN PIPELINE SELECT 1"},
	}
	for _, tc := range tests {
		got, err := clickhouseExplainStatement(tc.statement, tc.format)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q/%v", tc.statement, tc.format)
	}
}

// TestHiveKeepsItsOneFormat locks Hive to the plan it has always returned: it
// shares this file with ClickHouse but has no structured plan of its own.
func TestHiveKeepsItsOneFormat(t *testing.T) {
	for _, format := range []base.ExplainFormat{base.ExplainFormatDefault, base.ExplainFormatJSON} {
		got, err := explainStatement("SELECT 1", format)
		require.NoError(t, err)
		require.Equal(t, "EXPLAIN SELECT 1", got)
	}
}
