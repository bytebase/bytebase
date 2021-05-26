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
	_ api.TaskService = (*TaskService)(nil)
)

// TaskService represents a service for managing task.
type TaskService struct {
	l  *zap.Logger
	db *DB
}

// NewTaskService returns a new instance of TaskService.
func NewTaskService(logger *zap.Logger, db *DB) *TaskService {
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
		s.l.Warn(fmt.Sprintf("found mulitple tasks: %d, expect 1", len(list)))
	}
	return list[0], nil
}

// PatchTaskStatus updates an existing task status and the correspondng task run status atomically.
// Returns ENOTFOUND if task does not exist.
func (s *TaskService) PatchTaskStatus(ctx context.Context, patch *api.TaskStatusPatch) (*api.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	task, err := s.patchTaskStatus(ctx, tx, patch)
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
func (s *TaskService) patchTaskStatus(ctx context.Context, tx *Tx, patch *api.TaskStatusPatch) (*api.Task, error) {
	// Updates the corresponding task run if applicable.
	// We update the task run first because updating task below returns row and it's a bit complicated to
	// arrange code to prevent that opening row interfering with the task run update.
	if patch.TaskRunId != nil && patch.TaskRunStatus != nil {
		taskRunStatusPatch := &taskRunStatusPatch{
			ID:     patch.TaskRunId,
			TaskId: &patch.ID,
			Status: *patch.TaskRunStatus,
		}

		if err := patchTaskRunStatus(ctx, tx, taskRunStatusPatch); err != nil {
			return nil, FormatError(err)
		}
	}

	// Updates the task
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	set, args = append(set, "`status` = ?"), append(args, patch.Status)
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

type taskRunStatusPatch struct {
	ID *int

	// Related fields
	TaskId *int

	// Domain specific fields
	Status api.TaskRunStatus
}

// patchTaskRun updates a taskRun by ID. Returns the new state of the taskRun after update.
func patchTaskRunStatus(ctx context.Context, tx *Tx, patch *taskRunStatusPatch) error {
	// Build UPDATE clause.
	set, args := []string{}, []interface{}{}
	set, args = append(set, "`status` = ?"), append(args, patch.Status)

	// Build WHERE clause.
	where := []string{"1 = 1"}
	if v := patch.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := patch.TaskId; v != nil {
		where, args = append(where, "task_id = ?"), append(args, *v)
	}

	_, err := tx.ExecContext(ctx, `
		UPDATE task_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return err
	}

	return nil
}
