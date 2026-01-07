package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type TaskRunLog struct {
	T       time.Time
	Payload *storepb.TaskRunLog
}

func (s *Store) CreateTaskRunLogS(ctx context.Context, taskRunUID int, t time.Time, replicaID string, e *storepb.TaskRunLog) {
	if err := s.CreateTaskRunLog(ctx, taskRunUID, t, replicaID, e); err != nil {
		slog.Error("failed to create task run log", log.BBError(err))
	}
}

func (s *Store) CreateTaskRunLog(ctx context.Context, taskRunUID int, t time.Time, replicaID string, e *storepb.TaskRunLog) error {
	e.ReplicaId = replicaID
	p, err := protojson.Marshal(e)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal task run log")
	}

	q := qb.Q().Space(`
		INSERT INTO task_run_log (
			task_run_id,
			created_at,
			payload
		) VALUES (
			?,
			?,
			?
		)
	`, taskRunUID, t, p)

	sql, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to create task run log")
	}
	return nil
}

func (s *Store) ListTaskRunLogs(ctx context.Context, taskRunUID int) ([]*TaskRunLog, error) {
	q := qb.Q().Space(`
		SELECT
			created_at,
			payload
		FROM task_run_log
		WHERE task_run_log.task_run_id = ?
		ORDER BY id
	`, taskRunUID)

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, sql, args...)
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
