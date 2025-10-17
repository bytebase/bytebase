package tsql

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
						Text:     "-- first statement\ndeclare @temp table(a int)",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 2, Column: 1},
						End:      &storepb.Position{Line: 2, Column: 26},
						Empty:    false,
					},
					{
						Text:     "\n-- second statement\ninsert into @temp values(1)",
						BaseLine: 1,
						Start:    &storepb.Position{Line: 4, Column: 1},
						End:      &storepb.Position{Line: 4, Column: 27},
						Empty:    false,
					},
					{
						Text:     "\n-- third statement\nselect * from @temp\n-- go statement\ngo",
						BaseLine: 3,
						Start:    &storepb.Position{Line: 6, Column: 1},
						End:      &storepb.Position{Line: 8, Column: 1},
						Empty:    false,
					},
					{
						Text:     "\ngo",
						BaseLine: 7,
						Start:    &storepb.Position{Line: 9, Column: 1},
						End:      &storepb.Position{Line: 9, Column: 1},
						Empty:    false,
					},
					{
						Text:     "\ngo\ngo",
						BaseLine: 8,
						Start:    &storepb.Position{Line: 10, Column: 1},
						End:      &storepb.Position{Line: 11, Column: 1},
						Empty:    false,
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
