package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestClassifyQueryTypeExplain pins how EXPLAIN is classified, which decides the
// permission an explain request needs and whether the driver will run it. The
// security-critical case is EXPLAIN ANALYZE of a write: PostgreSQL executes the
// wrapped statement, so it must classify as the write it performs, not as a
// plain EXPLAIN.
func TestClassifyQueryTypeExplain(t *testing.T) {
	testCases := []struct {
		statement string
		wantType  base.QueryType
		wantAlyze bool
	}{
		// Plain EXPLAIN only plans; it never executes, whatever it wraps.
		{"EXPLAIN SELECT * FROM t", base.Explain, false},
		{"EXPLAIN DELETE FROM t", base.Explain, false},
		{"EXPLAIN INSERT INTO t VALUES (1)", base.Explain, false},

		// EXPLAIN ANALYZE executes the wrapped statement, so it inherits its type.
		{"EXPLAIN ANALYZE SELECT * FROM t", base.Select, true},
		{"EXPLAIN ANALYZE DELETE FROM t", base.DML, true},
		{"EXPLAIN ANALYZE INSERT INTO t VALUES (1)", base.DML, true},
		{"EXPLAIN ANALYZE UPDATE t SET a = 1", base.DML, true},

		// Bare statements keep classifying as before.
		{"SELECT * FROM t", base.Select, false},
		{"DELETE FROM t", base.DML, false},
	}

	for _, tc := range testCases {
		t.Run(tc.statement, func(t *testing.T) {
			stmts, err := ParsePg(tc.statement)
			require.NoError(t, err)
			require.Len(t, stmts, 1)

			gotType, gotAnalyze := classifyQueryType(stmts[0].AST, false /* allSystems */)
			require.Equal(t, tc.wantType, gotType)
			require.Equal(t, tc.wantAlyze, gotAnalyze)
		})
	}
}

func TestExplainFormat(t *testing.T) {
	testCases := []struct {
		statement string
		want      string
	}{
		{"EXPLAIN SELECT 1", "text"},
		{"-- plan\nexplain verbose select 1;", "text"},
		{"EXPLAIN (FORMAT JSON) SELECT 1", "json"},
		{"EXPLAIN (ANALYZE, FORMAT 'xml') SELECT 1", "xml"},
		// PostgreSQL reads the last FORMAT.
		{"EXPLAIN (FORMAT JSON, COSTS OFF, FORMAT YAML) SELECT 1", "yaml"},
		{"EXPLAIN (FORMAT 1) SELECT 1", ""},
	}
	for _, tc := range testCases {
		explain := ParseExplain(tc.statement)
		require.NotNil(t, explain, tc.statement)
		require.Equal(t, tc.want, ExplainFormat(explain), tc.statement)
	}

	for _, statement := range []string{
		"SELECT 1",
		"/* EXPLAIN */ SELECT 1",
		"EXPLAIN SELECT 1; EXPLAIN SELECT 2",
		"EXPLAIN (SELECT",
	} {
		require.Nil(t, ParseExplain(statement), statement)
	}
}
