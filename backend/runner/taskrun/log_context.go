package taskrun

import "log/slog"

func taskRunLogAttrs(projectID string, taskRunUID int64, replicaID string) []slog.Attr {
	return []slog.Attr{
		slog.String("project", projectID),
		slog.Int64("task_run_id", taskRunUID),
		slog.String("replica_id", replicaID),
	}
}

func taskRunLogger(projectID string, taskRunUID int64, replicaID string) *slog.Logger {
	attrs := taskRunLogAttrs(projectID, taskRunUID, replicaID)
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return slog.With(args...)
}
