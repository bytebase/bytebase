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

func TestExplainStatementRejects(t *testing.T) {
	for _, statement := range []string{
		"SELECT 1; SELECT 2",
		"",
	} {
		_, err := explainStatement(statement, base.ExplainFormatDefault)
		require.Errorf(t, err, "%q", statement)
	}
}
