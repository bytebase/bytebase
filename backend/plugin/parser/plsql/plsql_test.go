package plsql

import (
	"testing"

	parser "github.com/bytebase/plsql-parser"
	"github.com/stretchr/testify/require"
)

func TestPLSQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
		},
		{
			statement: "CREATE TABLE t1 (c1 NUMBER(10,2), c2 VARCHAR2(10));",
		},
		{
			statement: "SELECT * FROM t1;",
		},
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1",
		},
		{
			statement:    "SELECT * FROM t1 WHERE c1 = ",
			errorMessage: "Syntax error at line 1:26 \nrelated text: SELECT * FROM t1 WHERE c1 =",
		},
	}

	for _, test := range tests {
		tree, _, err := ParsePLSQL(test.statement)
		_, ok := tree.(*parser.Sql_scriptContext)
		if test.errorMessage == "" {
			require.True(t, ok)
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}
