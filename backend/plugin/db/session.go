package db

import (
	"context"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TaskRunSessionReader is the capability a driver offers when its engine can
// describe one of its own sessions by application_name, with the sessions it
// blocks and the sessions blocking it (pg: pg_stat_activity and
// pg_blocking_pids). A driver without it is read through SQLDB, and a caller
// that must keep working for both asks for this interface first.
type TaskRunSessionReader interface {
	TaskRunSession(ctx context.Context, applicationName string) (*v1pb.TaskRunSession, error)
}
