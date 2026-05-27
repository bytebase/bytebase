package taskrun

import "log/slog"

func taskRunLogAttrs(projectID string, taskRunUID int64, replicaID string) []slog.Attr {
	return []slog.Attr{
		slog.String("project", projectID),
		slog.Int64("task_run_id", taskRunUID),
		slog.String("replica_id", replicaID),
	}
}
