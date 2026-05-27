package taskrun

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/bytebase/bytebase/backend/common/log"

	"github.com/stretchr/testify/require"
)

func TestTaskRunLogAttrs(t *testing.T) {
	attrs := taskRunLogAttrs("project-a", 123)

	require.Equal(t, []slog.Attr{
		slog.String("project", "project-a"),
		slog.Int64("task_run_id", 123),
	}, attrs)
}

func TestTaskRunLogContext(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewContextHandler(slog.NewTextHandler(&buf, nil)))

	ctx := taskRunLogContext(context.Background(), "project-a", 123)
	logger.InfoContext(ctx, "task run started")

	output := buf.String()
	require.Contains(t, output, `project=project-a`)
	require.Contains(t, output, `task_run_id=123`)
}
