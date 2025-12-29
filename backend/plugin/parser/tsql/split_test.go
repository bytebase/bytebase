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
	res []base.Statement
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
				res: []base.Statement{
					{
						Text:     "-- first statement\ndeclare @temp table(a int)",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 2, Column: 1},
						End:      &storepb.Position{Line: 2, Column: 27},
						Empty:    false,
						Range:    &storepb.Range{Start: 0, End: 45},
					},
					{
						Text:     "\n-- second statement\ninsert into @temp values(1)",
						BaseLine: 1,
						Start:    &storepb.Position{Line: 4, Column: 1},
						End:      &storepb.Position{Line: 4, Column: 28},
						Empty:    false,
						Range:    &storepb.Range{Start: 45, End: 93},
					},
					{
						Text:     "\n-- third statement\nselect * from @temp\n-- go statement\ngo",
						BaseLine: 3,
						Start:    &storepb.Position{Line: 6, Column: 1},
						End:      &storepb.Position{Line: 8, Column: 3},
						Empty:    false,
						Range:    &storepb.Range{Start: 93, End: 151},
					},
					{
						Text:     "\ngo",
						BaseLine: 7,
						Start:    &storepb.Position{Line: 9, Column: 1},
						End:      &storepb.Position{Line: 9, Column: 3},
						Empty:    false,
						Range:    &storepb.Range{Start: 151, End: 154},
					},
					{
						Text:     "\ngo\ngo",
						BaseLine: 8,
						Start:    &storepb.Position{Line: 10, Column: 1},
						End:      &storepb.Position{Line: 11, Column: 3},
						Empty:    false,
						Range:    &storepb.Range{Start: 154, End: 160},
					},
				},
			},
		},
		{
			statement: `








UPDATE SalesLT.Address SET City = "Shanghai";

UPDATE SalesLT.Address SET PostalCode = 0;


UPDATE SalesLT.ProductModelProductDescription SET Culture = "zh-cn";



`,
			want: resData{
				res: []base.Statement{
					{
						Text:     "\n\n\n\n\n\n\n\n\nUPDATE SalesLT.Address SET City = \"Shanghai\";",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 10, Column: 1},
						End:      &storepb.Position{Line: 10, Column: 46},
						Empty:    false,
						Range:    &storepb.Range{Start: 0, End: 54},
					},
					{
						Text:     "\n\nUPDATE SalesLT.Address SET PostalCode = 0;",
						BaseLine: 9,
						Start:    &storepb.Position{Line: 12, Column: 1},
						End:      &storepb.Position{Line: 12, Column: 43},
						Empty:    false,
						Range:    &storepb.Range{Start: 54, End: 98},
					},
					{
						Text:     "\n\n\nUPDATE SalesLT.ProductModelProductDescription SET Culture = \"zh-cn\";",
						BaseLine: 11,
						Start:    &storepb.Position{Line: 15, Column: 1},
						End:      &storepb.Position{Line: 15, Column: 69},
						Empty:    false,
						Range:    &storepb.Range{Start: 98, End: 169},
					},
				},
			},
		},
		{
			statement: "SELECT * FROM 表名; INSERT INTO 表 VALUES (1);",
			want: resData{
				res: []base.Statement{
					{
						Text:     "SELECT * FROM 表名;",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 1, Column: 1},
						End:      &storepb.Position{Line: 1, Column: 18},
						Empty:    false,
						Range:    &storepb.Range{Start: 0, End: 21}, // Byte offset 0-21 (not 0-17)
					},
					{
						Text:     " INSERT INTO 表 VALUES (1);",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 1, Column: 19},
						End:      &storepb.Position{Line: 1, Column: 44},
						Empty:    false,
						Range:    &storepb.Range{Start: 21, End: 49}, // Byte offset 21-49 (not 17-43)
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
