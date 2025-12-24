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
						Start:    &storepb.Position{Line: 1, Column: 1},
						End:      &storepb.Position{Line: 1, Column: 8},
						Empty:    false,
						Range:    &storepb.Range{Start: 0, End: 8},
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
						Start:    &storepb.Position{Line: 1, Column: 1},
						End:      &storepb.Position{Line: 1, Column: 9},
						Empty:    false,
						Range:    &storepb.Range{Start: 0, End: 9},
					},
					{
						Text:     "\n SELECT\n 33;",
						Start:    &storepb.Position{Line: 2, Column: 2},
						End:      &storepb.Position{Line: 3, Column: 4},
						Empty:    false,
						Range:    &storepb.Range{Start: 9, End: 22},
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
