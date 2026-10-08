package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartsWithExplain(t *testing.T) {
	tests := []struct {
		statement string
		want      bool
	}{
		{"EXPLAIN SELECT 1", true},
		{"explain select 1", true},
		{"\n\t EXPLAIN SELECT 1", true},
		{"EXPLAIN(SELECT 1)", true},
		{"EXPLAIN", true},
		{"-- plan it\nEXPLAIN SELECT 1", true},
		{"/* plan it */EXPLAIN SELECT 1", true},
		{"/* one */ -- two\n /* three */ EXPLAIN SELECT 1", true},
		{"SELECT 1", false},
		{"EXPLAINED", false},
		{"EXPLAIN_PLAN()", false},
		{"", false},
		// A comment that never ends leaves no keyword to read.
		{"/* EXPLAIN SELECT 1", false},
		{"-- EXPLAIN SELECT 1", false},
	}
	for _, tc := range tests {
		require.Equalf(t, tc.want, StartsWithExplain(tc.statement), "%q", tc.statement)
	}
}
