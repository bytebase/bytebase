package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TaskMessage is the message for tasks.
type TaskMessage struct {
	ID int

	// Related fields
	PipelineID     int
	StageID        int
	InstanceID     string
	EnvironmentID  string // This is the stage.
	DatabaseName   *string
	TaskRunRawList []*TaskRunMessage

	// Domain specific fields
	Type              base.TaskType
	Payload           *storepb.TaskPayload
	EarliestAllowedAt *time.Time

	LatestTaskRunStatus base.TaskRunStatus
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
	StageID      *int
	InstanceID   *string
	DatabaseName *string

	// Domain specific fields
	TypeList *[]base.TaskType

	LatestTaskRunStatusList *[]base.TaskRunStatus
}

// TaskPatch is the API message for patching a task.
type TaskPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	DatabaseName            *string
	EarliestAllowedTS       *time.Time
	UpdateEarliestAllowedTS bool
	Type                    *base.TaskType

	SheetID               *int
	SchemaVersion         *string
	ExportFormat          *storepb.ExportFormat
	ExportPassword        *string
	PreUpdateBackupDetail *storepb.PreUpdateBackupDetail

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
	rows, err := s.db.QueryContext(ctx, query, pipelineUID, instanceID, databaseName)
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
			stage_id,
			instance,
			db_name,
			type,
			payload,
			earliest_allowed_at
		) SELECT
			unnest(CAST($1 AS INTEGER[])),
			unnest(CAST($2 AS INTEGER[])),
			unnest(CAST($3 AS TEXT[])),
			unnest(CAST($4 AS TEXT[])),
			unnest(CAST($5 AS TEXT[])),
			unnest(CAST($6 AS JSONB[])),
			unnest(CAST($7 AS TIMESTAMPTZ[]))
		RETURNING id, pipeline_id, stage_id, instance, db_name, type, payload, earliest_allowed_at`

	var (
		pipelineIDs        []int
		stageIDs           []int
		instances          []string
		databases          []*string
		types              []string
		payloads           [][]byte
		earliestAllowedAts []*time.Time
	)
	for _, create := range creates {
		if create.Payload == nil {
			create.Payload = &storepb.TaskPayload{}
		}
		payload, err := protojson.Marshal(create.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		pipelineIDs = append(pipelineIDs, create.PipelineID)
		stageIDs = append(stageIDs, create.StageID)
		instances = append(instances, create.InstanceID)
		databases = append(databases, create.DatabaseName)
		types = append(types, string(create.Type))
		payloads = append(payloads, payload)
		if create.EarliestAllowedAt == nil {
			earliestAllowedAts = append(earliestAllowedAts, nil)
		} else {
			earliestAllowedAts = append(earliestAllowedAts, create.EarliestAllowedAt)
		}
	}

	var tasks []*TaskMessage
	rows, err := txn.QueryContext(ctx, query,
		pipelineIDs,
		stageIDs,
		instances,
		databases,
		types,
		payloads,
		earliestAllowedAts,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()
	for rows.Next() {
		task := &TaskMessage{}
		var earliestAllowedAt sql.NullTime
		var payload []byte
		if err := rows.Scan(
			&task.ID,
			&task.PipelineID,
			&task.StageID,
			&task.InstanceID,
			&task.DatabaseName,
			&task.Type,
			&payload,
			&earliestAllowedAt,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan rows")
		}
		if earliestAllowedAt.Valid {
			task.EarliestAllowedAt = &earliestAllowedAt.Time
		}
		taskPayload := &storepb.TaskPayload{}
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
	if v := find.StageID; v != nil {
		where, args = append(where, fmt.Sprintf("task.stage_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("task.instance = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseName; v != nil {
		where, args = append(where, fmt.Sprintf("task.db_name = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.LatestTaskRunStatusList; v != nil {
		where = append(where, fmt.Sprintf("latest_task_run.status = ANY($%d)", len(args)+1))
		args = append(args, *v)
	}
	if v := find.TypeList; v != nil {
		var list []string
		for _, taskType := range *v {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, taskType)
		}
		where = append(where, fmt.Sprintf("task.type in (%s)", strings.Join(list, ",")))
	}

	args = append(args, base.TaskRunNotStarted)
	rows, err := txn.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			task.id,
			task.pipeline_id,
			task.stage_id,
			task.instance,
			task.db_name,
			latest_task_run.status AS latest_task_run_status,
			task.type,
			task.payload,
			task.earliest_allowed_at
		FROM task
		LEFT JOIN LATERAL (
			SELECT COALESCE(
				(SELECT
					task_run.status
				FROM task_run
				WHERE task_run.task_id = task.id
				ORDER BY task_run.id DESC
				LIMIT 1
				), $%d
			) AS status
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
		var earliestAllowedAt sql.NullTime
		var payload []byte
		if err := rows.Scan(
			&task.ID,
			&task.PipelineID,
			&task.StageID,
			&task.InstanceID,
			&task.DatabaseName,
			&task.LatestTaskRunStatus,
			&task.Type,
			&payload,
			&earliestAllowedAt,
		); err != nil {
			return nil, err
		}
		if earliestAllowedAt.Valid {
			task.EarliestAllowedAt = &earliestAllowedAt.Time
		}
		taskPayload := &storepb.TaskPayload{}
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
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
		set, args = append(set, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
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
	if v := patch.PreUpdateBackupDetail; v != nil {
		jsonb, err := json.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal preUpdateBackupDetail")
		}
		payloadSet, args = append(payloadSet, fmt.Sprintf(`jsonb_build_object('preUpdateBackupDetail', $%d::JSONB)`, len(args)+1)), append(args, jsonb)
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
	if patch.UpdateEarliestAllowedTS {
		if patch.EarliestAllowedTS == nil {
			set = append(set, "earliest_allowed_at = null")
		} else {
			set, args = append(set, fmt.Sprintf("earliest_allowed_at = $%d", len(args)+1)), append(args, *patch.EarliestAllowedTS)
		}
	}
	args = append(args, patch.ID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	task := &TaskMessage{}
	var earliestAllowedAt sql.NullTime
	var payload []byte
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE task
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, pipeline_id, stage_id, instance, db_name, type, payload, earliest_allowed_at
	`, len(args)),
		args...,
	).Scan(
		&task.ID,
		&task.PipelineID,
		&task.StageID,
		&task.InstanceID,
		&task.DatabaseName,
		&task.Type,
		&payload,
		&earliestAllowedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("task not found with ID %d", patch.ID)}
		}
		return nil, err
	}

	if earliestAllowedAt.Valid {
		task.EarliestAllowedAt = &earliestAllowedAt.Time
	}
	taskPayload := &storepb.TaskPayload{}
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

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to batch skip tasks")
	}

	return nil
}

// ListTasksToAutoRollout returns tasks that
// 1. have no task runs
// 2. are not skipped
// 3. are associated with an open issue or no issue
// 4. are in an environment that has auto rollout enabled
// 5. are in the stage that is the first among the selected stages in the pipeline
// 6. are not data export tasks.
func (s *Store) ListTasksToAutoRollout(ctx context.Context, environments []string) ([]int, error) {
	rows, err := s.db.QueryContext(ctx, `
	SELECT
		task.pipeline_id,
		task.stage_id,
		task.id
	FROM task
	LEFT JOIN stage ON stage.id = task.stage_id
	LEFT JOIN pipeline ON pipeline.id = task.pipeline_id
	LEFT JOIN issue ON issue.pipeline_id = pipeline.id
	WHERE NOT EXISTS (SELECT 1 FROM task_run WHERE task_run.task_id = task.id)
	AND task.type != 'bb.task.database.data.export'
	AND COALESCE((task.payload->>'skipped')::BOOLEAN, FALSE) IS FALSE
	AND COALESCE(issue.status, 'OPEN') = 'OPEN'
	AND stage.environment = ANY($1)
	`, environments)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pipelineStageTasks := map[int]map[int][]int{}
	for rows.Next() {
		var pipeline, stage, task int
		if err := rows.Scan(&pipeline, &stage, &task); err != nil {
			return nil, err
		}

		if _, ok := pipelineStageTasks[pipeline]; !ok {
			pipelineStageTasks[pipeline] = map[int][]int{}
		}
		pipelineStageTasks[pipeline][stage] = append(pipelineStageTasks[pipeline][stage], task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var ids []int
	for pipeline := range pipelineStageTasks {
		minStage := math.MaxInt32
		for stage := range pipelineStageTasks[pipeline] {
			if stage < minStage {
				minStage = stage
			}
		}
		if minStage == math.MaxInt32 {
			continue
		}
		ids = append(ids, pipelineStageTasks[pipeline][minStage]...)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	return ids, nil
}
