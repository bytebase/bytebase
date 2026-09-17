package standard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplainStatement(t *testing.T) {
	for _, statement := range []string{"SELECT 1", "SELECT 'EXPLAIN'", "SELECT explained FROM t"} {
		got, err := ExplainStatement(statement, "")
		require.NoError(t, err, statement)
		require.Equal(t, "EXPLAIN "+statement, got)
	}
	for _, statement := range []string{
		"EXPLAIN SELECT 1",
		"explain analyze select 1",
		"/* plan */ explain\nSELECT 1",
		"-- plan\nEXPLAIN FORMAT=JSON SELECT 1",
	} {
		_, err := ExplainStatement(statement, "")
		require.Error(t, err, statement)
	}
}
