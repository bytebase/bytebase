package stacktrace

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTakeStacktrace(t *testing.T) {
	buf := TakeStacktrace(10, 0)
	lines := strings.Split(string(buf), "\n")
	require.Contains(t, lines[0], "TestTakeStacktrace")
	require.Contains(t, lines[1], "stack_test.go:11")
	require.Contains(t, lines[2], "testing")
	require.Contains(t, lines[3], "testing.go")
}
