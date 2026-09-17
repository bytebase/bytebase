package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestExplainStatement(t *testing.T) {
	tests := []struct {
		statement string
		format    v1pb.QueryOption_ExplainFormat
		want      string
	}{
		{"SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN SELECT 1"},
		{"SELECT 1", v1pb.QueryOption_JSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"SELECT 1", v1pb.QueryOption_XML, "EXPLAIN (FORMAT XML) SELECT 1"},
		{"SELECT 1", v1pb.QueryOption_YAML, "EXPLAIN (FORMAT YAML) SELECT 1"},
		{"EXPLAIN SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1", v1pb.QueryOption_TEXT, "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1", v1pb.QueryOption_JSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN (FORMAT JSON) SELECT 1", v1pb.QueryOption_JSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN (FORMAT JSON) SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN (FORMAT TEXT) SELECT 1"},
		{"explain verbose select 1;", v1pb.QueryOption_JSON, "EXPLAIN (verbose, FORMAT JSON) select 1;"},
		{"EXPLAIN (COSTS OFF, FORMAT yaml, SETTINGS) SELECT 1", v1pb.QueryOption_XML, "EXPLAIN (costs off, settings, FORMAT XML) SELECT 1"},
		{"EXPLAIN (FORMAT JSON, SETTINGS) SELECT 1", v1pb.QueryOption_YAML, "EXPLAIN (settings, FORMAT YAML) SELECT 1"},
		{"EXPLAIN (BUFFERS 1) VALUES (1)", v1pb.QueryOption_JSON, "EXPLAIN (buffers 1, FORMAT JSON) VALUES (1)"},
		{"EXPLAIN (ANALYZE false) SELECT 1", v1pb.QueryOption_JSON, "EXPLAIN (analyze false, FORMAT JSON) SELECT 1"},
		{"/* plan */ EXPLAIN (\"verbose\" 'on', SERIALIZE TEXT) SELECT ')' -- done", v1pb.QueryOption_JSON, "EXPLAIN (verbose on, serialize text, FORMAT JSON) SELECT ')' -- done"},
		{"EXPLAIN (COSTS OFF, GENERIC_PLAN) SELECT a FROM t WHERE b = $1 ORDER BY a", v1pb.QueryOption_JSON, "EXPLAIN (costs off, generic_plan, FORMAT JSON) SELECT a FROM t WHERE b = $1 ORDER BY a"},
		{"EXPLAIN (GENERIC_PLAN) WITH x AS (SELECT $1) SELECT * FROM x", v1pb.QueryOption_JSON, "EXPLAIN (generic_plan, FORMAT JSON) WITH x AS (SELECT $1) SELECT * FROM x"},
		{"EXPLAIN WITH d AS (SELECT 1) DELETE FROM t WHERE a IN (SELECT * FROM d)", v1pb.QueryOption_XML, "EXPLAIN (FORMAT XML) WITH d AS (SELECT 1) DELETE FROM t WHERE a IN (SELECT * FROM d)"},
	}
	for _, tc := range tests {
		got, err := explainStatement(tc.statement, tc.format)
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
		for _, format := range []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, v1pb.QueryOption_JSON} {
			_, err := explainStatement(statement, format)
			require.Error(t, err, statement)
		}
	}
	// Prefixing EXPLAIN makes this an EXPLAIN ANALYZE.
	_, err := explainStatement("ANALYZE SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED)
	require.Error(t, err)
}
