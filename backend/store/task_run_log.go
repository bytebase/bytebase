package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (s *Store) CreateTaskRunLog(ctx context.Context, taskRunUID int, t time.Time, e *storepb.TaskRunLog) error {
	query := `
		INSERT INTO task_run_log (
			task_run_id,
			created_ts,
			payload
		) VALUES (
			$1,
			$2,
			$3
		)
	`
	p, err := protojson.Marshal(e)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal task run log")
	}
	if _, err := s.db.db.ExecContext(ctx, query, taskRunUID, t, p); err != nil {
		return errors.Wrapf(err, "failed to create task run log")
	}
	return nil
}
