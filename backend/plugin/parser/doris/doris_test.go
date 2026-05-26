package doris

import (
	"testing"

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
			// Truncated SELECT — must be reported as a syntax error.
			statement:    "SELECT a > (select max(a) from t1) FROM",
			errorMessage: "syntax error at end of input",
		},
	}

	for _, test := range tests {
		res, err := parseDorisSQL(test.statement)
		if test.errorMessage == "" {
			require.NoError(t, err)
			require.NotEmpty(t, res)
			require.NotNil(t, res[0].Node())
		} else {
			require.Error(t, err)
			require.Contains(t, err.Error(), test.errorMessage)
		}
	}
}
