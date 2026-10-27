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
		got, err := hiveExplainStatement(tc.statement, base.ExplainFormatDefault)
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

// TestDatabricksExplainStatement pins Spark's FORMATTED plan as the one
// Databricks is asked for: a numbered tree with a block of detail per
// operator, rather than the one dense tree a plain EXPLAIN prints.
func TestDatabricksExplainStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		{"SELECT 1", "EXPLAIN FORMATTED SELECT 1"},
		{"SELECT 1;", "EXPLAIN FORMATTED SELECT 1;"},
		// A write is planned, not run: EXPLAIN never executes what it plans.
		{"DELETE FROM t WHERE id = 1", "EXPLAIN FORMATTED DELETE FROM t WHERE id = 1"},
		// Without an AST the user's own EXPLAIN is left as written, so it keeps
		// whichever mode it named.
		{"EXPLAIN EXTENDED SELECT 1", "EXPLAIN EXTENDED SELECT 1"},
		{"-- plan it\nEXPLAIN COST SELECT 1", "-- plan it\nEXPLAIN COST SELECT 1"},
	}
	for _, tc := range tests {
		got, err := databricksExplainStatement(tc.statement, base.ExplainFormatDefault)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
	// Databricks prints this one plan whatever format the caller names.
	for _, format := range []base.ExplainFormat{base.ExplainFormatJSON, base.ExplainFormatText, base.ExplainFormatXML} {
		got, err := databricksExplainStatement("SELECT 1", format)
		require.NoError(t, err)
		require.Equal(t, "EXPLAIN FORMATTED SELECT 1", got)
	}
}

// TestDatabricksExplainReadsNestedComments locks the one thing between an
// explain request and a write on the engine with no read-only validator.
// Databricks block comments nest, so a statement whose EXPLAIN sits inside one
// is not a plan the caller asked for: Spark runs
// `/* a /* b */ EXPLAIN */ SELECT 1` as the SELECT, which this planned instead.
func TestDatabricksExplainReadsNestedComments(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		// The comment runs past the EXPLAIN, so the statement is the DELETE and
		// gets planned rather than preserved.
		{
			"/* outer /* inner */\nEXPLAIN\n*/\nDELETE FROM t",
			"EXPLAIN FORMATTED /* outer /* inner */\nEXPLAIN\n*/\nDELETE FROM t",
		},
		{"/* a /* b */ EXPLAIN */ SELECT 1", "EXPLAIN FORMATTED /* a /* b */ EXPLAIN */ SELECT 1"},
		// A nested comment that closes before a real EXPLAIN leaves the plan the
		// caller asked for.
		{"/* a /* b */ */ EXPLAIN SELECT 1", "/* a /* b */ */ EXPLAIN SELECT 1"},
		// A comment left open runs to the end, so there is no statement to
		// preserve.
		{"/* a /* b */ EXPLAIN SELECT 1", "EXPLAIN FORMATTED /* a /* b */ EXPLAIN SELECT 1"},
		// A line comment ends at its newline, nested or not.
		{"-- /* a\n/* b */ EXPLAIN SELECT 1", "-- /* a\n/* b */ EXPLAIN SELECT 1"},
	}
	for _, tc := range tests {
		got, err := databricksExplainStatement(tc.statement, base.ExplainFormatDefault)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
}

// TestHiveExplainStatementJSON pins the statement that asks Hive for the one
// plan it states as a document: FORMATTED answers with JSON, and every other
// format with a tree of lines.
func TestHiveExplainStatementJSON(t *testing.T) {
	got, err := hiveExplainStatement("SELECT 1", base.ExplainFormatJSON)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN FORMATTED SELECT 1", got)

	for _, format := range []base.ExplainFormat{base.ExplainFormatDefault, base.ExplainFormatText, base.ExplainFormatXML} {
		got, err := hiveExplainStatement("SELECT 1", format)
		require.NoError(t, err)
		require.Equal(t, "EXPLAIN SELECT 1", got)
	}

	// Without an AST, a statement that already asks for a plan runs as the user
	// wrote it, so a typed EXPLAIN keeps the format it named.
	got, err = hiveExplainStatement("EXPLAIN EXTENDED SELECT 1", base.ExplainFormatJSON)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN EXTENDED SELECT 1", got)
}
