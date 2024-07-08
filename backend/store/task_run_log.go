package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type TaskRunLog struct {
	T       time.Time
	Payload *storepb.TaskRunLog
}

func (s *Store) CreateTaskRunLogS(ctx context.Context, taskRunUID int, t time.Time, e *storepb.TaskRunLog) {
	if err := s.CreateTaskRunLog(ctx, taskRunUID, t, e); err != nil {
		slog.Error("failed to create task run log", log.BBError(err))
	}
}

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

func (s *Store) ListTaskRunLogs(ctx context.Context, taskRunUID int) ([]*TaskRunLog, error) {
	query := `
		SELECT
			created_ts,
			payload
		FROM task_run_log
		WHERE task_run_log.task_run_id = $1
		ORDER BY id
	`

	rows, err := s.db.db.QueryContext(ctx, query, taskRunUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query task run log")
	}
	defer rows.Close()

	var logs []*TaskRunLog
	for rows.Next() {
		l := TaskRunLog{
			Payload: &storepb.TaskRunLog{},
		}
		var p []byte

		if err := rows.Scan(
			&l.T,
			&p,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(p, l.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}

		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}

	return logs, nil
}
