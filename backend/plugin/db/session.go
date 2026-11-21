package db

import (
	"context"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TaskRunSessionReader is the capability a driver offers when its engine can
// describe one of its own sessions by application_name, with the sessions it
// blocks and the sessions blocking it (pg: pg_stat_activity and
// pg_blocking_pids; cockroachdb: crdb_internal.cluster_queries and
// crdb_internal.cluster_sessions).
type TaskRunSessionReader interface {
	TaskRunSession(ctx context.Context, applicationName string) (*v1pb.TaskRunSession, error)
}
