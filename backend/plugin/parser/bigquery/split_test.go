package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
						Text:                 "SELECT 1",
						BaseLine:             0,
						LastLine:             0,
						LastColumn:           7,
						FirstStatementLine:   0,
						FirstStatementColumn: 0,
						Empty:                false,
					},
				},
			},
		},
		{
			statement: "SELECT 1;\n SELECT\n 33;",
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "SELECT 1;",
						BaseLine:             0,
						LastLine:             0,
						LastColumn:           8,
						FirstStatementLine:   0,
						FirstStatementColumn: 0,
						Empty:                false,
					},
					{
						Text:                 "\n SELECT\n 33;",
						BaseLine:             0,
						LastLine:             2,
						LastColumn:           3,
						FirstStatementLine:   1,
						FirstStatementColumn: 1,
						Empty:                false,
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
