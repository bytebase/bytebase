package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithSearchPath(t *testing.T) {
	statement := WithSearchPath("DELETE FROM t;", []string{"app", `we"ird`})
	require.Equal(t, "SET LOCAL search_path TO \"app\", \"we\"\"ird\";\nDELETE FROM t;", statement)

	setup, rest := SplitSearchPath(statement)
	require.Equal(t, `SET LOCAL search_path TO "app", "we""ird"`, setup)
	require.Equal(t, "DELETE FROM t;", rest)

	setup, rest = SplitSearchPath("DELETE FROM t;")
	require.Empty(t, setup)
	require.Equal(t, "DELETE FROM t;", rest)
}
