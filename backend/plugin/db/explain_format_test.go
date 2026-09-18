package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
		{v1pb.QueryOption_YAML, base.ExplainFormatYAML},
	}
	for _, tc := range tests {
		require.Equalf(t, tc.want, ExplainFormat(tc.format), "%v", tc.format)
	}
	// A request with no option at all asks for no particular format.
	require.Equal(t, base.ExplainFormatDefault, ExplainFormat((*v1pb.QueryOption)(nil).GetExplainFormat()))
}

func TestExplainResultFormat(t *testing.T) {
	for _, tc := range []struct {
		name   string
		engine storepb.Engine
		format v1pb.QueryOption_ExplainFormat
		want   v1pb.QueryOption_ExplainFormat
		isPlan bool
	}{
		{name: "postgres default", engine: storepb.Engine_POSTGRES, want: v1pb.QueryOption_TEXT, isPlan: true},
		{name: "postgres json", engine: storepb.Engine_POSTGRES, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "mssql xml", engine: storepb.Engine_MSSQL, format: v1pb.QueryOption_XML, want: v1pb.QueryOption_XML, isPlan: true},
		{name: "spanner default", engine: storepb.Engine_SPANNER, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "mysql unknown", engine: storepb.Engine_MYSQL, format: v1pb.QueryOption_TEXT, want: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, isPlan: true},
		{name: "bigquery dry run", engine: storepb.Engine_BIGQUERY},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, isPlan := ExplainResultFormat(tc.engine, tc.format)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.isPlan, isPlan)
		})
	}
}
