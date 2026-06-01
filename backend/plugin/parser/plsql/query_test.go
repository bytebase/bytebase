package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		name        string
		statement   string
		valid       bool
		gotAllQuery bool
		wantErr     bool
	}{
		{
			name:        "single select",
			statement:   "SELECT * FROM t1;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			name:        "multiple selects",
			statement:   "SELECT * FROM t1; SELECT * FROM t2;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			name:        "explain plan",
			statement:   "EXPLAIN PLAN FOR SELECT * FROM t1;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			name:        "explain plan for update",
			statement:   "EXPLAIN PLAN FOR UPDATE t1 SET c1 = 1;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			name:        "cte select",
			statement:   "WITH x AS (SELECT * FROM t1) SELECT * FROM x;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			name:        "dml keywords in string and comment",
			statement:   "WITH x AS (SELECT 'update delete insert' AS c1 FROM t1 /* UPDATE */) SELECT c1 FROM x;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			name:      "mixed select and update rejected",
			statement: "SELECT * FROM t1; UPDATE t1 SET c1 = 1;",
		},
		{
			name:      "cte update rejected",
			statement: "WITH x AS (SELECT * FROM t1) UPDATE t1 SET c1 = 1;",
			wantErr:   true,
		},
		{
			name:      "sqlplus command rejected",
			statement: "SET DEFINE OFF\nSELECT * FROM t1;",
		},
		{
			name:      "update rejected",
			statement: "UPDATE t1 SET c1 = 1;",
		},
		{
			name:      "create rejected",
			statement: "CREATE TABLE t1 (c1 INT);",
		},
		{
			name:      "syntax error",
			statement: "SELECT 1 FROM DUAL;\nSELECT * FROM",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotValid, gotAllQuery, err := validateQuery(tc.statement)
			if tc.wantErr {
				require.Error(t, err)
				require.ErrorAs(t, err, new(*base.SyntaxError))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.valid, gotValid)
			require.Equal(t, tc.gotAllQuery, gotAllQuery)
		})
	}
}
