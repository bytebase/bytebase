package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitSQLWithBeginEnd(t *testing.T) {
	// Test with CASE expression (no semicolons inside CASE in SQL)
	statement := `SELECT
  CASE
    WHEN status = 'active' THEN 1
    WHEN status = 'inactive' THEN 0
  END
FROM users;
SELECT * FROM orders;`

	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, 2, len(list), "Should split into 2 statements")

	// First statement: SELECT with CASE
	require.Contains(t, list[0].Text, "SELECT")
	require.Contains(t, list[0].Text, "CASE")
	require.Contains(t, list[0].Text, "END")
	require.False(t, list[0].Empty)

	// Second statement: SELECT FROM orders
	require.Contains(t, list[1].Text, "SELECT * FROM orders")
	require.False(t, list[1].Empty)
}
