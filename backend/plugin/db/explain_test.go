package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestExplainFormats(t *testing.T) {
	for _, tc := range []struct {
		engine        storepb.Engine
		formats       []v1pb.QueryOption_ExplainFormat
		defaultFormat v1pb.QueryOption_ExplainFormat
	}{
		{storepb.Engine_POSTGRES, []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON, v1pb.QueryOption_XML, v1pb.QueryOption_YAML}, v1pb.QueryOption_TEXT},
		{storepb.Engine_MSSQL, []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT, v1pb.QueryOption_XML}, v1pb.QueryOption_TEXT},
		{storepb.Engine_SPANNER, []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_JSON}, v1pb.QueryOption_JSON},
		{storepb.Engine_MYSQL, []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT}, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED},
		{storepb.Engine_ORACLE, []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT}, v1pb.QueryOption_TEXT},
	} {
		formats, defaultFormat, ok := ExplainFormats(tc.engine)
		require.Truef(t, ok, "%s", tc.engine)
		require.Equalf(t, tc.formats, formats, "%s", tc.engine)
		require.Equalf(t, tc.defaultFormat, defaultFormat, "%s", tc.engine)
	}
	_, _, ok := ExplainFormats(storepb.Engine_MONGODB)
	require.False(t, ok)
}
