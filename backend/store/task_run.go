package store

import (
	"context"
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
func (s *TaskRunService) CreateTaskRun(ctx context.Context, create *api.TaskRunCreate) (*api.TaskRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	taskRun, err := createTaskRun(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return taskRun, nil
}

// FindTaskRunList retrieves a list of taskRuns based on find.
func (s *TaskRunService) FindTaskRunList(ctx context.Context, find *api.TaskRunFind) ([]*api.TaskRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findTaskRunList(ctx, tx, find)
	if err != nil {
		return []*api.TaskRun{}, err
	}

	return list, nil
}

// FindTaskRun retrieves a single taskRun based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *TaskRunService) FindTaskRun(ctx context.Context, find *api.TaskRunFind) (*api.TaskRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findTaskRunList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("task run not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warn(fmt.Sprintf("found mulitple task runs: %d, expect 1\n", len(list)))
	}
	return list[0], nil
}

// createTaskRun creates a new taskRun.
func createTaskRun(ctx context.Context, tx *Tx, create *api.TaskRunCreate) (*api.TaskRun, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO task_run (
			creator_id,
			updater_id,
			workspace_id,
			task_id,
			name,
			`+"`status`,"+`	
			`+"`type`,"+`
			payload	
		)
		VALUES (?, ?, ?, ?, ?, 'PENDING', ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, `+"`status`, `type`, payload"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.WorkspaceId,
		create.TaskId,
		create.Name,
		create.Type,
		create.Payload,
	)

	if err != nil {
		return nil, err
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
		&taskRun.Payload,
	); err != nil {
		return nil, err
	}

	return &taskRun, nil
}

func findTaskRunList(ctx context.Context, tx *Tx, find *api.TaskRunFind) (_ []*api.TaskRun, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.Status; v != nil {
		where, args = append(where, "`status` = ?"), append(args, *v)
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
			payload	
		FROM task_run
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
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
			&taskRun.Payload,
		); err != nil {
			return nil, err
		}

		list = append(list, &taskRun)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}
