package pg

import (
	"testing"

	parser "github.com/bytebase/parser/postgresql"
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
			errorMessage: "Syntax error at line 1:40 \nrelated text: SELECT a > (select max(a) from t1) FROM",
		},
		{
			statement:    "SELECT 1;\n   SELEC 5;\nSELECT 6;",
			errorMessage: "Syntax error at line 2:4 \nrelated text: \n   SELEC",
		},
	}

	for _, test := range tests {
		parseResults, err := ParsePostgreSQL(test.statement)
		// Assert each element's Tree is a RootContext
		for i, result := range parseResults {
			_, ok := result.Tree.(*parser.RootContext)
			require.True(t, ok, "parseResults[%d].Tree should be a RootContext", i)
		}
		if test.errorMessage == "" {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}
