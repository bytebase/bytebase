package taskrun

import "log/slog"

func taskRunLogAttrs(projectID string, taskRunUID int64) []slog.Attr {
	return []slog.Attr{
		slog.String("project", projectID),
		slog.Int64("task_run_id", taskRunUID),
	}
}

func taskRunLogger(projectID string, taskRunUID int64) *slog.Logger {
	attrs := taskRunLogAttrs(projectID, taskRunUID)
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return slog.With(args...)
}
