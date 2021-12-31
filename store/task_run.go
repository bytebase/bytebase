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
	_ api.TaskRunService = (*TaskRunService)(nil)
)

// TaskRunService represents a service for managing taskRun.
type TaskRunService struct {
	l  *zap.Logger
	db *DB
}

// NewTaskRunService returns a new TaskRunService.
func NewTaskRunService(logger *zap.Logger, db *DB) *TaskRunService {
	return &TaskRunService{l: logger, db: db}
}

// CreateTaskRunTx creates a new taskRun.
func (s *TaskRunService) CreateTaskRunTx(ctx context.Context, tx *sql.Tx, create *api.TaskRunCreate) (*api.TaskRun, error) {
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
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, `+"`status`, `type`, code, comment, result, payload"+`
	`,
		create.CreatorID,
		create.CreatorID,
		create.TaskID,
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
		&taskRun.CreatorID,
		&taskRun.CreatedTs,
		&taskRun.UpdaterID,
		&taskRun.UpdatedTs,
		&taskRun.TaskID,
		&taskRun.Name,
		&taskRun.Status,
		&taskRun.Type,
		&taskRun.Code,
		&taskRun.Comment,
		&taskRun.Result,
		&taskRun.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &taskRun, nil
}

// FindTaskRunListTx retrieves a list of taskRuns based on find.
func (s *TaskRunService) FindTaskRunListTx(ctx context.Context, tx *sql.Tx, find *api.TaskRunFind) ([]*api.TaskRun, error) {
	list, err := s.findTaskRunList(ctx, tx, find)
	if err != nil {
		return []*api.TaskRun{}, err
	}

	return list, nil
}

// FindTaskRunTx retrieves a single taskRun based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *TaskRunService) FindTaskRunTx(ctx context.Context, tx *sql.Tx, find *api.TaskRunFind) (*api.TaskRun, error) {
	list, err := s.findTaskRunList(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d task runs with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchTaskRunStatusTx updates a taskRun status. Returns the new state of the taskRun after update.
func (s *TaskRunService) PatchTaskRunStatusTx(ctx context.Context, tx *sql.Tx, patch *api.TaskRunStatusPatch) (*api.TaskRun, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "`status` = ?"), append(args, patch.Status)
	if v := patch.Code; v != nil {
		set, args = append(set, "code = ?"), append(args, *v)
	}
	if v := patch.Comment; v != nil {
		set, args = append(set, "comment = ?"), append(args, *v)
	}
	if v := patch.Result; v != nil {
		set, args = append(set, "result = ?"), append(args, *v)
	}

	// Build WHERE clause.
	where := []string{"1 = 1"}
	if v := patch.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := patch.TaskID; v != nil {
		where, args = append(where, "task_id = ?"), append(args, *v)
	}

	row, err := tx.QueryContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, `+"`status`, `type`, code, comment, result, payload"+`
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
		&taskRun.CreatorID,
		&taskRun.CreatedTs,
		&taskRun.UpdaterID,
		&taskRun.UpdatedTs,
		&taskRun.TaskID,
		&taskRun.Name,
		&taskRun.Status,
		&taskRun.Type,
		&taskRun.Code,
		&taskRun.Comment,
		&taskRun.Result,
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
	if v := find.TaskID; v != nil {
		where, args = append(where, "task_id = ?"), append(args, *v)
	}

	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, "?")
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("`status` in (%s)", strings.Join(list, ",")))
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
			code,
			comment,
			result,
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
			&taskRun.CreatorID,
			&taskRun.CreatedTs,
			&taskRun.UpdaterID,
			&taskRun.UpdatedTs,
			&taskRun.TaskID,
			&taskRun.Name,
			&taskRun.Status,
			&taskRun.Type,
			&taskRun.Code,
			&taskRun.Comment,
			&taskRun.Result,
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
