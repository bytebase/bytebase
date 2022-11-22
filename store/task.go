package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/metric"
)

// taskRaw is the store model for an Task.
// Fields have exactly the same meanings as Task.
type taskRaw struct {
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

// toTask creates an instance of Task based on the taskRaw.
// This is intended to be called when we need to compose an Task relationship.
func (raw *taskRaw) toTask() *api.Task {
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

// BatchCreateTask creates tasks in batch.
func (s *Store) BatchCreateTask(ctx context.Context, creates []*api.TaskCreate) ([]*api.Task, error) {
	taskRawList, err := s.batchCreateTaskRaw(ctx, creates)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Task with TaskCreate %+v", creates)
	}
	var taskList []*api.Task
	for _, taskRaw := range taskRawList {
		task, err := s.composeTask(ctx, taskRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Task with taskRaw[%+v]", taskRaw)
		}
		taskList = append(taskList, task)
	}
	return taskList, nil
}

// CreateTask creates an instance of Task.
func (s *Store) CreateTask(ctx context.Context, create *api.TaskCreate) (*api.Task, error) {
	taskRaw, err := s.createTaskRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Task with TaskCreate[%+v]", create)
	}
	task, err := s.composeTask(ctx, taskRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Task with taskRaw[%+v]", taskRaw)
	}
	return task, nil
}

// GetTaskByID gets an instance of Task.
func (s *Store) GetTaskByID(ctx context.Context, id int) (*api.Task, error) {
	find := &api.TaskFind{ID: &id}
	taskRaw, err := s.getTaskRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Task with ID %d", id)
	}
	if taskRaw == nil {
		return nil, nil
	}
	task, err := s.composeTask(ctx, taskRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Task with taskRaw[%+v]", taskRaw)
	}
	return task, nil
}

// FindTask finds a list of Task instances.
func (s *Store) FindTask(ctx context.Context, find *api.TaskFind, returnOnErr bool) ([]*api.Task, error) {
	taskRawList, err := s.findTaskRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Task list with TaskFind[%+v]", find)
	}
	var taskList []*api.Task
	for _, raw := range taskRawList {
		task, err := s.composeTask(ctx, raw)
		if err != nil {
			if returnOnErr {
				return nil, errors.Wrapf(err, "failed to compose Task with taskRaw[%+v]", raw)
			}
			log.Error("failed to compose Task",
				zap.Any("taskRaw", raw),
				zap.Error(err))
			continue
		}
		taskList = append(taskList, task)
	}
	return taskList, nil
}

// PatchTask patches an instance of Task.
func (s *Store) PatchTask(ctx context.Context, patch *api.TaskPatch) (*api.Task, error) {
	taskRaw, err := s.patchTaskRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Task with TaskPatch[%+v]", patch)
	}
	task, err := s.composeTask(ctx, taskRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Task with taskRaw[%+v]", taskRaw)
	}
	return task, nil
}

// PatchTaskStatus patches a list of TaskStatus.
func (s *Store) PatchTaskStatus(ctx context.Context, patch *api.TaskStatusPatch) ([]*api.Task, error) {
	taskRawList, err := s.patchTaskRawStatus(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch TaskStatus with TaskStatusPatch[%+v]", patch)
	}
	var taskList []*api.Task
	for _, taskRaw := range taskRawList {
		task, err := s.composeTask(ctx, taskRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose TaskStatus with taskRaw[%+v]", taskRaw)
		}
		taskList = append(taskList, task)
	}
	return taskList, nil
}

// CountTaskGroupByTypeAndStatus counts the number of TaskGroup and group by TaskType.
// Used for the metric collector.
func (s *Store) CountTaskGroupByTypeAndStatus(ctx context.Context) ([]*metric.TaskCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT type, status, COUNT(*)
		FROM task
		WHERE (id <= 102 AND updater_id != 1) OR id > 102
		GROUP BY type, status`,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var res []*metric.TaskCountMetric
	for rows.Next() {
		var metric metric.TaskCountMetric
		if err := rows.Scan(&metric.Type, &metric.Status, &metric.Count); err != nil {
			return nil, FormatError(err)
		}
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return res, nil
}

//
// private functions
//

func (s *Store) composeTask(ctx context.Context, raw *taskRaw) (*api.Task, error) {
	task := raw.toTask()

	creator, err := s.GetPrincipalByID(ctx, task.CreatorID)
	if err != nil {
		return nil, err
	}
	task.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, task.UpdaterID)
	if err != nil {
		return nil, err
	}
	task.Updater = updater

	for _, taskRun := range task.TaskRunList {
		creator, err := s.GetPrincipalByID(ctx, taskRun.CreatorID)
		if err != nil {
			return nil, err
		}
		taskRun.Creator = creator

		updater, err := s.GetPrincipalByID(ctx, taskRun.UpdaterID)
		if err != nil {
			return nil, err
		}
		taskRun.Updater = updater
	}

	for _, taskCheckRun := range task.TaskCheckRunList {
		creator, err := s.GetPrincipalByID(ctx, taskCheckRun.CreatorID)
		if err != nil {
			return nil, err
		}
		taskCheckRun.Creator = creator

		updater, err := s.GetPrincipalByID(ctx, taskCheckRun.UpdaterID)
		if err != nil {
			return nil, err
		}
		taskCheckRun.Updater = updater
	}

	blockedBy := []string{}
	taskDAGList, err := s.FindTaskDAGList(ctx, &api.TaskDAGFind{ToTaskID: &raw.ID})
	if err != nil {
		return nil, err
	}
	for _, taskDAG := range taskDAGList {
		blockedBy = append(blockedBy, strconv.Itoa(taskDAG.FromTaskID))
	}
	task.BlockedBy = blockedBy

	instance, err := s.GetInstanceByID(ctx, task.InstanceID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found with ID %v", task.InstanceID)
	}
	task.Instance = instance

	if task.DatabaseID != nil {
		database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: task.DatabaseID})
		if err != nil {
			return nil, err
		}
		if database == nil {
			return nil, errors.Errorf("database not found with ID %v", task.DatabaseID)
		}
		task.Database = database
	}

	return task, nil
}

// batchCreateTaskRaw creates tasks in batch.
func (s *Store) batchCreateTaskRaw(ctx context.Context, creates []*api.TaskCreate) ([]*taskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	taskList, err := s.createTaskImpl(ctx, tx, creates...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskList, nil
}

// createTaskRaw creates a new task.
func (s *Store) createTaskRaw(ctx context.Context, create *api.TaskCreate) (*taskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	taskRawList, err := s.createTaskImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if len(taskRawList) != 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d tasks, expect 1", len(taskRawList))}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	return taskRawList[0], nil
}

// findTaskRaw retrieves a list of tasks based on find.
func (s *Store) findTaskRaw(ctx context.Context, find *api.TaskFind) ([]*taskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTaskImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getTaskRaw retrieves a single task based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getTaskRaw(ctx context.Context, find *api.TaskFind) (*taskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	return s.getTaskRawTx(ctx, tx, find)
}

func (s *Store) getTaskRawTx(ctx context.Context, tx *Tx, find *api.TaskFind) (*taskRaw, error) {
	list, err := s.findTaskImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d tasks with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// patchTaskRaw updates an existing task.
// Returns ENOTFOUND if task does not exist.
func (s *Store) patchTaskRaw(ctx context.Context, patch *api.TaskPatch) (*taskRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	task, err := s.patchTaskImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return task, nil
}

// patchTaskRawStatus updates existing task statuses and the corresponding task run statuses atomically.
// Returns ENOTFOUND if tasks do not exist.
func (s *Store) patchTaskRawStatus(ctx context.Context, patch *api.TaskStatusPatch) ([]*taskRaw, error) {
	// Without using serializable isolation transaction, we will get race condition and have multiple task runs inserted because
	// we do a read and write on task, without guaranteed consistency on task runs.
	// Once we have multiple task runs, the task will get to unrecoverable state because find task run will fail with two active runs.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	taskList, err := s.patchTaskStatusImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskList, nil
}

// createTaskImpl creates tasks.
func (*Store) createTaskImpl(ctx context.Context, tx *Tx, creates ...*api.TaskCreate) ([]*taskRaw, error) {
	var query strings.Builder
	var values []interface{}
	var queryValues []string

	if _, err := query.WriteString(
		`INSERT INTO task (
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
		VALUES
    `); err != nil {
		return nil, err
	}
	for i, create := range creates {
		if create.Payload == "" {
			create.Payload = "{}"
		}
		values = append(values,
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
		const count = 11
		queryValues = append(queryValues, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*count+1, i*count+2, i*count+3, i*count+4, i*count+5, i*count+6, i*count+7, i*count+8, i*count+9, i*count+10, i*count+11))
	}
	if _, err := query.WriteString(strings.Join(queryValues, ",")); err != nil {
		return nil, err
	}
	if _, err := query.WriteString(` RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts`); err != nil {
		return nil, err
	}

	var taskRawList []*taskRaw
	rows, err := tx.QueryContext(ctx, query.String(), values...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var taskRaw taskRaw
		var databaseID sql.NullInt32
		if err := rows.Scan(
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
		taskRawList = append(taskRawList, &taskRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return taskRawList, nil
}

func (s *Store) findTaskImpl(ctx context.Context, tx *Tx, find *api.TaskFind) ([]*taskRaw, error) {
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
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("status in (%s)", strings.Join(list, ",")))
	}
	if v := find.TypeList; v != nil {
		var list []string
		for _, taskType := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, taskType)
		}
		where = append(where, fmt.Sprintf("type in (%s)", strings.Join(list, ",")))
	}
	if v := find.Payload; v != "" {
		where = append(where, v)
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
	var taskRawList []*taskRaw
	for rows.Next() {
		var taskRaw taskRaw
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
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	for _, taskRaw := range taskRawList {
		taskRunFind := &api.TaskRunFind{
			TaskID: &taskRaw.ID,
		}
		taskRunRawList, err := s.findTaskRunImpl(ctx, tx, taskRunFind)
		if err != nil {
			return nil, err
		}
		taskRaw.TaskRunRawList = taskRunRawList

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskID: &taskRaw.ID,
		}
		taskCheckRunRawList, err := s.findTaskCheckRunImpl(ctx, tx, taskCheckRunFind)
		if err != nil {
			return nil, err
		}
		taskRaw.TaskCheckRunRawList = taskCheckRunRawList
	}
	return taskRawList, nil
}

// patchTaskImpl updates a task by ID. Returns the new state of the task after update.
func (*Store) patchTaskImpl(ctx context.Context, tx *Tx, patch *api.TaskPatch) (*taskRaw, error) {
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
	if v := patch.RollbackFrom; v != nil {
		set, args = append(set, fmt.Sprintf("rollback_from = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ID)

	var taskRaw taskRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts
	`, len(args)),
		args...,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("task not found with ID %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &taskRaw, nil
}

// patchTaskStatusImpl updates task status by IDList. Returns the new state of the tasks after update.
func (s *Store) patchTaskStatusImpl(ctx context.Context, tx *Tx, patch *api.TaskStatusPatch) ([]*taskRaw, error) {
	// Updates the corresponding task run if applicable.
	// We update the task run first because updating task below returns row and it's a bit complicated to
	// arrange code to prevent that opening row interfering with the task run update.
	for _, id := range patch.IDList {
		taskFind := &api.TaskFind{
			ID: &id,
		}
		taskRawObj, err := s.getTaskRawTx(ctx, tx, taskFind)
		if err != nil {
			return nil, err
		}
		if taskRawObj == nil {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("task ID not found: %d", id)}
		}

		taskRunFind := &api.TaskRunFind{
			TaskID: &taskRawObj.ID,
			StatusList: &[]api.TaskRunStatus{
				api.TaskRunRunning,
			},
		}
		taskRunRaw, err := s.getTaskRunRawTx(ctx, tx, taskRunFind)
		if err != nil {
			return nil, err
		}
		if taskRunRaw == nil {
			if patch.Status == api.TaskRunning {
				taskRunCreate := &api.TaskRunCreate{
					CreatorID: patch.UpdaterID,
					TaskID:    taskRawObj.ID,
					Name:      fmt.Sprintf("%s %d", taskRawObj.Name, time.Now().Unix()),
					Type:      taskRawObj.Type,
					Payload:   taskRawObj.Payload,
				}
				// insert a running taskRun
				if _, err := s.createTaskRunImpl(ctx, tx, taskRunCreate); err != nil {
					return nil, err
				}
			}
		} else {
			if patch.Status == api.TaskRunning {
				return nil, errors.Errorf("task is already running: %v", taskRawObj.Name)
			}
			taskRunStatusPatch := &api.TaskRunStatusPatch{
				ID:        &taskRunRaw.ID,
				UpdaterID: patch.UpdaterID,
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
			if _, err := s.patchTaskRunStatusImpl(ctx, tx, taskRunStatusPatch); err != nil {
				return nil, err
			}
		}
	}

	// Updates the task
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	var ids []string
	for _, id := range patch.IDList {
		ids = append(ids, strconv.Itoa(id))
	}

	// Execute update query with RETURNING.
	rows, err := tx.QueryContext(ctx, `
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id in (`+strings.Join(ids, ",")+`) 
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, pipeline_id, stage_id, instance_id, database_id, name, status, type, payload, earliest_allowed_ts
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var taskRawList []*taskRaw
	for rows.Next() {
		var taskRaw taskRaw
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

	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	for _, taskRaw := range taskRawList {
		taskRunRawList, err := s.findTaskRunImpl(ctx, tx, &api.TaskRunFind{
			TaskID: &taskRaw.ID,
		})
		if err != nil {
			return nil, err
		}
		taskRaw.TaskRunRawList = taskRunRawList

		taskCheckRunFind := &api.TaskCheckRunFind{
			TaskID: &taskRaw.ID,
		}
		taskCheckRunRawList, err := s.findTaskCheckRunImpl(ctx, tx, taskCheckRunFind)
		if err != nil {
			return nil, err
		}
		taskRaw.TaskCheckRunRawList = taskCheckRunRawList
	}

	return taskRawList, nil
}
