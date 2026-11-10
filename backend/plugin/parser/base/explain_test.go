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
		// A comment holding another one reads differently on an engine that
		// nests them and on one that does not, and each reading leaves the
		// other a statement that returns rows. Both of these run as a SELECT of
		// ssn — the first on PostgreSQL, the second on MySQL.
		{"/* /* */ EXPLAIN */ SELECT ssn FROM t", false},
		{`/* /* */ SELECT '*/ EXPLAIN', ssn FROM t`, false},
		// The inner comment can open where the first `*/` begins: Spark runs
		// this one as the DELETE.
		{"/*/*/ EXPLAIN */ */ DELETE FROM t", false},
		{"/* a /* b */ EXPLAIN */ DELETE FROM t", false},
		{"/*/* EXPLAIN */*/ SELECT ssn FROM t", false},
		{"/* /* unclosed */ EXPLAIN SELECT 1", false},
		{"/* a /* b */ c */ EXPLAIN SELECT 1", false},
		// A comment of its own still reads, and so does a keyword before one.
		{"/* a */ /* b */ EXPLAIN SELECT 1", true},
		{"EXPLAIN /* a /* b */ c */ SELECT 1", true},
	}
	for _, tc := range tests {
		require.Equalf(t, tc.want, StartsWithExplain(tc.statement), "%q", tc.statement)
	}
}
