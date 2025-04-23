package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type splitTestData struct {
	statement string
	want      resData
}

type resData struct {
	res []base.SingleSQL
}

func TestBigQuerySplitMultiSQL(t *testing.T) {
	testCases := []splitTestData{
		{
			statement: "SELECT 1",
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       "SELECT 1",
						BaseLine:   0,
						LastLine:   0,
						LastColumn: 7,
						Start:      &storepb.Position{Line: 0, Column: 0},
						Empty:      false,
					},
				},
			},
		},
		{
			statement: "SELECT 1;\n SELECT\n 33;",
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       "SELECT 1;",
						BaseLine:   0,
						LastLine:   0,
						LastColumn: 8,
						Start:      &storepb.Position{Line: 0, Column: 0},
						Empty:      false,
					},
					{
						Text:       "\n SELECT\n 33;",
						BaseLine:   0,
						LastLine:   2,
						LastColumn: 3,
						Start:      &storepb.Position{Line: 1, Column: 1},
						Empty:      false,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		got, err := SplitSQL(tc.statement)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		require.Equal(t, tc.want.res, got)
	}
}
