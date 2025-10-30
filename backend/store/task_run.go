package store

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TaskRunMessage is message for task run.
type TaskRunMessage struct {
	TaskUID     int
	Environment string // Refer to the task's environment.
	PipelineUID int
	Status      storepb.TaskRun_Status
	Code        common.Code
	Result      string
	ResultProto *storepb.TaskRunResult
	SheetUID    *int

	// Output only.
	ID        int
	CreatorID int
	Creator   *UserMessage
	CreatedAt time.Time
	UpdatedAt time.Time
	ProjectID string
	StartedAt *time.Time
	RunAt     *time.Time
}

// FindTaskRunMessage is the message for finding task runs.
type FindTaskRunMessage struct {
	UID         *int
	UIDs        *[]int
	TaskUID     *int
	Environment *string
	PipelineUID *int
	Status      *[]storepb.TaskRun_Status
}

// TaskRunStatusPatch is the API message for patching a task run.
type TaskRunStatusPatch struct {
	ID int

	// Standard fields
	UpdaterID int

	// Domain specific fields
	Status storepb.TaskRun_Status
	Code   *common.Code
	Result *string
}

// ListTaskRunsV2 lists task runs.
func (s *Store) ListTaskRunsV2(ctx context.Context, find *FindTaskRunMessage) ([]*TaskRunMessage, error) {
	q := qb.Q().Space(`
		SELECT
			task_run.id,
			task_run.creator_id,
			task_run.created_at,
			task_run.updated_at,
			task_run.task_id,
			task_run.status,
			task_run.started_at,
			task_run.run_at,
			task_run.code,
			task_run.result,
			task_run.sheet_id,
			task.pipeline_id,
			task.environment,
			project.resource_id
		FROM task_run
		LEFT JOIN task ON task.id = task_run.task_id
		LEFT JOIN pipeline ON pipeline.id = task.pipeline_id
		LEFT JOIN project ON project.resource_id = pipeline.project
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
	if v := find.PipelineUID; v != nil {
		q.And("task.pipeline_id = ?", *v)
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
		if err := rows.Scan(
			&taskRun.ID,
			&taskRun.CreatorID,
			&taskRun.CreatedAt,
			&taskRun.UpdatedAt,
			&taskRun.TaskUID,
			&statusString,
			&startedAt,
			&runAt,
			&taskRun.Code,
			&taskRun.Result,
			&taskRun.SheetUID,
			&taskRun.PipelineUID,
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
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(taskRun.Result), &resultProto); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal task run result: %s", taskRun.Result)
		}
		taskRun.ResultProto = &resultProto

		taskRuns = append(taskRuns, &taskRun)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, taskRun := range taskRuns {
		creator, err := s.GetUserByID(ctx, taskRun.CreatorID)
		if err != nil {
			return nil, err
		}
		taskRun.Creator = creator
	}

	return taskRuns, nil
}

// GetTaskRunV1 gets a task run.
func (s *Store) GetTaskRunV1(ctx context.Context, find *FindTaskRunMessage) (*TaskRunMessage, error) {
	taskRuns, err := s.ListTaskRunsV2(ctx, find)
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

	// Get the pipeline ID for cache invalidation
	q := qb.Q().Space("SELECT pipeline_id FROM task WHERE id = ?", taskRun.TaskUID)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var pipelineID int
	if err := tx.QueryRowContext(ctx, sql, args...).Scan(&pipelineID); err != nil {
		return nil, errors.Wrapf(err, "failed to get pipeline ID for task %d", taskRun.TaskUID)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	// Invalidate pipeline cache since UpdatedAt depends on task run updates
	s.pipelineCache.Remove(pipelineID)

	return taskRun, nil
}

func (s *Store) UpdateTaskRunStartAt(ctx context.Context, taskRunID int) error {
	// Get the pipeline ID for cache invalidation
	q := qb.Q().Space(`
		UPDATE task_run
		SET started_at = now(), updated_at = now()
		WHERE id = ?
		RETURNING (SELECT pipeline_id FROM task WHERE task.id = task_run.task_id)
	`, taskRunID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	var pipelineID int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&pipelineID); err != nil {
		return errors.Wrapf(err, "failed to update task run start at")
	}

	// Invalidate pipeline cache since UpdatedAt depends on task run updates
	s.pipelineCache.Remove(pipelineID)

	return nil
}

// CreatePendingTaskRuns creates pending task runs.
func (s *Store) CreatePendingTaskRuns(ctx context.Context, creatorID int, creates ...*TaskRunMessage) error {
	if len(creates) == 0 {
		return nil
	}

	slices.SortFunc(creates, func(a, b *TaskRunMessage) int {
		return a.TaskUID - b.TaskUID
	})

	var taskIDs []int
	for _, create := range creates {
		taskIDs = append(taskIDs, create.TaskUID)
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	attempts, err := s.getTaskNextAttempt(ctx, tx, taskIDs)
	if err != nil {
		return errors.Wrapf(err, "failed to get task next attempt")
	}

	exist, err := s.checkTaskRunsExist(ctx, tx, taskIDs, []storepb.TaskRun_Status{storepb.TaskRun_PENDING, storepb.TaskRun_RUNNING, storepb.TaskRun_DONE})
	if err != nil {
		return errors.Wrapf(err, "failed to check if task runs exist")
	}
	if exist {
		return errors.Errorf("cannot create pending task runs because there are pending/running/done task runs")
	}

	if err := s.createPendingTaskRunsTx(ctx, tx, creatorID, attempts, creates); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit tx")
	}

	return nil
}

func (*Store) getTaskNextAttempt(ctx context.Context, txn *sql.Tx, taskIDs []int) ([]int, error) {
	q := qb.Q().Space(`
		WITH tasks AS (
			SELECT id FROM unnest(CAST(? AS INTEGER[])) AS id
		)
		SELECT
			(SELECT COALESCE(MAX(attempt)+1, 0) FROM task_run WHERE task_run.task_id = tasks.id)
		FROM tasks ORDER BY tasks.id ASC
	`, taskIDs)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := txn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	var attempts []int
	for rows.Next() {
		var attempt int
		if err := rows.Scan(&attempt); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to scan rows")
	}

	return attempts, nil
}

func (*Store) createPendingTaskRunsTx(ctx context.Context, txn *sql.Tx, creatorID int, attempts []int, creates []*TaskRunMessage) error {
	if len(attempts) != len(creates) {
		return errors.Errorf("length of attempts and creates are different")
	}
	var taskUIDs []int
	var sheetUIDs []*int
	var runAts []*time.Time
	for _, create := range creates {
		taskUIDs = append(taskUIDs, create.TaskUID)
		sheetUIDs = append(sheetUIDs, create.SheetUID)
		runAts = append(runAts, create.RunAt)
	}

	q := qb.Q().Space(`
		INSERT INTO task_run (
			creator_id,
			task_id,
			sheet_id,
			run_at,
			attempt,
			status
		) SELECT
			?,
			unnest(CAST(? AS INTEGER[])),
			unnest(CAST(? AS INTEGER[])),
			unnest(CAST(? AS TIMESTAMPTZ[])),
			unnest(CAST(? AS INTEGER[])),
			?
	`, creatorID, taskUIDs, sheetUIDs, runAts, attempts, storepb.TaskRun_PENDING.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := txn.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}
	return nil
}

func (*Store) checkTaskRunsExist(ctx context.Context, txn *sql.Tx, taskIDs []int, statuses []storepb.TaskRun_Status) (bool, error) {
	var statusStrings []string
	for _, status := range statuses {
		statusStrings = append(statusStrings, status.String())
	}

	q := qb.Q().Space(`
		SELECT EXISTS (
			SELECT 1
			FROM task_run
			WHERE task_run.task_id = ANY(?) AND task_run.status = ANY(?)
		)
	`, taskIDs, statusStrings)

	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}

	var exist bool
	if err := txn.QueryRowContext(ctx, query, args...).Scan(&exist); err != nil {
		return false, errors.Wrapf(err, "failed to query if task runs exist")
	}

	return exist, nil
}

// patchTaskRunStatusImpl updates a taskRun status. Returns the new state of the taskRun after update.
func (*Store) patchTaskRunStatusImpl(ctx context.Context, txn *sql.Tx, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	set := qb.Q()

	set.Comma("updated_at = ?, status = ?", time.Now(), patch.Status.String())

	if v := patch.Code; v != nil {
		set.Comma("code = ?", *v)
	}
	if v := patch.Result; v != nil {
		result := "{}"
		if *v != "" {
			result = *v
		}
		set.Comma("result = ?", result)
	}

	q := qb.Q().Space("UPDATE task_run SET ?", set).
		Space("WHERE id = ?", patch.ID).
		Space("RETURNING id, creator_id, created_at, updated_at, task_id, status, code, result")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var taskRun TaskRunMessage
	var statusString string
	if err := txn.QueryRowContext(ctx, query, args...).Scan(
		&taskRun.ID,
		&taskRun.CreatorID,
		&taskRun.CreatedAt,
		&taskRun.UpdatedAt,
		&taskRun.TaskUID,
		&statusString,
		&taskRun.Code,
		&taskRun.Result,
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
	return &taskRun, nil
}

// BatchCancelTaskRuns updates the status of taskRuns to CANCELED.
func (s *Store) BatchCancelTaskRuns(ctx context.Context, taskRunIDs []int) error {
	if len(taskRunIDs) == 0 {
		return nil
	}

	// Get affected pipeline IDs for cache invalidation
	q := qb.Q().Space(`
		SELECT DISTINCT task.pipeline_id
		FROM task_run
		JOIN task ON task.id = task_run.task_id
		WHERE task_run.id = ANY(?)
	`, taskRunIDs)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to get pipeline IDs")
	}
	defer rows.Close()

	var pipelineIDs []int
	for rows.Next() {
		var pipelineID int
		if err := rows.Scan(&pipelineID); err != nil {
			return errors.Wrapf(err, "failed to scan pipeline ID")
		}
		pipelineIDs = append(pipelineIDs, pipelineID)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to iterate pipeline IDs")
	}

	q2 := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now()
		WHERE id = ANY(?)
	`, storepb.TaskRun_CANCELED.String(), taskRunIDs)

	query2, args2, err := q2.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query2, args2...); err != nil {
		return err
	}

	// Invalidate pipeline caches since UpdatedAt depends on task run updates
	for _, pipelineID := range pipelineIDs {
		s.pipelineCache.Remove(pipelineID)
	}

	return nil
}
