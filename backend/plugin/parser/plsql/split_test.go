package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestOracleSplitMultiSQL(t *testing.T) {
	type resData struct {
		res []base.SingleSQL
		err string
	}
	type testData struct {
		statement string
		want      resData
	}
	tests := []testData{
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
			`,
		},
		{
			statement: `
				select * from t;
				create table table$1 (id int)
			`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:     `select * from t`,
						LastLine: 2,
					},
					{
						Text:     `create table table$1 (id int)`,
						LastLine: 3,
					},
				},
			},
		},
	}

	for _, test := range tests {
		res, err := SplitSQL(test.statement)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)
	}
}
