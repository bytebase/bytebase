package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/bytebase/bytebase"
)

var (
	_ TaskRunService = (*DefaultTaskRunService)(nil)
)

// TaskRunService represents a service for managing taskRun.
type DefaultTaskRunService struct {
	l  *log.Logger
	db *sql.DB
}

// newTaskRunService returns a new TaskRunService.
func newTaskRunService(logger *log.Logger, db *sql.DB) *DefaultTaskRunService {
	return &DefaultTaskRunService{l: logger, db: db}
}

// CreateTaskRun creates a new taskRun.
func (s *DefaultTaskRunService) CreateTaskRun(ctx context.Context, create *TaskRunCreate) (*TaskRun, error) {
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
func (s *DefaultTaskRunService) FindTaskRunList(ctx context.Context, find *TaskRunFind) ([]*TaskRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findTaskRunList(ctx, tx, find)
	if err != nil {
		return []*TaskRun{}, err
	}

	return list, nil
}

// FindTaskRun retrieves a single taskRun based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *DefaultTaskRunService) FindTaskRun(ctx context.Context, find *TaskRunFind) (*TaskRun, error) {
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
		s.l.Printf("found mulitple task runs: %d, expect 1\n", len(list))
	}
	return list[0], nil
}

// PatchTaskRunStatus updates an existing taskRun by ID.
// Returns ENOTFOUND if taskRun does not exist.
func (s *DefaultTaskRunService) PatchTaskRunStatus(ctx context.Context, patch *TaskRunStatusPatch) (*TaskRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	taskRunPatch := &taskRunPatch{
		ID:     patch.ID,
		Status: &patch.Status,
	}
	taskRun, err := patchTaskRun(ctx, tx, taskRunPatch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return taskRun, nil
}

// createTaskRun creates a new taskRun.
func createTaskRun(ctx context.Context, tx *sql.Tx, create *TaskRunCreate) (*TaskRun, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO task_run (
			task_id,
			name,
			`+"`status`,"+`	
			`+"`type`,"+`
			payload	
		)
		VALUES (?, ?, 'PENDING', ?, ?)
		RETURNING id, created_ts, updated_ts, task_id, name, `+"`status`, `type`, payload"+`
	`,
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
	var taskRun TaskRun
	if err := row.Scan(
		&taskRun.ID,
		&taskRun.CreatedTs,
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

func findTaskRunList(ctx context.Context, tx *sql.Tx, find *TaskRunFind) (_ []*TaskRun, err error) {
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
		    created_ts,
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
	list := make([]*TaskRun, 0)
	for rows.Next() {
		var taskRun TaskRun
		if err := rows.Scan(
			&taskRun.ID,
			&taskRun.CreatedTs,
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

type taskRunPatch struct {
	ID TaskRunId

	Status *TaskRunStatus
}

// patchTaskRun updates a taskRun by ID. Returns the new state of the taskRun after update.
func patchTaskRun(ctx context.Context, tx *sql.Tx, patch *taskRunPatch) (*TaskRun, error) {
	// Build UPDATE clause.
	set, args := []string{}, []interface{}{}
	if v := patch.Status; v != nil {
		set, args = append(set, "`status` = ?"), append(args, TaskRunStatus(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, created_ts, updated_ts, task_id, name, `+"`status`, `type`, payload"+`
	`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if row.Next() {
		var taskRun TaskRun
		if err := row.Scan(
			&taskRun.ID,
			&taskRun.CreatedTs,
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

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("task run ID not found: %s", patch.ID)}
}
