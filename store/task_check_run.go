package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.TaskCheckRunService = (*TaskCheckRunService)(nil)
)

// TaskCheckRunService represents a service for managing taskCheckRun.
type TaskCheckRunService struct {
	l  *zap.Logger
	db *DB
}

// NewTaskCheckRunService returns a new TaskCheckRunService.
func NewTaskCheckRunService(logger *zap.Logger, db *DB) *TaskCheckRunService {
	return &TaskCheckRunService{l: logger, db: db}
}

// CreateTaskCheckRunIfNeeded creates a new taskCheckRun. See interface for the expected behavior
func (s *TaskCheckRunService) CreateTaskCheckRunIfNeeded(ctx context.Context, create *api.TaskCheckRunCreate) (*api.TaskCheckRunRaw, error) {
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

	taskCheckRunList, err := s.FindTaskCheckRunListTx(ctx, tx.PTx, taskCheckRunFind)
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
			s.l.Warn(fmt.Sprintf("Found %d task check run, expect at most 1", len(taskCheckRunList)),
				zap.Int("task_id", create.TaskID),
				zap.String("task_check_type", string(create.Type)),
			)
		}
		return taskCheckRunList[0], nil
	}

	taskCheckRun, err := s.createTaskCheckRunTx(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRun, nil
}

// createTaskCheckRunTx creates a new taskCheckRun.
func (s *TaskCheckRunService) createTaskCheckRunTx(ctx context.Context, tx *sql.Tx, create *api.TaskCheckRunCreate) (*api.TaskCheckRunRaw, error) {
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

	var tRaw *api.TaskCheckRunRaw
	for rows.Next() {
		var taskCheckRunRaw api.TaskCheckRunRaw
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

// FindTaskCheckRunList retrieves a list of taskCheckRuns based on find.
func (s *TaskCheckRunService) FindTaskCheckRunList(ctx context.Context, find *api.TaskCheckRunFind) ([]*api.TaskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findTaskCheckRunList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// FindTaskCheckRunListTx retrieves a list of taskCheckRuns based on find.
func (s *TaskCheckRunService) FindTaskCheckRunListTx(ctx context.Context, tx *sql.Tx, find *api.TaskCheckRunFind) ([]*api.TaskCheckRunRaw, error) {
	list, err := s.findTaskCheckRunList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// PatchTaskCheckRunStatus updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *TaskCheckRunService) PatchTaskCheckRunStatus(ctx context.Context, patch *api.TaskCheckRunStatusPatch) (*api.TaskCheckRunRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	taskCheckRun, err := s.patchTaskCheckRunStatusTx(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRun, nil
}

// patchTaskCheckRunStatusTx updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *TaskCheckRunService) patchTaskCheckRunStatusTx(ctx context.Context, tx *sql.Tx, patch *api.TaskCheckRunStatusPatch) (*api.TaskCheckRunRaw, error) {
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

	var tRaw *api.TaskCheckRunRaw
	for rows.Next() {
		var taskCheckRunRaw api.TaskCheckRunRaw
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

func (s *TaskCheckRunService) findTaskCheckRunList(ctx context.Context, tx *sql.Tx, find *api.TaskCheckRunFind) ([]*api.TaskCheckRunRaw, error) {
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
	var taskCheckRunRawList []*api.TaskCheckRunRaw
	for rows.Next() {
		var taskCheckRun api.TaskCheckRunRaw
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
