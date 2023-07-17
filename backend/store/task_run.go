package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// TaskRunMessage is message for task run.
type TaskRunMessage struct {
	TaskUID     int
	StageUID    int
	PipelineUID int
	Name        string
	Status      api.TaskRunStatus
	Type        api.TaskType
	Code        common.Code
	Comment     string
	Result      string
	Payload     string

	// Output only.
	ID        int
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64
}

// TaskRunCreate is the API message for creating a task run.
type TaskRunCreate struct {
	CreatorID int
	TaskID    int
	Name      string
	Type      api.TaskType
	Payload   string
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
	ID *int

	// Standard fields
	UpdaterID int

	// Related fields
	TaskID *int

	// Domain specific fields
	Status api.TaskRunStatus
	Code   *common.Code
	// Records the status detail (e.g. error message on failure)
	Comment *string
	Result  *string
}

func (taskRun *TaskRunMessage) toTaskRun() *api.TaskRun {
	return &api.TaskRun{
		ID:        taskRun.ID,
		CreatorID: taskRun.CreatorID,
		CreatedTs: taskRun.CreatedTs,
		UpdaterID: taskRun.UpdaterID,
		UpdatedTs: taskRun.UpdatedTs,
		TaskID:    taskRun.TaskUID,
		Name:      taskRun.Name,
		Status:    taskRun.Status,
		Type:      taskRun.Type,
		Code:      taskRun.Code,
		Comment:   taskRun.Comment,
		Result:    taskRun.Result,
		Payload:   taskRun.Payload,
	}
}

// createTaskRunImpl creates a new taskRun.
func (*Store) createTaskRunImpl(ctx context.Context, tx *Tx, create *TaskRunMessage, creatorID int) error {
	if create.Payload == "" {
		create.Payload = "{}"
	}
	query := `
		INSERT INTO task_run (
			creator_id,
			updater_id,
			task_id,
			name,
			status,
			type,
			payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if _, err := tx.ExecContext(ctx, query,
		creatorID,
		creatorID,
		create.TaskUID,
		create.Name,
		api.TaskRunRunning,
		create.Type,
		"{}", /* payload */
	); err != nil {
		return err
	}
	return nil
}

func (s *Store) getTaskRunTx(ctx context.Context, tx *Tx, find *TaskRunFind) (*TaskRunMessage, error) {
	taskRuns, err := s.findTaskRunImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(taskRuns) == 0 {
		return nil, nil
	} else if len(taskRuns) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d task runs with filter %+v, expect 1", len(taskRuns), find)}
	}
	return taskRuns[0], nil
}

// patchTaskRunStatusImpl updates a taskRun status. Returns the new state of the taskRun after update.
func (*Store) patchTaskRunStatusImpl(ctx context.Context, tx *Tx, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	if v := patch.Code; v != nil {
		set, args = append(set, fmt.Sprintf("code = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("comment = $%d", len(args)+1)), append(args, *v)
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
	if v := patch.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.TaskID; v != nil {
		where, args = append(where, fmt.Sprintf("task_id = $%d", len(args)+1)), append(args, *v)
	}

	var taskRun TaskRunMessage
	if err := tx.QueryRowContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, status, type, code, comment, result, payload
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
		&taskRun.Type,
		&taskRun.Code,
		&taskRun.Comment,
		&taskRun.Result,
		&taskRun.Payload,
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
			task_run.type,
			task_run.code,
			task_run.comment,
			task_run.result,
			task_run.payload,
			task.pipeline_id,
			task.stage_id
		FROM task_run
		JOIN task ON task.id = task_run.task_id
		WHERE %s`, strings.Join(where, " AND ")),
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
			&taskRun.Type,
			&taskRun.Code,
			&taskRun.Comment,
			&taskRun.Result,
			&taskRun.Payload,
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

// BatchPatchTaskRunStatus updates the status of a list of taskRuns.
func (s *Store) BatchPatchTaskRunStatus(ctx context.Context, taskRunIDs []int, status api.TaskRunStatus, updaterID int) error {
	var ids []string
	for _, id := range taskRunIDs {
		ids = append(ids, fmt.Sprintf("%d", id))
	}
	query := fmt.Sprintf(`
		UPDATE task_run
		SET status = $1, updater_id = $2
		WHERE id IN (%s);
	`, strings.Join(ids, ","))
	if _, err := s.db.db.ExecContext(ctx, query, status, updaterID); err != nil {
		return err
	}
	return nil
}
