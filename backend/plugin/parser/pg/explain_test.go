package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplainStatement(t *testing.T) {
	tests := []struct {
		statement string
		format    string
		want      string
	}{
		{"SELECT 1", "", "EXPLAIN SELECT 1"},
		{"SELECT 1", "JSON", "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"SELECT 1", "XML", "EXPLAIN (FORMAT XML) SELECT 1"},
		{"SELECT 1", "YAML", "EXPLAIN (FORMAT YAML) SELECT 1"},
		{"EXPLAIN SELECT 1", "", "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1", "TEXT", "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1", "JSON", "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN (FORMAT JSON) SELECT 1", "JSON", "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN (FORMAT JSON) SELECT 1", "", "EXPLAIN (FORMAT TEXT) SELECT 1"},
		{"explain verbose select 1;", "JSON", "EXPLAIN (verbose, FORMAT JSON) select 1;"},
		{"EXPLAIN (COSTS OFF, FORMAT yaml, SETTINGS) SELECT 1", "XML", "EXPLAIN (costs off, settings, FORMAT XML) SELECT 1"},
		{"EXPLAIN (FORMAT JSON, SETTINGS) SELECT 1", "YAML", "EXPLAIN (settings, FORMAT YAML) SELECT 1"},
		{"EXPLAIN (BUFFERS 1) VALUES (1)", "JSON", "EXPLAIN (buffers 1, FORMAT JSON) VALUES (1)"},
		{"EXPLAIN (ANALYZE false) SELECT 1", "JSON", "EXPLAIN (analyze false, FORMAT JSON) SELECT 1"},
		{"/* plan */ EXPLAIN (\"verbose\" 'on', SERIALIZE TEXT) SELECT ')' -- done", "JSON", "EXPLAIN (verbose on, serialize text, FORMAT JSON) SELECT ')' -- done"},
		{"EXPLAIN (COSTS OFF, GENERIC_PLAN) SELECT a FROM t WHERE b = $1 ORDER BY a", "JSON", "EXPLAIN (costs off, generic_plan, FORMAT JSON) SELECT a FROM t WHERE b = $1 ORDER BY a"},
		{"EXPLAIN (GENERIC_PLAN) WITH x AS (SELECT $1) SELECT * FROM x", "JSON", "EXPLAIN (generic_plan, FORMAT JSON) WITH x AS (SELECT $1) SELECT * FROM x"},
		{"EXPLAIN WITH d AS (SELECT 1) DELETE FROM t WHERE a IN (SELECT * FROM d)", "XML", "EXPLAIN (FORMAT XML) WITH d AS (SELECT 1) DELETE FROM t WHERE a IN (SELECT * FROM d)"},
	}
	for _, tc := range tests {
		got, err := ExplainStatement(tc.statement, tc.format)
		require.NoError(t, err, tc.statement)
		require.Equal(t, tc.want, got, tc.statement)
	}
}

func TestExplainStatementRefusesExecution(t *testing.T) {
	for _, statement := range []string{
		"EXPLAIN ANALYZE SELECT 1",
		"EXPLAIN ANALYSE VERBOSE SELECT 1",
		"EXPLAIN (ANALYZE, FORMAT json) SELECT 1",
		"EXPLAIN (COSTS OFF, ANALYZE on) DELETE FROM t",
	} {
		for _, format := range []string{"", "JSON"} {
			_, err := ExplainStatement(statement, format)
			require.Error(t, err, statement)
		}
	}
	// Prefixing EXPLAIN makes this an EXPLAIN ANALYZE.
	_, err := ExplainStatement("ANALYZE SELECT 1", "")
	require.Error(t, err)
}

func TestDescribeExplain(t *testing.T) {
	tests := []struct {
		statement string
		format    string
		analyze   bool
	}{
		{"EXPLAIN SELECT 1", "text", false},
		{"-- plan\nexplain verbose select 1;", "text", false},
		{"EXPLAIN (FORMAT JSON) SELECT 1", "json", false},
		{"EXPLAIN (ANALYZE, FORMAT 'xml') SELECT 1", "xml", true},
		// PostgreSQL reads the last FORMAT.
		{"EXPLAIN (FORMAT JSON, COSTS OFF, FORMAT YAML) SELECT 1", "yaml", false},
		{"EXPLAIN (FORMAT 1) SELECT 1", "", false},
		{"EXPLAIN ANALYZE VERBOSE SELECT 1", "text", true},
		{"EXPLAIN (ANALYZE off) SELECT 1", "text", false},
	}
	for _, tc := range tests {
		format, analyze, ok := DescribeExplain(tc.statement)
		require.True(t, ok, tc.statement)
		require.Equal(t, tc.format, format, tc.statement)
		require.Equal(t, tc.analyze, analyze, tc.statement)
	}

	for _, statement := range []string{
		"SELECT 1",
		"/* EXPLAIN */ SELECT 1",
		"EXPLAIN SELECT 1; EXPLAIN SELECT 2",
		"EXPLAIN (SELECT",
	} {
		_, _, ok := DescribeExplain(statement)
		require.False(t, ok, statement)
	}
}
