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
		got, ok, err := ExplainStatement(tc.engine, "SELECT 1", tc.format)
		require.NoErrorf(t, err, "%s", tc.engine)
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
		{"explain verbose select 1;", v1pb.QueryOption_JSON, "EXPLAIN (VERBOSE, FORMAT JSON) select 1;"},
		{"EXPLAIN (COSTS OFF, FORMAT yaml, SETTINGS) SELECT 1", v1pb.QueryOption_XML, "EXPLAIN (COSTS OFF, SETTINGS, FORMAT XML) SELECT 1"},
		{"EXPLAIN (ANALYZE false, format json) SELECT 1", v1pb.QueryOption_XML, "EXPLAIN (ANALYZE false, FORMAT XML) SELECT 1"},
		{"/* plan */ EXPLAIN (\"verbose\" 'on', SERIALIZE TEXT) SELECT ')' -- done", v1pb.QueryOption_JSON, "EXPLAIN (\"verbose\" 'on', SERIALIZE TEXT, FORMAT JSON) SELECT ')' -- done"},
		{"EXPLAIN (GENERIC_PLAN) WITH x AS (SELECT $1) SELECT * FROM x", v1pb.QueryOption_JSON, "EXPLAIN (GENERIC_PLAN, FORMAT JSON) WITH x AS (SELECT $1) SELECT * FROM x"},
	}
	for _, tc := range tests {
		got, ok, err := ExplainStatement(storepb.Engine_POSTGRES, tc.statement, tc.format)
		require.NoError(t, err, tc.statement)
		require.True(t, ok, tc.statement)
		require.Equal(t, tc.want, got, tc.statement)
	}
}

func TestExplainStatementRefusesPostgresExecution(t *testing.T) {
	for _, statement := range []string{
		"EXPLAIN ANALYZE SELECT 1",
		"EXPLAIN ANALYSE VERBOSE SELECT 1",
		"EXPLAIN (ANALYZE, FORMAT JSON) SELECT 1",
		"EXPLAIN (COSTS OFF, ANALYZE on) DELETE FROM t",
	} {
		for _, format := range []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, v1pb.QueryOption_JSON} {
			_, _, err := ExplainStatement(storepb.Engine_POSTGRES, statement, format)
			require.Error(t, err, statement)
		}
	}
	// Prefixing EXPLAIN makes this an EXPLAIN ANALYZE.
	_, _, err := ExplainStatement(storepb.Engine_POSTGRES, "ANALYZE SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED)
	require.Error(t, err)

	_, _, err = ExplainStatement(storepb.Engine_POSTGRES, "EXPLAIN (ANALYZE false) SELECT 1", v1pb.QueryOption_JSON)
	require.NoError(t, err)
}
