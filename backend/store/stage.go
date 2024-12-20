package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

// StageMessage is the message for stage.
type StageMessage struct {
	Name          string
	EnvironmentID int
	PipelineID    int
	TaskList      []*TaskMessage

	// empty for legacy stages
	DeploymentID string

	// Output only.
	ID     int
	Active bool

	// TODO(d): this is used to create the tasks.
	TaskIndexDAGList []TaskIndexDAG
}

// TaskIndexDAG describes task dependency relationship using array index to represent task.
// It is needed because we don't know task id before insertion, so we describe the dependency
// using the in-memory representation, i.e, the array index.
type TaskIndexDAG struct {
	FromIndex int
	ToIndex   int
}

func (*Store) createStages(ctx context.Context, tx *Tx, stagesCreate []*StageMessage, pipelineUID int, creatorID int) ([]*StageMessage, error) {
	var environmentIDs []int
	var names []string
	var deploymentIDs []string
	for _, create := range stagesCreate {
		environmentIDs = append(environmentIDs, create.EnvironmentID)
		names = append(names, create.Name)
		deploymentIDs = append(deploymentIDs, create.DeploymentID)
	}

	query := `
		INSERT INTO stage (
			creator_id,
			updater_id,
			pipeline_id,
			environment_id,
			name,
			deployment_id
		) SELECT
			$1,
			$1,
			$2,
			unnest(CAST($3 AS INTEGER[])) AS environment_id,
			unnest(CAST($4 AS TEXT[])),
			unnest(CAST($5 AS TEXT[])) AS deployment_id
		RETURNING id
    `
	rows, err := tx.QueryContext(ctx, query, creatorID, pipelineUID, environmentIDs, names, deploymentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		stage := stagesCreate[i]
		if err := rows.Scan(
			&stage.ID,
		); err != nil {
			return nil, err
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stagesCreate, nil
}

// CreateStageV2 creates a list of stages.
func (s *Store) CreateStageV2(ctx context.Context, stagesCreate []*StageMessage, pipelineUID int, creatorID int) ([]*StageMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stages, err := s.createStages(ctx, tx, stagesCreate, pipelineUID, creatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create stages")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit")
	}

	return stages, nil
}

func (*Store) listStages(ctx context.Context, tx *Tx, pipelineUID int) ([]*StageMessage, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			stage.id,
			stage.pipeline_id,
			stage.environment_id,
			stage.deployment_id,
			stage.name,
			(
				SELECT EXISTS (
					SELECT 1 FROM task
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
					WHERE task.pipeline_id = stage.pipeline_id
					AND task.stage_id <= stage.id
					AND NOT (
						COALESCE((task.payload->>'skipped')::BOOLEAN, FALSE) IS TRUE
						OR latest_task_run.status = 'DONE'
					)
				)
			) AS active
		FROM stage
		WHERE pipeline_id = $1 ORDER BY id ASC`,
		pipelineUID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query rows")
	}
	defer rows.Close()

	var stages []*StageMessage
	for rows.Next() {
		var stage StageMessage
		if err := rows.Scan(
			&stage.ID,
			&stage.PipelineID,
			&stage.EnvironmentID,
			&stage.DeploymentID,
			&stage.Name,
			&stage.Active,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		stages = append(stages, &stage)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}
	return stages, nil
}

// ListStageV2 finds a list of stages based on find.
func (s *Store) ListStageV2(ctx context.Context, pipelineUID int) ([]*StageMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	stages, err := s.listStages(ctx, tx, pipelineUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list stages")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}
	return stages, nil
}
