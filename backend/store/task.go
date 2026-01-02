package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TaskMessage is the message for tasks.
type TaskMessage struct {
	ID int

	// Related fields
	PlanID       int64
	InstanceID   string
	Environment  string // The environment ID (was stage_id). Could be empty if the task does not have an environment.
	DatabaseName *string

	// Domain specific fields
	Type    storepb.Task_Type
	Payload *storepb.Task

	LatestTaskRunStatus storepb.TaskRun_Status
	// UpdatedAt is the updated_at of latest task run related to the task.
	// If there are no task runs, it will be empty.
	UpdatedAt *time.Time
	// RunAt is the run_at of latest task run related to the task.
	RunAt *time.Time
}

func (t *TaskMessage) GetDatabaseName() string {
	if t == nil {
		return ""
	}
	if t.DatabaseName == nil {
		return ""
	}
	return *t.DatabaseName
}

// TaskFind is the API message for finding tasks.
type TaskFind struct {
	ID  *int
	IDs *[]int

	// Related fields
	PlanID       *int64
	PlanIDs      *[]int64
	Environment  *string
	InstanceID   *string
	DatabaseName *string

	// Domain specific fields
	TypeList *[]storepb.Task_Type

	LatestTaskRunStatusList *[]storepb.TaskRun_Status
}

// GetTaskByID gets a task by ID.
func (s *Store) GetTaskByID(ctx context.Context, id int) (*TaskMessage, error) {
	tasks, err := s.ListTasks(ctx, &TaskFind{ID: &id})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Task with ID %d", id)
	}
	if len(tasks) == 0 {
		return nil, nil
	} else if len(tasks) > 1 {
		return nil, errors.Errorf("found %v tasks with id %v", len(tasks), id)
	}
	return tasks[0], nil
}

// Get a blocking task in the pipeline.
// A task is blocked by a task with a smaller schema version within the same pipeline.
func (s *Store) FindBlockingTaskByVersion(ctx context.Context, planUID int64, instanceID, databaseName string, version string) (*int, error) {
	myVersion, err := model.NewVersion(version)
	if err != nil {
		return nil, err
	}
	q := qb.Q().Space(`
		SELECT
			task.id,
			task.payload->>'schemaVersion'
		FROM task
		LEFT JOIN plan ON plan.id = task.plan_id
		LEFT JOIN issue ON issue.plan_id = plan.id
		LEFT JOIN LATERAL (
			SELECT COALESCE(
				(SELECT
					task_run.status
				FROM task_run
				WHERE task_run.task_id = task.id
				ORDER BY task_run.id DESC
				LIMIT 1
			), 'NOT_STARTED'
			) AS status
		) AS latest_task_run ON TRUE
		WHERE task.plan_id = ? AND task.instance = ? AND task.db_name = ?
		AND task.payload->>'schemaVersion' IS NOT NULL
		AND (task.payload->>'skipped')::BOOLEAN IS NOT TRUE
		AND latest_task_run.status != 'DONE'
		AND COALESCE(issue.status, 'OPEN') = 'OPEN'
		ORDER BY task.id ASC`, planUID, instanceID, databaseName)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var v string
		if err := rows.Scan(&id, &v); err != nil {
			return nil, errors.Wrapf(err, "failed to scan rows")
		}
		otherVersion, err := model.NewVersion(v)
		if err != nil {
			return nil, err
		}
		if otherVersion.LessThan(myVersion) {
			return &id, nil
		}
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return nil, nil
}

// createTasksTx creates tasks in a transaction.
func (*Store) createTasksTx(ctx context.Context, txn *sql.Tx, creates ...*TaskMessage) ([]*TaskMessage, error) {
	var (
		planIDs      []int64
		instances    []string
		databases    []*string
		environments []string
		types        []string
		payloads     [][]byte
	)
	for _, create := range creates {
		if create.Payload == nil {
			create.Payload = &storepb.Task{}
		}
		payload, err := protojson.Marshal(create.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		planIDs = append(planIDs, create.PlanID)
		instances = append(instances, create.InstanceID)
		databases = append(databases, create.DatabaseName)
		types = append(types, create.Type.String())
		environments = append(environments, create.Environment)
		payloads = append(payloads, payload)
	}

	q := qb.Q().Space(`
		INSERT INTO task (
			plan_id,
			instance,
			db_name,
			environment,
			type,
			payload
		) SELECT
			unnest(CAST(? AS BIGINT[])),
			unnest(CAST(? AS TEXT[])),
			unnest(CAST(? AS TEXT[])),
			unnest(CAST(? AS TEXT[])),
			unnest(CAST(? AS TEXT[])),
			unnest(CAST(? AS JSONB[]))
		RETURNING id, plan_id, instance, db_name, environment, type, payload`,
		planIDs,
		instances,
		databases,
		environments,
		types,
		payloads,
	)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var tasks []*TaskMessage
	rows, err := txn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()
	for rows.Next() {
		task := &TaskMessage{}
		var payload []byte
		var typeString string
		if err := rows.Scan(
			&task.ID,
			&task.PlanID,
			&task.InstanceID,
			&task.DatabaseName,
			&task.Environment,
			&typeString,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan rows")
		}
		if typeValue, ok := storepb.Task_Type_value[typeString]; ok {
			task.Type = storepb.Task_Type(typeValue)
		} else {
			return nil, errors.Errorf("invalid task type string: %s", typeString)
		}
		taskPayload := &storepb.Task{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, taskPayload); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal plan config")
		}
		task.Payload = taskPayload
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return tasks, nil
}

func (*Store) listTasksTx(ctx context.Context, txn *sql.Tx, find *TaskFind) ([]*TaskMessage, error) {
	q := qb.Q().Space(`
		SELECT
			task.id,
			task.plan_id,
			task.instance,
			task.db_name,
			task.environment,
			COALESCE(latest_task_run.status, ?) AS latest_task_run_status,
			task.type,
			task.payload,
			latest_task_run.updated_at,
			latest_task_run.run_at
		FROM task
		LEFT JOIN LATERAL (
			SELECT
				task_run.status,
				task_run.updated_at,
				task_run.run_at
			FROM task_run
			WHERE task_run.task_id = task.id
			ORDER BY task_run.id DESC
			LIMIT 1
		) AS latest_task_run ON TRUE
		WHERE TRUE`, storepb.TaskRun_NOT_STARTED.String())
	if v := find.ID; v != nil {
		q.Space("AND task.id = ?", *v)
	}
	if v := find.IDs; v != nil {
		q.Space("AND task.id = ANY(?)", *v)
	}
	if v := find.PlanID; v != nil {
		q.Space("AND task.plan_id = ?", *v)
	}
	if v := find.PlanIDs; v != nil {
		q.Space("AND task.plan_id = ANY(?)", *v)
	}
	if v := find.Environment; v != nil {
		q.Space("AND task.environment = ?", *v)
	}
	if v := find.InstanceID; v != nil {
		q.Space("AND task.instance = ?", *v)
	}
	if v := find.DatabaseName; v != nil {
		q.Space("AND task.db_name = ?", *v)
	}
	if v := find.LatestTaskRunStatusList; v != nil {
		var statusStrings []string
		for _, status := range *v {
			statusStrings = append(statusStrings, status.String())
		}
		q.Space("AND latest_task_run.status = ANY(?)", statusStrings)
	}
	if v := find.TypeList; v != nil {
		typeStrings := []string{}
		for _, taskType := range *v {
			typeStrings = append(typeStrings, taskType.String())
		}
		q.Space("AND task.type = ANY(?)", typeStrings)
	}
	q.Space("ORDER BY task.id ASC")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := txn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskMessage
	for rows.Next() {
		task := &TaskMessage{}
		var payload []byte
		var latestTaskRunStatusString string
		var typeString string
		if err := rows.Scan(
			&task.ID,
			&task.PlanID,
			&task.InstanceID,
			&task.DatabaseName,
			&task.Environment,
			&latestTaskRunStatusString,
			&typeString,
			&payload,
			&task.UpdatedAt,
			&task.RunAt,
		); err != nil {
			return nil, err
		}
		if typeValue, ok := storepb.Task_Type_value[typeString]; ok {
			task.Type = storepb.Task_Type(typeValue)
		} else {
			return nil, errors.Errorf("invalid task type string: %s", typeString)
		}
		if statusValue, ok := storepb.TaskRun_Status_value[latestTaskRunStatusString]; ok {
			task.LatestTaskRunStatus = storepb.TaskRun_Status(statusValue)
		} else {
			return nil, errors.Errorf("invalid task run status string: %s", latestTaskRunStatusString)
		}
		taskPayload := &storepb.Task{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, taskPayload); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal plan config")
		}
		task.Payload = taskPayload
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListTasks retrieves a list of tasks based on find.
func (s *Store) ListTasks(ctx context.Context, find *TaskFind) ([]*TaskMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	tasks, err := s.listTasksTx(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list tasks")
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// BatchSkipTasks batch skip tasks.
func (s *Store) BatchSkipTasks(ctx context.Context, taskUIDs []int, comment string) error {
	q := qb.Q().Space(`
		UPDATE task
		SET payload = payload || jsonb_build_object('skipped', ?::BOOLEAN) || jsonb_build_object('skippedReason', ?::TEXT)
		WHERE id = ANY(?)`, true, comment, taskUIDs)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to batch skip tasks")
	}

	return nil
}

// CreateTasks creates tasks for a plan.
func (s *Store) CreateTasks(ctx context.Context, planUID int64, tasks []*TaskMessage) ([]*TaskMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	// Check existing tasks to avoid duplicates
	existingTasks, err := s.listTasksTx(ctx, tx, &TaskFind{
		PlanID: &planUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list existing tasks")
	}

	type taskKey struct {
		instance string
		database string
		sheet    string
	}

	createdTasks := map[taskKey]struct{}{}
	for _, task := range existingTasks {
		k := taskKey{
			instance: task.InstanceID,
			sheet:    task.Payload.GetSheetSha256(),
		}
		if task.DatabaseName != nil {
			k.database = *task.DatabaseName
		}
		createdTasks[k] = struct{}{}
	}

	var taskCreateList []*TaskMessage

	for _, taskCreate := range tasks {
		k := taskKey{
			instance: taskCreate.InstanceID,
			sheet:    taskCreate.Payload.GetSheetSha256(),
		}
		if taskCreate.DatabaseName != nil {
			k.database = *taskCreate.DatabaseName
		}

		if _, ok := createdTasks[k]; ok {
			continue
		}
		taskCreate.PlanID = planUID
		taskCreateList = append(taskCreateList, taskCreate)
	}

	if len(taskCreateList) > 0 {
		tasks, err := s.createTasksTx(ctx, tx, taskCreateList...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create tasks")
		}
		if err := tx.Commit(); err != nil {
			return nil, errors.Wrapf(err, "failed to commit tx")
		}
		return tasks, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return []*TaskMessage{}, nil
}
