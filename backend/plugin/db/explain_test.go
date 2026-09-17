package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestExplainStatement(t *testing.T) {
	tests := []struct {
		engine storepb.Engine
		format v1pb.QueryOption_ExplainFormat
		want   string
		wantOK bool
	}{
		{storepb.Engine_POSTGRES, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN SELECT 1", true},
		{storepb.Engine_POSTGRES, v1pb.QueryOption_JSON, "EXPLAIN (FORMAT JSON) SELECT 1", true},
		{storepb.Engine_POSTGRES, v1pb.QueryOption_XML, "EXPLAIN (FORMAT XML) SELECT 1", true},
		{storepb.Engine_POSTGRES, v1pb.QueryOption_YAML, "EXPLAIN (FORMAT YAML) SELECT 1", true},
		{storepb.Engine_MYSQL, v1pb.QueryOption_JSON, "EXPLAIN SELECT 1", true},
		{storepb.Engine_TRINO, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN SELECT 1", true},
		{storepb.Engine_HIVE, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN SELECT 1", true},
		// Non-prefix engines (their driver builds the plan another way) report ok=false.
		{storepb.Engine_ORACLE, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "", false},
		{storepb.Engine_MSSQL, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "", false},
		{storepb.Engine_SPANNER, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "", false},
		{storepb.Engine_MONGODB, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "", false},
	}
	for _, tc := range tests {
		got, ok := ExplainStatement(tc.engine, "SELECT 1", tc.format)
		require.Equalf(t, tc.wantOK, ok, "%s ok", tc.engine)
		require.Equalf(t, tc.want, got, "%s statement", tc.engine)
	}
}

func TestExplainStatementSetsPostgresFormat(t *testing.T) {
	tests := []struct {
		statement string
		format    v1pb.QueryOption_ExplainFormat
		want      string
	}{
		{"EXPLAIN SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1", v1pb.QueryOption_TEXT, "EXPLAIN SELECT 1"},
		{"EXPLAIN SELECT 1", v1pb.QueryOption_JSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN (FORMAT JSON) SELECT 1", v1pb.QueryOption_JSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{"EXPLAIN (FORMAT JSON) SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "EXPLAIN (FORMAT TEXT) SELECT 1"},
		{"explain verbose select 1;", v1pb.QueryOption_JSON, "EXPLAIN (verbose, FORMAT JSON) select 1;"},
		{"EXPLAIN (COSTS OFF, FORMAT yaml, SETTINGS) SELECT 1", v1pb.QueryOption_XML, "EXPLAIN (costs off, settings, FORMAT XML) SELECT 1"},
		{"EXPLAIN (FORMAT JSON, SETTINGS) SELECT 1", v1pb.QueryOption_YAML, "EXPLAIN (settings, FORMAT YAML) SELECT 1"},
		{"EXPLAIN (BUFFERS 1) VALUES (1)", v1pb.QueryOption_JSON, "EXPLAIN (buffers 1, FORMAT JSON) VALUES (1)"},
		// ANALYZE is kept for the query validator to refuse.
		{"EXPLAIN (ANALYZE, FORMAT json) SELECT 1", v1pb.QueryOption_XML, "EXPLAIN (analyze, FORMAT XML) SELECT 1"},
		{"/* plan */ EXPLAIN (\"verbose\" 'on', SERIALIZE TEXT) SELECT ')' -- done", v1pb.QueryOption_JSON, "EXPLAIN (verbose on, serialize text, FORMAT JSON) SELECT ')' -- done"},
		{"EXPLAIN (COSTS OFF, GENERIC_PLAN) SELECT a FROM t WHERE b = $1 ORDER BY a", v1pb.QueryOption_JSON, "EXPLAIN (costs off, generic_plan, FORMAT JSON) SELECT a FROM t WHERE b = $1 ORDER BY a"},
		{"EXPLAIN (GENERIC_PLAN) WITH x AS (SELECT $1) SELECT * FROM x", v1pb.QueryOption_JSON, "EXPLAIN (generic_plan, FORMAT JSON) WITH x AS (SELECT $1) SELECT * FROM x"},
		{"EXPLAIN WITH d AS (SELECT 1) DELETE FROM t WHERE a IN (SELECT * FROM d)", v1pb.QueryOption_XML, "EXPLAIN (FORMAT XML) WITH d AS (SELECT 1) DELETE FROM t WHERE a IN (SELECT * FROM d)"},
	}
	for _, tc := range tests {
		got, ok := ExplainStatement(storepb.Engine_POSTGRES, tc.statement, tc.format)
		require.True(t, ok, tc.statement)
		require.Equal(t, tc.want, got, tc.statement)
	}
}
