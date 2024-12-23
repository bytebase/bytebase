package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// PipelineMessage is the message for pipelines.
type PipelineMessage struct {
	ProjectID string
	Name      string
	Stages    []*StageMessage
	// Output only.
	ID         int
	CreatorUID int
	CreatedTs  int64
	UpdaterUID int
	UpdatedTs  int64
	IssueID    *int
}

// PipelineFind is the API message for finding pipelines.
type PipelineFind struct {
	ID        *int
	ProjectID *string

	Limit  *int
	Offset *int
}

// targetStage == "" means deploy all stages.
func (s *Store) CreatePipelineAIO(ctx context.Context, planUID int64, pipeline *PipelineMessage, creatorUID int) (createdPipelineUID int, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	pipelineUIDMaybe, err := lockPlanAndGetPipelineUID(ctx, tx, planUID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to SELECT plan FOR UPDATE")
	}
	if pipelineUIDMaybe == nil {
		createdPipeline, err := s.createPipeline(ctx, tx, pipeline, creatorUID)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to create pipeline")
		}
		createdPipelineUID = createdPipeline.ID

		// update pipeline uid of associated issue and plan
		if err := updatePipelineUIDOfIssueAndPlan(ctx, tx, planUID, createdPipelineUID); err != nil {
			return 0, errors.Wrapf(err, "failed to update associated plan or issue")
		}
	} else {
		createdPipelineUID = *pipelineUIDMaybe
	}

	stages, err := s.listStages(ctx, tx, createdPipelineUID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to list stages")
	}
	stagesAlreadyExist := map[string]bool{}
	for _, stage := range stages {
		stagesAlreadyExist[stage.DeploymentID] = true
	}

	var stagesToCreate []*StageMessage
	for _, stage := range pipeline.Stages {
		if !stagesAlreadyExist[stage.DeploymentID] {
			stagesToCreate = append(stagesToCreate, stage)
		}
	}

	createdStages, err := s.createStages(ctx, tx, stagesToCreate, createdPipelineUID, creatorUID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to create stages")
	}

	for _, stage := range createdStages {
		var taskCreateList []*TaskMessage
		for _, taskCreate := range stage.TaskList {
			c := taskCreate
			c.CreatorID = creatorUID
			c.PipelineID = createdPipelineUID
			c.StageID = stage.ID
			taskCreateList = append(taskCreateList, c)
		}
		tasks, err := s.createTasks(ctx, tx, taskCreateList...)
		if err != nil {
			return 0, errors.Wrap(err, "failed to create tasks")
		}

		// TODO(p0ny): create task dags in batch.
		for _, indexDAG := range stage.TaskIndexDAGList {
			if err := s.createTaskDAG(ctx, tx, &TaskDAGMessage{
				FromTaskID: tasks[indexDAG.FromIndex].ID,
				ToTaskID:   tasks[indexDAG.ToIndex].ID,
			}); err != nil {
				return 0, errors.Wrap(err, "failed to create task DAG")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}

	return createdPipelineUID, nil
}

func updatePipelineUIDOfIssueAndPlan(ctx context.Context, tx *Tx, planUID int64, pipelineUID int) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE plan
		SET pipeline_id = $1
		WHERE id = $2
	`, pipelineUID, planUID); err != nil {
		return errors.Wrapf(err, "failed to update plan pipeline_id")
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE issue
		SET pipeline_id = $1
		WHERE plan_id = $2
	`, pipelineUID, planUID); err != nil {
		return errors.Wrapf(err, "failed to update issue pipeline_id")
	}
	return nil
}

func lockPlanAndGetPipelineUID(ctx context.Context, tx *Tx, planUID int64) (*int, error) {
	query := `
		SELECT pipeline_id FROM plan WHERE id = $1 FOR UPDATE
	`
	var uid sql.NullInt32
	if err := tx.QueryRowContext(ctx, query, planUID).Scan(&uid); err != nil {
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

func (*Store) createPipeline(ctx context.Context, tx *Tx, create *PipelineMessage, creatorUID int) (*PipelineMessage, error) {
	query := `
		INSERT INTO pipeline (
			project_id,
			creator_id,
			updater_id,
			name
		)
		VALUES (
			(SELECT project.id FROM project WHERE project.resource_id = $1),
			$2,
			$3,
			$4
		)
		RETURNING id, created_ts
	`
	pipeline := &PipelineMessage{
		ProjectID:  create.ProjectID,
		CreatorUID: creatorUID,
		UpdaterUID: creatorUID,
		Name:       create.Name,
	}
	if err := tx.QueryRowContext(ctx, query,
		create.ProjectID,
		creatorUID,
		creatorUID,
		create.Name,
	).Scan(
		&pipeline.ID,
		&pipeline.CreatedTs,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, errors.Wrapf(err, "failed to insert")
	}

	pipeline.UpdatedTs = pipeline.CreatedTs
	return pipeline, nil
}

// GetPipelineV2ByID gets the pipeline by ID.
func (s *Store) GetPipelineV2ByID(ctx context.Context, id int) (*PipelineMessage, error) {
	if v, ok := s.pipelineCache.Get(id); ok {
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
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("pipeline.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	query := fmt.Sprintf(`
		SELECT
			pipeline.id,
			pipeline.creator_id,
			pipeline.created_ts,
			pipeline.updater_id,
			pipeline.updated_ts,
			project.resource_id,
			pipeline.name,
			issue.id
		FROM pipeline
		LEFT JOIN project ON pipeline.project_id = project.id
		LEFT JOIN issue ON pipeline.id = issue.pipeline_id
		WHERE %s
		ORDER BY pipeline.id DESC`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
			&pipeline.CreatedTs,
			&pipeline.UpdaterUID,
			&pipeline.UpdatedTs,
			&pipeline.ProjectID,
			&pipeline.Name,
			&pipeline.IssueID,
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
