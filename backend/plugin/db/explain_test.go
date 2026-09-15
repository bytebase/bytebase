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
		got, ok := ExplainStatement(tc.engine, "SELECT 1", tc.format)
		require.Equalf(t, tc.wantOK, ok, "%s ok", tc.engine)
		require.Equalf(t, tc.want, got, "%s statement", tc.engine)
	}
}
