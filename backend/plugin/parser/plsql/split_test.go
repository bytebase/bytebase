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
						Text:            `select * from t`,
						LastLine:        2,
						ByteOffsetStart: 5,
						ByteOffsetEnd:   20,
					},
					{
						Text:            `create table table$1 (id int)`,
						LastLine:        3,
						ByteOffsetStart: 26,
						ByteOffsetEnd:   56,
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
						Text:          "ALTER TABLE DATA.TEST\nMODIFY PARTITION BY RANGE (TXN_DATE)\nINTERVAL (NUMTODSINTERVAL(1, 'DAY'))\n(\n\t\tPARTITION TEST_PO VALUES LESS THAN (\n\t\t\t\tTO_DATE('2000-01-01 00:00:00', 'SYYYY-MM-DD HH24:MI:SS', 'NLS_CALENDAR=GREGORIAN')\n\t\t)\n)\nONLINE",
						LastLine:      9,
						ByteOffsetEnd: 237,
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
