package mssql

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestShowplanStatistic(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option *v1pb.QueryOption
		want   string
	}{
		{name: "no option", option: nil, want: "SHOWPLAN_ALL"},
		{name: "empty option", option: &v1pb.QueryOption{}, want: "SHOWPLAN_ALL"},
		{
			name:   "text",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_TEXT},
			want:   "SHOWPLAN_ALL",
		},
		{
			name:   "xml",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_XML},
			want:   "SHOWPLAN_XML",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, showplanStatistic(tc.option))
		})
	}
}
