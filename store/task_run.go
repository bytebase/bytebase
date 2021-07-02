package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.TaskRunService = (*TaskRunService)(nil)
)

// TaskRunService represents a service for managing taskRun.
type TaskRunService struct {
	l  *zap.Logger
	db *DB
}

// newTaskRunService returns a new TaskRunService.
func NewTaskRunService(logger *zap.Logger, db *DB) *TaskRunService {
	return &TaskRunService{l: logger, db: db}
}

// CreateTaskRun creates a new taskRun.
func (s *TaskRunService) CreateTaskRun(ctx context.Context, tx *sql.Tx, create *api.TaskRunCreate) (*api.TaskRun, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO task_run (
			creator_id,
			updater_id,
			task_id,
			name,
			`+"`status`,"+`
			`+"`type`,"+`
			payload
		)
		VALUES (?, ?, ?, ?, 'RUNNING', ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, `+"`status`, `type`, detail, payload"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.TaskId,
		create.Name,
		create.Type,
		create.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var taskRun api.TaskRun
	if err := row.Scan(
		&taskRun.ID,
		&taskRun.CreatorId,
		&taskRun.CreatedTs,
		&taskRun.UpdaterId,
		&taskRun.UpdatedTs,
		&taskRun.TaskId,
		&taskRun.Name,
		&taskRun.Status,
		&taskRun.Type,
		&taskRun.Detail,
		&taskRun.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &taskRun, nil
}

// FindTaskRunList retrieves a list of taskRuns based on find.
func (s *TaskRunService) FindTaskRunList(ctx context.Context, tx *sql.Tx, find *api.TaskRunFind) ([]*api.TaskRun, error) {
	list, err := s.findTaskRunList(ctx, tx, find)
	if err != nil {
		return []*api.TaskRun{}, err
	}

	return list, nil
}

// FindTaskRun retrieves a single taskRun based on find.
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *TaskRunService) FindTaskRun(ctx context.Context, tx *sql.Tx, find *api.TaskRunFind) (*api.TaskRun, error) {
	list, err := s.findTaskRunList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("task run not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &bytebase.Error{Code: bytebase.ECONFLICT, Message: fmt.Sprintf("found %d task runs with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchTaskRunStatus updates a taskRun status. Returns the new state of the taskRun after update.
func (s *TaskRunService) PatchTaskRunStatus(ctx context.Context, tx *sql.Tx, patch *api.TaskRunStatusPatch) (*api.TaskRun, error) {
	// Build UPDATE clause.
	set, args := []string{}, []interface{}{}
	set, args = append(set, "`status` = ?"), append(args, patch.Status)
	set, args = append(set, "detail = ?"), append(args, patch.Detail)

	// Build WHERE clause.
	where := []string{"1 = 1"}
	if v := patch.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := patch.TaskId; v != nil {
		where, args = append(where, "task_id = ?"), append(args, *v)
	}

	row, err := tx.QueryContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, `+"`status`, `type`, detail, payload"+`
	`,
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var taskRun api.TaskRun
	if err := row.Scan(
		&taskRun.ID,
		&taskRun.CreatorId,
		&taskRun.CreatedTs,
		&taskRun.UpdaterId,
		&taskRun.UpdatedTs,
		&taskRun.TaskId,
		&taskRun.Name,
		&taskRun.Status,
		&taskRun.Type,
		&taskRun.Detail,
		&taskRun.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &taskRun, nil
}

func (s *TaskRunService) findTaskRunList(ctx context.Context, tx *sql.Tx, find *api.TaskRunFind) (_ []*api.TaskRun, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.TaskId; v != nil {
		where, args = append(where, "task_id = ?"), append(args, *v)
	}

	if len(find.StatusList) > 0 {
		statusList := []string{}
		for _, status := range find.StatusList {
			statusList = append(statusList, fmt.Sprintf("%v", status))
		}
		where, args = append(where, "`status` IN (?)"), append(args, strings.Join(statusList, ", "))
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
		    created_ts,
			updater_id,
		    updated_ts,
			task_id,
			name,
			`+"`status`,"+`
			`+"`type`,"+`
			detail,
			payload
		FROM task_run
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.TaskRun, 0)
	for rows.Next() {
		var taskRun api.TaskRun
		if err := rows.Scan(
			&taskRun.ID,
			&taskRun.CreatorId,
			&taskRun.CreatedTs,
			&taskRun.UpdaterId,
			&taskRun.UpdatedTs,
			&taskRun.TaskId,
			&taskRun.Name,
			&taskRun.Status,
			&taskRun.Type,
			&taskRun.Detail,
			&taskRun.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &taskRun)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}
