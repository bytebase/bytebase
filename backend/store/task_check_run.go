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
