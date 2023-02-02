package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// TaskCheckRunMessage is the store model for a TaskCheckRun.
// Fields have exactly the same meanings as TaskCheckRun.
type TaskCheckRunMessage struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	TaskID int

	// Domain specific fields
	Status  api.TaskCheckRunStatus
	Type    api.TaskCheckType
	Code    common.Code
	Comment string
	Result  string
	Payload string
}

func (run *TaskCheckRunMessage) toTaskCheckRun() *api.TaskCheckRun {
	return &api.TaskCheckRun{
		ID: run.ID,

		// Standard fields
		CreatorID: run.CreatorID,
		CreatedTs: run.CreatedTs,
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

// FindTaskCheckRun finds a list of TaskCheckRun instances.
func (s *Store) FindTaskCheckRun(ctx context.Context, find *TaskCheckRunFind) ([]*api.TaskCheckRun, error) {
	taskCheckRuns, err := s.ListTaskCheckRuns(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find TaskCheckRun list with TaskCheckRunFind[%+v]", find)
	}
	var composedTaskCheckRuns []*api.TaskCheckRun
	for _, run := range taskCheckRuns {
		composedTaskCheckRun, err := s.composeTaskCheckRun(ctx, run)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose task check run %#v", run)
		}
		composedTaskCheckRuns = append(composedTaskCheckRuns, composedTaskCheckRun)
	}
	return composedTaskCheckRuns, nil
}

func (s *Store) composeTaskCheckRun(ctx context.Context, run *TaskCheckRunMessage) (*api.TaskCheckRun, error) {
	composedTaskCheckRun := run.toTaskCheckRun()

	creator, err := s.GetPrincipalByID(ctx, composedTaskCheckRun.CreatorID)
	if err != nil {
		return nil, err
	}
	composedTaskCheckRun.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, composedTaskCheckRun.UpdaterID)
	if err != nil {
		return nil, err
	}
	composedTaskCheckRun.Updater = updater

	return composedTaskCheckRun, nil
}

// TaskCheckRunCreate is the API message for creating a task check run.
type TaskCheckRunCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	TaskID int

	// Domain specific fields
	Type    api.TaskCheckType
	Comment string
	Payload string
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

// CreateTaskCheckRunIfNeeded creates a new taskCheckRun.
func (s *Store) CreateTaskCheckRunIfNeeded(ctx context.Context, create *TaskCheckRunCreate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
	taskCheckRunFind := &TaskCheckRunFind{
		TaskID:     &create.TaskID,
		Type:       &create.Type,
		StatusList: &statusList,
	}

	taskCheckRunList, err := s.findTaskCheckRunImpl(ctx, tx, taskCheckRunFind)
	if err != nil {
		return err
	}

	if runningCount := len(taskCheckRunList); runningCount > 0 {
		if runningCount > 1 {
			// Normally, this should not happen, if it occurs, emit a warning
			log.Warn(fmt.Sprintf("Found %d task check run, expect at most 1", len(taskCheckRunList)),
				zap.Int("task_id", create.TaskID),
				zap.String("task_check_type", string(create.Type)),
			)
		}
		return nil
	}

	list, err := s.createTaskCheckRunImpl(ctx, tx, create)
	if err != nil {
		return err
	}

	if len(list) != 1 {
		return &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d task check runs, expect 1", len(list))}
	}

	return tx.Commit()
}

// BatchCreateTaskCheckRun creates task check runs in batch.
func (s *Store) BatchCreateTaskCheckRun(ctx context.Context, creates []*TaskCheckRunCreate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	if _, err := s.createTaskCheckRunImpl(ctx, tx, creates...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

func (*Store) createTaskCheckRunImpl(ctx context.Context, tx *Tx, creates ...*TaskCheckRunCreate) ([]*TaskCheckRunMessage, error) {
	var query strings.Builder
	var values []interface{}
	var queryValues []string
	if _, err := query.WriteString(
		`INSERT INTO task_check_run (
			creator_id,
			updater_id,
			task_id,
			status,
			type,
			comment,
			payload) VALUES
      `); err != nil {
		return nil, err
	}
	for i, create := range creates {
		if create.Payload == "" {
			create.Payload = "{}"
		}
		values = append(values,
			create.CreatorID,
			create.CreatorID,
			create.TaskID,
			create.Type,
			create.Comment,
			create.Payload,
		)
		const count = 6
		queryValues = append(queryValues, fmt.Sprintf("($%d, $%d, $%d, 'RUNNING', $%d, $%d, $%d)", i*count+1, i*count+2, i*count+3, i*count+4, i*count+5, i*count+6))
	}
	if _, err := query.WriteString(strings.Join(queryValues, ",")); err != nil {
		return nil, err
	}
	if _, err := query.WriteString(` RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, status, type, code, comment, result, payload`); err != nil {
		return nil, err
	}

	var taskCheckRuns []*TaskCheckRunMessage
	rows, err := tx.QueryContext(ctx, query.String(), values...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var run TaskCheckRunMessage
		if err := rows.Scan(
			&run.ID,
			&run.CreatorID,
			&run.CreatedTs,
			&run.UpdaterID,
			&run.UpdatedTs,
			&run.TaskID,
			&run.Status,
			&run.Type,
			&run.Code,
			&run.Comment,
			&run.Result,
			&run.Payload,
		); err != nil {
			return nil, FormatError(err)
		}
		taskCheckRuns = append(taskCheckRuns, &run)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return taskCheckRuns, nil
}

// PatchTaskCheckRunStatus updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *Store) PatchTaskCheckRunStatus(ctx context.Context, patch *TaskCheckRunStatusPatch) error {
	if patch.Result == "" {
		patch.Result = "{}"
	}
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
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
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTaskCheckRunImpl(ctx, tx, find)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

func (*Store) findTaskCheckRunImpl(ctx context.Context, tx *Tx, find *TaskCheckRunFind) ([]*TaskCheckRunMessage, error) {
	joinClause := ""
	where, args := []string{"TRUE"}, []interface{}{}
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
		where = append(where, fmt.Sprintf("status in (%s)", strings.Join(list, ",")))
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
		return nil, FormatError(err)
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
			return nil, FormatError(err)
		}

		taskCheckRuns = append(taskCheckRuns, &taskCheckRun)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRuns, nil
}
