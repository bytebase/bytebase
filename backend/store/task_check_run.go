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

// taskCheckRunRaw is the store model for a TaskCheckRun.
// Fields have exactly the same meanings as TaskCheckRun.
type taskCheckRunRaw struct {
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

// toTaskCheckRun creates an instance of TaskCheckRun based on the taskCheckRunRaw.
// This is intended to be called when we need to compose a TaskCheckRun relationship.
func (raw *taskCheckRunRaw) toTaskCheckRun() *api.TaskCheckRun {
	return &api.TaskCheckRun{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		TaskID: raw.TaskID,

		// Domain specific fields
		Status:  raw.Status,
		Type:    raw.Type,
		Code:    raw.Code,
		Comment: raw.Comment,
		Result:  raw.Result,
		Payload: raw.Payload,
	}
}

// CreateTaskCheckRunIfNeeded creates an instance of TaskCheckRun if needed.
func (s *Store) CreateTaskCheckRunIfNeeded(ctx context.Context, create *TaskCheckRunCreate) (*api.TaskCheckRun, error) {
	taskCheckRunRaw, err := s.createTaskCheckRunRawIfNeeded(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create TaskCheckRun with TaskCheckRunCreate[%+v]", create)
	}
	taskCheckRun, err := s.composeTaskCheckRun(ctx, taskCheckRunRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose TaskCheckRun with taskCheckRunRaw[%+v]", taskCheckRunRaw)
	}
	return taskCheckRun, nil
}

// BatchCreateTaskCheckRun inserts many TaskCheckRun instances, which is too slow otherwise to insert one by one.
func (s *Store) BatchCreateTaskCheckRun(ctx context.Context, creates []*TaskCheckRunCreate) ([]*api.TaskCheckRun, error) {
	if len(creates) == 0 {
		return nil, nil
	}
	taskCheckRunRawList, err := s.batchCreateTaskCheckRunRaw(ctx, creates)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create TaskCheckRun with TaskCheckRunCreates[%+v]", creates)
	}
	var taskCheckRunList []*api.TaskCheckRun
	for _, taskCheckRunRaw := range taskCheckRunRawList {
		taskCheckRun, err := s.composeTaskCheckRun(ctx, taskCheckRunRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose TaskCheckRun with taskCheckRunRaw[%+v]", taskCheckRunRaw)
		}
		taskCheckRunList = append(taskCheckRunList, taskCheckRun)
	}
	return taskCheckRunList, nil
}

// FindTaskCheckRun finds a list of TaskCheckRun instances.
func (s *Store) FindTaskCheckRun(ctx context.Context, find *TaskCheckRunFind) ([]*api.TaskCheckRun, error) {
	taskCheckRunRawList, err := s.findTaskCheckRunRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find TaskCheckRun list with TaskCheckRunFind[%+v]", find)
	}
	var taskCheckRunList []*api.TaskCheckRun
	for _, raw := range taskCheckRunRawList {
		taskCheckRun, err := s.composeTaskCheckRun(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose TaskCheckRun with taskCheckRunRaw[%+v]", raw)
		}
		taskCheckRunList = append(taskCheckRunList, taskCheckRun)
	}
	return taskCheckRunList, nil
}

// PatchTaskCheckRunStatus patches an instance of TaskCheckRunStatus.
func (s *Store) PatchTaskCheckRunStatus(ctx context.Context, patch *TaskCheckRunStatusPatch) (*api.TaskCheckRun, error) {
	taskCheckRunRaw, err := s.patchTaskCheckRunRawStatus(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch TaskCheckRunStatus with TaskCheckRunStatusPatch[%+v]", patch)
	}
	taskCheckRun, err := s.composeTaskCheckRun(ctx, taskCheckRunRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose TaskCheckRunStatus with taskCheckRunRaw[%+v]", taskCheckRunRaw)
	}
	return taskCheckRun, nil
}

//
// private functions
//

// composeTaskCheckRun composes an instance of TaskCheckRun by taskCheckRunRaw.
func (s *Store) composeTaskCheckRun(ctx context.Context, raw *taskCheckRunRaw) (*api.TaskCheckRun, error) {
	taskCheckRun := raw.toTaskCheckRun()

	creator, err := s.GetPrincipalByID(ctx, taskCheckRun.CreatorID)
	if err != nil {
		return nil, err
	}
	taskCheckRun.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, taskCheckRun.UpdaterID)
	if err != nil {
		return nil, err
	}
	taskCheckRun.Updater = updater

	return taskCheckRun, nil
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
	TaskID *int
	Type   *api.TaskCheckType

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

// createTaskCheckRunRawIfNeeded creates a new taskCheckRun. See interface for the expected behavior.
func (s *Store) createTaskCheckRunRawIfNeeded(ctx context.Context, create *TaskCheckRunCreate) (*taskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
	taskCheckRunFind := &TaskCheckRunFind{
		TaskID:     &create.TaskID,
		Type:       &create.Type,
		StatusList: &statusList,
	}

	taskCheckRunList, err := s.findTaskCheckRunRawTx(ctx, tx, taskCheckRunFind)
	if err != nil {
		return nil, err
	}

	if runningCount := len(taskCheckRunList); runningCount > 0 {
		if runningCount > 1 {
			// Normally, this should not happen, if it occurs, emit a warning
			log.Warn(fmt.Sprintf("Found %d task check run, expect at most 1", len(taskCheckRunList)),
				zap.Int("task_id", create.TaskID),
				zap.String("task_check_type", string(create.Type)),
			)
		}
		return taskCheckRunList[0], nil
	}

	list, err := s.createTaskCheckRunImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if len(list) != 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d task check runs, expect 1", len(list))}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return list[0], nil
}

func (s *Store) batchCreateTaskCheckRunRaw(ctx context.Context, creates []*TaskCheckRunCreate) ([]*taskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	taskCheckRunList, err := s.createTaskCheckRunImpl(ctx, tx, creates...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRunList, nil
}

func (*Store) createTaskCheckRunImpl(ctx context.Context, tx *Tx, creates ...*TaskCheckRunCreate) ([]*taskCheckRunRaw, error) {
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

	var taskCheckRunRawList []*taskCheckRunRaw
	rows, err := tx.QueryContext(ctx, query.String(), values...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var taskCheckRunRaw taskCheckRunRaw
		if err := rows.Scan(
			&taskCheckRunRaw.ID,
			&taskCheckRunRaw.CreatorID,
			&taskCheckRunRaw.CreatedTs,
			&taskCheckRunRaw.UpdaterID,
			&taskCheckRunRaw.UpdatedTs,
			&taskCheckRunRaw.TaskID,
			&taskCheckRunRaw.Status,
			&taskCheckRunRaw.Type,
			&taskCheckRunRaw.Code,
			&taskCheckRunRaw.Comment,
			&taskCheckRunRaw.Result,
			&taskCheckRunRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}
		taskCheckRunRawList = append(taskCheckRunRawList, &taskCheckRunRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return taskCheckRunRawList, nil
}

// findTaskCheckRunRaw retrieves a list of taskCheckRuns based on find.
func (s *Store) findTaskCheckRunRaw(ctx context.Context, find *TaskCheckRunFind) ([]*taskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTaskCheckRunImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// findTaskCheckRunRawTx retrieves a list of taskCheckRuns based on find.
func (s *Store) findTaskCheckRunRawTx(ctx context.Context, tx *Tx, find *TaskCheckRunFind) ([]*taskCheckRunRaw, error) {
	list, err := s.findTaskCheckRunImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// patchTaskCheckRunRawStatus updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *Store) patchTaskCheckRunRawStatus(ctx context.Context, patch *TaskCheckRunStatusPatch) (*taskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	taskCheckRun, err := s.patchTaskCheckRunStatusImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRun, nil
}

// patchTaskCheckRunStatusImpl updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (*Store) patchTaskCheckRunStatusImpl(ctx context.Context, tx *Tx, patch *TaskCheckRunStatusPatch) (*taskCheckRunRaw, error) {
	// Build UPDATE clause.
	if patch.Result == "" {
		patch.Result = "{}"
	}
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	set, args = append(set, "code = $3"), append(args, patch.Code)
	set, args = append(set, "result = $4"), append(args, patch.Result)

	// Build WHERE clause.
	where := []string{"TRUE"}
	if v := patch.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

	var taskCheckRunRaw taskCheckRunRaw
	if err := tx.QueryRowContext(ctx, `
		UPDATE task_check_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, status, type, code, comment, result, payload
	`,
		args...,
	).Scan(
		&taskCheckRunRaw.ID,
		&taskCheckRunRaw.CreatorID,
		&taskCheckRunRaw.CreatedTs,
		&taskCheckRunRaw.UpdaterID,
		&taskCheckRunRaw.UpdatedTs,
		&taskCheckRunRaw.TaskID,
		&taskCheckRunRaw.Status,
		&taskCheckRunRaw.Type,
		&taskCheckRunRaw.Code,
		&taskCheckRunRaw.Comment,
		&taskCheckRunRaw.Result,
		&taskCheckRunRaw.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("task check run ID not found: %d", *patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &taskCheckRunRaw, nil
}

func (s *Store) listTaskCheckRun(ctx context.Context, find *TaskCheckRunFind) ([]*taskCheckRunRaw, error) {
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

func (*Store) findTaskCheckRunImpl(ctx context.Context, tx *Tx, find *TaskCheckRunFind) ([]*taskCheckRunRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.TaskID; v != nil {
		where, args = append(where, fmt.Sprintf("task_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("status in (%s)", strings.Join(list, ",")))
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			task_id,
			status,
			type,
			code,
			comment,
			result,
			payload
		FROM task_check_run
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into taskCheckRunRawList.
	var taskCheckRunRawList []*taskCheckRunRaw
	for rows.Next() {
		var taskCheckRun taskCheckRunRaw
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

		taskCheckRunRawList = append(taskCheckRunRawList, &taskCheckRun)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRunRawList, nil
}
