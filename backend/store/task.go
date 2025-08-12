package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TaskMessage is the message for tasks.
type TaskMessage struct {
	ID int

	// Related fields
	PipelineID     int
	InstanceID     string
	Environment    string
	DatabaseName   *string
	TaskRunRawList []*TaskRunMessage

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
	PipelineID   *int
	Environment  *string
	InstanceID   *string
	DatabaseName *string

	// Domain specific fields
	TypeList *[]storepb.Task_Type

	LatestTaskRunStatusList *[]storepb.TaskRun_Status
}

// TaskPatch is the API message for patching a task.
type TaskPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	DatabaseName *string
	Type         *storepb.Task_Type

	SheetID           *int
	SchemaVersion     *string
	ExportFormat      *storepb.ExportFormat
	ExportPassword    *string
	EnablePriorBackup *bool

	// Flags for gh-ost.
	Flags *map[string]string
}

// GetTaskV2ByID gets a task by ID.
func (s *Store) GetTaskV2ByID(ctx context.Context, id int) (*TaskMessage, error) {
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
func (s *Store) FindBlockingTaskByVersion(ctx context.Context, pipelineUID int, instanceID, databaseName string, version string) (*int, error) {
	myVersion, err := model.NewVersion(version)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT
			task.id,
			task.payload->>'schemaVersion'
		FROM task
		LEFT JOIN pipeline ON task.pipeline_id = pipeline.id
		LEFT JOIN issue ON pipeline.id = issue.pipeline_id
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
		WHERE task.pipeline_id = $1 AND task.instance = $2 AND task.db_name = $3
		AND task.payload->>'schemaVersion' IS NOT NULL
		AND (task.payload->>'skipped')::BOOLEAN IS NOT TRUE
		AND latest_task_run.status != 'DONE'
		AND COALESCE(issue.status, 'OPEN') = 'OPEN'
		ORDER BY task.id ASC
	`
	rows, err := s.GetDB().QueryContext(ctx, query, pipelineUID, instanceID, databaseName)
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

func (*Store) createTasks(ctx context.Context, txn *sql.Tx, creates ...*TaskMessage) ([]*TaskMessage, error) {
	query := `INSERT INTO task (
			pipeline_id,
			instance,
			db_name,
			environment,
			type,
			payload
		) SELECT
			unnest(CAST($1 AS INTEGER[])),
			unnest(CAST($2 AS TEXT[])),
			unnest(CAST($3 AS TEXT[])),
			unnest(CAST($4 AS TEXT[])),
			unnest(CAST($5 AS TEXT[])),
			unnest(CAST($6 AS JSONB[]))
		RETURNING id, pipeline_id, instance, db_name, environment, type, payload`

	var (
		pipelineIDs  []int
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
		pipelineIDs = append(pipelineIDs, create.PipelineID)
		instances = append(instances, create.InstanceID)
		databases = append(databases, create.DatabaseName)
		types = append(types, create.Type.String())
		environments = append(environments, create.Environment)
		payloads = append(payloads, payload)
	}

	var tasks []*TaskMessage
	rows, err := txn.QueryContext(ctx, query,
		pipelineIDs,
		instances,
		databases,
		environments,
		types,
		payloads,
	)
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
			&task.PipelineID,
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
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("task.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.IDs; v != nil {
		where, args = append(where, fmt.Sprintf("task.id = ANY($%d)", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("task.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Environment; v != nil {
		where, args = append(where, fmt.Sprintf("task.environment = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("task.instance = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseName; v != nil {
		where, args = append(where, fmt.Sprintf("task.db_name = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.LatestTaskRunStatusList; v != nil {
		var statusStrings []string
		for _, status := range *v {
			statusStrings = append(statusStrings, status.String())
		}
		where = append(where, fmt.Sprintf("latest_task_run.status = ANY($%d)", len(args)+1))
		args = append(args, statusStrings)
	}
	if v := find.TypeList; v != nil {
		var list []string
		for _, taskType := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, taskType.String())
		}
		where = append(where, fmt.Sprintf("task.type in (%s)", strings.Join(list, ",")))
	}

	args = append(args, storepb.TaskRun_NOT_STARTED.String())
	rows, err := txn.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			task.id,
			task.pipeline_id,
			task.instance,
			task.db_name,
			task.environment,
			COALESCE(latest_task_run.status, $%d) AS latest_task_run_status,
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
		WHERE %s
		ORDER BY task.id ASC`, len(args), strings.Join(where, " AND ")),
		args...,
	)
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
			&task.PipelineID,
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

// UpdateTaskV2 updates an existing task.
// Returns ENOTFOUND if task does not exist.
func (s *Store) UpdateTaskV2(ctx context.Context, patch *TaskPatch) (*TaskMessage, error) {
	set, args := []string{}, []any{}
	if v := patch.DatabaseName; v != nil {
		set, args = append(set, fmt.Sprintf("db_name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Type; v != nil {
		set, args = append(set, fmt.Sprintf("type = $%d", len(args)+1)), append(args, v.String())
	}
	var payloadSet []string
	if v := patch.SheetID; v != nil {
		payloadSet, args = append(payloadSet, fmt.Sprintf(`jsonb_build_object('sheetId', $%d::INT)`, len(args)+1)), append(args, *v)
	}
	if v := patch.SchemaVersion; v != nil {
		payloadSet, args = append(payloadSet, fmt.Sprintf(`jsonb_build_object('schemaVersion', $%d::TEXT)`, len(args)+1)), append(args, *v)
	}
	if v := patch.ExportFormat; v != nil {
		payloadSet, args = append(payloadSet, fmt.Sprintf(`jsonb_build_object('format', $%d::INT)`, len(args)+1)), append(args, *v)
	}
	if v := patch.ExportPassword; v != nil {
		payloadSet, args = append(payloadSet, fmt.Sprintf(`jsonb_build_object('password', $%d::TEXT)`, len(args)+1)), append(args, *v)
	}
	if v := patch.EnablePriorBackup; v != nil {
		payloadSet, args = append(payloadSet, fmt.Sprintf(`jsonb_build_object('enablePriorBackup', $%d::BOOLEAN)`, len(args)+1)), append(args, *v)
	}
	if v := patch.Flags; v != nil {
		jsonb, err := json.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal flags")
		}
		payloadSet, args = append(payloadSet, fmt.Sprintf(`jsonb_build_object('flags', $%d::JSONB)`, len(args)+1)), append(args, jsonb)
	}
	if len(payloadSet) != 0 {
		set = append(set, fmt.Sprintf(`payload = payload || %s`, strings.Join(payloadSet, "||")))
	}
	args = append(args, patch.ID)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	task := &TaskMessage{}
	var payload []byte
	var typeString string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, pipeline_id, instance, db_name, environment, type, payload
	`, len(args)),
		args...,
	).Scan(
		&task.ID,
		&task.PipelineID,
		&task.InstanceID,
		&task.DatabaseName,
		&task.Environment,
		&typeString,
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("task not found with ID %d", patch.ID)}
		}
		return nil, err
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
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return task, nil
}

// BatchSkipTasks batch skip tasks.
func (s *Store) BatchSkipTasks(ctx context.Context, taskUIDs []int, comment string) error {
	query := `
	UPDATE task
	SET payload = payload || jsonb_build_object('skipped', $1::BOOLEAN) || jsonb_build_object('skippedReason', $2::TEXT)
	WHERE id = ANY($3)`
	args := []any{true, comment, taskUIDs}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to batch skip tasks")
	}

	return nil
}

// ListTasksToAutoRollout returns tasks that
// 1. have no task runs
// 2. are not skipped
// 3. are associated with an open issue or no issue
// 4. are in an environment that has auto rollout enabled
// 5. are the first task in the pipeline for their environment
// 6. are not data export tasks.
func (s *Store) ListTasksToAutoRollout(ctx context.Context, environments []string) ([]int, error) {
	rows, err := s.GetDB().QueryContext(ctx, `
	SELECT
		task.pipeline_id,
		task.environment,
		task.id
	FROM task
	LEFT JOIN pipeline ON pipeline.id = task.pipeline_id
	LEFT JOIN issue ON issue.pipeline_id = pipeline.id
	WHERE NOT EXISTS (SELECT 1 FROM task_run WHERE task_run.task_id = task.id)
	AND task.type != 'DATABASE_EXPORT'
	AND COALESCE((task.payload->>'skipped')::BOOLEAN, FALSE) IS FALSE
	AND COALESCE(issue.status, 'OPEN') = 'OPEN'
	AND task.environment = ANY($1)
	ORDER BY task.pipeline_id, task.id`, environments)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group tasks by pipeline and environment, keeping only the first task per environment
	pipelineEnvFirstTask := map[int]map[string]int{}
	for rows.Next() {
		var pipeline int
		var environment string
		var task int
		if err := rows.Scan(&pipeline, &environment, &task); err != nil {
			return nil, err
		}

		if _, ok := pipelineEnvFirstTask[pipeline]; !ok {
			pipelineEnvFirstTask[pipeline] = map[string]int{}
		}
		// Keep only the first task for each environment
		if _, exists := pipelineEnvFirstTask[pipeline][environment]; !exists {
			pipelineEnvFirstTask[pipeline][environment] = task
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var ids []int
	for _, envTasks := range pipelineEnvFirstTask {
		for _, taskID := range envTasks {
			ids = append(ids, taskID)
		}
	}

	slices.Sort(ids)

	return ids, nil
}
