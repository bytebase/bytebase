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
	TaskResourceID string
	Environment    string // Refer to the task's environment.
	PlanResourceID string
	Status         storepb.TaskRun_Status
	ResultProto    *storepb.TaskRunResult
	PayloadProto   *storepb.TaskRunPayload

	// Output only.
	ResourceID   string
	CreatorEmail string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ProjectID    string
	StartedAt    *time.Time
	RunAt        *time.Time
}

// FindTaskRunMessage is the message for finding task runs.
type FindTaskRunMessage struct {
	ResourceID     *string
	ResourceIDs    *[]string
	TaskResourceID *string
	Environment    *string
	PlanResourceID *string
	Status         *[]storepb.TaskRun_Status
}

// TaskRunStatusPatch is the API message for patching a task run.
type TaskRunStatusPatch struct {
	ResourceID string

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
			task_run.resource_id,
			task_run.creator,
			task_run.created_at,
			task_run.updated_at,
			task_run.task_id,
			task_run.status,
			task_run.started_at,
			task_run.run_at,
			task_run.result,
			task_run.payload,
			task.plan_id,
			task.environment,
			project.resource_id
		FROM task_run
		LEFT JOIN task ON task.resource_id = task_run.task_id
		LEFT JOIN plan ON plan.resource_id = task.plan_id
		LEFT JOIN project ON project.resource_id = plan.project
		WHERE TRUE
	`)

	if v := find.ResourceID; v != nil {
		q.And("task_run.resource_id = ?", *v)
	}
	if v := find.ResourceIDs; v != nil {
		q.And("task_run.resource_id = ANY(?)", *v)
	}
	if v := find.TaskResourceID; v != nil {
		q.And("task_run.task_id = ?", *v)
	}
	if v := find.Environment; v != nil {
		q.And("task.environment = ?", *v)
	}
	if v := find.PlanResourceID; v != nil {
		q.And("task.plan_id = ?", *v)
	}
	if v := find.Status; v != nil {
		var statusStrings []string
		for _, status := range *v {
			statusStrings = append(statusStrings, status.String())
		}
		q.And("task_run.status = ANY(?)", statusStrings)
	}

	q.Space("ORDER BY task_run.created_at ASC")

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
		var creatorEmail sql.NullString
		var statusString string
		var resultJSON, payloadJSON string
		if err := rows.Scan(
			&taskRun.ResourceID,
			&creatorEmail,
			&taskRun.CreatedAt,
			&taskRun.UpdatedAt,
			&taskRun.TaskResourceID,
			&statusString,
			&startedAt,
			&runAt,
			&resultJSON,
			&payloadJSON,
			&taskRun.PlanResourceID,
			&taskRun.Environment,
			&taskRun.ProjectID,
		); err != nil {
			return nil, err
		}
		if creatorEmail.Valid {
			taskRun.CreatorEmail = creatorEmail.String
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
		var payloadProto storepb.TaskRunPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadJSON), &payloadProto); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal task run payload: %s", payloadJSON)
		}
		taskRun.PayloadProto = &payloadProto

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

// GetTaskRunByResourceID gets a task run by resource ID.
func (s *Store) GetTaskRunByResourceID(ctx context.Context, resourceID string) (*TaskRunMessage, error) {
	return s.GetTaskRunV1(ctx, &FindTaskRunMessage{ResourceID: &resourceID})
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
	q := qb.Q().Space("SELECT plan_id FROM task WHERE resource_id = ?", taskRun.TaskResourceID)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var planID string
	if err := tx.QueryRowContext(ctx, sql, args...).Scan(&planID); err != nil {
		return nil, errors.Wrapf(err, "failed to get plan ID for task %s", taskRun.TaskResourceID)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}
	return taskRun, nil
}

// ClaimedTaskRun represents a claimed task run with its task resource ID.
type ClaimedTaskRun struct {
	TaskRunResourceID string
	TaskResourceID    string
}

// ClaimAvailableTaskRuns atomically claims all AVAILABLE task runs by updating them to RUNNING
// and returns the claimed task run and task resource IDs. This combines list + claim into a single atomic operation.
// Uses FOR UPDATE SKIP LOCKED to allow concurrent schedulers to claim different tasks.
func (s *Store) ClaimAvailableTaskRuns(ctx context.Context, replicaID string) ([]*ClaimedTaskRun, error) {
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now(), replica_id = ?
		WHERE resource_id IN (
			SELECT task_run.resource_id FROM task_run
			WHERE task_run.status = ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING resource_id, task_id
	`, storepb.TaskRun_RUNNING.String(), replicaID, storepb.TaskRun_AVAILABLE.String())

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
		if err := rows.Scan(&c.TaskRunResourceID, &c.TaskResourceID); err != nil {
			return nil, err
		}
		claimed = append(claimed, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s *Store) UpdateTaskRunStartAt(ctx context.Context, taskRunID string) error {
	// Get the pipeline ID for cache invalidation
	q := qb.Q().Space(`
		UPDATE task_run
		SET started_at = now(), updated_at = now()
		WHERE resource_id = ?
		RETURNING (SELECT plan_id FROM task WHERE task.resource_id = task_run.task_id)
	`, taskRunID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	var planID string
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

	var taskUIDs []string
	var runAts []*time.Time
	for _, create := range creates {
		taskUIDs = append(taskUIDs, create.TaskResourceID)
		runAts = append(runAts, create.RunAt)
	}

	// Convert empty string to NULL for system-created task runs
	var creatorPtr any
	if creator == "" {
		creatorPtr = nil
	} else {
		creatorPtr = creator
	}

	// Serialize payload from the first create (all creates in a batch share the same payload).
	payloadStr := "{}"
	if len(creates) > 0 && creates[0].PayloadProto != nil {
		payloadBytes, err := protojson.Marshal(creates[0].PayloadProto)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal task run payload")
		}
		if s := string(payloadBytes); s != "" {
			payloadStr = s
		}
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
			status,
			payload
		)
		SELECT
			?,
			tasks.task_id,
			tasks.run_at,
			COALESCE((SELECT MAX(attempt) + 1 FROM task_run WHERE task_run.task_id = tasks.task_id), 0) as attempt,
			?,
			?
		FROM (
			SELECT
				unnest(CAST(? AS TEXT[])) AS task_id,
				unnest(CAST(? AS TIMESTAMPTZ[])) AS run_at
		) tasks
		WHERE NOT EXISTS (
			SELECT 1 FROM task_run
			WHERE task_run.task_id = tasks.task_id
			AND task_run.status IN (?, ?, ?, ?)
		)
		ON CONFLICT (task_id, attempt) DO NOTHING
	`, creatorPtr, storepb.TaskRun_PENDING.String(), payloadStr, taskUIDs, runAts,
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
		Space("WHERE resource_id = ?", patch.ResourceID).
		Space("RETURNING resource_id, creator, created_at, updated_at, task_id, status, result")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var taskRun TaskRunMessage
	var creatorEmail sql.NullString
	var statusString string
	var resultJSON string
	if err := txn.QueryRowContext(ctx, query, args...).Scan(
		&taskRun.ResourceID,
		&creatorEmail,
		&taskRun.CreatedAt,
		&taskRun.UpdatedAt,
		&taskRun.TaskResourceID,
		&statusString,
		&resultJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project ID not found: %s", patch.ResourceID)}
		}
		return nil, err
	}
	if creatorEmail.Valid {
		taskRun.CreatorEmail = creatorEmail.String
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
func (s *Store) BatchCancelTaskRuns(ctx context.Context, taskRunIDs []string) error {
	if len(taskRunIDs) == 0 {
		return nil
	}
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now()
		WHERE resource_id = ANY(?)
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

// FailStaleTaskRuns marks RUNNING task runs as FAILED if their replica is dead.
// A replica is considered dead if:
// 1. Its replica_id is not in the replica_heartbeat table, OR
// 2. Its last_heartbeat is older than the staleness threshold
// Returns the number of task runs marked as failed.
func (s *Store) FailStaleTaskRuns(ctx context.Context, stalenessThreshold time.Duration) (int64, error) {
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?,
		    result = '{"detail": "Task run abandoned: owning replica stopped responding"}',
		    updated_at = now()
		WHERE status = ?
		  AND replica_id IS NOT NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM replica_heartbeat rh
		    WHERE rh.replica_id = task_run.replica_id
		      AND rh.last_heartbeat >= now() - ?::INTERVAL
		  )
	`, storepb.TaskRun_FAILED.String(), storepb.TaskRun_RUNNING.String(), stalenessThreshold.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to fail stale task runs")
	}
	return result.RowsAffected()
}

// UpdateTaskRunPayload updates the payload column for a task run.
func (s *Store) UpdateTaskRunPayload(ctx context.Context, taskRunID string, payload *storepb.TaskRunPayload) error {
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal task run payload")
	}
	payloadStr := string(payloadBytes)
	if payloadStr == "" {
		payloadStr = "{}"
	}

	q := qb.Q().Space(`
		UPDATE task_run
		SET payload = ?, updated_at = now()
		WHERE resource_id = ?
	`, payloadStr, taskRunID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update task run payload")
	}
	return nil
}
