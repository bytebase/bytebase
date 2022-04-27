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

// TaskService is the service for tasks.
type TaskService interface {
	CreateTask(ctx context.Context, create *api.TaskCreate) (*TaskRaw, error)
	FindTaskList(ctx context.Context, find *api.TaskFind) ([]*TaskRaw, error)
	FindTask(ctx context.Context, find *api.TaskFind) (*TaskRaw, error)
	PatchTask(ctx context.Context, patch *api.TaskPatch) (*TaskRaw, error)
	PatchTaskStatus(ctx context.Context, patch *api.TaskStatusPatch) (*TaskRaw, error)
}

// TaskRaw is the store model for an Task.
// Fields have exactly the same meanings as Task.
type TaskRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	PipelineID int
	StageID    int
	InstanceID int
	// Could be empty for creating database task when the task isn't yet completed successfully.
	DatabaseID          *int
	TaskRunRawList      []*taskRunRaw
	TaskCheckRunRawList []*taskCheckRunRaw

	// Domain specific fields
	Name              string
	Status            api.TaskStatus
	Type              api.TaskType
	Payload           string
	EarliestAllowedTs int64
	BlockedBy         []string
}

// ToTask creates an instance of Task based on the TaskRaw.
// This is intended to be called when we need to compose an Task relationship.
func (raw *TaskRaw) ToTask() *api.Task {
	task := &api.Task{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		PipelineID: raw.PipelineID,
		StageID:    raw.StageID,
		InstanceID: raw.InstanceID,
		// Could be empty for creating database task when the task isn't yet completed successfully.
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:              raw.Name,
		Status:            raw.Status,
		Type:              raw.Type,
		Payload:           raw.Payload,
		EarliestAllowedTs: raw.EarliestAllowedTs,
		BlockedBy:         raw.BlockedBy,
	}
	for _, taskRunRaw := range raw.TaskRunRawList {
		task.TaskRunList = append(task.TaskRunList, taskRunRaw.toTaskRun())
	}
	for _, taskCheckRunRaw := range raw.TaskCheckRunRawList {
		task.TaskCheckRunList = append(task.TaskCheckRunList, taskCheckRunRaw.toTaskCheckRun())
	}
	return task
}

var (
	_ TaskService = (*TaskServiceImpl)(nil)
)

// TaskServiceImpl represents a service for managing task.
type TaskServiceImpl struct {
	l  *zap.Logger
	db *DB

	store *Store
}

// NewTaskService returns a new instance of TaskService.
func NewTaskService(logger *zap.Logger, db *DB, store *Store) *TaskServiceImpl {
	return &TaskServiceImpl{l: logger, db: db, store: store}
}

// CreateTask creates a new task.
func (s *TaskServiceImpl) CreateTask(ctx context.Context, create *api.TaskCreate) (*TaskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	task, err := s.createTask(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return task, nil
}

// FindTaskList retrieves a list of tasks based on find.
func (s *TaskServiceImpl) FindTaskList(ctx context.Context, find *api.TaskFind) ([]*TaskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findTaskList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// FindTask retrieves a single task based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *TaskServiceImpl) FindTask(ctx context.Context, find *api.TaskFind) (*TaskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	return s.findTask(ctx, tx.PTx, find)
}

// PatchTask updates an existing task.
// Returns ENOTFOUND if task does not exist.
func (s *TaskServiceImpl) PatchTask(ctx context.Context, patch *api.TaskPatch) (*TaskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	task, err := s.patchTask(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return task, nil
}

// PatchTaskStatus updates an existing task status and the corresponding task run status atomically.
// Returns ENOTFOUND if task does not exist.
func (s *TaskServiceImpl) PatchTaskStatus(ctx context.Context, patch *api.TaskStatusPatch) (*TaskRaw, error) {
	// Without using serializable isolation transaction, we will get race condition and have multiple task runs inserted because
	// we do a read and write on task, without guanrantee consistency on task runs.
	// Once we have multiple task runs, the task will get to unrecoverable state because find task run will fail with two active runs.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	task, err := s.patchTaskStatus(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return task, nil
}

// createTask creates a new task.
func (s *TaskServiceImpl) createTask(ctx context.Context, tx *sql.Tx, create *api.TaskCreate) (*TaskRaw, error) {
	var row *sql.Rows
	var err error

	if create.Payload == "" {
		create.Payload = "{}"
	}
	if create.DatabaseID == nil {
		row, err = tx.QueryContext(ctx, `
		INSERT INTO task (
			creator_id,
			updater_id,
			pipeline_id,
			stage_id,
			instance_id,
			name,
			status,
			type,
			payload,
			earliest_allowed_ts
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts
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
			status,
			type,
			payload,
			earliest_allowed_ts
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts
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
	var taskRaw TaskRaw
	var databaseID sql.NullInt32
	if err := row.Scan(
		&taskRaw.ID,
		&taskRaw.CreatorID,
		&taskRaw.CreatedTs,
		&taskRaw.UpdaterID,
		&taskRaw.UpdatedTs,
		&taskRaw.PipelineID,
		&taskRaw.StageID,
		&taskRaw.InstanceID,
		&databaseID,
		&taskRaw.Name,
		&taskRaw.Status,
		&taskRaw.Type,
		&taskRaw.Payload,
		&taskRaw.EarliestAllowedTs,
	); err != nil {
		return nil, FormatError(err)
	}

	if databaseID.Valid {
		val := int(databaseID.Int32)
		taskRaw.DatabaseID = &val
	}

	return &taskRaw, nil
}

func (s *TaskServiceImpl) findTask(ctx context.Context, tx *sql.Tx, find *api.TaskFind) (*TaskRaw, error) {
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

func (s *TaskServiceImpl) findTaskList(ctx context.Context, tx *sql.Tx, find *api.TaskFind) ([]*TaskRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.StageID; v != nil {
		where, args = append(where, fmt.Sprintf("stage_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("status in (%s)", strings.Join(list, ",")))
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
			status,
			type,
			payload,
			earliest_allowed_ts
		FROM task
		WHERE `+strings.Join(where, " AND ")+` ORDER BY id ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into taskRawList.
	var taskRawList []*TaskRaw
	for rows.Next() {
		var taskRaw TaskRaw
		if err := rows.Scan(
			&taskRaw.ID,
			&taskRaw.CreatorID,
			&taskRaw.CreatedTs,
			&taskRaw.UpdaterID,
			&taskRaw.UpdatedTs,
			&taskRaw.PipelineID,
			&taskRaw.StageID,
			&taskRaw.InstanceID,
			&taskRaw.DatabaseID,
			&taskRaw.Name,
			&taskRaw.Status,
			&taskRaw.Type,
			&taskRaw.Payload,
			&taskRaw.EarliestAllowedTs,
		); err != nil {
			return nil, FormatError(err)
		}
		taskRawList = append(taskRawList, &taskRaw)
	}

	for _, taskRaw := range taskRawList {
		taskRunFind := &api.TaskRunFind{
			TaskID: &taskRaw.ID,
		}
		taskRunRawList, err := s.store.findTaskRunImpl(ctx, tx, taskRunFind)
		if err != nil {
			return nil, err
		}
		taskRaw.TaskRunRawList = taskRunRawList

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskID: &taskRaw.ID,
		}
		taskCheckRunRawList, err := s.store.findTaskCheckRunImpl(ctx, tx, taskCheckRunFind)
		if err != nil {
			return nil, err
		}
		taskRaw.TaskCheckRunRawList = taskCheckRunRawList
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return taskRawList, nil
}

// patchTask updates a task by ID. Returns the new state of the task after update.
func (s *TaskServiceImpl) patchTask(ctx context.Context, tx *sql.Tx, patch *api.TaskPatch) (*TaskRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.DatabaseID; v != nil {
		set, args = append(set, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		payload := "{}"
		if *v != "" {
			payload = *v
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
	}
	if v := patch.EarliestAllowedTs; v != nil {
		set, args = append(set, fmt.Sprintf("earliest_allowed_ts = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var taskRaw TaskRaw
		if err := row.Scan(
			&taskRaw.ID,
			&taskRaw.CreatorID,
			&taskRaw.CreatedTs,
			&taskRaw.UpdaterID,
			&taskRaw.UpdatedTs,
			&taskRaw.PipelineID,
			&taskRaw.StageID,
			&taskRaw.InstanceID,
			&taskRaw.DatabaseID,
			&taskRaw.Name,
			&taskRaw.Status,
			&taskRaw.Type,
			&taskRaw.Payload,
			&taskRaw.EarliestAllowedTs,
		); err != nil {
			return nil, FormatError(err)
		}

		return &taskRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task not found with ID %d", patch.ID)}
}

// patchTaskStatus updates a task status by ID. Returns the new state of the task after update.
func (s *TaskServiceImpl) patchTaskStatus(ctx context.Context, tx *sql.Tx, patch *api.TaskStatusPatch) (*TaskRaw, error) {
	// Updates the corresponding task run if applicable.
	// We update the task run first because updating task below returns row and it's a bit complicated to
	// arrange code to prevent that opening row interfering with the task run update.
	taskFind := &api.TaskFind{
		ID: &patch.ID,
	}
	taskRaw, err := s.findTask(ctx, tx, taskFind)
	if err != nil {
		return nil, err
	}
	if taskRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task ID not found: %d", patch.ID)}
	}

	if !(taskRaw.Status == api.TaskPendingApproval && patch.Status == api.TaskPending) {
		taskRunFind := &api.TaskRunFind{
			TaskID: &taskRaw.ID,
			StatusList: &[]api.TaskRunStatus{
				api.TaskRunRunning,
			},
		}
		taskRunRaw, err := s.store.getTaskRunRawTx(ctx, tx, taskRunFind)
		if err != nil {
			return nil, err
		}
		if taskRunRaw == nil {
			if patch.Status != api.TaskRunning {
				return nil, fmt.Errorf("no applicable running task to change status")
			}
			taskRunCreate := &api.TaskRunCreate{
				CreatorID: patch.UpdaterID,
				TaskID:    taskRaw.ID,
				Name:      fmt.Sprintf("%s %d", taskRaw.Name, time.Now().Unix()),
				Type:      taskRaw.Type,
				Payload:   taskRaw.Payload,
			}
			if _, err := s.store.createTaskRunImpl(ctx, tx, taskRunCreate); err != nil {
				return nil, err
			}
		} else {
			if patch.Status == api.TaskRunning {
				return nil, fmt.Errorf("task is already running: %v", taskRaw.Name)
			}
			taskRunStatusPatch := &api.TaskRunStatusPatch{
				ID:        &taskRunRaw.ID,
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
			if _, err := s.store.patchTaskRunStatusImpl(ctx, tx, taskRunStatusPatch); err != nil {
				return nil, err
			}
		}
	}

	// Updates the task
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = $3
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}

	var taskPatchedRaw *TaskRaw
	if row.Next() {
		var taskRaw TaskRaw
		if err := row.Scan(
			&taskRaw.ID,
			&taskRaw.CreatorID,
			&taskRaw.CreatedTs,
			&taskRaw.UpdaterID,
			&taskRaw.UpdatedTs,
			&taskRaw.PipelineID,
			&taskRaw.StageID,
			&taskRaw.InstanceID,
			&taskRaw.DatabaseID,
			&taskRaw.Name,
			&taskRaw.Status,
			&taskRaw.Type,
			&taskRaw.Payload,
			&taskRaw.EarliestAllowedTs,
		); err != nil {
			return nil, FormatError(err)
		}
		taskPatchedRaw = &taskRaw
	}
	if err := row.Close(); err != nil {
		return nil, err
	}

	if taskPatchedRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("task ID not found: %d", patch.ID)}
	}

	taskRunFind := &api.TaskRunFind{
		TaskID: &taskRaw.ID,
	}
	taskRunRawList, err := s.store.findTaskRunImpl(ctx, tx, taskRunFind)
	if err != nil {
		return nil, err
	}
	taskRaw.TaskRunRawList = taskRunRawList

	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID: &taskRaw.ID,
	}
	taskCheckRunRawList, err := s.store.findTaskCheckRunImpl(ctx, tx, taskCheckRunFind)
	if err != nil {
		return nil, err
	}
	taskRaw.TaskCheckRunRawList = taskCheckRunRawList

	return taskPatchedRaw, nil
}
