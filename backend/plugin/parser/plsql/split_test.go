package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestOracleSplitMultiSQL(t *testing.T) {
	type resData struct {
		res []base.Statement
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
				res: []base.Statement{
					{
						Text:            `select * from t`,
						BaseLine:        1,
						ByteOffsetStart: 5,
						ByteOffsetEnd:   20,
						Start:           &storepb.Position{Line: 2, Column: 5},
						End:             &storepb.Position{Line: 2, Column: 19},
					},
					{
						Text:            `create table table$1 (id int)`,
						BaseLine:        2,
						ByteOffsetStart: 26,
						ByteOffsetEnd:   56,
						Start:           &storepb.Position{Line: 3, Column: 5},
						End:             &storepb.Position{Line: 3, Column: 33},
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
				res: []base.Statement{
					{
						Text:          "ALTER TABLE DATA.TEST\nMODIFY PARTITION BY RANGE (TXN_DATE)\nINTERVAL (NUMTODSINTERVAL(1, 'DAY'))\n(\n\tPARTITION TEST_PO VALUES LESS THAN (\n\t\tTO_DATE('2000-01-01 00:00:00', 'SYYYY-MM-DD HH24:MI:SS', 'NLS_CALENDAR=GREGORIAN')\n\t)\n)\nONLINE",
						Start:         &storepb.Position{Line: 1, Column: 1},
						End:           &storepb.Position{Line: 9, Column: 1},
						ByteOffsetEnd: 233,
					},
				},
			},
		},
		// BYT-8268: Test cases for CALL keyword requirement
		// This prevents misidentifying keywords like CASCADE as procedure calls
		{
			statement: "DROP TABLESPACE xxx CASCADE",
			want: resData{
				err: "Syntax error at line 1:28 \nrelated text: DROP TABLESPACE xxx CASCADE;",
			},
		},
		{
			statement: "DROP TABLESPACE xxx; CASCADE",
			want: resData{
				err: "Syntax error at line 1:29 \nrelated text: DROP TABLESPACE xxx; CASCADE;",
			},
		},
		{
			statement: "DROP TABLESPACE xxx INCLUDING CONTENTS CASCADE CONSTRAINTS",
			want: resData{
				res: []base.Statement{
					{
						Text:          "DROP TABLESPACE xxx INCLUDING CONTENTS CASCADE CONSTRAINTS",
						Start:         &storepb.Position{Line: 1, Column: 1},
						End:           &storepb.Position{Line: 1, Column: 48},
						ByteOffsetEnd: 58,
					},
				},
			},
		},
		{
			statement: "CALL my_procedure()",
			want: resData{
				res: []base.Statement{
					{
						Text:          "CALL my_procedure()",
						Start:         &storepb.Position{Line: 1, Column: 1},
						End:           &storepb.Position{Line: 1, Column: 19},
						ByteOffsetEnd: 19,
					},
				},
			},
		},
		{
			statement: "SELECT * FROM t1; SELECT * FROM t2",
			want: resData{
				res: []base.Statement{
					{
						Text:            "SELECT * FROM t1",
						Start:           &storepb.Position{Line: 1, Column: 1},
						End:             &storepb.Position{Line: 1, Column: 15},
						ByteOffsetStart: 0,
						ByteOffsetEnd:   16,
					},
					{
						Text:            "SELECT * FROM t2",
						Start:           &storepb.Position{Line: 1, Column: 19},
						End:             &storepb.Position{Line: 1, Column: 33},
						ByteOffsetStart: 18,
						ByteOffsetEnd:   34,
					},
				},
			},
		},
		// Test forward slash (/) as statement separator in PL/SQL
		{
			statement: `CREATE OR REPLACE PROCEDURE test_proc IS
BEGIN
    CALL DBMS_OUTPUT.PUT_LINE('Hello');
END;
/`,
			want: resData{
				res: []base.Statement{
					{
						Text:          "CREATE OR REPLACE PROCEDURE test_proc IS\nBEGIN\n    CALL DBMS_OUTPUT.PUT_LINE('Hello');\nEND;",
						Start:         &storepb.Position{Line: 1, Column: 1},
						End:           &storepb.Position{Line: 4, Column: 4},
						ByteOffsetEnd: 91,
					},
				},
			},
		},
		{
			statement: `CREATE OR REPLACE PROCEDURE proc1 IS
BEGIN
    NULL;
END;
/
CREATE OR REPLACE PROCEDURE proc2 IS
BEGIN
    NULL;
END;
/`,
			want: resData{
				res: []base.Statement{
					{
						Text:            "CREATE OR REPLACE PROCEDURE proc1 IS\nBEGIN\n    NULL;\nEND;",
						BaseLine:        0,
						Start:           &storepb.Position{Line: 1, Column: 1},
						End:             &storepb.Position{Line: 4, Column: 4},
						ByteOffsetStart: 0,
						ByteOffsetEnd:   57,
					},
					{
						Text:            "CREATE OR REPLACE PROCEDURE proc2 IS\nBEGIN\n    NULL;\nEND;",
						BaseLine:        5,
						Start:           &storepb.Position{Line: 6, Column: 1},
						End:             &storepb.Position{Line: 9, Column: 4},
						ByteOffsetStart: 60,
						ByteOffsetEnd:   117,
					},
				},
			},
		},
		{
			statement: `SELECT * FROM t1;
/
SELECT * FROM t2;`,
			want: resData{
				res: []base.Statement{
					{
						Text:            "SELECT * FROM t1",
						BaseLine:        0,
						Start:           &storepb.Position{Line: 1, Column: 1},
						End:             &storepb.Position{Line: 1, Column: 15},
						ByteOffsetStart: 0,
						ByteOffsetEnd:   16,
					},
					{
						Text:            "SELECT * FROM t2",
						BaseLine:        2,
						Start:           &storepb.Position{Line: 3, Column: 1},
						End:             &storepb.Position{Line: 3, Column: 15},
						ByteOffsetStart: 20,
						ByteOffsetEnd:   36,
					},
				},
			},
		},
		{
			statement: `BEGIN
    CALL DBMS_OUTPUT.PUT_LINE('Test');
END;
/`,
			want: resData{
				res: []base.Statement{
					{
						Text:          "BEGIN\n    CALL DBMS_OUTPUT.PUT_LINE('Test');\nEND;",
						Start:         &storepb.Position{Line: 1, Column: 1},
						End:           &storepb.Position{Line: 3, Column: 4},
						ByteOffsetEnd: 49,
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
