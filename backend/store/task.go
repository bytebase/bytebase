package store

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	TaskRunService api.TaskRunService
}

// NewTaskService returns a new instance of TaskService.
func NewTaskService(logger *zap.Logger, db *DB, taskRunService api.TaskRunService) *TaskService {
	return &TaskService{l: logger, db: db, TaskRunService: taskRunService}
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

	return s.findTask(ctx, tx, find)
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
			pipeline_id,
			stage_id,
			instance_id,
			database_id,
			name,
			`+"`status`,"+`	
			`+"`type`,"+`
			payload	
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, `+"`status`, `type`, payload"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.PipelineId,
		create.StageId,
		create.InstanceId,
		create.DatabaseId,
		create.Name,
		create.Status,
		create.Type,
		create.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var task api.Task
	task.TaskRunList = []*api.TaskRun{}
	if err := row.Scan(
		&task.ID,
		&task.CreatorId,
		&task.CreatedTs,
		&task.UpdaterId,
		&task.UpdatedTs,
		&task.PipelineId,
		&task.StageId,
		&task.InstanceId,
		&task.DatabaseId,
		&task.Name,
		&task.Status,
		&task.Type,
		&task.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &task, nil
}

func (s *TaskService) findTask(ctx context.Context, tx *Tx, find *api.TaskFind) (_ *api.Task, err error) {
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

func (s *TaskService) findTaskList(ctx context.Context, tx *Tx, find *api.TaskFind) (_ []*api.Task, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.PipelineId; v != nil {
		where, args = append(where, "pipeline_id = ?"), append(args, *v)
	}
	if v := find.StageId; v != nil {
		where, args = append(where, "stage_id = ?"), append(args, *v)
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
			pipeline_id,
			stage_id,
			instance_id,
			database_id,
		    name,
		    `+"`status`,"+`
			`+"`type`,"+`
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
			&task.PipelineId,
			&task.StageId,
			&task.InstanceId,
			&task.DatabaseId,
			&task.Name,
			&task.Status,
			&task.Type,
			&task.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		taskRunFind := &api.TaskRunFind{
			TaskId: &task.ID,
		}
		task.TaskRunList, err = s.TaskRunService.FindTaskRunList(ctx, tx.Tx, taskRunFind)
		if err != nil {
			return nil, err
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
	taskFind := &api.TaskFind{
		ID: &patch.ID,
	}
	task, err := s.findTask(ctx, tx, taskFind)
	if err != nil {
		return nil, err
	}

	if !(task.Status == api.TaskPendingApproval && patch.Status == api.TaskPending) {
		taskRunFind := &api.TaskRunFind{
			TaskId: &task.ID,
			StatusList: []api.TaskRunStatus{
				api.TaskRunRunning,
			},
		}
		taskRun, err := s.TaskRunService.FindTaskRun(ctx, tx.Tx, taskRunFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				if patch.Status != api.TaskRunning {
					return nil, fmt.Errorf("no applicable running task to change status")
				}
			} else {
				return nil, err
			}
		} else if patch.Status == api.TaskRunning {
			return nil, fmt.Errorf("task is already running: %v", task.Name)
		}

		if patch.Status == api.TaskRunning {
			taskRunCreate := &api.TaskRunCreate{
				CreatorId: patch.UpdaterId,
				TaskId:    task.ID,
				Name:      fmt.Sprintf("%s %d", task.Name, time.Now().Unix()),
				Type:      task.Type,
				Payload:   task.Payload,
			}
			if _, err := s.TaskRunService.CreateTaskRun(ctx, tx.Tx, taskRunCreate); err != nil {
				return nil, err
			}
		} else {
			taskRunStatusPatch := &api.TaskRunStatusPatch{
				ID:     &taskRun.ID,
				TaskId: &patch.ID,
			}
			switch patch.Status {
			case api.TaskDone:
				taskRunStatusPatch.Status = api.TaskRunDone
			case api.TaskFailed:
				taskRunStatusPatch.Status = api.TaskRunFailed
				taskRunStatusPatch.Error = &patch.Comment
			case api.TaskPending:
			case api.TaskPendingApproval:
			case api.TaskCanceled:
				taskRunStatusPatch.Status = api.TaskRunCanceled
			}
			if _, err := s.TaskRunService.PatchTaskRunStatus(ctx, tx.Tx, taskRunStatusPatch); err != nil {
				return nil, err
			}
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
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, `+"`status`, `type`, payload"+`
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
			&task.PipelineId,
			&task.StageId,
			&task.InstanceId,
			&task.DatabaseId,
			&task.Name,
			&task.Status,
			&task.Type,
			&task.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		taskRunFind := &api.TaskRunFind{
			TaskId: &task.ID,
		}
		task.TaskRunList, err = s.TaskRunService.FindTaskRunList(ctx, tx.Tx, taskRunFind)
		if err != nil {
			return nil, err
		}

		return &task, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("task ID not found: %d", patch.ID)}
}
