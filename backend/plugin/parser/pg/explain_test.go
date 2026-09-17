package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatement(t *testing.T) {
	tests := []struct {
		statement string
		format    base.ExplainFormat
		want      string
	}{
		{"SELECT 1", base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
		{"SELECT 1", base.ExplainFormatText, "EXPLAIN SELECT 1"},
		{"SELECT 1", base.ExplainFormatJSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"SELECT 1", base.ExplainFormatXML, "EXPLAIN (FORMAT XML) SELECT 1"},
		{"SELECT 1;", base.ExplainFormatDefault, "EXPLAIN SELECT 1;"},
		{"DELETE FROM t", base.ExplainFormatDefault, "EXPLAIN DELETE FROM t"},
		{"WITH x AS (SELECT 1) SELECT * FROM x", base.ExplainFormatDefault, "EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x"},
		// A statement that already asks for a plan is rebuilt around the statement
		// it plans, so the request's format replaces the user's own options and no
		// request plans a plan.
		{"EXPLAIN SELECT 1", base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1", base.ExplainFormatJSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN (FORMAT XML) SELECT 1", base.ExplainFormatJSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN /* plan it */ SELECT 1", base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
		{"EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x", base.ExplainFormatDefault, "EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x"},
		{"EXPLAIN SELECT 1 ORDER BY 1", base.ExplainFormatDefault, "EXPLAIN SELECT 1 ORDER BY 1"},
		{"EXPLAIN (ANALYZE false) SELECT 1", base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
	}
	for _, tc := range tests {
		got, err := explainStatement(tc.statement, tc.format)
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
		"EXPLAIN ANALYZE SELECT 1",
		"EXPLAIN ANALYZE DELETE FROM t",
		"EXPLAIN (ANALYZE, VERBOSE) DELETE FROM t",
		"(ANALYZE) DELETE FROM t",
		"VERBOSE SELECT 1",
		"SELECT 1; SELECT 2",
		"",
		"-- nothing here",
	} {
		_, err := explainStatement(statement, base.ExplainFormatDefault)
		require.Errorf(t, err, "%q", statement)
	}
}

// TestExplainStatementNeverPlansAPlan holds the invariant over the shapes
// PostgreSQL accepts after EXPLAIN, including those whose planned statement the
// parser may fail to locate: the statement that comes back says EXPLAIN once.
func TestExplainStatementNeverPlansAPlan(t *testing.T) {
	for _, statement := range []string{
		"EXPLAIN SELECT 1",
		"EXPLAIN INSERT INTO t VALUES (1)",
		"EXPLAIN EXECUTE p",
		"EXPLAIN DECLARE c CURSOR FOR SELECT 1",
		"EXPLAIN CREATE TABLE t AS SELECT 1",
		"EXPLAIN (COSTS OFF) VALUES (1)",
		"EXPLAIN TABLE t",
	} {
		for _, format := range []base.ExplainFormat{base.ExplainFormatDefault, base.ExplainFormatJSON} {
			got, err := explainStatement(statement, format)
			require.NoErrorf(t, err, "%q", statement)
			require.Equalf(t, 1, strings.Count(strings.ToUpper(got), "EXPLAIN"), "%q became %q", statement, got)
		}
	}
}

func TestExplainStatementCockroachDBIgnoresFormat(t *testing.T) {
	got, err := explainStatementDefaultFormat("SELECT 1", base.ExplainFormatJSON)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)
}
