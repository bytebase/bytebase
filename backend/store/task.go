package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.TaskService = (*TaskService)(nil)
)

// TaskService represents a service for managing task.
type TaskService struct {
	l  *bytebase.Logger
	db *DB
}

// NewTaskService returns a new instance of TaskService.
func NewTaskService(logger *bytebase.Logger, db *DB) *TaskService {
	return &TaskService{l: logger, db: db}
}

// CreateTask creates a new task.
func (s *TaskService) CreateTask(ctx context.Context, create *api.TaskCreate) (*api.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	task, err := s.createTask(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return task, nil
}

// FindTaskList retrieves a list of tasks based on find.
func (s *TaskService) FindTaskList(ctx context.Context, find *api.TaskFind) ([]*api.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTaskList(ctx, tx, find)
	if err != nil {
		return []*api.Task{}, err
	}

	return list, nil
}

// FindTask retrieves a single task based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *TaskService) FindTask(ctx context.Context, find *api.TaskFind) (*api.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTaskList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("task not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found mulitple tasks: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchTaskByID updates an existing task by ID.
// Returns ENOTFOUND if task does not exist.
func (s *TaskService) PatchTaskByID(ctx context.Context, patch *api.TaskPatch) (*api.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	task, err := s.patchTask(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return task, nil
}

// createTask creates a new task.
func (s *TaskService) createTask(ctx context.Context, tx *Tx, create *api.TaskCreate) (*api.Task, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO task (
			creator_id,
			updater_id,
			workspace_id,
			pipeline_id,
			stage_id,
			database_id,
			name,
			`+"`status`,"+`	
			`+"`type`,"+`
			`+"`when`,"+`
			payload	
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, workspace_id, pipeline_id, stage_id, database_id, name, `+"`status`, `type`, `when`, payload"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.WorkspaceId,
		create.PipelineId,
		create.StageId,
		create.DatabaseId,
		create.Name,
		create.Type,
		create.When,
		create.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var task api.Task
	if err := row.Scan(
		&task.ID,
		&task.CreatorId,
		&task.CreatedTs,
		&task.UpdaterId,
		&task.UpdatedTs,
		&task.WorkspaceId,
		&task.PipelineId,
		&task.StageId,
		&task.DatabaseId,
		&task.Name,
		&task.Status,
		&task.Type,
		&task.When,
		&task.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &task, nil
}

func (s *TaskService) findTaskList(ctx context.Context, tx *Tx, find *api.TaskFind) (_ []*api.Task, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}
	if v := find.PipelineId; v != nil {
		where, args = append(where, "pipeline_id = ?"), append(args, *v)
	}
	if v := find.StageId; v != nil {
		where, args = append(where, "stage_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			workspace_id,
			pipeline_id,
			stage_id,
			database_id,
		    name,
		    `+"`status`,"+`
			`+"`type`,"+`
			`+"`when`,"+`
			payload
		FROM task
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Task, 0)
	for rows.Next() {
		var task api.Task
		if err := rows.Scan(
			&task.ID,
			&task.CreatorId,
			&task.CreatedTs,
			&task.UpdaterId,
			&task.UpdatedTs,
			&task.WorkspaceId,
			&task.PipelineId,
			&task.StageId,
			&task.DatabaseId,
			&task.Name,
			&task.Status,
			&task.Type,
			&task.When,
			&task.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &task)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchTask updates a task by ID. Returns the new state of the task after update.
func (s *TaskService) patchTask(ctx context.Context, tx *Tx, patch *api.TaskPatch) (*api.Task, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.Status; v != nil {
		set, args = append(set, "`status = ?"), append(args, api.TaskStatus(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, workspace_id, pipeline_id, stage_id, database_id, name, `+"`status`, `type`, `when`, payload"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var task api.Task
		if err := row.Scan(
			&task.ID,
			&task.CreatorId,
			&task.CreatedTs,
			&task.UpdaterId,
			&task.UpdatedTs,
			&task.WorkspaceId,
			&task.PipelineId,
			&task.StageId,
			&task.DatabaseId,
			&task.Name,
			&task.Status,
			&task.Type,
			&task.When,
			&task.Payload,
		); err != nil {
			return nil, FormatError(err)
		}
		return &task, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("task ID not found: %d", patch.ID)}
}
