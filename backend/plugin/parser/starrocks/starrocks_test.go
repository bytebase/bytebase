package starrocks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStarRocksSQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			// StarRocks has no Hive-style LATERAL VIEW (its lateral form is
			// `, [LATERAL] unnest(...)`) — container-verified engine reject.
			// The old parser only "accepted" this by silently truncating at
			// LATERAL (BYT-10085); strict parsing reports it like the engine.
			statement:    "SELECT * FROM person LATERAL VIEW EXPLODE(ARRAY(30, 60)) tableName AS c_age;",
			errorMessage: "syntax error at or near LATERAL",
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
		res, err := parseStarRocksSQL(test.statement)
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
