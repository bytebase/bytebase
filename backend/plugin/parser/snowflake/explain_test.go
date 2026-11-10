package snowflake

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
		{"WITH x AS (SELECT 1) SELECT * FROM x", "EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x"},
		// A statement that already asks for a plan never becomes the plan of a plan.
		{"EXPLAIN SELECT 1", "EXPLAIN SELECT 1"},
		{"EXPLAIN USING JSON SELECT 1", "EXPLAIN SELECT 1"},
		{"  explain select 1", "EXPLAIN select 1"},
	}
	for _, tc := range tests {
		got, err := explainStatement(tc.statement, base.ExplainFormatDefault)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
}

// TestExplainStatementJSON pins the statement that asks Snowflake for the
// structured plan the viewer draws, and that a caller's own EXPLAIN is
// replanned in the format asked for rather than wrapped in a second one.
func TestExplainStatementJSON(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		{"SELECT 1", "EXPLAIN USING JSON SELECT 1"},
		{"EXPLAIN SELECT 1", "EXPLAIN USING JSON SELECT 1"},
		{"EXPLAIN USING TABULAR SELECT 1", "EXPLAIN USING JSON SELECT 1"},
		{"EXPLAIN USING JSON SELECT 1", "EXPLAIN USING JSON SELECT 1"},
	}
	for _, tc := range tests {
		got, err := explainStatement(tc.statement, base.ExplainFormatJSON)
		require.NoErrorf(t, err, "%q", tc.statement)
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
	// Only JSON has a reader; every other format gets the tabular plan.
	for _, format := range []base.ExplainFormat{base.ExplainFormatText, base.ExplainFormatXML, base.ExplainFormatYAML} {
		got, err := explainStatement("SELECT 1", format)
		require.NoError(t, err)
		require.Equal(t, "EXPLAIN SELECT 1", got)
	}
}

// TestDescribeExplain pins what a result says about a plan the caller asked for
// in their own statement. Snowflake's EXPLAIN compiles without running, so no
// format of it executes what it plans.
func TestDescribeExplain(t *testing.T) {
	for _, tc := range []struct {
		statement string
		format    string
		isExplain bool
	}{
		{statement: "SELECT 1"},
		{statement: "EXPLAIN SELECT 1", format: "text", isExplain: true},
		{statement: "EXPLAIN USING TABULAR SELECT 1", format: "text", isExplain: true},
		{statement: "EXPLAIN USING TEXT SELECT 1", format: "text", isExplain: true},
		{statement: "EXPLAIN USING JSON SELECT 1", format: "json", isExplain: true},
		{statement: "explain using json select 1", format: "json", isExplain: true},
	} {
		format, executed, isExplain := DescribeExplain(tc.statement)
		require.Equalf(t, tc.format, format, "%q", tc.statement)
		require.Falsef(t, executed, "%q", tc.statement)
		require.Equalf(t, tc.isExplain, isExplain, "%q", tc.statement)
	}
}

func TestExplainStatementRejects(t *testing.T) {
	for _, statement := range []string{
		"SELECT 1; SELECT 2",
		"",
	} {
		_, err := explainStatement(statement, base.ExplainFormatDefault)
		require.Errorf(t, err, "%q", statement)
	}
}
