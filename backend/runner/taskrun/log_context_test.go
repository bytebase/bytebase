package taskrun

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskRunLogAttrs(t *testing.T) {
	attrs := taskRunLogAttrs("project-a", 123)

	require.Equal(t, []slog.Attr{
		slog.String("project", "project-a"),
		slog.Int64("task_run_id", 123),
	}, attrs)
}
