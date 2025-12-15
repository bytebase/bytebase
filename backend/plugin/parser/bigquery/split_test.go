package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type splitTestData struct {
	statement string
	want      resData
}

type resData struct {
	res []base.Statement
}

func TestBigQuerySplitMultiSQL(t *testing.T) {
	testCases := []splitTestData{
		{
			statement: "SELECT 1",
			want: resData{
				res: []base.Statement{
					{
						Text:     "SELECT 1",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 1, Column: 1},
						End:      &storepb.Position{Line: 1, Column: 8},
						Empty:    false,
					},
				},
			},
		},
		{
			statement: "SELECT 1;\n SELECT\n 33;",
			want: resData{
				res: []base.Statement{
					{
						Text:     "SELECT 1;",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 1, Column: 1},
						End:      &storepb.Position{Line: 1, Column: 9},
						Empty:    false,
					},
					{
						Text:     "\n SELECT\n 33;",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 2, Column: 2},
						End:      &storepb.Position{Line: 3, Column: 4},
						Empty:    false,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		got, err := SplitSQL(tc.statement)
		if err != nil {
			t.Errorf("unexpected error: %v, statement: %s", err, tc.statement)
		}
		require.Equalf(t, tc.want.res, got, "statement: %s", tc.statement)
	}
}
