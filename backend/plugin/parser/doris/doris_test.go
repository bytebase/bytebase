package doris

import (
	"testing"

	parser "github.com/bytebase/doris-parser"
	"github.com/stretchr/testify/require"
)

func TestDorisSQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			statement: "SELECT * FROM person LATERAL VIEW EXPLODE(ARRAY(30, 60)) tableName AS c_age;",
		},
		{
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1",
		},
		{
			statement:    "SELECT a > (select max(a) from t1) FROM",
			errorMessage: "Syntax error at line 0:39 \nrelated text: SELECT a > (select max(a) from t1) FROM",
		},
	}

	for _, test := range tests {
		res, err := ParseDorisSQL(test.statement)
		if res != nil {
			_, ok := res.Tree.(*parser.SqlStatementsContext)
			require.True(t, ok)
		}
		if test.errorMessage == "" {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}
