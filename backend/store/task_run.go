package store

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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

// TaskRunFind is the API message for finding task runs.
type TaskRunFind struct {
	// Related fields
	TaskID      *int
	Environment *string
	PipelineID  *int

	// Domain specific fields
	StatusList *[]storepb.TaskRun_Status
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
	where, args := []string{"TRUE"}, []any{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("task_run.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UIDs; v != nil {
		where, args = append(where, fmt.Sprintf("task_run.id = ANY($%d)", len(args)+1)), append(args, *v)
	}
	if v := find.TaskUID; v != nil {
		where, args = append(where, fmt.Sprintf("task_run.task_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Environment; v != nil {
		where, args = append(where, fmt.Sprintf("task.environment = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineUID; v != nil {
		where, args = append(where, fmt.Sprintf("task.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Status; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status.String())
		}
		where = append(where, fmt.Sprintf("task_run.status in (%s)", strings.Join(list, ",")))
	}

	rows, err := s.GetDB().QueryContext(ctx, fmt.Sprintf(`
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
		WHERE %s
		ORDER BY task_run.id ASC`, strings.Join(where, " AND ")),
		args...,
	)
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

// GetTaskRun gets a task run by uid.
func (s *Store) GetTaskRun(ctx context.Context, uid int) (*TaskRunMessage, error) {
	taskRuns, err := s.ListTaskRunsV2(ctx, &FindTaskRunMessage{UID: &uid})
	if err != nil {
		return nil, err
	}
	if len(taskRuns) == 0 {
		return nil, errors.Errorf("task run not found: %d", uid)
	}
	if len(taskRuns) > 1 {
		return nil, errors.Errorf("found multiple task runs for uid: %d", uid)
	}
	return taskRuns[0], nil
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
	var pipelineID int
	if err := tx.QueryRowContext(ctx, `
		SELECT pipeline_id FROM task WHERE id = $1
	`, taskRun.TaskUID).Scan(&pipelineID); err != nil {
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
	var pipelineID int
	if err := s.GetDB().QueryRowContext(ctx, `
		UPDATE task_run
		SET started_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING (SELECT pipeline_id FROM task WHERE task.id = task_run.task_id)
	`, taskRunID).Scan(&pipelineID); err != nil {
		return errors.Wrapf(err, "failed to update task run start at")
	}

	// Invalidate pipeline cache since UpdatedAt depends on task run updates
	s.pipelineCache.Remove(pipelineID)

	return nil
}

// CreatePendingTaskRuns creates pending task runs.
func (s *Store) CreatePendingTaskRuns(ctx context.Context, creates ...*TaskRunMessage) error {
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

	if err := s.createPendingTaskRunsTx(ctx, tx, attempts, creates); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit tx")
	}

	return nil
}

func (*Store) getTaskNextAttempt(ctx context.Context, txn *sql.Tx, taskIDs []int) ([]int, error) {
	query := `
	WITH tasks AS (
		SELECT id FROM unnest(CAST($1 AS INTEGER[])) AS id
	)
	SELECT
		(SELECT COALESCE(MAX(attempt)+1, 0) FROM task_run WHERE task_run.task_id = tasks.id)
	FROM tasks ORDER BY tasks.id ASC;
	`

	rows, err := txn.QueryContext(ctx, query, taskIDs)
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

func (s *Store) createPendingTaskRunsTx(ctx context.Context, txn *sql.Tx, attempts []int, creates []*TaskRunMessage) error {
	if len(attempts) != len(creates) {
		return errors.Errorf("length of attempts and creates are different")
	}

	// TODO(p0ny): batch create.
	for i, create := range creates {
		if err := s.createTaskRunImpl(ctx, txn, create, attempts[i], storepb.TaskRun_PENDING, create.CreatorID); err != nil {
			return err
		}
	}
	return nil
}

func (*Store) checkTaskRunsExist(ctx context.Context, txn *sql.Tx, taskIDs []int, statuses []storepb.TaskRun_Status) (bool, error) {
	query := `
	SELECT EXISTS (
		SELECT 1
		FROM task_run
		WHERE task_run.task_id = ANY($1) AND task_run.status = ANY($2)
	)`

	var exist bool
	var statusStrings []string
	for _, status := range statuses {
		statusStrings = append(statusStrings, status.String())
	}
	if err := txn.QueryRowContext(ctx, query, taskIDs, statusStrings).Scan(&exist); err != nil {
		return false, errors.Wrapf(err, "failed to query if task runs exist")
	}

	return exist, nil
}

// createTaskRunImpl creates a new taskRun.
func (*Store) createTaskRunImpl(ctx context.Context, txn *sql.Tx, create *TaskRunMessage, attempt int, status storepb.TaskRun_Status, creatorID int) error {
	query := `
		INSERT INTO task_run (
			creator_id,
			task_id,
			sheet_id,
			run_at,
			attempt,
			status
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	if _, err := txn.ExecContext(ctx, query,
		creatorID,
		create.TaskUID,
		create.SheetUID,
		create.RunAt,
		attempt,
		status.String(),
	); err != nil {
		return err
	}
	return nil
}

// patchTaskRunStatusImpl updates a taskRun status. Returns the new state of the taskRun after update.
func (*Store) patchTaskRunStatusImpl(ctx context.Context, txn *sql.Tx, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	set, args := []string{"updated_at = $1", "status = $2"}, []any{time.Now(), patch.Status.String()}
	if v := patch.Code; v != nil {
		set, args = append(set, fmt.Sprintf("code = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Result; v != nil {
		result := "{}"
		if *v != "" {
			result = *v
		}
		set, args = append(set, fmt.Sprintf("result = $%d", len(args)+1)), append(args, result)
	}

	// Build WHERE clause.
	where := []string{"TRUE"}
	where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, patch.ID)

	var taskRun TaskRunMessage
	var statusString string
	if err := txn.QueryRowContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_at, updated_at, task_id, status, code, result
	`,
		args...,
	).Scan(
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

// ListTaskRun returns a list of taskRuns.
func (s *Store) ListTaskRun(ctx context.Context, find *TaskRunFind) ([]*TaskRunMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := s.findTaskRunImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

func (*Store) findTaskRunImpl(ctx context.Context, txn *sql.Tx, find *TaskRunFind) ([]*TaskRunMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.TaskID; v != nil {
		where, args = append(where, fmt.Sprintf("task_run.task_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Environment; v != nil {
		where, args = append(where, fmt.Sprintf("task.environment = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("task.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}

	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("task_run.status in (%s)", strings.Join(list, ",")))
	}

	rows, err := txn.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			task_run.id,
			task_run.creator_id,
			task_run.created_at,
			task_run.updated_at,
			task_run.task_id,
			task_run.status,
			task_run.code,
			task_run.result,
			task.pipeline_id,
			task.environment
		FROM task_run
		JOIN task ON task.id = task_run.task_id
		WHERE %s
		ORDER BY task_run.id ASC`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taskRuns []*TaskRunMessage
	for rows.Next() {
		var taskRun TaskRunMessage
		if err := rows.Scan(
			&taskRun.ID,
			&taskRun.CreatorID,
			&taskRun.CreatedAt,
			&taskRun.UpdatedAt,
			&taskRun.TaskUID,
			&taskRun.Status,
			&taskRun.Code,
			&taskRun.Result,
			&taskRun.PipelineUID,
			&taskRun.Environment,
		); err != nil {
			return nil, err
		}

		taskRuns = append(taskRuns, &taskRun)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return taskRuns, nil
}

// BatchCancelTaskRuns updates the status of taskRuns to CANCELED.
func (s *Store) BatchCancelTaskRuns(ctx context.Context, taskRunIDs []int) error {
	if len(taskRunIDs) == 0 {
		return nil
	}

	// Get affected pipeline IDs for cache invalidation
	rows, err := s.GetDB().QueryContext(ctx, `
		SELECT DISTINCT task.pipeline_id
		FROM task_run
		JOIN task ON task.id = task_run.task_id
		WHERE task_run.id = ANY($1)
	`, taskRunIDs)
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

	query := `
		UPDATE task_run
		SET status = $1, updated_at = now()
		WHERE id = ANY($2)`
	if _, err := s.GetDB().ExecContext(ctx, query, storepb.TaskRun_CANCELED.String(), taskRunIDs); err != nil {
		return err
	}

	// Invalidate pipeline caches since UpdatedAt depends on task run updates
	for _, pipelineID := range pipelineIDs {
		s.pipelineCache.Remove(pipelineID)
	}

	return nil
}
