package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.TaskService = (*TaskService)(nil)
)

// TaskService represents a service for managing task.
type TaskService struct {
	l  *zap.Logger
	db *DB

	TaskRunService      api.TaskRunService
	TaskCheckRunService api.TaskCheckRunService
}

// NewTaskService returns a new instance of TaskService.
func NewTaskService(logger *zap.Logger, db *DB, taskRunService api.TaskRunService, taskCheckRunService api.TaskCheckRunService) *TaskService {
	return &TaskService{l: logger, db: db, TaskRunService: taskRunService, TaskCheckRunService: taskCheckRunService}
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
// Returns ECONFLICT if finding more than 1 matching records.
func (s *TaskService) FindTask(ctx context.Context, find *api.TaskFind) (*api.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	return s.findTask(ctx, tx, find)
}

// PatchTask updates an existing task.
// Returns ENOTFOUND if task does not exist.
func (s *TaskService) PatchTask(ctx context.Context, patch *api.TaskPatch) (*api.Task, error) {
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
	var row *sql.Rows
	var err error

	if create.DatabaseID == nil {
		row, err = tx.QueryContext(ctx, `
		INSERT INTO task (
			creator_id,
			updater_id,
			pipeline_id,
			stage_id,
			instance_id,
			name,
			`+"`status`,"+`
			`+"`type`,"+`
			payload,
			earliest_allowed_ts
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, `+"`status`, `type`, payload, earliest_allowed_ts"+`
	`,
			create.CreatorID,
			create.CreatorID,
			create.PipelineID,
			create.StageID,
			create.InstanceID,
			create.Name,
			create.Status,
			create.Type,
			create.Payload,
			create.EarliestAllowedTs,
		)
	} else {
		row, err = tx.QueryContext(ctx, `
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
			payload,
			earliest_allowed_ts
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, `+"`status`, `type`, payload, earliest_allowed_ts"+`
	`,
			create.CreatorID,
			create.CreatorID,
			create.PipelineID,
			create.StageID,
			create.InstanceID,
			create.DatabaseID,
			create.Name,
			create.Status,
			create.Type,
			create.Payload,
			create.EarliestAllowedTs,
		)
	}

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var task api.Task
	var databaseID sql.NullInt32
	task.TaskRunList = []*api.TaskRun{}
	task.TaskCheckRunList = []*api.TaskCheckRun{}
	if err := row.Scan(
		&task.ID,
		&task.CreatorID,
		&task.CreatedTs,
		&task.UpdaterID,
		&task.UpdatedTs,
		&task.PipelineID,
		&task.StageID,
		&task.InstanceID,
		&databaseID,
		&task.Name,
		&task.Status,
		&task.Type,
		&task.Payload,
		&task.EarliestAllowedTs,
	); err != nil {
		return nil, FormatError(err)
	}

	if databaseID.Valid {
		val := int(databaseID.Int32)
		task.DatabaseID = &val
	}

	return &task, nil
}

func (s *TaskService) findTask(ctx context.Context, tx *Tx, find *api.TaskFind) (_ *api.Task, err error) {
	list, err := s.findTaskList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d tasks with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

func (s *TaskService) findTaskList(ctx context.Context, tx *Tx, find *api.TaskFind) (_ []*api.Task, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, "pipeline_id = ?"), append(args, *v)
	}
	if v := find.StageID; v != nil {
		where, args = append(where, "stage_id = ?"), append(args, *v)
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
			pipeline_id,
			stage_id,
			instance_id,
			database_id,
		    name,
		    `+"`status`,"+`
			`+"`type`,"+`
			payload,
			earliest_allowed_ts
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
			&task.CreatorID,
			&task.CreatedTs,
			&task.UpdaterID,
			&task.UpdatedTs,
			&task.PipelineID,
			&task.StageID,
			&task.InstanceID,
			&task.DatabaseID,
			&task.Name,
			&task.Status,
			&task.Type,
			&task.Payload,
			&task.EarliestAllowedTs,
		); err != nil {
			return nil, FormatError(err)
		}

		taskRunFind := &api.TaskRunFind{
			TaskID: &task.ID,
		}
		task.TaskRunList, err = s.TaskRunService.FindTaskRunListTx(ctx, tx.Tx, taskRunFind)
		if err != nil {
			return nil, err
		}

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskID: &task.ID,
		}
		task.TaskCheckRunList, err = s.TaskCheckRunService.FindTaskCheckRunListTx(ctx, tx.Tx, taskCheckRunFind)
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
func (s *TaskService) patchTask(ctx context.Context, tx *Tx, patch *api.TaskPatch) (*api.Task, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Payload; v != nil {
		set, args = append(set, "payload = ?"), append(args, *v)
	}
	if v := patch.EarliestAllowedTs; v != nil {
		set, args = append(set, "earliest_allowed_ts = ?"), append(args, *v)
	}
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, `+"`status`, `type`, payload, earliest_allowed_ts"+`
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
			&task.CreatorID,
			&task.CreatedTs,
			&task.UpdaterID,
			&task.UpdatedTs,
			&task.PipelineID,
			&task.StageID,
			&task.InstanceID,
			&task.DatabaseID,
			&task.Name,
			&task.Status,
			&task.Type,
			&task.Payload,
			&task.EarliestAllowedTs,
		); err != nil {
			return nil, FormatError(err)
		}

		return &task, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task ID not found: %d", patch.ID)}
}

// patchTaskStatus updates a task status by ID. Returns the new state of the task after update.
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
	if task == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task ID not found: %d", patch.ID)}
	}

	if !(task.Status == api.TaskPendingApproval && patch.Status == api.TaskPending) {
		taskRunFind := &api.TaskRunFind{
			TaskID: &task.ID,
			StatusList: &[]api.TaskRunStatus{
				api.TaskRunRunning,
			},
		}
		taskRun, err := s.TaskRunService.FindTaskRunTx(ctx, tx.Tx, taskRunFind)
		if err != nil {
			return nil, err
		}
		if taskRun == nil {
			if patch.Status != api.TaskRunning {
				return nil, fmt.Errorf("no applicable running task to change status")
			}
			taskRunCreate := &api.TaskRunCreate{
				CreatorID: patch.UpdaterID,
				TaskID:    task.ID,
				Name:      fmt.Sprintf("%s %d", task.Name, time.Now().Unix()),
				Type:      task.Type,
				Payload:   task.Payload,
			}
			if _, err := s.TaskRunService.CreateTaskRunTx(ctx, tx.Tx, taskRunCreate); err != nil {
				return nil, err
			}
		} else {
			if patch.Status == api.TaskRunning {
				return nil, fmt.Errorf("task is already running: %v", task.Name)
			}
			taskRunStatusPatch := &api.TaskRunStatusPatch{
				ID:        &taskRun.ID,
				UpdaterID: patch.UpdaterID,
				TaskID:    &patch.ID,
				Code:      patch.Code,
				Result:    patch.Result,
				Comment:   patch.Comment,
			}
			switch patch.Status {
			case api.TaskDone:
				taskRunStatusPatch.Status = api.TaskRunDone
			case api.TaskFailed:
				taskRunStatusPatch.Status = api.TaskRunFailed
			case api.TaskPending:
			case api.TaskPendingApproval:
			case api.TaskCanceled:
				taskRunStatusPatch.Status = api.TaskRunCanceled
			}
			if _, err := s.TaskRunService.PatchTaskRunStatusTx(ctx, tx.Tx, taskRunStatusPatch); err != nil {
				return nil, err
			}
		}
	}

	// Updates the task
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "`status` = ?"), append(args, patch.Status)
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, `+"`status`, `type`, payload, earliest_allowed_ts"+`
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
			&task.CreatorID,
			&task.CreatedTs,
			&task.UpdaterID,
			&task.UpdatedTs,
			&task.PipelineID,
			&task.StageID,
			&task.InstanceID,
			&task.DatabaseID,
			&task.Name,
			&task.Status,
			&task.Type,
			&task.Payload,
			&task.EarliestAllowedTs,
		); err != nil {
			return nil, FormatError(err)
		}

		taskRunFind := &api.TaskRunFind{
			TaskID: &task.ID,
		}
		task.TaskRunList, err = s.TaskRunService.FindTaskRunListTx(ctx, tx.Tx, taskRunFind)
		if err != nil {
			return nil, err
		}

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskID: &task.ID,
		}
		task.TaskCheckRunList, err = s.TaskCheckRunService.FindTaskCheckRunListTx(ctx, tx.Tx, taskCheckRunFind)
		if err != nil {
			return nil, err
		}

		return &task, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task ID not found: %d", patch.ID)}
}
