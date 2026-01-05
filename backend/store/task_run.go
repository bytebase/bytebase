package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TaskRunMessage is message for task run.
type TaskRunMessage struct {
	TaskUID     int
	Environment string // Refer to the task's environment.
	PlanUID     int64
	Status      storepb.TaskRun_Status
	ResultProto *storepb.TaskRunResult

	// Output only.
	ID           int
	CreatorEmail string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ProjectID    string
	StartedAt    *time.Time
	RunAt        *time.Time
}

// FindTaskRunMessage is the message for finding task runs.
type FindTaskRunMessage struct {
	UID         *int
	UIDs        *[]int
	TaskUID     *int
	Environment *string
	PlanUID     *int64
	Status      *[]storepb.TaskRun_Status
}

// TaskRunStatusPatch is the API message for patching a task run.
type TaskRunStatusPatch struct {
	ID int

	// Standard fields
	Updater string

	// Domain specific fields
	Status      storepb.TaskRun_Status
	ResultProto *storepb.TaskRunResult
}

// ListTaskRuns lists task runs.
func (s *Store) ListTaskRuns(ctx context.Context, find *FindTaskRunMessage) ([]*TaskRunMessage, error) {
	q := qb.Q().Space(`
		SELECT
			task_run.id,
			task_run.creator,
			task_run.created_at,
			task_run.updated_at,
			task_run.task_id,
			task_run.status,
			task_run.started_at,
			task_run.run_at,
			task_run.result,
			task.plan_id,
			task.environment,
			project.resource_id
		FROM task_run
		LEFT JOIN task ON task.id = task_run.task_id
		LEFT JOIN plan ON plan.id = task.plan_id
		LEFT JOIN project ON project.resource_id = plan.project
		WHERE TRUE
	`)

	if v := find.UID; v != nil {
		q.And("task_run.id = ?", *v)
	}
	if v := find.UIDs; v != nil {
		q.And("task_run.id = ANY(?)", *v)
	}
	if v := find.TaskUID; v != nil {
		q.And("task_run.task_id = ?", *v)
	}
	if v := find.Environment; v != nil {
		q.And("task.environment = ?", *v)
	}
	if v := find.PlanUID; v != nil {
		q.And("task.plan_id = ?", *v)
	}
	if v := find.Status; v != nil {
		var statusStrings []string
		for _, status := range *v {
			statusStrings = append(statusStrings, status.String())
		}
		q.And("task_run.status = ANY(?)", statusStrings)
	}

	q.Space("ORDER BY task_run.id ASC")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taskRuns []*TaskRunMessage
	for rows.Next() {
		var taskRun TaskRunMessage
		var startedAt, runAt sql.NullTime
		var statusString string
		var resultJSON string
		if err := rows.Scan(
			&taskRun.ID,
			&taskRun.CreatorEmail,
			&taskRun.CreatedAt,
			&taskRun.UpdatedAt,
			&taskRun.TaskUID,
			&statusString,
			&startedAt,
			&runAt,
			&resultJSON,
			&taskRun.PlanUID,
			&taskRun.Environment,
			&taskRun.ProjectID,
		); err != nil {
			return nil, err
		}
		if statusValue, ok := storepb.TaskRun_Status_value[statusString]; ok {
			taskRun.Status = storepb.TaskRun_Status(statusValue)
		} else {
			return nil, errors.Errorf("invalid task run status string: %s", statusString)
		}

		if startedAt.Valid {
			taskRun.StartedAt = &startedAt.Time
		}
		if runAt.Valid {
			taskRun.RunAt = &runAt.Time
		}
		var resultProto storepb.TaskRunResult
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(resultJSON), &resultProto); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal task run result: %s", resultJSON)
		}
		taskRun.ResultProto = &resultProto

		taskRuns = append(taskRuns, &taskRun)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return taskRuns, nil
}

// GetTaskRunV1 gets a task run.
func (s *Store) GetTaskRunV1(ctx context.Context, find *FindTaskRunMessage) (*TaskRunMessage, error) {
	taskRuns, err := s.ListTaskRuns(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(taskRuns) == 0 {
		return nil, errors.Errorf("task run not found")
	}
	if len(taskRuns) > 1 {
		return nil, errors.Errorf("expected to get one task run, but got %d", len(taskRuns))
	}
	return taskRuns[0], nil
}

// GetTaskRunByUID gets a task run by uid.
func (s *Store) GetTaskRunByUID(ctx context.Context, uid int) (*TaskRunMessage, error) {
	return s.GetTaskRunV1(ctx, &FindTaskRunMessage{UID: &uid})
}

// UpdateTaskRunStatus updates task run status.
func (s *Store) UpdateTaskRunStatus(ctx context.Context, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	taskRun, err := s.patchTaskRunStatusImpl(ctx, tx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update task run")
	}

	// Get the plan ID for cache invalidation
	q := qb.Q().Space("SELECT plan_id FROM task WHERE id = ?", taskRun.TaskUID)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var planID int64
	if err := tx.QueryRowContext(ctx, sql, args...).Scan(&planID); err != nil {
		return nil, errors.Wrapf(err, "failed to get plan ID for task %d", taskRun.TaskUID)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}
	return taskRun, nil
}

// ClaimedTaskRun represents a claimed task run with its task UID.
type ClaimedTaskRun struct {
	TaskRunUID int
	TaskUID    int
}

// ClaimAvailableTaskRuns atomically claims all AVAILABLE task runs by updating them to RUNNING
// and returns the claimed task run and task UIDs. This combines list + claim into a single atomic operation.
// Uses FOR UPDATE SKIP LOCKED to allow concurrent schedulers to claim different tasks.
func (s *Store) ClaimAvailableTaskRuns(ctx context.Context) ([]*ClaimedTaskRun, error) {
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now()
		WHERE id IN (
			SELECT task_run.id FROM task_run
			WHERE task_run.status = ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, task_id
	`, storepb.TaskRun_RUNNING.String(), storepb.TaskRun_AVAILABLE.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to claim task runs")
	}
	defer rows.Close()

	var claimed []*ClaimedTaskRun
	for rows.Next() {
		var c ClaimedTaskRun
		if err := rows.Scan(&c.TaskRunUID, &c.TaskUID); err != nil {
			return nil, err
		}
		claimed = append(claimed, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s *Store) UpdateTaskRunStartAt(ctx context.Context, taskRunID int) error {
	// Get the pipeline ID for cache invalidation
	q := qb.Q().Space(`
		UPDATE task_run
		SET started_at = now(), updated_at = now()
		WHERE id = ?
		RETURNING (SELECT plan_id FROM task WHERE task.id = task_run.task_id)
	`, taskRunID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	var planID int64
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&planID); err != nil {
		return errors.Wrapf(err, "failed to update task run start at")
	}
	return nil
}

// CreatePendingTaskRuns creates pending task runs.
// This operation is idempotent and safe for concurrent calls:
// - Uses WHERE NOT EXISTS to skip tasks that already have active (PENDING/RUNNING/DONE) task runs
// - Uses ON CONFLICT DO NOTHING to handle race conditions where two requests try to create the same task run
// - The unique constraint on (task_id, attempt) ensures no duplicates
func (s *Store) CreatePendingTaskRuns(ctx context.Context, creator string, creates ...*TaskRunMessage) error {
	if len(creates) == 0 {
		return nil
	}

	var taskUIDs []int
	var runAts []*time.Time
	for _, create := range creates {
		taskUIDs = append(taskUIDs, create.TaskUID)
		runAts = append(runAts, create.RunAt)
	}

	// Single query that:
	// 1. Filters out tasks with existing PENDING/RUNNING/DONE task runs (idempotent)
	// 2. Calculates next attempt for each remaining task
	// 3. Inserts task runs
	// 4. Uses ON CONFLICT DO NOTHING to handle race conditions
	q := qb.Q().Space(`
		INSERT INTO task_run (
			creator,
			task_id,
			run_at,
			attempt,
			status
		)
		SELECT
			?,
			tasks.task_id,
			tasks.run_at,
			COALESCE((SELECT MAX(attempt) + 1 FROM task_run WHERE task_run.task_id = tasks.task_id), 0) as attempt,
			?
		FROM (
			SELECT
				unnest(CAST(? AS INTEGER[])) AS task_id,
				unnest(CAST(? AS TIMESTAMPTZ[])) AS run_at
		) tasks
		WHERE NOT EXISTS (
			SELECT 1 FROM task_run
			WHERE task_run.task_id = tasks.task_id
			AND task_run.status IN (?, ?, ?, ?)
		)
		ON CONFLICT (task_id, attempt) DO NOTHING
	`, creator, storepb.TaskRun_PENDING.String(), taskUIDs, runAts,
		storepb.TaskRun_PENDING.String(), storepb.TaskRun_AVAILABLE.String(), storepb.TaskRun_RUNNING.String(), storepb.TaskRun_DONE.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	return nil
}

// patchTaskRunStatusImpl updates a taskRun status. Returns the new state of the taskRun after update.
func (*Store) patchTaskRunStatusImpl(ctx context.Context, txn *sql.Tx, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	set := qb.Q()

	set.Comma("updated_at = ?, status = ?", time.Now(), patch.Status.String())

	if v := patch.ResultProto; v != nil {
		resultBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal task run result")
		}
		result := string(resultBytes)
		if result == "" {
			result = "{}"
		}
		set.Comma("result = ?", result)
	}

	q := qb.Q().Space("UPDATE task_run SET ?", set).
		Space("WHERE id = ?", patch.ID).
		Space("RETURNING id, creator, created_at, updated_at, task_id, status, result")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var taskRun TaskRunMessage
	var statusString string
	var resultJSON string
	if err := txn.QueryRowContext(ctx, query, args...).Scan(
		&taskRun.ID,
		&taskRun.CreatorEmail,
		&taskRun.CreatedAt,
		&taskRun.UpdatedAt,
		&taskRun.TaskUID,
		&statusString,
		&resultJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	if statusValue, ok := storepb.TaskRun_Status_value[statusString]; ok {
		taskRun.Status = storepb.TaskRun_Status(statusValue)
	} else {
		return nil, errors.Errorf("invalid task run status string: %s", statusString)
	}
	var resultProto storepb.TaskRunResult
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(resultJSON), &resultProto); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task run result: %s", resultJSON)
	}
	taskRun.ResultProto = &resultProto
	return &taskRun, nil
}

// BatchCancelTaskRuns updates the status of taskRuns to CANCELED.
func (s *Store) BatchCancelTaskRuns(ctx context.Context, taskRunIDs []int) error {
	if len(taskRunIDs) == 0 {
		return nil
	}
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now()
		WHERE id = ANY(?)
	`, storepb.TaskRun_CANCELED.String(), taskRunIDs)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}
