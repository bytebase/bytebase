package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// TaskCheckRunMessage is the store model for a TaskCheckRun.
// Fields have exactly the same meanings as TaskCheckRun.
type TaskCheckRunMessage struct {
	TaskID  int
	Type    api.TaskCheckType
	Payload string
	Code    common.Code
	Status  api.TaskCheckRunStatus
	Comment string
	Result  string

	// Output only.
	ID        int
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64
}

// TaskCheckRunFind is the API message for finding task check runs.
type TaskCheckRunFind struct {
	// Related fields
	TaskID     *int
	StageID    *int
	PipelineID *int
	Type       *api.TaskCheckType

	// Domain specific fields
	StatusList *[]api.TaskCheckRunStatus
}

// TaskCheckRunStatusPatch is the API message for patching a task check run.
type TaskCheckRunStatusPatch struct {
	ID        *int
	UpdaterID int

	Status api.TaskCheckRunStatus
	Code   common.Code
	Result string
}

func (run *TaskCheckRunMessage) toTaskCheckRun() *api.TaskCheckRun {
	return &api.TaskCheckRun{
		ID: run.ID,

		// Standard fields
		UpdaterID: run.UpdaterID,
		UpdatedTs: run.UpdatedTs,

		// Related fields
		TaskID: run.TaskID,

		// Domain specific fields
		Status:  run.Status,
		Type:    run.Type,
		Code:    run.Code,
		Comment: run.Comment,
		Result:  run.Result,
		Payload: run.Payload,
	}
}

// CreateTaskCheckRun creates task check runs in batch.
func (s *Store) CreateTaskCheckRun(ctx context.Context, creates ...*TaskCheckRunMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.createTaskCheckRunImpl(ctx, tx, creates...); err != nil {
		return err
	}

	return tx.Commit()
}

func (*Store) createTaskCheckRunImpl(ctx context.Context, tx *Tx, creates ...*TaskCheckRunMessage) error {
	if len(creates) == 0 {
		return nil
	}
	var query strings.Builder
	var values []any
	var queryValues []string
	if _, err := query.WriteString(
		`INSERT INTO task_check_run (
			creator_id,
			updater_id,
			task_id,
			status,
			type,
			payload) VALUES
      `); err != nil {
		return err
	}
	for i, create := range creates {
		if create.Payload == "" {
			create.Payload = "{}"
		}
		values = append(values,
			create.CreatorID,
			create.CreatorID,
			create.TaskID,
			api.TaskCheckRunRunning,
			create.Type,
			create.Payload,
		)
		const count = 6
		queryValues = append(queryValues, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", i*count+1, i*count+2, i*count+3, i*count+4, i*count+5, i*count+6))
	}
	if _, err := query.WriteString(strings.Join(queryValues, ",")); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, query.String(), values...); err != nil {
		return err
	}
	return nil
}

// PatchTaskCheckRunStatus updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *Store) PatchTaskCheckRunStatus(ctx context.Context, patch *TaskCheckRunStatusPatch) error {
	if patch.Result == "" {
		patch.Result = "{}"
	}
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	set, args = append(set, "code = $3"), append(args, patch.Code)
	set, args = append(set, "result = $4"), append(args, patch.Result)

	where := []string{"TRUE"}
	if v := patch.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE task_check_run
		SET %s
		WHERE %s`, strings.Join(set, ", "), strings.Join(where, " AND ")),
		args...,
	); err != nil {
		return err
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ListTaskCheckRuns list task check runs.
func (s *Store) ListTaskCheckRuns(ctx context.Context, find *TaskCheckRunFind) ([]*TaskCheckRunMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := s.findTaskCheckRunImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

func (*Store) findTaskCheckRunImpl(ctx context.Context, tx *Tx, find *TaskCheckRunFind) ([]*TaskCheckRunMessage, error) {
	joinClause := ""
	where, args := []string{"TRUE"}, []any{}
	if v := find.TaskID; v != nil {
		where, args = append(where, fmt.Sprintf("task_check_run.task_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.StageID; v != nil {
		where, args = append(where, fmt.Sprintf("task.stage_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("task.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if find.StageID != nil || find.PipelineID != nil {
		joinClause = "JOIN task ON task.id = task_check_run.task_id"
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("task_check_run.type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("task_check_run.status in (%s)", strings.Join(list, ",")))
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			task_check_run.id,
			task_check_run.creator_id,
			task_check_run.created_ts,
			task_check_run.updater_id,
			task_check_run.updated_ts,
			task_check_run.task_id,
			task_check_run.status,
			task_check_run.type,
			task_check_run.code,
			task_check_run.comment,
			task_check_run.result,
			task_check_run.payload
		FROM task_check_run
		%s
		WHERE %s`, joinClause, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taskCheckRuns []*TaskCheckRunMessage
	for rows.Next() {
		var taskCheckRun TaskCheckRunMessage
		if err := rows.Scan(
			&taskCheckRun.ID,
			&taskCheckRun.CreatorID,
			&taskCheckRun.CreatedTs,
			&taskCheckRun.UpdaterID,
			&taskCheckRun.UpdatedTs,
			&taskCheckRun.TaskID,
			&taskCheckRun.Status,
			&taskCheckRun.Type,
			&taskCheckRun.Code,
			&taskCheckRun.Comment,
			&taskCheckRun.Result,
			&taskCheckRun.Payload,
		); err != nil {
			return nil, err
		}

		taskCheckRuns = append(taskCheckRuns, &taskCheckRun)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return taskCheckRuns, nil
}
