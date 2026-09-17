package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainFormat(t *testing.T) {
	tests := []struct {
		format v1pb.QueryOption_ExplainFormat
		want   base.ExplainFormat
	}{
		{v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, base.ExplainFormatDefault},
		{v1pb.QueryOption_TEXT, base.ExplainFormatText},
		{v1pb.QueryOption_JSON, base.ExplainFormatJSON},
		{v1pb.QueryOption_XML, base.ExplainFormatXML},
	}
	for _, tc := range tests {
		require.Equalf(t, tc.want, ExplainFormat(tc.format), "%v", tc.format)
	}
	// A request with no option at all asks for no particular format.
	require.Equal(t, base.ExplainFormatDefault, ExplainFormat((*v1pb.QueryOption)(nil).GetExplainFormat()))
}
