package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"go.uber.org/zap"
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

// CreateTaskCheckRunIfNeeded creates an instance of TaskCheckRun if needed
func (s *Store) CreateTaskCheckRunIfNeeded(ctx context.Context, create *api.TaskCheckRunCreate) (*api.TaskCheckRun, error) {
	taskCheckRunRaw, err := s.createTaskCheckRunRawIfNeeded(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create TaskCheckRun with TaskCheckRunCreate[%+v], error[%w]", create, err)
	}
	taskCheckRun, err := s.composeTaskCheckRun(ctx, taskCheckRunRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose TaskCheckRun with taskCheckRunRaw[%+v], error[%w]", taskCheckRunRaw, err)
	}
	return taskCheckRun, nil
}

// FindTaskCheckRun finds a list of TaskCheckRun instances
func (s *Store) FindTaskCheckRun(ctx context.Context, find *api.TaskCheckRunFind) ([]*api.TaskCheckRun, error) {
	taskCheckRunRawList, err := s.findTaskCheckRunRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find TaskCheckRun list with TaskCheckRunFind[%+v], error[%w]", find, err)
	}
	var taskCheckRunList []*api.TaskCheckRun
	for _, raw := range taskCheckRunRawList {
		taskCheckRun, err := s.composeTaskCheckRun(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose TaskCheckRun with taskCheckRunRaw[%+v], error[%w]", raw, err)
		}
		taskCheckRunList = append(taskCheckRunList, taskCheckRun)
	}
	return taskCheckRunList, nil
}

// PatchTaskCheckRunStatus patches an instance of TaskCheckRunStatus
func (s *Store) PatchTaskCheckRunStatus(ctx context.Context, patch *api.TaskCheckRunStatusPatch) (*api.TaskCheckRun, error) {
	taskCheckRunRaw, err := s.patchTaskCheckRunRawStatus(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch TaskCheckRunStatus with TaskCheckRunStatusPatch[%+v], error[%w]", patch, err)
	}
	taskCheckRun, err := s.composeTaskCheckRun(ctx, taskCheckRunRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose TaskCheckRunStatus with taskCheckRunRaw[%+v], error[%w]", taskCheckRunRaw, err)
	}
	return taskCheckRun, nil
}

//
// private functions
//

// composeTaskCheckRun composes an instance of TaskCheckRun by taskCheckRunRaw
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

// createTaskCheckRunRawIfNeeded creates a new taskCheckRun. See interface for the expected behavior
func (s *Store) createTaskCheckRunRawIfNeeded(ctx context.Context, create *api.TaskCheckRunCreate) (*taskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
	if create.SkipIfAlreadyTerminated {
		statusList = append(statusList, api.TaskCheckRunDone)
		statusList = append(statusList, api.TaskCheckRunFailed)
		statusList = append(statusList, api.TaskCheckRunCanceled)
	}
	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID:     &create.TaskID,
		Type:       &create.Type,
		StatusList: &statusList,
	}

	taskCheckRunList, err := s.findTaskCheckRunRawTx(ctx, tx.PTx, taskCheckRunFind)
	if err != nil {
		return nil, err
	}

	runningCount := 0
	if create.SkipIfAlreadyTerminated {
		for _, taskCheckRun := range taskCheckRunList {
			switch taskCheckRun.Status {
			case api.TaskCheckRunDone, api.TaskCheckRunFailed, api.TaskCheckRunCanceled:
				return taskCheckRun, nil
			case api.TaskCheckRunRunning:
				runningCount++
			}
		}
	} else {
		runningCount = len(taskCheckRunList)
	}

	if runningCount > 0 {
		if runningCount > 1 {
			// Normally, this should not happen, if it occurs, emit a warning
			log.Warn(fmt.Sprintf("Found %d task check run, expect at most 1", len(taskCheckRunList)),
				zap.Int("task_id", create.TaskID),
				zap.String("task_check_type", string(create.Type)),
			)
		}
		return taskCheckRunList[0], nil
	}

	taskCheckRun, err := s.createTaskCheckRunImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRun, nil
}

// createTaskCheckRunImpl creates a new taskCheckRun.
func (s *Store) createTaskCheckRunImpl(ctx context.Context, tx *sql.Tx, create *api.TaskCheckRunCreate) (*taskCheckRunRaw, error) {
	if create.Payload == "" {
		create.Payload = "{}"
	}
	rows, err := tx.QueryContext(ctx, `
		INSERT INTO task_check_run (
			creator_id,
			updater_id,
			task_id,
			status,
			type,
			comment,
			payload
		)
		VALUES ($1, $2, $3, 'RUNNING', $4, $5, $6)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, status, type, code, comment, result, payload
	`,
		create.CreatorID,
		create.CreatorID,
		create.TaskID,
		create.Type,
		create.Comment,
		create.Payload,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var tRaw *taskCheckRunRaw
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
		tRaw = &taskCheckRunRaw
	}
	if tRaw != nil {
		return tRaw, nil
	}
	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task check run cannot be created for task ID %v", create.TaskID)}
}

// findTaskCheckRunRaw retrieves a list of taskCheckRuns based on find.
func (s *Store) findTaskCheckRunRaw(ctx context.Context, find *api.TaskCheckRunFind) ([]*taskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findTaskCheckRunImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// findTaskCheckRunRawTx retrieves a list of taskCheckRuns based on find.
func (s *Store) findTaskCheckRunRawTx(ctx context.Context, tx *sql.Tx, find *api.TaskCheckRunFind) ([]*taskCheckRunRaw, error) {
	list, err := s.findTaskCheckRunImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// patchTaskCheckRunRawStatus updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *Store) patchTaskCheckRunRawStatus(ctx context.Context, patch *api.TaskCheckRunStatusPatch) (*taskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	taskCheckRun, err := s.patchTaskCheckRunStatusImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRun, nil
}

// patchTaskCheckRunStatusImpl updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *Store) patchTaskCheckRunStatusImpl(ctx context.Context, tx *sql.Tx, patch *api.TaskCheckRunStatusPatch) (*taskCheckRunRaw, error) {
	// Build UPDATE clause.
	if patch.Result == "" {
		patch.Result = "{}"
	}
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	set, args = append(set, "code = $3"), append(args, patch.Code)
	set, args = append(set, "result = $4"), append(args, patch.Result)

	// Build WHERE clause.
	where := []string{"1 = 1"}
	if v := patch.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		UPDATE task_check_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, status, type, code, comment, result, payload
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var tRaw *taskCheckRunRaw
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
		tRaw = &taskCheckRunRaw
	}
	if tRaw != nil {
		return tRaw, nil
	}
	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task check run ID not found: %d", *patch.ID)}
}

func (s *Store) findTaskCheckRunImpl(ctx context.Context, tx *sql.Tx, find *api.TaskCheckRunFind) ([]*taskCheckRunRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
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

	orderAndLimit := ""
	if find.Latest {
		orderAndLimit = " ORDER BY updated_ts DESC LIMIT 1"
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
		WHERE `+strings.Join(where, " AND ")+orderAndLimit,
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
