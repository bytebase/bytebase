package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatSQLText(t *testing.T) {
	tests := []struct {
		before string
		after  string
	}{
		{
			before: `
				SELECT
				*
				FROM
				t
				WHERE
				a > 0 and b < 1;
			`,
			after: "SELECT * FROM t WHERE a > 0 and b < 1;",
		},
		{
			before: `
			SELECT
			* FROM t
			WHERE
			a LIKE 'abcd\'xxx\n'
			`,
			after: `SELECT * FROM t WHERE a LIKE 'abcd\'xxx\n'`,
		},
		{
			before: "SELECT * FROM `t`			 WHERE a != 'Chinese \\'      测试'",
			after: "SELECT * FROM `t` WHERE a != 'Chinese \\'      测试'",
		},
	}

	for _, test := range tests {
		require.Equal(t, test.after, formatSQLText(test.before))
	}
}
