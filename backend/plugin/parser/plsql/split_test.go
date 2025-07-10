package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
						Text:            `select * from t`,
						ByteOffsetStart: 5,
						ByteOffsetEnd:   20,
						Start:           &storepb.Position{Line: 1, Column: 4},
						End:             &storepb.Position{Line: 1, Column: 18},
					},
					{
						Text:            `create table table$1 (id int)`,
						ByteOffsetStart: 26,
						ByteOffsetEnd:   56,
						Start:           &storepb.Position{Line: 2, Column: 4},
						End:             &storepb.Position{Line: 2, Column: 32},
					},
				},
			},
		},
		{
			statement: `ALTER TABLE DATA.TEST
MODIFY PARTITION BY RANGE (TXN_DATE)
INTERVAL (NUMTODSINTERVAL(1, 'DAY'))
(
	PARTITION TEST_PO VALUES LESS THAN (
		TO_DATE('2000-01-01 00:00:00', 'SYYYY-MM-DD HH24:MI:SS', 'NLS_CALENDAR=GREGORIAN')
	)
)
ONLINE;`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:          "ALTER TABLE DATA.TEST\nMODIFY PARTITION BY RANGE (TXN_DATE)\nINTERVAL (NUMTODSINTERVAL(1, 'DAY'))\n(\n\tPARTITION TEST_PO VALUES LESS THAN (\n\t\tTO_DATE('2000-01-01 00:00:00', 'SYYYY-MM-DD HH24:MI:SS', 'NLS_CALENDAR=GREGORIAN')\n\t)\n)\nONLINE",
						Start:         &storepb.Position{Line: 0, Column: 0},
						End:           &storepb.Position{Line: 8, Column: 0},
						ByteOffsetEnd: 233,
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
