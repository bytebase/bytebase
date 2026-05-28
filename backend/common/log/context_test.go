package log //nolint:revive // intentional package name

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextHandlerAddsAttrsFromContext(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewTextHandler(&buf, nil)))
	ctx := WithAttrs(context.Background(),
		slog.String("project", "db333"),
		slog.Int64("task_run_id", 9213),
	)

	logger.InfoContext(ctx, "migration started")

	output := buf.String()
	require.Contains(t, output, `msg="migration started"`)
	require.Contains(t, output, `project=db333`)
	require.Contains(t, output, `task_run_id=9213`)
}

func TestContextHandlerSkipsEmptyContext(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewTextHandler(&buf, nil)))

	logger.InfoContext(context.Background(), "migration started")

	output := buf.String()
	require.Contains(t, output, `msg="migration started"`)
	require.NotContains(t, output, `project=`)
	require.NotContains(t, output, `task_run_id=`)
}
