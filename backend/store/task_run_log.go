package store

import (
	"context"
	"database/sql"
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

func (s *Store) CreateTaskRunLogS(ctx context.Context, projectID string, taskRunUID int64, t time.Time, replicaID string, e *storepb.TaskRunLog) {
	if err := s.CreateTaskRunLog(ctx, projectID, taskRunUID, t, replicaID, e); err != nil {
		slog.Error("failed to create task run log", log.BBError(err))
	}
}

func (s *Store) CreateTaskRunLog(ctx context.Context, projectID string, taskRunUID int64, t time.Time, replicaID string, e *storepb.TaskRunLog) error {
	var instanceID string
	if err := s.GetDB().QueryRowContext(ctx, `
		SELECT task.instance
		FROM task_run
		JOIN task ON task.project = task_run.project AND task.id = task_run.task_id
		WHERE task_run.project = $1 AND task_run.id = $2
	`, projectID, taskRunUID).Scan(&instanceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return common.Errorf(common.NotFound, "task run %s/%d not found", projectID, taskRunUID)
		}
		return errors.Wrap(err, "failed to resolve task run lifecycle scope")
	}

	e.ReplicaId = replicaID
	p, err := protojson.Marshal(e)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal task run log")
	}

	q := qb.Q().Space(`
		INSERT INTO task_run_log (
			project,
			task_run_id,
			created_at,
			payload
		) VALUES (
			?,
			?,
			?,
			?
		)
	`, projectID, taskRunUID, t, p)

	sqlText, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	scope := lifecycleScope{}
	scope.addProject(projectID, lifecycleExisting)
	scope.addInstance(instanceID, lifecycleActive)
	return s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, sqlText, args...); err != nil {
			return errors.Wrapf(err, "failed to create task run log")
		}
		return nil
	})
}

func (s *Store) ListTaskRunLogs(ctx context.Context, projectID string, taskRunUID int64) ([]*TaskRunLog, error) {
	// created_at can tie across entries (no per-row sequence exists); ctid
	// breaks ties in insertion order, which is stable because the table is
	// append-only and rows are never updated.
	q := qb.Q().Space(`
		SELECT
			created_at,
			payload
		FROM task_run_log
		WHERE task_run_log.project = ? AND task_run_log.task_run_id = ?
		ORDER BY created_at, ctid
	`, projectID, taskRunUID)

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
