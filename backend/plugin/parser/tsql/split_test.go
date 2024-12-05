package tsql

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
	err string
}

func TestSplitSQL(t *testing.T) {
	// TODO(parser): `go` should not be recognized as execute_body_batch.
	tests := []splitTestData{
		{
			statement: `-- first statement
declare @temp table(a int)
-- second statement
insert into @temp values(1)
-- third statement
select * from @temp
-- go statement
go
go
go
go
			`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "-- first statement\ndeclare @temp table(a int)",
						BaseLine:             0,
						FirstStatementLine:   1,
						FirstStatementColumn: 0,
						LastLine:             1,
						LastColumn:           25,
						Empty:                false,
					},
					{
						Text:                 "\n-- second statement\ninsert into @temp values(1)",
						BaseLine:             1,
						FirstStatementLine:   3,
						FirstStatementColumn: 0,
						LastLine:             3,
						LastColumn:           26,
						Empty:                false,
					},
					{
						Text:                 "\n-- third statement\nselect * from @temp\n-- go statement\ngo",
						BaseLine:             3,
						FirstStatementLine:   5,
						FirstStatementColumn: 0,
						LastLine:             7,
						LastColumn:           0,
						Empty:                false,
					},
					{
						Text:                 "\ngo",
						BaseLine:             7,
						FirstStatementLine:   8,
						FirstStatementColumn: 0,
						LastLine:             8,
						LastColumn:           0,
						Empty:                false,
					},
					{
						Text:                 "\ngo\ngo",
						BaseLine:             8,
						FirstStatementLine:   9,
						FirstStatementColumn: 0,
						LastLine:             10,
						LastColumn:           0,
						Empty:                false,
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
