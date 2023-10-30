package pg

import (
	"testing"

	parser "github.com/bytebase/postgresql-parser"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
		},
		{
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1",
		},
		{
			statement:    "SELECT a > (select max(a) from t1) FROM",
			errorMessage: "Syntax error at line 1:39 \nrelated text: SELECT a > (select max(a) from t1) FROM",
		},
	}

	for _, test := range tests {
		res, err := ParsePostgreSQL(test.statement)
		if res != nil {
			_, ok := res.Tree.(*parser.RootContext)
			require.True(t, ok)
		}
		if test.errorMessage == "" {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}
