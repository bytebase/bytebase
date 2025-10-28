package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
)

// PipelineMessage is the message for pipelines.
type PipelineMessage struct {
	ProjectID string
	Tasks     []*TaskMessage
	// Output only.
	ID         int
	CreatorUID int
	CreatedAt  time.Time
	// The UpdatedAt is a latest time of task/taskRun updates.
	// If there are no tasks, it will be the same as CreatedAt.
	UpdatedAt time.Time
	IssueID   *int
}

// PipelineFind is the API message for finding pipelines.
type PipelineFind struct {
	ID        *int
	ProjectID *string

	Limit  *int
	Offset *int

	Filter *ListResourceFilter
}

// CreatePipelineAIO creates a pipeline with tasks all in one.
func (s *Store) CreatePipelineAIO(ctx context.Context, planUID int64, pipeline *PipelineMessage, creatorUID int) (createdPipelineUID int, err error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	pipelineUIDMaybe, err := lockPlanAndGetPipelineUID(ctx, tx, planUID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to SELECT plan FOR UPDATE")
	}
	invalidateCacheF := func() {}
	if pipelineUIDMaybe == nil {
		createdPipeline, err := s.createPipeline(ctx, tx, pipeline, creatorUID)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to create pipeline")
		}
		createdPipelineUID = createdPipeline.ID

		// update pipeline uid of associated issue and plan
		if invalidateCacheF, err = s.updatePipelineUIDOfPlan(ctx, tx, planUID, createdPipelineUID); err != nil {
			return 0, errors.Wrapf(err, "failed to update associated plan or issue")
		}
	} else {
		createdPipelineUID = *pipelineUIDMaybe
	}

	// Check existing tasks to avoid duplicates
	existingTasks, err := s.listTasksTx(ctx, tx, &TaskFind{
		PipelineID: &createdPipelineUID,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to list existing tasks")
	}

	type taskKey struct {
		instance string
		database string
		sheet    int
	}

	createdTasks := map[taskKey]struct{}{}
	for _, task := range existingTasks {
		k := taskKey{
			instance: task.InstanceID,
			sheet:    int(task.Payload.GetSheetId()),
		}
		if task.DatabaseName != nil {
			k.database = *task.DatabaseName
		}
		createdTasks[k] = struct{}{}
	}

	var taskCreateList []*TaskMessage

	for _, taskCreate := range pipeline.Tasks {
		k := taskKey{
			instance: taskCreate.InstanceID,
			sheet:    int(taskCreate.Payload.GetSheetId()),
		}
		if taskCreate.DatabaseName != nil {
			k.database = *taskCreate.DatabaseName
		}

		if _, ok := createdTasks[k]; ok {
			continue
		}
		taskCreate.PipelineID = createdPipelineUID
		taskCreateList = append(taskCreateList, taskCreate)
	}

	if len(taskCreateList) > 0 {
		if _, err := s.createTasks(ctx, tx, taskCreateList...); err != nil {
			return 0, errors.Wrap(err, "failed to create tasks")
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}
	invalidateCacheF()

	return createdPipelineUID, nil
}

// returns func() to invalidate cache.
func (*Store) updatePipelineUIDOfPlan(ctx context.Context, txn *sql.Tx, planUID int64, pipelineUID int) (func(), error) {
	q := qb.Q().Space(`
		UPDATE plan
		SET pipeline_id = ?
		WHERE id = ?
	`, pipelineUID, planUID)
	querySQL, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if _, err := txn.ExecContext(ctx, querySQL, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to update plan pipeline_id")
	}

	return func() {
		// TODO: need to remove planCache once we add planCache
	}, nil
}

func lockPlanAndGetPipelineUID(ctx context.Context, txn *sql.Tx, planUID int64) (*int, error) {
	q := qb.Q().Space("SELECT pipeline_id FROM plan WHERE id = ? FOR UPDATE", planUID)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	var uid sql.NullInt32
	if err := txn.QueryRowContext(ctx, query, args...).Scan(&uid); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Errorf("plan %d not found", planUID)
		}
		return nil, errors.Wrapf(err, "failed to get pipeline uid")
	}

	if uid.Valid {
		uidInt := int(uid.Int32)
		return &uidInt, nil
	}
	return nil, nil
}

func (*Store) createPipeline(ctx context.Context, txn *sql.Tx, create *PipelineMessage, creatorUID int) (*PipelineMessage, error) {
	q := qb.Q().Space(`
		INSERT INTO pipeline (
			project,
			creator_id
		)
		VALUES (
			?,
			?
		)
		RETURNING id, created_at
	`, create.ProjectID, creatorUID)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	pipeline := &PipelineMessage{
		ProjectID:  create.ProjectID,
		CreatorUID: creatorUID,
	}
	if err := txn.QueryRowContext(ctx, query, args...).Scan(
		&pipeline.ID,
		&pipeline.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, errors.Wrapf(err, "failed to insert")
	}
	// Initialize UpdatedAt with CreatedAt for new pipelines
	pipeline.UpdatedAt = pipeline.CreatedAt

	return pipeline, nil
}

// GetPipelineV2ByID gets the pipeline by ID.
func (s *Store) GetPipelineV2ByID(ctx context.Context, id int) (*PipelineMessage, error) {
	if v, ok := s.pipelineCache.Get(id); ok && s.enableCache {
		return v, nil
	}
	pipelines, err := s.ListPipelineV2(ctx, &PipelineFind{ID: &id})
	if err != nil {
		return nil, err
	}

	if len(pipelines) == 0 {
		return nil, nil
	} else if len(pipelines) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d pipelines, expect 1", len(pipelines))}
	}
	pipeline := pipelines[0]
	return pipeline, nil
}

// ListPipelineV2 lists pipelines.
func (s *Store) ListPipelineV2(ctx context.Context, find *PipelineFind) ([]*PipelineMessage, error) {
	q := qb.Q().Space(`
		SELECT
			pipeline.id,
			pipeline.creator_id,
			pipeline.created_at,
			pipeline.project,
			issue.id,
			COALESCE(
				(
					SELECT MAX(task_run.updated_at)
					FROM task
					JOIN task_run ON task_run.task_id = task.id
					WHERE task.pipeline_id = pipeline.id
				),
				pipeline.created_at
			) AS updated_at
		FROM pipeline
		LEFT JOIN plan ON plan.pipeline_id = pipeline.id
		LEFT JOIN issue ON issue.plan_id = plan.id
		WHERE TRUE
	`)

	if filter := find.Filter; filter != nil {
		q.And(ConvertDollarPlaceholders(filter.Where), filter.Args...)
	}
	if v := find.ID; v != nil {
		q.And("pipeline.id = ?", *v)
	}
	if v := find.ProjectID; v != nil {
		q.And("pipeline.project = ?", *v)
	}

	q.Space("ORDER BY pipeline.id DESC")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pipelines []*PipelineMessage
	for rows.Next() {
		var pipeline PipelineMessage
		if err := rows.Scan(
			&pipeline.ID,
			&pipeline.CreatorUID,
			&pipeline.CreatedAt,
			&pipeline.ProjectID,
			&pipeline.IssueID,
			&pipeline.UpdatedAt,
		); err != nil {
			return nil, err
		}
		pipelines = append(pipelines, &pipeline)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, pipeline := range pipelines {
		s.pipelineCache.Add(pipeline.ID, pipeline)
	}
	return pipelines, nil
}
