package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TaskRunMessage is message for task run.
type TaskRunMessage struct {
	TaskUID     int
	StageUID    int
	PipelineUID int
	Name        string
	Status      api.TaskRunStatus
	Code        common.Code
	Result      string
	ResultProto *storepb.TaskRunResult

	// Output only.
	ID        int
	CreatorID int
	Creator   *UserMessage
	CreatedTs int64
	UpdaterID int
	Updater   *UserMessage
	UpdatedTs int64
	ProjectID string
	StartedTs int64
}

// FindTaskRunMessage is the message for finding task runs.
type FindTaskRunMessage struct {
	UID         *int
	UIDs        *[]int
	TaskUID     *int
	StageUID    *int
	PipelineUID *int
	Status      *[]api.TaskRunStatus
}

// TaskRunFind is the API message for finding task runs.
type TaskRunFind struct {
	// Related fields
	TaskID     *int
	StageID    *int
	PipelineID *int

	// Domain specific fields
	StatusList *[]api.TaskRunStatus
}

// TaskRunStatusPatch is the API message for patching a task run.
type TaskRunStatusPatch struct {
	ID int

	// Standard fields
	UpdaterID int

	// Domain specific fields
	Status api.TaskRunStatus
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
	if v := find.StageUID; v != nil {
		where, args = append(where, fmt.Sprintf("task.stage_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineUID; v != nil {
		where, args = append(where, fmt.Sprintf("task.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Status; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("task_run.status in (%s)", strings.Join(list, ",")))
	}

	rows, err := s.db.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			task_run.id,
			task_run.creator_id,
			task_run.created_ts,
			task_run.updater_id,
			task_run.updated_ts,
			task_run.task_id,
			task_run.name,
			task_run.status,
			task_run.started_ts,
			task_run.code,
			task_run.result,
			task.pipeline_id,
			task.stage_id,
			project.resource_id
		FROM task_run
		LEFT JOIN task ON task.id = task_run.task_id
		LEFT JOIN pipeline ON pipeline.id = task.pipeline_id
		LEFT JOIN project ON project.id = pipeline.project_id
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
			&taskRun.CreatedTs,
			&taskRun.UpdaterID,
			&taskRun.UpdatedTs,
			&taskRun.TaskUID,
			&taskRun.Name,
			&taskRun.Status,
			&taskRun.StartedTs,
			&taskRun.Code,
			&taskRun.Result,
			&taskRun.PipelineUID,
			&taskRun.StageUID,
			&taskRun.ProjectID,
		); err != nil {
			return nil, err
		}

		var resultProto storepb.TaskRunResult
		decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := decoder.Unmarshal([]byte(taskRun.Result), &resultProto); err != nil {
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
		updater, err := s.GetUserByID(ctx, taskRun.UpdaterID)
		if err != nil {
			return nil, err
		}
		taskRun.Updater = updater
	}

	return taskRuns, nil
}

// UpdateTaskRunStatus updates task run status.
func (s *Store) UpdateTaskRunStatus(ctx context.Context, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	taskRun, err := s.patchTaskRunStatusImpl(ctx, tx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update task run")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return taskRun, nil
}

// CreatePendingTaskRuns creates pending task runs.
func (s *Store) CreatePendingTaskRuns(ctx context.Context, creates ...*TaskRunMessage) error {
	if len(creates) == 0 {
		return nil
	}

	sort.Slice(creates, func(i, j int) bool {
		return creates[i].TaskUID < creates[j].TaskUID
	})

	var taskIDs []int
	for _, create := range creates {
		taskIDs = append(taskIDs, create.TaskUID)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	exist, err := s.checkTaskRunsExist(ctx, tx, taskIDs, []api.TaskRunStatus{api.TaskRunPending, api.TaskRunRunning, api.TaskRunDone})
	if err != nil {
		return errors.Wrapf(err, "failed to check if task runs exist")
	}
	if exist {
		return errors.Errorf("cannot create pending task runs because there are pending/running/done task runs")
	}

	attempts, err := s.getTaskNextAttempt(ctx, tx, taskIDs)
	if err != nil {
		return errors.Wrapf(err, "failed to get task next attempt")
	}

	if err := s.createPendingTaskRunsTx(ctx, tx, attempts, creates); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit tx")
	}

	return nil
}

func (*Store) getTaskNextAttempt(ctx context.Context, tx *Tx, taskIDs []int) ([]int, error) {
	query := `
	WITH tasks AS (
		SELECT id FROM unnest(CAST($1 AS INTEGER[])) AS id
	)
	SELECT
		(SELECT COALESCE(MAX(attempt)+1, 0) FROM task_run WHERE task_run.task_id = tasks.id)
	FROM tasks ORDER BY tasks.id ASC;
	`

	rows, err := tx.QueryContext(ctx, query, taskIDs)
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

func (s *Store) createPendingTaskRunsTx(ctx context.Context, tx *Tx, attempts []int, creates []*TaskRunMessage) error {
	if len(attempts) != len(creates) {
		return errors.Errorf("length of attempts and creates are different")
	}

	// TODO(p0ny): batch create.
	for i, create := range creates {
		if err := s.createTaskRunImpl(ctx, tx, create, attempts[i], api.TaskRunPending, create.CreatorID); err != nil {
			return err
		}
	}
	return nil
}

func (*Store) checkTaskRunsExist(ctx context.Context, tx *Tx, taskIDs []int, statuses []api.TaskRunStatus) (bool, error) {
	query := `
	SELECT EXISTS (
		SELECT 1
		FROM task_run
		WHERE task_run.task_id = ANY($1) AND task_run.status = ANY($2)
	)`

	var exist bool
	if err := tx.QueryRowContext(ctx, query, taskIDs, statuses).Scan(&exist); err != nil {
		return false, errors.Wrapf(err, "failed to query if task runs exist")
	}

	return exist, nil
}

// createTaskRunImpl creates a new taskRun.
func (*Store) createTaskRunImpl(ctx context.Context, tx *Tx, create *TaskRunMessage, attempt int, status api.TaskRunStatus, creatorID int) error {
	query := `
		INSERT INTO task_run (
			creator_id,
			updater_id,
			task_id,
			attempt,
			name,
			status
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	if _, err := tx.ExecContext(ctx, query,
		creatorID,
		creatorID,
		create.TaskUID,
		attempt,
		create.Name,
		status,
	); err != nil {
		return err
	}
	return nil
}

// patchTaskRunStatusImpl updates a taskRun status. Returns the new state of the taskRun after update.
func (*Store) patchTaskRunStatusImpl(ctx context.Context, tx *Tx, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
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
	if patch.Status == api.TaskRunRunning {
		set = append(set, "started_ts = extract(epoch from now())")
	}

	// Build WHERE clause.
	where := []string{"TRUE"}
	where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, patch.ID)

	var taskRun TaskRunMessage
	if err := tx.QueryRowContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, status, code, result
	`,
		args...,
	).Scan(
		&taskRun.ID,
		&taskRun.CreatorID,
		&taskRun.CreatedTs,
		&taskRun.UpdaterID,
		&taskRun.UpdatedTs,
		&taskRun.TaskUID,
		&taskRun.Name,
		&taskRun.Status,
		&taskRun.Code,
		&taskRun.Result,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	return &taskRun, nil
}

// ListTaskRun returns a list of taskRuns.
func (s *Store) ListTaskRun(ctx context.Context, find *TaskRunFind) ([]*TaskRunMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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

func (*Store) findTaskRunImpl(ctx context.Context, tx *Tx, find *TaskRunFind) ([]*TaskRunMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.TaskID; v != nil {
		where, args = append(where, fmt.Sprintf("task_run.task_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.StageID; v != nil {
		where, args = append(where, fmt.Sprintf("task.stage_id = $%d", len(args)+1)), append(args, *v)
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

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			task_run.id,
			task_run.creator_id,
			task_run.created_ts,
			task_run.updater_id,
			task_run.updated_ts,
			task_run.task_id,
			task_run.name,
			task_run.status,
			task_run.code,
			task_run.result,
			task.pipeline_id,
			task.stage_id
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
			&taskRun.CreatedTs,
			&taskRun.UpdaterID,
			&taskRun.UpdatedTs,
			&taskRun.TaskUID,
			&taskRun.Name,
			&taskRun.Status,
			&taskRun.Code,
			&taskRun.Result,
			&taskRun.PipelineUID,
			&taskRun.StageUID,
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
func (s *Store) BatchCancelTaskRuns(ctx context.Context, taskRunIDs []int, updaterID int) error {
	query := `
		UPDATE task_run
		SET status = $1, updater_id = $2
		WHERE id = ANY($3)`
	if _, err := s.db.db.ExecContext(ctx, query, api.TaskRunCanceled, updaterID, taskRunIDs); err != nil {
		return err
	}
	return nil
}
