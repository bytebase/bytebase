package store

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TaskRunMessage is message for task run.
type TaskRunMessage struct {
	TaskUID      int64
	Environment  string // Refer to the task's environment.
	PlanUID      int64
	Status       storepb.TaskRun_Status
	ResultProto  *storepb.TaskRunResult
	PayloadProto *storepb.TaskRunPayload

	// Output only.
	ID           int64
	CreatorEmail string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ProjectID    string
	StartedAt    *time.Time
	RunAt        *time.Time
}

// FindTaskRunMessage is the message for finding task runs.
type FindTaskRunMessage struct {
	// Workspace filters task runs by the parent project's workspace.
	// Empty string skips filtering (for cross-workspace queries like runners).
	Workspace   string
	UID         *int64
	UIDs        *[]int64
	ProjectID   string // Empty string skips filtering for cross-project queries like runners.
	TaskUID     *int64
	Environment *string
	PlanUID     *int64
	Status      *[]storepb.TaskRun_Status
}

// TaskRunStatusPatch is the API message for patching a task run.
type TaskRunStatusPatch struct {
	ID        int64
	ProjectID string

	// Standard fields
	Updater string

	// Domain specific fields
	Status          storepb.TaskRun_Status
	AllowedStatuses []storepb.TaskRun_Status
	ResultProto     *storepb.TaskRunResult
}

// ListTaskRuns lists task runs.
func (s *Store) ListTaskRuns(ctx context.Context, find *FindTaskRunMessage) ([]*TaskRunMessage, error) {
	if find.ProjectID == "" && (find.UID != nil || find.UIDs != nil || find.TaskUID != nil || find.Environment != nil || find.PlanUID != nil) {
		return nil, errors.Errorf("project ID is required for scoped task run filters")
	}

	where := qb.Q()
	if find.ProjectID != "" {
		where.And("task_run.project = ?", find.ProjectID)
	}
	if find.Workspace != "" {
		where.And("task_run.project IN (SELECT resource_id FROM project WHERE workspace = ?)", find.Workspace)
	}
	if v := find.UID; v != nil {
		where.And("task_run.id = ?", *v)
	}
	if v := find.UIDs; v != nil {
		where.And("task_run.id = ANY(?)", *v)
	}
	if v := find.TaskUID; v != nil {
		where.And("task_run.task_id = ?", *v)
	}
	if v := find.Environment; v != nil {
		where.And("task.environment = ?", *v)
	}
	if v := find.PlanUID; v != nil {
		where.And("task.plan_id = ?", *v)
	}
	if v := find.Status; v != nil {
		var statusStrings []string
		for _, status := range *v {
			statusStrings = append(statusStrings, status.String())
		}
		where.And("task_run.status = ANY(?)", statusStrings)
	}

	q := qb.Q().Space(`
		SELECT
			task_run.id,
			task_run.creator,
			task_run.created_at,
			task_run.updated_at,
			task_run.task_id,
			task_run.status,
			task_run.started_at,
			task_run.run_at,
			task_run.result,
			task_run.payload,
			task.plan_id,
			task.environment,
			task_run.project
		FROM task_run
		LEFT JOIN task ON task.project = task_run.project AND task.id = task_run.task_id
	`)
	if where.Len() > 0 {
		q.Space("WHERE ?", where)
	}

	q.Space("ORDER BY task_run.id ASC")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taskRuns []*TaskRunMessage
	for rows.Next() {
		var taskRun TaskRunMessage
		var startedAt, runAt sql.NullTime
		var creatorEmail sql.NullString
		var statusString string
		var resultJSON, payloadJSON string
		if err := rows.Scan(
			&taskRun.ID,
			&creatorEmail,
			&taskRun.CreatedAt,
			&taskRun.UpdatedAt,
			&taskRun.TaskUID,
			&statusString,
			&startedAt,
			&runAt,
			&resultJSON,
			&payloadJSON,
			&taskRun.PlanUID,
			&taskRun.Environment,
			&taskRun.ProjectID,
		); err != nil {
			return nil, err
		}
		if creatorEmail.Valid {
			taskRun.CreatorEmail = creatorEmail.String
		}
		if statusValue, ok := storepb.TaskRun_Status_value[statusString]; ok {
			taskRun.Status = storepb.TaskRun_Status(statusValue)
		} else {
			return nil, errors.Errorf("invalid task run status string: %s", statusString)
		}

		if startedAt.Valid {
			taskRun.StartedAt = &startedAt.Time
		}
		if runAt.Valid {
			taskRun.RunAt = &runAt.Time
		}
		var resultProto storepb.TaskRunResult
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(resultJSON), &resultProto); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal task run result: %s", resultJSON)
		}
		taskRun.ResultProto = &resultProto
		var payloadProto storepb.TaskRunPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadJSON), &payloadProto); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal task run payload: %s", payloadJSON)
		}
		taskRun.PayloadProto = &payloadProto

		taskRuns = append(taskRuns, &taskRun)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return taskRuns, nil
}

// GetTaskRunV1 gets a task run.
func (s *Store) GetTaskRunV1(ctx context.Context, find *FindTaskRunMessage) (*TaskRunMessage, error) {
	taskRuns, err := s.ListTaskRuns(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(taskRuns) == 0 {
		return nil, nil
	}
	if len(taskRuns) > 1 {
		return nil, errors.Errorf("expected to get one task run, but got %d", len(taskRuns))
	}
	return taskRuns[0], nil
}

// UpdateTaskRunStatus updates task run status.
func (s *Store) UpdateTaskRunStatus(ctx context.Context, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	scope, err := s.taskRunLifecycleScope(ctx, patch.ProjectID, []int64{patch.ID})
	if err != nil {
		return nil, err
	}
	var taskRun *TaskRunMessage
	if err := s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		var err error
		taskRun, err = s.patchTaskRunStatusImpl(ctx, tx, patch)
		return errors.Wrap(err, "failed to update task run")
	}); err != nil {
		return nil, err
	}
	return taskRun, nil
}

// ClaimedTaskRun represents a claimed task run with its task UID.
type ClaimedTaskRun struct {
	TaskRunUID int64
	TaskUID    int64
	ProjectID  string
}

// ClaimAvailableTaskRuns atomically claims all AVAILABLE task runs by updating them to RUNNING
// and returns the claimed task run and task UIDs. This combines list + claim into a single atomic operation.
// Uses FOR UPDATE SKIP LOCKED to allow concurrent schedulers to claim different tasks.
func (s *Store) ClaimAvailableTaskRuns(ctx context.Context, replicaID string) ([]*ClaimedTaskRun, error) {
	type candidate struct {
		projectID string
		id        int64
		instance  string
	}
	rows, err := s.GetDB().QueryContext(ctx, `
		SELECT task_run.project, task_run.id, task.instance
		FROM task_run
		JOIN project ON project.resource_id = task_run.project
		JOIN task ON task.project = task_run.project AND task.id = task_run.task_id
		JOIN instance ON instance.resource_id = task.instance
		WHERE task_run.status = $1 AND project.deleted = FALSE AND instance.deleted = FALSE
		ORDER BY task_run.project, task_run.id
	`, storepb.TaskRun_AVAILABLE.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to discover claimable task runs")
	}
	var candidates []candidate
	if err := func() error {
		defer rows.Close()
		for rows.Next() {
			var c candidate
			if err := rows.Scan(&c.projectID, &c.id, &c.instance); err != nil {
				return err
			}
			candidates = append(candidates, c)
		}
		return rows.Err()
	}(); err != nil {
		return nil, errors.Wrap(err, "failed to read claimable task runs")
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	scope := lifecycleScope{}
	projects := make([]string, 0, len(candidates))
	ids := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		scope.addProject(candidate.projectID, lifecycleActive)
		scope.addInstance(candidate.instance, lifecycleActive)
		projects = append(projects, candidate.projectID)
		ids = append(ids, candidate.id)
	}
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now(), replica_id = ?
		WHERE (project, id) IN (
			SELECT task_run.project, task_run.id
			FROM task_run
			JOIN project ON project.resource_id = task_run.project
			JOIN task ON task.project = task_run.project AND task.id = task_run.task_id
			JOIN instance ON instance.resource_id = task.instance
			WHERE task_run.status = ? AND project.deleted = FALSE AND instance.deleted = FALSE
			  AND (task_run.project, task_run.id) IN (
				SELECT unnest(CAST(? AS TEXT[])), unnest(CAST(? AS BIGINT[]))
			  )
			FOR UPDATE OF task_run SKIP LOCKED
		)
		RETURNING id, task_id, project
	`, storepb.TaskRun_RUNNING.String(), replicaID, storepb.TaskRun_AVAILABLE.String(), projects, ids)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var claimed []*ClaimedTaskRun
	err = s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return errors.Wrap(err, "failed to claim task runs")
		}
		defer rows.Close()
		for rows.Next() {
			var c ClaimedTaskRun
			if err := rows.Scan(&c.TaskRunUID, &c.TaskUID, &c.ProjectID); err != nil {
				return err
			}
			claimed = append(claimed, &c)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s *Store) UpdateTaskRunStartAt(ctx context.Context, projectID string, taskRunID int64) error {
	q := qb.Q().Space(`
		UPDATE task_run
		SET started_at = now(), updated_at = now()
		WHERE id = ? AND project = ?
	`, taskRunID, projectID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	scope, err := s.taskRunLifecycleScope(ctx, projectID, []int64{taskRunID})
	if err != nil {
		return err
	}
	var result sql.Result
	err = s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		var err error
		result, err = tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		return errors.Wrapf(err, "failed to update task run start at")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get affected task run count")
	}
	if rowsAffected == 0 {
		return errors.Errorf("task run %d not found in project %s", taskRunID, projectID)
	}
	return nil
}

// CreatePendingTaskRuns creates pending task runs.
// This operation is idempotent and safe for concurrent calls:
// - Excludes skipped tasks and tasks that already have active (PENDING/RUNNING/DONE) task runs
// - Uses ON CONFLICT DO NOTHING to handle race conditions where two requests try to create the same task run
// - The unique constraint on (task_id, attempt) ensures no duplicates
func (s *Store) CreatePendingTaskRuns(ctx context.Context, creator string, creates ...*TaskRunMessage) error {
	if len(creates) == 0 {
		return nil
	}
	projectID := creates[0].ProjectID
	taskUIDs := make([]int64, 0, len(creates))
	projects := make([]string, 0, len(creates))
	scope := lifecycleScope{}
	for _, create := range creates {
		if create.ProjectID != projectID {
			return common.Errorf(common.Invalid, "all task runs in a batch must belong to the same project")
		}
		taskUIDs = append(taskUIDs, create.TaskUID)
		projects = append(projects, create.ProjectID)
		scope.addProject(create.ProjectID, lifecycleActive)
	}
	instanceIDs, err := s.listTaskRunCreationInstances(ctx, projects, taskUIDs)
	if err != nil {
		return err
	}
	for _, instanceID := range instanceIDs {
		scope.addInstance(instanceID, lifecycleActive)
	}
	return s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		return s.createPendingTaskRunsInLifecycleTransaction(ctx, tx, creator, creates...)
	})
}

func (s *Store) createPendingTaskRunsInLifecycleTransaction(ctx context.Context, tx *sql.Tx, creator string, creates ...*TaskRunMessage) error {
	if len(creates) == 0 {
		return nil
	}
	projectID := creates[0].ProjectID
	for _, create := range creates[1:] {
		if create.ProjectID != projectID {
			return common.Errorf(common.Invalid, "all task runs in a batch must belong to the same project")
		}
	}

	var taskUIDs []int64
	var runAts []*time.Time
	var projects []string
	for _, create := range creates {
		taskUIDs = append(taskUIDs, create.TaskUID)
		runAts = append(runAts, create.RunAt)
		projects = append(projects, create.ProjectID)
	}

	// Convert empty string to NULL for system-created task runs
	var creatorPtr any
	if creator == "" {
		creatorPtr = nil
	} else {
		creatorPtr = creator
	}

	// Serialize payload from the first create (all creates in a batch share the same payload).
	payloadStr := "{}"
	if len(creates) > 0 && creates[0].PayloadProto != nil {
		payloadBytes, err := protojson.Marshal(creates[0].PayloadProto)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal task run payload")
		}
		if s := string(payloadBytes); s != "" {
			payloadStr = s
		}
	}
	instanceIDs, err := s.listTaskRunCreationInstances(ctx, projects, taskUIDs)
	if err != nil {
		return err
	}

	lockQ := qb.Q().Space(`
		SELECT task.project, task.id, task.instance
		FROM (
			SELECT
				unnest(CAST(? AS TEXT[])) AS project,
				unnest(CAST(? AS BIGINT[])) AS task_id
		) requested_tasks
		JOIN task ON task.project = requested_tasks.project AND task.id = requested_tasks.task_id
		ORDER BY task.project, task.id
		FOR UPDATE OF task`, projects, taskUIDs)
	lockQuery, lockArgs, err := lockQ.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build task lock sql")
	}
	lockedTaskCount := 0
	lockedInstanceIDs := make([]string, 0, len(instanceIDs))
	if err := func() error {
		rows, err := tx.QueryContext(ctx, lockQuery, lockArgs...)
		if err != nil {
			return errors.Wrapf(err, "failed to lock tasks")
		}
		defer rows.Close()
		for rows.Next() {
			var lockedProjectID string
			var lockedTaskUID int64
			var lockedInstanceID string
			if err := rows.Scan(&lockedProjectID, &lockedTaskUID, &lockedInstanceID); err != nil {
				return errors.Wrapf(err, "failed to scan locked task")
			}
			if !slices.Contains(instanceIDs, lockedInstanceID) {
				return common.Errorf(common.Conflict, "task %s/%d changed to instance %s; retry", lockedProjectID, lockedTaskUID, lockedInstanceID)
			}
			lockedInstanceIDs = append(lockedInstanceIDs, lockedInstanceID)
			lockedTaskCount++
		}
		if err := rows.Err(); err != nil {
			return errors.Wrapf(err, "failed to read locked tasks")
		}
		return nil
	}(); err != nil {
		return err
	}
	if lockedTaskCount == 0 {
		return nil
	}
	slices.Sort(lockedInstanceIDs)
	lockedInstanceIDs = slices.Compact(lockedInstanceIDs)
	for _, instanceID := range lockedInstanceIDs {
		var deleted bool
		if err := tx.QueryRowContext(ctx, `
			SELECT deleted FROM instance WHERE resource_id = $1 FOR UPDATE
		`, instanceID).Scan(&deleted); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return common.Errorf(common.NotFound, "instance %s not found", instanceID)
			}
			return errors.Wrapf(err, "failed to lock instance %s", instanceID)
		}
		if deleted {
			return common.Errorf(common.Conflict, "instance %s is archived", instanceID)
		}
	}

	// Keep nextProjectID's allocator lock after the lifecycle gates and task locks.
	baseID, err := nextProjectID(ctx, tx, "task_run", projectID)
	if err != nil {
		return err
	}

	// Assign per-project IDs, filter existing task runs, and insert new pending runs.
	q := qb.Q().Space(`
		WITH requested_tasks AS (
			SELECT
				tasks.project,
				tasks.task_id,
				tasks.run_at
			FROM (
				SELECT
					unnest(CAST(? AS TEXT[])) AS project,
					unnest(CAST(? AS BIGINT[])) AS task_id,
					unnest(CAST(? AS TIMESTAMPTZ[])) AS run_at
			) tasks
			JOIN task ON task.project = tasks.project AND task.id = tasks.task_id
			WHERE (task.payload->>'skipped')::BOOLEAN IS NOT TRUE
		), candidates AS (
			SELECT
				(ROW_NUMBER() OVER ()) + ? - 1 AS new_id,
				tasks.project,
				tasks.task_id,
				tasks.run_at
			FROM requested_tasks tasks
			WHERE NOT EXISTS (
				SELECT 1 FROM task_run
				WHERE task_run.task_id = tasks.task_id
				AND task_run.project = tasks.project
				AND task_run.status IN (?, ?, ?, ?)
			)
		)
		INSERT INTO task_run (
			id,
			creator,
			project,
			task_id,
			run_at,
			attempt,
			status,
			payload
		)
		SELECT
			candidates.new_id,
			?,
			candidates.project,
			candidates.task_id,
			candidates.run_at,
			COALESCE((SELECT MAX(attempt) + 1 FROM task_run WHERE task_run.task_id = candidates.task_id AND task_run.project = candidates.project), 0),
			?,
			?
		FROM candidates
		ON CONFLICT (project, task_id, attempt) DO NOTHING
	`, projects, taskUIDs, runAts, baseID,
		storepb.TaskRun_PENDING.String(), storepb.TaskRun_AVAILABLE.String(), storepb.TaskRun_RUNNING.String(), storepb.TaskRun_DONE.String(),
		creatorPtr, storepb.TaskRun_PENDING.String(), payloadStr)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	return nil
}

func (s *Store) listTaskRunCreationInstances(ctx context.Context, projects []string, taskUIDs []int64) ([]string, error) {
	q := qb.Q().Space(`
		SELECT DISTINCT task.instance
		FROM (
			SELECT
				unnest(CAST(? AS TEXT[])) AS project,
				unnest(CAST(? AS BIGINT[])) AS task_id
		) requested_tasks
		JOIN task ON task.project = requested_tasks.project AND task.id = requested_tasks.task_id
		ORDER BY task.instance
	`, projects, taskUIDs)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build task run instance query")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list task run instances")
	}
	defer rows.Close()
	var instanceIDs []string
	for rows.Next() {
		var instanceID string
		if err := rows.Scan(&instanceID); err != nil {
			return nil, errors.Wrap(err, "failed to scan task run instance")
		}
		instanceIDs = append(instanceIDs, instanceID)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read task run instances")
	}
	return instanceIDs, nil
}

// patchTaskRunStatusImpl updates a taskRun status. Returns the new state of the taskRun after update.
func (*Store) patchTaskRunStatusImpl(ctx context.Context, txn *sql.Tx, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	set := qb.Q()

	set.Comma("updated_at = ?, status = ?", time.Now(), patch.Status.String())

	if v := patch.ResultProto; v != nil {
		resultBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal task run result")
		}
		result := string(resultBytes)
		if result == "" {
			result = "{}"
		}
		set.Comma("result = ?", result)
	}

	q := qb.Q().Space("UPDATE task_run SET ?", set).
		Space("WHERE id = ? AND project = ?", patch.ID, patch.ProjectID)

	if len(patch.AllowedStatuses) > 0 {
		var statusStrings []string
		for _, status := range patch.AllowedStatuses {
			statusStrings = append(statusStrings, status.String())
		}
		q.Space("AND status = ANY(?)", statusStrings)
	}

	q.Space("RETURNING id, creator, created_at, updated_at, task_id, status, result")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var taskRun TaskRunMessage
	var creatorEmail sql.NullString
	var statusString string
	var resultJSON string
	if err := txn.QueryRowContext(ctx, query, args...).Scan(
		&taskRun.ID,
		&creatorEmail,
		&taskRun.CreatedAt,
		&taskRun.UpdatedAt,
		&taskRun.TaskUID,
		&statusString,
		&resultJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			if len(patch.AllowedStatuses) > 0 {
				return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("task run %d status changed", patch.ID)}
			}
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	if creatorEmail.Valid {
		taskRun.CreatorEmail = creatorEmail.String
	}
	if statusValue, ok := storepb.TaskRun_Status_value[statusString]; ok {
		taskRun.Status = storepb.TaskRun_Status(statusValue)
	} else {
		return nil, errors.Errorf("invalid task run status string: %s", statusString)
	}
	var resultProto storepb.TaskRunResult
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(resultJSON), &resultProto); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal task run result: %s", resultJSON)
	}
	taskRun.ResultProto = &resultProto
	return &taskRun, nil
}

// BatchCancelTaskRuns updates non-running task runs to CANCELED.
func (s *Store) BatchCancelTaskRuns(ctx context.Context, projectID string, taskRunIDs []int64) error {
	if len(taskRunIDs) == 0 {
		return nil
	}
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now()
		WHERE id = ANY(?) AND project = ?
		AND status IN (?, ?)
	`, storepb.TaskRun_CANCELED.String(), taskRunIDs, projectID,
		storepb.TaskRun_PENDING.String(), storepb.TaskRun_AVAILABLE.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	scope, err := s.taskRunLifecycleScope(ctx, projectID, taskRunIDs)
	if err != nil {
		return err
	}
	return s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
}

// FailStaleTaskRuns marks RUNNING task runs as FAILED if their replica is dead.
// A replica is considered dead if:
// 1. Its replica_id is not in the replica_heartbeat table, OR
// 2. Its last_heartbeat is older than the staleness threshold
// Returns the number of task runs marked as failed.
func (s *Store) FailStaleTaskRuns(ctx context.Context, stalenessThreshold time.Duration) (int64, error) {
	rows, err := s.GetDB().QueryContext(ctx, `
		SELECT task_run.project, task_run.id, task.instance
		FROM task_run
		JOIN task ON task.project = task_run.project AND task.id = task_run.task_id
		WHERE task_run.status = $1
		  AND task_run.replica_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM replica_heartbeat rh
			WHERE rh.replica_id = task_run.replica_id
			  AND rh.last_heartbeat >= now() - $2::INTERVAL
		  )
		ORDER BY task_run.project, task_run.id
	`, storepb.TaskRun_RUNNING.String(), stalenessThreshold.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to discover stale task runs")
	}
	scope := lifecycleScope{}
	var projects []string
	var ids []int64
	if err := func() error {
		defer rows.Close()
		for rows.Next() {
			var projectID, instanceID string
			var id int64
			if err := rows.Scan(&projectID, &id, &instanceID); err != nil {
				return err
			}
			scope.addProject(projectID, lifecycleExisting)
			scope.addInstance(instanceID, lifecycleExisting)
			projects = append(projects, projectID)
			ids = append(ids, id)
		}
		return rows.Err()
	}(); err != nil {
		return 0, errors.Wrap(err, "failed to read stale task runs")
	}
	if len(ids) == 0 {
		return 0, nil
	}
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?,
		    result = '{"detail": "Task run abandoned: owning replica stopped responding"}',
		    updated_at = now()
		WHERE status = ?
		  AND replica_id IS NOT NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM replica_heartbeat rh
		    WHERE rh.replica_id = task_run.replica_id
		      AND rh.last_heartbeat >= now() - ?::INTERVAL
		  )
		  AND (task_run.project, task_run.id) IN (
			SELECT unnest(CAST(? AS TEXT[])), unnest(CAST(? AS BIGINT[]))
		  )
	`, storepb.TaskRun_FAILED.String(), storepb.TaskRun_RUNNING.String(), stalenessThreshold.String(), projects, ids)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var result sql.Result
	err = s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		var err error
		result, err = tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to fail stale task runs")
	}
	return result.RowsAffected()
}

// UpdateTaskRunPayload updates the payload column for a task run.
func (s *Store) UpdateTaskRunPayload(ctx context.Context, projectID string, taskRunID int64, payload *storepb.TaskRunPayload) error {
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal task run payload")
	}
	payloadStr := string(payloadBytes)
	if payloadStr == "" {
		payloadStr = "{}"
	}

	q := qb.Q().Space(`
		UPDATE task_run
		SET payload = ?, updated_at = now()
		WHERE id = ? AND project = ?
	`, payloadStr, taskRunID, projectID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	scope, err := s.taskRunLifecycleScope(ctx, projectID, []int64{taskRunID})
	if err != nil {
		return err
	}
	if err := s.runLifecycleWrite(ctx, scope, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		return errors.Wrapf(err, "failed to update task run payload")
	}
	return nil
}

func (s *Store) taskRunLifecycleScope(ctx context.Context, projectID string, taskRunIDs []int64) (lifecycleScope, error) {
	scope := lifecycleScope{}
	scope.addProject(projectID, lifecycleExisting)
	q := qb.Q().Space(`
		SELECT DISTINCT task.instance
		FROM task_run
		JOIN task ON task.project = task_run.project AND task.id = task_run.task_id
		WHERE task_run.project = ? AND task_run.id = ANY(?)
		ORDER BY task.instance
	`, projectID, taskRunIDs)
	query, args, err := q.ToSQL()
	if err != nil {
		return scope, errors.Wrap(err, "failed to build task run lifecycle scope query")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return scope, errors.Wrap(err, "failed to resolve task run lifecycle scope")
	}
	defer rows.Close()
	for rows.Next() {
		var instanceID string
		if err := rows.Scan(&instanceID); err != nil {
			return scope, errors.Wrap(err, "failed to scan task run lifecycle scope")
		}
		scope.addInstance(instanceID, lifecycleExisting)
	}
	return scope, errors.Wrap(rows.Err(), "failed to read task run lifecycle scope")
}
