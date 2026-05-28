package taskrun

import (
	"context"
	"log/slog"

	"github.com/bytebase/bytebase/backend/common/log"
)

func taskRunLogAttrs(projectID string, taskRunUID int64) []slog.Attr {
	return []slog.Attr{
		slog.String("project", projectID),
		slog.Int64("task_run_id", taskRunUID),
	}
}

func taskRunLogContext(ctx context.Context, projectID string, taskRunUID int64) context.Context {
	return log.WithAttrs(ctx, taskRunLogAttrs(projectID, taskRunUID)...)
}
