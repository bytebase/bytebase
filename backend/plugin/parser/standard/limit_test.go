package standard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatementWithResultLimit(t *testing.T) {
	tests := []struct {
		statement string
		limit     int
		want      string
	}{
		{"SELECT * FROM t", 10, "WITH result AS (SELECT * FROM t) SELECT * FROM result LIMIT 10;"},
		{"SELECT * FROM t;", 10, "WITH result AS (SELECT * FROM t) SELECT * FROM result LIMIT 10;"},
		{"DELETE FROM t", 5, "WITH result AS (DELETE FROM t) SELECT * FROM result LIMIT 5;"},
	}
	for _, tc := range tests {
		got := statementWithResultLimit(tc.statement, tc.limit, "")
		require.Equalf(t, tc.want, got, "%q", tc.statement)
	}
}
